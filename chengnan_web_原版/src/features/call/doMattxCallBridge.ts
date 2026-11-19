import { nextTick, reactive } from "vue";
import TUIChatEngine, {
  StoreName,
  TUIStore,
  type IMessageModel,
} from "@tencentcloud/chat-uikit-engine-lite";
import TUICore, { TUIConstants } from "@tencentcloud/tui-core-lite";
import type TRTCCloud from "trtc-cloud-js-sdk";
let rtcModule: Promise<typeof import("trtc-cloud-js-sdk")> | null = null;
function loadRTC() {
  if (!rtcModule) rtcModule = import("trtc-cloud-js-sdk").catch(error => { rtcModule = null; throw error; });
  return rtcModule;
}
import { useAuthSession } from "@/features/auth/session";
import { rememberCallFromMessage, rememberLocalCall } from "./localCallHistory";
import {
  createDoMattxRoomId,
  DoMattxCallMediaType,
  getDoMattxCallMediaLabel,
  getDoMattxCallSignalFromMessage,
  getDoMattxCallSummaryText,
  isDoMattxCallSignalMessage,
  parseDoMattxCallSignal,
  stringifyDoMattxCallSignal,
  stringifyDoMattxCallSummary,
} from "./doMattxCallProtocol";

type DoMattxCallPhase = "idle" | "incoming" | "outgoing" | "connecting" | "connected";

interface SignalingEventPayload {
  inviteID?: string;
  inviteId?: string;
  inviter?: string;
  invitee?: string;
  groupID?: string;
  inviteeList?: string[];
  data?: string;
}

interface DoMattxCallState {
  visible: boolean;
  phase: DoMattxCallPhase;
  inviteID: string;
  peerUserID: string;
  peerName: string;
  peerAvatar: string;
  mediaType: DoMattxCallMediaType;
  roomId: string;
  roomType: "string" | "number";
  isCaller: boolean;
  connectedAt: number;
  remoteUserID: string;
  remoteVideoAvailable: boolean;
  micMuted: boolean;
  cameraMuted: boolean;
  error: string;
}

type DoMattxCallSnapshot = Pick<
  DoMattxCallState,
  | "inviteID"
  | "peerUserID"
  | "peerName"
  | "peerAvatar"
  | "mediaType"
  | "roomId"
  | "roomType"
  | "isCaller"
  | "connectedAt"
>;

const SIGNALING_EVENT = {
  NEW_INVITATION_RECEIVED: "newInvitationReceived",
  INVITEE_ACCEPTED: "ts_invitee_accepted",
  INVITEE_REJECTED: "ts_invitee_rejected",
  INVITATION_CANCELLED: "ts_invitation_cancelled",
  INVITATION_TIMEOUT: "ts_invitation_timeout",
};

const LOCAL_VIDEO_DOM_ID = "do-mattx-local-video";
const REMOTE_VIDEO_DOM_ID = "do-mattx-remote-video";
const CALL_TIMEOUT_SECONDS = 45;
const TENCENT_SIGNALING_BUSINESS_ID = 1;
const TENCENT_SIGNALING_ACTION_ACCEPT = 3;

let boundChat: any = null;
let boundIdentity = '';
let bindingEpoch = 0;
let callEpoch = 0;
let waitingChat: any = null;
let waitingReady: (() => void) | null = null;
let detachBindings: (() => void) | null = null;
let rtcCleanup = Promise.resolve();
const rtcBindings = new Map<TRTCCloud, Array<[string, (...args: any[]) => void]>>();
let trtc: TRTCCloud | null = null;
let callTimer: number | undefined;
const sentCallRecordKeys = new Set<string>();

export const doMattxCallState = reactive<DoMattxCallState>({
  visible: false,
  phase: "idle",
  inviteID: "",
  peerUserID: "",
  peerName: "",
  peerAvatar: "",
  mediaType: DoMattxCallMediaType.Audio,
  roomId: "",
  roomType: "string",
  isCaller: false,
  connectedAt: 0,
  remoteUserID: "",
  remoteVideoAvailable: false,
  micMuted: false,
  cameraMuted: false,
  error: "",
});

function clearWaitingChat() {
  if (waitingChat && waitingReady) waitingChat.off?.((TUIChatEngine as any).EVENT?.SDK_READY || 'onSdkReady', waitingReady);
  waitingChat = null; waitingReady = null;
}

function currentIdentity() { const session = getCurrentSession(); return session?.userID ? `${session.sdkAppID}:${session.userID}` : ''; }

