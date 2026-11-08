import { afterEach, describe, expect, it, vi } from "vitest";
import { getCurrentUserFromCookie } from "./server-auth";

const user = {
  user: {
    id: "019b0000-0000-7000-8000-000000000001",
    username: "tester",
    displayName: "Tester",
    status: "active",
    userType: "admin",
    version: 1,
  },
  roles: ["admin"],
  permissions: ["user:read"],
  menus: [],
};

afterEach(() => {
  vi.unstubAllEnvs();
  vi.unstubAllGlobals();
});

describe("getCurrentUserFromCookie", () => {
  it("does not call the API without a cookie", async () => {
    const fetchMock = vi.fn();
    vi.stubGlobal("fetch", fetchMock);
    await expect(getCurrentUserFromCookie(undefined)).resolves.toEqual({
      kind: "unauthorized",
    });
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it("uses the unified API base URL", async () => {
    vi.stubEnv("NODE_ENV", "test");
    vi.stubEnv("API_BASE_URL", "http://api.test:8081");
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify(user), {
        status: 200,
        headers: { "content-type": "application/json" },
      }),
    );
    vi.stubGlobal("fetch", fetchMock);

    await expect(getCurrentUserFromCookie("mars_session=session")).resolves.toEqual({
      kind: "ok",
      user,
    });
    expect(fetchMock).toHaveBeenCalledWith(
      "http://api.test:8081/api/v1/auth/me",
      expect.objectContaining({
        headers: { cookie: "mars_session=session" },
        cache: "no-store",
      }),
    );
  });

  it("distinguishes unauthorized from upstream errors", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(new Response(null, { status: 401 })));
    await expect(getCurrentUserFromCookie("mars_session=expired")).resolves.toEqual({
      kind: "unauthorized",
    });

    vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(new Response(null, { status: 503 })));
    await expect(getCurrentUserFromCookie("mars_session=error")).resolves.toEqual({
      kind: "error",
      status: 503,
    });

    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new Error("offline")));
    await expect(getCurrentUserFromCookie("mars_session=offline")).resolves.toEqual({
      kind: "error",
    });
  });
});
