"use client";

import { useEffect, useRef } from "react";
import { prefersReducedMotion } from "@/lib/motion";

export function NeuralNet({ className }: { className?: string }) {
  const ref = useRef<HTMLCanvasElement | null>(null);

  useEffect(() => {
    const canvas = ref.current;
    if (!canvas) return;
    const context = canvas.getContext("2d");
    if (!context) return;
    let frame = 0;
    let width = 0;
    let height = 0;
    const points = Array.from({ length: 18 }, (_, index) => ({
      x: (index * 0.137 + 0.08) % 1,
      y: (index * 0.239 + 0.16) % 1,
      phase: index * 0.7,
    }));
    const resize = () => {
      const rect = canvas.getBoundingClientRect();
      const dpr = Math.min(window.devicePixelRatio || 1, 2);
      width = rect.width;
      height = rect.height;
      canvas.width = Math.max(1, Math.floor(width * dpr));
      canvas.height = Math.max(1, Math.floor(height * dpr));
      context.setTransform(dpr, 0, 0, dpr, 0, 0);
    };
    const draw = (time: number) => {
      context.clearRect(0, 0, width, height);
      const animated = !prefersReducedMotion();
      points.forEach((point, index) => {
        const drift = animated ? Math.sin(time / 2200 + point.phase) * 5 : 0;
        const x = point.x * width + drift;
        const y = point.y * height + Math.cos(time / 2800 + point.phase) * 4;
        points.slice(index + 1, index + 4).forEach((other) => {
          const otherX = other.x * width;
          const otherY = other.y * height;
          const distance = Math.hypot(otherX - x, otherY - y);
          if (distance < Math.min(width, height) * 0.3) {
            context.beginPath();
            context.moveTo(x, y);
            context.lineTo(otherX, otherY);
            context.strokeStyle = "rgba(130, 221, 45, 0.12)";
            context.lineWidth = 0.8;
            context.stroke();
          }
        });
        context.beginPath();
        context.arc(x, y, 2.2, 0, Math.PI * 2);
        context.fillStyle =
          index % 3 === 0 ? "rgba(130, 221, 45, 0.72)" : "rgba(255, 255, 255, 0.42)";
        context.fill();
      });
      if (animated) frame = requestAnimationFrame(draw);
    };
    resize();
    const observer = new ResizeObserver(resize);
    observer.observe(canvas);
    draw(0);
    return () => {
      cancelAnimationFrame(frame);
      observer.disconnect();
    };
  }, []);

  return <canvas ref={ref} className={className} aria-hidden="true" />;
}
