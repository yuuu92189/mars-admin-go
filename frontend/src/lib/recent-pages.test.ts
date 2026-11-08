import { describe, expect, it } from "vitest";
import { readRecentPages, recordRecentPage } from "./recent-pages";

function memoryStorage(initial: Record<string, string> = {}) {
  const data = new Map(Object.entries(initial));
  return {
    getItem: (key: string) => data.get(key) ?? null,
    setItem: (key: string, value: string) => data.set(key, value),
    dump: () => Object.fromEntries(data),
  };
}

describe("recent pages", () => {
  it("reads only safe local paths", () => {
    const storage = memoryStorage({
      "scaffold.recent": JSON.stringify(["/dashboard", "https://evil.test", "/system/user"]),
    });
    expect(readRecentPages(storage)).toEqual(["/dashboard", "/system/user"]);
  });

  it("deduplicates and keeps the five newest allowed paths", () => {
    const storage = memoryStorage();
    const allowed = new Set(["/a", "/b", "/c", "/d", "/e", "/f"]);
    for (const path of ["/a", "/b", "/c", "/d", "/e", "/f", "/c"]) {
      recordRecentPage(storage, path, allowed);
    }
    expect(readRecentPages(storage)).toEqual(["/c", "/f", "/e", "/d", "/b"]);
  });

  it("does not record unknown routes", () => {
    const storage = memoryStorage();
    recordRecentPage(storage, "/unknown", new Set(["/dashboard"]));
    expect(readRecentPages(storage)).toEqual([]);
  });
});
