import { computed, reactive, readonly } from "vue";
import { newSessionFamilyID, sameSessionFamily, sessionFamilyID, sessionRevision } from "./sessionLineage";
import { createRefreshBackoff, isInvalidSessionError, withTimeout } from "./recovery";
import { TUIUserService } from "@tencentcloud/chat-uikit-engine-lite";
import { APP_NAME, SDK_APP_ID } from "@/config/runtime";
import type {
  ChatAuthResult,
  ChatSessionRecord,
  LoginPayload,
  RegisterPayload,
  TOTPChallenge,
  TOTPSetupChallenge,
} from "./types";
import {
  changePassword,
  deleteAccount,
  login,
  logout,
  refreshSession as refreshAuthSession,
  register,
  setChatAuthToken,
  setupVerifyLoginTOTP,
  updateProfile,
  verifyLoginTOTP,
} from "@/api";
import {
  getInvisiblePreference,
  restoreSelfInvisibleStatus,
} from "@/features/presence/invisible";
import { useSingleFlight } from "@/composables/useSingleFlight";

const STORAGE_KEY = `${APP_NAME}-session`;
const LEGACY_STORAGE_KEYS = ["无忧IM-session"];
const ACCESS_TOKEN_REFRESH_LEEWAY_MS = 60 * 1000;
const USER_SIG_REFRESH_LEEWAY_MS = 24 * 60 * 60 * 1000;

type RedirectRouteName =
  | "welcome"
  | "login"
  | "register"
  | "change-password"
  | "first-setup"
  | "home";

interface AuthSessionState {
  bootstrapped: boolean;
  restoring: boolean;
  authenticating: boolean;
  session: ChatSessionRecord | null;
  recoveryStatus: "idle" | "retrying" | "unavailable";
  imLogoutStatus: "idle" | "confirmed" | "unconfirmed";
  chatStatus: "idle" | "connecting" | "ready" | "error";
}

const state = reactive<AuthSessionState>({
  bootstrapped: false,
  restoring: false,
  authenticating: false,
  session: null,
  recoveryStatus: "idle",
  imLogoutStatus: "idle",
  chatStatus: "idle",
});

let restorePromise: Promise<void> | null = null;
let refreshPromise: { epoch: number; promise: Promise<boolean> } | null = null;
let sessionEpoch = 0;
let sdkQueue: Promise<unknown> = Promise.resolve();
let chatIdentity = "";
let chatPromise: Promise<void> | null = null;
const refreshBackoff = createRefreshBackoff();
let refreshRetryTimer: ReturnType<typeof setTimeout> | null = null;
const authAction = useSingleFlight();

async function getTUILogin() {
  const { TUILogin } = await import("@tencentcloud/tui-core-lite");
  return TUILogin;
}

function requireString(value: unknown, message: string) {
  if (typeof value !== "string" || !value.trim()) {
    throw new Error(message);
  }
  return value;
}

function requireExpectedSDKAppID(value: unknown) {
  if (
    typeof value !== "number" ||
    !Number.isInteger(value) ||
    value !== SDK_APP_ID
  ) {
    throw new Error("聊天服务配置版本不一致，请刷新或更新应用后重试");
  }
  return value;
}

function requireFutureTimestamp(value: unknown, message: string) {
  const raw = requireString(value, message);
  const parsed = Date.parse(raw);
  if (Number.isNaN(parsed) || parsed <= Date.now()) {
    throw new Error(message);
  }
  return raw;
}

function requireFreshUserSigBundle(payload: ChatAuthResult) {
  return {
    userID: requireString(payload.user_id, "聊天登录数据不完整，请重新登录"),
    userSig: requireString(payload.user_sig, "聊天登录凭证缺失，请重新登录"),
    userSigExpiresAt: requireFutureTimestamp(
      payload.user_sig_expires_at,
      "聊天登录凭证已失效，请重新登录"
    ),
  };
}

