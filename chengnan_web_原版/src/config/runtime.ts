import {
  CLOUDFLARE_CHAT_API_BASE_URL,
  resolveProdChatAPIBaseURL,
} from "./chat-api-routing";

const DEFAULT_DEV_CHAT_API_BASE_URL = '/api/v1/chat';
const DEFAULT_SDK_APP_ID = 1600163137;
const DEFAULT_CHAT_API_TIMEOUT_MS = 15_000;

const parsedSDKAppID = Number(
  import.meta.env.VITE_IM_SDK_APP_ID ?? DEFAULT_SDK_APP_ID
);
const parsedChatAPITimeout = Number(
  import.meta.env.VITE_CHAT_API_TIMEOUT_MS ?? DEFAULT_CHAT_API_TIMEOUT_MS
);

function resolveCurrentProdChatAPIBaseURL() {
  if (typeof window === "undefined") {
    return CLOUDFLARE_CHAT_API_BASE_URL;
  }
  return resolveProdChatAPIBaseURL(window.location.hostname);
}

export const APP_NAME = "睿之禾";
export const DYNAMIC_API_ROUTING = import.meta.env.PROD && !import.meta.env.VITE_CHAT_API_BASE_URL;
export const CHAT_API_BASE_URL =
  import.meta.env.VITE_CHAT_API_BASE_URL ||
  (import.meta.env.PROD
    ? resolveCurrentProdChatAPIBaseURL()
    : DEFAULT_DEV_CHAT_API_BASE_URL);
export const CHAT_API_TIMEOUT_MS =
  Number.isFinite(parsedChatAPITimeout) &&
  parsedChatAPITimeout >= 5_000 &&
  parsedChatAPITimeout <= 60_000
    ? parsedChatAPITimeout
    : DEFAULT_CHAT_API_TIMEOUT_MS;
export const SDK_APP_ID =
  Number.isFinite(parsedSDKAppID) && parsedSDKAppID > 0
    ? parsedSDKAppID
    : DEFAULT_SDK_APP_ID;
export const WEB_TURNSTILE_SITE_KEY =
  import.meta.env.VITE_WEB_TURNSTILE_SITE_KEY || "";

export function warmUpChatAPIConnection() {
  if (
    typeof document === 'undefined' ||
    !CHAT_API_BASE_URL.startsWith('http')
  ) {
    return;
  }

  const origin = new URL(CHAT_API_BASE_URL).origin;
  if (document.head.querySelector(`link[data-chat-api-origin="${origin}"]`)) {
    return;
  }

  const dnsPrefetch = document.createElement('link');
  dnsPrefetch.rel = 'dns-prefetch';
  dnsPrefetch.href = origin;
  dnsPrefetch.dataset.chatApiOrigin = origin;
  document.head.appendChild(dnsPrefetch);

  const preconnect = document.createElement('link');
  preconnect.rel = 'preconnect';
  preconnect.href = origin;
  preconnect.crossOrigin = 'anonymous';
  preconnect.dataset.chatApiOrigin = origin;
  document.head.appendChild(preconnect);
}
