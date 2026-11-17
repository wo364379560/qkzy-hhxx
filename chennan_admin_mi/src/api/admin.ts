import { normalizeAdminUsername } from '../utils/adminAccount'
import request from './request'
import type { ApiResponse, AdminUser, AdminUserIPWhitelistDetail, PageResult } from '@/types'

export interface AdminListParams {
  page?: number
  pageSize?: number
}

export interface CreateAdminParams {
  username: string
  password: string
  display_name: string
}

export interface UpdateAdminProfileParams {
  username?: string
  display_name: string
}

export interface AdminStatusChangeResult {
  adminId: number
  username: string
  status: string
}

export function getAdminList(params: AdminListParams = {}) {
  return request.get<ApiResponse<PageResult<AdminUser>>>('/admins', { params })
}

export function createAdmin(params: CreateAdminParams) {
  return request.post<ApiResponse<AdminUser>>('/admins', { ...params, username: normalizeAdminUsername(params.username) })
}

export function updateAdminProfile(id: number, params: UpdateAdminProfileParams) {
  return request.post<ApiResponse<AdminUser>>(`/admins/${id}/profile`, {
    ...params,
    ...(params.username === undefined ? {} : { username: normalizeAdminUsername(params.username) }),
  })
}

export function deleteAdmin(id: number) {
  return request.post<ApiResponse<null>>(`/admins/${id}/delete`)
}

export function banAdmin(id: number) {
  return request.post<ApiResponse<AdminStatusChangeResult>>(`/admins/${id}/ban`)
}

export function unbanAdmin(id: number) {
  return request.post<ApiResponse<AdminStatusChangeResult>>(`/admins/${id}/unban`)
}

export function resetAdminPassword(id: number, password: string) {
  return request.post<ApiResponse<null>>(`/admins/${id}/reset-password`, { password })
}

export function resetAdminTOTP(id: number) {
  return request.post<ApiResponse<null>>(`/admins/${id}/reset-totp`)
}

export function resolveAdminLoginCooldown(id: number) {
  return request.post<ApiResponse<null>>(`/admins/${id}/login-cooldown/resolve`)
}

export interface SaveAdminIPWhitelistRuleParams {
  cidr: string
  note: string
}

export interface SaveAdminIPWhitelistParams {
  rules: SaveAdminIPWhitelistRuleParams[]
}

export function getAdminIPWhitelist(id: number) {
  return request.get<ApiResponse<AdminUserIPWhitelistDetail>>(`/admins/${id}/ip-whitelist`)
}

export function saveAdminIPWhitelist(id: number, params: SaveAdminIPWhitelistParams) {
  return request.post<ApiResponse<AdminUserIPWhitelistDetail>>(`/admins/${id}/ip-whitelist`, params)
}
