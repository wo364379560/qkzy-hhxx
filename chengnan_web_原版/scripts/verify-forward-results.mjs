import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { createRequire } from "node:module";
import vm from "node:vm";
import ts from "typescript";
const require = createRequire(import.meta.url);
const source = readFileSync(
  new URL(
    "../src/TUIKit/components/TUIChat/forward/forwardBatch.ts",
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
  { exports, require, setTimeout, clearTimeout, Promise, Date, console }
);
const {
  createForwardBatch,
  runForwardBatch,
  canRetryForwardTarget,
  snapshotForwardMessages,
} = exports;
const wait = (ms) => new Promise((resolve) => setTimeout(resolve, ms));
const batch = (targets = 2, messages = 3) =>
  createForwardBatch(
    Array.from({ length: targets }, (_, i) => ({
      conversationID: `GROUP${i}`,
      name: `group${i}`,
    })),
    Array.from({ length: messages }, (_, i) => ({
      ID: `m${i}`,
      payload: { text: `${i}` },
    })),
    false,
    (c) => c.name
  );
const deps = (send, extra = {}) => ({
  send,
  resend: async () => {
    throw new Error("unexpected resend");
  },
  isActive: () => true,
  onChange: () => {},
  intervalMs: 0,
  timeoutMs: 200,
  autoRetryCount: 0, // These cases exercise explicit/manual retry; direct tests cover automatic retry.
  ...extra,
});
let active = 0,
  peak = 0,
  order = new Map();
const many = batch(100, 3);
await runForwardBatch(
  many,
  deps(async (target, item) => {
    active++;
    peak = Math.max(peak, active);
    const id = target.conversation.conversationID;
    order.set(id, [...(order.get(id) || []), item.messages[0].ID]);
    await wait(1);
    active--;
  })
);
assert.equal(peak, 4);
for (const sequence of order.values())
  assert.deepEqual(sequence, ["m0", "m1", "m2"]);
assert.ok(
  many.targets.every((t) => t.items.every((i) => i.status === "success"))
);
const partial = batch(2, 3);
let fail = true,
  sent = [];
const sendPartial = deps(async (target, item) => {
  const key = `${target.conversation.conversationID}:${item.messages[0].ID}`;
  sent.push(key);
  if (key === "GROUP0:m1" && fail) throw { code: 10017 };
});
await runForwardBatch(partial, sendPartial);
assert.deepEqual(
  Array.from(partial.targets[0].items, (i) => i.status),
  ["success", "failed", "pending"]
);
fail = false;
await runForwardBatch(partial, sendPartial);
assert.equal(sent.filter((id) => id === "GROUP0:m0").length, 1);
assert.equal(sent.filter((id) => id === "GROUP0:m1").length, 2);
assert.equal(sent.filter((id) => id.startsWith("GROUP1")).length, 3);
let resolveLate;
const delayed = batch(10, 2);
let calls = 0;
await runForwardBatch(
  delayed,
  deps(
    async () => {
      calls++;
      await new Promise((resolve) => {
        resolveLate = resolve;
      });
    },
    { timeoutMs: 5 }
  )
);
assert.equal(calls, 4, "timeout must not release the native in-flight bound");
assert.equal(delayed.targets[0].items[0].status, "unknown");
assert.equal(canRetryForwardTarget(delayed.targets[0]), false);
await runForwardBatch(
  delayed,
  deps(async () => calls++)
);
assert.equal(
  calls,
  4,
  "unresolved SDK operations must block restarting the batch"
);
resolveLate();
await wait(0);
assert.equal(
  delayed.targets[3].items[0].status,
  "success",
  "late ACK reconciles without duplicate send"
);
const uncertain = batch(1, 2);
await runForwardBatch(
  uncertain,
  deps(async (target) => {
    throw {
      code: 2800,
      data: {
        message: {
          ID: "sdk1",
          conversationID: target.conversation.conversationID,
        },
      },
    };
  })
);
assert.equal(
  canRetryForwardTarget(uncertain.targets[0]),
  false,
  "SDK object does not make an unknown transport result safe to resend"
);
const guarded = batch(2, 1);
let dispatched = 0;
await runForwardBatch(
  guarded,
  deps(async () => dispatched++, {
    prepare: async () => {
      throw new Error("permission unavailable");
    },
  })
);
assert.equal(dispatched, 0);
assert.ok(guarded.targets.every((t) => t.items[0].status === "failed"));
assert.ok(guarded.targets.every(canRetryForwardTarget));
let sameAccount = true;
const switched = batch(2, 2);
await runForwardBatch(
  switched,
  deps(
    async () => {
      throw new Error("should not dispatch");
    },
    {
      isActive: () => sameAccount,
      prepare: async () => {
        sameAccount = false;
      },
    }
  )
);
assert.ok(
  switched.targets.every((t) => t.items.every((i) => i.status !== "success"))
);
const original = { ID: "m1", payload: { text: "before" } };
const snapshot = snapshotForwardMessages(["m1", "m1"], () => original);
original.payload.text = "after";
assert.equal(snapshot.length, 1);
assert.equal(snapshot[0].payload.text, "before");
assert.throws(() => snapshotForwardMessages(["gone"], () => null));
const pacing = batch(5, 1),
  starts = [];
await runForwardBatch(
  pacing,
  deps(async () => starts.push(Date.now()), { intervalMs: 12 })
);
for (let i = 1; i < starts.length; i++)
  assert.ok(starts[i] - starts[i - 1] >= 11);

// Slow group permission requests can finish together. Pacing their start time
// does not pace SDK dispatch: the first four workers and a new fifth target
// used to send at once after this barrier opened.
const slowPreflight = batch(8, 7);
const actualDispatches = [];
const slowIntervalMs = 20;
let preparedTargets = 0;
let releasePermissions;
const permissionsReady = new Promise((resolve) => {
  releasePermissions = resolve;
});
await runForwardBatch(
  slowPreflight,
  deps(
    async (target, item) => {
      actualDispatches.push({
        target: target.conversation.conversationID,
        message: item.messages[0].ID,
        at: Date.now(),
      });
    },
    {
      intervalMs: slowIntervalMs,
      prepare: async () => {
        preparedTargets++;
        if (preparedTargets > 4) return;
        if (preparedTargets === 4)
          setTimeout(releasePermissions, slowIntervalMs * 5);
        await permissionsReady;
      },
    }
  )
);
assert.equal(actualDispatches.length, 56);
for (let i = 1; i < actualDispatches.length; i++) {
  const gap = actualDispatches[i].at - actualDispatches[i - 1].at;
  assert.ok(
    gap >= slowIntervalMs - 1,
    `actual SDK dispatches after slow permission checks must be paced; gap=${gap}ms`
  );
}
for (const target of slowPreflight.targets) {
  assert.deepEqual(
    actualDispatches
      .filter((item) => item.target === target.conversation.conversationID)
      .map((item) => item.message),
    ["m0", "m1", "m2", "m3", "m4", "m5", "m6"]
  );
  assert.ok(target.items.every((item) => item.status === "success"));
}
console.log(
  "PASS: 100 targets bounded at 4; ordered per-message progress; precise retry; unknown/no-duplicate; late ACK; preflight failure; account switch; snapshots; actual dispatch pacing after simultaneous slow preflight completion (8 targets × 7 messages)."
);
