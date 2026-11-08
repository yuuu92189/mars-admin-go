"use client";

import { Dialog } from "./overlay";
import { Button } from "./button";

export function ConfirmDialog({
  open,
  title,
  message,
  confirmText = "确认",
  cancelText = "取消",
  danger = false,
  loading = false,
  onConfirm,
  onClose,
}: {
  open: boolean;
  title: string;
  message: string;
  confirmText?: string;
  cancelText?: string;
  danger?: boolean;
  loading?: boolean;
  onConfirm: () => void;
  onClose: () => void;
}) {
  return (
    <Dialog
      open={open}
      title={title}
      description={message}
      onClose={loading ? () => undefined : onClose}
      width={430}
      footer={
        <>
          <Button variant="secondary" disabled={loading} onClick={onClose}>
            {cancelText}
          </Button>
          <Button variant={danger ? "danger" : "primary"} loading={loading} onClick={onConfirm}>
            {confirmText}
          </Button>
        </>
      }
    >
      <span />
    </Dialog>
  );
}
