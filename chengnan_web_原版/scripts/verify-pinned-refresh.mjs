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

const { createRefreshCoordinator } = await importTypeScript(
  "src/TUIKit/components/TUIChat/composables/refresh-coordinator.ts"
);

let now = 1_000;
const coordinator = createRefreshCoordinator({
  minIntervalMs: 5 * 60 * 1000,
  now: () => now,
});
let calls = 0;
const task = async () => {
  calls += 1;
};

assert.equal(await coordinator.run("group-a", task), true);
assert.equal(await coordinator.run("group-a", task), false);
assert.equal(calls, 1, "5 分钟内的被动刷新必须复用结果");

assert.equal(await coordinator.run("group-a", task, { force: true }), true);
assert.equal(calls, 2, "真实置顶变更必须绕过被动刷新间隔");

now += 5 * 60 * 1000;
assert.equal(await coordinator.run("group-a", task), true);
assert.equal(calls, 3, "刷新间隔结束后必须允许再次查询");

let releaseInflight;
const inflightBarrier = new Promise((resolvePromise) => {
  releaseInflight = resolvePromise;
});
let inflightCalls = 0;
const inflightTask = async () => {
  inflightCalls += 1;
  await inflightBarrier;
};
const firstInflight = coordinator.run("group-b", inflightTask);
const joinedInflight = coordinator.run("group-b", inflightTask, {
  force: true,
});
assert.equal(inflightCalls, 1, "并发刷新必须合并为一个请求");
releaseInflight();
assert.deepEqual(await Promise.all([firstInflight, joinedInflight]), [
  true,
  false,
]);

let releaseBeforeMutation;
const beforeMutationBarrier = new Promise((resolvePromise) => {
  releaseBeforeMutation = resolvePromise;
});
let mutationCalls = 0;
const beforeMutationTask = async () => {
  mutationCalls += 1;
  await beforeMutationBarrier;
};
const beforeMutation = coordinator.run("group-c", beforeMutationTask);
const afterMutation = coordinator.run(
  "group-c",
  async () => {
    mutationCalls += 1;
  },
  {
    force: true,
    ensureAfterInflight: true,
  }
);
releaseBeforeMutation();
assert.deepEqual(await Promise.all([beforeMutation, afterMutation]), [
  true,
  true,
]);
assert.equal(mutationCalls, 2, "本地置顶变更后必须补一次确定的新查询");

const composableSource = read(
  "src/TUIKit/components/TUIChat/composables/useGroupPinnedMessage.ts"
);
assert.match(
  composableSource,
  /PASSIVE_PIN_REFRESH_INTERVAL_MS\s*=\s*5\s*\*\s*60\s*\*\s*1000/,
  "被动刷新必须在 5 分钟内去重"
);
assert.doesNotMatch(
  composableSource,
  /setInterval\s*\(/,
  "置顶消息不得恢复定时轮询"
);
assert.match(
  composableSource,
  /onPinnedGroupMessageUpdated[\s\S]*?force:\s*true,[\s\S]*?ensureAfterInflight:\s*true/,
  "SDK 置顶事件必须立即刷新，并在已有查询后补一次读取"
);

const chatSource = read("src/TUIKit/components/TUIChat/index.vue");
assert.match(
  chatSource,
  /handleGroupMessagePinChanged[\s\S]*?force:\s*true,[\s\S]*?ensureAfterInflight:\s*true/,
  "本地置顶变更必须在已有查询后补一次刷新"
);

console.log("pinned message refresh verification passed");
