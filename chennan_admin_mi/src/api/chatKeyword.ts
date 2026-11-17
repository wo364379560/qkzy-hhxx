import request from './request'
import type { ApiResponse, BatchImportChatKeywordsResult, ChatKeyword, PageResult } from '@/types'

export interface GetChatKeywordListParams {
  page?: number
  pageSize?: number
  keyword?: string
}

export interface SaveChatKeywordParams {
  keyword: string
  replacement?: string
  enabled: boolean
}

export interface ImportChatKeywordsParams {
  content: string
  enabled: boolean
}

export function getChatKeywordList(params: GetChatKeywordListParams) {
  return request.get<ApiResponse<PageResult<ChatKeyword>>>('/im/keywords', { params })
}

export function createChatKeyword(params: SaveChatKeywordParams) {
  return request.post<ApiResponse<ChatKeyword>>('/im/keywords', params)
}

export function updateChatKeyword(id: number, params: SaveChatKeywordParams) {
  return request.post<ApiResponse<ChatKeyword>>(`/im/keywords/${id}/update`, params)
}

export function deleteChatKeyword(id: number) {
  return request.post<ApiResponse<null>>(`/im/keywords/${id}/delete`)
}

export function importChatKeywords(params: ImportChatKeywordsParams) {
  return request.post<ApiResponse<BatchImportChatKeywordsResult>>('/im/keywords/import', params)
}
