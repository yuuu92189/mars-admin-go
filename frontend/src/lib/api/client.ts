import type { Problem } from "./types";

const UNSAFE_METHODS = new Set(["POST", "PUT", "PATCH", "DELETE"]);

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
};

function apiBase(): string {
  return (process.env.NEXT_PUBLIC_API_BASE_URL ?? "").replace(/\/$/, "");
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

export async function apiRequest<T>(path: string, options: ApiRequestOptions = {}): Promise<T> {
  const method = (options.method ?? "GET").toUpperCase();
  const headers = new Headers(options.headers);
  const body = options.body;
  const csrfToken = readCookie("mars_csrf");

  if (body !== undefined && !isBodyInit(body) && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }
  if (UNSAFE_METHODS.has(method) && csrfToken && !headers.has("X-CSRF-Token")) {
    headers.set("X-CSRF-Token", csrfToken);
  }

  const response = await fetch(`${apiBase()}${path}`, {
    ...options,
    body: body === undefined ? undefined : isBodyInit(body) ? body : JSON.stringify(body),
    headers,
    credentials: "include",
    cache: options.cache ?? "no-store",
  });

  if (!response.ok) {
    const contentType = response.headers.get("content-type") ?? "";
    const payload = contentType.includes("json")
      ? await response.json().catch(() => undefined)
      : await response.text().catch(() => undefined);
    const problem = problemFromPayload(payload, response.status);
    if (response.status === 401 && typeof window !== "undefined") {
      window.dispatchEvent(new CustomEvent("mars:unauthorized", { detail: problem }));
    }
    throw new ApiError(problem, response.status);
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
  const method = (options.method ?? "GET").toUpperCase();
  const headers = new Headers(options.headers);
  const csrfToken = readCookie("mars_csrf");
  if (UNSAFE_METHODS.has(method) && csrfToken && !headers.has("X-CSRF-Token")) {
    headers.set("X-CSRF-Token", csrfToken);
  }
  const requestInit: RequestInit = {
    ...options,
    body: undefined,
    headers,
    credentials: "include",
    cache: options.cache ?? "no-store",
  };
  const response = await fetch(`${apiBase()}${path}`, {
    ...requestInit,
  });
  if (!response.ok) {
    const contentType = response.headers.get("content-type") ?? "";
    const payload = contentType.includes("json")
      ? await response.json().catch(() => undefined)
      : await response.text().catch(() => undefined);
    throw new ApiError(problemFromPayload(payload, response.status), response.status);
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
