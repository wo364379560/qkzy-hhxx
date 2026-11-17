import { reactive } from "vue";
import type { IMessageModel } from "@tencentcloud/chat-uikit-engine-lite";
import {
  getCallMessageCallerUserID,
  getCallMessageData,
  isCallMessage,
} from "@/TUIKit/plugins/plugin-components";
import { DoMattxCallMediaType } from "./doMattxCallProtocol";

export type CallHistoryStatus = "answered" | "canceled" | "rejected" | "missed";
export type CallHistoryDirection = "in" | "out";

export interface CallHistoryItem {
  id: string;
  ownerUserID: string;
  peerUserID: string;
  peerName: string;
  peerAvatar: string;
  mediaType: DoMattxCallMediaType;
  direction: CallHistoryDirection;
  status: CallHistoryStatus;
  duration: number;
  at: number;
}

export const callHistoryState = reactive({
  ownerUserID: "",
  items: [] as CallHistoryItem[],
});

const STORAGE_PREFIX = "ld.callHistory.v1.";
const MAX_ITEMS = 200;
const DEDUP_WINDOW_MS = 8000;

export function callHistoryStorageKey(ownerUserID: string) {
  return `${STORAGE_PREFIX}${ownerUserID}`;
}

export function bindCallHistoryOwner(ownerUserID: string) {
  const nextOwner = String(ownerUserID || "").trim();
  if (!nextOwner) {
    callHistoryState.ownerUserID = "";
    callHistoryState.items = [];
    return;
  }
  if (callHistoryState.ownerUserID === nextOwner) {
    callHistoryState.items = readOwnerHistory(nextOwner);
    return;
  }
  callHistoryState.ownerUserID = nextOwner;
  callHistoryState.items = readOwnerHistory(nextOwner);
}

export function rememberLocalCall(input: {
  ownerUserID: string;
  id?: string;
  peerUserID: string;
  peerName?: string;
  peerAvatar?: string;
  mediaType: DoMattxCallMediaType;
  direction: CallHistoryDirection;
  actionType: number;
  duration?: number;
  at?: number;
}) {
  const ownerUserID = String(input.ownerUserID || "").trim();
  const peerUserID = String(input.peerUserID || "").trim();
  if (!ownerUserID || !peerUserID) {
    return;
  }

  const item: CallHistoryItem = {
    id: String(input.id || `${peerUserID}-${input.actionType}-${Date.now()}`),
    ownerUserID,
    peerUserID,
    peerName: String(input.peerName || peerUserID),
    peerAvatar: String(input.peerAvatar || ""),
    mediaType:
      Number(input.mediaType) === DoMattxCallMediaType.Video
        ? DoMattxCallMediaType.Video
        : DoMattxCallMediaType.Audio,
    direction: input.direction,
    status: statusFromActionType(input.actionType, Number(input.duration || 0)),
    duration: Math.max(0, Number(input.duration || 0)),
    at: Number(input.at || Date.now()),
  };

  if (callHistoryState.ownerUserID !== ownerUserID) {
    bindCallHistoryOwner(ownerUserID);
  }

  const next = upsertHistoryItem(readOwnerHistory(ownerUserID), item);
  persistOwnerHistory(ownerUserID, next);
  if (callHistoryState.ownerUserID === ownerUserID) {
    callHistoryState.items = next;
  }
}

export function rememberCallFromMessage(
  message: IMessageModel,
  ownerUserID: string,
) {
  if (!isCallMessage(message)) {
    return;
  }
  const owner = String(ownerUserID || "").trim();
  if (!owner) {
    return;
  }
  const data = getCallMessageData(message);
  if (!data) {
    return;
  }
  const peerUserID = resolvePeerUserID(message, owner);
  if (!peerUserID) {
    return;
  }
  const caller = String(getCallMessageCallerUserID(message) || "").trim();
  const mediaType =
    Number(data.call_type || (data.cmd === "videoCall" ? 2 : 1)) ===
    DoMattxCallMediaType.Video
      ? DoMattxCallMediaType.Video
      : DoMattxCallMediaType.Audio;
  rememberLocalCall({
    ownerUserID: owner,
    id: String(message.ID || `${peerUserID}-${message.time || Date.now()}`),
    peerUserID,
    peerName: resolvePeerName(message, peerUserID),
    peerAvatar: resolvePeerAvatar(message),
    mediaType,
    direction: caller ? (caller === owner ? "out" : "in") : message.flow === "out" ? "out" : "in",
    actionType: Number(data.actionType || 0),
    duration: Number(data.call_end || 0),
    at: resolveMessageTime(message),
  });
}

export function formatCallDuration(seconds: number) {
  const total = Math.max(0, Math.floor(seconds));
  const minutes = `${Math.floor(total / 60)}`.padStart(2, "0");
  const rest = `${total % 60}`.padStart(2, "0");
  return `${minutes}:${rest}`;
}

