import { PRIMARY } from '@/config/domains'

const PRIVACY_KEY = 'cn_ziyan_privacy_accepted'
const TOKEN_KEY = 'cn_ziyan_token'
const USER_KEY = 'cn_ziyan_user'
const API_SCOPE_KEY = 'cn_ziyan_session_api_scope'
const API_SCOPE = import.meta.env.VITE_CHAT_API_BASE_URL || PRIMARY.api

let sessionEpoch = 0

export function getSessionContext() {
  return `${sessionEpoch}:${getLocalSession()?.userId || ""}`
}

function isCurrentApiSession() {
  return localStorage.getItem(API_SCOPE_KEY) === API_SCOPE
}

export type LocalSession = {
  userId: string
  userSig: string
  token: string
  nickname?: string
  avatar?: string
  sdkAppId?: number
  loginAccount?: string
  refreshToken?: string
  tokenExpiresAt?: string
  userSigExpiresAt?: string
}

export function isPrivacyAccepted() {
  return localStorage.getItem(PRIVACY_KEY) === '1'
}

export function acceptPrivacy() {
  localStorage.setItem(PRIVACY_KEY, '1')
}

export function getLocalSession(): LocalSession | null {
  try {
    if (!isCurrentApiSession()) return null
    const raw = localStorage.getItem(USER_KEY)
    if (!raw) return null
    return JSON.parse(raw) as LocalSession
  } catch {
    return null
  }
}

export function saveLocalSession(s: LocalSession) {
  if (getLocalSession()?.userId !== s.userId) sessionEpoch++
  localStorage.setItem(API_SCOPE_KEY, API_SCOPE)
  localStorage.setItem(USER_KEY, JSON.stringify(s))
  localStorage.setItem(TOKEN_KEY, s.token)
  localStorage.setItem('token', s.token)
  localStorage.setItem('im_user_id', s.userId)
}

export function clearLocalSession() {
  sessionEpoch++
  localStorage.removeItem(API_SCOPE_KEY)
  localStorage.removeItem(USER_KEY)
  localStorage.removeItem(TOKEN_KEY)
  localStorage.removeItem('token')
  localStorage.removeItem('im_user_id')
}

export function getToken() {
  if (!isCurrentApiSession()) return ''
  return localStorage.getItem(TOKEN_KEY) || localStorage.getItem('token') || ''
}
