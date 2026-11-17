import axios from 'axios'
import { ADMIN_API_BASE_URL } from '@/config/runtime'
import request from './request'
import type {
  ApiResponse,
  BatchSendC2CResult,
  C2CHistoryResult,
  Conversation,
  GroupHistoryResult,
  IMGroupOption,
  IMUserOption,
} from '@/types'

export interface SendC2CMessagesParams {
  fromUserID: string
  toUserIDs: string[]
  content: string
  syncToSender?: boolean
}

export interface SearchIMUsersParams {
  keyword: string
}

export interface SearchIMGroupsParams {
  keyword: string
}

export interface QueryC2CHistoryParams {
  operatorUserID: string
  peerUserID: string
  pageSize?: number
  minTime?: number
  maxTime?: number
  lastMsgKey?: string
}

export interface WithdrawC2CMessageParams {
  fromUserID: string
  toUserID: string
  msgKey: string
}

export interface ListConversationsParams {
  fromUserID: string
  limit?: number
}

export interface DeleteConversationParams {
  fromUserID: string
  type: 'C2C' | 'Group'
  peerID: string
  clearRamble?: boolean
}

export interface QueryGroupHistoryParams {
  groupID: string
  operatorUserID?: string
  pageSize?: number
  lastMsgSeq?: number
}

export interface ExportC2CPairHistoryParams {
  operatorUserID: string
  peerUserID: string
}

export interface ExportUserC2CHistoryParams {
  userID: string
}

export function sendC2CMessages(params: SendC2CMessagesParams) {
  return request.post<ApiResponse<BatchSendC2CResult>>('/im/c2c/send', params)
}

export function searchIMUsers(params: SearchIMUsersParams) {
  return request.get<ApiResponse<IMUserOption[]>>('/im/users/search', { params })
}

export function searchIMGroups(params: SearchIMGroupsParams) {
  return request.get<ApiResponse<IMGroupOption[]>>('/im/groups/search', { params })
}

export function queryC2CHistory(params: QueryC2CHistoryParams) {
  return request.get<ApiResponse<C2CHistoryResult>>('/im/c2c/history', { params })
}

export function withdrawC2CMessage(params: WithdrawC2CMessageParams) {
  return request.post<ApiResponse<null>>('/im/c2c/withdraw', params)
}

export function listConversations(params: ListConversationsParams) {
  return request.get<ApiResponse<Conversation[]>>('/im/conversations', { params })
}

export function deleteConversation(params: DeleteConversationParams) {
  return request.post<ApiResponse<null>>('/im/conversations/delete', params)
}

export function queryGroupHistory(params: QueryGroupHistoryParams) {
  return request.get<ApiResponse<GroupHistoryResult>>('/im/group/history', { params })
}

function getAdminAuthHeaders() {
  const token = localStorage.getItem('admin_token')
  return token ? { Authorization: `Bearer ${token}` } : {}
}

function resolveDownloadFilename(contentDisposition: string | undefined, fallback: string) {
  if (!contentDisposition) return fallback
  const utf8Match = contentDisposition.match(/filename\*=UTF-8''([^;]+)/i)
  if (utf8Match?.[1]) {
    return decodeURIComponent(utf8Match[1])
  }
  const asciiMatch = contentDisposition.match(/filename="?([^"]+)"?/i)
  if (asciiMatch?.[1]) {
    return asciiMatch[1]
  }
  return fallback
}

async function normalizeBlobError(error: any) {
  const blob = error?.response?.data
  if (blob instanceof Blob) {
    const text = await blob.text()
    try {
      const parsed = JSON.parse(text)
      return new Error(parsed.message || error.message || '导出失败')
    } catch {
      return new Error(text || error.message || '导出失败')
    }
  }
  return new Error(error?.message || '导出失败')
}

async function downloadExport(
  url: string,
  data: Record<string, string>,
  fallbackFilename: string,
) {
  try {
    const response = await axios.post(url, data, {
      baseURL: ADMIN_API_BASE_URL,
      responseType: 'blob',
      headers: {
        'Content-Type': 'application/json',
        ...getAdminAuthHeaders(),
      },
    })

    const filename = resolveDownloadFilename(
      response.headers['content-disposition'],
      fallbackFilename,
    )
    const blobUrl = window.URL.createObjectURL(response.data)
    const link = document.createElement('a')
    link.href = blobUrl
    link.download = filename
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    window.URL.revokeObjectURL(blobUrl)
  } catch (error: any) {
    throw await normalizeBlobError(error)
  }
}

export function exportC2CPairHistory(params: ExportC2CPairHistoryParams) {
  return downloadExport(
    '/im/c2c/export/pair',
    params,
    `c2c_pair_export_${params.operatorUserID}_${params.peerUserID}.csv`,
  )
}

export function exportUserC2CHistory(params: ExportUserC2CHistoryParams) {
  return downloadExport(
    '/im/c2c/export/user',
    params,
    `c2c_user_export_${params.userID}.csv`,
  )
}
