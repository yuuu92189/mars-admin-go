import { afterEach, describe, expect, it, vi } from "vitest";
import { listAuditEvents, listLoginEvents } from "./audit";
import { downloadGeneratedCode, listImportedTables } from "./codegen";
import { listScheduledJobs } from "./scheduler";
import { listUsers } from "./users";

function jsonResponse(body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status: 200,
    headers: { "content-type": "application/json" },
  });
}

function requestUrl(input: Request | string): URL {
  return new URL(typeof input === "string" ? input : input.url);
}

afterEach(() => {
  vi.unstubAllGlobals();
  document.cookie = "mars_csrf=; Max-Age=0; Path=/";
});

describe("Phase 3 API adapters", () => {
  it("serializes user pagination, status and organization filters", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue(jsonResponse({ items: [], total: 0, page: 2, pageSize: 50 }));
    vi.stubGlobal("fetch", fetchMock);

    await listUsers({
      page: 2,
      pageSize: 50,
      keyword: " admin ",
      status: "active",
      departmentId: "dept-1",
      positionId: "post-1",
    });

    const url = requestUrl(fetchMock.mock.calls[0]?.[0] as Request);
    expect(url.pathname).toBe("/api/v1/admin/users");
    expect(url.searchParams.get("page")).toBe("2");
    expect(url.searchParams.get("pageSize")).toBe("50");
    expect(url.searchParams.get("keyword")).toBe("admin");
    expect(url.searchParams.get("departmentId")).toBe("dept-1");
    expect(url.searchParams.get("positionId")).toBe("post-1");
  });

  it("serializes audit time filters and login pagination", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(jsonResponse({ items: [], total: 0, page: 1, pageSize: 20 }))
      .mockResolvedValueOnce(jsonResponse({ items: [], total: 0, page: 3, pageSize: 25 }));
    vi.stubGlobal("fetch", fetchMock);

    await listAuditEvents({
      page: 1,
      pageSize: 20,
      action: "POST",
      resourceType: "users",
      from: "2026-01-01T00:00:00.000Z",
      to: "2026-01-02T00:00:00.000Z",
    });
    await listLoginEvents(3, 25);

    const auditUrl = requestUrl(fetchMock.mock.calls[0]?.[0] as Request);
    expect(auditUrl.searchParams.get("action")).toBe("POST");
    expect(auditUrl.searchParams.get("resourceType")).toBe("users");
    expect(auditUrl.searchParams.get("from")).toBe("2026-01-01T00:00:00.000Z");

    const loginUrl = requestUrl(fetchMock.mock.calls[1]?.[0] as Request);
    expect(loginUrl.pathname).toBe("/api/v1/admin/audit/logins");
    expect(loginUrl.searchParams.get("page")).toBe("3");
    expect(loginUrl.searchParams.get("pageSize")).toBe("25");
  });

  it("serializes scheduler and codegen pagination", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(jsonResponse({ items: [], total: 0, page: 1, pageSize: 15 }))
      .mockResolvedValueOnce(jsonResponse({ items: [], total: 0, page: 2, pageSize: 12 }));
    vi.stubGlobal("fetch", fetchMock);

    await listScheduledJobs({ page: 1, pageSize: 15, status: "active" });
    await listImportedTables(" audit ", 2, 12);

    const jobUrl = requestUrl(fetchMock.mock.calls[0]?.[0] as Request);
    expect(jobUrl.searchParams.get("status")).toBe("active");

    const codegenUrl = requestUrl(fetchMock.mock.calls[1]?.[0] as Request);
    expect(codegenUrl.pathname).toBe("/api/v1/admin/codegen/imported");
    expect(codegenUrl.searchParams.get("query")).toBe("audit");
    expect(codegenUrl.searchParams.get("page")).toBe("2");
  });

  it("requests a real ZIP download with imported table ids", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response("zip-bytes", {
        status: 200,
        headers: {
          "content-type": "application/zip",
          "content-disposition": 'attachment; filename="mars-admin-codegen.zip"',
        },
      }),
    );
    vi.stubGlobal("fetch", fetchMock);

    const result = await downloadGeneratedCode(["id-1", "id-2"]);

    const url = requestUrl(fetchMock.mock.calls[0]?.[0] as Request);
    expect(url.pathname).toBe("/api/v1/admin/codegen/download");
    expect(url.searchParams.get("ids")).toBe("id-1,id-2");
    expect(result.filename).toBe("mars-admin-codegen.zip");
    expect(await result.blob.text()).toBe("zip-bytes");
  });
});
