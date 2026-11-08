import { afterEach, describe, expect, it, vi } from "vitest";
import { login } from "./auth";
import { setSessionRefreshEnabled } from "./client";

function problemResponse(status: number, detail: string): Response {
  return new Response(
    JSON.stringify({
      type: "https://example.test/problem",
      title: detail,
      status,
      detail,
    }),
    {
      status,
      headers: { "content-type": "application/problem+json" },
    },
  );
}

afterEach(() => {
  vi.unstubAllGlobals();
  setSessionRefreshEnabled(true);
});

describe("login", () => {
  it("does not treat invalid credentials as an expired session", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue(problemResponse(401, "Username or password is incorrect."));
    vi.stubGlobal("fetch", fetchMock);

    await expect(
      login({
        username: "tester",
        password: "wrong-password",
        clientType: "admin",
      }),
    ).rejects.toMatchObject({ status: 401 });

    expect(fetchMock).toHaveBeenCalledTimes(1);
    const request = fetchMock.mock.calls[0]?.[0] as Request;
    expect(new URL(request.url).pathname).toBe("/api/v1/auth/login");
  });
});
