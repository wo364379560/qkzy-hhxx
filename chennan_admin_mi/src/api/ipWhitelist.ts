import request from './request'
import type {
  ApiResponse,
  AssistantIPWhitelistDetail,
  IPWhitelistOverviewItem,
  PageResult,
} from '@/types'

export interface IPWhitelistOverviewParams {
  page?: number
  pageSize?: number
  keyword?: string
  status?: string
  enabled?: 'true' | 'false'
}

export interface SaveIPWhitelistRuleParams {
  cidr: string
  note: string
}

export interface SaveIPWhitelistParams {
  rules: SaveIPWhitelistRuleParams[]
}

export function getIPWhitelistOverview(params: IPWhitelistOverviewParams = {}) {
  return request.get<ApiResponse<PageResult<IPWhitelistOverviewItem>>>('/ip-whitelists', { params })
}

export function getIPWhitelistDetail(id: number) {
  return request.get<ApiResponse<AssistantIPWhitelistDetail>>(`/ip-whitelists/${id}`)
}

export function saveIPWhitelist(id: number, params: SaveIPWhitelistParams) {
  return request.post<ApiResponse<AssistantIPWhitelistDetail>>(`/ip-whitelists/${id}`, params)
}
