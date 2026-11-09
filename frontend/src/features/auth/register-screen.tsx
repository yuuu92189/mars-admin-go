"use client";

import { useCallback, useState, type FormEvent } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { RefreshCw } from "lucide-react";
import { LoginBrandPanel } from "./login-brand-panel";
import { Button } from "@/components/ui/button";
import { Checkbox, Field, TextInput } from "@/components/ui/input";
import { useToast } from "@/components/providers/toast-provider";
import { ApiError } from "@/lib/api/client";
import { getPublicSettings, issueCaptcha, register } from "@/lib/api/auth";
import type { CaptchaResponse } from "@/lib/api/types";
import { useDeferredEffect } from "@/lib/use-deferred-effect";
import { booleanPublicSetting, isRegistrationEnabled, publicSetting } from "./public-settings";
import styles from "./register-screen.module.css";

type FormState = {
  username: string;
  displayName: string;
  password: string;
  confirmPassword: string;
  email: string;
  phone: string;
  captcha: string;
};

type PublicRegistrationSettings = {
  enabled: boolean;
  verifyEmail: boolean;
  verifyPhone: boolean;
  captchaEnabled: boolean;
  minPasswordLength: number;
};

function readSettings(settings: Record<string, unknown>): PublicRegistrationSettings {
  return {
    enabled: isRegistrationEnabled(settings),
    verifyEmail: booleanPublicSetting(settings, ["register.verifyEmail", "register.verify_email"]),
    verifyPhone: booleanPublicSetting(settings, ["register.verifyPhone", "register.verify_phone"]),
    captchaEnabled: booleanPublicSetting(settings, ["security.auth.captcha_enabled"]),
    minPasswordLength: Number(publicSetting(settings, ["password.minLength"]) ?? 8),
  };
}

