import { describe, expect, it } from "vitest";
import { readCookieValue, safeRefreshRedirect } from "./refresh";

describe("server refresh helpers", () => {
  it("accepts only relative redirect targets", () => {
    expect(safeRefreshRedirect("/settings/profile?tab=security")).toBe(
      "/settings/profile?tab=security",
    );
    expect(safeRefreshRedirect("//evil.test")).toBe("/dashboard");
    expect(safeRefreshRedirect("https://evil.test")).toBe("/dashboard");
    expect(safeRefreshRedirect("\\evil")).toBe("/dashboard");
    expect(safeRefreshRedirect(null)).toBe("/dashboard");
  });

  it("reads the named cookie without exposing other cookies", () => {
    expect(readCookieValue("a=1; mars_csrf=token; b=2", "mars_csrf")).toBe("token");
    expect(readCookieValue("a=1", "mars_csrf")).toBeUndefined();
    expect(readCookieValue(null, "mars_csrf")).toBeUndefined();
  });
});