export function ensureDoMattxCallBridge(earlyChat?: any): boolean {
  const chat = earlyChat || getChat();
  const identity = currentIdentity();
  if (!chat || !identity) return false;
  if (boundChat && (boundChat !== chat || boundIdentity !== identity)) {
    ++bindingEpoch; detachBindings?.(); detachBindings = null; boundChat = null; boundIdentity = '';
    void endLocalCall();
  }
  if (typeof chat.addSignalingListener !== 'function') return false;
  const session = getCurrentSession();
  const ready = () => (typeof chat.isReady !== 'function' || chat.isReady()) &&
    (typeof chat.getLoginUser !== 'function' || chat.getLoginUser() === session?.userID);
  clearWaitingChat();
  if (boundChat === chat && boundIdentity === identity) return ready();
  detachBindings?.();
  if (boundIdentity && boundIdentity !== identity) void endLocalCall();
  const epoch = ++bindingEpoch;
  const active = () => epoch === bindingEpoch && boundChat === chat && currentIdentity() === identity;
  const events: Array<[string, (...args: any[]) => void]> = [
    [getSignalingEventName('NEW_INVITATION_RECEIVED'), onNewInvitationReceived],
    [getSignalingEventName('INVITEE_ACCEPTED'), onInviteeAccepted],
    [getSignalingEventName('INVITEE_REJECTED'), onInviteeRejected],
    [getSignalingEventName('INVITATION_CANCELLED'), onInvitationCancelled],
    [getSignalingEventName('INVITATION_TIMEOUT'), onInvitationTimeout],
  ];
  const registered = events.map(([event, handler]) => {
    const guarded = (...args: any[]) => { if (active()) void handler(...args); };
    chat.addSignalingListener(event, guarded); return [event, guarded] as const;
  });
  const messageEvent = (TUIChatEngine as any).EVENT?.MESSAGE_RECEIVED || 'onMessageReceived';
  const messageHandler = (event: any) => { if (active()) void onSdkMessageReceived(event); };
  const watched = { newMessageList: (messages: IMessageModel[]) => { if (active()) void onNewMessageList(messages); } };
  chat.on?.(messageEvent, messageHandler);
  TUIStore.watch(StoreName.CHAT, watched);
  detachBindings = () => {
    registered.forEach(([event, handler]) => chat.removeSignalingListener?.(event, handler));
    chat.off?.(messageEvent, messageHandler);
    TUIStore.unwatch(StoreName.CHAT, watched);
  };
  boundChat = chat; boundIdentity = identity;
  return ready();
}

export async function disposeDoMattxCallBridge() {
  const chat = boundChat;
  const snapshot = getCallSnapshot();
  const phase = doMattxCallState.phase;
  if (chat && snapshot.inviteID && (phase === 'outgoing' || phase === 'incoming')) {
    try {
      const data = stringifyDoMattxCallSignal({ mediaType: snapshot.mediaType, roomId: snapshot.roomId, roomType: snapshot.roomType,
        caller: chat.getLoginUser?.() || '', action: phase === 'outgoing' ? 'cancel' : 'reject' });
      const notification = phase === 'outgoing' ? chat.cancel({ inviteID: snapshot.inviteID, data }) : chat.reject({ inviteID: snapshot.inviteID, data });
      void Promise.resolve(notification).catch(() => {});
    } catch { /* SDK logout still proceeds and the invitation has a finite timeout. */ }
  }
  ++bindingEpoch; clearWaitingChat(); detachBindings?.(); detachBindings = null;
  boundChat = null; boundIdentity = ''; sentCallRecordKeys.clear();
  await endLocalCall();
}

export async function startDoMattxC2CCall(
  peerUserID: string,
  mediaType: DoMattxCallMediaType,
  peerName?: string,
  peerAvatar?: string,
) {
  if (!ensureDoMattxCallBridge()) throw new Error("聊天尚未就绪，请连接成功后重试");
  const session = getCurrentSession();
  if (!session?.userID || !session?.userSig || !session?.sdkAppID) {
    throw new Error("通话未登录，请重新登录");
  }
  if (!peerUserID) {
    throw new Error("缺少通话对象");
  }
  if (isBusy()) {
    throw new Error("当前正在通话中");
  }

  const epoch = ++callEpoch;
  const identity = currentIdentity();
  const chat = getChat();
  const roomId = createDoMattxRoomId(`${session.userID}_${peerUserID}_${Date.now()}`);
  const data = stringifyDoMattxCallSignal({
    mediaType,
    roomId,
    caller: session.userID,
    nickname: getSelfDisplayName(),
    avatar: getSelfAvatar(),
  });
  doMattxCallState.visible = true; doMattxCallState.phase = 'outgoing';
  let response: any;
  try { response = await chat.invite({ userID: peerUserID, data, timeout: CALL_TIMEOUT_SECONDS, onlineUserOnly: false }); }
  catch (error) { if (epoch === callEpoch) await endLocalCall(); throw error; }
  if (epoch !== callEpoch || currentIdentity() !== identity || chat !== getChat()) return;
  const inviteID = getInviteIDFromResponse(response);
  if (!inviteID) {
    await endLocalCall();
    throw new Error("发起通话失败");
  }

  setCallState({
    visible: true,
    phase: "outgoing",
    inviteID,
    peerUserID,
    peerName: peerName || peerUserID,
    peerAvatar: peerAvatar || "",
    mediaType,
    roomId,
    roomType: "string",
    isCaller: true,
    connectedAt: 0,
    remoteUserID: peerUserID,
    remoteVideoAvailable: false,
    micMuted: false,
    cameraMuted: false,
    error: "",
  });
  resetTimeoutTimer();
}

