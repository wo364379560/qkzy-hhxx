export type Role = 'super_admin' | 'admin' | 'assistant'
export type UserStatus = 'active' | 'banned' | 'deleted'
export type InviteCodeStatus = 'active' | 'used' | 'expired'

export interface AdminUser {
  id: number
  username: string
  displayName: string
  role: Role
  status: string
  assistantCount?: number
  createdAt: string
  totpEnabled: boolean
  totpBoundAt: string | null
  ipWhitelistEnabled?: boolean
  loginCooldown?: boolean
  loginCooldownUntil?: string
  loginCooldownReason?: string
}

export interface AssistantDetail extends AdminUser {
  parentAdminID?: number | null
  parentAdminName?: string
  parentAdminRole?: Role | ''
  inviteCodeID: number | null
  inviteCode: string | null
  inviteCodeStatus: string | null
  inviteCodeUsed: number
  inviteCodeMax: number       // -1 = unlimited
  inviteCodeExpiredAt: string | null
  userCount: number
  totpEnabled: boolean
  totpBoundAt: string | null
  ipWhitelistEnabled: boolean
}

export interface AssistantIPWhitelistRule {
  id: number
  cidr: string
  note: string
}

export interface AssistantIPWhitelistDetail {
  assistantId: number
  enabled: boolean
  rules: AssistantIPWhitelistRule[]
}

export interface AdminUserIPWhitelistDetail {
  adminUserId: number
  enabled: boolean
  rules: AssistantIPWhitelistRule[]
}

export interface IPWhitelistOverviewItem {
  id: number
  username: string
  role: Role
  status: string
  managedCount: number
  ipWhitelistEnabled: boolean
  ruleCount: number
  rulePreview: string[]
  lastUpdatedAt: string | null
}

export type AssistantIPWhitelistOverview = IPWhitelistOverviewItem

export interface AdminAccessAudit {
  id: number
  adminUserId: number | null
  adminUsername: string
  adminRole: string
  accessType: 'system_user' | 'illegal_access'
  module: string
  actionLabel: string
  actionKey: string
  actionSummary: string
  actionDetail: string
  method: string
  route: string
  targetId: number | null
  targetType: string
  targetUsername: string
  targetDisplayName: string
  changeSet: string
  success: boolean
  statusCode: number
  ipAddress: string
  userAgent: string
  failureReason: string
  createdAt: string
}

export interface AdminLoginAudit {
  id: number
  adminUserId: number | null
  username: string
  adminRole: string
  stage: 'turnstile' | 'whitelist' | 'password' | 'totp_setup' | 'totp_verify'
  result: 'success' | 'failed' | 'require_totp_setup' | 'require_totp_verify'
  failureReason: string
  ipAddress: string
  userAgent: string
  createdAt: string
}

export interface ChatQuietHoursHitLog {
  id: number
  action: string
  source: string
  callbackCommand: string
  userId: number | null
  ownerId: number | null
  imUserId: string
  phone: string
  targetAccount: string
  targetGroupId: string
  ipAddress: string
  userAgent: string
  reason: string
  createdAt: string
}

export interface SMSSendLog {
  id: number
  phone: string
  ipAddress: string
  deviceID: string
  scene: string
  status: string
  rejectReason: string
  provider: string
  createdAt: string
}

export interface UserOperationLog {
  id: number
  userId: number | null
  imUserId: string
  phone: string
  nickname: string
  ownerId: number | null
  ownerName: string
  operationType: string
  operationName: string
  operationDetail: string
  ipAddress: string
  userAgent: string
  deviceId: string
  deviceName: string
  platform: string
  appVersion: string
  status: 'success' | 'failed' | string
  failureReason: string
  createdAt: string
}

export interface AppUser {
  id: number
  imUserID: string
  wananID: string
  phone: string
  currentLoginIP: string
  ipWhitelistEnabled: boolean
  ipWhitelistIP: string
  totpEnabled: boolean
  nickname: string
  avatarUrl: string
  gender: number        // 0=未知 1=男 2=女
  birthday: string      // "YYYY-MM-DD" 或 ""
  location: string
  selfSignature: string
  ownerRevision?: number
  ownerState?: string
  ownerID: number
  ownerName: string     // 归属人用户名
  ownerRole: string     // "admin" | "assistant"
  ownerStatus: string
  inviteCode: string    // 注册使用的邀请码，空字符串表示管理员建号或直接注册
  status: UserStatus
  accountStatus: UserStatus
  loginStatus: UserStatus
  loginReason: '' | 'owner_banned'
  loginCooldown: boolean
  loginCooldownUntil: string
  loginCooldownReason: string
  cleanupStatus: '' | 'auto_banned' | 'manual_banned' | 'whitelisted' | 'restored'
  cleanupReason: string
  isActivated: boolean
  canCreateGroup: boolean
  globallySearchable: boolean
  lastOnlineAt: string
  lastOfflineAt: string
  createdAt: string
  onlineStatus?: string // "online" | "offline"
  platform?: string     // 在线平台
  realNameVerified?: boolean
  realName?: string
  idNumberMasked?: string
}

export interface UserOwnerOption {
  id: number
  username: string
  role: Role
  status: string
}

export interface InviteCode {
  id: number
  code: string
  ownerID: number
  ownerName: string
  maxUses: number
  usedCount: number
  expiredAt: string | null
  status: InviteCodeStatus
  createdAt: string
}

export interface Group {
  id: number
  imGroupId: string
  name: string
  groupType: string
  ownerID: number
  ownerName: string
  ownerNickname: string
  memberCount: number
  isMuted: boolean
  status: string
  createdAt: string
}

