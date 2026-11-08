import type { Problem } from "./types";

const UNSAFE_METHODS = new Set(["POST", "PUT", "PATCH", "DELETE"]);
const REFRESH_PATH = "/api/v1/auth/refresh";

export class ApiError extends Error {
  readonly status: number;
  readonly code?: string;
  readonly requestId?: string;
  readonly traceId?: string;
  readonly problem?: Problem;

  constructor(problem: Problem, fallbackStatus = 500) {
    super(problem.detail || problem.title || "Request failed");
    this.name = "ApiError";
    this.status = problem.status || fallbackStatus;
    this.code = problem.code;
    this.requestId = problem.requestId;
    this.traceId = problem.traceId;
    this.problem = problem;
  }
}

export type ApiRequestOptions = Omit<RequestInit, "body"> & {
  body?: BodyInit | Record<string, unknown>;
  skipAuthRefresh?: boolean;
  retryOnUnauthorized?: boolean;
};

type RefreshResult =
  | { kind: "ok" }
  | { kind: "unauthorized"; error: ApiError }
  | { kind: "error"; error: ApiError }
  | { kind: "disabled" };

let refreshInFlight: Promise<RefreshResult> | null = null;
let refreshEnabled = true;
let unauthorizedNotified = false;

function apiBase(): string {
  // Browser traffic must always stay on the Next same-origin proxy. The Go
  // target is server-only and is resolved by the Route Handler.
  return "";
}

function requestUrl(path: string): string {
  const origin =
    typeof window !== "undefined" && window.location.origin
      ? window.location.origin
      : "http://localhost";
  return `${origin}${apiBase()}${path}`;
}

function readCookie(name: string): string | undefined {
  if (typeof document === "undefined") return undefined;
  const prefix = `${name}=`;
  const value = document.cookie
    .split("; ")
    .find((entry) => entry.startsWith(prefix))
    ?.slice(prefix.length);
  return value ? decodeURIComponent(value) : undefined;
}

function isBodyInit(value: ApiRequestOptions["body"]): value is BodyInit {
  return (
    typeof value === "string" ||
    value instanceof FormData ||
    value instanceof Blob ||
    value instanceof URLSearchParams ||
    value instanceof ArrayBuffer ||
    (typeof ReadableStream !== "undefined" && value instanceof ReadableStream)
  );
}

function problemFromPayload(payload: unknown, status: number): Problem {
  if (payload && typeof payload === "object") {
    const candidate = payload as Partial<Problem>;
    if (
      typeof candidate.type === "string" ||
      typeof candidate.title === "string" ||
      typeof candidate.detail === "string"
    ) {
      return {
        type: candidate.type ?? "about:blank",
        title: candidate.title ?? "Request failed",
        status: candidate.status ?? status,
        detail: candidate.detail ?? candidate.title ?? "Request failed",
        code: candidate.code,
        requestId: candidate.requestId,
        traceId: candidate.traceId,
      };
    }
  }
  return {
    type: "about:blank",
    title: "Request failed",
    status,
    detail: typeof payload === "string" && payload ? payload : "The server returned an error.",
  };
}

async function responseToError(response: Response): Promise<ApiError> {
  const contentType = response.headers.get("content-type") ?? "";
  const payload = contentType.includes("json")
    ? await response.json().catch(() => undefined)
    : await response.text().catch(() => undefined);
  return new ApiError(problemFromPayload(payload, response.status), response.status);
}

function notifyUnauthorized(problem?: Problem): void {
  if (unauthorizedNotified || typeof window === "undefined") return;
  unauthorizedNotified = true;
  window.dispatchEvent(new CustomEvent("mars:unauthorized", { detail: problem }));
}

async function performRefresh(): Promise<RefreshResult> {
  const headers = new Headers({ Accept: "application/json" });
  const csrfToken = readCookie("mars_csrf");
  if (csrfToken) headers.set("X-CSRF-Token", csrfToken);

  try {
    const response = await fetch(`${apiBase()}${REFRESH_PATH}`, {
      method: "POST",
      headers,
      credentials: "include",
      cache: "no-store",
    });
    if (response.ok) {
      unauthorizedNotified = false;
      return { kind: "ok" };
    }
    const error = await responseToError(response);
    return response.status === 401 ? { kind: "unauthorized", error } : { kind: "error", error };
  } catch {
    return {
      kind: "error",
      error: new ApiError(
        {
          type: "about:blank",
          title: "Network error",
          status: 0,
          detail: "Unable to reach the refresh endpoint.",
        },
        0,
      ),
    };
  }
}

