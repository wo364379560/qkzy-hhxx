import { domainPool, retryableRouteFailure, waitWithinDeadline } from '@/config/domain-pool'
import { getSessionContext } from './session'

/** One logical deadline; only GET can recover once on another published origin. */
export async function routedFetch(
  path: string,
  init: RequestInit = {},
  prefix = '/api/v1/chat',
  timeoutMs = 30000,
): Promise<Response> {
  const session = getSessionContext()
  const deadline = Date.now() + timeoutMs
  const override = prefix === '/api/v1/chat'
    ? import.meta.env.VITE_CHAT_API_BASE_URL : import.meta.env.VITE_API_BASE_URL
  const fixed = override || (import.meta.env.DEV ? prefix : '')
  const assertSession = () => {
    if (getSessionContext() !== session) throw new Error('登录账号已切换，请重新操作')
    if (init.signal?.aborted) throw new DOMException('请求已取消', 'AbortError')
  }
  assertSession()
  if (!fixed) { await waitWithinDeadline(domainPool.refresh(), deadline); await waitWithinDeadline(domainPool.waitForRecovery(), deadline) }
  assertSession()
  let origin = domainPool.currentOrigin()
  const method = (init.method || 'GET').toUpperCase()
  for (let attempt = 0; attempt < 2; attempt++) {
    assertSession()
    const remaining = deadline - Date.now()
    if (remaining <= 0) throw new Error('请求超时，请检查网络后重试')
    const controller = new AbortController()
    const cancel = () => controller.abort()
    init.signal?.addEventListener('abort', cancel, { once: true })
    const canRetry = method === 'GET' && attempt === 0 && !fixed
    const timer = setTimeout(cancel, Math.max(1, canRetry ? Math.floor(remaining / 2) : remaining))
    const pool = domainPool.snapshot()
    if (!pool.origins.includes(origin)) origin = domainPool.currentOrigin()
    const revision = pool.revision
    let recover = false
    let responseStatus: number | undefined
    try {
      const response = await fetch(`${fixed || origin + prefix}${path}`, {
        ...init, method, signal: controller.signal, redirect: 'error', credentials: 'omit', cache: 'no-store',
      })
      responseStatus = response.status
      // Include body consumption in the timeout and account fence.
      const text = await response.text()
      assertSession()
      let json: unknown
      try { json = JSON.parse(text) } catch { /* Keep the original non-JSON error for the caller. */ }
      const code = json && typeof json === 'object' ? (json as { code?: unknown }).code : undefined
      if (response.ok && (code === 0 || code === 200) && !fixed) domainPool.remember(origin, revision)
      recover = canRetry && retryableRouteFailure(response.status, json)
      if (!recover || !domainPool.alternative(origin)) {
        if (!fixed && retryableRouteFailure(response.status, json)) void domainPool.recover(origin)
        return new Response(text || null, { status: response.status, statusText: response.statusText, headers: response.headers })
      }
    } catch (error) {
      assertSession()
      const networkError = error instanceof TypeError || (error instanceof Error && error.name === 'AbortError')
      if (!canRetry || (responseStatus !== undefined && responseStatus >= 400 && ![502, 503, 504].includes(responseStatus)) || !networkError) {
        if (!fixed && networkError && (responseStatus === undefined || responseStatus < 400 || [502, 503, 504].includes(responseStatus))) void domainPool.recover(origin)
        throw error
      }
      recover = true
    } finally {
      clearTimeout(timer)
      init.signal?.removeEventListener('abort', cancel)
    }
    if (recover) {
      const current = domainPool.currentOrigin()
      const next = current !== origin ? current : domainPool.alternative(origin)
      if (!next) throw new Error('网络连接异常，请检查网络后重试')
      origin = next
    }
  }
  throw new Error('网络连接异常，请检查网络后重试')
}
