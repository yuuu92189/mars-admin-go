export type ThemeMode = "light" | "dark" | "auto";
export type Density = "comfy" | "regular" | "compact";

export type Appearance = {
  theme: ThemeMode;
  density: Density;
  railCollapsed: boolean;
};

export const APPEARANCE_STORAGE_KEY = "scaffold.appearance";
export const APPEARANCE_COOKIE_NAME = "mars_appearance";
export const APPEARANCE_COOKIE_MAX_AGE = 60 * 60 * 24 * 365;

export const DEFAULT_APPEARANCE: Appearance = {
  theme: "light",
  density: "regular",
  railCollapsed: false,
};

export const THEME_OPTIONS: Array<{ value: ThemeMode; label: string }> = [
  { value: "light", label: "浅色" },
  { value: "dark", label: "深色" },
  { value: "auto", label: "跟随系统" },
];

export const DENSITY_OPTIONS: Array<{ value: Density; label: string }> = [
  { value: "comfy", label: "宽松" },
  { value: "regular", label: "标准" },
  { value: "compact", label: "紧凑" },
];

export function isThemeMode(value: unknown): value is ThemeMode {
  return value === "light" || value === "dark" || value === "auto";
}

export function isDensity(value: unknown): value is Density {
  return value === "comfy" || value === "regular" || value === "compact";
}

export function normalizeAppearance(value: unknown): Appearance {
  if (!value || typeof value !== "object") return { ...DEFAULT_APPEARANCE };
  const saved = value as {
    theme?: unknown;
    density?: unknown;
    railCollapsed?: unknown;
  };
  const legacyTheme = saved.theme === "system" ? "auto" : saved.theme;
  const legacyDensity = saved.density === "comfortable" ? "regular" : saved.density;
  return {
    theme: isThemeMode(legacyTheme) ? legacyTheme : DEFAULT_APPEARANCE.theme,
    density: isDensity(legacyDensity) ? legacyDensity : DEFAULT_APPEARANCE.density,
    railCollapsed: saved.railCollapsed === true,
  };
}

export function parseAppearanceJson(raw: string | null | undefined): Appearance {
  if (!raw) return { ...DEFAULT_APPEARANCE };
  try {
    return normalizeAppearance(JSON.parse(raw));
  } catch {
    return { ...DEFAULT_APPEARANCE };
  }
}

export function parseAppearanceCookie(raw: string | null | undefined): Appearance {
  if (!raw) return { ...DEFAULT_APPEARANCE };
  try {
    return normalizeAppearance(JSON.parse(decodeURIComponent(raw)));
  } catch {
    return { ...DEFAULT_APPEARANCE };
  }
}

export function serializeAppearance(value: Appearance): string {
  return JSON.stringify(normalizeAppearance(value));
}

export function resolvedTheme(theme: ThemeMode, systemPrefersDark: boolean): "light" | "dark" {
  if (theme === "auto") return systemPrefersDark ? "dark" : "light";
  return theme;
}

export function readLegacyAppearance(storage: Pick<Storage, "getItem"> | undefined): Appearance {
  if (!storage) return { ...DEFAULT_APPEARANCE };
  const stored = parseAppearanceJson(storage.getItem(APPEARANCE_STORAGE_KEY));
  if (storage.getItem(APPEARANCE_STORAGE_KEY)) return stored;

  const legacyTheme = storage.getItem("mars:theme");
  const legacyDensity = storage.getItem("mars:density");
  return normalizeAppearance({
    ...DEFAULT_APPEARANCE,
    theme: legacyTheme === "system" ? "auto" : legacyTheme,
    density: legacyDensity === "comfortable" ? "regular" : legacyDensity,
  });
}

export function appearanceCookie(value: Appearance): string {
  return `${APPEARANCE_COOKIE_NAME}=${encodeURIComponent(serializeAppearance(value))}; Path=/; Max-Age=${APPEARANCE_COOKIE_MAX_AGE}; SameSite=Lax`;
}
