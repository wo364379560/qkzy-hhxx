import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { build } from 'esbuild';

const root = resolve(process.argv[2] || '.');
const base = resolve(root, 'src/TUIKit/components/TUIChat/utils');
async function load(file) {
  const result = await build({ entryPoints: [resolve(base, file)], bundle: true, platform: 'node', format: 'esm', write: false });
  return import(`data:text/javascript;base64,${Buffer.from(result.outputFiles[0].text).toString('base64')}`);
}
const { installScopedHistory } = await load('scopedHistory.ts');
const { dedupeCallSummaries } = await load('dedupeCallSummaries.ts');
const { messageOffsets, messageWindow, MAX_RENDERED_MESSAGES } = await load('messageWindow.ts');
let checks = 0;
const check = (condition, message) => { assert.ok(condition, message); checks++; };

function harness() {
  const state = { conv: { currentConversationID: 'GROUP_A' }, chat: { messageList: [], nextReqMessageID: '', isCompleted: false, userInfo: {} } };
  const watchers = new Map();
  const store = {
    watch(scope, callbacks) {
      for (const [key, callback] of Object.entries(callbacks)) {
        const id = scope + key;
        watchers.set(id, [...(watchers.get(id) || []), callback]);
        callback(state[scope][key]);
      }
    },
    getData: (scope, key) => state[scope][key],
    update(scope, key, value) {
      state[scope][key] = value;
      for (const fn of watchers.get(scope + key) || []) fn(value);
    },
    getConversationModel: () => ({ operationType: 0 }),
  };
  const calls = [];
  const fetch = options => new Promise((resolve, reject) => calls.push({ options, resolve, reject }));
  const engine = { chat: { isReady: () => true, getMessageList: fetch, getMessageListHopping: fetch } };
  let extras = 0;
  const service = {
    hoppingConfigMap: new Map(), updateLocalMessage: () => {},
    updateMessageList: list => { state.chat.messageList = [...list, ...state.chat.messageList]; },
    getMessageReactions: () => { extras++; },
    readReceiptHandler: { getMessageReadReceiptList: () => { extras++; } },
  };
  installScopedHistory(engine, service, store, { CONV: 'conv', CHAT: 'chat' });
  const switchTo = id => {
    store.update('conv', 'currentConversationID', id);
    state.chat = { messageList: [], nextReqMessageID: '', isCompleted: false, userInfo: {} };
    service.hoppingConfigMap.clear();
  };
  const response = (id, cursor = id + '_next', completed = false) => ({ data: { messageList: [{ ID: id + '_msg', conversationID: id }], nextReqMessageID: cursor, isCompleted: completed, nextMessageSeq: 10 } });
  return { state, calls, engine, service, store, switchTo, response, extras: () => extras };
}