function toSessionRecord(payload: ChatAuthResult): ChatSessionRecord {
  const sdkAppID = requireExpectedSDKAppID(payload.sdk_app_id);
  const credentials = requireFreshUserSigBundle(payload);
  return {
    sessionFamilyID: newSessionFamilyID(),
    credentialsRevision: 0,
    token: requireString(payload.token, "登录状态无效，请重新登录"),
    refreshToken: requireString(
      payload.refresh_token,
      "登录状态无效，请重新登录"
    ),
    accessTokenExpiresAt: requireFutureTimestamp(
      payload.token_expires_at,
      "登录状态已失效，请重新登录"
    ),
    userID: credentials.userID,
    sdkAppID,
    userSig: credentials.userSig,
    userSigExpiresAt: credentials.userSigExpiresAt,
    mustChangePassword: payload.must_change_password,
    isActivated: payload.is_activated,
  };
}

function mergeAuthResultIntoSession(
  currentSession: ChatSessionRecord | null,
  payload: ChatAuthResult,
  requireNewUserSig: boolean
): ChatSessionRecord {
  if (!currentSession) {
    throw new Error("登录状态无效，请重新登录");
  }
  const sdkAppID = requireExpectedSDKAppID(payload.sdk_app_id);
  const userID = requireString(
    payload.user_id,
    "登录状态无效，请重新登录"
  );
  if (userID !== currentSession.userID) {
    throw new Error("登录账号校验失败，请重新登录");
  }

  const responseContainsUserSig = Boolean(
    payload.user_sig || payload.user_sig_expires_at
  );
  const credentials =
    requireNewUserSig || responseContainsUserSig
      ? requireFreshUserSigBundle(payload)
      : {
          userID: currentSession.userID,
          userSig: requireString(
            currentSession.userSig,
            "聊天登录凭证缺失，请重新登录"
          ),
          userSigExpiresAt: requireFutureTimestamp(
            currentSession.userSigExpiresAt,
            "聊天登录凭证已失效，请重新登录"
          ),
        };

  return {
    sessionFamilyID: sessionFamilyID(currentSession),
    credentialsRevision: sessionRevision(currentSession) + 1,
    token: requireString(payload.token, "登录状态无效，请重新登录"),
    refreshToken: requireString(
      payload.refresh_token,
      "登录状态无效，请重新登录"
    ),
    accessTokenExpiresAt: requireFutureTimestamp(
      payload.token_expires_at,
      "登录状态已失效，请重新登录"
    ),
    userID,
    sdkAppID,
    userSig: credentials.userSig,
    userSigExpiresAt: credentials.userSigExpiresAt,
    mustChangePassword: payload.must_change_password,
    isActivated: payload.is_activated,
  };
}

function readSharedSession(): ChatSessionRecord | null {
  const raw = localStorage.getItem(STORAGE_KEY);
  return raw ? parsePersistedSession(raw) : null;
}

function syncPersistedSession(session: ChatSessionRecord | null, options: { newLogin?: boolean; clearedSession?: ChatSessionRecord | null } = {}) {
  setChatAuthToken(session?.token ?? null, session ? `${sessionEpoch}:${session.userID}` : null);
  const stored = readSharedSession();
  if (session) {
    // An old tab cannot overwrite a newer login (including the same user), or
    // resurrect a shared session another tab explicitly removed.
    if (options.newLogin || (sameSessionFamily(stored, session) && sessionRevision(stored!) <= sessionRevision(session))) {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(session));
    }
    sessionStorage.removeItem(STORAGE_KEY);
  } else {
    if (sameSessionFamily(stored, options.clearedSession || null) && sessionRevision(stored!) <= sessionRevision(options.clearedSession!)) {
      localStorage.removeItem(STORAGE_KEY);
      localStorage.removeItem("TUIKit-userInfo");
    }
    sessionStorage.removeItem(STORAGE_KEY);
    sessionStorage.removeItem("TUIKit-userInfo");
  }
}

function adoptSharedCredentials() {
  const current = state.session;
  const stored = readSharedSession();
  if (!current || !stored || !sameSessionFamily(current, stored) || sessionRevision(stored) <= sessionRevision(current)) return false;
  state.session = stored;
  syncPersistedSession(stored);
  resetRefreshRecovery();
  return true;
}

