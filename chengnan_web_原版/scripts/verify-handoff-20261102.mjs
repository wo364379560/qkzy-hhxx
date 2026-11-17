import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { createRequire } from 'node:module';
import ts from 'typescript';
const require = createRequire(import.meta.url);
const modules = new Map();
function load(relative, stubs = {}) {
  const key = relative;
  if (modules.has(key)) return modules.get(key);
  const source = readFileSync(new URL('../src/' + relative, import.meta.url), 'utf8');
  const code = ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2020 } }).outputText;
  const module = { exports: {} };
  new Function('require', 'module', 'exports', code)(name => Object.hasOwn(stubs, name) ? stubs[name] : require(name), module, module.exports);
  modules.set(key, module.exports); return module.exports;
}
const json = { JSONToObject: value => { try { return JSON.parse(value); } catch { return undefined; } } };
const protocol = load('features/call/doMattxCallProtocol.ts', { '@/TUIKit/utils/type-check': json });
const fixture = JSON.parse(readFileSync(new URL('../../TodayToTrackRunKt/app/src/test/resources/rui_call_v2.json', import.meta.url), 'utf8'));
const parsed = protocol.parseDoMattxCallSignal(fixture);
assert.equal(parsed.roomId, fixture.roomId); assert.equal(parsed.roomType, 'string');
assert.equal(protocol.parseDoMattxCallSignal({ businessID: 1, data: JSON.stringify(fixture) }).roomId, fixture.roomId);
assert.equal(protocol.parseDoMattxCallSignal({ ...fixture, version: 99 }), undefined);
assert.equal(protocol.parseDoMattxCallSignal({ ...fixture, expiresAtMs: 1 }), undefined);
const written = JSON.parse(protocol.stringifyDoMattxCallSignal({ mediaType: 2, roomId: 'rui_web_generated', caller: 'alice' }));
assert.equal(written.businessID, 'rtc_call_v1'); assert.equal(written.version, 2); assert.equal(written.roomType, 'string');
assert.equal(protocol.parseDoMattxCallSignal({ longdo_type: 'longdo_rtc_call_invite_v1', longdo_room_id: 123, longdo_media_type: 'longdo_video' }).roomType, 'number');
assert.equal(protocol.parseDoMattxCallSignal({ ...fixture, version: 1, roomType: undefined }).roomType, 'string');
const cards = load('TUIKit/components/TUIChat/utils/sharedCard.ts');
const groupCard = cards.parseSharedCard({ businessID: 'group_card_v1', groupId: '@TGS#_@TGS#actual', groupName: 'Test' });
assert.equal(groupCard.id, '@TGS#_@TGS#actual'); assert.deepEqual(cards.parseSharedCard(cards.createSharedCardPayload(groupCard)), { ...groupCard, version: 2 });
assert.equal(cards.parseSharedCard({ businessID: 'contact_card_v1', userId: 'alice', avatar: 'javascript:alert(1)' }).avatar, '');
assert.equal(cards.parseSharedCard({ businessID: 'group_card_v1', groupId: '' }), null);
console.log('PASS protocol: shared Android/WEB fixture, typed legacy rooms, expired/unknown versions, exact group ID/card forwarding and URL validation.');
const access = load('TUIKit/components/TUIChat/utils/sharedCardAccess.ts');
let joined = false, joinCalls = 0;
const publicGroup = { groupID: 'public', type: 'Public', joinOption: 'NeedPermission' };
const groupSDK = { getGroupProfile: async () => ({ data: { group: publicGroup } }), getGroupList: async () => ({ data: { groupList: joined ? [publicGroup] : [] } }) };
assert.equal((await access.readSharedGroupAccess(groupSDK, 'public', () => true)).member, false);
const approval = await access.joinSharedGroupCard(groupSDK, 'public', () => true, async () => { joinCalls++; return { code: 0 }; });
assert.equal(approval.pending, true); assert.equal(approval.member, false);
joined = true;
const memberJoin = await access.joinSharedGroupCard(groupSDK, 'public', () => true, async () => { joinCalls++; });
assert.equal(memberJoin.member, true); assert.equal(joinCalls, 1);
assert.equal(access.groupCardJoinMode({ type: 'Work' }, 'work'), 'invitation');
assert.equal(access.groupCardJoinMode({ type: 'Public', joinOption: 'DisableApply' }, 'blocked'), 'forbidden');
let scopeActive = true;
const staleSDK = { getGroupProfile: async () => { scopeActive = false; return { data: { group: publicGroup } }; } };
await assert.rejects(access.readSharedGroupAccess(staleSDK, 'public', () => scopeActive), error => error.name === 'AbortError');
await assert.rejects(access.readSharedGroupAccess({ getGroupProfile: async () => ({ code: 10007 }) }, 'public', () => true), error => error.code === 10007);
console.log('PASS group card: confirmed membership, pending approval, idempotent joined path, invitation-only/forbidden modes, stale reads and real permission code.');


