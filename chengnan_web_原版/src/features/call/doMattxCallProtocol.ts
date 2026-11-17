import type { IMessageModel } from "@tencentcloud/chat-uikit-engine-lite";
import { JSONToObject } from "@/TUIKit/utils/type-check";

export const LONGDO_CALL_INVITE = "longdo_rtc_call_invite_v1";
export const LONGDO_CALL_ACCEPT = "longdo_rtc_call_accept_v1";
export const LONGDO_CALL_REJECT = "longdo_rtc_call_reject_v1";
export const LONGDO_CALL_CANCEL = "longdo_rtc_call_cancel_v1";
export const LONGDO_CALL_BUSY = "longdo_rtc_call_busy_v1";
export const LONGDO_CALL_RECORD = "longdo_rtc_call_record_v1";

export const RUI_CALL_BUSINESS_ID = "rtc_call_v1";
export const RUI_CALL_VERSION = 2;
const LEGACY_CALL_BUSINESS_ID = "mllx_trtc_call";
const TENCENT_SIGNALING_BUSINESS_ID = 1;
const TENCENT_SIGNALING_ACTION_SET = new Set([1, 2, 3, 4, 5]);
const CALL_RECORD_TYPES = new Set(["av_call", "rtc_call"]);

export enum DoMattxCallMediaType {
  Audio = 1,
  Video = 2,
}

export interface DoMattxCallSignal {
  version: number;
  mediaType: DoMattxCallMediaType;
  roomId: string;
  roomType: "string" | "number";
  caller: string;
  nickname: string;
  avatar?: string;
  action?: "hangup" | "busy" | "accept" | "reject" | "cancel" | string;
  longdoType?: string;
}

export interface DoMattxCallSummaryOptions {
  mediaType: DoMattxCallMediaType;
  actionType: number;
  callEnd?: number;
  caller?: string;
}

function parseJSONMap(raw: unknown): Record<string, any> | undefined {
  if (!raw) {
    return undefined;
  }
  const value = typeof raw === "string" ? JSONToObject(raw) : raw;
  return value && typeof value === "object" ? (value as Record<string, any>) : undefined;
}

function unwrapPayload(raw: unknown): Record<string, any> | undefined {
  const data = parseJSONMap(raw);
  if (!data) {
    return undefined;
  }
  if (data.businessID === TENCENT_SIGNALING_BUSINESS_ID && data.data) {
    return unwrapPayload(data.data);
  }
  return data;
}

function mediaTypeFromPayload(data: Record<string, any>): DoMattxCallMediaType {
  if (
    data.longdo_media_type === "longdo_video" ||
    Number(data.mediaType) === DoMattxCallMediaType.Video ||
    Number(data.call_type) === 2 ||
    data.cmd === "videoCall"
  ) {
    return DoMattxCallMediaType.Video;
  }
  return DoMattxCallMediaType.Audio;
}

function actionFromLongdoType(type: string): DoMattxCallSignal["action"] {
  switch (type) {
    case LONGDO_CALL_ACCEPT:
      return "accept";
    case LONGDO_CALL_REJECT:
      return "reject";
    case LONGDO_CALL_CANCEL:
      return "cancel";
    case LONGDO_CALL_BUSY:
      return "busy";
    default:
      return undefined;
  }
}

