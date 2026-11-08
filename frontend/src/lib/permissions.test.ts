import { describe, expect, it } from "vitest";
import { hasAllPermissions, hasAnyPermission, hasPermission } from "./permissions";

const user = { permissions: ["user:read", "role:read"] };

describe("permission helpers", () => {
  it("checks a single permission", () => {
    expect(hasPermission(user, "user:read")).toBe(true);
    expect(hasPermission(user, "user:create")).toBe(false);
  });

  it("supports any and all semantics", () => {
    expect(hasAnyPermission(user, ["user:create", "role:read"])).toBe(true);
    expect(hasAnyPermission(user, ["user:create", "notice:read"])).toBe(false);
    expect(hasAllPermissions(user, ["user:read", "role:read"])).toBe(true);
    expect(hasAllPermissions(user, ["user:read", "notice:read"])).toBe(false);
  });

  it("treats a missing user as unauthenticated", () => {
    expect(hasPermission(null, "user:read")).toBe(false);
  });
});
