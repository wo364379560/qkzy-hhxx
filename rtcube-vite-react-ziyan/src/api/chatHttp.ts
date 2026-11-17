import { domainPool } from '@/config/domain-pool'
import { getToken } from './session'
import { routedFetch } from './transport'

type Envelope<T> = { code?: number; message?: string; msg?: string; data?: T }

export async function chatRequest<T>(
  method: 'GET' | 'POST' | 'DELETE', path: string,
  body?: Record<string, unknown>, allowEmpty = false, token = getToken(),
): Promise<T> {
  const headers: Record<string, string> = {}
  if (token) headers.Authorization = `Bearer ${token}`
  if (method === 'POST') headers['Content-Type'] = 'application/json'
  const res = await routedFetch(path, {
    method, headers, body: method === 'POST' ? JSON.stringify(body || {}) : undefined,
  })
  const text = await res.text()
  let json: Envelope<T>
  try { json = text.trim() ? JSON.parse(text) as Envelope<T> : {} }
  catch { throw new Error(`接口返回非 JSON（HTTP ${res.status}）`) }
  if (!res.ok) {
    const error = new Error(json.message || json.msg || `HTTP ${res.status}`) as Error & { status?: number }
    error.status = res.status
    throw error
  }
  if (json.code !== undefined && json.code !== 0 && json.code !== 200) {
    throw new Error(json.message || json.msg || `业务错误 ${json.code}`)
  }
  if (json.data === undefined || json.data === null) {
    if (allowEmpty) return undefined as T
    throw new Error(json.message || json.msg || '响应数据为空')
  }
  return json.data
}

export function chatPost<T = unknown>(path: string, body: Record<string, unknown>, allowEmpty = false): Promise<T> {
  return chatRequest<T>('POST', path, body, allowEmpty)
}
export function chatGet<T = unknown>(path: string, allowEmpty = false): Promise<T> {
  return chatRequest<T>('GET', path, undefined, allowEmpty)
}
export function chatDelete<T = unknown>(path: string, allowEmpty = false): Promise<T> {
  return chatRequest<T>('DELETE', path, undefined, allowEmpty)
}
export function getActiveApiOrigin() {
  const override = import.meta.env.VITE_CHAT_API_BASE_URL
  if (override?.startsWith('https://')) return new URL(override).origin
  return domainPool.currentOrigin()
}
