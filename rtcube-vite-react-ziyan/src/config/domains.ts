import { BOOTSTRAP_ORIGINS, BOOTSTRAP_PATH, domainPool, normalizeOrigin } from './domain-pool'

// Keep the existing session namespace stable across routing changes.
export const PRIMARY = { admin: 'https://api.prfgsu.com', api: 'https://api.prfgsu.com', web: 'https://api.prfgsu.com' } as const
export const BACKUP = { admin: 'https://api.uktjwg.cn', api: 'https://api.uktjwg.cn', web: 'https://api.uktjwg.cn' } as const
export const APP_BOOTSTRAP_ORIGINS = BOOTSTRAP_ORIGINS
export const APP_BOOTSTRAP_URLS = APP_BOOTSTRAP_ORIGINS.map(origin => origin + BOOTSTRAP_PATH)
export const APP_BOOTSTRAP_ORIGIN = APP_BOOTSTRAP_ORIGINS[0]
export const APP_BOOTSTRAP_URL = APP_BOOTSTRAP_URLS[0]
export let API_ORIGINS = domainPool.snapshot().origins
export function parseApiOrigins(value: unknown): string[] {
  if (!Array.isArray(value) || !value.length || value.length > 16) return []
  const origins = value.map(normalizeOrigin)
  return origins.some(o => !o) ? [] : [...new Set(origins as string[])]
}
export async function refreshApiOrigins(): Promise<void> {
  await domainPool.refresh()
  API_ORIGINS = domainPool.snapshot().origins
}
export function apiOrigin(preferBackup = false) {
  return preferBackup ? domainPool.alternative(domainPool.currentOrigin()) || domainPool.currentOrigin() : domainPool.currentOrigin()
}
