import request from './request'
import type { ApiResponse, PageResult } from '@/types'
export interface OwnershipOperation {
  id: number; operation_id: string; source: string; caller: string; target: string
  state: string; reason_code: string; attempts: number; available_at: string
  manual_retries: number; last_retried_by?: number; last_retried_at?: string
  retry_history?: string
  needs_review: boolean; effects_pending: boolean; before: string; after: string; created_at: string
}
export function listOwnershipOperations(page: number, state: string) {
  return request.get<ApiResponse<PageResult<OwnershipOperation>>>('/ownership-operations', { params: { page, pageSize: 25, state } })
}
export function retryOwnershipOperation(id: number) {
  return request.post<ApiResponse<{ queued: boolean }>>(`/ownership-operations/${id}/retry`)
}
