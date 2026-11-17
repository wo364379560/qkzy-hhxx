export interface ChatAuthResult {
  user_id: string;
  sdk_app_id?: number;
  user_sig?: string;
  user_sig_expires_at?: string;
  token?: string;
  token_expires_at?: string;
  refresh_token?: string;
  must_change_password: boolean;
  is_activated: boolean;
  require_totp_verify?: boolean;
  require_totp_setup?: boolean;
  temp_token?: string;
  totp_secret?: string;
  totp_uri?: string;

  need_device_approval?: boolean;
  challenge_id?: number;
  pre_auth_token?: string;
  challenge_expires_at?: string;
  challenge_short_code?: string;
  challenge_device_name?: string;
  challenge_platform?: string;
  challenge_ip?: string;
}

export interface ChatSessionRecord {
  sessionFamilyID?: string;
  credentialsRevision?: number;
  token: string;
  refreshToken: string;
  accessTokenExpiresAt: string;
  userID: string;
  sdkAppID: number;
  userSig: string;
  userSigExpiresAt: string;
  mustChangePassword: boolean;
  isActivated: boolean;
}

export interface LoginPayload {
  phone: string;
  password?: string;
  device_id?: string;
  device_name?: string;
  turnstile_token?: string;
}

export interface TOTPChallenge {
  type: "totp";
  tempToken: string;
}

export interface TOTPSetupChallenge {
  type: "totp_setup";
  tempToken: string;
  totpSecret: string;
  totpUri: string;
}

export type LoginFlowResult = string | TOTPChallenge | TOTPSetupChallenge;

export interface RegisterPayload {
  phone: string;
  code: string;
  nickname: string;
  password: string;
  inviteCode?: string;
  device_id?: string;
}

export interface UpdateProfilePayload {
  wananID?: string;
  nickname?: string;
  avatarURL?: string;
  selfSignature?: string;
  gender?: number;
  birthday?: number;
  allowType?: number;
}

export interface ChatProfile {
  im_user_id: string;
  phone: string;
  wanan_id: string;
  nickname: string;
  avatar_url: string;
  self_signature: string;
  gender: number;
  birthday: number;
  allow_type: number;
  can_create_group: boolean;
  invite_code: string;
  can_bind_invite_code: boolean;
  owner_id?: number;
  owner_name?: string;
  owner_revision?: number;
  is_assigned?: boolean;
  owner_state?: 'pool' | 'assigned' | 'unavailable';
  owner_role?: string;
  owner_reason_code?: string;
}

export interface FriendOwnerAssignment {
  state: 'migrated' | 'unchanged' | 'pending' | 'blocked' | 'conflict';
  reason_code?: string;
  request_id: string;
  caller?: Partial<ChatProfile>;
  target?: Partial<ChatProfile>;
  affected_groups?: number[];
}

export interface AvatarPresignResult {
  put_url: string;
  avatar_url: string;
}

export interface ChatMediaPresignResult {
  put_url: string;
  file_url: string;
  object_key: string;
  max_size_mb: number;
}

export interface ChatAttachmentUploadSettings {
  enabled: boolean;
  max_size_mb: number;
  hard_limit_mb: number;
  native_limit_mb: number;
  put_url_ttl_seconds: number;
  get_url_ttl_seconds: number;
  retention_days: number;
  updated_at?: string;
}

export interface ChatAttachmentPresignResult {
  file_id: string;
  file_name?: string;
  put_url: string;
  download_url: string;
  preview_url: string;
  object_key: string;
  content_type?: string;
  max_size_mb: number;
  native_limit_mb: number;
  is_video: boolean;
  expires_at?: string;
}

export interface ChatAttachmentAccessResult {
  url: string;
  file_id: string;
  file_name: string;
  file_size: number;
  content_type: string;
  is_video: boolean;
  expires_at?: string;
}

export interface SearchUserResult {
 owner_revision?: number;
 owner_state?: "pool" | "assigned" | "unavailable";
 is_assigned?: boolean;
  im_user_id: string;
  wanan_id?: string;
  phone?: string;
  nickname: string;
  avatar_url?: string;
  user_tag?: string;
  owner_id?: number;
  owner_name?: string;
  can_add_friend?: boolean;
  add_blocked_reason?: string;
}

export interface SearchGroupResult {
  group_id: string;
  group_name: string;
  avatar_url?: string;
  group_type?: string;
}

export interface SubmitReportPayload {
  targetId: string;
  targetType: "user" | "group";
  targetName: string;
  category: "spam" | "abuse" | "fraud" | "other";
  description?: string;
}