let session = { sdkAppID: 1600163137, userID: 'alice', userSig: 'test-only-not-a-real-credential' };
const watched = new Set();
const store = {
  getData: (_store, key) => key === 'userProfile' ? { userID: session?.userID, nick: 'Alice' } : [],
  watch: (_store, options) => watched.add(options), unwatch: (_store, options) => watched.delete(options),
};
const engine = { chat: null, EVENT: { SDK_READY: 'ready', MESSAGE_RECEIVED: 'message' }, TYPES: { CONV_C2C: 'C2C', CONV_GROUP: 'GROUP' } };
function fakeChat() {
  const signals = new Map(), events = new Map();
  return { signals, events, ready: true, rejects: 0,
    isReady() { return this.ready; }, getLoginUser() { return session?.userID; },
    addSignalingListener: (key, callback) => signals.set(key, callback), removeSignalingListener: (key, callback) => { if (signals.get(key) === callback) signals.delete(key); },
    on: (key, callback) => events.set(key, callback), off: (key, callback) => { if (events.get(key) === callback) events.delete(key); },
    invite: async () => ({ data: { inviteID: 'outgoing-1' } }), accept: async () => {}, cancel: async () => {},
    reject() { this.rejects++; return Promise.resolve(); },
    createCustomMessage: options => ({ ID: 'record', conversationID: 'C2Cbob', ...options }), sendMessage: async message => ({ data: { message } }),
  };
}
const clouds = [];
class Cloud {
  static getTRTCShareInstance() { const value = new Cloud(); clouds.push(value); return value; }
  events = new Map(); params = null; audio = 0;
  on(event, callback) { this.events.set(event, callback); } off(event, callback) { if (this.events.get(event) === callback) this.events.delete(event); }
  emit(event, ...args) { this.events.get(event)?.(...args); }
  async enterRoom(params) { this.params = params; this.emit('onEnterRoom', 1); }
  async startLocalAudio() { this.audio++; } async stopLocalAudio() {} async stopLocalPreview() {} async exitRoom() {}
  async startLocalPreview() {} async startRemoteView() {} async stopRemoteView() {}
}
class Params { constructor(sdkAppId, userId, userSig, roomId, strRoomId) { Object.assign(this, { sdkAppId, userId, userSig, roomId, strRoomId }); } }
globalThis.window = { setTimeout, clearTimeout }; globalThis.document = { getElementById: () => null };
const flush = async () => { for (let i = 0; i < 12; i++) await Promise.resolve(); };
const bridge = load('features/call/doMattxCallBridge.ts', {
  vue: { reactive: value => value, nextTick: async () => {} },
  '@tencentcloud/chat-uikit-engine-lite': { default: engine, StoreName: { CHAT: 'chat', USER: 'user' }, TUIStore: store },
  '@tencentcloud/tui-core-lite': { default: { callService() {} }, TUIConstants: { TUIChat: { SERVICE: { NAME: 'chat', METHOD: { UPDATE_MESSAGE_LIST: 'update' } } } } },
  'trtc-cloud-js-sdk': { default: Cloud, TRTCParams: Params, TRTCAppScene: {}, TRTCAudioQuality: {}, TRTCRoleType: {}, TRTCVideoStreamType: {} },
  '@/features/auth/session': { useAuthSession: () => ({ currentSession: { value: session } }) },
  './localCallHistory': { rememberCallFromMessage() {}, rememberLocalCall() {} }, './doMattxCallProtocol': protocol,
});
assert.equal(bridge.ensureDoMattxCallBridge(), false);
const chat = fakeChat(); engine.chat = chat; chat.ready = false;
assert.equal(bridge.ensureDoMattxCallBridge(), false); assert.equal(chat.signals.size, 0);
chat.ready = true; chat.events.get('ready')();
assert.equal(chat.signals.size, 5); assert.equal(watched.size, 1);
bridge.ensureDoMattxCallBridge(); assert.equal(chat.signals.size, 5); assert.equal(watched.size, 1);
const priorHandler = chat.signals.get('newInvitationReceived');
const replacement = fakeChat(); engine.chat = replacement; bridge.ensureDoMattxCallBridge();
assert.equal(chat.signals.size, 0); assert.equal(replacement.signals.size, 5); assert.equal(watched.size, 1);
priorHandler({ inviteID: 'stale', inviter: 'bob', data: JSON.stringify(fixture) }); assert.equal(bridge.doMattxCallState.phase, 'idle');
replacement.signals.get('newInvitationReceived')({ inviteID: 'incoming-1', inviter: 'bob', data: JSON.stringify(fixture) });
assert.equal(bridge.doMattxCallState.phase, 'incoming');
replacement.signals.get('newInvitationReceived')({ inviteID: 'incoming-1', inviter: 'bob', data: JSON.stringify(fixture) }); assert.equal(replacement.rejects, 0);
await bridge.acceptDoMattxCall(); await flush();
assert.equal(clouds.at(-1).params.roomId, 0); assert.equal(clouds.at(-1).params.strRoomId, fixture.roomId);
assert.equal(bridge.doMattxCallState.phase, 'connected');
clouds.at(-1).emit('onUserVideoAvailable', 'bob', false); assert.equal(bridge.doMattxCallState.phase, 'connected');
clouds.at(-1).emit('onRemoteUserLeaveRoom', 'unrelated'); assert.equal(bridge.doMattxCallState.phase, 'connected');
clouds.at(-1).emit('onRemoteUserLeaveRoom', 'bob'); await flush(); assert.equal(bridge.doMattxCallState.phase, 'idle');
let finishAccept;
replacement.accept = () => new Promise(resolve => { finishAccept = resolve; });
replacement.signals.get('newInvitationReceived')({ inviteID: 'incoming-2', inviter: 'bob', data: JSON.stringify(fixture) });
const pendingAccept = bridge.acceptDoMattxCall(); const before = clouds.length;
await bridge.disposeDoMattxCallBridge(); finishAccept(); await pendingAccept;
assert.equal(clouds.length, before); assert.equal(replacement.signals.size, 0); assert.equal(watched.size, 0);
assert.equal(bridge.doMattxCallState.phase, 'idle');
replacement.accept = async () => {};
let finishReject;
const rejection = new Promise(resolve => { finishReject = resolve; });
replacement.reject = () => rejection;
bridge.ensureDoMattxCallBridge();
replacement.signals.get('newInvitationReceived')({ inviteID: 'incoming-old', inviter: 'bob', data: JSON.stringify(fixture) });
const pendingReject = bridge.rejectDoMattxCall();
await bridge.hangupDoMattxCall();
replacement.signals.get('newInvitationReceived')({ inviteID: 'incoming-new', inviter: 'bob', data: JSON.stringify(fixture) });
finishReject(); await pendingReject;
assert.equal(bridge.doMattxCallState.inviteID, 'incoming-new');
replacement.reject = async () => {};
await bridge.disposeDoMattxCallBridge();
console.log('PASS call lifecycle: slow SDK binding, exact instance cleanup, stale callback fencing, duplicate invite, matching string room, camera toggle and delayed accept after logout.');