export function formatCallTime(at: number) {
  const date = new Date(at);
  if (Number.isNaN(date.getTime())) {
    return "";
  }
  const now = new Date();
  const sameDay =
    date.getFullYear() === now.getFullYear() &&
    date.getMonth() === now.getMonth() &&
    date.getDate() === now.getDate();
  const hh = `${date.getHours()}`.padStart(2, "0");
  const mm = `${date.getMinutes()}`.padStart(2, "0");
  if (sameDay) {
    return `${hh}:${mm}`;
  }
  return `${date.getMonth() + 1}/${date.getDate()} ${hh}:${mm}`;
}

export function getCallStatusLabel(item: CallHistoryItem) {
  const media = item.mediaType === DoMattxCallMediaType.Video ? "视频通话" : "语音通话";
  if (item.status === "answered" && item.duration > 0) {
    return `${media} · ${formatCallDuration(item.duration)}`;
  }
  if (item.status === "canceled") {
    return `${media} · 已取消`;
  }
  if (item.status === "rejected") {
    return `${media} · 已拒绝`;
  }
  if (item.status === "missed") {
    return `${media} · 未接听`;
  }
  return media;
}

function statusFromActionType(actionType: number, duration: number): CallHistoryStatus {
  if (duration > 0 || actionType === 1) {
    return "answered";
  }
  if (actionType === 2) {
    return "canceled";
  }
  if (actionType === 4) {
    return "rejected";
  }
  return "missed";
}

function upsertHistoryItem(list: CallHistoryItem[], item: CallHistoryItem) {
  const duplicateIndex = list.findIndex((existing) => isSameCallRecord(existing, item));
  const next = [...list];
  if (duplicateIndex >= 0) {
    next[duplicateIndex] = {
      ...next[duplicateIndex],
      ...item,
      peerName: item.peerName || next[duplicateIndex].peerName,
      peerAvatar: item.peerAvatar || next[duplicateIndex].peerAvatar,
    };
  } else {
    next.unshift(item);
  }
  next.sort((a, b) => b.at - a.at);
  return next.slice(0, MAX_ITEMS);
}

function isSameCallRecord(a: CallHistoryItem, b: CallHistoryItem) {
  if (a.id && b.id && a.id === b.id) {
    return true;
  }
  return (
    a.peerUserID === b.peerUserID &&
    a.mediaType === b.mediaType &&
    a.status === b.status &&
    a.direction === b.direction &&
    a.duration === b.duration &&
    Math.abs(a.at - b.at) < DEDUP_WINDOW_MS
  );
}

function readOwnerHistory(ownerUserID: string): CallHistoryItem[] {
  if (typeof localStorage === "undefined") {
    return [];
  }
  try {
    const raw = localStorage.getItem(callHistoryStorageKey(ownerUserID));
    if (!raw) {
      return [];
    }
    const parsed = JSON.parse(raw);
    if (!Array.isArray(parsed)) {
      return [];
    }
    return parsed
      .map(normalizeStoredItem)
      .filter((item): item is CallHistoryItem => Boolean(item && item.ownerUserID === ownerUserID));
  } catch {
    return [];
  }
}

function persistOwnerHistory(ownerUserID: string, items: CallHistoryItem[]) {
  if (typeof localStorage === "undefined") {
    return;
  }
  localStorage.setItem(callHistoryStorageKey(ownerUserID), JSON.stringify(items));
}

function normalizeStoredItem(value: any): CallHistoryItem | null {
  const ownerUserID = String(value?.ownerUserID || "").trim();
  const peerUserID = String(value?.peerUserID || "").trim();
  if (!ownerUserID || !peerUserID) {
    return null;
  }
  return {
    id: String(value.id || `${peerUserID}-${value.at || Date.now()}`),
    ownerUserID,
    peerUserID,
    peerName: String(value.peerName || peerUserID),
    peerAvatar: String(value.peerAvatar || ""),
    mediaType:
      Number(value.mediaType) === DoMattxCallMediaType.Video
        ? DoMattxCallMediaType.Video
        : DoMattxCallMediaType.Audio,
    direction: value.direction === "in" ? "in" : "out",
    status: (
      ["answered", "canceled", "rejected", "missed"] as CallHistoryStatus[]
    ).includes(value.status)
      ? value.status
      : "missed",
    duration: Math.max(0, Number(value.duration || 0)),
    at: Number(value.at || Date.now()),
  };
}

function resolvePeerUserID(message: IMessageModel, ownerUserID: string) {
  const conversationID = String(message.conversationID || "");
  if (conversationID.startsWith("C2C")) {
    return conversationID.slice(3);
  }
  const from = String(message.from || "");
  const to = String((message as any).to || "");
  if (from && from !== ownerUserID) {
    return from;
  }
  if (to && to !== ownerUserID) {
    return to;
  }
  return "";
}

function resolvePeerName(message: IMessageModel, peerUserID: string) {
  return String(
    (message as any)?.nick ||
      (message as any)?.nameCard ||
      message?.nick ||
      peerUserID,
  );
}

function resolvePeerAvatar(message: IMessageModel) {
  return String((message as any)?.avatar || message?.avatar || "");
}

function resolveMessageTime(message: IMessageModel) {
  const rawTime = Number(message.time || 0);
  if (rawTime > 1e12) {
    return rawTime;
  }
  if (rawTime > 0) {
    return rawTime * 1000;
  }
  return Date.now();
}
