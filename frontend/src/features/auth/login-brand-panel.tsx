"use client";

import {
  Fragment,
  useEffect,
  useRef,
  useState,
  useSyncExternalStore,
  type CSSProperties,
  type ReactNode,
} from "react";
import { ChevronRight, Database, Shield, ShieldCheck, TriangleAlert, Wrench } from "lucide-react";
import { BrandMark } from "@/components/business/brand-mark";
import { APP } from "@/lib/config";
import styles from "./login-brand-panel.module.css";

type StageIcon = "database" | "alert" | "tool" | "role";

type Stage = {
  name: string;
  icon: StageIcon;
  brief: string;
  tone: number;
  mark: string;
  cmd: string;
  out: string;
  spin: string;
  dots: number;
};

const STAGES: Stage[] = [
  {
    name: "资产管理",
    icon: "database",
    brief: "厘清资产全貌",
    tone: 2,
    mark: "✓",
    cmd: "vulnctl assets --scan",
    out: "inventory synced · 0 drift",
    spin: "streaming",
    dots: 21,
  },
  {
    name: "漏洞分级",
    icon: "alert",
    brief: "聚焦关键风险",
    tone: 1,
    mark: "▲",
    cmd: "vulnctl vuln --triage",
    out: "severity mapped · flagged for review",
    spin: "scoring",
    dots: 23,
  },
  {
    name: "修复工单",
    icon: "tool",
    brief: "让修复有序推进",
    tone: 3,
    mark: "→",
    cmd: "vulnctl fix --track",
    out: "patch in progress · owner assigned",
    spin: "syncing",
    dots: 23,
  },
  {
    name: "复测闭环",
    icon: "role",
    brief: "让结果可追溯",
    tone: 5,
    mark: "✓",
    cmd: "vulnctl verify --close",
    out: "loop closed · risk cleared",
    spin: "watching",
    dots: 22,
  },
];

const SPIN_FRAMES = ["⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"];
const REDUCED_MOTION_QUERY = "(prefers-reduced-motion: reduce)";

function subscribeReducedMotion(onChange: () => void) {
  const media = window.matchMedia(REDUCED_MOTION_QUERY);
  media.addEventListener("change", onChange);
  return () => media.removeEventListener("change", onChange);
}

function getReducedMotionSnapshot() {
  return window.matchMedia(REDUCED_MOTION_QUERY).matches;
}

function getReducedMotionServerSnapshot() {
  return false;
}

function stageIcon(name: StageIcon): ReactNode {
  const props = { size: 18, strokeWidth: 1.6 };
  if (name === "database") return <Database {...props} />;
  if (name === "alert") return <TriangleAlert {...props} />;
  if (name === "tool") return <Wrench {...props} />;
  return <ShieldCheck {...props} />;
}

