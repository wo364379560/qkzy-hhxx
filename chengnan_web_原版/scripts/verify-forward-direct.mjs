import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { createRequire } from "node:module";
import vm from "node:vm";
import ts from "typescript";

const require = createRequire(import.meta.url);
function load(file) {
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
}
const { createForwardBatch, runForwardBatch } = load("forwardBatch");
const { createCapturedForwardSender } = load("forwardDispatch");
const texts = ["11", "22", "33", "44", "55", "66", "最后一条消息来测试777"];
const messages = texts.map((text, i) => ({
  ID: `original-${i}`,
  payload: { text },
}));
let reads = 0,
  created = 0,
  loginUser = "alice";
let rejection = () => undefined;
const attempts = [],
  delivered = new Map();
const forbiddenRead = () => {
  reads++;
  throw new Error("unexpected permission/profile query");
};
const dispatch = async (message) => {
  attempts.push({ id: message.ID, to: message.to, text: message.payload.text });
  const error = rejection(message);
  if (error) throw { ...error, data: { message } };
  delivered.set(message.to, [
    ...(delivered.get(message.to) || []),
    message.payload.text,
  ]);
  message.status = "success";
  return { data: { message } };
};
const sdk = {
  getLoginUser: () => loginUser,
  getGroupProfile: forbiddenRead,
  getGroupMemberProfile: async () => ({data:{memberList:[{userID:loginUser,role:'Admin',muteUntil:0}]}}),
  getUserProfile: forbiddenRead,
  checkFriend: forbiddenRead,
  getBlacklist: forbiddenRead,
  createForwardMessage: (options) => ({
    ...options,
    payload: { ...options.payload.payload },
    ID: `forward-${++created}`,
    conversationID: options.conversationType + options.to,
  }),
  sendMessage: dispatch,
  resendMessage: dispatch,
};
const sender = createCapturedForwardSender(
  sdk,
  () => {},
  () => true,
  "alice"
);
const deps = {
  prepare: sender.prepare,
  isActive: () => true,
  onChange: () => {},
  intervalMs: 0,
  retryDelayMs: 0,
  send: sender.send,
  resend: sender.resend,
};
const many = createForwardBatch(
  Array.from({ length: 69 }, (_, i) => ({
    conversationID: i < 60 ? `GROUPgroup${i}` : `C2Cfriend${i}`,
    // Local profile flags must not preclude the SDK call.
    groupProfile: { muteAllMembers: true },
    isFriend: false,
    isBlocked: true,
  })),
  messages,
  false,
  (target) => target.conversationID
);
let rejectedOnce = false;
rejection = (message) => {
  if (
    !rejectedOnce &&
    message.to === "group0" &&
    message.payload.text === "33"
  ) {
    rejectedOnce = true;
    // Same wrapped frequency error seen in the production reproduction.
    return {
      message:
        'send failed: {"message":"websocket frequency block","code":60018}',
    };
  }
};
await runForwardBatch(many, deps);
assert.equal(
  reads,
  0,
  "69 × 7 forwarding performs no extra profile/friend/block/mute queries"
);
assert.equal(
  created,
  483,
  "retry reuses the failed SDK message, not a new message"
);
assert.equal(
  attempts.length,
  484,
  "only the explicitly rejected message is retried once"
);
assert.equal(delivered.size, 69);
for (const list of delivered.values()) assert.deepEqual(list, texts);
assert.ok(
  many.targets.every((target) =>
    target.items.every((item) => item.status === "success")
  )
);
const retried = attempts.filter(
  (attempt) => attempt.to === "group0" && attempt.text === "33"
);
assert.equal(retried.length, 2);
assert.equal(retried[0].id, retried[1].id);

attempts.length = 0;
delivered.clear();
let stillRejected = true;
rejection = (message) => {
  if (message.to === "blocked" && stillRejected) return { code: 10017 };
  if (message.to === "uncertain") return { code: 2800 };
};
const mixed = createForwardBatch(
  [
    { conversationID: "GROUPblocked" },
    { conversationID: "GROUPuncertain" },
    { conversationID: "C2Cgood" },
  ],
  messages.slice(0, 2),
  false,
  (target) => target.conversationID
);
await runForwardBatch(mixed, deps);
assert.equal(
  attempts.filter((attempt) => attempt.to === "blocked").length,
  1,
  "mute rejection must not consume automatic retry attempts"
);
assert.equal(
  attempts.filter((attempt) => attempt.to === "uncertain").length,
  1,
  "uncertain network result must never be retried automatically"
);
assert.deepEqual(
  Array.from(mixed.targets[0].items, (item) => item.status),
  ["failed", "pending"]
);
assert.deepEqual(
  Array.from(mixed.targets[1].items, (item) => item.status),
  ["unknown", "pending"]
);
assert.deepEqual(delivered.get("good"), texts.slice(0, 2));
stillRejected = false;
await runForwardBatch(mixed, deps);
assert.deepEqual(delivered.get("blocked"), texts.slice(0, 2));
assert.equal(
  attempts.filter((attempt) => attempt.to === "good").length,
  2,
  "manual retry skips successful messages"
);
assert.equal(
  attempts.filter((attempt) => attempt.to === "uncertain").length,
  1
);
assert.equal(reads, 0);

// Switching accounts while an automatic retry is queued cannot dispatch as the new user.
attempts.length = 0;
loginUser = "alice";
rejection = () => {
  loginUser = "bob";
  return { code: 10017 };
};
const switched = createForwardBatch(
  [{ conversationID: "GROUPswitch" }],
  messages.slice(0, 1),
  false,
  () => "switch"
);
await runForwardBatch(switched, deps);
assert.equal(attempts.length, 1);
assert.equal(switched.targets[0].items[0].status, "failed");
console.log(
  "PASS: direct SDK 69 × 7, personal mute checks coalesced; same-message throttle recovery; ordered delivery; permanent failure retained without automatic retry; unknown/success never repeated; account switch blocked."
);
