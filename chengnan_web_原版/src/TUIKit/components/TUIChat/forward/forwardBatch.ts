import { cloneDeep } from "lodash";
import type { SendForwardMessageOptions } from "@tencentcloud/chat-uikit-engine-lite";

type ForwardStatus = "pending" | "sending" | "retrying" | "success" | "failed" | "unknown";

export interface ForwardItem {
  messages: any[];
  status: ForwardStatus;
  error?: any;
  sdkMessage?: any;
  sendOptions?: SendForwardMessageOptions;
  settling?: boolean;
  retryCount?: number;
}

export interface ForwardTarget {
  conversation: any;
  name: string;
  items: ForwardItem[];
}

export interface ForwardBatch {
  targets: ForwardTarget[];
  merge: boolean;
  running: boolean;
  serverRequestID?: string;
  serverAccepted?: boolean;
  serverNotice?: string;
  serverState?: any;
  serverPlan?: any;
  lastRun?: { targetIDs: string[]; messageCount: number; skippedSuccess: number };
}

export function snapshotForwardMessages(
  ids: string[],
  getModel: (id: string) => any
) {
  const uniqueIDs = [...new Set(ids)];
  if (!uniqueIDs.length) throw new Error("未选择消息");
  return uniqueIDs.map((id) => {
    const model = getModel(id);
    if (!model || model.isDeleted || model.isRevoked) {
      throw new Error("部分原消息已失效，请返回聊天记录重新选择");
    }
    return cloneDeep(
      typeof model.getMessage === "function" ? model.getMessage() : model
    );
  });
}

interface LazyForwardItems {
  size: number;
  materialized: Map<number, ForwardItem>;
}
const lazyForwardItems = new WeakMap<ForwardItem[], LazyForwardItems>();
const itemIndex = (property: PropertyKey) => typeof property === "string" && /^(0|[1-9]\d*)$/.test(property) ? Number(property) : -1;

// Pending work stores only shared source snapshots. Reading a page or summary
// does not materialize the targets × messages product; only a status/SDK identity
// mutation creates a persistent per-delivery record.
function createLazyItems(messageGroups: any[][]): ForwardItem[] {
  const state: LazyForwardItems = { size: messageGroups.length, materialized: new Map() };
  const items = new Proxy([] as ForwardItem[], {
    has(target, property) {
      const index = itemIndex(property);
      return index >= 0 && index < state.size || Reflect.has(target, property);
    },
    get(target, property, receiver) {
      if (property === "length") return state.size;
      const index = itemIndex(property);
      if (index < 0 || index >= state.size) return Reflect.get(target, property, receiver);
      const existing = state.materialized.get(index);
      if (existing) return existing;
      const pending: ForwardItem = { messages: messageGroups[index], status: "pending" };
      return new Proxy(pending, {
        get(base, key) { return Reflect.get(state.materialized.get(index) || base, key); },
        set(base, key, value) {
          let record = state.materialized.get(index);
          if (!record) { record = { ...base }; state.materialized.set(index, record); }
          return Reflect.set(record, key, value);
        },
      });
    },
  });
  lazyForwardItems.set(items, state);
  return items;
}

export function hasSettlingForwardItems(target: ForwardTarget): boolean {
  const state = lazyForwardItems.get(target.items);
  for (const item of state ? state.materialized.values() : target.items) if (item.settling) return true;
  return false;
}

export function forwardTargetError(target: ForwardTarget): any {
  const state = lazyForwardItems.get(target.items);
  const items = state ? state.materialized.values() : target.items;
  for (const item of items) if (item.status === "failed" || item.status === "unknown") return item.error;
}

export function createForwardBatch(
  conversations: any[],
  messages: any[],
  merge: boolean,
  getName: (conversation: any) => string
): ForwardBatch {
  if (!messages.length) throw new Error("未选择消息");
  const sources = cloneDeep(messages);
  const groups = merge ? [sources] : sources.map(message => [message]);
  return {
    merge,
    running: false,
    targets: [
      ...new Map(conversations.map((c) => [c.conversationID, c])).values(),
    ].map((conversation) => ({
      conversation: cloneDeep(conversation),
      name: getName(conversation),
      items: createLazyItems(groups),
    })),
  };
}

