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
  ).toString("base64")}`;
  return import(moduleURL);
}

const { createAsyncReadCache } = await importTypeScript(
  "src/api/async-read-cache.ts"
);

let now = 1_000;
const cache = createAsyncReadCache({
  ttlMs: 60_000,
  maxEntries: 2,
  now: () => now,
});

let loads = 0;
let release;
const barrier = new Promise((resolvePromise) => {
  release = resolvePromise;
});
const loader = async () => {
  loads += 1;
  await barrier;
  return null;
};
const first = cache.get("group-a", loader);
const joined = cache.get("group-a", loader);
release();
assert.deepEqual(await Promise.all([first, joined]), [null, null]);
assert.equal(loads, 1, "并发读取必须合并，且空值也必须缓存");
assert.equal(await cache.get("group-a", loader), null);
assert.equal(loads, 1, "有效期内不得重复加载");

now += 60_000;
assert.equal(await cache.get("group-a", async () => ++loads), 2);
assert.equal(loads, 2, "缓存过期后必须重新加载");

cache.invalidate("group-a");
assert.equal(await cache.get("group-a", async () => ++loads), 3);
assert.equal(loads, 3, "主动失效后必须重新加载");

let attempts = 0;
await assert.rejects(
  cache.get("group-error", async () => {
    attempts += 1;
    throw new Error("expected");
  })
);
assert.equal(
  await cache.get("group-error", async () => {
    attempts += 1;
    return "recovered";
  }),
  "recovered"
);
assert.equal(attempts, 2, "失败请求不得进入缓存");

let releaseStale;
const staleBarrier = new Promise((resolvePromise) => {
  releaseStale = resolvePromise;
});
const stale = cache.get("group-stale", async () => {
  await staleBarrier;
  return "stale";
});
cache.invalidate("group-stale");
const fresh = cache.get("group-stale", async () => "fresh");
releaseStale();
assert.equal(await stale, "stale");
assert.equal(await fresh, "fresh");
assert.equal(
  await cache.get("group-stale", async () => "unexpected"),
  "fresh",
  "失效前的并发读取不得覆盖写入后的新结果"
);

const apiSource = read("src/api/index.ts");
assert.match(apiSource, /GROUP_MESSAGE_PIN_CACHE_TTL_MS\s*=\s*5\s*\*\s*60/);
assert.match(apiSource, /GROUP_ADMINS_CACHE_TTL_MS\s*=\s*2\s*\*\s*60/);
assert.match(
  apiSource,
  /getGroupMessagePin[\s\S]*?groupMessagePinCache\.get/
);
assert.match(apiSource, /getGroupAdmins[\s\S]*?groupAdminsCache\.get/);
assert.match(
  apiSource,
  /setChatAuthToken[\s\S]*?groupMessagePinCache\.clear\(\)[\s\S]*?groupAdminsCache\.clear\(\)/
);

console.log("API read cache verification passed");
