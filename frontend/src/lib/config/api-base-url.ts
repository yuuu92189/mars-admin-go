export const DEVELOPMENT_API_BASE_URL = "http://localhost:8080";

type ApiEnvironment = Record<string, string | undefined>;

export class ApiBaseUrlError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "ApiBaseUrlError";
  }
}

export function normalizeApiBaseUrl(value: string, variableName = "API_BASE_URL"): string {
  const trimmed = value.trim();
  if (!trimmed) {
    throw new ApiBaseUrlError(`${variableName} must not be empty.`);
  }

  let parsed: URL;
  try {
    parsed = new URL(trimmed);
  } catch {
    throw new ApiBaseUrlError(`${variableName} must be an absolute HTTP(S) URL.`);
  }

  if (parsed.protocol !== "http:" && parsed.protocol !== "https:") {
    throw new ApiBaseUrlError(`${variableName} must use http or https.`);
  }
  if (parsed.username || parsed.password) {
    throw new ApiBaseUrlError(`${variableName} must not contain credentials.`);
  }
  if (parsed.search || parsed.hash) {
    throw new ApiBaseUrlError(`${variableName} must not contain query parameters or fragments.`);
  }

  const pathname = parsed.pathname.replace(/\/+$/, "");
  if (pathname && pathname !== "") {
    throw new ApiBaseUrlError(`${variableName} must not contain a path.`);
  }

  return parsed.origin;
}

function productionRequiresExplicitUrl(env: ApiEnvironment): boolean {
  return (
    env.NODE_ENV === "production" ||
    env.CI === "true" ||
    env.NEXT_PHASE === "phase-production-build"
  );
}

export function resolveApiBaseUrl(env: ApiEnvironment = process.env): string {
  const canonicalValue = env.API_BASE_URL?.trim();
  const legacyValues = [
    ["API_PROXY_TARGET", env.API_PROXY_TARGET?.trim()],
    ["API_INTERNAL_BASE_URL", env.API_INTERNAL_BASE_URL?.trim()],
  ].filter((entry): entry is [string, string] => Boolean(entry[1]));

  const canonical = canonicalValue
    ? normalizeApiBaseUrl(canonicalValue, "API_BASE_URL")
    : undefined;
  const normalizedLegacy = legacyValues.map(([name, value]) => ({
    name,
    value: normalizeApiBaseUrl(value, name),
  }));
  const uniqueLegacy = [...new Set(normalizedLegacy.map((entry) => entry.value))];

  if (uniqueLegacy.length > 1) {
    throw new ApiBaseUrlError(
      "API_PROXY_TARGET and API_INTERNAL_BASE_URL point to different backends. Set one API_BASE_URL instead.",
    );
  }

  const legacy = uniqueLegacy[0];
  if (canonical && legacy && canonical !== legacy) {
    throw new ApiBaseUrlError(
      "API_BASE_URL conflicts with the legacy API target variables. Remove the legacy variables or make all values match.",
    );
  }

  if (canonical) return canonical;
  if (legacy) return legacy;
  if (!productionRequiresExplicitUrl(env)) return DEVELOPMENT_API_BASE_URL;

  throw new ApiBaseUrlError(
    "API_BASE_URL is required in production and CI. Refusing to fall back to localhost.",
  );
}