export function parseDoMattxCallSignal(raw: unknown): DoMattxCallSignal | undefined {
  const data = unwrapPayload(raw);
  if (!data) {
    return undefined;
  }

  if (data.businessID === RUI_CALL_BUSINESS_ID) {
    const version = Number(data.version || 1);
    const roomType = data.roomType || 'string';
    const roomId = String(data.roomId || '');
    const action = String(data.action || 'invite');
    if (![1, 2].includes(version) || !['string', 'number'].includes(roomType) || roomId.length > 128 ||
      !['invite', 'accept', 'reject', 'busy', 'cancel', 'no_response', 'hangup'].includes(action)) return undefined;
    if (action === 'invite' && (!roomId || (Number(data.expiresAtMs) > 0 && Number(data.expiresAtMs) + 5000 < Date.now()))) return undefined;
    if (roomType === 'number' && roomId && (!Number.isInteger(Number(roomId)) || Number(roomId) < 1 || Number(roomId) > 0x7fffffff)) return undefined;
    return { version, roomType, roomId, action, mediaType: mediaTypeFromPayload(data),
      caller: String(data.caller || ''), nickname: String(data.peerName || data.nickname || ''),
      avatar: String(data.peerAvatar || data.avatar || '') };
  }

  const longdoType = String(data.longdo_type || "");
  if (longdoType === LONGDO_CALL_RECORD) {
    return undefined;
  }
  if (longdoType.startsWith("longdo_rtc_call_")) {
    const roomId = Number(data.longdo_room_id || 0);
    return {
      version: Number(data.longdo_version || 1),
      mediaType: mediaTypeFromPayload(data),
      roomId: Number.isFinite(roomId) && roomId > 0 ? String(roomId) : "",
      roomType: "number",
      caller: String(data.longdo_caller_id || data.caller || ""),
      nickname: String(data.longdo_caller_name || data.nickname || ""),
      avatar: data.longdo_caller_avatar_url
        ? String(data.longdo_caller_avatar_url)
        : undefined,
      action: actionFromLongdoType(longdoType),
      longdoType,
    };
  }

  if (data.businessID === LEGACY_CALL_BUSINESS_ID) {
    const roomId = Number(data.roomId || 0);
    if (!Number.isFinite(roomId) || roomId <= 0) {
      return undefined;
    }
    return {
      version: Number(data.version || 1),
      mediaType: mediaTypeFromPayload(data),
      roomId: String(roomId),
      roomType: "number",
      caller: String(data.caller || ""),
      nickname: String(data.nickname || ""),
      avatar: data.avatar ? String(data.avatar) : undefined,
      action: data.action ? String(data.action) : undefined,
    };
  }

  return undefined;
}

export function getDoMattxCallSignalFromMessage(
  message: IMessageModel,
): DoMattxCallSignal | undefined {
  const signalingInfo = message?.getSignalingInfo?.();
  return (
    parseDoMattxCallSignal(signalingInfo?.data) ||
    parseDoMattxCallSignal(message?.payload?.data)
  );
}

export function isRawLongdoCallDisplayText(text?: string): boolean {
  const value = String(text || "").trim();
  return (
    value === LONGDO_CALL_RECORD || /^longdo_rtc_call_[a-z0-9_]+$/i.test(value)
  );
}

function looksLikeCallRecordText(text: string): boolean {
  const value = text.trim();
  return value.startsWith("[语音通话]") || value.startsWith("[视频通话]");
}

function isTencentSignalingEnvelope(raw: unknown): boolean {
  const wrapped = parseJSONMap(raw);
  return Boolean(
    wrapped?.businessID === TENCENT_SIGNALING_BUSINESS_ID &&
      TENCENT_SIGNALING_ACTION_SET.has(Number(wrapped.actionType || 0)),
  );
}

export function isLongdoCallRecordPayload(raw: unknown): boolean {
  if (isTencentSignalingEnvelope(raw)) {
    return unwrapPayload(raw)?.longdo_type === LONGDO_CALL_RECORD;
  }
  const data = unwrapPayload(raw);
  if (!data) {
    return false;
  }
  if (data.businessID === RUI_CALL_BUSINESS_ID && data.action === 'hangup') return true;
  if (data.longdo_type === LONGDO_CALL_RECORD) {
    return true;
  }
  if (!CALL_RECORD_TYPES.has(String(data.businessID || ""))) {
    return false;
  }
  const summary = String(data.summary || "").trim();
  const callEnd = Number(data.call_end || 0);
  const actionType = Number(data.actionType || 0);
  return looksLikeCallRecordText(summary) || callEnd > 0 || actionType === 2 || actionType === 4 || actionType === 5;
}

