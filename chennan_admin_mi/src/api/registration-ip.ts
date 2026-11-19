import request from './request'
import type { ApiResponse } from '@/types'
export interface RegistrationIPEntry { ip: string; note: string; updated_by: number; updated_at: string }
export function listRegistrationIPs() { return request.get<ApiResponse<{ items: RegistrationIPEntry[] }>>('/security/registration-ips') }
export function changeRegistrationIP(data: { ip: string; note: string; current_password: string; remove: boolean }) {
  return request.post<ApiResponse>('/security/registration-ips', data)
}
