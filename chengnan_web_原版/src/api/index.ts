import { tracePhase, newTraceID } from "@/utils/networkDiagnostics";
import { prepareAvatar } from "./avatarFile";
import { uploadMultipart } from "./multipart";
import axios from "axios";
import { ApiError, apiErrorDetails } from "@/features/auth/recovery";
import { CHAT_API_BASE_URL, CHAT_API_TIMEOUT_MS, DYNAMIC_API_ROUTING } from "@/config/runtime";
import { domainPool, retryableRouteFailure, waitWithinDeadline } from "@/config/domain-pool";
import { getOrCreateDeviceID } from "@/features/auth/device";
import { createAsyncReadCache } from "./async-read-cache";
import type {
  AvatarPresignResult,
  ChatAttachmentAccessResult,
  ChatAttachmentPresignResult,
  ChatAttachmentUploadSettings,
  ChatMediaPresignResult,
  ChatAuthResult,
  ChatProfile,
  LoginPayload,
  RegisterPayload,
  SearchGroupResult,
  SearchUserResult,
  SubmitReportPayload,
  UpdateProfilePayload,
} from "@/features/auth/types";

type ApiResponse<T> = {
  code: number;
  message?: string;
  msg?: string;
  data: T;
};

type UnauthorizedHandler = () => boolean | Promise<boolean>;
type UploadProgressCallback = (progress: number) => void;
type UploadRequestOptions = {
  onProgress?: UploadProgressCallback;
  signal?: AbortSignal;
  isVideo?: boolean;
  contentType?: string;
};

export class UploadCanceledError extends Error {
  constructor() {
    super("上传已取消");
    this.name = "UploadCanceledError";
  }
}

class DirectCosUploadError extends Error {
  readonly progress: number;

  constructor(progress: number) {
    super(
      progress >= 0.99
        ? "COS 已接收文件，但浏览器未收到上传响应"
        : "上传连接中断，未收到存储服务响应；请检查网络后重试"
    );
    this.name = "DirectCosUploadError";
    this.progress = progress;
  }
}

export function isUploadCanceledError(error: unknown) {
  return (
    error instanceof UploadCanceledError ||
    (error instanceof Error && ["UploadCanceledError", "AbortError"].includes(error.name))
  );
}

export type ConversationCustomPin = {
  conversationID: string;
  pinOrder: number;
  pinnedAt: string;
};

export type GroupNotice = {
  pinned: boolean;
  groupID: number;
  content: string;
  summary: string;
  source: string;
  hasValue: boolean;
};

export type GroupMessagePin = {
  group_id: number;
  im_group_id: string;
  message_id: string;
  message_seq: number;
  message_sender_im_user_id: string;
  message_sender_name: string;
  message_type: string;
  message_abstract: string;
  status: string;
  pinned_at: string;
};

export type GroupAdminMember = {
  user_id: string;
  nickname?: string;
  avatar_url?: string;
  role: "Owner" | "Admin" | "Member" | string;
  role_source: "tencent" | "business" | string;
};

export type GroupAdmins = {
  owner?: GroupAdminMember;
  admins: GroupAdminMember[];
};

export type GroupSlowMode = {
  group_id: string;
  slow_mode_enabled: boolean;
  slow_mode_seconds: number;
  can_set_slow_mode: boolean;
};

export type UserLastOnline = {
  im_user_id: string;
  last_online_at?: string;
  last_offline_at?: string;
};

export type UserLoginIPResult = {
  im_user_id?: string;
  current_login_ip?: string;
  current_login_location?: string;
};

const USER_LAST_ONLINE_BATCH_SIZE = 100;
const USER_LAST_ONLINE_BATCH_DELAY_MS = 1000;
const GROUP_MESSAGE_PIN_CACHE_TTL_MS = 5 * 60 * 1000;
const GROUP_ADMINS_CACHE_TTL_MS = 2 * 60 * 1000;

// Collapse concurrent component reads only; never retain permission/profile data.
const memberFilterReads = createAsyncReadCache<FilterGroupMembersResult>({ ttlMs: 0 });
const profileReads = createAsyncReadCache<ChatProfile>({ ttlMs: 0 });
const conversationPinReads = createAsyncReadCache<ConversationCustomPin[]>({ ttlMs: 0 });
let authToken: string | null = null;
let unauthorizedHandler: UnauthorizedHandler | null = null;
let unauthorizedHandlingPromise: { epoch: number; promise: Promise<boolean> } | null = null;
let authEpoch = 0;
let authContext: string | null = null;
const userLastOnlineCache = new Map<string, UserLastOnline>();
const groupMessagePinCache = createAsyncReadCache<GroupMessagePin | null>({
  ttlMs: GROUP_MESSAGE_PIN_CACHE_TTL_MS,
});
const groupAdminsCache = createAsyncReadCache<GroupAdmins>({
  ttlMs: GROUP_ADMINS_CACHE_TTL_MS,
});
type PendingUserLastOnlineRequest = {
  resolve: () => void;
  reject: (error: unknown) => void;
};
const pendingUserLastOnlineIDs = new Set<string>();
const pendingUserLastOnlineRequests: PendingUserLastOnlineRequest[] = [];
let pendingUserLastOnlineTimer: ReturnType<typeof setTimeout> | null = null;

const chatClient = axios.create({
  baseURL: CHAT_API_BASE_URL,
  timeout: CHAT_API_TIMEOUT_MS,
  headers: {
    "Content-Type": "application/json",
  },
});

