export const MAIN_CHAT_API_BASE_URL = "https://api.prfgsu.com/api/v1/chat";
export const BACKUP_CHAT_API_BASE_URL = "https://api.uktjwg.cn/api/v1/chat";
export const MAIN_CDN_CHAT_API_BASE_URL = "https://api1.prfgsu.com/api/v1/chat";
export const BACKUP_CDN_CHAT_API_BASE_URL = "https://api1.uktjwg.cn/api/v1/chat";
export const CLOUDFLARE_CHAT_API_BASE_URL =
  "https://api.ruizhihe.com/api/v1/chat";

export const APP_BOOTSTRAP_URLS = [
  "https://api.prfgsu.com/api/v1/public/bootstrap/domain-pool",
  "https://api.uktjwg.cn/api/v1/public/bootstrap/domain-pool",
  "https://api1.prfgsu.com/api/v1/public/bootstrap/domain-pool",
  "https://api1.uktjwg.cn/api/v1/public/bootstrap/domain-pool",
  "https://api.ruizhihe.com/api/v1/public/bootstrap/domain-pool",
] as const;

const CHAT_API_BASE_URL_BY_HOST: Readonly<Record<string, string>> = {
  "api.prfgsu.com": MAIN_CHAT_API_BASE_URL,
  "api.uktjwg.cn": BACKUP_CHAT_API_BASE_URL,
  "api1.prfgsu.com": MAIN_CDN_CHAT_API_BASE_URL,
  "api1.uktjwg.cn": BACKUP_CDN_CHAT_API_BASE_URL,
  "api.ruizhihe.com": CLOUDFLARE_CHAT_API_BASE_URL,
};

export function resolveProdChatAPIBaseURL(hostname: string) {
  return (
    CHAT_API_BASE_URL_BY_HOST[hostname.trim().toLowerCase()] ||
    CLOUDFLARE_CHAT_API_BASE_URL
  );
}
