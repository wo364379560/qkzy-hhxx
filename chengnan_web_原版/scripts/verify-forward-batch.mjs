import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const webRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const read = (relativePath) =>
  readFileSync(resolve(webRoot, relativePath), "utf8");

async function importTypeScript(relativePath) {
  const source = read(relativePath);
  let compiledSource;
  try {
    const { default: ts } = await import("typescript");
    compiledSource = ts.transpileModule(source, {
      compilerOptions: {
        module: ts.ModuleKind.ES2020,
        target: ts.ScriptTarget.ES2020,
      },
    }).outputText;
  } catch (typescriptImportError) {
    const { stripTypeScriptTypes } = await import("node:module");
    if (typeof stripTypeScriptTypes !== "function") {
      throw typescriptImportError;
    }
    compiledSource = stripTypeScriptTypes(source, {
      mode: "strip",
      sourceMap: false,
    });
  }
  const moduleURL = `data:text/javascript;base64,${Buffer.from(
    compiledSource
  ).toString("base64")}#${encodeURIComponent(relativePath)}`;
  return import(moduleURL);
}

const { runForwardBatch } = await importTypeScript(
  "src/TUIKit/components/TUIChat/utils/forward-batch.ts"
);

let running = 0;
let maxRunning = 0;
const progress = [];
const inputs = Array.from({ length: 12 }, (_, index) => index);
const results = await runForwardBatch(
  inputs,
  async (item) => {
    running += 1;
    maxRunning = Math.max(maxRunning, running);
    await new Promise((resolvePromise) => setTimeout(resolvePromise, 2));
    running -= 1;
    if (item === 5 || item === 9) {
      throw Object.assign(new Error(`failed-${item}`), { code: 10023 });
    }
    return item * 2;
  },
  {
    concurrency: 4,
    onProgress: ({ completed, total }) => progress.push([completed, total]),
  }
);

assert.equal(maxRunning, 4, "批量转发并发数必须受限为 4");
assert.deepEqual(
  results.map((result) => result.item),
  inputs,
  "批量结果必须保持目标原始顺序"
);
assert.deepEqual(
  results
    .filter((result) => result.status === "rejected")
    .map((result) => result.item),
  [5, 9],
  "必须精确保留失败目标供安全重试"
);
assert.deepEqual(progress.at(-1), [12, 12], "进度必须最终到达全部目标");

const emptyResults = await runForwardBatch([], async () => undefined, {
  concurrency: 4,
});
assert.deepEqual(emptyResults, [], "空目标不应启动任务");

// Stateful per-message/panel regressions are executed in verify-forward-results
// and verify-forward-panel; this suite verifies the shared bounded scheduler.
console.log("bounded forward scheduler verification passed");
