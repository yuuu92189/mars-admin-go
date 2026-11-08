import { NextRequest } from "next/server";

const target = (process.env.API_PROXY_TARGET ?? "http://localhost:8080").replace(/\/$/, "");
const BODYLESS_METHODS = new Set(["GET", "HEAD"]);

async function proxy(request: NextRequest, context: { params: Promise<{ path: string[] }> }) {
  const { path } = await context.params;
  const upstream = new URL(`${target}/api/${path.join("/")}`);
  upstream.search = request.nextUrl.search;

  const headers = new Headers(request.headers);
  headers.delete("host");
  headers.delete("origin");
  headers.delete("content-length");
  const contentType = headers.get("content-type") ?? "";
  const body = BODYLESS_METHODS.has(request.method)
    ? undefined
    : contentType.includes("application/json")
      ? await request.text()
      : await request.arrayBuffer();

  const response = await fetch(upstream, {
    method: request.method,
    headers,
    body,
    redirect: "manual",
    cache: "no-store",
  });

  const responseHeaders = new Headers(response.headers);
  responseHeaders.delete("content-length");
  const setCookies = (
    response.headers as Headers & { getSetCookie?: () => string[] }
  ).getSetCookie?.();
  if (setCookies?.length) {
    responseHeaders.delete("set-cookie");
    for (const cookie of setCookies) responseHeaders.append("set-cookie", cookie);
  }

  return new Response(response.body, {
    status: response.status,
    headers: responseHeaders,
  });
}

export const GET = proxy;
export const POST = proxy;
export const PUT = proxy;
export const PATCH = proxy;
export const DELETE = proxy;
export const OPTIONS = proxy;
export const HEAD = proxy;
