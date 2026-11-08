import { cn } from "@/lib/utils/cn";

export function BrandMark({ className }: { className?: string }) {
  return (
    <span className={cn("brand-mark", className)} aria-hidden="true">
      <svg viewBox="0 0 24 24">
        <path
          d="M16.88 7.31 18.5 8.25v7.5L12 19.5l-6.5-3.75v-7.5L12 4.5l2.25 1.13"
          fill="none"
          stroke="rgba(255, 255, 255, .85)"
          strokeWidth="1.4"
          strokeLinecap="round"
          strokeLinejoin="round"
        />
        <circle cx="15.6" cy="6.5" r="1.5" fill="#82dd2d" />
      </svg>
    </span>
  );
}
