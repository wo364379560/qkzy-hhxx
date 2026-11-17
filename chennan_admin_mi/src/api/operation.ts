import request from './request'
import type { ApiResponse, AppUser, PageResult } from '@/types'

export interface GovernanceOverview {
  userCount: number
  groupCount: number
  cleanupCandidateCount: number
  loginRiskCount: number
  ipBlockCount: number
  pendingReportCount: number
  whitelistCount: number
  manualBannedCount: number
  restoredCount: number
}

export interface CleanupStats {
  candidateCount: number
  bannedCount: number
  whitelistCount: number
  manualBannedCount: number
  restoredCount: number
  protectedAAccounts: number
  createdByNonZero: number
  withInviteCode: number
}

export interface SecurityRiskSummary {
  accountPending: number
  accountWatching: number
  accountCooldown: number
  accountAlerted: number
  ipOpen: number
  ipWatching: number
  ipAlerted: number
  ipBlocked: number
  manualBannedAccounts: number
  quietGroupPending: number
}

export interface CleanupCandidate extends AppUser {
  type: string
  cleanupStatus: string
  reason: string
}

export interface LoginRisk {
  id: number
  scope: 'account' | 'ip'
  riskKey: string
  username: string
  imUserID: string
  userId: number | null
  userStatus: string
  ipAddress: string
  status: string
  level: string
  reason: string
  failureCount: number
  successCount: number
  cooldownUntil: string
  updatedAt: string
}

export interface IPAccessBlock {
  id: number
  ipAddress: string
  status: string
  reason: string
  triggerUsername: string
  triggerFailCount: number
  triggerWindow: string
  createdAt: string
  updatedAt: string
}

export interface QuietHoursGroupRisk {
  id: number
  imGroupId: string
  groupName: string
  groupType: string
  ownerImUserId: string
  ownerId: number | null
  ownerName: string
  policyScope: string
  policyStart: string
  policyEnd: string
  tencentCreateTime: string
  triggerSource: string
  action: string
  status: string
  reason: string
  tencentResult: string
  dryRun: boolean
  lastCheckedAt: string
  dismissedAt: string
  resolvedAt: string
  resolveReason: string
  createdAt: string
  updatedAt: string
}

export function getGovernanceOverview() {
  return request.get<ApiResponse<GovernanceOverview>>('/operation/overview')
}

export function getSecurityRiskSummary() {
  return request.get<ApiResponse<SecurityRiskSummary>>('/security/risk-summary')
}

export function getCleanupStats() {
  return request.get<ApiResponse<CleanupStats>>('/cleanup/stats')
}

export function getCleanupCandidates(params: {
  page?: number
  pageSize?: number
  keyword?: string
  currentLoginIP?: string
} = {}) {
  return request.get<ApiResponse<PageResult<CleanupCandidate>>>('/cleanup/candidates', { params })
}

export function whitelistCleanupCandidate(id: number, reason = '') {
  return request.post<ApiResponse<null>>(`/cleanup/candidates/${id}/whitelist`, { reason })
}

export function banCleanupCandidate(id: number, reason = '') {
  return request.post<ApiResponse<null>>(`/cleanup/candidates/${id}/ban`, { reason })
}

export function restoreCleanupUser(id: number, reason = '') {
  return request.post<ApiResponse<null>>(`/cleanup/users/${id}/restore`, { reason })
}

export function getLoginRisks(params: {
  page?: number
  pageSize?: number
  scope?: 'account' | 'ip'
  status?: string
  keyword?: string
  source?: string
} = {}) {
  return request.get<ApiResponse<PageResult<LoginRisk>>>('/login-risks', { params })
}

export function resolveLoginRisk(id: number, reason = '') {
  return request.post<ApiResponse<null>>(`/login-risks/${id}/resolve`, { reason })
}

export function getIPAccessBlocks(params: {
  page?: number
  pageSize?: number
  status?: string
  ipAddress?: string
} = {}) {
  return request.get<ApiResponse<PageResult<IPAccessBlock>>>('/security/ip-blocks', { params })
}

export function blockIP(ipAddress: string, reason = '') {
  return request.post<ApiResponse<null>>('/security/ip-blocks', {
    ip_address: ipAddress,
    reason,
  })
}

export function unblockIP(id: number, reason = '') {
  return request.post<ApiResponse<null>>(`/security/ip-blocks/${id}/unblock`, { reason })
}

export function getQuietHoursGroupRisks(params: {
  page?: number
  pageSize?: number
  keyword?: string
  status?: string
  ownerID?: number
} = {}) {
  return request.get<ApiResponse<PageResult<QuietHoursGroupRisk>>>('/security/quiet-hour-groups', { params })
}

export function resolveQuietHoursGroupRisk(id: number, reason = '') {
  return request.post<ApiResponse<null>>(`/security/quiet-hour-groups/${id}/resolve`, { reason })
}

export function syncQuietHoursGroupRisk(id: number) {
  return request.post<ApiResponse<null>>(`/security/quiet-hour-groups/${id}/sync`, {})
}

export function dismissQuietHoursGroupRisk(id: number, reason = '') {
  return request.post<ApiResponse<null>>(`/security/quiet-hour-groups/${id}/dismiss`, { reason })
}
