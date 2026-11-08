import { mkdir, readFile, writeFile } from "node:fs/promises";
import path from "node:path";
import pixelmatch from "pixelmatch";
import { PNG } from "pngjs";

const root = path.resolve("../docs/phase-3/visual-regression");
const vueDir = path.join(root, "vue");
const reactDir = path.join(root, "react");
const diffDir = path.join(root, "diff");
const pages = [
  "dashboard",
  "users",
  "roles",
  "menus",
  "departments",
  "positions",
  "dictionaries",
  "settings",
  "notices",
  "scheduler",
  "operation-logs",
  "login-logs",
  "codegen",
];
const scenarios = [
  "desktop-light",
  "desktop-dark",
  "desktop-compact",
  "tablet-landscape",
  "tablet-portrait",
  "mobile",
];

function padImage(image, width, height) {
  if (image.width === width && image.height === height) return image;
  const output = new PNG({ width, height });
  output.data.fill(255);
  PNG.bitblt(image, output, 0, 0, image.width, image.height, 0, 0);
  return output;
}

await mkdir(diffDir, { recursive: true });
const records = [];
for (const page of pages) {
  for (const scenario of scenarios) {
    const fileName = `${page}-${scenario}.png`;
    const vue = PNG.sync.read(await readFile(path.join(vueDir, fileName)));
    const react = PNG.sync.read(await readFile(path.join(reactDir, fileName)));
    const width = Math.max(vue.width, react.width);
    const height = Math.max(vue.height, react.height);
    const vuePadded = padImage(vue, width, height);
    const reactPadded = padImage(react, width, height);
    const diff = new PNG({ width, height });
    const mismatchPixels = pixelmatch(vuePadded.data, reactPadded.data, diff.data, width, height, {
      threshold: 0.12,
    });
    const diffName = `${page}-${scenario}.png`;
    await writeFile(path.join(diffDir, diffName), PNG.sync.write(diff));
    records.push({
      page,
      scenario,
      vue: { width: vue.width, height: vue.height },
      react: { width: react.width, height: react.height },
      diff: { width, height, fileName: diffName },
      mismatchPixels,
      mismatchRatio: Number((mismatchPixels / (width * height)).toFixed(4)),
      heightDelta: react.height - vue.height,
      note: "Vue uses deterministic visual fixtures; React uses the live Go-backed business data.",
    });
  }
}

await writeFile(
  path.join(root, "phase3-visual-metrics.json"),
  `${JSON.stringify({ generatedAt: new Date().toISOString(), records }, null, 2)}\n`,
);
console.log(`Generated ${records.length} Phase 3 visual diffs.`);
