import { PRIMARY } from '@/config/domains'
import { refreshLoginSession, toLocalSession } from './auth'
import { getLocalSession, type LocalSession } from './session'

const WEB_VAULT_KEY = 'cn_ziyan_account_vault'
const API_SCOPE = import.meta.env.VITE_CHAT_API_BASE_URL || PRIMARY.api
const MAX_ACCOUNTS = 10

export type SavedAccount = LocalSession & {
  lastUsedAt: number
}

type AccountVault = {
  version: 1
  scopes: Record<string, SavedAccount[]>
}

function emptyVault(): AccountVault {
  return { version: 1, scopes: {} }
}

function isSavedAccount(value: unknown): value is SavedAccount {
  if (!value || typeof value !== 'object') return false
  const item = value as Partial<SavedAccount>
  return Boolean(
    typeof item.userId === 'string' &&
      item.userId &&
      typeof item.userSig === 'string' &&
      typeof item.token === 'string',
  )
}

async function readRawVault() {
  if (window.desktop?.readAccountVault) return window.desktop.readAccountVault()
  return localStorage.getItem(WEB_VAULT_KEY) || ''
}

async function writeRawVault(raw: string) {
  if (window.desktop?.writeAccountVault) {
    await window.desktop.writeAccountVault(raw)
    return
  }
  localStorage.setItem(WEB_VAULT_KEY, raw)
}

async function readVault(): Promise<AccountVault> {
  try {
    const raw = await readRawVault()
    if (!raw) return emptyVault()
    const parsed = JSON.parse(raw) as Partial<AccountVault>
    if (parsed.version !== 1 || !parsed.scopes || typeof parsed.scopes !== 'object') {
      return emptyVault()
    }
    const scopes: Record<string, SavedAccount[]> = {}
    for (const [scope, items] of Object.entries(parsed.scopes)) {
      if (!Array.isArray(items)) continue
      scopes[scope] = items.filter(isSavedAccount).slice(0, MAX_ACCOUNTS)
    }
    return { version: 1, scopes }
  } catch {
    return emptyVault()
  }
}

async function writeVault(vault: AccountVault) {
  await writeRawVault(JSON.stringify(vault))
}

function withLastUsedAt(session: LocalSession, fallback = Date.now()): SavedAccount {
  return { ...session, lastUsedAt: fallback }
}

export async function rememberAccount(session: LocalSession) {
  const vault = await readVault()
  const current = vault.scopes[API_SCOPE] || []
  vault.scopes[API_SCOPE] = [
    withLastUsedAt(session),
    ...current.filter((item) => item.userId !== session.userId),
  ].slice(0, MAX_ACCOUNTS)
  await writeVault(vault)
}

export async function listSavedAccounts(): Promise<SavedAccount[]> {
  const vault = await readVault()
  let accounts = vault.scopes[API_SCOPE] || []
  const active = getLocalSession()
  if (active) {
    const existing = accounts.find((item) => item.userId === active.userId)
    accounts = [
      withLastUsedAt(active, existing?.lastUsedAt),
      ...accounts.filter((item) => item.userId !== active.userId),
    ].slice(0, MAX_ACCOUNTS)
    vault.scopes[API_SCOPE] = accounts
    await writeVault(vault)
  }
  return accounts
}

/** 切换前刷新一次，后端会轮换 refresh token，因此成功后必须立刻覆盖保险库。 */
export async function restoreSavedAccount(account: SavedAccount): Promise<LocalSession> {
  const info = await refreshLoginSession(account)
  const refreshed = await toLocalSession({
    ...info,
    user_sig: info.user_sig || account.userSig,
    sdk_app_id: info.sdk_app_id || account.sdkAppId || 0,
    refresh_token: info.refresh_token || account.refreshToken,
    nickname: account.nickname,
    avatar: account.avatar,
  })
  const restored = { ...refreshed, loginAccount: account.loginAccount }
  await rememberAccount(restored)
  return restored
}
