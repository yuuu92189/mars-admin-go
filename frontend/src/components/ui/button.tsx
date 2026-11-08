import type { ButtonHTMLAttributes, ReactNode } from "react";
import { LoaderCircle } from "lucide-react";
import { cn } from "@/lib/utils/cn";
import { AppIcon } from "./app-icon";

type ButtonVariant = "primary" | "secondary" | "ghost" | "text" | "danger";

export type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: ButtonVariant;
  size?: "default" | "sm";
  block?: boolean;
  loading?: boolean;
  icon?: string;
  children?: ReactNode;
};

export function Button({
  variant = "secondary",
  size = "default",
  block = false,
  loading = false,
  icon,
  className,
  children,
  disabled,
  ...props
}: ButtonProps) {
  return (
    <button
      className={cn(
        "btn",
        `btn-${variant}`,
        size === "sm" && "btn-sm",
        block && "btn-block",
        className,
      )}
      disabled={disabled || loading}
      {...props}
    >
      {loading ? (
        <LoaderCircle className="spinner" size={14} />
      ) : icon ? (
        <AppIcon name={icon} size={15} />
      ) : null}
      {children}
    </button>
  );
}

export type IconButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  label: string;
  icon: string;
  small?: boolean;
  bordered?: boolean;
  danger?: boolean;
};

export function IconButton({
  label,
  icon,
  small = false,
  bordered = false,
  danger = false,
  className,
  ...props
}: IconButtonProps) {
  return (
    <button
      type="button"
      className={cn(
        "icon-btn",
        small && "icon-btn-sm",
        bordered && "is-bordered",
        danger && "is-danger",
        className,
      )}
      aria-label={label}
      title={label}
      {...props}
    >
      <AppIcon name={icon} size={small ? 15 : 17} />
    </button>
  );
}
