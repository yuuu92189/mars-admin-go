"use client";

import { useCallback, useRef, useState, type ChangeEvent, type ReactNode } from "react";
import { Camera, Check, LoaderCircle, Mail, Phone, ShieldCheck, UserRound } from "lucide-react";
import { PageContainer } from "@/components/layout/page-container";
import { useAuth } from "@/components/auth/auth-provider";
import { useToast } from "@/components/providers/toast-provider";
import { Button } from "@/components/ui/button";
import { Field, TextInput } from "@/components/ui/input";
import { Segmented } from "@/components/ui/select";
import { Avatar, Card, PageHeader, Tag } from "@/components/ui/surface";
import { ApiError } from "@/lib/api/client";
import { getProfile, updateProfile } from "@/lib/api/auth";
import { uploadFile } from "@/lib/api/storage";
import type { User } from "@/lib/api/types";
import { useDeferredEffect } from "@/lib/use-deferred-effect";
import styles from "./profile-view.module.css";

type ProfileForm = {
  displayName: string;
  email: string;
  phone: string;
  avatarUrl: string;
  gender: string;
  userType: string;
  status: string;
  version: number;
};

function formFromUser(user: User): ProfileForm {
  return {
    displayName: user.displayName ?? "",
    email: user.email ?? "",
    phone: user.phone ?? "",
    avatarUrl: user.avatarUrl ?? "",
    gender: String(user.gender ?? 0),
    userType: user.userType,
    status: user.status,
    version: user.version ?? 0,
  };
}

function userTypeLabel(value: string): string {
  return { admin: "后台管理员", pc: "PC 前台用户", app: "App 用户" }[value] ?? value;
}

function genderLabel(value: string): string {
  return { "0": "保密", "1": "男", "2": "女" }[value] ?? "保密";
}

function displayError(error: unknown): string {
  return error instanceof ApiError ? error.message : "请求失败，请稍后重试";
}

