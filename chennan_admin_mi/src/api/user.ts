import request from "./request";
import { ADMIN_API_BASE_URL } from "@/config/runtime";
import type {
  ApiResponse,
  AppUser,
  PageResult,
  UserOwnerOption,
} from "@/types";

export interface UserListParams {
  page?: number;
  pageSize?: number;
  keyword?: string;
  status?: string;
  currentLoginIP?: string;
  isActivated?: "true" | "false";
  ipWhitelistEnabled?: "true" | "false";
  ownerID?: number;
}

export interface ExportUsersParams {
  keyword?: string;
  status?: string;
  currentLoginIP?: string;
  isActivated?: boolean;
  ipWhitelistEnabled?: boolean;
  ownerID?: number;
}

export interface CreateUserParams {
  phone: string;
  password: string;
  nickname?: string;
  avatar_url?: string;
  owner_id?: number;
}

export interface BatchCreateResult {
  index: number;
  success: boolean;
  message: string;
}

export type GeneratedPasswordMode = "fixed" | "prefix_random" | "random_strong";
export type GeneratedPhoneMiddleMode = "sequence" | "random";

export interface GenerateUsersParams {
  count: number;
  owner_id?: number;
  /** 默认 true：批量生成账号直接标记已实名 */
  real_name_verified?: boolean;
  /** 默认 false：批量生成账号禁止创建群聊 */
  can_create_group?: boolean;
  phone_template: {
    prefix: string;
    middle_mode: GeneratedPhoneMiddleMode;
    middle_start: number;
    suffix: string;
  };
  nickname_template: {
    prefix: string;
    start: number;
    padding: number;
  };
  password_template: {
    mode: GeneratedPasswordMode;
    fixed_value?: string;
    prefix?: string;
    random_length?: number;
  };
}

export interface GenerateUsersResultItem {
  index: number;
  success: boolean;
  message: string;
  phone: string;
  nickname: string;
  password: string;
  im_user_id: string;
}

export interface GenerateUsersResult {
  requested: number;
  success: number;
  failed: number;
  items: GenerateUsersResultItem[];
}

export interface BatchUserActionDetail {
  userID: number;
  nickname: string;
  success: boolean;
  message: string;
}

export interface BatchUserActionResult {
  success: number;
  failed: number;
  details: BatchUserActionDetail[];
}

export function getUserList(params: UserListParams = {}) {
  return request.get<ApiResponse<PageResult<AppUser>>>("/users", { params });
}

export async function exportUsers(params: ExportUsersParams = {}) {
  const token = localStorage.getItem("admin_token") ?? "";
  const response = await fetch(`${ADMIN_API_BASE_URL}/users/export`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    body: JSON.stringify(params),
  });

  if (!response.ok) {
    let message = "导出失败";
    try {
      const data = await response.json();
      message = data?.message || message;
    } catch {
      // ignore parse failure
    }
    throw new Error(message);
  }

  return response.blob();
}

export function getUserOwners() {
  return request.get<ApiResponse<UserOwnerOption[]>>("/users/owners");
}

export function createUser(params: CreateUserParams) {
  return request.post<ApiResponse<AppUser>>("/users", params);
}

export function batchCreateUsers(users: CreateUserParams[]) {
  return request.post<ApiResponse<BatchCreateResult[]>>("/users/batch", {
    users,
  });
}

export function generateUsers(params: GenerateUsersParams) {
  return request.post<ApiResponse<GenerateUsersResult>>(
    "/users/generate",
    params,
    {
      timeout: 120000,
    },
  );
}

export function banUser(id: number) {
  return request.post<ApiResponse<null>>(`/users/${id}/ban`);
}

export function unbanUser(id: number) {
  return request.post<ApiResponse<null>>(`/users/${id}/unban`);
}

export function kickUser(id: number) {
  return request.post<ApiResponse<null>>(`/users/${id}/kick`);
}

export function resolveUserLoginCooldown(id: number) {
  return request.post<ApiResponse<null>>(`/users/${id}/login-cooldown/resolve`);
}

