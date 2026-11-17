import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { createRequire } from "node:module";
import vm from "node:vm";
import ts from "typescript";
const require = createRequire(import.meta.url);
const load = (file) => {
  const exports = {};
  vm.runInNewContext(
    ts.transpileModule(
      readFileSync(
        new URL(
          `../src/TUIKit/components/TUIChat/forward/${file}.ts`,
          import.meta.url
        ),
        "utf8"
      ),
      {
        compilerOptions: {
          target: ts.ScriptTarget.ES2020,
          module: ts.ModuleKind.CommonJS,
        },
      }
    ).outputText,
    { exports, require, setTimeout, clearTimeout, Promise, Date, console }
  );
  return exports;
};
const { createCapturedForwardSender } = load("forwardDispatch");
const { createForwardBatch, runForwardBatch } = load("forwardBatch");
let loginUser = "alice",
  active = true,
  sent = [],
  updates = [],
  release;
const sdk = {
  getLoginUser: () => loginUser,
  getGroupMemberProfile: async () => ({data:{memberList:[{userID:loginUser,role:'Admin',muteUntil:0}]}}),
  getUserProfile: () => {
    throw new Error("forwarding must never await a global profile lookup");
  },
  createForwardMessage: (options) => ({
    ...options,
    ID: "forward",
    kind: "single",
  }),
  createMergerMessage: (options) => ({
    ...options,
    ID: "merger",
    kind: "merge",
  }),
  sendMessage: (message, options) => {
    sent.push({ message, options, user: loginUser });
    return new Promise((resolve) => {
      release = () => resolve({ data: { message } });
    });
  },
  resendMessage: () => {
    throw new Error("unexpected resend");
  },
};
const sender = createCapturedForwardSender(
  sdk,
  (messages, mode) => updates.push({ messages, mode }),
  () => active,
  "alice"
);
const target = {
  conversation: { type: "GROUP", conversationID: "GROUPtarget" },
};
const item = {
  messages: [
    {
      ID: "source",
      from: "alice",
      nick: "Alice",
      conversationType: "C2C",
      payload: { text: "original" },
      cloudCustomData: '{"reply":"keep"}',
    },
  ],
  sendOptions: {
    needMerge: true,
    params: { needReadReceipt: true },
    offlinePushInfo: { title: "push" },
  },
};
await sender.prepare(target);
const sending = sender.send(target, item, true);
assert.equal(
  sent.length,
  1,
  "dispatch after personal mute check"
);
assert.equal(sent[0].message.kind, "merge");
assert.equal(sent[0].message.needReadReceipt, true);
assert.equal(sent[0].options.offlinePushInfo.title, "push");
item.messages[0].payload.text = "edited later";
assert.equal(sent[0].message.payload.messageList[0].payload.text, "original");
assert.equal(
  sent[0].message.payload.messageList[0].cloudCustomData,
  '{"reply":"keep"}'
);
loginUser = "bob";
release();
await sending;
assert.equal(
  updates.length,
  1,
  "late old-account SDK ACK cannot edit new-account message list"
);
assert.throws(
  () => sender.send(target, item, true),
  (error) => error.notDispatched === true,
  "SDK instance reused for a new login must be rejected even if UI profile lags"
);
// Reproduce an account switch while an awaited profile/preflight dependency is
// outstanding. Its late resolution must not reach either account's SDK send.
loginUser = "alice";
sent = [];
updates = [];
let releaseProfile;
const profile = new Promise((resolve) => {
  releaseProfile = resolve;
});
const batch = createForwardBatch(
  [target.conversation],
  item.messages,
  true,
  () => "target"
);
const run = runForwardBatch(batch, {
  intervalMs: 0,
  prepare: () => profile,
  isActive: () => active,
  onChange: () => {},
  send: (target, item, merge) => sender.send(target, item, merge),
  resend: sender.resend,
});
await new Promise((resolve) => setTimeout(resolve, 0));
loginUser = "bob";
releaseProfile();
await run;
assert.equal(sent.length, 0);
assert.equal(batch.targets[0].items[0].status, "failed");
// A synchronous store watcher may also change identity during bubble insertion.
loginUser = "alice";
const switchingSender = createCapturedForwardSender(
  sdk,
  () => {
    loginUser = "bob";
  },
  () => true,
  "alice"
);
await switchingSender.prepare(target);
assert.throws(
  () => switchingSender.send(target, item, false),
  (error) => error.notDispatched === true
);
assert.equal(sent.length, 0);
console.log(
  "PASS: captured raw SDK single/merge dispatch, snapshots/options, permission-read account switch, reused SDK identity, late ACK store isolation."
);
