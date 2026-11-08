"use client";

import { useEffect } from "react";

export function useDeferredEffect(effect: () => void | Promise<void>): void {
  useEffect(() => {
    const timer = window.setTimeout(() => {
      void effect();
    }, 0);
    return () => window.clearTimeout(timer);
  }, [effect]);
}