export function deleteUser(id: number) {
  return request.post<ApiResponse<null>>(`/users/${id}/delete`);
}

export function disableUserTOTP(id: number) {
  return request.post<ApiResponse<null>>(`/users/${id}/totp/disable`);
}

export function enableUserTOTP(id: number) {
  return request.post<ApiResponse<null>>(`/users/${id}/totp/enable`);
}

export function resetUserRealname(id: number) {
  return request.post<ApiResponse<null>>(`/users/${id}/realname/reset`);
}

export function verifyUserRealname(id: number, params: { real_name: string; id_number: string }) {
  return request.post<ApiResponse<null>>(`/users/${id}/realname/verify`, params);
}

export function setUserRealnameStatus(id: number, verified: boolean) {
  return request.post<ApiResponse<null>>(`/users/${id}/realname/status`, { verified });
}

export interface UserLoginDeviceItem {
  id: string;
  device_id: string;
  device_name: string;
  platform: string;
  is_web: boolean;
  ip: string;
  last_login_at: string;
  is_online: boolean;
  status: string;
}

export function listUserLoginDevices(id: number) {
  return request.get<ApiResponse<UserLoginDeviceItem[]>>(`/users/${id}/login-devices`);
}

export interface UserDeletePreview {
  userID: number;
  nickname: string;
  phone: string;
  imUserID: string;
  mode: "soft_delete" | string;
  willDeleteLocalData: boolean;
  willDeleteTencentIM: boolean;
  willKickOnline: boolean;
  operationDetail: string;
  communityPosts?: number;
  communityComments?: number;
  communityLikes?: number;
  habits?: number;
  habitCheckins?: number;
  challenges?: number;
  challengeMembers?: number;
  challengeGroups?: number;
  offlineRegistrations?: number;
  reports?: number;

  savingPosts?: number;
  savingComments?: number;
  savingLikes?: number;
  ledgers?: number;
  ledgerEntries?: number;
  splitRecords?: number;
  splitParticipants?: number;
  splitDiscussionGroups?: number;
  offlineEventRegistrations?: number;
}

export function getUserDeletePreview(id: number) {
  return request.get<ApiResponse<UserDeletePreview>>(
    `/users/${id}/delete-preview`,
  );
}

export function transferUser(id: number, targetAdminID: number) {
  return request.post<ApiResponse<null>>(`/users/${id}/transfer`, {
    target_admin_id: targetAdminID,
  });
}

export function batchBanUsers(userIDs: number[]) {
  return request.post<ApiResponse<BatchUserActionResult>>("/users/batch-ban", {
    userIDs,
  });
}

export function batchUnbanUsers(userIDs: number[]) {
  return request.post<ApiResponse<BatchUserActionResult>>(
    "/users/batch-unban",
    { userIDs },
  );
}

export function batchDeleteUsers(userIDs: number[]) {
  return request.post<ApiResponse<BatchUserActionResult>>(
    "/users/batch-delete",
    { userIDs },
  );
}

export function batchSetUserCreateGroupPermission(
  userIDs: number[],
  enabled: boolean,
) {
  return request.post<ApiResponse<BatchUserActionResult>>(
    "/users/batch-create-group-permission",
    {
      userIDs,
      enabled,
    },
  );
}

export function batchSetUserGlobalSearch(userIDs: number[], enabled: boolean) {
  return request.post<ApiResponse<BatchUserActionResult>>(
    "/users/batch-global-search",
    {
      userIDs,
      enabled,
    },
  );
}

export function batchTransferUsers(userIDs: number[], targetAdminID: number) {
  return request.post<ApiResponse<BatchUserActionResult>>(
    "/users/batch-transfer",
    {
      userIDs,
      target_admin_id: targetAdminID,
    },
  );
}

export function syncUsersPreview() {
  return request.post<ApiResponse<import("@/types").SyncUserPreview[]>>(
    "/users/sync/preview",
  );
}

export interface SyncUserConfirmItem {
  imUserID: string;
  action: Extract<import("@/types").SyncAction, "skip" | "update">;
}

