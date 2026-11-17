import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import vm from "node:vm";
import ts from "typescript";
const source = readFileSync(
  new URL(
    "../src/TUIKit/components/TUIChat/utils/messageQueue.ts",
    import.meta.url
  ),
  "utf8"
);
const exports = {};
vm.runInNewContext(
  ts.transpileModule(source, {
    compilerOptions: {
      target: ts.ScriptTarget.ES2020,
      module: ts.ModuleKind.CommonJS,
    },
  }).outputText,
  { exports, setTimeout, clearTimeout, Promise, Date, console }
);
const { createMessageQueue } = exports;
const wait = (ms = 0) => new Promise((resolve) => setTimeout(resolve, ms));
const release = [];
let starts = [];
const queue = createMessageQueue(() => {}, 3, 5);
const job = (id, lane = "text", active = () => true) => ({
  lane,
  isActive: active,
  run: async () => {
    starts.push(id);
    await new Promise((resolve) => release.push(resolve));
    return "success";
  },
});
const a = job("upload", "attachment"),
  b = job("first-text"),
  c = job("second-text");
assert.equal(queue.enqueue([a, b, c]), true);
assert.equal(queue.enqueue([job("overflow")]), false);
await wait(8);
assert.deepEqual(
  starts,
  ["first-text", "upload"],
  "slow upload must not delay text lane"
);
assert.equal(a.status, "unknown");
assert.equal(b.status, "unknown");
assert.equal(
  queue.pendingCount,
  3,
  "deadline cannot free capacity while SDK is still alive"
);
assert.equal(queue.enqueue([job("after-timeout")]), false);
release[0]();
await wait();
assert.deepEqual(starts, ["first-text", "upload", "second-text"]);
assert.equal(
  b.status,
  "success",
  "late SDK success corrects unknown status without resend"
);
release[1]();
release[2]();
await wait();
assert.equal(queue.pendingCount, 0);
let owner = "first";
starts = [];
const scoped = createMessageQueue(() => {}, 20, 100);
const first = job("already-dispatched", "text", () => owner === "first");
const second = job("must-not-send", "text", () => owner === "first");
scoped.enqueue([first, second]);
await wait();
owner = "second";
release.at(-1)();
await wait();
assert.deepEqual(starts, ["already-dispatched"]);
assert.equal(scoped.jobs.length, 0);
assert.equal(scoped.enqueue([second]), false);
assert.equal(scoped.pendingCount, 0);
const bounded = createMessageQueue(() => {}, 2);
for (let i = 0; i < 15; i++) {
  bounded.enqueue([
    { lane: "text", isActive: () => true, run: async () => "success" },
  ]);
  await wait();
}
assert.ok(bounded.jobs.length <= 4, "display history must remain bounded");
console.log(
  "PASS: queue admission preserves overflow draft; independent text/upload lanes; true in-flight capacity; slow result reconciliation; account isolation; bounded history."
);

const sharedA = exports.getSharedMessageQueue();
const sharedB = exports.getSharedMessageQueue();
assert.equal(
  sharedA,
  sharedB,
  "composer remount must reuse admission and physical lanes"
);
let subscriberUpdates = 0;
const unsubscribe = sharedB.subscribe(() => {
  subscriberUpdates++;
});
sharedA.enqueue([
  { lane: "text", isActive: () => true, run: async () => "success" },
]);
await wait();
assert.ok(subscriberUpdates >= 3);
unsubscribe();
console.log(
  "PASS: composer remount shares the same queue and result subscription."
);

// A failed precheck has no SDK bubble. Retain the entire admitted payload and
// retry the exact captured closure, including a long body and original quote.
const retained = createMessageQueue(() => {}, 2, 50, true);
const fullPayload = {
  text: "正文".repeat(200),
  quote: "captured-original-quote",
};
const capturedPayload = { ...fullPayload };
const receivedPayloads = [];
let attempts = 0;
const failed = {
  lane: "text",
  isActive: () => true,
  label: capturedPayload.text,
  run: async () => {
    receivedPayloads.push({ ...capturedPayload });
    return ++attempts === 1 ? "failed" : "success";
  },
};
const uncertain = {
  lane: "text",
  isActive: () => true,
  run: async () => "unknown",
};
assert.equal(retained.enqueue([failed, uncertain]), true);
fullPayload.text = "new editor content";
fullPayload.quote = "new quote";
await wait();
assert.equal(
  retained.hasCapacity(),
  false,
  "failed payloads reserve capacity until handled"
);
assert.equal(
  retained.retry(uncertain),
  false,
  "uncertain results never authorize resend"
);
assert.equal(
  retained.retry(failed),
  true,
  "retry reuses a reserved slot even at capacity"
);
await wait();
assert.equal(failed.status, "success");
assert.deepEqual(
  receivedPayloads,
  [capturedPayload, capturedPayload],
  "retry retains full content and original quote"
);
assert.equal(failed.label.length, 400);
assert.equal(retained.hasCapacity(), true);
assert.equal(retained.dismiss(uncertain), true);
assert.equal(retained.jobs.includes(uncertain), false);

// Unmount/remount only detaches listeners, it cannot create a second physical
// lane or admit another 20 jobs while the first composer's SDK remains pending.
let releaseShared;
let sharedStarts = 0;
const sharedJobs = Array.from({ length: 20 }, () => ({
  lane: "text",
  isActive: () => true,
  run: async () => {
    sharedStarts++;
    await new Promise((resolve) => {
      releaseShared = resolve;
    });
    return "success";
  },
}));
assert.equal(sharedA.enqueue(sharedJobs), true);
await wait();
assert.equal(sharedStarts, 1);
const remountedQueue = exports.getSharedMessageQueue();
const remountUnsubscribe = remountedQueue.subscribe(() => {});
assert.equal(
  remountedQueue.enqueue([
    { lane: "text", isActive: () => true, run: async () => "success" },
  ]),
  false
);
assert.equal(remountedQueue.pendingCount, 20);
for (let i = 0; i < 20; i++) {
  releaseShared();
  await wait();
}
assert.equal(sharedStarts, 20);
assert.equal(remountedQueue.pendingCount, 0);
remountUnsubscribe();
console.log(
  "PASS: full failed payload retry, unknown retry rejection, retained result capacity, and shared remount capacity."
);