const avatarSync = load('features/account/avatarSync.ts');
let account = 'alice', groupID = 'group-a', uploads = 0, sdkUpdates = 0, resolveUpload;
const avatarStore = { getData: () => ({ userID: account }) };
const groupProfile = load('features/group/profile.ts', {
  '@tencentcloud/chat-uikit-engine-lite': { TUIGroupService: { updateGroupProfile: async () => { sdkUpdates++; return { code: 0 }; } }, TUIStore: avatarStore, StoreName: { USER: 'user' } },
  '@/api': { uploadGroupAvatar: () => { uploads++; return new Promise(resolve => { resolveUpload = resolve; }); } }, '@/features/account/avatarSync': avatarSync,
});
const upload = groupProfile.uploadAndSyncGroupAvatar('group-a', {}, () => groupID === 'group-a');
groupID = 'group-b'; resolveUpload('https://example.invalid/already-uploaded.jpg');
await assert.rejects(upload, error => error.name === 'AbortError'); assert.equal(sdkUpdates, 0);
let failSync = true;
const personal = load('features/account/profile.ts', {
  '@tencentcloud/chat-uikit-engine-lite': { default: { TYPES: {} }, TUIStore: avatarStore, StoreName: { USER: 'user' }, TUIUserService: { updateMyProfile: async () => ({ code: 0 }) } },
  '@/api': { uploadAvatar: async () => { uploads++; return 'https://example.invalid/avatar.jpg'; }, updateProfile: async () => { if (failSync) throw Error('provider offline'); } }, './avatarSync': avatarSync,
});
let pending;
try { await personal.uploadAndSyncAvatar('alice', {}); } catch (error) { assert.ok(error instanceof avatarSync.AvatarSyncError); pending = error.avatarURL; }
const count = uploads; failSync = false; await personal.syncUploadedAvatar('alice', pending); assert.equal(uploads, count);
console.log('PASS avatar: switching groups discards the old upload callback; failed IM synchronization retries the saved URL without re-upload.');