export async function acceptDoMattxCall() {
  if (doMattxCallState.phase !== "incoming" || !doMattxCallState.inviteID) {
    return;
  }
  const epoch = callEpoch;
  doMattxCallState.phase = "connecting";
  stopTimeoutTimer();
  try { await getChat().accept({
    inviteID: doMattxCallState.inviteID,
    data: stringifyDoMattxCallSignal({
      mediaType: doMattxCallState.mediaType,
      roomId: doMattxCallState.roomId,
      roomType: doMattxCallState.roomType,
      caller: getCurrentSession()?.userID || "",
      action: "accept",
    }),
  });
  if (epoch !== callEpoch) return;
  await enterTRTCRoom();
  } catch (error) { if (epoch === callEpoch) await endLocalCall("接听失败，请重试"); throw error; }
}

export async function rejectDoMattxCall() {
  if (doMattxCallState.phase !== "incoming" || !doMattxCallState.inviteID) {
    return;
  }
  const epoch = callEpoch;
  const snapshot = getCallSnapshot();
  await getChat().reject({
    inviteID: doMattxCallState.inviteID,
    data: stringifyDoMattxCallSignal({
      mediaType: doMattxCallState.mediaType,
      roomId: doMattxCallState.roomId,
      roomType: doMattxCallState.roomType,
      caller: getCurrentSession()?.userID || "",
      action: "reject",
    }),
  });
  if (epoch !== callEpoch) return;
  await sendCallSummaryWithSnapshot(snapshot, 4);
  if (epoch === callEpoch) await endLocalCall();
}

export async function hangupDoMattxCall() {
  if (!isBusy()) {
    return;
  }
  const phase = doMattxCallState.phase;
  const snapshot = getCallSnapshot();
  notifyCallEnded(phase, snapshot);
  await endLocalCall();
}

export async function toggleDoMattxMicMuted() {
  doMattxCallState.micMuted = !doMattxCallState.micMuted;
  try {
    await trtc?.muteLocalAudio(doMattxCallState.micMuted);
  } catch (error) {
    console.warn("toggle mic failed", error);
  }
}

export async function toggleDoMattxCameraMuted() {
  if (doMattxCallState.mediaType !== DoMattxCallMediaType.Video) {
    return;
  }
  const nextMuted = !doMattxCallState.cameraMuted;
  doMattxCallState.cameraMuted = nextMuted;
  doMattxCallState.error = "";
  try {
    if (nextMuted) {
      await trtc?.stopLocalPreview();
      return;
    }
    const localView = document.getElementById(LOCAL_VIDEO_DOM_ID);
    await trtc?.startLocalPreview(localView || undefined);
  } catch (error) {
    doMattxCallState.cameraMuted = true;
    doMattxCallState.error = getFriendlyCallErrorMessage(error);
    console.warn("toggle camera failed", error);
  }
}

export async function refreshDoMattxVideoViews() {
  if (!isBusy() || doMattxCallState.mediaType !== DoMattxCallMediaType.Video) {
    return;
  }
  const epoch = callEpoch;
  const cloud = trtc;
  if (!cloud) return;
  const { TRTCVideoStreamType } = await loadRTC();
  await nextTick();
  if (epoch !== callEpoch || cloud !== trtc) return;
  try {
    const localView = document.getElementById(LOCAL_VIDEO_DOM_ID);
    if (localView && !doMattxCallState.cameraMuted) {
      await cloud?.startLocalPreview(localView);
    }
    if (epoch !== callEpoch || cloud !== trtc) return;
    const remoteView = document.getElementById(REMOTE_VIDEO_DOM_ID);
    if (remoteView && doMattxCallState.remoteUserID) {
      await cloud?.startRemoteView(
        doMattxCallState.remoteUserID,
        remoteView,
        TRTCVideoStreamType.TRTCVideoStreamTypeBig,
      );
    }
  } catch (error) {
    if (epoch !== callEpoch) return;
    doMattxCallState.error = getFriendlyCallErrorMessage(error);
    console.warn("refresh trtc video views failed", error);
  }
}

export function isDoMattxCallBusy() {
  return isBusy();
}

