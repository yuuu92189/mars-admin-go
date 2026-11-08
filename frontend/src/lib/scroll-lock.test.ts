import { afterEach, describe, expect, it } from "vitest";
import { lockDocumentScroll } from "./scroll-lock";

afterEach(() => {
  document.body.style.overflow = "";
});

describe("lockDocumentScroll", () => {
  it("restores the original overflow only after the final release", () => {
    document.body.style.overflow = "auto";
    const releaseFirst = lockDocumentScroll();
    const releaseSecond = lockDocumentScroll();

    expect(document.body.style.overflow).toBe("hidden");
    releaseFirst();
    expect(document.body.style.overflow).toBe("hidden");
    releaseSecond();
    expect(document.body.style.overflow).toBe("auto");
  });
});
