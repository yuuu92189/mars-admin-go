import { describe, expect, it } from "vitest";
import { hasAllPermissions, hasAnyPermission, hasPermission } from "./permissions";

describe("permissions", () => {
  it("matches explicit permission codes", () => {
    const user = { permissions: ["user:read", "role:read"], isSuperAdmin: false };
    expect(hasPermission(user, "user:read")).toBe(true);
    expect(hasPermission(user, "user:delete")).toBe(false);
    expect(hasAnyPermission(user, ["user:delete", "role:read"])).toBe(true);
    expect(hasAllPermissions(user, ["user:read", "role:read"])).toBe(true);
  });

  it("lets the backend-verified super administrator bypass the permission list", () => {
    const user = { permissions: [], isSuperAdmin: true };
    expect(hasPermission(user, "user:read")).toBe(true);
    expect(hasAnyPermission(user, ["user:create"])).toBe(true);
    expect(hasAllPermissions(user, ["user:read", "user:delete"])).toBe(true);
  });
});