function onNewInvitationReceived(...args: any[]) {
  const payload = normalizeInvitationPayload(args);
  const signal = parseDoMattxCallSignal(payload.data);
  if (!signal || !signal.roomId || (signal.action && signal.action !== "invite") || payload.groupID) {
    return;
  }
  if (signal.longdoType && signal.longdoType !== "longdo_rtc_call_invite_v1") {
    return;
  }

  const inviteID = String(payload.inviteID || payload.inviteId || "");
  const inviter = String(payload.inviter || signal.caller || "");
  if (!inviteID || !inviter) {
    return;
  }
  if (doMattxCallState.inviteID === inviteID) return;
  if (isBusy()) {
    getChat().reject({
      inviteID,
      data: stringifyDoMattxCallSignal({
        mediaType: signal.mediaType,
        roomId: signal.roomId,
        roomType: signal.roomType,
        caller: signal.caller,
        nickname: signal.nickname,
        avatar: signal.avatar,
        action: "busy",
      }),
    }).catch(() => {});
    return;
  }

  ++callEpoch;
  setCallState({
    visible: true,
    phase: "incoming",
    inviteID,
    peerUserID: inviter,
    peerName: signal.nickname || inviter,
    peerAvatar: signal.avatar || "",
    mediaType: signal.mediaType,
    roomId: signal.roomId,
    roomType: signal.roomType,
    isCaller: false,
    connectedAt: 0,
    remoteUserID: inviter,
    remoteVideoAvailable: false,
    micMuted: false,
    cameraMuted: false,
    error: "",
  });
  resetTimeoutTimer();
}

async function onInviteeAccepted(...args: any[]) {
  const payload = normalizeInvitationPayload(args);
  await handleInviteeAcceptedPayload(payload);
}

async function onInviteeRejected(...args: any[]) {
  const payload = normalizeInvitationPayload(args);
  if (!isCurrentInvite(payload.inviteID || payload.inviteId)) {
    return;
  }
  const rejectedSignal = parseDoMattxCallSignal(payload.data);
  const epoch = callEpoch;
  const snapshot = getCallSnapshot();
  if (rejectedSignal?.action === "busy") {
    await sendCallSummaryWithSnapshot(snapshot, 5);
  } else {
    recordCallHistory(snapshot, 4, 0);
  }
  if (epoch !== callEpoch) return;
  await endLocalCall(rejectedSignal?.action === "busy" ? "对方忙线中" : "对方已拒绝");
}

async function onInvitationCancelled(...args: any[]) {
  const payload = normalizeInvitationPayload(args);
  if (!isCurrentInvite(payload.inviteID || payload.inviteId)) {
    return;
  }
  recordCallHistory(getCallSnapshot(), 2, 0);
  await endLocalCall("对方已取消");
}

async function onInvitationTimeout(...args: any[]) {
  const payload = normalizeInvitationPayload(args);
  if (!isCurrentInvite(payload.inviteID || payload.inviteId)) {
    return;
  }
  const epoch = callEpoch;
  const snapshot = getCallSnapshot();
  if (snapshot.isCaller) {
    await sendCallSummaryWithSnapshot(snapshot, 5);
  } else {
    recordCallHistory(snapshot, 5, 0);
  }
  if (epoch === callEpoch) await endLocalCall("无人接听");
}

async function onNewMessageList(messages: IMessageModel[] = []) {
  for (const message of messages) {
    if (await handleDoMattxCallMessage(message)) {
      return;
    }
  }
}

async function onSdkMessageReceived(event: any) {
  const messages = normalizeMessageEvent(event);
  for (const message of messages) {
    if (await handleDoMattxCallMessage(message)) {
      return;
    }
  }
}

async function handleDoMattxCallMessage(message: IMessageModel): Promise<boolean> {
  ingestCallSummaryMessage(message);
  if (await handleDoMattxSignalingActionMessage(message)) {
    return true;
  }
  return handleDoMattxCallSignalMessage(message);
}

async function handleDoMattxSignalingActionMessage(message: IMessageModel): Promise<boolean> {
  const payload = getSignalingPayloadFromMessage(message);
  if (!isTencentSignalingAcceptPayload(payload)) {
    return false;
  }
  return handleInviteeAcceptedPayload(payload);
}

async function handleInviteeAcceptedPayload(payload: SignalingEventPayload): Promise<boolean> {
  if (!isCurrentInvite(payload.inviteID || payload.inviteId)) {
    return false;
  }
  if (doMattxCallState.phase !== "outgoing") {
    return true;
  }
  stopTimeoutTimer();
  doMattxCallState.phase = "connecting";
  await enterTRTCRoom();
  return true;
}