chatClient.interceptors.request.use(async (config) => {
  const requestConfig = config as any;
  // Retried requests keep their original account epoch. Never replay an old
  // account's action using the credentials of a later login.
  if (requestConfig._authEpoch !== undefined && requestConfig._authEpoch !== authEpoch) {
    return Promise.reject(new ApiError("登录账号已切换，请重新操作", { code: "SESSION_CHANGED" }));
  }
  requestConfig._authEpoch = authEpoch;
  requestConfig._traceID ??= newTraceID();
  const routeStarted = Date.now();
  requestConfig._deadline ??= Date.now() + Math.min(60_000, config.timeout || CHAT_API_TIMEOUT_MS);
  if (DYNAMIC_API_ROUTING) {
    // Public metadata refresh must not delay an available business route.
    // Actual route recovery below still shares the request's original deadline.
    void domainPool.refresh().catch(() => {});
    try { await waitWithinDeadline(domainPool.waitForRecovery(), requestConfig._deadline); }
    catch (error) { tracePhase('route_failed', routeStarted, { requestId: requestConfig._traceID }); throw error; }
    if (requestConfig._authEpoch !== authEpoch) throw new ApiError("登录账号已切换，请重新操作", { code: "SESSION_CHANGED" });
    const pool = domainPool.snapshot();
    if (!requestConfig._routeRetry || !config.baseURL || !pool.origins.includes(new URL(config.baseURL).origin)) {
      config.baseURL = domainPool.currentOrigin() + "/api/v1/chat";
    }
    requestConfig._poolRevision = pool.revision;
  }
  const remaining = requestConfig._deadline - Date.now();
  if (remaining <= 0) throw new ApiError("请求超时，请检查网络后重试", { code: "ETIMEDOUT" });
  // Share one deadline across bootstrap, route retry and the existing auth refresh.
  config.timeout = config.method?.toLowerCase() === "get" && !requestConfig._routeRetry
    ? Math.max(1, Math.floor(remaining / 2)) : remaining;
  tracePhase('route_wait', routeStarted, { requestId: requestConfig._traceID, origin: config.baseURL });
  requestConfig._sentAt = Date.now();
  requestConfig._authToken = authToken;
  if (authToken) {
    config.headers = { ...config.headers, Authorization: `Bearer ${authToken}` };
  } else if (config.headers) {
    delete config.headers.Authorization;
  }
  return config;
});

chatClient.interceptors.response.use(
  (response) => {
    const diagnostic = response.config as any;
    const serverID = response.headers?.['x-request-id'];
    tracePhase('response', diagnostic._sentAt || Date.now(), { requestId: diagnostic._traceID, serverRequestId: serverID,
      origin: diagnostic.baseURL, status: response.status, code: response.data?.code });
    if ((response.config as any)._authEpoch !== authEpoch) {
      return Promise.reject(new ApiError("登录账号已切换，请重新操作", { code: "SESSION_CHANGED" }));
    }
    if (DYNAMIC_API_ROUTING && [0, 200].includes(response.data?.code)) {
      const config = response.config as any;
      domainPool.remember(new URL(config.baseURL).origin, config._poolRevision);
    }
    return response;
  },
  async (error) => {
    const requestConfig = error.config as any;
    const serverID = error.response?.headers?.['x-request-id'];
    if (requestConfig) tracePhase(error.response ? 'response' : 'browser_network', requestConfig._sentAt || Date.now(),
      { requestId: requestConfig._traceID, serverRequestId: serverID, origin: requestConfig.baseURL, status: error.response?.status, code: error.response?.data?.code });
    if (requestConfig && requestConfig._authEpoch !== authEpoch) return Promise.reject(error);

    if (DYNAMIC_API_ROUTING && axios.isAxiosError(error) && requestConfig &&
        (requestConfig.method?.toLowerCase() !== "get" || requestConfig._routeRetry) &&
        retryableRouteFailure(error.response?.status, error.response?.data,
          axios.isCancel(error) || Boolean(requestConfig.signal?.aborted))) {
      void domainPool.recover(new URL(requestConfig.baseURL).origin);
    }

    if (DYNAMIC_API_ROUTING && axios.isAxiosError(error) && requestConfig &&
        requestConfig.method?.toLowerCase() === "get" && !requestConfig._routeRetry &&
        retryableRouteFailure(error.response?.status, error.response?.data,
          axios.isCancel(error) || Boolean(requestConfig.signal?.aborted)) && Date.now() < requestConfig._deadline) {
      const failed = new URL(requestConfig.baseURL).origin;
      const current = domainPool.currentOrigin();
      const next = current !== failed ? current : domainPool.alternative(failed);
      if (next) {
        requestConfig._routeRetry = true;
        requestConfig.baseURL = next + "/api/v1/chat";
        return chatClient.request(requestConfig);
      }
    }

    if (
      axios.isAxiosError(error) && error.response?.status === 401 &&
      shouldHandleUnauthorized(error.config?.url) &&
      !requestConfig?._skipUnauthorizedHandler && !requestConfig?._retry && unauthorizedHandler
    ) {
      const requestEpoch = authEpoch;
      // A previous request may already have refreshed this exact session.
      let recovered = Boolean(authToken && requestConfig?._authToken !== authToken);
      if (!recovered) {
        if (!unauthorizedHandlingPromise || unauthorizedHandlingPromise.epoch !== requestEpoch) {
          const operation = { epoch: requestEpoch, promise: Promise.resolve(false) };
          operation.promise = Promise.resolve().then(() => unauthorizedHandler!()).then(result => {
            if (unauthorizedHandlingPromise === operation) unauthorizedHandlingPromise = null;
            return result;
          }, error => {
            if (unauthorizedHandlingPromise === operation) unauthorizedHandlingPromise = null;
            throw error;
          });
          unauthorizedHandlingPromise = operation;
        }
        recovered = await unauthorizedHandlingPromise.promise;
      }
      if (recovered && requestConfig && authToken && requestEpoch === authEpoch) {
        requestConfig._retry = true;
        return chatClient.request(requestConfig);
      }
      // The session owner clears credentials only after a definitive refresh
      // rejection. A timeout/rate limit must not silently remove the API token.
    }
    return Promise.reject(error);
  }
);

