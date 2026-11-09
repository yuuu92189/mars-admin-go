export const REGISTRATION_ENABLED_KEYS = [
  "site.site.registration_enabled",
  "site.registration_enabled",
  "system.site.registration_enabled",
  "register.enabled",
] as const;

export function publicSetting(settings: Record<string, unknown>, keys: readonly string[]): unknown {
  for (const key of keys) {
    if (settings[key] !== undefined) return settings[key];
  }
  return undefined;
}

export function booleanPublicSetting(
  settings: Record<string, unknown>,
  keys: readonly string[],
  fallback = false,
): boolean {
  const value = publicSetting(settings, keys);
  return typeof value === "boolean" ? value : fallback;
}

export function isRegistrationEnabled(settings: Record<string, unknown>): boolean {
  return booleanPublicSetting(settings, REGISTRATION_ENABLED_KEYS);
}