export function RegisterScreen() {
  const router = useRouter();
  const { pushToast } = useToast();
  const [form, setForm] = useState<FormState>({
    username: "",
    displayName: "",
    password: "",
    confirmPassword: "",
    email: "",
    phone: "",
    captcha: "",
  });
  const [errors, setErrors] = useState<Partial<Record<keyof FormState, string>>>({});
  const [settings, setSettings] = useState<PublicRegistrationSettings | null>(null);
  const [settingsError, setSettingsError] = useState<string | null>(null);
  const [captcha, setCaptcha] = useState<CaptchaResponse | null>(null);
  const [captchaLoading, setCaptchaLoading] = useState(false);
  const [agreed, setAgreed] = useState(false);
  const [submitting, setSubmitting] = useState(false);

  const loadSettings = useCallback(async () => {
    setSettingsError(null);
    try {
      const result = await getPublicSettings();
      const next = readSettings(result);
      setSettings(next);
      if (next.enabled && next.captchaEnabled) {
        setCaptchaLoading(true);
        try {
          setCaptcha(await issueCaptcha());
        } catch {
          setCaptcha(null);
          pushToast({ title: "验证码加载失败", message: "请稍后重试", tone: "danger" });
        } finally {
          setCaptchaLoading(false);
        }
      }
    } catch {
      setSettings(null);
      setSettingsError("注册设置暂时不可用，请稍后重试。");
    }
  }, [pushToast]);

  async function refreshCaptcha() {
    setCaptchaLoading(true);
    try {
      setCaptcha(await issueCaptcha());
    } catch {
      setCaptcha(null);
      pushToast({ title: "验证码加载失败", message: "请稍后重试", tone: "danger" });
    } finally {
      setCaptchaLoading(false);
    }
  }

  useDeferredEffect(loadSettings);

  function update(key: keyof FormState, value: string) {
    setForm((current) => ({ ...current, [key]: value }));
    setErrors((current) => ({ ...current, [key]: undefined }));
  }

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const nextErrors: Partial<Record<keyof FormState, string>> = {};
    if (!form.username.trim()) nextErrors.username = "请输入用户名";
    if (!form.displayName.trim()) nextErrors.displayName = "请输入昵称";
    if (form.password.length < (settings?.minPasswordLength ?? 8)) {
      nextErrors.password = `密码至少 ${settings?.minPasswordLength ?? 8} 位`;
    }
    if (form.password !== form.confirmPassword) nextErrors.confirmPassword = "两次密码不一致";
    if (settings?.verifyEmail && !form.email.trim()) nextErrors.email = "请输入邮箱";
    if (settings?.verifyPhone && !form.phone.trim()) nextErrors.phone = "请输入手机号";
    if (settings?.captchaEnabled && !form.captcha.trim()) nextErrors.captcha = "请输入验证码";
    if (!agreed) nextErrors.confirmPassword = "请先同意用户协议和隐私政策";
    setErrors(nextErrors);
    if (Object.keys(nextErrors).length) return;

    setSubmitting(true);
    try {
      await register({
        username: form.username.trim(),
        password: form.password,
        displayName: form.displayName.trim(),
        email: form.email.trim() || undefined,
        phone: form.phone.trim() || undefined,
        userType: "pc",
        captchaId: settings?.captchaEnabled ? captcha?.uuid : undefined,
        captchaCode: settings?.captchaEnabled ? form.captcha.trim() : undefined,
      });
      pushToast({ title: "注册成功", message: "账号已创建，请登录", tone: "ok" });
      router.replace("/login");
    } catch (error) {
      const message = error instanceof ApiError ? error.message : "注册失败，请稍后重试";
      pushToast({ title: "注册失败", message, tone: "danger" });
      if (settings?.captchaEnabled) {
        setForm((current) => ({ ...current, captcha: "" }));
        await refreshCaptcha();
      }
    } finally {
      setSubmitting(false);
    }
  }

  const unavailable = settingsError || (settings && !settings.enabled);

  return (
    <div className={`${styles.loginStage} login-stage`}>
      <div className={`login login-pearl ${styles.loginPearl} is-entrance-ready`}>
        <LoginBrandPanel />
        <section className={`login-side ${styles.loginSide}`} aria-labelledby="register-heading">
          <div className={`login-card m-float ${styles.loginCard} ${styles.registerCard}`}>
            <div className="login-card-inner">
              <span className="eyebrow">创建账号</span>
              <h1 id="register-heading" className="login-card-title">
                注册
              </h1>
              <p className="login-card-sub">填写账号信息，注册结果由 Go 后端策略校验。</p>
              {settingsError ? (
                <div className={styles.notice} role="status">
                  <span>{settingsError}</span>
                  <Button
                    type="button"
                    variant="text"
                    icon="refresh"
                    onClick={() => void loadSettings()}
                  >
                    重试
                  </Button>
                </div>
              ) : settings && !settings.enabled ? (
                <div className={styles.notice} role="status">
                  <strong>系统暂未开放注册</strong>
                  <span>请联系管理员开通账号，或返回登录。</span>
                </div>
              ) : null}
              {!unavailable ? (
                <form className={`login-form ${styles.registerForm}`} noValidate onSubmit={submit}>
                  <Field label="用户名" required error={errors.username}>
                    <TextInput
                      icon="user"
                      placeholder="4-20 位字母、数字或下划线"
                      autoComplete="username"
                      value={form.username}
                      onChange={(event) => update("username", event.target.value)}
                    />
                  </Field>
                  <Field label="昵称" required error={errors.displayName}>
                    <TextInput
                      icon="user"
                      placeholder="用于界面展示"
                      value={form.displayName}
                      onChange={(event) => update("displayName", event.target.value)}
                    />
                  </Field>
                  <div className={styles.twoColumns}>
                    <Field label="密码" required error={errors.password}>
                      <TextInput
                        icon="lock"
                        type="password"
                        placeholder="请输入密码"
                        autoComplete="new-password"
                        value={form.password}
                        onChange={(event) => update("password", event.target.value)}
                      />
                    </Field>
                    <Field label="确认密码" required error={errors.confirmPassword}>
                      <TextInput
                        icon="lock"
                        type="password"
                        placeholder="再次输入密码"
                        autoComplete="new-password"
                        value={form.confirmPassword}
                        onChange={(event) => update("confirmPassword", event.target.value)}
                      />
                    </Field>
                  </div>
                  {settings?.verifyEmail || settings?.verifyPhone ? (
                    <div className={styles.twoColumns}>
                      {settings.verifyEmail ? (
                        <Field label="邮箱" required error={errors.email}>
                          <TextInput
                            icon="mail"
                            type="email"
                            placeholder="请输入邮箱"
                            autoComplete="email"
                            value={form.email}
                            onChange={(event) => update("email", event.target.value)}
                          />
                        </Field>
                      ) : null}
                      {settings.verifyPhone ? (
                        <Field label="手机号" required error={errors.phone}>
                          <TextInput
                            icon="phone"
                            type="tel"
                            placeholder="请输入手机号"
                            autoComplete="tel"
                            value={form.phone}
                            onChange={(event) => update("phone", event.target.value)}
                          />
                        </Field>
                      ) : null}
                    </div>
                  ) : null}
                  {settings?.captchaEnabled ? (
                    <Field label="验证码" required error={errors.captcha}>
                      <div className="captcha-row">
                        <TextInput
                          icon="shield"
                          placeholder="请输入验证码"
                          value={form.captcha}
                          onChange={(event) => update("captcha", event.target.value)}
                        />
                        <button
                          type="button"
                          className="captcha-box m-groove"
                          aria-label="刷新验证码"
                          disabled={captchaLoading}
                          onClick={() => void refreshCaptcha()}
                        >
                          {captcha?.img ? (
                            // The image is generated by the Go captcha endpoint.
                            // eslint-disable-next-line @next/next/no-img-element
                            <img className="captcha-img" src={captcha.img} alt="" />
                          ) : (
                            <RefreshCw size={16} className={captchaLoading ? "spin" : undefined} />
                          )}
                        </button>
                      </div>
                    </Field>
                  ) : null}
                  <Checkbox
                    checked={agreed}
                    onChange={setAgreed}
                    label="我已阅读并同意用户协议和隐私政策"
                  />
                  <Button type="submit" variant="primary" block loading={submitting}>
                    注册
                  </Button>
                </form>
              ) : null}
              <div className={styles.footer}>
                已有账号？
                <Link href="/login">立即登录</Link>
              </div>
            </div>
          </div>
        </section>
      </div>
    </div>
  );
}