async function handleDoMattxCallSignalMessage(message: IMessageModel): Promise<boolean> {
  const signal = getDoMattxCallSignalFromMessage(message);
  if (!signal) {
    return false;
  }
  if (signal.action === "hangup") {
    if (signal.roomId === doMattxCallState.roomId && signal.roomType === doMattxCallState.roomType && message.from === doMattxCallState.peerUserID && isBusy()) {
      const snapshot = getCallSnapshot();
      const callEnd = getCallDurationSecondsFromSnapshot(snapshot);
      recordCallHistory(snapshot, callEnd > 0 ? 1 : 5, callEnd);
      await endLocalCall();
      return true;
    }
    return false;
  }
  if (!signal.roomId || (signal.action && signal.action !== "invite")) {
    return false;
  }
  if (signal.longdoType && signal.longdoType !== "longdo_rtc_call_invite_v1") {
    return false;
  }

  const payload = getSignalingPayloadFromMessage(message);
  const inviteID = String(payload.inviteID || payload.inviteId || "");
  const inviter = String(payload.inviter || signal.caller || message.from || "");
  const session = getCurrentSession();
  if (!inviteID || !inviter || inviter === session?.userID) {
    return false;
  }
  if (doMattxCallState.inviteID && doMattxCallState.inviteID === inviteID) {
    return true;
  }
  if (isBusy()) {
    getChat().reject({
      inviteID,
      data: stringifyDoMattxCallSignal({
        mediaType: signal.mediaType,
        roomId: signal.roomId,
        roomType: signal.roomType,
        caller: signal.caller,
        nickname: signal.nickname,
        avatar: signal.avatar,
        action: "busy",
      }),
    }).catch(() => {});
    return true;
  }

  ++callEpoch;
  setCallState({
    visible: true,
    phase: "incoming",
    inviteID,
    peerUserID: inviter,
    peerName: signal.nickname || inviter,
    peerAvatar: signal.avatar || "",
    mediaType: signal.mediaType,
    roomId: signal.roomId,
    roomType: signal.roomType,
    isCaller: false,
    connectedAt: 0,
    remoteUserID: inviter,
    remoteVideoAvailable: false,
    micMuted: false,
    cameraMuted: false,
    error: "",
  });
  resetTimeoutTimer();
  return true;
}

async function enterTRTCRoom() {
  const session = getCurrentSession();
  if (!session?.sdkAppID || !session?.userID || !session?.userSig) {
    await endLocalCall("通话凭证缺失，请重新登录后再拨打");
    return;
  }
  const epoch = callEpoch;
  try {
    await rtcCleanup;
    if (epoch !== callEpoch) return;
    await destroyTRTC();
    if (epoch !== callEpoch) return;
    const { default: TRTCCloud, TRTCParams, TRTCRoleType, TRTCAppScene, TRTCAudioQuality } = await loadRTC();
    if (epoch !== callEpoch) return;
    const cloud = TRTCCloud.getTRTCShareInstance();
    trtc = cloud;
    bindTRTCEvents();
    const params = new TRTCParams(
      Number(session.sdkAppID),
      session.userID,
      session.userSig,
      doMattxCallState.roomType === "number" ? Number(doMattxCallState.roomId) : 0,
      doMattxCallState.roomType === "string" ? doMattxCallState.roomId : "",
    );
    params.role = TRTCRoleType.TRTCRoleAnchor;
    const scene = doMattxCallState.mediaType === DoMattxCallMediaType.Video
      ? TRTCAppScene.TRTCAppSceneVideoCall
      : TRTCAppScene.TRTCAppSceneAudioCall;
    await cloud.enterRoom(params, scene);
    if (epoch !== callEpoch || trtc !== cloud) return;
    await cloud.startLocalAudio(TRTCAudioQuality.TRTCAudioQualitySpeech);
    if (epoch !== callEpoch || trtc !== cloud) return;
    if (doMattxCallState.mediaType === DoMattxCallMediaType.Video) {
      await refreshDoMattxVideoViews();
    }
  } catch (error: any) {
    if (epoch !== callEpoch) return;
    const message = getFriendlyCallErrorMessage(error);
    doMattxCallState.error = message;
    await sendCallSummary(5);
    if (epoch === callEpoch) await endLocalCall(message);
  }
}

function bindTRTCEvents() {
  const cloud = trtc; if (!cloud) return;
  const epoch = callEpoch;
  const handlers: Array<[string, (...args: any[]) => void]> = [
    ['onEnterRoom', onTRTCEnterRoom], ['onRemoteUserEnterRoom', onTRTCRemoteUserEnterRoom],
    ['onRemoteUserLeaveRoom', onTRTCRemoteUserLeaveRoom], ['onUserVideoAvailable', onTRTCUserVideoAvailable], ['onError', onTRTCError],
  ];
  const bound = handlers.map(([event, handler]) => [event, (...args: any[]) => { if (trtc === cloud && epoch === callEpoch) handler(...args); }] as [string, (...args: any[]) => void]);
  bound.forEach(([event, handler]) => cloud.on(event, handler)); rtcBindings.set(cloud, bound);
}

function unbindTRTCEvents(cloud: TRTCCloud) {
  (rtcBindings.get(cloud) || []).forEach(([event, handler]) => cloud.off(event, handler)); rtcBindings.delete(cloud);
}

function onTRTCEnterRoom(result: number) {
  if (result < 0) {
    const epoch = callEpoch;
    const error = `通话进房失败(${result})`;
    doMattxCallState.error = error;
    sendCallSummary(5).catch(() => {}).finally(() => { if (epoch === callEpoch) return endLocalCall(error); });
    return;
  }
  if (!isBusy()) {
    return;
  }
  doMattxCallState.phase = "connected";
  doMattxCallState.connectedAt = doMattxCallState.connectedAt || Date.now();
  refreshDoMattxVideoViews();
}

function onTRTCRemoteUserEnterRoom(userID: string) {
  doMattxCallState.remoteUserID = userID || doMattxCallState.remoteUserID;
}

function onTRTCRemoteUserLeaveRoom(userID: string) {
  if (!isBusy() || userID !== doMattxCallState.peerUserID) {
    return;
  }
  endLocalCall();
}