export function getLongdoCallRecordText(raw: unknown): string | undefined {
  const data = unwrapPayload(raw);
  if (!data) {
    return undefined;
  }
  if (data.businessID === RUI_CALL_BUSINESS_ID && data.action === 'hangup') {
    return getDoMattxCallSummaryText({ mediaType: mediaTypeFromPayload(data), actionType: 1, callEnd: Math.max(0, Number(data.durationSec) || 0) });
  }
  if (data.longdo_type === LONGDO_CALL_RECORD) {
    const payload = data.longdo_payload;
    const text =
      (payload && typeof payload === "object"
        ? String((payload as Record<string, any>).longdo_text || "")
        : "") || String(data.longdo_text || "");
    return text.trim() || undefined;
  }
  const summary = String(data.summary || "").trim();
  if (looksLikeCallRecordText(summary) && CALL_RECORD_TYPES.has(String(data.businessID || ""))) {
    return summary;
  }
  return undefined;
}

export function isLongdoCallControlPayload(raw: unknown): boolean {
  const data = unwrapPayload(raw);
  if (!data) {
    const wrapped = parseJSONMap(raw);
    if (
      wrapped?.businessID === TENCENT_SIGNALING_BUSINESS_ID &&
      TENCENT_SIGNALING_ACTION_SET.has(Number(wrapped.actionType || 0))
    ) {
      return !isLongdoCallRecordPayload(wrapped.data);
    }
    return false;
  }
  if (data.businessID === RUI_CALL_BUSINESS_ID) return data.action !== "hangup";
  const type = String(data.longdo_type || "");
  if (type === LONGDO_CALL_RECORD) {
    return false;
  }
  if (type.startsWith("longdo_rtc_call_")) {
    return true;
  }
  if (data.businessID === LEGACY_CALL_BUSINESS_ID) {
    return true;
  }
  const wrapped = parseJSONMap(raw);
  if (
    wrapped?.businessID === TENCENT_SIGNALING_BUSINESS_ID &&
    TENCENT_SIGNALING_ACTION_SET.has(Number(wrapped.actionType || 0))
  ) {
    return !isLongdoCallRecordPayload(wrapped.data);
  }
  return false;
}

export function isDoMattxCallSignalMessage(message: IMessageModel): boolean {
  if (isLongdoCallRecordPayload(message?.payload?.data)) {
    return false;
  }
  return Boolean(
    getDoMattxCallSignalFromMessage(message) ||
      isLongdoCallControlPayload(message?.payload?.data) ||
      isLongdoCallControlPayload(message?.getSignalingInfo?.()),
  );
}

export function createDoMattxRoomId(_seed?: string): string {
  const bytes = new Uint8Array(16);
  if (globalThis.crypto?.getRandomValues) globalThis.crypto.getRandomValues(bytes);
  else for (let i = 0; i < bytes.length; i++) bytes[i] = Math.floor(Math.random() * 256);
  return `rui_${Array.from(bytes, b => b.toString(16).padStart(2, '0')).join('')}`;
}

export function stringifyDoMattxCallSignal(options: {
  mediaType: DoMattxCallMediaType; roomId: string; roomType?: 'string' | 'number';
  caller: string; nickname?: string; avatar?: string; action?: string; durationSec?: number; inviteId?: string;
}): string {
  return JSON.stringify({ businessID: RUI_CALL_BUSINESS_ID, version: RUI_CALL_VERSION,
    action: options.action || 'invite', mediaType: options.mediaType,
    ...(!options.action || options.action === 'invite' ? { expiresAtMs: Date.now() + 45000 } : {}),
    roomType: options.roomType || 'string', roomId: options.roomId,
    caller: options.caller, peerName: options.nickname || options.caller, peerAvatar: options.avatar || '',
    ...(options.durationSec !== undefined ? { durationSec: options.durationSec } : {}),
    ...(options.inviteId ? { inviteId: options.inviteId } : {}),
  });
}

export function stringifyLongdoCallAccept(roomId: string | number): string {
  return JSON.stringify({
    longdo_type: LONGDO_CALL_ACCEPT,
    longdo_version: 1,
    longdo_room_id: roomId,
  });
}

export function stringifyLongdoCallAction(type: string): string {
  return JSON.stringify({
    longdo_type: type,
    longdo_version: 1,
  });
}

