"use client";

import {
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
  type FormEvent,
  type PointerEvent as ReactPointerEvent,
} from "react";
import { useRouter } from "next/navigation";
import { RefreshCw } from "lucide-react";
import { gsap } from "gsap";
import { LoginBrandPanel } from "./login-brand-panel";
import { Button } from "@/components/ui/button";
import { Checkbox, Field, TextInput } from "@/components/ui/input";
import { useToast } from "@/components/providers/toast-provider";
import { ApiError } from "@/lib/api/client";
import { getPublicSettings, issueCaptcha, login } from "@/lib/api/auth";
import type { CaptchaResponse } from "@/lib/api/types";
import { EASE, prefersReducedMotion } from "@/lib/motion";
import { cn } from "@/lib/utils/cn";
import styles from "./login-screen.module.css";

type FormState = { username: string; password: string; captcha: string };

export function LoginScreen() {
  const router = useRouter();
  const { pushToast } = useToast();
  const [form, setForm] = useState<FormState>({ username: "", password: "", captcha: "" });
  const [errors, setErrors] = useState<Partial<Record<keyof FormState, string>>>({});
  const [remember, setRemember] = useState(true);
  const [captcha, setCaptcha] = useState<CaptchaResponse | null>(null);
  const [captchaLoading, setCaptchaLoading] = useState(false);
  const [captchaEnabled, setCaptchaEnabled] = useState(false);
  const [settingsReady, setSettingsReady] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [succeeded, setSucceeded] = useState(false);
  const rootRef = useRef<HTMLDivElement>(null);
  const cardRef = useRef<HTMLDivElement>(null);
  const tilt = useRef<{ x: (value: number) => void; y: (value: number) => void } | null>(null);

  useEffect(() => {
    const lastUsername = localStorage.getItem("mars:last-username");
    // Hydration-safe restoration of the non-sensitive username hint.
    // eslint-disable-next-line react-hooks/set-state-in-effect
    if (lastUsername) setForm((current) => ({ ...current, username: lastUsername }));
    let active = true;
    getPublicSettings()
      .then((settings) => {
        if (!active) return;
        const nestedLogin = settings.login;
        const nestedEnabled =
          nestedLogin && typeof nestedLogin === "object"
            ? (nestedLogin as Record<string, unknown>).captchaEnabled
            : undefined;
        const enabled = Boolean(
          settings["security.auth.captcha_enabled"] ?? nestedEnabled ?? false,
        );
        setCaptchaEnabled(enabled);
        setSettingsReady(true);
        if (enabled) void refreshCaptcha();
      })
      .catch(() => {
        if (!active) return;
        setCaptchaEnabled(true);
        setSettingsReady(true);
        void refreshCaptcha();
      });
    return () => {
      active = false;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useLayoutEffect(() => {
    const root = rootRef.current;
    if (!root) return;
    const targets = root.querySelectorAll<HTMLElement>(
      ".brand-top, .brand-copy, .journey, .brand-footer, .login-card",
    );
    if (prefersReducedMotion()) {
      root.classList.add("is-entrance-ready");
      if (styles.isEntranceReady) root.classList.add(styles.isEntranceReady);
      return;
    }

    gsap.set(targets, { opacity: 0, y: 16 });
    root.classList.add("is-entrance-ready");
    if (styles.isEntranceReady) root.classList.add(styles.isEntranceReady);
    let seen = false;
    try {
      seen = sessionStorage.getItem("mars:login-intro") === "1";
      sessionStorage.setItem("mars:login-intro", "1");
    } catch {
      // Session storage is optional.
    }
    const timeline = gsap.timeline();
    timeline.to(root.querySelectorAll(".brand-top"), {
      opacity: 1,
      y: 0,
      duration: 0.7,
      ease: EASE.premium,
    });
    timeline.to(
      root.querySelectorAll(".brand-copy"),
      {
        opacity: 1,
        y: 0,
        duration: 0.9,
        ease: EASE.premium,
        stagger: 0.08,
      },
      0.12,
    );
    timeline.to(
      root.querySelectorAll(".journey"),
      {
        opacity: 1,
        y: 0,
        duration: 0.7,
        ease: EASE.premium,
        stagger: 0.06,
      },
      0.36,
    );
    timeline.to(
      root.querySelectorAll(".brand-footer"),
      {
        opacity: 1,
        y: 0,
        duration: 0.7,
        ease: EASE.premium,
        stagger: 0.08,
      },
      0.5,
    );
    timeline.to(
      root.querySelectorAll(".login-card"),
      {
        opacity: 1,
        y: 0,
        duration: 0.8,
        ease: EASE.glide,
      },
      0.22,
    );
    if (seen) timeline.timeScale(2.4);

    const fastForward = () => timeline.timeScale(6);
    window.addEventListener("pointerdown", fastForward, { once: true });
    window.addEventListener("keydown", fastForward, { once: true });
    return () => {
      window.removeEventListener("pointerdown", fastForward);
      window.removeEventListener("keydown", fastForward);
      timeline.revert();
    };
  }, []);

  useEffect(() => {
    const card = cardRef.current;
    if (!card || prefersReducedMotion() || !window.matchMedia("(hover: hover)").matches) return;
    const x = gsap.quickTo(card, "rotationX", { duration: 0.6, ease: "power3.out" });
    const y = gsap.quickTo(card, "rotationY", { duration: 0.6, ease: "power3.out" });
    tilt.current = { x, y };
    return () => {
      tilt.current = null;
      gsap.killTweensOf(card);
    };
  }, []);

  function onTilt(event: ReactPointerEvent<HTMLElement>) {
    const card = cardRef.current;
    if (!card || !tilt.current || prefersReducedMotion()) return;
    const rect = card.getBoundingClientRect();
    const px = (event.clientX - rect.left) / rect.width - 0.5;
    const py = (event.clientY - rect.top) / rect.height - 0.5;
    tilt.current.y(Math.max(-1, Math.min(1, px * 2)) * 4);
    tilt.current.x(Math.max(-1, Math.min(1, py * 2)) * -3);
  }

  function resetTilt() {
    tilt.current?.x(0);
    tilt.current?.y(0);
  }

  function trackGlow(event: ReactPointerEvent<HTMLDivElement>) {
    const card = event.currentTarget;
    const rect = card.getBoundingClientRect();
    card.style.setProperty("--px", `${event.clientX - rect.left}px`);
    card.style.setProperty("--py", `${event.clientY - rect.top}px`);
  }

  async function playSuccess() {
    setSucceeded(true);
    await new Promise<void>((resolve) => requestAnimationFrame(() => resolve()));
    const orb = cardRef.current?.querySelector<HTMLElement>(".success-orb");
    if (!orb || prefersReducedMotion()) return;
    const ring = orb.querySelector<SVGCircleElement>("circle");
    const check = orb.querySelector<SVGPathElement>("path");
    await new Promise<void>((resolve) => {
      const timeline = gsap.timeline({ onComplete: resolve });
      timeline.fromTo(
        orb,
        { width: orb.parentElement?.clientWidth ?? 46 },
        { width: 46, duration: 0.42, ease: EASE.glide },
      );
      if (ring) {
        timeline.fromTo(
          ring,
          { strokeDashoffset: 126 },
          { strokeDashoffset: 0, duration: 0.5, ease: EASE.premium },
          "-=0.08",
        );
      }
      if (check) {
        timeline.fromTo(
          check,
          { strokeDashoffset: 24 },
          { strokeDashoffset: 0, duration: 0.32, ease: EASE.settle },
          "-=0.22",
        );
      }
      timeline
        .to(orb, { scale: 1.06, duration: 0.16, ease: "power1.out" })
        .to(orb, { scale: 1, duration: 0.3, ease: EASE.settle });
    });
  }

  async function refreshCaptcha() {
    setCaptchaLoading(true);
    try {
      setCaptcha(await issueCaptcha());
    } catch {
      setCaptcha(null);
      pushToast({ title: "无法连接认证服务", message: "请确认 Go 后端已启动", tone: "danger" });
    } finally {
      setCaptchaLoading(false);
    }
  }

  function update(key: keyof FormState, value: string) {
    setForm((current) => ({ ...current, [key]: value }));
    setErrors((current) => ({ ...current, [key]: undefined }));
  }

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const nextErrors: Partial<Record<keyof FormState, string>> = {};
    if (!form.username.trim()) nextErrors.username = "请输入账号";
    if (!form.password) nextErrors.password = "请输入密码";
    if (captchaEnabled && !form.captcha.trim()) nextErrors.captcha = "请输入验证码";
    setErrors(nextErrors);
    if (Object.keys(nextErrors).length) return;

    setSubmitting(true);
    try {
      await login({
        username: form.username.trim(),
        password: form.password,
        captchaId: captchaEnabled ? captcha?.uuid : undefined,
        captchaCode: captchaEnabled ? form.captcha.trim() : undefined,
        clientType: "admin",
        rememberMe: remember,
      });
      if (remember) localStorage.setItem("mars:last-username", form.username.trim());
      else localStorage.removeItem("mars:last-username");
      const params = new URLSearchParams(window.location.search);
      const redirect = params.get("redirect");
      const destination =
        redirect &&
        redirect.startsWith("/") &&
        !redirect.startsWith("//") &&
        !redirect.includes("\\")
          ? redirect
          : "/dashboard";
      void playSuccess();
      router.replace(destination);
    } catch (error) {
      setSucceeded(false);
      const message = error instanceof ApiError ? error.message : "登录失败，请稍后重试";
      pushToast({ title: "登录失败", message, tone: "danger" });
      setForm((current) => ({ ...current, captcha: "" }));
      if (captchaEnabled) await refreshCaptcha();
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className={`${styles.loginStage} login-stage`}>
      <div ref={rootRef} className={`login login-pearl ${styles.loginPearl}`}>
        <LoginBrandPanel />
        <section
          className={`login-side ${styles.loginSide}`}
          aria-labelledby="login-heading"
          onPointerMove={onTilt}
          onPointerLeave={resetTilt}
        >
          <div
            ref={cardRef}
            className={`login-card m-float m-glow ${styles.loginCard}`}
            onPointerMove={trackGlow}
          >
            <span className="glow-layer" aria-hidden="true" />
            <div className="login-card-inner">
              <span className="eyebrow">账号登录</span>
              <h2 id="login-heading" className="login-card-title">
                登录
              </h2>
              <p className="login-card-sub">使用管理员分配的账号登录系统</p>
              <form className="login-form" noValidate onSubmit={submit}>
                <Field label="账号" error={errors.username}>
                  <TextInput
                    icon="user"
                    placeholder="请输入账号"
                    autoComplete="username"
                    value={form.username}
                    invalid={Boolean(errors.username)}
                    onChange={(event) => update("username", event.target.value)}
                  />
                </Field>
                <Field label="密码" error={errors.password}>
                  <TextInput
                    icon="lock"
                    type="password"
                    placeholder="请输入密码"
                    autoComplete="current-password"
                    value={form.password}
                    invalid={Boolean(errors.password)}
                    onChange={(event) => update("password", event.target.value)}
                  />
                </Field>
                {captchaEnabled ? (
                  <Field label="验证码" error={errors.captcha}>
                    <div className="captcha-row">
                      <TextInput
                        icon="shield"
                        placeholder="请输入计算结果"
                        maxLength={6}
                        autoComplete="off"
                        value={form.captcha}
                        invalid={Boolean(errors.captcha)}
                        onChange={(event) => update("captcha", event.target.value)}
                      />
                      <button
                        type="button"
                        className="captcha-box m-groove"
                        disabled={captchaLoading}
                        aria-label="刷新验证码"
                        title="刷新验证码"
                        onClick={() => void refreshCaptcha()}
                      >
                        {captcha?.img ? (
                          // The captcha image is generated by the Go backend, not a remote asset.
                          // eslint-disable-next-line @next/next/no-img-element
                          <img
                            className="captcha-img captcha-img--single"
                            src={captcha.img}
                            alt=""
                          />
                        ) : (
                          <RefreshCw size={16} className={cn(captchaLoading && "spin")} />
                        )}
                      </button>
                    </div>
                  </Field>
                ) : null}
                <div className="login-options">
                  <Checkbox checked={remember} onChange={setRemember} label="记住账号" />
                </div>
                <div className="login-submit-slot">
                  {succeeded ? (
                    <div className="success-orb" role="status" aria-label="登录成功，正在进入">
                      <svg width="30" height="30" viewBox="0 0 46 46" aria-hidden="true">
                        <circle cx="23" cy="23" r="20" strokeDasharray="126" />
                        <path d="m15 23.5 5.5 5.5L31.5 18" strokeDasharray="24" />
                      </svg>
                    </div>
                  ) : (
                    <Button
                      type="submit"
                      variant="primary"
                      block
                      loading={submitting || !settingsReady}
                      className="login-submit"
                    >
                      登录
                    </Button>
                  )}
                </div>
              </form>
            </div>
          </div>
        </section>
      </div>
    </div>
  );
}