function onTRTCUserVideoAvailable(userID: string, available: boolean) {
  doMattxCallState.remoteUserID = userID || doMattxCallState.remoteUserID;
  doMattxCallState.remoteVideoAvailable = Boolean(available);
  if (available) {
    refreshDoMattxVideoViews();
  }
}

function onTRTCError(code: number, message?: string) {
  console.warn("TRTC error", code);
  doMattxCallState.error = getFriendlyCallErrorMessage(message, code);
}

async function sendCallSummary(actionType: number, callEnd = 0) {
  await sendCallSummaryWithSnapshot(getCallSnapshot(), actionType, callEnd);
}

async function sendCallSummaryWithSnapshot(
  snapshot: DoMattxCallSnapshot,
  actionType: number,
  callEnd = 0,
) {
  if (!snapshot.peerUserID) {
    return;
  }
  const recordKey = snapshot.inviteID || `${snapshot.peerUserID}-${snapshot.roomId}`;
  if (recordKey && sentCallRecordKeys.has(recordKey)) {
    recordCallHistory(snapshot, actionType, callEnd);
    return;
  }
  if (recordKey) {
    sentCallRecordKeys.add(recordKey);
  }
  const currentUserID = getCurrentSession()?.userID || "";
  recordCallHistory(snapshot, actionType, callEnd);
  const payload = stringifyDoMattxCallSummary({
    mediaType: snapshot.mediaType,
    actionType,
    callEnd,
    caller: snapshot.isCaller ? currentUserID : snapshot.peerUserID,
  });
  const summary = getDoMattxCallSummaryText({
    mediaType: snapshot.mediaType,
    actionType,
    callEnd,
  });
  const message = getChat().createCustomMessage({
    to: snapshot.peerUserID,
    conversationType: TUIChatEngine.TYPES.CONV_C2C,
    payload: {
      data: payload,
      description: summary,
      extension: summary,
    },
    needReadReceipt: false,
  });
  const response = await getChat().sendMessage(message, { needReadReceipt: false });
  syncSentCallSummaryToMessageList(response);
}

function notifyCallEnded(phase: DoMattxCallPhase, snapshot: DoMattxCallSnapshot) {
  const identity = currentIdentity();
  const chat = getChat();
  void (async () => {
    try {
      if (identity !== currentIdentity() || chat !== getChat()) return;
      if (phase === "outgoing" && snapshot.inviteID) {
        await getChat().cancel({
          inviteID: snapshot.inviteID,
          data: stringifyDoMattxCallSignal({
            mediaType: snapshot.mediaType,
            roomId: snapshot.roomId,
            roomType: snapshot.roomType,
            caller: getCurrentSession()?.userID || "",
            action: "cancel",
          }),
        });
        if (identity === currentIdentity() && chat === getChat()) await sendCallSummaryWithSnapshot(snapshot, 2);
      } else if (phase === "incoming" && snapshot.inviteID) {
        await getChat().reject({
          inviteID: snapshot.inviteID,
          data: stringifyDoMattxCallSignal({
            mediaType: snapshot.mediaType,
            roomId: snapshot.roomId,
            roomType: snapshot.roomType,
            caller: getCurrentSession()?.userID || "",
            action: "reject",
          }),
        });
        if (identity === currentIdentity() && chat === getChat()) await sendCallSummaryWithSnapshot(snapshot, 4);
      } else {
        const callEnd = getCallDurationSecondsFromSnapshot(snapshot);
        recordCallHistory(snapshot, callEnd > 0 ? 1 : 5, callEnd);
        const message = getChat().createCustomMessage({ to: snapshot.peerUserID, conversationType: TUIChatEngine.TYPES.CONV_C2C,
          payload: { data: stringifyDoMattxCallSignal({ mediaType: snapshot.mediaType, roomId: snapshot.roomId, roomType: snapshot.roomType,
            caller: getCurrentSession()?.userID || '', action: 'hangup', durationSec: callEnd, inviteId: snapshot.inviteID }),
            description: getDoMattxCallSummaryText({ mediaType: snapshot.mediaType, actionType: 1, callEnd }), extension: '' }, needReadReceipt: false });
        syncSentCallSummaryToMessageList(await getChat().sendMessage(message, { needReadReceipt: false }));
      }
    } catch (error) {
      console.warn("notify call ended failed", error);
    }
  })();
}

async function endLocalCall(reason = "") {
  ++callEpoch; stopTimeoutTimer();
  setCallState({ visible: false, phase: 'idle', inviteID: '', peerUserID: '', peerName: '', peerAvatar: '',
    mediaType: DoMattxCallMediaType.Audio, roomId: '', roomType: 'string', isCaller: false, connectedAt: 0,
    remoteUserID: '', remoteVideoAvailable: false, micMuted: false, cameraMuted: false, error: reason });
  rtcCleanup = Promise.all([rtcCleanup.catch(() => {}), destroyTRTC().catch(() => {})]).then(() => {});
  await rtcCleanup;
}