function ensureChatConnected() {
  const session = state.session;
  return session ? loginIM(session) : Promise.resolve();
}

function loginIM(session: ChatSessionRecord) {
  const epoch = sessionEpoch;
  const identity = `${session.sdkAppID}:${session.userID}:${session.userSig}`;
  if (chatIdentity === identity && (chatPromise || state.chatStatus === 'ready')) return chatPromise || Promise.resolve();
  chatIdentity = identity;
  state.chatStatus = 'connecting';
  const operation = sdkQueue.catch(() => {}).then(async () => {
    if (epoch !== sessionEpoch) return;
    try {
      await performLoginIM(session, epoch);
      if (epoch === sessionEpoch) state.chatStatus = 'ready';
    } catch (error) { if (epoch === sessionEpoch) state.chatStatus = 'error'; throw error; }
  });
  sdkQueue = operation;
  const visible = withTimeout(operation, 12000);
  chatPromise = visible;
  void visible.then(() => { if (chatPromise === visible) chatPromise = null; }, () => {
    if (chatPromise === visible) chatPromise = null;
    if (epoch === sessionEpoch) state.chatStatus = 'error';
  });
  return visible;
}

async function performLoginIM(session: ChatSessionRecord, epoch: number) {
  requireExpectedSDKAppID(session.sdkAppID);
  const TUILogin = await getTUILogin();
  const [{ framework }, runtime, callRuntime] = await Promise.all([
    import("@/TUIKit/adapter-vue"),
    import("@/bootstrap/chatRuntime"),
    import("@/bootstrap/callRuntime"),
  ]);
  await callRuntime.prepareCallIntegration();
  if (epoch !== sessionEpoch) return;
  runtime.ensureChatRuntime();
  if (epoch !== sessionEpoch) return;
  await TUILogin.login({
    SDKAppID: session.sdkAppID,
    userID: session.userID,
    userSig: session.userSig,
    useUploadPlugin: true,
    framework,
  } as any);
  if (epoch !== sessionEpoch) return;
  runtime.loginChatRuntime();
  (await import("@/features/call/doMattxCallBridge")).ensureDoMattxCallBridge();
  try {
    await TUIUserService.switchUserStatus({ displayOnlineStatus: true });
  } catch (error) {
    console.warn("enable online status display failed", error);
  }
  const invisiblePreference = getInvisiblePreference();
  try {
    if (invisiblePreference) {
      await restoreSelfInvisibleStatus();
    }
  } catch (error) {
    console.warn("restore invisible status failed", error);
  }
}

async function applyAuthResult(result: ChatAuthResult) {
  const nextSession = toSessionRecord(result);
  const epoch = ++sessionEpoch;
  resetRefreshRecovery();
  state.session = nextSession;
  syncPersistedSession(nextSession, { newLogin: true });
  try { await loginIM(nextSession); }
  catch (error) {
    if (epoch === sessionEpoch) state.recoveryStatus = "unavailable";
    throw error;
  }
}

function readPersistedSession(): ChatSessionRecord | null {
  const rawSession =
    localStorage.getItem(STORAGE_KEY) ??
    sessionStorage.getItem(STORAGE_KEY) ??
    LEGACY_STORAGE_KEYS
      .map((key) => localStorage.getItem(key))
      .find((value) => Boolean(value));
  if (!rawSession) {
    return null;
  }

  localStorage.setItem(STORAGE_KEY, rawSession);
  sessionStorage.removeItem(STORAGE_KEY);
  for (const legacyKey of LEGACY_STORAGE_KEYS) {
    if (legacyKey !== STORAGE_KEY) {
      localStorage.removeItem(legacyKey);
    }
  }
  localStorage.removeItem("TUIKit-userInfo");

  return parsePersistedSession(rawSession);
}

