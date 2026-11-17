import request from './request'
import type { ApiResponse, OfflineActivity, OfflineActivityRegistration, PageResult } from '@/types'

export interface OfflineActivityListParams {
  page?: number
  pageSize?: number
  status?: string
}

export interface OfflineActivityPayload {
  title: string
  subtitle?: string
  description?: string
  city: string
  venue: string
  startAt: string
  endAt: string
  quota: number
  status: string
}

export function getOfflineActivityList(params: OfflineActivityListParams = {}) {
  return request.get<ApiResponse<PageResult<OfflineActivity>>>('/offline-activities', { params })
}

export const getOfflineEventList = getOfflineActivityList

export function createOfflineActivity(payload: OfflineActivityPayload) {
  return request.post<ApiResponse<OfflineActivity>>('/offline-activities', payload)
}

export const createOfflineEvent = createOfflineActivity

export function updateOfflineActivity(id: number, payload: OfflineActivityPayload) {
  return request.post<ApiResponse<OfflineActivity>>(`/offline-activities/${id}/update`, payload)
}

export const updateOfflineEvent = updateOfflineActivity

export function closeOfflineActivity(id: number) {
  return request.post<ApiResponse<null>>(`/offline-activities/${id}/close`)
}

export const closeOfflineEvent = closeOfflineActivity

export function getOfflineActivityRegistrations(id: number) {
  return request.get<ApiResponse<OfflineActivityRegistration[]>>(`/offline-activities/${id}/registrations`)
}

export const getOfflineEventRegistrations = getOfflineActivityRegistrations
