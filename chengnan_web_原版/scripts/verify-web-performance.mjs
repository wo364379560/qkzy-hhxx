import assert from "node:assert/strict";
import { readFileSync, readdirSync, statSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import ts from "typescript";

const webRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const read = (relativePath) =>
  readFileSync(resolve(webRoot, relativePath), "utf8");

async function importTypeScript(relativePath) {
  const compiledSource = ts.transpileModule(read(relativePath), {
    compilerOptions: {
      module: ts.ModuleKind.ES2020,
      target: ts.ScriptTarget.ES2020,
    },
  }).outputText;
  const moduleURL = `data:text/javascript;base64,${Buffer.from(
    compiledSource
  ).toString("base64")}`;
  return import(moduleURL);
}

const routing = await importTypeScript("src/config/chat-api-routing.ts");
// These are Rui's configured origins, not the reference application's hosts.
const expectedRoutes = new Map([
  ["api.prfgsu.com", routing.MAIN_CHAT_API_BASE_URL],
  ["api.uktjwg.cn", routing.BACKUP_CHAT_API_BASE_URL],
  ["api1.prfgsu.com", routing.MAIN_CDN_CHAT_API_BASE_URL],
  ["api1.uktjwg.cn", routing.BACKUP_CDN_CHAT_API_BASE_URL],
  ["api.ruizhihe.com", routing.CLOUDFLARE_CHAT_API_BASE_URL],
]);

for (const [hostname, expectedBaseURL] of expectedRoutes) {
  assert.equal(
    routing.resolveProdChatAPIBaseURL(hostname),
    expectedBaseURL,
    `${hostname} 必须使用对应边缘平台的 API`
  );
}
assert.equal(
  routing.resolveProdChatAPIBaseURL("unknown.example"),
  routing.CLOUDFLARE_CHAT_API_BASE_URL,
  "默认启动使用正式池的首条 CF API"
);

const apiSource = read("src/api/index.ts");
assert.match(
  apiSource,
  /timeout:\s*CHAT_API_TIMEOUT_MS/,
  "Axios 必须设置明确超时"
);

const runtimeSource = read("src/config/runtime.ts");
assert.match(
  runtimeSource,
  /rel = 'preconnect'/,
  "登录页必须预热对应 API 连接"
);

for (const relativePath of [
  "src/styles/web/login.scss",
  "src/components/ChatDefaultContent.vue",
]) {
  const source = read(relativePath);
  assert.match(
    source,
    /login-background\.webp/,
    `${relativePath} 必须使用 WebP 背景图`
  );
  assert.doesNotMatch(
    source,
    /login-background\.png/,
    `${relativePath} 不得继续加载大 PNG`
  );
}

const webpPath = resolve(webRoot, "src/assets/image/login-background.webp");
const pngPath = resolve(webRoot, "src/assets/image/login-background.png");
assert.ok(
  statSync(webpPath).size < statSync(pngPath).size / 10,
  "WebP 背景图至少应缩小 90%"
);

const distAssets = resolve(webRoot, "dist/assets");
const emittedBackground = readdirSync(distAssets).find((fileName) =>
  /^login-background-.*\.webp$/.test(fileName)
);
assert.ok(emittedBackground, "生产构建必须输出 WebP 背景图");

console.log("web performance verification passed");
