import { describe, expect, it } from "vitest";
import {
  DEFAULT_APPEARANCE,
  normalizeAppearance,
  parseAppearanceCookie,
  parseAppearanceJson,
  readLegacyAppearance,
  resolvedTheme,
} from "./appearance";

describe("appearance", () => {
  it("uses the Vue defaults", () => {
    expect(DEFAULT_APPEARANCE).toEqual({
      theme: "light",
      density: "regular",
      railCollapsed: false,
    });
  });

  it("accepts all Vue theme and density values", () => {
    expect(
      normalizeAppearance({
        theme: "auto",
        density: "comfy",
        railCollapsed: true,
      }),
    ).toEqual({ theme: "auto", density: "comfy", railCollapsed: true });
  });

  it("migrates the Phase 2 enum aliases", () => {
    expect(normalizeAppearance({ theme: "system", density: "comfortable" })).toEqual({
      theme: "auto",
      density: "regular",
      railCollapsed: false,
    });
  });

  it("parses storage and cookie values safely", () => {
    expect(parseAppearanceJson('{"theme":"dark","density":"compact"}')).toEqual({
      theme: "dark",
      density: "compact",
      railCollapsed: false,
    });
    expect(parseAppearanceJson("{broken")).toEqual(DEFAULT_APPEARANCE);
    expect(parseAppearanceCookie(encodeURIComponent('{"theme":"auto","density":"comfy"}'))).toEqual(
      {
        theme: "auto",
        density: "comfy",
        railCollapsed: false,
      },
    );
  });

  it("reads legacy browser keys when the Vue key is absent", () => {
    const storage = {
      getItem: (key: string) =>
        key === "mars:theme" ? "dark" : key === "mars:density" ? "compact" : null,
    };
    expect(readLegacyAppearance(storage)).toEqual({
      theme: "dark",
      density: "compact",
      railCollapsed: false,
    });
  });

  it("resolves auto mode from the system preference", () => {
    expect(resolvedTheme("auto", true)).toBe("dark");
    expect(resolvedTheme("auto", false)).toBe("light");
    expect(resolvedTheme("dark", false)).toBe("dark");
  });
});
