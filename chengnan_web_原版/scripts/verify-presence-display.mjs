import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import vm from 'node:vm';
import ts from 'typescript';

const read = (p) => readFileSync(new URL('../' + p, import.meta.url), 'utf8');
const compile = (s) => ts.transpileModule(s, { compilerOptions: { target: ts.ScriptTarget.ES2020, module: ts.ModuleKind.CommonJS } }).outputText;
const helper = { exports: {} };
vm.runInNewContext(compile(read('src/features/presence/last-online.ts')), helper);
const { resolveLastOnlineAt, formatLastOnlineText } = helper.exports;
const now = Date.parse('2026-11-09T12:00:00Z');
assert.equal(resolveLastOnlineAt({last_offline_at: '2026-11-09T11:55:00Z'}), '2026-11-09T11:55:00.000Z');
assert.equal(resolveLastOnlineAt({last_online_at: '2026-11-09T10:00:00Z'}, now), '2026-11-09T12:00:00.000Z');
assert.equal(formatLastOnlineText('invalid', now), '');
assert.equal(formatLastOnlineText(undefined, now), '');
const listSource = read('src/TUIKit/components/TUIConversation/conversation-list/index.vue');
const begin = listSource.indexOf('const getConversationOnlineText =');
const end = listSource.indexOf('\nfunction getConversationUserID', begin);
assert.ok(begin > 0 && end > begin);
const ctx = {
  isShowUserOnlineStatus: () => true,
  isConversationPeerOnline: () => false,
  getConversationUserID: () => 'peer',
  lastSeenByUser: {value: {peer: '2026-11-09T11:55:00Z'}},
  presenceNow: {value: now}, formatLastOnlineText,
};
vm.createContext(ctx);
vm.runInContext(compile(listSource.slice(begin, end) + '\nglobalThis.getLabel = getConversationOnlineText;'), ctx);
assert.equal(ctx.getLabel({}), '5分钟前在线');
ctx.presenceNow.value += 60000;
assert.equal(ctx.getLabel({}), '6分钟前在线');
ctx.isConversationPeerOnline = () => true;
assert.equal(ctx.getLabel({}), '在线');
ctx.isShowUserOnlineStatus = () => false;
assert.equal(ctx.getLabel({}), '');
ctx.isShowUserOnlineStatus = () => true;
ctx.isConversationPeerOnline = () => false;
ctx.lastSeenByUser.value = {};
assert.equal(ctx.getLabel({}), '离线');

// Execute the actual header request function, including stale-response protection.
const chat = read('src/TUIKit/components/TUIChat/index.vue');
const from = chat.indexOf('async function syncPeerLastOnline(');
const to = chat.indexOf('\nfunction syncOnlineStatusFromStore', from);
assert.ok(from > 0 && to > from);
const header = {
  peerLastOnlineRequestSeq: 0, peerPresenceDisposed: false, peerPresenceFetching: false,
  peerLastOnlineAt: {value: ''}, peerObservedOfflineAt: 0,
  getConversationPeerUserID: c => c.id, resolveLastOnlineAt, console,
  getUserLastOnline: async () => [{im_user_id: 'peer', last_offline_at: '2026-11-09T11:55:00Z'}],
};
vm.createContext(header);
vm.runInContext(compile(chat.slice(from, to)), header);
await header.syncPeerLastOnline({id: 'peer'});
assert.equal(header.peerLastOnlineAt.value, '2026-11-09T11:55:00.000Z');
let finish;
header.getUserLastOnline = () => new Promise(resolve => { finish = resolve; });
const pending = header.syncPeerLastOnline({id: 'old-peer'});
header.peerLastOnlineRequestSeq++;
header.peerLastOnlineAt.value = 'new-conversation';
finish([{im_user_id: 'old-peer', last_offline_at: '2026-11-09T11:00:00Z'}]);
await pending;
assert.equal(header.peerLastOnlineAt.value, 'new-conversation');
console.log('PASS: offline-only data, elapsed label updates, online/hidden/unknown states and stale conversation responses');
