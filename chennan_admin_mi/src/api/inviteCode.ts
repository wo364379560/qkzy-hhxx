import request from './request'
import type { ApiResponse, InviteCode, PageResult } from '@/types'

export interface InviteCodeListParams {
  page?: number
  pageSize?: number
  status?: string
}

export interface CreateInviteCodeParams {
  maxUses?: number
  expiredAt?: string | null
}

export interface BatchCreateInviteCodeParams {
  count: number
  maxUses?: number
  expiredAt?: string | null
  ownerID?: number
}

export interface UpdateInviteCodeParams {
  maxUses?: number
  status?: string
  code?: string
}

export function getInviteCodeList(params: InviteCodeListParams = {}) {
  return request.get<ApiResponse<PageResult<InviteCode>>>('/invite-codes', { params })
}

export function createInviteCode(params: CreateInviteCodeParams) {
  return request.post<ApiResponse<InviteCode>>('/invite-codes', params)
}

export function batchCreateInviteCodes(params: BatchCreateInviteCodeParams) {
  return request.post<ApiResponse<InviteCode[]>>('/invite-codes/batch', params)
}

export function updateInviteCode(id: number, params: UpdateInviteCodeParams) {
  return request.post<ApiResponse<InviteCode>>(`/invite-codes/${id}/update`, params)
}

export function regenerateInviteCode(id: number) {
  return request.post<ApiResponse<InviteCode>>(`/invite-codes/${id}/regenerate`)
}

export function deleteInviteCode(id: number) {
  return request.post<ApiResponse<null>>(`/invite-codes/${id}/delete`)
}