export function stringifyLongdoCallRecord(text: string): string {
  return JSON.stringify({
    longdo_type: LONGDO_CALL_RECORD,
    longdo_version: 1,
    longdo_payload: { longdo_text: text },
  });
}

export function getDoMattxCallMediaLabel(mediaType: DoMattxCallMediaType): string {
  return mediaType === DoMattxCallMediaType.Video ? "视频通话" : "语音通话";
}

export function getDoMattxCallSummaryText(options: DoMattxCallSummaryOptions): string {
  const mediaLabel = getDoMattxCallMediaLabel(options.mediaType);
  const callEnd = Number(options.callEnd || 0);
  if (callEnd > 0) {
    const minutes = `${Math.floor(callEnd / 60)}`.padStart(2, "0");
    const seconds = `${callEnd % 60}`.padStart(2, "0");
    return `[${mediaLabel}] 通话时长 ${minutes}:${seconds}`;
  }

  switch (Number(options.actionType || 0)) {
    case 2:
      return `[${mediaLabel}] 已取消`;
    case 4:
      return `[${mediaLabel}] 已拒绝`;
    case 5:
      return `[${mediaLabel}] 未接听`;
    default:
      return `[${mediaLabel}] 通话结束`;
  }
}

export function stringifyDoMattxCallSummary(options: DoMattxCallSummaryOptions): string {
  return stringifyLongdoCallRecord(getDoMattxCallSummaryText(options));
}

export function parseCallRecordDisplay(raw: unknown): {
  summary: string;
  mediaType: DoMattxCallMediaType;
  actionType: number;
  callEnd: number;
} | undefined {
  const text = getLongdoCallRecordText(raw);
  const data = unwrapPayload(raw);
  if (text) {
    const isVideo = text.includes("视频通话");
    const durationMatch = text.match(/通话时长\s+(\d{2}):(\d{2})/);
    const callEnd = durationMatch
      ? Number(durationMatch[1]) * 60 + Number(durationMatch[2])
      : 0;
    let actionType = 1;
    if (text.includes("已取消")) actionType = 2;
    else if (text.includes("已拒绝")) actionType = 4;
    else if (text.includes("未接听") || text.includes("忙线")) actionType = 5;
    else if (callEnd > 0) actionType = 1;
    return {
      summary: text,
      mediaType: isVideo ? DoMattxCallMediaType.Video : DoMattxCallMediaType.Audio,
      actionType,
      callEnd,
    };
  }
  if (data && CALL_RECORD_TYPES.has(String(data.businessID || ""))) {
    const mediaType = mediaTypeFromPayload(data);
    const callEnd = Number(data.call_end || 0);
    const actionType = Number(data.actionType || 0);
    if (callEnd <= 0 && actionType !== 2 && actionType !== 4 && actionType !== 5) {
      return undefined;
    }
    return {
      summary:
        String(data.summary || "").trim() ||
        getDoMattxCallSummaryText({ mediaType, actionType, callEnd }),
      mediaType,
      actionType,
      callEnd,
    };
  }
  return undefined;
}

export function getConversationCallPreviewText(conversation: {
  lastMessage?: any;
  getLastMessage?: (key: string) => string | null;
}): string {
  const last = conversation?.lastMessage;
  const data = last?.payload?.data;
  const record = getLongdoCallRecordText(data) || parseCallRecordDisplay(data)?.summary;
  if (record && !isRawLongdoCallDisplayText(record)) {
    return record;
  }
  const description = String(last?.payload?.description || "").trim();
  if (looksLikeCallRecordText(description)) {
    return description;
  }
  if (isLongdoCallControlPayload(data)) {
    const signal = parseDoMattxCallSignal(data);
    return signal ? `[${getDoMattxCallMediaLabel(signal.mediaType)}]` : "";
  }
  const fallback = conversation.getLastMessage?.("text") ?? "";
  if (
    isRawLongdoCallDisplayText(fallback) ||
    fallback === "[自定义消息]" ||
    fallback === "发起通话" ||
    fallback === "当前版本不支持该消息类型"
  ) {
    return "";
  }
  return fallback || "";
}