export function ProfileView() {
  const { user: currentUser, refreshUser } = useAuth();
  const { pushToast } = useToast();
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [profile, setProfile] = useState<User | null>(currentUser?.user ?? null);
  const [form, setForm] = useState<ProfileForm | null>(
    currentUser?.user ? formFromUser(currentUser.user) : null,
  );
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [errors, setErrors] = useState<Partial<Record<keyof ProfileForm, string>>>({});

  const loadProfile = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const result = await getProfile();
      setProfile(result);
      setForm(formFromUser(result));
    } catch (cause) {
      setError(displayError(cause));
    } finally {
      setLoading(false);
    }
  }, []);

  useDeferredEffect(loadProfile);

  function update<K extends keyof ProfileForm>(key: K, value: ProfileForm[K]) {
    setForm((current) => (current ? { ...current, [key]: value } : current));
    setErrors((current) => ({ ...current, [key]: undefined }));
  }

  function validate(): boolean {
    if (!form) return false;
    const next: Partial<Record<keyof ProfileForm, string>> = {};
    if (!form.displayName.trim()) next.displayName = "请输入显示名称";
    if (form.email && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(form.email)) {
      next.email = "请输入正确的邮箱";
    }
    if (form.phone && !/^[+\d][\d\s-]{5,19}$/.test(form.phone)) {
      next.phone = "请输入正确的手机号";
    }
    setErrors(next);
    return Object.keys(next).length === 0;
  }

  async function save() {
    if (!form || !profile || !validate()) return;
    setSaving(true);
    try {
      const updated = await updateProfile({
        displayName: form.displayName.trim(),
        email: form.email.trim(),
        phone: form.phone.trim(),
        avatarUrl: form.avatarUrl.trim(),
        gender: Number(form.gender),
        userType: form.userType,
        status: form.status,
        version: form.version,
      });
      setProfile(updated);
      setForm(formFromUser(updated));
      await refreshUser();
      pushToast({ title: "个人资料已保存", tone: "ok" });
    } catch (cause) {
      pushToast({ title: "保存失败", message: displayError(cause), tone: "danger" });
    } finally {
      setSaving(false);
    }
  }

  function reset() {
    if (profile) {
      setForm(formFromUser(profile));
      setErrors({});
    }
  }

  async function onAvatarChange(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    event.target.value = "";
    if (!file) return;
    if (!file.type.startsWith("image/")) {
      pushToast({ title: "头像格式不支持", message: "请选择图片文件", tone: "danger" });
      return;
    }
    if (file.size > 5 * 1024 * 1024) {
      pushToast({ title: "头像过大", message: "图片不能超过 5 MB", tone: "danger" });
      return;
    }
    setUploading(true);
    try {
      const result = await uploadFile(file);
      update("avatarUrl", result.url);
      pushToast({ title: "头像已上传", message: "保存资料后生效", tone: "ok" });
    } catch (cause) {
      pushToast({ title: "头像上传失败", message: displayError(cause), tone: "danger" });
    } finally {
      setUploading(false);
    }
  }

  return (
    <PageContainer>
      <PageHeader
        title="个人资料"
        description="编辑当前账号资料；显示名称、联系方式与头像由 Go 后端校验并持久化。"
      >
        <Tag tone="info">
          <ShieldCheck size={13} /> 账号安全
        </Tag>
      </PageHeader>

      {error ? (
        <Card className={styles.errorCard}>
          <strong>个人资料加载失败</strong>
          <span>{error}</span>
          <Button variant="secondary" icon="refresh" onClick={() => void loadProfile()}>
            重试
          </Button>
        </Card>
      ) : (
        <div className={styles.layout}>
          <aside className={styles.side}>
            <Card className={styles.identityCard}>
              <button
                type="button"
                className={styles.avatarButton}
                aria-label="上传头像"
                onClick={() => fileInputRef.current?.click()}
                disabled={uploading || !form}
              >
                {form?.avatarUrl ? (
                  // Avatar URLs are returned by the authenticated storage API.
                  // eslint-disable-next-line @next/next/no-img-element
                  <img src={form.avatarUrl} alt="" className={styles.avatarImage} />
                ) : (
                  <Avatar name={form?.displayName || profile?.username} size={92} />
                )}
                <span className={styles.avatarOverlay}>
                  {uploading ? <LoaderCircle className="spin" size={20} /> : <Camera size={20} />}
                </span>
              </button>
              <input
                ref={fileInputRef}
                type="file"
                accept="image/*"
                hidden
                onChange={(event) => void onAvatarChange(event)}
              />
              <h2>{form?.displayName || "未设置昵称"}</h2>
              <span className={styles.username}>@{profile?.username ?? "—"}</span>
              <Tag tone={profile?.status === "active" ? "ok" : "danger"} dot>
                {profile?.status === "active" ? "正常" : (profile?.status ?? "未知")}
              </Tag>
            </Card>
            <Card className={styles.infoCard}>
              <span className="section-label">账号信息</span>
              <InfoRow
                icon={<UserRound size={15} />}
                label="用户类型"
                value={userTypeLabel(profile?.userType ?? "")}
              />
              <InfoRow
                icon={<ShieldCheck size={15} />}
                label="性别"
                value={genderLabel(form?.gender ?? "0")}
              />
              <InfoRow
                icon={<Mail size={15} />}
                label="MFA"
                value={profile?.mfaEnabled ? "已启用" : "未启用"}
              />
              <InfoRow
                icon={<Phone size={15} />}
                label="创建时间"
                value={formatDate(profile?.createdAt)}
              />
            </Card>
          </aside>

          <Card className={styles.formCard}>
            <div className={styles.formHead}>
              <div>
                <span className="section-label">编辑资料</span>
                <p>修改后将立即影响当前账号的展示信息。</p>
              </div>
              {loading ? <LoaderCircle className="spin" size={18} /> : <Check size={18} />}
            </div>
            {form ? (
              <div className={styles.formGrid}>
                <Field label="显示名称" required error={errors.displayName}>
                  <TextInput
                    icon="user"
                    value={form.displayName}
                    onChange={(event) => update("displayName", event.target.value)}
                  />
                </Field>
                <Field label="邮箱" error={errors.email}>
                  <TextInput
                    icon="mail"
                    type="email"
                    value={form.email}
                    onChange={(event) => update("email", event.target.value)}
                  />
                </Field>
                <Field label="手机号" error={errors.phone}>
                  <TextInput
                    icon="phone"
                    type="tel"
                    value={form.phone}
                    onChange={(event) => update("phone", event.target.value)}
                  />
                </Field>
                <Field label="性别">
                  <Segmented
                    value={form.gender}
                    onChange={(value) => update("gender", value)}
                    options={[
                      { value: "0", label: "保密" },
                      { value: "1", label: "男" },
                      { value: "2", label: "女" },
                    ]}
                    label="性别"
                  />
                </Field>
                <div className={styles.formActions}>
                  <Button variant="ghost" icon="refresh" onClick={reset}>
                    重置
                  </Button>
                  <Button
                    variant="primary"
                    icon="save"
                    loading={saving}
                    onClick={() => void save()}
                  >
                    保存修改
                  </Button>
                </div>
              </div>
            ) : (
              <div className={styles.empty}>正在读取个人资料…</div>
            )}
          </Card>
        </div>
      )}
    </PageContainer>
  );
}

function InfoRow({ icon, label, value }: { icon: ReactNode; label: string; value: string }) {
  return (
    <div className={styles.infoRow}>
      <span>
        {icon}
        {label}
      </span>
      <strong>{value}</strong>
    </div>
  );
}

function formatDate(value?: string) {
  if (!value) return "—";
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleDateString("zh-CN");
}
