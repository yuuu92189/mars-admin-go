import { useEffect, useRef } from "react";
import { gsap } from "gsap";

export const EASE = {
  premium: "power3.out",
  settle: "back.out(1.25)",
  glide: "power2.out",
} as const;

export function prefersReducedMotion(): boolean {
  return (
    typeof window !== "undefined" && window.matchMedia("(prefers-reduced-motion: reduce)").matches
  );
}

export function useShellEntrance<T extends HTMLElement>() {
  const ref = useRef<T | null>(null);

  useEffect(() => {
    if (!ref.current || prefersReducedMotion()) return;
    const ctx = gsap.context(() => {
      gsap.fromTo(
        ".rail-brand, .nav-group, .topbar > *",
        { opacity: 0, y: 8 },
        {
          opacity: 1,
          y: 0,
          duration: 0.55,
          stagger: 0.035,
          ease: EASE.premium,
          clearProps: "transform,opacity",
        },
      );
    }, ref.current);
    return () => ctx.revert();
  }, []);

  return ref;
}

export function useCardEntrance<T extends HTMLElement>() {
  const ref = useRef<T | null>(null);

  useEffect(() => {
    if (!ref.current || prefersReducedMotion()) return;
    const ctx = gsap.context(() => {
      gsap.fromTo(
        ".reveal",
        { opacity: 0, y: 12 },
        {
          opacity: 1,
          y: 0,
          duration: 0.55,
          stagger: 0.05,
          ease: EASE.premium,
          clearProps: "transform,opacity",
        },
      );
    }, ref.current);
    return () => ctx.revert();
  }, []);

  return ref;
}
