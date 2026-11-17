import type { ChatSessionRecord } from "./types";

// Local bookkeeping only: these IDs are never authentication credentials or sent
// to the server. Explicit login starts a new family; refresh preserves it.
export function newSessionFamilyID() {
  return globalThis.crypto?.randomUUID?.() || `login-${Date.now()}-${Math.random().toString(36).slice(2)}`;
}

export function sessionFamilyID(session: Partial<ChatSessionRecord>): string {
  if (session.sessionFamilyID) return session.sessionFamilyID;
  // Stable migration lets two tabs reading the same pre-upgrade record agree.
  // Use the old credential as input so a later login of the same user is distinct.
  const input = `${session.sdkAppID}:${session.userID}:${session.refreshToken || session.token}`;
  let a = 2166136261, b = 2246822507;
  for (let i = 0; i < input.length; i++) {
    a = Math.imul(a ^ input.charCodeAt(i), 16777619);
    b = Math.imul(b ^ input.charCodeAt(i), 3266489909);
  }
  return `legacy-${(a >>> 0).toString(36)}-${(b >>> 0).toString(36)}`;
}

export function sameSessionFamily(a: ChatSessionRecord | null, b: ChatSessionRecord | null) {
  return Boolean(a && b && a.userID === b.userID && (!a.sdkAppID || !b.sdkAppID || a.sdkAppID === b.sdkAppID) && sessionFamilyID(a) === sessionFamilyID(b));
}

export function sessionRevision(session: Partial<ChatSessionRecord>) {
  const value = session.credentialsRevision;
  return typeof value === "number" && Number.isSafeInteger(value) && value >= 0 ? value : 0;
}
