import { describe, expect, it } from "vitest";
import {
  ApiBaseUrlError,
  DEVELOPMENT_API_BASE_URL,
  normalizeApiBaseUrl,
  resolveApiBaseUrl,
} from "./api-base-url";

describe("API base URL configuration", () => {
  it("uses the development default outside production", () => {
    expect(resolveApiBaseUrl({ NODE_ENV: "development" })).toBe(DEVELOPMENT_API_BASE_URL);
    expect(resolveApiBaseUrl({ NODE_ENV: "test" })).toBe(DEVELOPMENT_API_BASE_URL);
  });

  it("uses and normalizes the explicit canonical URL", () => {
    expect(
      resolveApiBaseUrl({
        NODE_ENV: "production",
        API_BASE_URL: "https://api.example.test/",
      }),
    ).toBe("https://api.example.test");
  });

  it("accepts matching legacy aliases during migration", () => {
    expect(
      resolveApiBaseUrl({
        NODE_ENV: "production",
        API_PROXY_TARGET: "http://api:8080",
        API_INTERNAL_BASE_URL: "http://api:8080/",
      }),
    ).toBe("http://api:8080");
  });

  it("rejects conflicting legacy targets", () => {
    expect(() =>
      resolveApiBaseUrl({
        API_PROXY_TARGET: "http://api-a:8080",
        API_INTERNAL_BASE_URL: "http://api-b:8080",
      }),
    ).toThrow(ApiBaseUrlError);
  });

  it("rejects a canonical URL that conflicts with a legacy alias", () => {
    expect(() =>
      resolveApiBaseUrl({
        API_BASE_URL: "https://api-a.example.test",
        API_PROXY_TARGET: "https://api-b.example.test",
      }),
    ).toThrow(/conflicts/i);
  });

  it("rejects malformed, credentialed and pathed URLs", () => {
    expect(() => normalizeApiBaseUrl("localhost:8080")).toThrow(/http/i);
    expect(() => normalizeApiBaseUrl("ftp://api.example.test")).toThrow(/http/i);
    expect(() => normalizeApiBaseUrl("https://user:secret@api.example.test")).toThrow(
      /credentials/i,
    );
    expect(() => normalizeApiBaseUrl("https://api.example.test/backend")).toThrow(/path/i);
  });

  it("fails closed when production has no API target", () => {
    expect(() => resolveApiBaseUrl({ NODE_ENV: "production" })).toThrow(/required/i);
    expect(() => resolveApiBaseUrl({ CI: "true" })).toThrow(/required/i);
  });
});
