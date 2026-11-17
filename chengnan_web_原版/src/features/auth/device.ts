const DEVICE_STORAGE_KEY = "chengnanim-device-id";
const LEGACY_DEVICE_STORAGE_KEYS = ["wuyouim-device-id"];

function generateDeviceID() {
  if (typeof crypto !== "undefined" && typeof crypto.randomUUID === "function") {
    return `web_${crypto.randomUUID().replace(/-/g, "").toLowerCase()}`;
  }

  const random = Math.random().toString(36).slice(2);
  const timestamp = Date.now().toString(36);
  return `web_${timestamp}${random}`;
}

export function getOrCreateDeviceID() {
  if (typeof window === "undefined") {
    return "web_server";
  }

  const existing =
    window.localStorage.getItem(DEVICE_STORAGE_KEY)?.trim() ??
    window.sessionStorage.getItem(DEVICE_STORAGE_KEY)?.trim() ??
    "";
  if (existing) {
    window.localStorage.setItem(DEVICE_STORAGE_KEY, existing);
    window.sessionStorage.removeItem(DEVICE_STORAGE_KEY);
    return existing;
  }

  for (const legacyKey of LEGACY_DEVICE_STORAGE_KEYS) {
    const legacyValue = window.localStorage.getItem(legacyKey)?.trim() ?? "";
    if (legacyValue) {
      window.localStorage.setItem(DEVICE_STORAGE_KEY, legacyValue);
      window.sessionStorage.removeItem(DEVICE_STORAGE_KEY);
      return legacyValue;
    }
  }

  const next = generateDeviceID();
  window.localStorage.setItem(DEVICE_STORAGE_KEY, next);
  return next;
}
