import { afterEach, describe, expect, it, vi } from "vitest";
import {
  ApiError,
  apiDownload,
  apiRequest,
  resetUnauthorizedNotification,
  setSessionRefreshEnabled,
} from "./client";

function problemResponse(status: number, detail = "Request failed"): Response {
  return new Response(
    JSON.stringify({
      type: `https://example.test/problems/${status}`,
      title: detail,
      status,
      detail,
      code: `status_${status}`,
    }),
    {
      status,
      headers: { "content-type": "application/problem+json" },
    },
  );
}

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "content-type": "application/json" },
  });
}

function requestPath(input: Request | string): string {
  return typeof input === "string" ? input : new URL(input.url).pathname;
}

afterEach(() => {
  vi.unstubAllGlobals();
  setSessionRefreshEnabled(true);
  resetUnauthorizedNotification();
  document.cookie = "mars_csrf=; Max-Age=0; Path=/";
});

describe("apiRequest", () => {
  it("parses RFC 9457 problem details", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(problemResponse(403, "You do not have permission.")),
    );

    await expect(apiRequest("/api/v1/example")).rejects.toMatchObject({
      name: "ApiError",
      status: 403,
      code: "status_403",
    } satisfies Partial<ApiError>);
  });

  it("adds CSRF for unsafe cookie requests", async () => {
    document.cookie = "mars_csrf=token-123; Path=/";
    const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }));
    vi.stubGlobal("fetch", fetchMock);

    await apiRequest("/api/v1/auth/logout", { method: "POST" });

    const request = fetchMock.mock.calls[0]?.[0] as Request;
    expect(request.headers.get("X-CSRF-Token")).toBe("token-123");
    expect(request.credentials).toBe("include");
  });

  it("refreshes once and retries the original request once", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(problemResponse(401, "Session expired."))
      .mockResolvedValueOnce(jsonResponse({ ok: true }))
      .mockResolvedValueOnce(jsonResponse({ value: 42 }));
    vi.stubGlobal("fetch", fetchMock);

    await expect(apiRequest<{ value: number }>("/api/v1/example")).resolves.toEqual({
      value: 42,
    });
    expect(fetchMock).toHaveBeenCalledTimes(3);
    expect(requestPath(fetchMock.mock.calls[1]?.[0] as Request | string)).toBe(
      "/api/v1/auth/refresh",
    );
  });

  it("does not refresh a 403 or a 5xx", async () => {
    const fetchMock = vi.fn().mockResolvedValue(problemResponse(403));
    vi.stubGlobal("fetch", fetchMock);
    await expect(apiRequest("/api/v1/example")).rejects.toMatchObject({ status: 403 });
    expect(fetchMock).toHaveBeenCalledTimes(1);

    fetchMock.mockReset().mockResolvedValue(problemResponse(503));
    await expect(apiRequest("/api/v1/example")).rejects.toMatchObject({ status: 503 });
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it("dispatches one unauthorized event when refresh is rejected", async () => {
    const onUnauthorized = vi.fn();
    window.addEventListener("mars:unauthorized", onUnauthorized);
    vi.stubGlobal(
      "fetch",
      vi
        .fn()
        .mockResolvedValueOnce(problemResponse(401, "Session expired."))
        .mockResolvedValueOnce(problemResponse(401, "Refresh expired.")),
    );

    await expect(apiRequest("/api/v1/example")).rejects.toMatchObject({ status: 401 });
    expect(onUnauthorized).toHaveBeenCalledTimes(1);
    window.removeEventListener("mars:unauthorized", onUnauthorized);
  });

  it("shares one refresh across concurrent requests", async () => {
    let refreshCalls = 0;
    const attempts = new Map<string, number>();
    const fetchMock = vi.fn(async (input: Request | string) => {
      const path = requestPath(input);
      if (path === "/api/v1/auth/refresh") {
        refreshCalls += 1;
        await new Promise((resolve) => setTimeout(resolve, 20));
        return jsonResponse({ ok: true });
      }
      const count = (attempts.get(path) ?? 0) + 1;
      attempts.set(path, count);
      return count === 1 ? problemResponse(401, "Session expired.") : jsonResponse({ path });
    });
    vi.stubGlobal("fetch", fetchMock);

    const results = await Promise.all([
      apiRequest("/api/v1/a"),
      apiRequest("/api/v1/b"),
      apiRequest("/api/v1/c"),
    ]);
    expect(results).toHaveLength(3);
    expect(refreshCalls).toBe(1);
    expect(fetchMock).toHaveBeenCalledTimes(7);
  });

  it("does not retry forever when the retried request is also 401", async () => {
    const onUnauthorized = vi.fn();
    window.addEventListener("mars:unauthorized", onUnauthorized);
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(problemResponse(401, "Session expired."))
      .mockResolvedValueOnce(jsonResponse({ ok: true }))
      .mockResolvedValueOnce(problemResponse(401, "Still unauthorized."));
    vi.stubGlobal("fetch", fetchMock);

    await expect(apiRequest("/api/v1/example")).rejects.toMatchObject({ status: 401 });
    expect(fetchMock).toHaveBeenCalledTimes(3);
    expect(onUnauthorized).toHaveBeenCalledTimes(1);
    window.removeEventListener("mars:unauthorized", onUnauthorized);
  });

  it("does not refresh after logout is requested", async () => {
    setSessionRefreshEnabled(false);
    const fetchMock = vi.fn().mockResolvedValue(problemResponse(401));
    vi.stubGlobal("fetch", fetchMock);

    await expect(apiRequest("/api/v1/example")).rejects.toMatchObject({ status: 401 });
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });
});

describe("apiDownload", () => {
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
