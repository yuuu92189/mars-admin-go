"use client";

import { useEffect, useId, useRef, useState } from "react";
import { Check, ChevronDown } from "lucide-react";
import { cn } from "@/lib/utils/cn";

export type SelectOption = { value: string; label: string; disabled?: boolean };

export function Select({
  value,
  onChange,
  options,
  placeholder = "请选择",
  disabled = false,
  invalid = false,
  filterable = false,
}: {
  value?: string;
  onChange: (value: string) => void;
  options: SelectOption[];
  placeholder?: string;
  disabled?: boolean;
  invalid?: boolean;
  filterable?: boolean;
}) {
  const id = useId();
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const rootRef = useRef<HTMLDivElement>(null);
  const selected = options.find((option) => option.value === value);
  const filtered = options.filter((option) =>
    option.label.toLocaleLowerCase().includes(query.toLocaleLowerCase()),
  );

  useEffect(() => {
    if (!open) return;
    const onPointerDown = (event: PointerEvent) => {
      if (!rootRef.current?.contains(event.target as Node)) setOpen(false);
    };
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") setOpen(false);
    };
    document.addEventListener("pointerdown", onPointerDown);
    document.addEventListener("keydown", onKeyDown);
    return () => {
      document.removeEventListener("pointerdown", onPointerDown);
      document.removeEventListener("keydown", onKeyDown);
    };
  }, [open]);

  return (
    <div className="select-wrap" ref={rootRef}>
      <button
        id={id}
        type="button"
        className={cn("input-shell select-trigger", invalid && "is-invalid")}
        aria-haspopup="listbox"
        aria-expanded={open}
        disabled={disabled}
        onClick={() => setOpen((current) => !current)}
      >
        <span className={cn("select-value", !selected && "is-placeholder")}>
          {selected?.label ?? placeholder}
        </span>
        <ChevronDown className="select-caret" size={15} />
      </button>
      {open ? (
        <div className="select-menu m-float" role="listbox" aria-labelledby={id}>
          {filterable ? (
            <div className="select-search">
              <div className="input-shell">
                <input
                  autoFocus
                  value={query}
                  placeholder="筛选"
                  onChange={(event) => setQuery(event.target.value)}
                />
              </div>
            </div>
          ) : null}
          <div className="select-options">
            {filtered.length ? (
              filtered.map((option) => (
                <button
                  type="button"
                  className={cn(
                    "select-option",
                    option.value === value && "is-selected",
                    option.disabled && "is-disabled",
                  )}
                  key={option.value}
                  role="option"
                  aria-selected={option.value === value}
                  disabled={option.disabled}
                  onClick={() => {
                    onChange(option.value);
                    setOpen(false);
                    setQuery("");
                  }}
                >
                  <span className="opt-label">{option.label}</span>
                  {option.value === value ? <Check size={14} /> : null}
                </button>
              ))
            ) : (
              <div className="select-empty">没有匹配项</div>
            )}
          </div>
        </div>
      ) : null}
    </div>
  );
}

export function Segmented({
  value,
  onChange,
  options,
  label,
}: {
  value: string;
  onChange: (value: string) => void;
  options: Array<{ value: string; label: string }>;
  label: string;
}) {
  return (
    <div className="segmented m-groove" role="group" aria-label={label}>
      {options.map((option) => (
        <button
          type="button"
          key={option.value}
          className={cn(option.value === value && "is-active")}
          aria-pressed={option.value === value}
          onClick={() => onChange(option.value)}
        >
          {option.label}
        </button>
      ))}
    </div>
  );
}
