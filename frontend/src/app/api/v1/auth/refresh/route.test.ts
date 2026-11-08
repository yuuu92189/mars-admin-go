import { NextRequest } from "next/server";
import { afterEach, describe, expect, it, vi } from "vitest";
import { GET } from "./route";

function refreshRequest(): NextRequest {
  return new NextRequest("http://127.0.0.1:3000/api/v1/auth/refresh?redirect=%2Fdashboard", {
    headers: {
      cookie: "mars_refresh=refresh-token; mars_csrf=csrf-token",
    },
  });
}

afterEach(() => {
  vi.unstubAllEnvs();
  vi.unstubAllGlobals();
});

describe("GET refresh BFF", () => {
  it("redirects to login only for an authentication failure", async () => {
    vi.stubEnv("NODE_ENV", "test");
    vi.stubEnv("API_BASE_URL", "http://api.test:8080");
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(null, { status: 401 })));

    const response = await GET(refreshRequest());

    expect(response.status).toBe(307);
    expect(response.headers.get("location")).toBe("/login?redirect=%2Fdashboard");
  });

  it("returns a problem instead of misclassifying upstream failures as logout", async () => {
    vi.stubEnv("NODE_ENV", "test");
    vi.stubEnv("API_BASE_URL", "http://api.test:8080");
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(null, { status: 503 })));

    const response = await GET(refreshRequest());

    expect(response.status).toBe(503);
    expect(response.headers.get("content-type")).toContain("application/problem+json");
  });

  it("preserves rotated cookies after a successful refresh", async () => {
    vi.stubEnv("NODE_ENV", "test");
    vi.stubEnv("API_BASE_URL", "http://api.test:8080");
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(null, {
          status: 200,
          headers: { "set-cookie": "mars_session=rotated; Path=/; HttpOnly" },
        }),
      ),
    );

    const response = await GET(refreshRequest());

    expect(response.status).toBe(307);
    expect(response.headers.getSetCookie()).toContain("mars_session=rotated; Path=/; HttpOnly");
  });
});
