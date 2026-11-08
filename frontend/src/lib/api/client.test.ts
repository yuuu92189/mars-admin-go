import { afterEach, describe, expect, it, vi } from "vitest";
import { ApiError, apiDownload, apiRequest } from "./client";

afterEach(() => {
  vi.unstubAllGlobals();
  document.cookie = "mars_csrf=; Max-Age=0; Path=/";
});

describe("apiRequest", () => {
  it("parses RFC 9457 problem details", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            type: "https://example.test/problem",
            title: "Permission denied",
            status: 403,
            detail: "You do not have permission.",
            code: "permission_denied",
          }),
          { status: 403, headers: { "content-type": "application/problem+json" } },
        ),
      ),
    );

    await expect(apiRequest("/api/v1/example")).rejects.toMatchObject({
      name: "ApiError",
      status: 403,
      code: "permission_denied",
    } satisfies Partial<ApiError>);
  });

  it("adds CSRF for unsafe cookie requests", async () => {
    document.cookie = "mars_csrf=token-123; Path=/";
    const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }));
    vi.stubGlobal("fetch", fetchMock);

    await apiRequest("/api/v1/auth/logout", { method: "POST" });

    const [, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(new Headers(init.headers).get("X-CSRF-Token")).toBe("token-123");
    expect(init.credentials).toBe("include");
  });

  it("returns a downloadable blob and filename", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response("file-content", {
          status: 200,
          headers: { "content-disposition": 'attachment; filename="report.csv"' },
        }),
      ),
    );
    const result = await apiDownload("/api/v1/files/example/download");
    expect(result.filename).toBe("report.csv");
    expect(await result.blob.text()).toBe("file-content");
  });
});
