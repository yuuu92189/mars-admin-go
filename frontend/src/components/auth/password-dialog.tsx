"use client";

import { useEffect, useState, type FormEvent } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Field, TextInput } from "@/components/ui/input";
import { Dialog } from "@/components/ui/overlay";
import { useToast } from "@/components/providers/toast-provider";
import { ApiError } from "@/lib/api/client";
import { changePassword } from "@/lib/api/auth";

type PasswordForm = {
  oldPassword: string;
  newPassword: string;
  confirmPassword: string;
};

const EMPTY_FORM: PasswordForm = {
  oldPassword: "",
  newPassword: "",
  confirmPassword: "",
};

export function PasswordDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  const router = useRouter();
  const { pushToast } = useToast();
  const [form, setForm] = useState<PasswordForm>(EMPTY_FORM);
  const [errors, setErrors] = useState<Partial<Record<keyof PasswordForm, string>>>({});
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (!open) return;
    // Opening a dialog is a user event; reset the previous attempt.
    const timer = window.setTimeout(() => {
      setForm(EMPTY_FORM);
      setErrors({});
    }, 0);
    return () => window.clearTimeout(timer);
  }, [open]);

  const update = (key: keyof PasswordForm, value: string) => {
    setForm((current) => ({ ...current, [key]: value }));
    setErrors((current) => ({ ...current, [key]: undefined }));
  };

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const nextErrors: typeof errors = {};
    if (!form.oldPassword) nextErrors.oldPassword = "请输入当前密码";
    if (!form.newPassword) nextErrors.newPassword = "请输入新密码";
    else if (form.newPassword.length < 8) nextErrors.newPassword = "新密码至少 8 位";
    if (!form.confirmPassword) nextErrors.confirmPassword = "请再次输入新密码";
    else if (form.confirmPassword !== form.newPassword) {
      nextErrors.confirmPassword = "两次输入的密码不一致";
    }
    setErrors(nextErrors);
    if (Object.keys(nextErrors).length) return;

    setSaving(true);
    try {
      await changePassword({
        oldPassword: form.oldPassword,
        newPassword: form.newPassword,
      });
      onClose();
      pushToast({
        title: "密码已修改",
        message: "会话已失效，请使用新密码重新登录",
        tone: "ok",
      });
      router.replace("/login?reason=password-changed");
    } catch (error) {
      pushToast({
        title: "密码修改失败",
        message: error instanceof ApiError ? error.message : "请稍后重试",
        tone: "danger",
      });
    } finally {
      setSaving(false);
    }
  }

  return (
    <Dialog
      open={open}
      title="修改密码"
      description="修改成功后当前会话会被服务端撤销，需要重新登录。"
      onClose={saving ? () => undefined : onClose}
      width={440}
      footer={
        <>
          <Button variant="secondary" disabled={saving} onClick={onClose}>
            取消
          </Button>
          <Button type="submit" form="password-change-form" variant="primary" loading={saving}>
            确认修改
          </Button>
        </>
      }
    >
      <form id="password-change-form" className="form-stack" onSubmit={submit} noValidate>
        <Field label="当前密码" required error={errors.oldPassword}>
          <TextInput
            type="password"
            autoComplete="current-password"
            value={form.oldPassword}
            invalid={Boolean(errors.oldPassword)}
            onChange={(event) => update("oldPassword", event.target.value)}
          />
        </Field>
        <Field label="新密码" required error={errors.newPassword} hint="至少 8 位">
          <TextInput
            type="password"
            autoComplete="new-password"
            data-autofocus
            value={form.newPassword}
            invalid={Boolean(errors.newPassword)}
            onChange={(event) => update("newPassword", event.target.value)}
          />
        </Field>
        <Field label="确认新密码" required error={errors.confirmPassword}>
          <TextInput
            type="password"
            autoComplete="new-password"
            value={form.confirmPassword}
            invalid={Boolean(errors.confirmPassword)}
            onChange={(event) => update("confirmPassword", event.target.value)}
          />
        </Field>
      </form>
    </Dialog>
  );
}
