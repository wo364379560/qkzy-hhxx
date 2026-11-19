import { reactive, nextTick } from 'vue';
import { withTimeout } from '@/features/auth/recovery';
import { createCallSignalInbox } from '@/features/call/callSignalInbox';
import TUICore, { TUIConstants } from '@tencentcloud/tui-core-lite';
import TUIChatEngine from '@tencentcloud/chat-uikit-engine-lite';
export const callIntegrationState = reactive({ status: 'idle' as 'idle' | 'loading' | 'ready' | 'error', loaded: false });
let inbox: ReturnType<typeof createCallSignalInbox> | null = null;
let inboxChat: any = null;
let inboxIdentity = '';
let retry: (() => Promise<void>) | null = null;

export function captureCallSignals(session: { sdkAppID: number; userID: string }, chat: any, active: () => boolean, scope = 0) {
  const identity = `${scope}:${session.sdkAppID}:${session.userID}`;
  if (inbox && inboxChat === chat && inboxIdentity === identity) return;
  // A replaced IM scope invalidates pending idle waits even if its login later
  // fails, and closes any old call without waiting for the new SDK_READY.
  if (inbox) resetCallIntegration();
  inboxChat = chat; inboxIdentity = identity;
  inbox = createCallSignalInbox(chat, { active, userID: session.userID,
    events: TUIChatEngine.EVENT as any, signaling: (TUIChatEngine as any).TSignaling || {
      NEW_INVITATION_RECEIVED: 'newInvitationReceived', INVITEE_ACCEPTED: 'ts_invitee_accepted',
      INVITEE_REJECTED: 'ts_invitee_rejected', INVITATION_CANCELLED: 'ts_invitation_cancelled',
      INVITATION_TIMEOUT: 'ts_invitation_timeout',
    } });
}

export function retryCallIntegration() { return retry?.() || Promise.resolve(); }

let integration: Promise<typeof import('@trtc/calls-uikit-vue')> | null = null;
let loaded: typeof import('@trtc/calls-uikit-vue') | null = null;
let engineReady: Promise<void> | null = null;
function waitForCallEngineReady() {
  // CallEngine's WASM constructor readiness is separate from Chat SDK_READY.
  // once('ready') also immediately fires when its WASM was initialized earlier.
  if (!engineReady) engineReady = import('@trtc/call-engine-lite-js').then(({ TUICallEngine }) =>
    new Promise<void>(resolve => TUICallEngine.once('ready', resolve))
  ).catch(error => { engineReady = null; throw error; });
  return withTimeout(engineReady, 15000);
}
type CallInitialization = { identity: string; inbox: ReturnType<typeof createCallSignalInbox>; promise: Promise<void>; cancelWait?: () => void };
let initialization: CallInitialization | null = null;
let engineInbox: ReturnType<typeof createCallSignalInbox> | null = null;
let generation = 0;
let queue: Promise<unknown> = Promise.resolve();
const callService = () => TUICore.getService(TUIConstants.TUICalling.SERVICE.NAME) as any;
function serial<T>(operation: () => Promise<T>) {
  const next = queue.catch(() => {}).then(operation);
  queue = next;
  return next;
}
/** TUI-core 1.0 renotifies ready logins synchronously during registration.
 * Intercept only CallKit 4.2's ChatCombine observer before evaluating its chunk;
 * unrelated chat plugins keep their normal immediate login notifications. The
 * installed-vendor regression test verifies this narrow observer contract. */
export function prepareCallIntegration() {
  if (!integration) {
    const register = TUICore.registerEvent;
    const managedObservers = new Set<any>();
    const loginEvent = TUIConstants.TUILogin.EVENT.LOGIN_STATE_CHANGED;
    const loginSuccess = TUIConstants.TUILogin.EVENT_SUB_KEY.USER_LOGIN_SUCCESS;
    const guardedRegister: typeof register = function (event, sub, observer: any) {
      const isCallObserver = observer?._callService &&
        typeof observer.onGetExtension === 'function' && typeof observer.onCall === 'function' &&
        typeof observer.getGroupAttributes === 'function' && typeof observer.getCallKitMessage === 'function';
      if (event === loginEvent && sub === loginSuccess && isCallObserver) {
        managedObservers.add(observer);
        return;
      }
      return register.call(TUICore, event, sub, observer);
    };
    TUICore.registerEvent = guardedRegister;
    integration = import('@trtc/calls-uikit-vue').then(async module => {
      loaded = module;
      const service = callService();
      // This service is driven exclusively by the serialized lifecycle below.
      if (service) for (const sub of [loginSuccess, TUIConstants.TUILogin.EVENT_SUB_KEY.USER_LOGOUT_SUCCESS]) {
        TUICore.unregisterEvent(loginEvent, sub, service);
      }
      for (const observer of managedObservers) {
        if (observer !== service) register.call(TUICore, loginEvent, loginSuccess, observer);
      }
      await waitForCallEngineReady();
      return module;
    }).catch(error => { integration = null; throw error; }).finally(() => {
      if (TUICore.registerEvent === guardedRegister) TUICore.registerEvent = register;
    });
  }
  return integration;
}

