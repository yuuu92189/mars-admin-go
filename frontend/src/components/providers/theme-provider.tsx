"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";

export type Theme = "light" | "dark" | "system";
export type Density = "comfortable" | "compact";

type ThemeContextValue = {
  theme: Theme;
  resolvedTheme: "light" | "dark";
  density: Density;
  setTheme: (theme: Theme) => void;
  setDensity: (density: Density) => void;
  cycleTheme: () => void;
};

const ThemeContext = createContext<ThemeContextValue | null>(null);

function getSystemTheme(): "light" | "dark" {
  if (typeof window === "undefined") return "light";
  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
}

function applyTheme(theme: Theme, density: Density): "light" | "dark" {
  const resolved = theme === "system" ? getSystemTheme() : theme;
  if (typeof document !== "undefined") {
    document.documentElement.dataset.theme = resolved;
    document.documentElement.dataset.density = density;
    document.documentElement.style.colorScheme = resolved;
  }
  return resolved;
}

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [theme, setThemeState] = useState<Theme>("system");
  const [density, setDensityState] = useState<Density>("comfortable");
  const [resolvedTheme, setResolvedTheme] = useState<"light" | "dark">("light");

  useEffect(() => {
    const storedTheme = localStorage.getItem("mars:theme") as Theme | null;
    const storedDensity = localStorage.getItem("mars:density") as Density | null;
    const nextTheme = storedTheme ?? "system";
    const nextDensity = storedDensity === "compact" ? "compact" : "comfortable";
    // Restore persisted UI preferences after hydration; the inline script has
    // already applied the DOM values to avoid a first-paint flash.
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setThemeState(nextTheme);
    setDensityState(nextDensity);
    setResolvedTheme(applyTheme(nextTheme, nextDensity));
  }, []);

  useEffect(() => {
    if (theme !== "system") return;
    const media = window.matchMedia("(prefers-color-scheme: dark)");
    const onChange = () => setResolvedTheme(applyTheme("system", density));
    media.addEventListener("change", onChange);
    return () => media.removeEventListener("change", onChange);
  }, [theme, density]);

  const setTheme = useCallback(
    (nextTheme: Theme) => {
      setThemeState(nextTheme);
      localStorage.setItem("mars:theme", nextTheme);
      setResolvedTheme(applyTheme(nextTheme, density));
    },
    [density],
  );

  const setDensity = useCallback(
    (nextDensity: Density) => {
      setDensityState(nextDensity);
      localStorage.setItem("mars:density", nextDensity);
      setResolvedTheme(applyTheme(theme, nextDensity));
    },
    [theme],
  );

  const cycleTheme = useCallback(() => {
    const next: Theme = theme === "light" ? "dark" : theme === "dark" ? "system" : "light";
    setTheme(next);
  }, [theme, setTheme]);

  const value = useMemo(
    () => ({ theme, resolvedTheme, density, setTheme, setDensity, cycleTheme }),
    [theme, resolvedTheme, density, setTheme, setDensity, cycleTheme],
  );

  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>;
}

export function useTheme(): ThemeContextValue {
  const value = useContext(ThemeContext);
  if (!value) throw new Error("useTheme must be used inside ThemeProvider");
  return value;
}

export const themeInitScript = `
(function () {
  try {
    var theme = localStorage.getItem("mars:theme") || "system";
    var density = localStorage.getItem("mars:density") === "compact" ? "compact" : "comfortable";
    var dark = window.matchMedia("(prefers-color-scheme: dark)").matches;
    var resolved = theme === "system" ? (dark ? "dark" : "light") : theme;
    document.documentElement.dataset.theme = resolved;
    document.documentElement.dataset.density = density;
    document.documentElement.style.colorScheme = resolved;
  } catch (_) {}
})();
`;
