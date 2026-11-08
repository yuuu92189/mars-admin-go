import { NextRequest, NextResponse } from "next/server";
import { resolveApiBaseUrl } from "@/lib/config/api-base-url";
import { readCookieValue, safeRefreshRedirect } from "@/lib/server/refresh";

export const dynamic = "force-dynamic";

function copySetCookies(source: Headers, target: Headers): void {
  const getSetCookie = (source as Headers & { getSetCookie?: () => string[] }).getSetCookie;
  const cookies = getSetCookie?.call(source) ?? [];
  if (!cookies.length) return;
  target.delete("set-cookie");
  for (const cookie of cookies) target.append("set-cookie", cookie);
}

async function forwardRefresh(request: NextRequest, body?: BodyInit): Promise<Response> {
  const target = resolveApiBaseUrl();
  const headers = new Headers(request.headers);
  headers.delete("host");
  headers.delete("origin");
  headers.delete("content-length");

  return fetch(`${target}/api/v1/auth/refresh`, {
    method: "POST",
    headers,
    body,
    redirect: "manual",
    cache: "no-store",
  });
}

function refreshProblem(status: number, title: string, detail: string): Response {
  return NextResponse.json(
    {
      type: "about:blank",
      title,
      status,
      detail,
    },
    { status, headers: { "content-type": "application/problem+json" } },
  );
}

function relativeRedirect(path: string, sourceHeaders?: Headers): Response {
  const headers = new Headers({ location: path });
  if (sourceHeaders) copySetCookies(sourceHeaders, headers);
  return new Response(null, { status: 307, headers });
}

export async function POST(request: NextRequest): Promise<Response> {
  try {
    const response = await forwardRefresh(request, await request.arrayBuffer());
    const headers = new Headers(response.headers);
    headers.delete("content-length");
    copySetCookies(response.headers, headers);
    return new Response(response.body, {
      status: response.status,
      headers,
    });
  } catch {
    return refreshProblem(503, "Refresh unavailable", "The refresh endpoint could not be reached.");
  }
}

export async function GET(request: NextRequest): Promise<Response> {
  const redirectTo = safeRefreshRedirect(request.nextUrl.searchParams.get("redirect"));
  const loginPath = `/login?redirect=${encodeURIComponent(redirectTo)}`;
  const cookieHeader = request.headers.get("cookie");
  const csrfToken = readCookieValue(cookieHeader, "mars_csrf");
  const headers = new Headers({ Accept: "application/json" });
  if (cookieHeader) headers.set("cookie", cookieHeader);
  if (csrfToken) headers.set("X-CSRF-Token", csrfToken);

  let response: Response;
  try {
    response = await fetch(`${resolveApiBaseUrl()}/api/v1/auth/refresh`, {
      method: "POST",
      headers,
      redirect: "manual",
      cache: "no-store",
    });
  } catch {
    return refreshProblem(503, "Refresh unavailable", "The refresh endpoint could not be reached.");
  }

  if (!response.ok) {
    if (response.status === 400 || response.status === 401) {
      return relativeRedirect(loginPath);
    }
    return refreshProblem(
      response.status >= 500 ? 503 : response.status,
      "Refresh failed",
      `The refresh endpoint returned HTTP ${response.status}.`,
    );
  }

  return relativeRedirect(redirectTo, response.headers);
}
