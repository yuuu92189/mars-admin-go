import {
  forwardRef,
  useId,
  type InputHTMLAttributes,
  type ReactNode,
  type TextareaHTMLAttributes,
} from "react";
import { X } from "lucide-react";
import { cn } from "@/lib/utils/cn";
import { AppIcon } from "./app-icon";

export type TextInputProps = Omit<InputHTMLAttributes<HTMLInputElement>, "size"> & {
  icon?: string;
  clearable?: boolean;
  multiline?: boolean;
  rows?: number;
  invalid?: boolean;
  onClear?: () => void;
};

export const TextInput = forwardRef<HTMLInputElement, TextInputProps>(function TextInput(
  { icon, clearable, multiline, rows = 4, invalid, onClear, className, value, ...props },
  ref,
) {
  if (multiline) {
    return (
      <div className={cn("input-shell", "is-multiline", invalid && "is-invalid", className)}>
        <textarea
          {...(props as TextareaHTMLAttributes<HTMLTextAreaElement>)}
          rows={rows}
          value={value}
        />
      </div>
    );
  }

  return (
    <div className={cn("input-shell", invalid && "is-invalid", className)}>
      {icon ? <AppIcon name={icon} size={16} /> : null}
      <input ref={ref} value={value} {...props} />
      {clearable && value ? (
        <button type="button" className="input-clear" aria-label="清除输入" onClick={onClear}>
          <X size={14} />
        </button>
      ) : null}
    </div>
  );
});

export function Field({
  label,
  error,
  hint,
  required,
  wide,
  children,
  className,
}: {
  label: string;
  error?: string;
  hint?: string;
  required?: boolean;
  wide?: boolean;
  children: ReactNode;
  className?: string;
}) {
  return (
    <label className={cn("field", wide && "is-wide", className)}>
      <span className="field-label">
        {label}
        {required ? <span className="req">*</span> : null}
      </span>
      {children}
      {error ? (
        <span className="field-error">{error}</span>
      ) : hint ? (
        <span className="field-hint">{hint}</span>
      ) : null}
    </label>
  );
}

export function Checkbox({
  checked,
  onChange,
  label,
  disabled = false,
}: {
  checked: boolean;
  onChange: (checked: boolean) => void;
  label: string;
  disabled?: boolean;
}) {
  const id = useId();
  return (
    <label className={cn("checkbox", checked && "is-on", disabled && "is-disabled")} htmlFor={id}>
      <input
        id={id}
        type="checkbox"
        checked={checked}
        disabled={disabled}
        onChange={(event) => onChange(event.target.checked)}
      />
      <span className="checkbox-box" aria-hidden="true">
        {checked ? <AppIcon name="check" size={12} strokeWidth={3} /> : null}
      </span>
      <span>{label}</span>
    </label>
  );
}

export function Switch({
  checked,
  onChange,
  label,
  disabled = false,
}: {
  checked: boolean;
  onChange: (checked: boolean) => void;
  label?: string;
  disabled?: boolean;
}) {
  const id = useId();
  return (
    <label className={cn("switch", checked && "is-on", disabled && "is-disabled")} htmlFor={id}>
      <input
        id={id}
        type="checkbox"
        checked={checked}
        disabled={disabled}
        onChange={(event) => onChange(event.target.checked)}
      />
      <span className="switch-track" aria-hidden="true">
        <span className="switch-knob" />
      </span>
      {label ? <span>{label}</span> : null}
    </label>
  );
}