function parsePersistedSession(rawSession: string): ChatSessionRecord | null {
  try {
    const parsed = JSON.parse(rawSession) as Partial<ChatSessionRecord>;
    if (!parsed?.token || !parsed?.userID || !parsed?.userSig) {
      return null;
    }

    return {
      sessionFamilyID: sessionFamilyID(parsed),
      credentialsRevision: sessionRevision(parsed),
      token: parsed.token,
      refreshToken: parsed.refreshToken ?? "",
      accessTokenExpiresAt: parsed.accessTokenExpiresAt ?? "",
      userID: parsed.userID,
      sdkAppID:
        typeof parsed.sdkAppID === "number" && Number.isInteger(parsed.sdkAppID)
          ? parsed.sdkAppID
          : 0,
      userSig: parsed.userSig ?? "",
      userSigExpiresAt: parsed.userSigExpiresAt ?? "",
      mustChangePassword: Boolean(parsed.mustChangePassword),
      isActivated: Boolean(parsed.isActivated),
    };
  } catch (error) {
    console.warn("parse session failed", error);
    return null;
  }
}

function parseTimestamp(raw?: string) {
  if (!raw) {
    return null;
  }
  const value = Date.parse(raw);
  if (Number.isNaN(value)) {
    return null;
  }
  return value;
}

function isTimestampExpiring(raw: string, leewayMs: number) {
  const parsed = parseTimestamp(raw);
  if (!parsed) {
    return true;
  }
  return Date.now() >= parsed - leewayMs;
}

function needsAccessTokenRefresh(session: ChatSessionRecord) {
  return isTimestampExpiring(
    session.accessTokenExpiresAt,
    ACCESS_TOKEN_REFRESH_LEEWAY_MS
  );
}

function needsUserSigRefresh(session: ChatSessionRecord) {
  if (!session.userSig || session.sdkAppID !== SDK_APP_ID) {
    return true;
  }
  return isTimestampExpiring(
    session.userSigExpiresAt,
    USER_SIG_REFRESH_LEEWAY_MS
  );
}

function resolveAuthenticatedRouteName(): RedirectRouteName {
  return state.session ? "home" : "welcome";
}

function resetRefreshRecovery() {
  if (refreshRetryTimer) clearTimeout(refreshRetryTimer);
  refreshRetryTimer = null;
  refreshBackoff.reset();
  state.recoveryStatus = "idle";
}

function scheduleRefreshRecovery(error: unknown, epoch: number) {
  if (refreshRetryTimer) clearTimeout(refreshRetryTimer);
  refreshRetryTimer = null;
  const { delayMs, retryAutomatically } = refreshBackoff.fail(error);
  state.recoveryStatus = retryAutomatically ? "retrying" : "unavailable";
  if (retryAutomatically) {
    refreshRetryTimer = setTimeout(() => {
      refreshRetryTimer = null;
      if (epoch !== sessionEpoch || !state.session) return;
      void refreshCurrentSession({ force: true }).catch(() => {});
    }, delayMs);
  }
}

