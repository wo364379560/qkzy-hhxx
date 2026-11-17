import { APP_NAME } from "@/config/runtime";

const STORAGE_KEY = `${APP_NAME}-remembered-login`;

export interface RememberedLoginCredentials {
  phone: string;
  password: string;
}

function getRememberCredentialsStorage() {
  if (typeof window === "undefined") {
    return null;
  }
  try {
    return window.localStorage;
  } catch (error) {
    console.warn("remember credentials storage unavailable", error);
    return null;
  }
}

export function readRememberedLoginCredentials(): RememberedLoginCredentials | null {
  const storage = getRememberCredentialsStorage();
  if (!storage) {
    return null;
  }

  try {
    const rawValue =
      storage.getItem(STORAGE_KEY) ?? window.sessionStorage.getItem(STORAGE_KEY);
    if (!rawValue) {
      return null;
    }
    storage.setItem(STORAGE_KEY, rawValue);
    window.sessionStorage.removeItem(STORAGE_KEY);
    const parsed = JSON.parse(rawValue) as Partial<RememberedLoginCredentials>;
    if (typeof parsed.phone !== "string" || typeof parsed.password !== "string") {
      return null;
    }
    return {
      phone: parsed.phone,
      password: parsed.password,
    };
  } catch (error) {
    console.warn("read remembered credentials failed", error);
    return null;
  }
}

export function saveRememberedLoginCredentials(credentials: RememberedLoginCredentials) {
  const storage = getRememberCredentialsStorage();
  if (!storage) {
    return;
  }

  storage.setItem(
    STORAGE_KEY,
    JSON.stringify({
      phone: credentials.phone,
      password: credentials.password,
    })
  );
  window.sessionStorage.removeItem(STORAGE_KEY);
}

export function clearRememberedLoginCredentials() {
  getRememberCredentialsStorage()?.removeItem(STORAGE_KEY);
  window.sessionStorage.removeItem(STORAGE_KEY);
}