for (const kind of ['getMessageList', 'getMessageListHopping']) {
  const h = harness();
  const a = h.service[kind]({ conversationID: 'GROUP_A', sequence: 20 });
  await Promise.resolve();
  h.switchTo('GROUP_B');
  const b = h.service[kind]({ conversationID: 'GROUP_B', sequence: 20 });
  await Promise.resolve();
  h.calls[1].resolve(h.response('GROUP_B'));
  await b;
  const before = JSON.stringify(h.state.chat);
  const beforeHop = [...h.service.hoppingConfigMap.entries()];
  h.calls[0].resolve(h.response('GROUP_A', '', true));
  check((await a).stale, `${kind}: marks old response stale`);
  assert.equal(JSON.stringify(h.state.chat), before); checks++;
  assert.deepEqual([...h.service.hoppingConfigMap.entries()], beforeHop); checks++;
}
{
  const h = harness();
  const a = h.service.getMessageList();
  await Promise.resolve();
  h.switchTo('GROUP_B'); h.switchTo('GROUP_A');
  h.calls[0].resolve(h.response('GROUP_A', 'obsolete', true));
  check((await a).stale, 'A -> B -> A invalidates the original visit');
  check(!h.state.chat.isCompleted, 'old visit cannot complete new visit');
}
{
  const h = harness();
  const a = h.service.getMessageList();
  check(a === h.service.getMessageList(), 'same page shares one pending request');
  await Promise.resolve();
  check(h.calls.length === 1, 'one SDK submission');
  h.calls[0].resolve(h.response('GROUP_A'));
  await a;
  check(h.state.chat.nextReqMessageID === 'GROUP_A_next' && h.extras() === 2, 'valid page, pagination and optional metadata commit');
  const b = h.service.getMessageList(); await Promise.resolve();
  check(h.calls[1].options.nextReqMessageID === 'GROUP_A_next', 'next page uses current cursor');
  h.calls[1].reject(new Error('offline'));
  await assert.rejects(b, /offline/); checks++;
  const retry = h.service.getMessageList(); await Promise.resolve();
  check(h.calls.length === 3, 'failed page releases pending slot');
  h.calls[2].resolve(h.response('GROUP_A')); await retry;
}
{
  const h = harness();
  const a = h.service.getMessageList(); await Promise.resolve();
  h.switchTo('GROUP_B'); h.calls[0].reject(new Error('old error'));
  check((await a).stale, 'old request failure does not affect the new group');
  const b = h.service.getMessageList(); await Promise.resolve();
  h.store.update('chat', 'messageSource', { ID: 'quoted', conversationID: 'GROUP_B' });
  h.calls[1].resolve(h.response('GROUP_B'));
  check((await b).stale, 'quote jump supersedes the ordinary page');
  h.service.hoppingConfigMap.set('nextMessageSeq', 30);
  const jump = h.service.getMessageList(); await Promise.resolve();
  check(h.calls[2].options.sequence === 30, 'continuing a quote jump uses its hopping cursor');
  h.calls[2].resolve(h.response('GROUP_B')); await jump;
}
{
  const h = harness();
  const a = h.service.getMessageList(); await Promise.resolve();
  h.engine.chat = { ...h.engine.chat };
  h.calls[0].resolve(h.response('GROUP_A'));
  check((await a).stale, 'replaced SDK session discards prior responses');
}

// Reference implementation preserves the previous caller-preference and
// tolerance semantics, including timestamps that are not in sorted order.
function reference(list) {
  const result = [];
  for (const next of list) {
    const index = result.findIndex(previous => previous.call && next.call
      && previous.conversationID === next.conversationID && previous.call.key === next.call.key
      && (!(previous.call.duration > 0 || next.call.duration > 0) || Math.abs(previous.call.duration - next.call.duration) <= 3)
      && Math.abs(previous.time - next.time) <= 5);
    if (index < 0) { result.push(next); continue; }
    const previous = result[index];
    const oldCaller = previous.call.caller, newCaller = next.call.caller;
    const oldIsCaller = Boolean(oldCaller && oldCaller === previous.from);
    const newIsCaller = Boolean(newCaller && newCaller === next.from);
    if (oldIsCaller !== newIsCaller ? newIsCaller : oldCaller && !newCaller && next.from === oldCaller) result[index] = next;
  }
  return result;
}
let seed = 42;
const random = max => { seed = (seed * 1664525 + 1013904223) >>> 0; return seed % max; };
for (let round = 0; round < 30; round++) {
  const list = Array.from({ length: 500 }, (_, ID) => ({ ID, conversationID: `GROUP_${random(3)}`, time: random(100), from: `u${random(3)}`,
    call: random(4) ? { key: `kind${random(2)}`, duration: random(30), caller: random(3) ? `u${random(3)}` : '' } : undefined }));
  assert.deepEqual(dedupeCallSummaries(list, message => message.call), reference(list)); checks++;
}
let descriptions = 0;
const ordinary = Array.from({ length: 10000 }, (_, ID) => ({ ID, time: ID }));
check(dedupeCallSummaries(ordinary, () => { descriptions++; }).length === 10000 && descriptions === 10000,
  '10,000 ordinary messages are classified once each, not pairwise');

const ids = ordinary.map(message => String(message.ID));
const offsets = messageOffsets(ids, new Map([['5000', 600]]));
for (const position of [0, 500, 500000, offsets[10000] - 600]) {
  const window = messageWindow(offsets, position, 600);
  check(window.end - window.start <= MAX_RENDERED_MESSAGES, 'render count remains bounded for 10,000 messages');
  check(window.top + (offsets[window.end] - offsets[window.start]) + window.bottom === offsets[10000], 'spacers preserve scroll extent');
}
const component = readFileSync(resolve(root, 'src/TUIKit/components/TUIChat/message-list/index.vue'), 'utf8');
check(component.includes('virtualRange.value.start, virtualRange.value.end'), 'desktop uses the virtual slice');
console.log(`Chat history isolation, deduplication and virtual geometry: ${checks} checks passed (${root})`);
