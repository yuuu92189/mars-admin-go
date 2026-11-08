export const RECENT_PAGES_STORAGE_KEY = "scaffold.recent";
export const RECENT_PAGES_LIMIT = 5;

export function readRecentPages(storage: Pick<Storage, "getItem"> | undefined): string[] {
  if (!storage) return [];
  try {
    const parsed: unknown = JSON.parse(storage.getItem(RECENT_PAGES_STORAGE_KEY) ?? "[]");
    if (!Array.isArray(parsed)) return [];
    return parsed
      .filter((value): value is string => typeof value === "string" && value.startsWith("/"))
      .slice(0, RECENT_PAGES_LIMIT);
  } catch {
    return [];
  }
}

export function recordRecentPage(
  storage: Pick<Storage, "getItem" | "setItem"> | undefined,
  path: string,
  allowedPaths: ReadonlySet<string>,
): string[] {
  if (!storage || !allowedPaths.has(path)) return readRecentPages(storage);
  const next = [path, ...readRecentPages(storage).filter((item) => item !== path)].slice(
    0,
    RECENT_PAGES_LIMIT,
  );
  try {
    storage.setItem(RECENT_PAGES_STORAGE_KEY, JSON.stringify(next));
  } catch {
    // Recent navigation is a convenience; storage failures are non-fatal.
  }
  return next;
}