// Numeric SDK/transport outcomes take precedence over wording in a wrapped
// message. The production preflight failure can contain no code at all; identify
// its exact websocket rejection signature without inventing a numeric code.
function errorCodes(error: any): number[] {
  const codes: number[] = [];
  const add = (value: any) => {
    if ((typeof value === "number" || typeof value === "string" && /^-?\d+$/.test(value)) && Number.isSafeInteger(Number(value)) && Number(value) !== 0)
      codes.push(Number(value));
  };
  add(error?.code);
  const pattern = /(?:["']code["']|\bcode\b)\s*:\s*["']?(-?\d+)/g;
  let match: RegExpExecArray | null;
  while ((match = pattern.exec(forwardErrorText(error)))) add(match[1]);
  return codes;
}

function forwardErrorText(error: any): string {
  return typeof error === "string" ? error : String(error?.message || "");
}

const rateCodes = new Set([60018, 10023]);
const transportCodes = new Set([2800, 2801, 2805, 60008]);
export function isForwardRateLimited(error: any) {
  if (error?.forwardTimeout || error?.unknown || error?.Unknown ||
      typeof error?.code === "string" && error.code && !/^-?\d+$/.test(error.code)) return false;
  const codes = errorCodes(error);
  // Any explicit non-rate rejection (including permission/friendship failures)
  // vetoes a stale frequency message nested inside that error.
  if (codes.some(code => !rateCodes.has(code))) return false;
  if (codes.length) return true;
  const text = forwardErrorText(error);
  if (/\b(?:timeout|timed\s*out|ECONNRESET|ETIMEDOUT|ERR_NETWORK)\b|network\s+error|connection\s+(?:closed|reset)/i.test(text)) return false;
  return /\bwebsocket\s+frequency\s+block\b/i.test(text);
}

function isDefiniteRejection(error: any) {
  if (error?.notDispatched) return true;
  const codes = errorCodes(error);
  if (codes.some(code => transportCodes.has(code)) || error?.forwardTimeout || error?.unknown || error?.Unknown)
    return false;
  if (isForwardRateLimited(error)) return true;
  return codes.some((code) =>
    [
      2024, 2105, 2453, 2454, 60018, 10023, 3123, 4004, 4005, 7004, 7005, 8001,
      80001, 80004, 10010, 10017, 20007, 20009, 120003,
    ].includes(code)
  );
}

export interface ForwardPacingPolicy {
  sendIntervalMs: number;
  prepareIntervalMs: number;
  concurrency: number;
}
const forwardingPolicy: ForwardPacingPolicy = {
  sendIntervalMs: 1000,
  prepareIntervalMs: 1000,
  concurrency: 4,
};

// Caller supplies this project's trusted server settings. Updating a policy
// never clears in-flight rate gates or a cooldown from a real SDK rejection.
export function configureForwardPacingPolicy(policy: Partial<ForwardPacingPolicy>) {
  const next = { ...forwardingPolicy, ...policy };
  if (![next.sendIntervalMs, next.prepareIntervalMs].every(value => Number.isFinite(value) && value >= 1 && value <= 60000) ||
      !Number.isInteger(next.concurrency) || next.concurrency < 1 || next.concurrency > 64)
    throw new Error("转发调度配置无效");
  Object.assign(forwardingPolicy, next);
}

// Snapshot only unfinished work. A retry must never rebuild targets/messages
// from the original picker selection. Keep the original item identities/ACKs.
export function getForwardPendingWork(batch: ForwardBatch) {
  return batch.targets.flatMap((target) => {
    const state = lazyForwardItems.get(target.items);
    let end = target.items.length, successes = 0;
    const records = () => state ? state.materialized.entries() : target.items.entries();
    for (const [index, item] of records()) {
      if (item.status !== "success" && (item.settling || !["pending", "failed"].includes(item.status))) end = Math.min(end, index);
    }
    for (const [index, item] of records()) if (index < end && item.status === "success") successes++;
    const length = end - successes;
    if (!length) return [];
    const items = {
      length,
      *[Symbol.iterator]() {
        for (let index = 0; index < end; index++) {
          const item = target.items[index];
          if (item.status === "success") continue;
          if (item.settling || !["pending", "failed"].includes(item.status)) return;
          yield item;
        }
      },
    };
    return [{ target, items }];
  });
}

export function canRetryForwardTarget(target: ForwardTarget) {
  return getForwardPendingWork({ targets: [target], merge: false, running: false }).length > 0;
}

export function summarizeForwardTarget(target: ForwardTarget) {
  const state = lazyForwardItems.get(target.items);
  const result = { total: target.items.length, success: 0, failed: 0, unknown: 0, pending: state ? state.size - state.materialized.size : 0, sending: 0, retrying: 0 };
  for (const item of state ? state.materialized.values() : target.items) result[item.status]++;
  return result;
}

// Shared per-window pacing also covers sequential messages within a target.
let nextSendAt = 0;
let nextPrepareAt = 0;
let prepareTail: Promise<unknown> = Promise.resolve();
let throttleUntil = 0;
let adaptiveIntervalMs = 0;
let adaptiveUntil = 0;
let lastSlowdownAt = 0;
let pacingTail: Promise<unknown> = Promise.resolve();

async function waitUntil(getTime: () => number, active: () => boolean, deadline: number) {
  while (active() && Date.now() < deadline) {
    const remaining = getTime() - Date.now();
    if (remaining <= 0) return true;
    // Account switches/unmounts stop a queued retry without waiting a full backoff.
    await new Promise((resolve) => setTimeout(resolve, Math.min(remaining, 200, deadline - Date.now())));
  }
  return false;
}

function slowDown(intervalMs: number, delayMs: number) {
  const now = Date.now();
  throttleUntil = Math.max(throttleUntil, now + delayMs);
  nextSendAt = Math.max(nextSendAt, throttleUntil);
  // Concurrent rejections from the same burst count as one slowdown. After
  // 30 seconds without throttling, the next dispatch resumes normal pacing.
  if (now >= adaptiveUntil) adaptiveIntervalMs = 0;
  if (now >= adaptiveUntil || now - lastSlowdownAt >= 1000) {
    adaptiveIntervalMs = Math.min(2000, Math.max(intervalMs, adaptiveIntervalMs) * 2);
    lastSlowdownAt = now;
  }
  adaptiveUntil = now + 30000;
}

async function waitForSend(intervalMs: number, active: () => boolean, deadline: number) {
  const turn = pacingTail.then(async () => {
    if (!await waitUntil(() => nextSendAt, active, deadline)) return false;
    const interval = Date.now() < adaptiveUntil ? Math.max(intervalMs, adaptiveIntervalMs) : intervalMs;
    nextSendAt = Date.now() + interval;
    return true;
  });
  pacingTail = turn;
  return turn;
}

// Permission checks have their own shared start budget, including normal (not
// yet throttled) batches. Wait outside the network timeout and recheck cooldown.
async function waitForPrepare(intervalMs: number, active: () => boolean, deadline: number) {
  const turn = prepareTail.then(async () => {
    if (!await waitUntil(() => Math.max(nextPrepareAt, throttleUntil), active, deadline)) return false;
    nextPrepareAt = Date.now() + intervalMs;
    return true;
  });
  prepareTail = turn;
  return turn;
}

async function timeout<T>(operation: Promise<T>, ms: number): Promise<T> {
  let timer: ReturnType<typeof setTimeout>;
  try {
    return await Promise.race([
      operation,
      new Promise<never>((_, reject) => {
        timer = setTimeout(
          () =>
            reject(
              Object.assign(new Error("发送结果待确认，请核对目标会话"), {
                forwardTimeout: true,
              })
            ),
          ms
        );
      }),
    ]);
  } finally {
    clearTimeout(timer!);
  }
}

export async function runForwardBatch(
  batch: ForwardBatch,
  dependencies: {
    prepare?: (target: ForwardTarget) => Promise<void>;
    send: (
      target: ForwardTarget,
      item: ForwardItem,
      merge: boolean
    ) => Promise<any>;
    resend: (message: any, item: ForwardItem) => Promise<any>;
    isActive: () => boolean;
    onChange: () => void;
    // Injectable budgets keep offline regression tests deterministic.
    timeoutMs?: number;
    prepareTimeoutMs?: number;
    intervalMs?: number;
    prepareIntervalMs?: number;
    concurrency?: number;
    retryDelayMs?: number;
    autoRetryCount?: number;
    retryBudgetMs?: number;
  }
) {
  // A timed-out SDK operation is still alive. Do not create another batch of
  // network sends until it settles; a timeout is never permission to duplicate.
  if (
    batch.running ||
    batch.targets.some(hasSettlingForwardItems)
  )
    return;
  const work = getForwardPendingWork(batch);
  if (!work.length) return;
  batch.lastRun = {
    targetIDs: work.map(({ target }) => target.conversation.conversationID),
    messageCount: work.reduce((sum, entry) => sum + entry.items.length, 0),
    skippedSuccess: batch.targets.reduce((sum, target) =>
      sum + summarizeForwardTarget(target).success, 0),
  };
  batch.running = true;
  let nextTarget = 0;
  let halted = false;
  const active = () => !halted && dependencies.isActive();
  const change = () => {
    if (dependencies.isActive()) dependencies.onChange();
  };
  const runTarget = async ({ target, items }: (typeof work)[number]) => {
    for (const item of items) {
      if (!active()) return;
      if (item.status === "success") continue;
      // An unknown delivery must be checked by the user, even if an SDK message
      // object exists. SDK resend is reserved for a confirmed rejection.
      if (item.settling || !["pending", "failed"].includes(item.status)) return;
      // Only explicit throttling is automatically retried. Permission/friend
      // rejections cannot heal with retries; unknown deliveries may duplicate.
      const retryLimit = Math.min(
        6,
        Math.max(0, dependencies.autoRetryCount ?? 6)
      );
      // Read and send commands have independent budgets. Test-only dependency
      // overrides do not change the trusted runtime policy for other batches.
      const intervalMs = dependencies.intervalMs ?? forwardingPolicy.sendIntervalMs;
      const prepareIntervalMs = dependencies.prepareIntervalMs ?? dependencies.intervalMs ?? forwardingPolicy.prepareIntervalMs;
      let retryDeadline = Infinity;
      let lastRateError: any;
      for (let attempt = 0; ; attempt++) {
        item.status = attempt ? "retrying" : "sending";
        item.retryCount = attempt;
        change();
        let dispatched = false;
        let slowedDown = false;
        const retryDelayMs = Math.min(20000, (dependencies.retryDelayMs ?? 2000) * 2 ** attempt);
        let operation: Promise<any> | undefined;
        const markFailure = (error: any) => {
          const sdkMessage = error?.data?.message;
          if (
            sdkMessage?.ID &&
            sdkMessage.conversationID === target.conversation.conversationID
          ) {
            item.sdkMessage = sdkMessage;
          }
          item.error = error;
          item.status =
            !dispatched || isDefiniteRejection(error) ? "failed" : "unknown";
          if (!slowedDown && isForwardRateLimited(error)) {
            slowedDown = true;
            slowDown(intervalMs, retryDelayMs);
          }
        };
        try {
          if (!active()) {
            item.status = "pending";
            change();
            return;
          }
          // Pause permission reads too: they share the SDK connection with sends.
          if (!await waitUntil(() => throttleUntil, active, retryDeadline)) {
            item.status = active() ? "failed" : "pending";
            item.error = lastRateError;
            change();
            return;
          }
          item.status = "sending";
          item.error = undefined;
          change();
          if (dependencies.prepare) {
            if (!await waitForPrepare(prepareIntervalMs, active, retryDeadline)) {
              item.status = active() ? "failed" : "pending";
              item.error = lastRateError;
              change();
              return;
            }
            await timeout(
              Promise.resolve().then(() => dependencies.prepare!(target)),
              dependencies.prepareTimeoutMs ?? 4000
            );
          }
          if (!active()) {
            item.status = "pending";
            change();
            return;
          }
          // Pace actual SDK dispatch, not permission-request starts. Several
          // slow group lookups can finish together and otherwise burst-send.
          const slot = await waitForSend(intervalMs, active, retryDeadline);
          if (!slot && active()) {
            item.status = "failed";
            item.error = lastRateError;
            change();
            return;
          }
          if (!active()) {
            item.status = "pending";
            change();
            return;
          }
          dispatched = true;
          item.settling = true;
          operation = Promise.resolve().then(() => {
            if (!dependencies.isActive()) {
              dispatched = false;
              throw new Error("账号已切换，未发送");
            }
            return item.sdkMessage
              ? dependencies.resend(item.sdkMessage, item)
              : dependencies.send(target, item, batch.merge);
          });
          // Keep observing the original SDK operation after our UI deadline. This
          // reconciles late success without creating a second message.
          operation.then(
            () => {
              item.settling = false;
              if (dependencies.isActive()) {
                item.status = "success";
                item.error = undefined;
                change();
              }
            },
            (error) => {
              item.settling = false;
              if (dependencies.isActive()) {
                markFailure(error);
                change();
              }
            }
          );
          await timeout(operation, dependencies.timeoutMs ?? 15000);
          if (!dependencies.isActive()) return;
          item.status = "success";
          item.error = undefined;
        } catch (error: any) {
          if (!dependencies.isActive()) return;
          markFailure(error);
          if (error?.forwardTimeout && dispatched) halted = true;
          if (error?.forwardTimeout && !dispatched)
            item.error = new Error(
              "群发言权限暂时无法确认，消息尚未发送，可重试"
            );
          change();
          if (
            isForwardRateLimited(error) &&
            attempt < retryLimit &&
            active()
          ) {
            lastRateError = error;
            if (!Number.isFinite(retryDeadline))
              retryDeadline = Date.now() + (dependencies.retryBudgetMs ?? 120000);
            if (throttleUntil < retryDeadline) continue;
          }
          return;
        }
        change();
        break;
      }
    }
  };
  try {
    await Promise.all(
      Array.from({ length: Math.min(Math.max(1, dependencies.concurrency ?? forwardingPolicy.concurrency), work.length) }, async () => {
        while (active() && nextTarget < work.length) {
          await runTarget(work[nextTarget++]);
        }
      })
    );
  } finally {
    batch.running = false;
    change();
  }
}
