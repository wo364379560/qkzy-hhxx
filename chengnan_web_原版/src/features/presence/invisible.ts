import TUIChatEngine from "@tencentcloud/chat-uikit-engine-lite";

export const INVISIBLE_CUSTOM_STATUS = "offline";

const INVISIBLE_STATUS_STORAGE_KEY_PREFIX = "chengnan:h5:invisible-status";
const INVISIBLE_CUSTOM_STATUS_SET = new Set([
  INVISIBLE_CUSTOM_STATUS,
  "chengnan:invisible",
  "chengnan_invisible",
  "invisible",
]);

interface UserStatusLike {
  statusType?: number;
  customStatus?: string | null;
}

function normalizeCustomStatus(customStatus?: string | null) {
  return String(customStatus || "")
    .trim()
    .toLowerCase();
}

function canUseLocalStorage() {
  try {
    return typeof window !== "undefined" && Boolean(window.localStorage);
  } catch (_error) {
    return false;
  }
}

function getInvisibleStatusStorageKey() {
  const userID = TUIChatEngine.getMyUserID?.() || "";
  return userID
    ? `${INVISIBLE_STATUS_STORAGE_KEY_PREFIX}:${userID}`
    : INVISIBLE_STATUS_STORAGE_KEY_PREFIX;
}

export function isInvisibleCustomStatus(customStatus?: string | null) {
  return INVISIBLE_CUSTOM_STATUS_SET.has(normalizeCustomStatus(customStatus));
}

export function isUserOnlineStatus(status?: UserStatusLike | null) {
  return Boolean(
    status &&
      Number(status.statusType || 0) ===
        TUIChatEngine.TYPES.USER_STATUS_ONLINE &&
      !isInvisibleCustomStatus(status.customStatus)
  );
}

export function getInvisiblePreference() {
  if (!canUseLocalStorage()) {
    return false;
  }
  try {
    return window.localStorage.getItem(getInvisibleStatusStorageKey()) === "1";
  } catch (_error) {
    return false;
  }
}

export function setInvisiblePreference(enabled: boolean) {
  if (!canUseLocalStorage()) {
    return;
  }
  try {
    window.localStorage.setItem(
      getInvisibleStatusStorageKey(),
      enabled ? "1" : "0"
    );
  } catch (_error) {
    // Ignore storage failures; the SDK status has already been set.
  }
}

async function setSelfCustomStatus(customStatus: string) {
  const chat = TUIChatEngine.chat as
    | {
        setSelfStatus?: (options: { customStatus: string }) => Promise<unknown>;
      }
    | undefined;

  if (typeof chat?.setSelfStatus !== "function") {
    throw new Error("setSelfStatus is not available");
  }

  await chat.setSelfStatus({ customStatus });
}

export async function setSelfInvisibleStatus(enabled: boolean) {
  await setSelfCustomStatus(enabled ? INVISIBLE_CUSTOM_STATUS : "");
  setInvisiblePreference(enabled);
}

export async function restoreSelfInvisibleStatus() {
  if (!getInvisiblePreference()) {
    return;
  }
  await setSelfInvisibleStatus(true);
}