export function refreshSession(): Promise<RefreshResult> {
  if (!refreshEnabled) return Promise.resolve({ kind: "disabled" });
  if (!refreshInFlight) {
    refreshInFlight = performRefresh().finally(() => {
      refreshInFlight = null;
    });
  }
  return refreshInFlight;
}

export function setSessionRefreshEnabled(enabled: boolean): void {
  refreshEnabled = enabled;
  if (!enabled) unauthorizedNotified = false;
}

export function resetUnauthorizedNotification(): void {
  unauthorizedNotified = false;
}

function createRequest(path: string, options: ApiRequestOptions): Request {
  const {
    body,
    skipAuthRefresh: _skipAuthRefresh,
    retryOnUnauthorized: _retryOnUnauthorized,
    headers: inputHeaders,
    ...requestInit
  } = options;
  const method = (requestInit.method ?? "GET").toUpperCase();
  const headers = new Headers(inputHeaders);
  const csrfToken = readCookie("mars_csrf");

  if (body !== undefined && !isBodyInit(body) && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }
  if (UNSAFE_METHODS.has(method) && csrfToken && !headers.has("X-CSRF-Token")) {
    headers.set("X-CSRF-Token", csrfToken);
  }

  return new Request(requestUrl(path), {
    ...requestInit,
    method,
    body: body === undefined ? undefined : isBodyInit(body) ? body : JSON.stringify(body),
    headers,
    credentials: "include",
    cache: requestInit.cache ?? "no-store",
  });
}

async function retryAfterRefresh(
  original: Request,
  options: ApiRequestOptions,
): Promise<Response | null> {
  if (options.skipAuthRefresh || options.retryOnUnauthorized === false) return null;
  let retryRequest: Request;
  try {
    retryRequest = original.clone();
  } catch {
    return null;
  }

  const refresh = await refreshSession();
  if (refresh.kind === "ok") return fetch(retryRequest);
  if (refresh.kind === "unauthorized") {
    notifyUnauthorized(refresh.error.problem);
    throw refresh.error;
  }
  if (refresh.kind === "error") throw refresh.error;
  return null;
}

export async function apiRequest<T>(path: string, options: ApiRequestOptions = {}): Promise<T> {
  const request = createRequest(path, options);
  let response = await fetch(request);
  let retried = false;

  if (response.status === 401) {
    const retry = await retryAfterRefresh(request, options);
    if (retry) {
      response = retry;
      retried = true;
    }
  }

  if (!response.ok) {
    const error = await responseToError(response);
    if (response.status === 401 && retried) notifyUnauthorized(error.problem);
    throw error;
  }

  if (response.status === 204 || response.status === 205) {
    return undefined as T;
  }
  return (await response.json()) as T;
}

export async function apiDownload(
  path: string,
  options: ApiRequestOptions = {},
): Promise<{ blob: Blob; filename?: string }> {
  const request = createRequest(path, {
    ...options,
    method: options.method ?? "GET",
  });
  let response = await fetch(request);
  let retried = false;

  if (response.status === 401) {
    const retry = await retryAfterRefresh(request, options);
    if (retry) {
      response = retry;
      retried = true;
    }
  }

  if (!response.ok) {
    const error = await responseToError(response);
    if (response.status === 401 && retried) notifyUnauthorized(error.problem);
    throw error;
  }

  const disposition = response.headers.get("content-disposition") ?? "";
  const filenameMatch = disposition.match(/filename="?([^";]+)"?/i);
  return {
    blob: await response.blob(),
    filename: filenameMatch?.[1],
  };
}

export function clearReadableSessionCookie(): void {
  if (typeof document === "undefined") return;
  document.cookie = "mars_csrf=; Max-Age=0; Path=/; SameSite=Lax";
}