async function destroyTRTC() {
  const cloud = trtc; trtc = null;
  if (!cloud) return;
  unbindTRTCEvents(cloud);
  try { await cloud.stopLocalAudio(); await cloud.stopLocalPreview(); await cloud.exitRoom(); }
  catch { console.warn('通话媒体清理未确认'); }
}

function normalizeInvitationPayload(args: any[]): SignalingEventPayload {
  if (args.length === 1 && args[0] && typeof args[0] === "object") {
    const event = args[0] as Record<string, any>;
    if (event.data && typeof event.data === "object" && !Array.isArray(event.data)) {
      return event.data as SignalingEventPayload;
    }
    return event as SignalingEventPayload;
  }
  return {
    inviteID: args[0],
    inviter: args[1],
    groupID: args[2],
    inviteeList: Array.isArray(args[3]) ? args[3] : undefined,
    data: typeof args[4] === "string" ? args[4] : typeof args[2] === "string" ? args[2] : "",
  };
}

function normalizeMessageEvent(event: any): IMessageModel[] {
  if (Array.isArray(event)) {
    return event;
  }
  if (Array.isArray(event?.data)) {
    return event.data;
  }
  if (Array.isArray(event?.messageList)) {
    return event.messageList;
  }
  return [];
}

function getSignalingPayloadFromMessage(message: IMessageModel): SignalingEventPayload {
  const signalingInfo = message?.getSignalingInfo?.();
  if (signalingInfo) {
    const payload = normalizeSignalingPayload(signalingInfo);
    if (payload) {
      return payload;
    }
  }
  const payloadData = parseJSONMap(message?.payload?.data);
  const payload = normalizeSignalingPayload(payloadData);
  if (payload) {
    return payload;
  }
  return {
    inviter: message?.from,
    data: message?.payload?.data,
  };
}

function getInviteIDFromResponse(response: any): string {
  const candidates = [
    response?.inviteID,
    response?.inviteId,
    response?.data?.inviteID,
    response?.data?.inviteId,
    response?.data,
  ];
  for (const candidate of candidates) {
    if (typeof candidate === "string" || typeof candidate === "number") {
      const inviteID = String(candidate).trim();
      if (inviteID) {
        return inviteID;
      }
    }
  }
  return "";
}

function syncSentCallSummaryToMessageList(response: any) {
  const message = getSentMessageFromResponse(response);
  if (!message?.ID) {
    return;
  }
  const currentMessageList = TUIStore.getData(StoreName.CHAT, "messageList") || [];
  if (
    Array.isArray(currentMessageList) &&
    currentMessageList.some((item: IMessageModel) => item?.ID === message.ID)
  ) {
    return;
  }
  TUICore.callService({
    serviceName: TUIConstants.TUIChat.SERVICE.NAME,
    method: TUIConstants.TUIChat.SERVICE.METHOD.UPDATE_MESSAGE_LIST,
    params: { message },
  });
}

function getSentMessageFromResponse(response: any): IMessageModel | undefined {
  const candidates = [
    response?.data?.message,
    response?.data?.messageModel,
    response?.data,
    response?.message,
  ];
  return candidates.find(isMessageModelLike) as IMessageModel | undefined;
}

function isMessageModelLike(value: unknown): value is IMessageModel {
  return Boolean(
    value &&
      typeof value === "object" &&
      typeof (value as IMessageModel).ID === "string" &&
      typeof (value as IMessageModel).conversationID === "string"
  );
}

function normalizeSignalingPayload(raw: unknown): (SignalingEventPayload & Record<string, any>) | undefined {
  const data = parseJSONMap(raw);
  if (!data) {
    return undefined;
  }
  if (data.businessID === TENCENT_SIGNALING_BUSINESS_ID || data.inviteID || data.inviteId) {
    return data as SignalingEventPayload & Record<string, any>;
  }
  if (data.data && typeof data.data === "object" && !Array.isArray(data.data)) {
    return normalizeSignalingPayload(data.data);
  }
  return undefined;
}

function isTencentSignalingAcceptPayload(payload: SignalingEventPayload & Record<string, any>): boolean {
  const actionType = Number(payload.actionType ?? payload.action_type);
  return actionType === TENCENT_SIGNALING_ACTION_ACCEPT;
}

function parseJSONMap(raw: unknown): Record<string, any> | undefined {
  if (!raw) {
    return undefined;
  }
  if (typeof raw !== "string") {
    return typeof raw === "object" ? raw as Record<string, any> : undefined;
  }
  try {
    const value = JSON.parse(raw);
    return value && typeof value === "object" ? value as Record<string, any> : undefined;
  } catch {
    return undefined;
  }
}

function getSignalingEventName(key: keyof typeof SIGNALING_EVENT): string {
  const signaling = (TUIChatEngine as any)?.TSignaling || {};
  return signaling[key] || SIGNALING_EVENT[key];
}

function getChat(): any {
  return boundChat || TUIChatEngine.chat as any;
}