export function syncUsersConfirm(selections: SyncUserConfirmItem[]) {
  return request.post<ApiResponse<{ success: number; failed: number }>>(
    "/users/sync/confirm",
    { selections },
  );
}

export interface UserDetail {
  id: number;
  im_user_id: string;
  wanan_id: string;
  user_tag: string;
  phone: string;
  current_login_ip: string;
  ip_whitelist_enabled: boolean;
  ip_whitelist_ip: string;
  nickname: string;
  avatar_url: string;
  self_signature: string;
  gender: number;
  birthday: string | null;
  owner_id: number;
  owner_name: string;
  owner_status: string;
  status: string;
  account_status: string;
  login_status: string;
  login_reason: string;
  cleanup_status: string;
  cleanup_reason: string;
  is_activated: boolean;
  can_create_group: boolean;
  globally_searchable: boolean;
  community_post_disabled: boolean;
  community_comment_disabled: boolean;
  saving_post_disabled?: boolean;
  saving_comment_disabled?: boolean;
  real_name_verified?: boolean;
  real_name?: string;
  id_number_masked?: string;
  created_at: string;
}

export interface UpdateUserParams {
  wanan_id?: string;
  nickname?: string;
  user_tag?: string;
  avatar_url?: string;
  self_signature?: string;
  gender?: number;
  birthday?: string;
  can_create_group?: boolean;
  globally_searchable?: boolean;
}

export interface SaveUserIPWhitelistParams {
  enabled: boolean;
  ip?: string;
}

export interface SaveUserIPWhitelistResult {
  enabled: boolean;
  ip: string;
  forced_logout: boolean;
}

export interface BatchUserIPWhitelistDetail {
  userID: number;
  nickname: string;
  success: boolean;
  message: string;
  forced_logout: boolean;
}

export interface BatchUserIPWhitelistResult {
  success: number;
  failed: number;
  forced_logout: number;
  details: BatchUserIPWhitelistDetail[];
}

export interface ResetPasswordParams {
  mode: "set" | "default";
  new_password?: string;
  must_change: boolean;
}

export interface UserStatusResult {
  im_user_id: string;
  status: string;
  platform: string;
  last_offline_at?: string;
}

export function getUserDetail(id: number) {
  return request.get<ApiResponse<UserDetail>>(`/users/${id}`);
}

export function updateUser(id: number, params: UpdateUserParams) {
  return request.post<ApiResponse<null>>(`/users/${id}/update`, params);
}

export function resetUserPassword(id: number, params: ResetPasswordParams) {
  return request.post<ApiResponse<null>>(`/users/${id}/reset-password`, params);
}

export function updateUserPhone(id: number, newPhone: string) {
  return request.post<ApiResponse<null>>(`/users/${id}/update-phone`, {
    new_phone: newPhone,
  });
}

export function saveUserIPWhitelist(id: number, params: SaveUserIPWhitelistParams) {
  return request.post<ApiResponse<SaveUserIPWhitelistResult>>(
    `/users/${id}/ip-whitelist`,
    params,
  );
}

export function batchSaveUserIPWhitelist(
  userIDs: number[],
  params: SaveUserIPWhitelistParams,
) {
  return request.post<ApiResponse<BatchUserIPWhitelistResult>>(
    "/users/batch-ip-whitelist",
    {
      userIDs,
      ...params,
    },
  );
}

export function getUserStatus(id: number) {
  return request.get<ApiResponse<UserStatusResult>>(`/users/${id}/status`);
}

export function batchGetUserStatus(imUserIDs: string[]) {
  return request.post<ApiResponse<UserStatusResult[]>>("/users/batch-status", {
    im_user_ids: imUserIDs,
  });
}

export const getIMApplication = () => request.get<ApiResponse<{ sdk_app_id: number }>>('/im/application')
export const repairUserIM = (id: number, sdkAppId: number) => request.post(`/users/${id}/im/repair`, { expected_sdk_app_id: sdkAppId })