export function setChatAuthToken(token: string | null, context: string | null = null) {
  if (context !== authContext || (!token && authToken)) {
    authEpoch++;
    authContext = context;
    unauthorizedHandlingPromise = null;
  }
  authToken = token;
  profileReads.clear();
  conversationPinReads.clear();
  userLastOnlineCache.clear();
  groupMessagePinCache.clear();
  groupAdminsCache.clear();
  memberFilterReads.clear();
}

export function setChatUnauthorizedHandler(
  handler: UnauthorizedHandler | null
) {
  unauthorizedHandler = handler;
}

function shouldHandleUnauthorized(url?: string) {
  if (!url) {
    return false;
  }

  const ignoredPaths = [
    "/login",
    "/web-login",
    "/register",
    "/sms/send",
    "/password/reset",
    "/auth/refresh",
    "/totp/verify",
    "/totp/setup-verify",
    "/logout",
    "/login-challenges/",
  ];
  return !ignoredPaths.some((path) => url.includes(path));
}

function unwrapResponse<T>(response: ApiResponse<T>) {
  return response.data;
}

async function forwardRequest(method: "get" | "post" | "put", path: string, payload?: unknown) {
  try {
    const { data } = await chatClient.request<ApiResponse<any>>({method, url:path, ...(method === "get" ? {params:payload} : {data:payload})});
    return unwrapResponse(data);
  } catch (error) { throw toApiError(error); }
}
const forwardPath = (id: string) => `/forward-jobs/${encodeURIComponent(id)}`;
export const initForwardJob = (payload: unknown) => forwardRequest("post", "/forward-jobs/init", payload);
export const putForwardTargets = (id: string, payload: unknown) => forwardRequest("put", `${forwardPath(id)}/targets`, payload);
export const putForwardMessages = (id: string, payload: unknown) => forwardRequest("put", `${forwardPath(id)}/messages`, payload);
export const commitForwardJob = (id: string) => forwardRequest("post", `${forwardPath(id)}/commit`);
export const readForwardStatus = (id: string) => forwardRequest("get", `${forwardPath(id)}/status`);
export const readForwardJob = (id: string, cursor = "0") => forwardRequest("get", forwardPath(id), {paged:1, cursor});
export const retryForwardJob = (id: string, retryID: string) => forwardRequest("post", `${forwardPath(id)}/retry`, {retry_id:retryID});
export const cancelForwardJob = (id: string) => forwardRequest("post", `${forwardPath(id)}/cancel`);
export const listForwardJobs = (cursor = "") => forwardRequest("get", "/forward-jobs", { cursor });
export const readForwardSettings = () => forwardRequest("get", "/forward-settings");

function toApiError(error: unknown) {
  if (axios.isAxiosError(error)) {
    const details = apiErrorDetails(error);
    if (!error.response) {
      const timedOut = error.code === "ECONNABORTED" || error.code === "ETIMEDOUT" || error.message?.toLowerCase().includes("timeout");
      return new ApiError(timedOut ? "请求超时，请检查网络后重试" : "网络连接异常，请检查网络或稍后重试", details);
    }
    return new ApiError((error.response.data as ApiResponse<unknown> | undefined)?.message ||
      (error.response.data as ApiResponse<unknown> | undefined)?.msg || "请求失败", details);
  }
  return error instanceof Error ? error : new ApiError("请求失败");
}

export async function sendSMS(
  phone: string,
  scene: "register" | "reset_password" | "general" = "general"
) {
  try {
    await chatClient.post("/sms/send", {
      phone,
      scene,
      device_id: getOrCreateDeviceID(),
    });
  } catch (error) {
    throw toApiError(error);
  }
}

export async function login(payload: LoginPayload) {
  try {
    const deviceID = payload.device_id || getOrCreateDeviceID();
    const { data } = await chatClient.post<ApiResponse<ChatAuthResult>>(
      "/web-login",
      {
        ...payload,
        device_id: deviceID,
        device_name: payload.device_name || guessWebDeviceName(),
        platform: "web",
      }
    );
    return unwrapResponse(data);
  } catch (error) {
    throw toApiError(error);
  }
}

