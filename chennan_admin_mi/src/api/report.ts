import request from './request'
import type { ApiResponse, Report, PageResult, ReportStatus } from '@/types'

export interface ReportListParams {
  page?: number
  pageSize?: number
  status?: ReportStatus | ''
}

export function getReportList(params: ReportListParams = {}) {
  return request.get<ApiResponse<PageResult<Report>>>('/reports', { params })
}

export function reviewReport(id: number) {
  return request.post<ApiResponse<null>>(`/reports/${id}/review`)
}

export function dismissReport(id: number) {
  return request.post<ApiResponse<null>>(`/reports/${id}/dismiss`)
}