async function stopCallEngine(module: typeof import('@trtc/calls-uikit-vue')) {
  // destroyed() rejects during ringing/connected calls. The vendor's public
  // exit path chooses reject vs hangup and returns the local call state to idle.
  await module.TUICallKitAPI.handleExceptionExit();
  await callService()?.onNotifyEvent?.(TUIConstants.TUILogin.EVENT.LOGIN_STATE_CHANGED,
    TUIConstants.TUILogin.EVENT_SUB_KEY.USER_LOGOUT_SUCCESS);
  // onNotifyEvent swallows failures: confirm cleanup through the public API.
  await module.TUICallKitAPI.destroyed();
  engineInbox = null;
}

/** This queue never blocks IM login. Lightweight incoming signaling is already
 * installed; download, engine initialization and cleanup belong only to calls. */
export function startCallIntegration(session: { sdkAppID: number; userID: string; userSig: string }, isCurrent: () => boolean) {
  const identity = `${session.sdkAppID}:${session.userID}:${session.userSig}`;
  if (!inbox) captureCallSignals(session, TUIChatEngine.chat, isCurrent);
  const currentInbox = inbox!;
  if (initialization?.identity === identity && initialization.inbox === currentInbox) return initialization.promise;
  initialization?.cancelWait?.();
  const epoch = generation;
  retry = () => startCallIntegration(session, isCurrent);
  if (engineInbox !== currentInbox || callIntegrationState.status !== 'ready') callIntegrationState.status = 'loading';
  let current: CallInitialization;
  const active = () => epoch === generation && initialization === current && isCurrent();
  const promise = (async () => {
    const module = await prepareCallIntegration();
    const busy = () => !!module.TUIStore && engineInbox === currentInbox && module.TUIStore.getData(module.StoreName.CALL, module.NAME.CALL_STATUS) !== 'idle';
    const waitUntilIdle = () => new Promise<void>(resolve => {
      const finish = () => {
        module.TUIStore.unwatch(module.StoreName.CALL, watch);
        if (current.cancelWait === finish) current.cancelWait = undefined;
        resolve();
      };
      const watch = { [module.NAME.CALL_STATUS]: (status: unknown) => { if (status === 'idle') finish(); } };
      current.cancelWait = finish;
      module.TUIStore.watch(module.StoreName.CALL, watch);
      if (!busy() || !active()) finish();
    });
    // UserSig rotation must not end a ringing/connected call. Wait outside the
    // lifecycle queue so logout can still reject/hang up and destroy promptly.
    while (active()) {
      if (busy()) { await waitUntilIdle(); continue; }
      const completed = await serial(async () => {
        if (!active()) return true;
        if (busy()) return false; // A new invitation may arrive before our turn.
        callIntegrationState.status = 'loading';
        currentInbox.suspend();
        try {
          await stopCallEngine(module);
          if (!active()) return true;
          await module.TUICallKitAPI.init({ SDKAppID: session.sdkAppID, userID: session.userID,
            userSig: session.userSig, tim: currentInbox.chat, isFromChat: true } as any);
          if (!active()) { await stopCallEngine(module); return true; }
          engineInbox = currentInbox;
          // Public init propagates errors; this idempotent vendor notification
          // only adds its chat/group-attribute hooks after successful init.
          await callService()?.onNotifyEvent?.(TUIConstants.TUILogin.EVENT.LOGIN_STATE_CHANGED,
            TUIConstants.TUILogin.EVENT_SUB_KEY.USER_LOGIN_SUCCESS);
          if (!active()) { await stopCallEngine(module); return true; }
          callIntegrationState.loaded = true;
          await nextTick();
          if (!active()) return true;
          currentInbox.activate();
          callIntegrationState.status = 'ready';
          return true;
        } catch (error) {
          await stopCallEngine(module).catch(() => {});
          throw error;
        }
      });
      if (completed) return;
    }
  })();
  current = { identity, inbox: currentInbox, promise };
  initialization = current;
  void promise.catch(() => { if (initialization === current) { initialization = null; callIntegrationState.status = 'error'; } });
  return promise;
}

export function resetCallIntegration() {
  ++generation;
  initialization?.cancelWait?.();
  inbox?.dispose(); inbox = null; inboxChat = null; inboxIdentity = '';
  retry = null; callIntegrationState.status = 'idle'; callIntegrationState.loaded = false;
  initialization = null;
  // Do not wait for a blocked call chunk (or call engine) before logging out IM.
  // A late initializer is generation-checked and its cleanup precedes the next.
  void serial(async () => {
    if (!loaded) return;
    await stopCallEngine(loaded);
  }).catch(() => { console.warn('通话清理未确认，普通聊天不受影响'); });
}
