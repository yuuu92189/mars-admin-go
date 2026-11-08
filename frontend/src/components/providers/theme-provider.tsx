"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  useSyncExternalStore,
  type ReactNode,
} from "react";
import {
  APPEARANCE_COOKIE_NAME,
  APPEARANCE_STORAGE_KEY,
  DEFAULT_APPEARANCE,
  appearanceCookie,
  normalizeAppearance,
  readLegacyAppearance,
  resolvedTheme as resolveTheme,
  serializeAppearance,
  type Appearance,
  type Density,
  type ThemeMode,
} from "@/lib/appearance";

type ThemeContextValue = {
  theme: ThemeMode;
  resolvedTheme: "light" | "dark";
  density: Density;
  railCollapsed: boolean;
  setTheme: (theme: ThemeMode) => void;
  setDensity: (density: Density) => void;
  setRailCollapsed: (collapsed: boolean) => void;
  cycleTheme: () => void;
};

const ThemeContext = createContext<ThemeContextValue | null>(null);
const DARK_QUERY = "(prefers-color-scheme: dark)";

function subscribeSystemTheme(onChange: () => void) {
  const media = window.matchMedia(DARK_QUERY);
  media.addEventListener("change", onChange);
  return () => media.removeEventListener("change", onChange);
}

function getSystemThemeSnapshot() {
  return window.matchMedia(DARK_QUERY).matches;
}

function getServerThemeSnapshot() {
  return false;
}

function applyAppearance(appearance: Appearance, systemDark: boolean) {
  if (typeof document === "undefined") return;
  const theme = resolveTheme(appearance.theme, systemDark);
  document.documentElement.dataset.theme = theme;
  document.documentElement.dataset.density = appearance.density;
  document.documentElement.dataset.railCollapsed = String(appearance.railCollapsed);
  document.documentElement.style.colorScheme = theme;
}

function persistAppearance(appearance: Appearance) {
  try {
    localStorage.setItem(APPEARANCE_STORAGE_KEY, serializeAppearance(appearance));
  } catch {
    // Browser storage can be disabled; the cookie mirror still works.
  }
  document.cookie = appearanceCookie(appearance);
}

export function ThemeProvider({
  children,
  initialAppearance = DEFAULT_APPEARANCE,
  appearanceInitialized = false,
}: {
  children: ReactNode;
  initialAppearance?: Appearance;
  appearanceInitialized?: boolean;
}) {
  const [appearance, setAppearanceState] = useState<Appearance>(() =>
    normalizeAppearance(initialAppearance),
  );
  const systemDark = useSyncExternalStore(
    subscribeSystemTheme,
    getSystemThemeSnapshot,
    getServerThemeSnapshot,
  );
  const resolvedTheme = resolveTheme(appearance.theme, systemDark);

  useEffect(() => {
    applyAppearance(appearance, systemDark);
  }, [appearance, systemDark]);

  useEffect(() => {
    if (appearanceInitialized) return;
    try {
      const stored = readLegacyAppearance(localStorage);
      const hasStoredPreference =
        localStorage.getItem(APPEARANCE_STORAGE_KEY) !== null ||
        localStorage.getItem("mars:theme") !== null ||
        localStorage.getItem("mars:density") !== null;
      if (
        stored.theme !== DEFAULT_APPEARANCE.theme ||
        stored.density !== DEFAULT_APPEARANCE.density ||
        stored.railCollapsed !== DEFAULT_APPEARANCE.railCollapsed
      ) {
        window.setTimeout(() => setAppearanceState(stored), 0);
      }
      if (hasStoredPreference) document.cookie = appearanceCookie(stored);
    } catch {
      // Storage can be unavailable in private browsing.
    }
  }, [appearanceInitialized]);

  const updateAppearance = useCallback((patch: Partial<Appearance>) => {
    setAppearanceState((current) => {
      const next = normalizeAppearance({ ...current, ...patch });
      persistAppearance(next);
      return next;
    });
  }, []);

  const setTheme = useCallback(
    (theme: ThemeMode) => updateAppearance({ theme }),
    [updateAppearance],
  );
  const setDensity = useCallback(
    (density: Density) => updateAppearance({ density }),
    [updateAppearance],
  );
  const setRailCollapsed = useCallback(
    (railCollapsed: boolean) => updateAppearance({ railCollapsed }),
    [updateAppearance],
  );
  const cycleTheme = useCallback(() => {
    setAppearanceState((current) => {
      const nextTheme: ThemeMode =
        current.theme === "light" ? "dark" : current.theme === "dark" ? "auto" : "light";
      const next = { ...current, theme: nextTheme };
      persistAppearance(next);
      return next;
    });
  }, []);

  const value = useMemo(
    () => ({
      theme: appearance.theme,
      resolvedTheme,
      density: appearance.density,
      railCollapsed: appearance.railCollapsed,
      setTheme,
      setDensity,
      setRailCollapsed,
      cycleTheme,
    }),
    [
      appearance.theme,
      appearance.density,
      appearance.railCollapsed,
      resolvedTheme,
      setTheme,
      setDensity,
      setRailCollapsed,
      cycleTheme,
    ],
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
    var defaults = ${JSON.stringify(DEFAULT_APPEARANCE)};
    var raw = document.cookie.split("; ").find(function (item) {
      return item.indexOf("${APPEARANCE_COOKIE_NAME}=") === 0;
    });
    var stored = null;
    if (raw) {
      stored = JSON.parse(decodeURIComponent(raw.slice(${APPEARANCE_COOKIE_NAME.length + 1})));
    } else {
      var local = localStorage.getItem("${APPEARANCE_STORAGE_KEY}");
      if (local) {
        stored = JSON.parse(local);
      } else {
        stored = {
          theme: localStorage.getItem("mars:theme"),
          density: localStorage.getItem("mars:density")
        };
      }
    }
    var theme = stored && ["light", "dark", "auto"].indexOf(stored.theme) >= 0
      ? stored.theme
      : stored && stored.theme === "system" ? "auto" : defaults.theme;
    var density = stored && ["comfy", "regular", "compact"].indexOf(stored.density) >= 0
      ? stored.density
      : stored && stored.density === "comfortable" ? "regular" : defaults.density;
    var railCollapsed = stored && stored.railCollapsed === true;
    var dark = window.matchMedia("(prefers-color-scheme: dark)").matches;
    var resolved = theme === "auto" ? (dark ? "dark" : "light") : theme;
    document.documentElement.dataset.theme = resolved;
    document.documentElement.dataset.density = density;
    document.documentElement.dataset.railCollapsed = String(railCollapsed);
    document.documentElement.style.colorScheme = resolved;
  } catch (_) {}
})();
`;