function getCurrentSession() {
  return useAuthSession().currentSession.value;
}

function getSelfDisplayName(): string {
  const userProfile = TUIStore.getData(StoreName.USER, "userProfile");
  const session = getCurrentSession();
  return userProfile?.nick || userProfile?.userID || session?.userID || "";
}

function getSelfAvatar(): string {
  return TUIStore.getData(StoreName.USER, "userProfile")?.avatar || "";
}

function isCurrentInvite(inviteID: unknown): boolean {
  return Boolean(inviteID && String(inviteID) === doMattxCallState.inviteID);
}

function isBusy(): boolean {
  return doMattxCallState.visible && doMattxCallState.phase !== "idle";
}

function setCallState(patch: DoMattxCallState) {
  Object.assign(doMattxCallState, patch);
}

function getFriendlyCallErrorMessage(error: unknown, code?: number): string {
  const raw = typeof error === "string"
    ? error
    : (error as any)?.message || (error as any)?.toString?.() || "";
  const lower = raw.toLowerCase();
  if (
    lower.includes("microphone") ||
    lower.includes("startlocalaudio") ||
    lower.includes("audio capture")
  ) {
    if (
      lower.includes("permission") ||
      lower.includes("notallowed") ||
      lower.includes("denied")
    ) {
      return "未获得麦克风权限，请检查浏览器权限";
    }
    return "未检测到麦克风，请检查设备或浏览器权限";
  }
  if (
    lower.includes("camera") ||
    lower.includes("startlocalvideo") ||
    lower.includes("startlocalpreview") ||
    lower.includes("video capture")
  ) {
    if (
      lower.includes("permission") ||
      lower.includes("notallowed") ||
      lower.includes("denied")
    ) {
      return "未获得摄像头权限，请检查浏览器权限";
    }
    return "未检测到摄像头，请检查设备或浏览器权限";
  }
  if (
    lower.includes("permission") ||
    lower.includes("notallowed") ||
    lower.includes("denied")
  ) {
    return doMattxCallState.mediaType === DoMattxCallMediaType.Video
      ? "未获得麦克风或摄像头权限，请检查浏览器权限"
      : "未获得麦克风权限，请检查浏览器权限";
  }
  return code ? `通话错误(${code})，请稍后重试` : "通话启动失败，请重试";
}

function getCallSnapshot(): DoMattxCallSnapshot {
  return {
    inviteID: doMattxCallState.inviteID,
    peerUserID: doMattxCallState.peerUserID,
    peerName: doMattxCallState.peerName,
    peerAvatar: doMattxCallState.peerAvatar,
    mediaType: doMattxCallState.mediaType,
    roomId: doMattxCallState.roomId,
    roomType: doMattxCallState.roomType,
    isCaller: doMattxCallState.isCaller,
    connectedAt: doMattxCallState.connectedAt,
  };
}

function recordCallHistory(
  snapshot: DoMattxCallSnapshot,
  actionType: number,
  duration = 0,
) {
  const ownerUserID = getCurrentSession()?.userID || "";
  if (!ownerUserID || !snapshot.peerUserID) {
    return;
  }
  rememberLocalCall({
    ownerUserID,
    id: snapshot.inviteID || `${snapshot.roomId}-${actionType}-${Date.now()}`,
    peerUserID: snapshot.peerUserID,
    peerName: snapshot.peerName,
    peerAvatar: snapshot.peerAvatar,
    mediaType: snapshot.mediaType,
    direction: snapshot.isCaller ? "out" : "in",
    actionType,
    duration,
  });
}

function ingestCallSummaryMessage(message: IMessageModel) {
  const ownerUserID = getCurrentSession()?.userID || "";
  if (!ownerUserID) {
    return;
  }
  rememberCallFromMessage(message, ownerUserID);
}

function getCallDurationSecondsFromSnapshot(snapshot: DoMattxCallSnapshot): number {
  if (!snapshot.connectedAt) {
    return 0;
  }
  return Math.max(0, Math.floor((Date.now() - snapshot.connectedAt) / 1000));
}

function resetTimeoutTimer() {
  stopTimeoutTimer();
  callTimer = window.setTimeout(() => {
    if (!isBusy() || doMattxCallState.phase === "connected") {
      return;
    }
    const snapshot = getCallSnapshot();
    const finish = snapshot.isCaller
      ? sendCallSummaryWithSnapshot(snapshot, 5)
      : Promise.resolve(recordCallHistory(snapshot, 5, 0));
    const epoch = callEpoch;
    finish.finally(() => { if (epoch === callEpoch) return endLocalCall("无人接听"); });
  }, CALL_TIMEOUT_SECONDS * 1000);
}

function stopTimeoutTimer() {
  if (callTimer) {
    window.clearTimeout(callTimer);
    callTimer = undefined;
  }
}

export const doMattxCallView = {
  localVideoDomID: LOCAL_VIDEO_DOM_ID,
  remoteVideoDomID: REMOTE_VIDEO_DOM_ID,
  getMediaLabel: getDoMattxCallMediaLabel,
};