async function refreshCurrentSession(options?: {
  force?: boolean;
  reloginIMIfUserSigChanges?: boolean;
  requireUserSig?: boolean;
}) {
  const adopted = adoptSharedCredentials();
  const currentSession = state.session;
  if (!currentSession) return false;
  const force = options?.force ?? false;
  const mustRefreshUserSig = Boolean(options?.requireUserSig) || needsUserSigRefresh(currentSession);
  if ((!force || adopted) && !needsAccessTokenRefresh(currentSession) && !mustRefreshUserSig) {
    if (adopted && options?.reloginIMIfUserSigChanges !== false) void ensureChatConnected().catch(() => { state.recoveryStatus = "unavailable"; });
    return true;
  }
  if (!currentSession.refreshToken) {
    if (force || needsAccessTokenRefresh(currentSession)) {
      void clearSession();
      return false;
    }
    return true;
  }
  const epoch = sessionEpoch;
  if (refreshPromise?.epoch === epoch) return refreshPromise.promise;
  refreshBackoff.assertReady();
  const operation = { epoch, promise: Promise.resolve(false) };
  operation.promise = (async () => {
    let sourceSession = currentSession;
    try {
      // One additional attempt is allowed only when another tab rotated this
      // exact login family while our old-token request was in flight.
      for (let attempt = 0; attempt < 2; attempt++) {
        let refreshed: ChatAuthResult;
        try {
          refreshed = await refreshAuthSession({
            refreshToken: sourceSession.refreshToken,
            needUserSigRefresh: mustRefreshUserSig,
            userSigExpiresAt: sourceSession.userSigExpiresAt,
            sdkAppID: SDK_APP_ID,
          });
        } catch (error) {
          if (epoch !== sessionEpoch || !state.session) return false;
          const sharedAdvanced = adoptSharedCredentials() || sessionRevision(state.session) > sessionRevision(sourceSession);
          if (isInvalidSessionError(error) && attempt === 0 && sharedAdvanced && state.session.refreshToken !== sourceSession.refreshToken) {
            sourceSession = state.session;
            continue;
          }
          throw error;
        }
        if (epoch !== sessionEpoch || !state.session) return false;
        adoptSharedCredentials();
        // Prefer a newer shared rotation over a late reply based on an older one.
        if (sessionRevision(state.session) <= sessionRevision(sourceSession)) {
          const nextSession = mergeAuthResultIntoSession(state.session, refreshed, mustRefreshUserSig);
          state.session = nextSession;
          syncPersistedSession(nextSession);
        }
        resetRefreshRecovery();
        if (options?.reloginIMIfUserSigChanges !== false && (refreshed.user_sig || adopted)) {
          void ensureChatConnected().catch(() => { state.recoveryStatus = "unavailable"; });
        }
        return true;
      }
      return false;
    } catch (error) {
      if (epoch !== sessionEpoch) return false;
      if (isInvalidSessionError(error)) {
        if (adoptSharedCredentials() || (state.session && sessionRevision(state.session) > sessionRevision(sourceSession))) {
          if (options?.reloginIMIfUserSigChanges !== false) void ensureChatConnected().catch(() => { state.recoveryStatus = "unavailable"; });
          return true;
        }
        void clearSession();
        return false;
      }
      scheduleRefreshRecovery(error, epoch);
      throw error;
    } finally {
      if (refreshPromise === operation) refreshPromise = null;
    }
  })();
  refreshPromise = operation;
  return operation.promise;
}

async function ensureInitialized() {
  if (state.bootstrapped) return;
  if (!restorePromise) {
    restorePromise = (async () => {
      state.restoring = true;
      try {
        const persisted = readPersistedSession();
        if (!persisted) return;
        ++sessionEpoch;
        state.session = persisted;
        syncPersistedSession(persisted);
        try { await refreshCurrentSession({ reloginIMIfUserSigChanges: false }); }
        catch { /* A transient refresh failure preserves credentials. */ }
        if (state.session && !needsUserSigRefresh(state.session)) {
          try { await ensureChatConnected(); }
          catch { state.recoveryStatus = "unavailable"; }
        }
      } finally {
        state.bootstrapped = true;
        state.restoring = false;
        restorePromise = null;
      }
    })();
  }
  await restorePromise;
}

async function retryConnection() {
  refreshBackoff.assertReady();
  resetRefreshRecovery();
  if (await refreshCurrentSession({ force: true })) await ensureChatConnected();
}

async function withAuthAction<T>(handler: () => Promise<T>) {
  return authAction.run(async () => {
    state.authenticating = true;
    try {
      return await handler();
    } finally {
      state.authenticating = false;
    }
  });
}

async function loginWithPassword(phone: string, password: string, turnstileToken?: string) {
  return withAuthAction(async () => {
    const payload: LoginPayload = { phone, password, turnstile_token: turnstileToken };
    const result = await login(payload);
    if (result.require_totp_setup && result.temp_token) {
      return {
        type: "totp_setup",
        tempToken: result.temp_token,
        totpSecret: result.totp_secret || "",
        totpUri: result.totp_uri || "",
      } as TOTPSetupChallenge;
    }
    if (result.require_totp_verify && result.temp_token) {
      return { type: "totp", tempToken: result.temp_token } as TOTPChallenge;
    }
    await applyAuthResult(result);
    return resolveAuthenticatedRouteName();
  });
}

