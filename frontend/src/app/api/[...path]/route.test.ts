import { NextRequest } from "next/server";
import { afterEach, describe, expect, it, vi } from "vitest";
import { POST } from "./route";

afterEach(() => {
  vi.unstubAllEnvs();
  vi.unstubAllGlobals();
});

describe("same-origin API proxy", () => {
  it("forwards request bodies as streams", async () => {
    vi.stubEnv("NODE_ENV", "test");
    vi.stubEnv("API_BASE_URL", "http://api.test:8080");
    const body = new ReadableStream<Uint8Array>({
      start(controller) {
        controller.enqueue(new TextEncoder().encode("upload"));
        controller.close();
      },
    });
    const requestInit = {
      method: "POST",
      headers: { "content-type": "application/octet-stream" },
      body,
      duplex: "half",
    } as unknown as ConstructorParameters<typeof NextRequest>[1];
    const request = new NextRequest("http://127.0.0.1:3000/api/files", requestInit);
    const fetchMock = vi.fn(async (_input: RequestInfo | URL, init?: RequestInit) => {
      expect(init?.body).toBeInstanceOf(ReadableStream);
      return new Response("{}", {
        status: 200,
        headers: { "content-type": "application/json" },
      });
    });
    vi.stubGlobal("fetch", fetchMock);

    const response = await POST(request, {
      params: Promise.resolve({ path: ["files"] }),
    });

    expect(response.status).toBe(200);
    expect(fetchMock).toHaveBeenCalledOnce();
  });
});