export interface GroupNotice {
  groupID: number
  content: string
  summary: string
  source: string
  hasValue: boolean
}

export type OfflineActivityStatus = 'draft' | 'published' | 'closed' | 'cancelled'

export interface OfflineActivity {
  id: number
  title: string
  subtitle: string
  description: string
  city: string
  venue: string
  startAt: string
  endAt: string
  quota: number
  status: OfflineActivityStatus
  registrationCount: number
  createdBy: number
  createdAt: string
}

export interface OfflineActivityRegistration {
  userID: number
  imUserID: string
  nickname: string
  phone: string
  joinedAt: string
}

export interface GroupMember {
  imUserID: string
  nickname: string
  phone: string
  role: string    // Owner / Admin / Member
  joinTime: number
}

export type SyncAction = 'skip' | 'import' | 'update'
export type SyncUserStatus = 'missing_in_im' | 'nickname_changed' | 'avatar_changed' | 'profile_changed' | 'changed' | 'synced'

export interface SyncGroupPreview {
  imGroupId: string
  name: string
  ownerNickname: string
  memberCount: number
  isNew: boolean
  localGroupId?: number
  ownerName: string
  ownerId: number
  action: SyncAction
}

export interface SyncUserPreview {
  imUserID: string
  localUserId: number
  localNickname: string
  imNickname: string
  localAvatarUrl: string
  imAvatarUrl: string
  diffFields: string[]
  status: SyncUserStatus
  action: Extract<SyncAction, 'skip' | 'update'>
}

export type ReportStatus = 'pending' | 'reviewed' | 'dismissed'
export type ReportCategory = 'spam' | 'abuse' | 'fraud' | 'other'

export interface Report {
  id: number
  reporterID: number
  reporterName: string
  targetID: number
  targetType: string
  targetName: string
  category: ReportCategory
  description: string
  status: ReportStatus
  createdAt: string

  contentTargetType?: string
  contentTargetName?: string
}

export type SavingContentTargetType = 'saving_post' | 'saving_comment'

export interface StatsResult {
  userCount: number
  inviteCodeCount: number
  groupCount: number
  assistantCount: number
  reportCount: number
  pendingReportCount: number
  userTotal?: number
  inviteCodeTotal?: number
  groupTotal?: number
  assistantTotal?: number
  reportTotal?: number
  pendingReviewCount?: number
}

export interface AdminCommunityAuthor {
  userID: number
  imUserID: string
  wananID: string
  userNo?: string
  nickname: string
  avatarURL: string
  ownerID: number
  ownerName: string
  communityPostDisabled: boolean
  communityCommentDisabled: boolean
  savingPostDisabled?: boolean
  savingCommentDisabled?: boolean
}

export interface AdminCommunityPost {
  id: number
  postType: string
  contentType?: string
  title: string
  content: string
  likeCount: number
  commentCount: number
  createdAt: string
  updatedAt: string
  author: AdminCommunityAuthor
}

export interface AdminCommunityComment {
  id: number
  postID: number
  content: string
  status: string
  createdAt: string
  updatedAt: string
  author: AdminCommunityAuthor
}

export type OfflineEventStatus = OfflineActivityStatus

export interface OfflineEvent extends OfflineActivity {
  eventID?: number
  eventTitle?: string
  eventSubtitle?: string
  eventDescription?: string
  eventCity?: string
  eventVenue?: string
  eventStartAt?: string
  eventEndAt?: string
  eventQuota?: number
  eventStatus?: string
  eventRegistrationCount?: number
}

export interface OfflineEventRegistration extends OfflineActivityRegistration {
  participantUserID?: number
  participantIMUserID?: string
  participantNickname?: string
  participantPhone?: string
  participantJoinedAt?: string
}

export interface IMHistoryMessage {
  msgKey?: string
  msgSeq: number
  msgRandom?: number
  fromUserID: string
  toUserID?: string
  groupID?: string
  msgTime: number
  content: string
  isRecalled: boolean
  isPlaceMsg: boolean
}

export interface C2CHistoryResult {
  complete: boolean
  nextLastMsgKey?: string
  nextMaxTime?: number
  list: IMHistoryMessage[]
}

export interface GroupHistoryResult {
  isFinished: boolean
  nextMsgSeq?: number
  list: IMHistoryMessage[]
}

export interface IMUserOption {
  id: number
  nickname: string
  phone: string
  imUserID: string
}

export interface IMGroupOption {
  id: number
  name: string
  imGroupId: string
  groupType: string
}

export interface ChatKeyword {
  id: number
  keyword: string
  replacement: string
  enabled: boolean
  createdAt: string
  updatedAt: string
}

export interface ChatKeywordImportItem {
  line: number
  keyword: string
  replacement: string
  action: 'created' | 'updated' | 'skipped' | 'error'
  message: string
}

export interface BatchImportChatKeywordsResult {
  created: number
  updated: number
  skipped: number
  results: ChatKeywordImportItem[]
}

export interface C2CSendResult {
  toUserID: string
  success: boolean
  message: string
}

export interface BatchSendC2CResult {
  msgKey: string
  msgId: string
  results: C2CSendResult[]
}

export interface Conversation {
  type: 'C2C' | 'Group'
  peerID: string
  peerLabel: string
  lastMsgTime: number
  topFlag: boolean
}

export interface PageResult<T> {
  list: T[]
  total: number
  page: number
  pageSize: number
}

export interface ApiResponse<T = any> {
  code: number
  message: string
  data: T
}