export function LoginBrandPanel() {
  const [selected, setSelected] = useState(0);
  const [typedCmd, setTypedCmd] = useState("");
  const [typedOut, setTypedOut] = useState("");
  const [showLine, setShowLine] = useState(false);
  const [spinChar, setSpinChar] = useState(SPIN_FRAMES[1] ?? "⠙");
  const autoTimer = useRef<number | null>(null);
  const reducedMotion = useSyncExternalStore(
    subscribeReducedMotion,
    getReducedMotionSnapshot,
    getReducedMotionServerSnapshot,
  );

  useEffect(() => {
    if (reducedMotion) return;
    autoTimer.current = window.setInterval(() => {
      setSelected((current) => (current + 1) % STAGES.length);
    }, 3800);
    return () => {
      if (autoTimer.current !== null) window.clearInterval(autoTimer.current);
      autoTimer.current = null;
    };
  }, [reducedMotion]);

  useEffect(() => {
    if (reducedMotion) return;
    let frame = 0;
    const timer = window.setInterval(() => {
      frame = (frame + 1) % SPIN_FRAMES.length;
      setSpinChar(SPIN_FRAMES[frame] ?? "⠙");
    }, 90);
    return () => window.clearInterval(timer);
  }, [reducedMotion]);

  useEffect(() => {
    const stage = STAGES[selected];
    if (!stage) return;
    let lineTimer: number | null = null;
    let typeTimer: number | null = null;

    if (reducedMotion) return;

    const resetTimer = window.setTimeout(() => {
      setTypedCmd("");
      setTypedOut("");
      setShowLine(false);
      let index = 0;
      typeTimer = window.setInterval(() => {
        index += 1;
        setTypedCmd(stage.cmd.slice(0, index));
        if (index >= stage.cmd.length && typeTimer !== null) {
          window.clearInterval(typeTimer);
          setTypedOut(stage.out);
          lineTimer = window.setTimeout(() => setShowLine(true), 260);
        }
      }, 34);
    }, 0);

    return () => {
      window.clearTimeout(resetTimer);
      if (typeTimer !== null) window.clearInterval(typeTimer);
      if (lineTimer !== null) window.clearTimeout(lineTimer);
    };
  }, [reducedMotion, selected]);

  const pickStage = (index: number) => {
    if (autoTimer.current !== null) {
      window.clearInterval(autoTimer.current);
      autoTimer.current = null;
    }
    setSelected(index);
  };

  const activeStage = STAGES[selected] ?? STAGES[0]!;
  const displayedCmd = reducedMotion ? activeStage.cmd : typedCmd;
  const displayedOut = reducedMotion ? activeStage.out : typedOut;
  const displayedLine = reducedMotion || showLine;
  const displayedSpin = reducedMotion ? "⠙" : spinChar;

  return (
    <section className={`${styles.brandPanel} brand-panel`} aria-labelledby="brand-title">
      <div className={`${styles.brandTop} brand-top`}>
        <div className={styles.platformBrand}>
          <BrandMark />
          <div>
            <div className="brand-name">{APP.shortName}</div>
            <div className="brand-sub">{APP.subtitle}</div>
          </div>
        </div>
        <span className={styles.brandEdition}>
          <Shield size={13} />
          风险治理 · 全程可循
        </span>
      </div>

      <div className={`${styles.brandCopy} brand-copy`}>
        <div className={styles.introLabel}>
          <span />
          让风险治理，更进一步
        </div>
        <h1 id="brand-title">
          万物皆有裂痕，
          <br />
          <span>那是光照进来的地方。</span>
        </h1>
        <p>
          以金修补裂痕，以流程治理风险——
          <br />
          发现、修复、复测，皆有迹可循。
        </p>
      </div>

      <div className={styles.stageBox} aria-hidden="true">
        <div className={styles.terminal}>
          <div className={styles.termChrome}>
            <span className={styles.termDots}>
              <i />
              <i />
              <i />
            </span>
            <span className={styles.termTag}>vulnctl — live</span>
          </div>
          <code>
            <span className={styles.prompt}>$</span> {displayedCmd}
            <span
              className={styles.cursor}
              style={{ background: `var(--cat-${activeStage.tone}-ink)` }}
            />
          </code>
          <code style={{ color: `var(--cat-${activeStage.tone}-ink)` }}>
            <span className={styles.outMark}>{activeStage.mark}</span>
            {displayedOut}
          </code>
          <code className={`${styles.termLine} ${displayedLine ? styles.isIn : ""}`}>
            <span
              className={styles.spinChar}
              style={{ color: `var(--cat-${activeStage.tone}-ink)` }}
            >
              {displayedSpin}
            </span>{" "}
            {activeStage.spin} <span>{"·".repeat(activeStage.dots)}</span> live
          </code>
        </div>
      </div>

      <div className={`${styles.journey} journey`} role="group" aria-label="漏洞治理流程预览">
        {STAGES.map((stage, index) => (
          <Fragment key={stage.name}>
            <button
              type="button"
              aria-pressed={selected === index}
              className={`${styles.journeyStep} journey-step ${
                selected === index ? styles.isActive : ""
              }`}
              style={
                {
                  "--step-accent": `var(--cat-${stage.tone}-ink)`,
                } as CSSProperties
              }
              onClick={() => pickStage(index)}
            >
              <span className={styles.stepIcon}>{stageIcon(stage.icon)}</span>
              <strong>{stage.name}</strong>
              <small>{stage.brief}</small>
            </button>
            {index < STAGES.length - 1 ? (
              <span className={styles.stepArrow} aria-hidden="true">
                <ChevronRight size={13} />
              </span>
            ) : null}
          </Fragment>
        ))}
      </div>

      <footer className={`${styles.brandFooter} brand-footer`}>
        <span>{APP.copyright}</span>
        <span className={styles.footerSign}>
          <Shield size={12} />
          资产有序 · 风险可控
        </span>
      </footer>
    </section>
  );
}
