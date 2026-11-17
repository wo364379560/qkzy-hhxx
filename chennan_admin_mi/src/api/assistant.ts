import { normalizeAdminUsername } from '../utils/adminAccount'
import request from './request'
import type {
  ApiResponse,
  AssistantDetail,
  AssistantIPWhitelistDetail,
  AssistantIPWhitelistOverview,
  InviteCode,
  PageResult,
} from '@/types'

export interface AssistantListParams {
  page?: number
  pageSize?: number
}

export interface AssistantIPWhitelistOverviewParams {
  page?: number
  pageSize?: number
  keyword?: string
  status?: string
  enabled?: 'true' | 'false'
}

export interface CreateAssistantParams {
  username: string
  password: string
  display_name?: string
}

export interface UpdateAssistantProfileParams {
  username?: string
  display_name: string
}

export interface AssistantStatusChangeResult {
  assistantId: number
  username: string
  status: string
  affectedUserCount: number
}

export function getAssistantList(params: AssistantListParams = {}) {
  return request.get<ApiResponse<PageResult<AssistantDetail>>>('/assistants', { params })
}

export function getAssistantIPWhitelistOverview(params: AssistantIPWhitelistOverviewParams = {}) {
  return request.get<ApiResponse<PageResult<AssistantIPWhitelistOverview>>>('/assistants/ip-whitelist-overview', { params })
}

export function getAssistantMe() {
  return request.get<ApiResponse<AssistantDetail>>('/assistants/me')
}

export function createAssistant(params: CreateAssistantParams) {
  return request.post<ApiResponse<AssistantDetail>>('/assistants', normalizeCreateAssistantParams(params))
}

export function normalizeCreateAssistantParams(params: CreateAssistantParams): Required<CreateAssistantParams> {
  const username = normalizeAdminUsername(params.username)
  if (Array.from(params.password).length < 8) {
    throw new Error('密码至少 8 位')
  }
  const display_name = params.display_name?.trim() || username
  if (Array.from(display_name).length > 64) {
    throw new Error('昵称最多 64 个字符')
  }
  return { username, password: params.password, display_name }
}

export function updateAssistantProfile(id: number, params: UpdateAssistantProfileParams) {
  return request.post<ApiResponse<AssistantDetail>>(`/assistants/${id}/profile`, {
    ...params,
    ...(params.username === undefined ? {} : { username: normalizeAdminUsername(params.username) }),
  })
}

export function deleteAssistant(id: number) {
  return request.post<ApiResponse<null>>(`/assistants/${id}/delete`)
}

export function generateAssistantInviteCode(id: number) {
  return request.post<ApiResponse<InviteCode>>(`/assistants/${id}/invite-code`)
}

export function updateInviteCode(codeId: number, data: { maxUses: number; status: string; code?: string }) {
  return request.post<ApiResponse<InviteCode>>(`/invite-codes/${codeId}/update`, data)
}

export function regenerateInviteCode(codeId: number) {
  return request.post<ApiResponse<InviteCode>>(`/invite-codes/${codeId}/regenerate`)
}

export function banAssistant(id: number) {
  return request.post<ApiResponse<AssistantStatusChangeResult>>(`/assistants/${id}/ban`)
}

export function unbanAssistant(id: number) {
  return request.post<ApiResponse<AssistantStatusChangeResult>>(`/assistants/${id}/unban`)
}

export function resetAssistantPassword(id: number, password: string) {
  return request.post<ApiResponse<null>>(`/assistants/${id}/reset-password`, { password })
}

export function resetAssistantTOTP(id: number) {
  return request.post<ApiResponse<null>>(`/assistants/${id}/reset-totp`)
}

export function resolveAssistantLoginCooldown(id: number) {
  return request.post<ApiResponse<null>>(`/assistants/${id}/login-cooldown/resolve`)
}

export interface SaveAssistantIPWhitelistRuleParams {
  cidr: string
  note: string
}

export interface SaveAssistantIPWhitelistParams {
  rules: SaveAssistantIPWhitelistRuleParams[]
}

export function getAssistantIPWhitelist(id: number) {
  return request.get<ApiResponse<AssistantIPWhitelistDetail>>(`/assistants/${id}/ip-whitelist`)
}

export function saveAssistantIPWhitelist(id: number, params: SaveAssistantIPWhitelistParams) {
  return request.post<ApiResponse<AssistantIPWhitelistDetail>>(`/assistants/${id}/ip-whitelist`, params)
}

export interface StatsData {
  userCount: number
  inviteCodeCount: number
  groupCount: number
  assistantCount: number
  reportCount: number
  pendingReportCount: number
}

export function getStats() {
  return request.get<ApiResponse<StatsData>>('/stats')
}