async function completeTOTPLogin(
  tempToken: string,
  code: string,
  mode: "verify" | "setup" = "verify"
) {
  return withAuthAction(async () => {
    const result =
      mode === "setup"
        ? await setupVerifyLoginTOTP(tempToken, code)
        : await verifyLoginTOTP(tempToken, code);
    await applyAuthResult(result);
    return resolveAuthenticatedRouteName();
  });
}

async function registerAccount(payload: RegisterPayload) {
  return withAuthAction(async () => {
    const result = await register(payload);
    await applyAuthResult(result);
    return resolveAuthenticatedRouteName();
  });
}

async function completePasswordChange(newPassword: string) {
  return withAuthAction(async () => {
    // 服务端修改密码后会废弃旧 token，并在同一响应里返回新登录态。
    // IM 当前会话不受业务密码影响，只更换业务登录态，避免重复登录造成界面闪动。
    const result = await changePassword(newPassword);
    const nextSession = toSessionRecord(result);
    ++sessionEpoch;
    state.session = nextSession;
    syncPersistedSession(nextSession, { newLogin: true });
    return resolveAuthenticatedRouteName();
  });
}

async function completeFirstSetup(nickname: string) {
  return withAuthAction(async () => {
    await updateProfile({ nickname });
    const { TUIUserService } = await import("@tencentcloud/chat-uikit-engine-lite");
    await TUIUserService.updateMyProfile({ nick: nickname } as any);
    if (state.session) {
      state.session.isActivated = true;
      syncPersistedSession(state.session);
    }
    return resolveAuthenticatedRouteName();
  });
}

async function clearSession() {
  const clearedSession = state.session;
  ++sessionEpoch;
  resetRefreshRecovery();
  state.session = null;
  chatIdentity = "";
  chatPromise = null;
  state.chatStatus = "idle";
  syncPersistedSession(null, { clearedSession });
  const cleanup = sdkQueue.catch(() => {}).then(async () => {
    try {
      await (await import("@/features/call/doMattxCallBridge")).disposeDoMattxCallBridge();
      const TUILogin = await getTUILogin();
      await TUILogin.logout();
      state.imLogoutStatus = "confirmed";
    } catch (error) {
      state.imLogoutStatus = "unconfirmed";
      console.warn("IM退出未确认，需重连核对");
      throw error;
    }
  });
  sdkQueue = cleanup;
  await withTimeout(cleanup, 12000);
}

async function logoutSession(options?: { skipServerLogout?: boolean }) {
  adoptSharedCredentials();
  const refreshToken = state.session?.refreshToken;
  const cleanup = clearSession();
  const serverLogout = !options?.skipServerLogout && refreshToken ? logout(refreshToken) : Promise.resolve();
  const outcomes = await Promise.allSettled([cleanup, withTimeout(serverLogout, 5000)]);
  if (outcomes.some(result => result.status === "rejected")) console.warn("退出清理存在未确认结果");
}

async function deleteCurrentAccount() {
  return withAuthAction(async () => {
    await deleteAccount();
    await logoutSession({ skipServerLogout: true });
  });
}

function updateSessionFlags(
  patch: Partial<Pick<ChatSessionRecord, "mustChangePassword" | "isActivated">>
) {
  if (!state.session) {
    return;
  }
  state.session = {
    ...state.session,
    ...patch,
  };
  syncPersistedSession(state.session);
}

export function useAuthSession() {
  return {
    state: readonly(state),
    isAuthenticated: computed(() =>
      Boolean(
        state.session?.token && state.session?.userID && state.session?.userSig
      )
    ),
    currentSession: computed(() => state.session),
    resolveAuthenticatedRouteName,
    ensureInitialized,
    ensureChatConnected,
    refreshCurrentSession,
    retryConnection,
    loginWithPassword,
    completeTOTPLogin,
    registerAccount,
    completePasswordChange,
    completeFirstSetup,
    logoutSession,
    clearSession,
    deleteCurrentAccount,
    updateSessionFlags,
  };
}

window.addEventListener("storage", event => {
  if (event.key !== STORAGE_KEY || !state.session) return;
  const stored = readSharedSession();
  if (!sameSessionFamily(state.session, stored)) {
    void clearSession().catch(() => {});
  } else adoptSharedCredentials();
});