function guessWebDeviceName() {
  if (typeof navigator === "undefined") return "Web";
  const ua = navigator.userAgent || "";
  if (/Edg\//i.test(ua)) return "Edge Web";
  if (/Chrome\//i.test(ua) && !/Edg\//i.test(ua)) return "Chrome Web";
  if (/Safari\//i.test(ua) && !/Chrome\//i.test(ua)) return "Safari Web";
  if (/Firefox\//i.test(ua)) return "Firefox Web";
  return "Web 浏览器";
}

export interface UserTOTPStatus {
  enabled: boolean;
  bound_at: string;
}

export interface UserTOTPSetupResult {
  temp_token: string;
  totp_secret: string;
  totp_uri: string;
}

export interface UserTOTPBindResult {
  enabled: boolean;
}

export async function verifyLoginTOTP(tempToken: string, code: string) {
  try {
    const deviceID = getOrCreateDeviceID();
    const { data } = await chatClient.post<ApiResponse<ChatAuthResult>>(
      "/totp/verify",
      {
        temp_token: tempToken,
        code,
        device_id: deviceID,
        platform: "web",
      }
    );
    return unwrapResponse(data);
  } catch (error) {
    throw toApiError(error);
  }
}

export async function setupVerifyLoginTOTP(tempToken: string, code: string) {
  try {
    const deviceID = getOrCreateDeviceID();
    const { data } = await chatClient.post<ApiResponse<ChatAuthResult>>(
      "/totp/setup-verify",
      {
        temp_token: tempToken,
        code,
        device_id: deviceID,
        platform: "web",
      }
    );
    return unwrapResponse(data);
  } catch (error) {
    throw toApiError(error);
  }
}

export async function getUserTOTPStatus() {
  try {
    const { data } = await chatClient.get<ApiResponse<UserTOTPStatus>>(
      "/totp/status"
    );
    return unwrapResponse(data);
  } catch (error) {
    throw toApiError(error);
  }
}

export async function setupUserTOTP() {
  try {
    const { data } = await chatClient.post<ApiResponse<UserTOTPSetupResult>>(
      "/totp/setup"
    );
    return unwrapResponse(data);
  } catch (error) {
    throw toApiError(error);
  }
}

export async function bindUserTOTP(tempToken: string, code: string) {
  try {
    const { data } = await chatClient.post<ApiResponse<UserTOTPBindResult>>(
      "/totp/bind",
      {
        temp_token: tempToken,
        code,
      }
    );
    return unwrapResponse(data);
  } catch (error) {
    throw toApiError(error);
  }
}

export async function register(payload: RegisterPayload) {
  try {
    const deviceID = payload.device_id || getOrCreateDeviceID();
    const { data } = await chatClient.post<ApiResponse<ChatAuthResult>>(
      "/register",
      {
        phone: payload.phone,
        code: payload.code,
        nickname: payload.nickname,
        password: payload.password,
        invite_code: payload.inviteCode,
        device_id: deviceID,
      }
    );
    return unwrapResponse(data);
  } catch (error) {
    throw toApiError(error);
  }
}

export async function refreshSession(payload: {
  refreshToken: string;
  needUserSigRefresh?: boolean;
  userSigExpiresAt?: string;
  sdkAppID?: number;
}) {
  try {
    const deviceID = getOrCreateDeviceID();
    const { data } = await chatClient.post<ApiResponse<ChatAuthResult>>(
      "/auth/refresh",
      {
        refresh_token: payload.refreshToken,
        need_user_sig_refresh: payload.needUserSigRefresh ?? false,
        user_sig_expires_at: payload.userSigExpiresAt,
        sdk_app_id: payload.sdkAppID,
        device_id: deviceID,
      },
      { timeout: 8_000 }
    );
    return unwrapResponse(data);
  } catch (error) {
    throw toApiError(error);
  }
}

export async function logout(refreshToken?: string) {
  try {
    await chatClient.post("/logout", {
      refresh_token: refreshToken,
    });
  } catch (error) {
    throw toApiError(error);
  }
}

export async function getGroupMessagePin(groupID: string) {
  try {
    return await groupMessagePinCache.get(groupID, async () => {
      const { data } = await chatClient.get<ApiResponse<GroupMessagePin | null>>(
        "/groups/pin-message",
        { params: { group_id: groupID } }
      );
      return unwrapResponse(data);
    });
  } catch (error) {
    throw toApiError(error);
  }
}

export async function saveGroupMessagePin(payload: {
  groupID: string;
  messageID: string;
  messageSeq?: number;
  messageSenderIMUserID?: string;
  messageSenderName?: string;
  messageType?: string;
  messageAbstract?: string;
}) {
  groupMessagePinCache.invalidate(payload.groupID);
  try {
    const { data } = await chatClient.post<ApiResponse<GroupMessagePin>>(
      "/groups/pin-message",
      {
        group_id: payload.groupID,
        message_id: payload.messageID,
        message_seq: payload.messageSeq ?? 0,
        message_sender_im_user_id: payload.messageSenderIMUserID ?? "",
        message_sender_name: payload.messageSenderName ?? "",
        message_type: payload.messageType ?? "",
        message_abstract: payload.messageAbstract ?? "",
      }
    );
    const result = unwrapResponse(data);
    groupMessagePinCache.invalidate(payload.groupID);
    return result;
  } catch (error) {
    throw toApiError(error);
  }
}

export async function removeGroupMessagePin(groupID: string) {
  groupMessagePinCache.invalidate(groupID);
  try {
    await chatClient.post("/groups/pin-message/remove", {
      group_id: groupID,
    });
    groupMessagePinCache.invalidate(groupID);
  } catch (error) {
    throw toApiError(error);
  }
}

export async function revokeGroupMessage(groupID: string, msgSeq: number) {
  try {
    await chatClient.post("/messages/group/revoke", {
      group_id: groupID,
      msg_seq: msgSeq,
    });
  } catch (error) {
    throw toApiError(error);
  }
}

export async function getGroupAdmins(groupID: string) {
  try {
    return await groupAdminsCache.get(groupID, async () => {
      const { data } = await chatClient.get<ApiResponse<GroupAdmins>>(
        `/groups/${encodeURIComponent(groupID)}/admins`
      );
      return unwrapResponse(data);
    });
  } catch (error) {
    throw toApiError(error);
  }
}

export async function addGroupAdmin(groupID: string, userID: string) {
  groupAdminsCache.invalidate(groupID);
  try {
    const { data } = await chatClient.post<ApiResponse<GroupAdminMember>>(
      `/groups/${encodeURIComponent(groupID)}/admins`,
      { user_id: userID }
    );
    const result = unwrapResponse(data);
    groupAdminsCache.invalidate(groupID);
    return result;
  } catch (error) {
    throw toApiError(error);
  }
}

export async function removeGroupAdmin(groupID: string, userID: string) {
  groupAdminsCache.invalidate(groupID);
  try {
    await chatClient.delete(
      `/groups/${encodeURIComponent(groupID)}/admins/${encodeURIComponent(userID)}`
    );
    groupAdminsCache.invalidate(groupID);
  } catch (error) {
    throw toApiError(error);
  }
}

export async function getGroupSlowMode(groupID: string) {
  try {
    const { data } = await chatClient.get<ApiResponse<GroupSlowMode>>(
      `/groups/${encodeURIComponent(groupID)}/slow-mode`
    );
    return unwrapResponse(data);
  } catch (error) {
    throw toApiError(error);
  }
}

export async function saveGroupSlowMode(
  groupID: string,
  payload: { enabled: boolean; seconds: number }
) {
  try {
    const { data } = await chatClient.post<ApiResponse<GroupSlowMode>>(
      `/groups/${encodeURIComponent(groupID)}/slow-mode`,
      payload
    );
    return unwrapResponse(data);
  } catch (error) {
    throw toApiError(error);
  }
}

export type AddGroupMembersResult = {
  added: string[];
  already_members: string[];
  failed: Array<{
    account: string;
    result: number;
    reason?: string;
  }>;
};

export type FilterGroupMembersResult = {
  in_group: string[];
  not_in_group: string[];
};

export async function filterGroupMembers(groupID: string, userIDs: string[]) {
  const ids = [...new Set(userIDs)].sort();
  try {
    return await memberFilterReads.get(JSON.stringify([groupID, ids]), async () => {
      const { data } = await chatClient.post<ApiResponse<FilterGroupMembersResult>>(
        `/groups/${encodeURIComponent(groupID)}/members/filter`, { user_ids: ids });
      return unwrapResponse(data);
    });
  } catch (error) { throw toApiError(error); }
}

export async function addGroupMembers(groupID: string, userIDs: string[]) {
  memberFilterReads.clear();
  try {
    const { data } = await chatClient.post<ApiResponse<AddGroupMembersResult>>(
      `/groups/${encodeURIComponent(groupID)}/members`,
      {
        user_ids: userIDs,
      }
    );
    return unwrapResponse(data);
  } catch (error) {
    throw toApiError(error);
  } finally { memberFilterReads.clear(); }
}

export async function removeGroupMember(groupID: string, userID: string) {
  memberFilterReads.clear();
  try {
    await chatClient.delete(
      `/groups/${encodeURIComponent(groupID)}/members/${encodeURIComponent(userID)}`
    );
  } catch (error) {
    throw toApiError(error);
  } finally { memberFilterReads.clear(); }
}

export async function muteGroupMember(groupID: string, userID: string) {
  try {
    await chatClient.post(
      `/groups/${encodeURIComponent(groupID)}/members/${encodeURIComponent(userID)}/mute`
    );
  } catch (error) {
    throw toApiError(error);
  }
}

export async function unmuteGroupMember(groupID: string, userID: string) {
  try {
    await chatClient.delete(
      `/groups/${encodeURIComponent(groupID)}/members/${encodeURIComponent(userID)}/mute`
    );
  } catch (error) {
    throw toApiError(error);
  }
}

export async function getProfile(options?: { fresh?: boolean }) {
  try {
    if (options?.fresh) profileReads.invalidate('self');
    return await profileReads.get("self", async () => {
      const { data } = await chatClient.get<ApiResponse<ChatProfile>>("/profile");
      return unwrapResponse(data);
    });
  } catch (error) {
    throw toApiError(error);
  }
}

export async function updateProfile(payload: UpdateProfilePayload) {
  const requestBody: Record<string, unknown> = {};

  if (payload.wananID !== undefined) {
    requestBody.wanan_id = payload.wananID;
  }
  if (payload.nickname !== undefined) {
    requestBody.nickname = payload.nickname;
  }
  if (payload.avatarURL !== undefined) {
    requestBody.avatar_url = payload.avatarURL;
  }
  if (payload.selfSignature !== undefined) {
    requestBody.self_signature = payload.selfSignature;
  }
  if (payload.gender !== undefined) {
    requestBody.gender = payload.gender;
  }
  if (payload.birthday !== undefined) {
    requestBody.birthday = payload.birthday;
  }
  if (payload.allowType !== undefined) {
    requestBody.allow_type = payload.allowType;
  }

  profileReads.invalidate("self");
  try {
    const { data } = await chatClient.post<ApiResponse<ChatProfile>>(
      "/profile/update",
      requestBody
    );
    return unwrapResponse(data);
  } catch (error) {
    throw toApiError(error);
  } finally {
    profileReads.invalidate("self");
  }
}

export async function getConversationCustomPins() {
  try {
    return await conversationPinReads.get("self", async () => {
      const { data } = await chatClient.get<ApiResponse<ConversationCustomPin[]>>(
        "/conversation/pins"
      );
      return unwrapResponse(data);
    });
  } catch (error) {
    throw toApiError(error);
  }
}

export async function getUserLastOnline(imUserIDs: string[]) {
  try {
    const normalizedIDs = normalizeIMUserIDList(imUserIDs);
    if (!normalizedIDs.length) {
      return [];
    }

    await queueUserLastOnlineFetch(normalizedIDs);

    return normalizedIDs
      .map((id) => userLastOnlineCache.get(id) || null)
      .filter((item): item is UserLastOnline => Boolean(item));
  } catch (error) {
    throw toApiError(error);
  }
}

export async function getUserLoginIP(imUserID: string) {
  try {
    const { data } = await chatClient.get<ApiResponse<UserLoginIPResult>>(
      "/users/login-ip",
      {
        params: { im_user_id: imUserID },
      }
    );
    return unwrapResponse(data);
  } catch (error) {
    throw toApiError(error);
  }
}

function queueUserLastOnlineFetch(imUserIDs: string[]) {
  imUserIDs.forEach((id) => pendingUserLastOnlineIDs.add(id));

  return new Promise<void>((resolve, reject) => {
    pendingUserLastOnlineRequests.push({ resolve, reject });

    if (!pendingUserLastOnlineTimer) {
      pendingUserLastOnlineTimer = setTimeout(() => {
        pendingUserLastOnlineTimer = null;
        const ids = Array.from(pendingUserLastOnlineIDs);
        const requests = pendingUserLastOnlineRequests.splice(0);
        pendingUserLastOnlineIDs.clear();
        fetchAndCacheUserLastOnline(ids)
          .then(() => {
            requests.forEach((request) => request.resolve());
          })
          .catch((error) => {
            requests.forEach((request) => request.reject(error));
          });
      }, USER_LAST_ONLINE_BATCH_DELAY_MS);
    }
  });
}

async function fetchAndCacheUserLastOnline(normalizedIDs: string[]) {
  const items = await fetchUserLastOnline(normalizedIDs);
  const itemMap = new Map(items.map((item) => [item.im_user_id, item]));

  normalizedIDs.forEach((id) => {
    userLastOnlineCache.set(id, itemMap.get(id) || { im_user_id: id });
  });
}

async function fetchUserLastOnline(normalizedIDs: string[]) {
  try {
    const results: UserLastOnline[] = [];
    for (const batch of chunkList(normalizedIDs, USER_LAST_ONLINE_BATCH_SIZE)) {
      const { data } = await chatClient.post<
        ApiResponse<{ user_last_online: UserLastOnline[] }>
      >("/users/last-online", {
        im_user_ids: batch,
      });
      results.push(...(unwrapResponse(data).user_last_online || []));
    }
    return results;
  } catch (error) {
    throw toApiError(error);
  }
}

function normalizeIMUserIDList(values: string[]) {
  return Array.from(
    new Set(values.map((value) => String(value || "").trim()).filter(Boolean))
  );
}

function chunkList<T>(values: T[], size: number) {
  const chunks: T[][] = [];
  for (let index = 0; index < values.length; index += size) {
    chunks.push(values.slice(index, index + size));
  }
  return chunks;
}

export async function pinConversationCustom(payload: {
  conversationID: string;
  pinOrder?: number;
}) {
  conversationPinReads.invalidate("self");
  try {
    await chatClient.post("/conversation/pin", payload);
  } catch (error) {
    throw toApiError(error);
  } finally {
    conversationPinReads.invalidate("self");
  }
}

export async function unpinConversationCustom(payload: {
  conversationID: string;
}) {
  conversationPinReads.invalidate("self");
  try {
    await chatClient.post("/conversation/unpin", payload);
  } catch (error) {
    throw toApiError(error);
  } finally {
    conversationPinReads.invalidate("self");
  }
}

export async function getGroupNotice(groupID: string) {
  try {
    const { data } = await chatClient.get<ApiResponse<GroupNotice>>(
      "/groups/notice",
      {
        params: { group_id: groupID },
      }
    );
    return unwrapResponse(data);
  } catch (error) {
    throw toApiError(error);
  }
}

export async function pinGroupNotice(groupID: string, pinned: boolean) {
  try {
    const { data } = await chatClient.post<ApiResponse<GroupNotice>>("/groups/notice/pin", { group_id: groupID, pinned });
    return unwrapResponse(data);
  } catch (error) { throw toApiError(error); }
}

export async function saveGroupNotice(groupID: string, content: string, pinned?: boolean) {
  try {
    const { data } = await chatClient.post<ApiResponse<GroupNotice>>(
      `/groups/notice`,
      {
        group_id: groupID,
        content,
        ...(pinned === undefined ? {} : { pinned }),
      }
    );
    return unwrapResponse(data);
  } catch (error) {
    throw toApiError(error);
  }
}

export async function changePassword(newPassword: string) {
  try {
    const { data } = await chatClient.post<ApiResponse<ChatAuthResult>>(
      "/password/change",
      {
        new_password: newPassword,
        device_id: getOrCreateDeviceID(),
        device_name: guessWebDeviceName(),
        platform: "web",
      }
    );
    return unwrapResponse(data);
  } catch (error) {
    throw toApiError(error);
  }
}

export async function resetPassword(
  phone: string,
  code: string,
  newPassword: string
) {
  try {
    await chatClient.post("/password/reset", {
      phone,
      code,
      new_password: newPassword,
    });
  } catch (error) {
    throw toApiError(error);
  }
}

export async function updatePassword(
  currentPassword: string,
  newPassword: string
) {
  try {
    await chatClient.post("/password/update", {
      current_password: currentPassword,
      new_password: newPassword,
    });
  } catch (error) {
    throw toApiError(error);
  }
}

export async function bindInviteCode(inviteCode: string) {
  try {
    const { data } = await chatClient.post<ApiResponse<ChatProfile>>(
      "/invite-code/bind",
      {
        invite_code: inviteCode,
      }
    );
    return unwrapResponse(data);
  } catch (error) {
    throw toApiError(error);
  }
}

export async function deleteAccount() {
  try {
    await chatClient.post("/account/delete");
  } catch (error) {
    throw toApiError(error);
  }
}

export async function getAvatarPresignUrl() {
  try {
    const { data } = await chatClient.get<ApiResponse<AvatarPresignResult>>(
      "/upload/avatar/presign"
    );
    return unwrapResponse(data);
  } catch (error) {
    throw toApiError(error);
  }
}

export async function getGroupAvatarPresignUrl(groupID: string) {
  try {
    const { data } = await chatClient.get<ApiResponse<AvatarPresignResult>>(
      "/upload/avatar/presign",
      {
        params: {
          type: "group",
          group_id: groupID,
        },
      }
    );
    return unwrapResponse(data);
  } catch (error) {
    throw toApiError(error);
  }
}

export async function getChatMediaPresignUrl(file: File, mediaType = "video") {
  try {
    const { data } = await chatClient.get<ApiResponse<ChatMediaPresignResult>>(
      "/upload/media/presign",
      {
        params: {
          media_type: mediaType,
          file_name: file.name,
          file_size: file.size,
          content_type: file.type,
        },
      }
    );
    return unwrapResponse(data);
  } catch (error) {
    throw toApiError(error);
  }
}

function normalizeUploadProgress(progress: number) {
  if (!Number.isFinite(progress)) {
    return 0;
  }
  return Math.max(0, Math.min(1, progress));
}

function normalizeUploadRequestOptions(
  options?: UploadProgressCallback | UploadRequestOptions
): UploadRequestOptions {
  if (typeof options === "function") {
    return {
      onProgress: options,
    };
  }
  return options || {};
}

function isAxiosRequestCanceled(error: unknown) {
  return (
    axios.isCancel(error) ||
    (axios.isAxiosError(error) &&
      ((error as any).code === "ERR_CANCELED" ||
        error.message === "canceled" ||
        error.message === "Cancel"))
  );
}

function delay(ms: number) {
  return new Promise<void>((resolve) => {
    window.setTimeout(resolve, ms);
  });
}

function isRetriableAttachmentCompleteError(error: unknown) {
  const message = error instanceof Error ? error.message : String(error || "");
  return message.includes("文件尚未上传完成");
}

export async function putFileToCos(
  putUrl: string,
  file: File,
  options?: UploadProgressCallback | UploadRequestOptions
) {
  const uploadOptions = normalizeUploadRequestOptions(options);
  let uploadedProgress = 0;
  try {
    await axios.put(putUrl, file, {
      headers: {
        "Content-Type":
          uploadOptions.contentType || file.type || "application/octet-stream",
      },
      signal: uploadOptions.signal,
      onUploadProgress: (progressEvent) => {
        const total = progressEvent.total || file.size;
        if (!total) {
          return;
        }
        uploadedProgress = normalizeUploadProgress(
          (progressEvent.loaded || 0) / total
        );
        uploadOptions.onProgress?.(uploadedProgress);
      },
    });
    uploadOptions.onProgress?.(1);
  } catch (error) {
    if (isAxiosRequestCanceled(error)) {
      throw new UploadCanceledError();
    }
    if (axios.isAxiosError(error) && !error.response) {
      throw new DirectCosUploadError(uploadedProgress);
    }
    throw toApiError(error);
  }
}

export async function uploadChatVideo(file: File) {
  const presign = await getChatMediaPresignUrl(file, "video");
  await putFileToCos(presign.put_url, file);
  return presign.file_url;
}

export async function getChatAttachmentUploadSettings() {
  try {
    const { data } = await chatClient.get<
      ApiResponse<ChatAttachmentUploadSettings>
    >("/upload/attachment/settings");
    return unwrapResponse(data);
  } catch (error) {
    throw toApiError(error);
  }
}

function normalizeChatAttachmentConversationID(conversationID: string) {
  const raw = conversationID.trim();
  if (raw.startsWith("c2c_") || raw.startsWith("group_")) {
    return raw;
  }
  if (raw.startsWith("C2C")) {
    return `c2c_${raw.slice(3)}`;
  }
  if (raw.startsWith("GROUP")) {
    return `group_${raw.slice(5)}`;
  }
  return raw;
}

export async function getChatAttachmentPresignUrl(
  file: File,
  conversationID: string,
  options?: Pick<UploadRequestOptions, "isVideo" | "contentType">
) {
  const normalizedConversationID =
    normalizeChatAttachmentConversationID(conversationID);
  const isVideo = options?.isVideo === true;
  const contentType =
    options?.contentType ||
    (isVideo &&
    !String(file.type || "")
      .toLowerCase()
      .startsWith("video/")
      ? "video/mp4"
      : file.type || "application/octet-stream");
  try {
    const { data } = await chatClient.post<
      ApiResponse<ChatAttachmentPresignResult>
    >("/upload/attachment/presign", {
      conversation_id: normalizedConversationID,
      file_name: file.name,
      file_size: file.size,
      content_type: contentType,
      is_video: isVideo,
    });
    return unwrapResponse(data);
  } catch (error) {
    throw toApiError(error);
  }
}

export async function completeChatAttachmentUpload(
  fileID: string,
  maxAttempts = 1
) {
  for (let attempt = 1; attempt <= maxAttempts; attempt += 1) {
    try {
      await chatClient.post<ApiResponse<null>>("/upload/attachment/complete", {
        file_id: fileID,
      });
      return;
    } catch (error) {
      const normalizedError = toApiError(error);
      if (
        attempt >= maxAttempts ||
        !isRetriableAttachmentCompleteError(normalizedError)
      ) {
        throw normalizedError;
      }
      await delay(250 * attempt);
    }
  }
}

export async function getChatAttachmentAccessUrl(
  fileID: string,
  options?: { download?: boolean }
) {
  const normalizedFileID = fileID.trim();
  if (!normalizedFileID) {
    throw new Error("文件信息不完整");
  }
  try {
    const { data } = await chatClient.get<
      ApiResponse<ChatAttachmentAccessResult>
    >(`/files/${encodeURIComponent(normalizedFileID)}/access`, {
      params: options?.download ? { download: "1" } : undefined,
    });
    return unwrapResponse(data);
  } catch (error) {
    throw toApiError(error);
  }
}

export async function uploadChatAttachment(
  file: File,
  conversationID: string,
  options?: UploadProgressCallback | UploadRequestOptions
) {
  const uploadOptions = normalizeUploadRequestOptions(options);
  const epoch = authEpoch;
  const owner = authContext || '';
  return uploadMultipart(file, normalizeChatAttachmentConversationID(conversationID), owner,
    (method, path, body) => forwardRequest(method, path, body), uploadOptions,
    () => authEpoch === epoch && authContext === owner && !!authToken);

}

export async function uploadAvatar(userID: string, file: File, signal?: AbortSignal) {
  const epoch = authEpoch;
  file = await prepareAvatar(file, signal);
  if (signal?.aborted || epoch !== authEpoch) throw new UploadCanceledError();
  try {
    const formData = new FormData();
    formData.append("file", file);
    formData.append("user_id", userID);

    const { data } = await chatClient.post<ApiResponse<{ avatar_url: string }>>(
      "/upload/avatar",
      formData,
      { signal, timeout: 30000 }
    );
    return unwrapResponse(data).avatar_url;
  } catch (error) {
    throw toApiError(error);
  }
}

export async function uploadGroupAvatar(groupID: string, file: File, signal?: AbortSignal) {
  const epoch = authEpoch;
  file = await prepareAvatar(file, signal);
  if (signal?.aborted || epoch !== authEpoch) throw new UploadCanceledError();
  try {
    const formData = new FormData();
    formData.append("file", file);
    formData.append("type", "group");
    formData.append("group_id", groupID);

    const { data } = await chatClient.post<ApiResponse<{ avatar_url: string }>>(
      "/upload/avatar",
      formData,
      { signal, timeout: 30000 }
    );
    return unwrapResponse(data).avatar_url;
  } catch (error) {
    throw toApiError(error);
  }
}

export async function searchUsers(keyword: string) {
  try {
    const { data } = await chatClient.get<
      ApiResponse<{ users: SearchUserResult[] }>
    >("/users/search", {
      params: { q: keyword },
    });
    return unwrapResponse(data).users;
  } catch (error) {
    throw toApiError(error);
  }
}

export async function searchUsersExact(keyword: string) {
  try {
    const { data } = await chatClient.get<
      ApiResponse<{ users: SearchUserResult[] }>
    >("/users/search-exact", {
      params: { q: keyword },
    });
    return unwrapResponse(data).users;
  } catch (error) {
    throw toApiError(error);
  }
}

export async function precheckAddFriend(imUserId?: string) {
  try {
    const { data } = await chatClient.post<
      ApiResponse<{ can_add_friend: boolean; reason?: string }>
    >("/friends/precheck", imUserId ? { im_user_id: imUserId } : {});
    return unwrapResponse(data);
  } catch (error) {
    throw toApiError(error);
  }
}

export async function getUserBriefs(imUserIDs: string[]) {
  const ids = Array.from(
    new Set(
      (imUserIDs || [])
        .map((id) => String(id || "").trim())
        .filter((id) => id.length > 0)
    )
  ).slice(0, 50);
  if (ids.length === 0) {
    return [] as SearchUserResult[];
  }
  try {
    const { data } = await chatClient.post<
      ApiResponse<{ users: SearchUserResult[] }>
    >("/users/briefs", {
      im_user_ids: ids,
    });
    return unwrapResponse(data).users || [];
  } catch (error) {
    throw toApiError(error);
  }
}

export async function bindFriendOwnerFromMe(imUserID: string) {
  return (await syncFriendOwnerFromMe(imUserID)).user;
}

export async function syncFriendOwnerFromMe(imUserID: string, requestID?: string) {
  const targetID = String(imUserID || "").trim();
  if (!targetID) {
    throw new Error("用户不存在");
  }
  try {
    const { data } = await chatClient.post<
      ApiResponse<{ user: SearchUserResult; assignment?: import('@/features/auth/types').FriendOwnerAssignment }>
    >(`/users/${encodeURIComponent(targetID)}/bind-owner-from-me`, requestID ? { request_id: requestID } : undefined);
    profileReads.invalidate('self');
    return unwrapResponse(data);
  } catch (error) {
    throw toApiError(error);
  }
}

export async function searchGroupByID(groupID: string) {
  try {
    const { data } = await chatClient.get<
      ApiResponse<{ group: SearchGroupResult | null }>
    >("/groups/search", {
      params: { group_id: groupID },
    });
    return unwrapResponse(data).group;
  } catch (error) {
    throw toApiError(error);
  }
}

export async function searchGroups(keyword: string) {
  try {
    const { data } = await chatClient.get<
      ApiResponse<{ groups: SearchGroupResult[] }>
    >("/groups/search", {
      params: { q: keyword },
    });
    return unwrapResponse(data).groups;
  } catch (error) {
    throw toApiError(error);
  }
}

export async function submitReport(payload: SubmitReportPayload) {
  try {
    await chatClient.post("/reports", {
      target_id: payload.targetId,
      target_type: payload.targetType,
      target_name: payload.targetName,
      category: payload.category,
      description: payload.description,
    });
  } catch (error) {
    throw toApiError(error);
  }
}
