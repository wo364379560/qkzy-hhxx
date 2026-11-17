import request from './request'
import type { ApiResponse, Role } from '@/types'
import type { ChatQuietHoursSettings, SetChatQuietHoursParams } from './auth'

export interface AccountChatQuietHoursPolicy {
  adminUserId: number
  username: string
  role: Role
  hasPolicy: boolean
  policy: ChatQuietHoursSettings
}

export function getManagerChatQuietHoursPolicy(id: number) {
  return request.get<ApiResponse<AccountChatQuietHoursPolicy>>(`/admins/${id}/chat-quiet-hours`)
}

export function saveManagerChatQuietHoursPolicy(id: number, params: SetChatQuietHoursParams) {
  return request.post<ApiResponse<null>>(`/admins/${id}/chat-quiet-hours`, params)
}

export function clearManagerChatQuietHoursPolicy(id: number, currentPassword: string) {
  return request.post<ApiResponse<null>>(`/admins/${id}/chat-quiet-hours/delete`, {
    current_password: currentPassword,
  })
}

export function getAssistantChatQuietHoursPolicy(id: number) {
  return request.get<ApiResponse<AccountChatQuietHoursPolicy>>(`/assistants/${id}/chat-quiet-hours`)
}

export function saveAssistantChatQuietHoursPolicy(id: number, params: SetChatQuietHoursParams) {
  return request.post<ApiResponse<null>>(`/assistants/${id}/chat-quiet-hours`, params)
}

export function clearAssistantChatQuietHoursPolicy(id: number, currentPassword: string) {
  return request.post<ApiResponse<null>>(`/assistants/${id}/chat-quiet-hours/delete`, {
    current_password: currentPassword,
  })
}
