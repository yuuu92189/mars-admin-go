"use client";

import {
  createContext,
  useCallback,
  useContext,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from "react";
import { X } from "lucide-react";
import { cn } from "@/lib/utils/cn";

export type ToastTone = "info" | "ok" | "danger";
export type ToastInput = {
  title: string;
  message?: string;
  tone?: ToastTone;
};
type ToastItem = ToastInput & { id: string; tone: ToastTone };

type ToastContextValue = {
  pushToast: (toast: ToastInput) => void;
};

const ToastContext = createContext<ToastContextValue | null>(null);

export function ToastProvider({ children }: { children: ReactNode }) {
  const [items, setItems] = useState<ToastItem[]>([]);
  const timers = useRef(new Map<string, ReturnType<typeof setTimeout>>());

  const remove = useCallback((id: string) => {
    setItems((current) => current.filter((item) => item.id !== id));
    const timer = timers.current.get(id);
    if (timer) clearTimeout(timer);
    timers.current.delete(id);
  }, []);

  const pushToast = useCallback(
    (toast: ToastInput) => {
      const id = crypto.randomUUID();
      const item: ToastItem = { ...toast, id, tone: toast.tone ?? "info" };
      setItems((current) => [...current.slice(-2), item]);
      timers.current.set(
        id,
        setTimeout(() => remove(id), 4200),
      );
    },
    [remove],
  );

  const value = useMemo(() => ({ pushToast }), [pushToast]);

  return (
    <ToastContext.Provider value={value}>
      {children}
      <div className="toast-host" aria-live="polite" aria-atomic="false">
        {items.map((item, index) => (
          <div
            className="toast-slot"
            key={item.id}
            style={{ ["--depth" as string]: String(items.length - index - 1) }}
          >
            <div className={cn("toast m-float", `tone-${item.tone}`)} role="status">
              <span className="toast-dot" aria-hidden="true" />
              <span className="toast-msg">
                <strong>{item.title}</strong>
                {item.message ? <span> · {item.message}</span> : null}
              </span>
              <button
                className="toast-close"
                type="button"
                aria-label="关闭提示"
                onClick={() => remove(item.id)}
              >
                <X size={14} />
              </button>
            </div>
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  );
}

export function useToast(): ToastContextValue {
  const value = useContext(ToastContext);
  if (!value) throw new Error("useToast must be used inside ToastProvider");
  return value;
}
