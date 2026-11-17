/**
 * 对照 web-h5：登录走 /web-login，资料走 /profile。
 */
import { chatRequest } from './chatHttp'
import type { LocalSession } from './session'
import { getToken } from './session'

export type LoginUserInfo = {
  user_id: string
  sdk_app_id: number
  user_sig: string
  user_sig_expires_at?: string
  token: string
  token_expires_at?: string
  refresh_token?: string
  must_change_password?: boolean
  is_activated?: boolean
  nickname?: string
  avatar?: string
}

export type ChatProfile = {
  im_user_id: string
  wanan_id?: string
  nickname: string
  avatar_url: string
  self_signature?: string
  gender?: number
  birthday?: number
  invite_code?: string
  can_create_group?: boolean
}

export type UpdateProfilePayload = {
  wananID?: string
  nickname?: string
  avatarURL?: string
  selfSignature?: string
  gender?: number
  birthday?: number
}

export type SystemConfig = {
  require_sms_code?: boolean
  require_invite_code?: boolean
  enable_security_question?: boolean
  allow_create_group?: boolean
  app_name?: string
}

export type SecurityQuestion = {
  id: number
  question: string
}

function getDeviceId() {
  const key = 'cn_ziyan_device_id'
  let id = localStorage.getItem(key)
  if (!id) {
    id = `ziyan_${Math.random().toString(36).slice(2)}${Date.now().toString(36)}`
    localStorage.setItem(key, id)
  }
  return id
}

async function chatPost<T>(
  path: string,
  body: Record<string, unknown>,
  token?: string,
  allowEmpty = false,
): Promise<T> {
  return chatRequest<T>('POST', path, body, allowEmpty, token || '')
}

async function chatGet<T>(path: string, token?: string): Promise<T> {
  return chatRequest<T>('GET', path, undefined, false, token || '')
}

/** 对照 web-h5：POST /web-login */
export async function loginByPassword(phone: string, password: string) {
  return chatPost<LoginUserInfo>('/web-login', {
    phone: phone.trim(),
    password,
    platform: 'web',
    device_id: getDeviceId(),
  })
}

/** 用已保存的 refresh token 换取新的业务 token 和 UserSig。 */
export async function refreshLoginSession(session: LocalSession) {
  if (!session.refreshToken) throw new Error('该账号需要重新输入密码')
  return chatPost<LoginUserInfo>('/auth/refresh', {
    refresh_token: session.refreshToken,
    need_user_sig_refresh: true,
    user_sig_expires_at: session.userSigExpiresAt || '',
    sdk_app_id: session.sdkAppId || 0,
    platform: 'web',
    device_id: getDeviceId(),
  })
}

export async function fetchProfile(token?: string) {
  return chatGet<ChatProfile>('/profile', token || getToken())
}

export async function updateProfile(payload: UpdateProfilePayload, token?: string) {
  const body: Record<string, unknown> = {}
  if (payload.wananID !== undefined) body.wanan_id = payload.wananID
  if (payload.nickname !== undefined) body.nickname = payload.nickname
  if (payload.avatarURL !== undefined) body.avatar_url = payload.avatarURL
  if (payload.selfSignature !== undefined) body.self_signature = payload.selfSignature
  if (payload.gender !== undefined) body.gender = payload.gender
  if (payload.birthday !== undefined) body.birthday = payload.birthday
  return chatPost<ChatProfile>('/profile/update', body, token || getToken())
}

export async function sendSmsCode(phone: string, type: 'register' | 'reset_password' = 'register') {
  await chatPost(
    '/sms/send',
    {
      phone: phone.trim(),
      scene: type === 'reset_password' ? 'reset_password' : 'register',
      device_id: getDeviceId(),
    },
    undefined,
    true,
  )
  return { expire_time: 60 }
}

export async function registerByPhone(params: {
  phone: string
  sms_code: string
  password: string
  nickname?: string
  invite_code?: string
}) {
  return chatPost<LoginUserInfo>('/register', {
    phone: params.phone.trim(),
    code: params.sms_code.trim(),
    password: params.password,
    nickname: params.nickname?.trim() || params.phone.trim(),
    invite_code: params.invite_code?.trim() || undefined,
    platform: 'web',
    device_id: getDeviceId(),
  })
}

export async function registerByUsername(_params: {
  username: string
  password: string
  nickname?: string
  invite_code?: string
  security_question_id?: number
  security_answer?: string
}): Promise<LoginUserInfo> {
  throw new Error('当前环境仅支持手机号注册（对照 web-h5）')
}

export async function getSystemConfig(): Promise<SystemConfig> {
  return {
    require_invite_code: false,
    enable_security_question: false,
    require_sms_code: true,
  }
}

export async function getSecurityQuestions(): Promise<SecurityQuestion[]> {
  return []
}

/** 对照 web-h5：POST /account/delete */
export async function deleteAccount(token?: string) {
  return chatPost<unknown>('/account/delete', { platform: 'web', device_id: getDeviceId() }, token || getToken(), true)
}

export async function changePassword(newPassword: string, token?: string) {
  return chatPost<unknown>(
    '/password/change',
    { new_password: newPassword },
    token || getToken(),
    true,
  )
}

export async function toLocalSession(info: LoginUserInfo): Promise<LocalSession> {
  let nickname = info.nickname || info.user_id
  let avatar = info.avatar || ''
  try {
    const profile = await fetchProfile(info.token)
    if (profile.nickname) nickname = profile.nickname
    if (profile.avatar_url) avatar = profile.avatar_url
  } catch {
    /* 资料失败不阻断登录 */
  }
  return {
    userId: info.user_id,
    userSig: info.user_sig,
    token: info.token,
    nickname,
    avatar,
    sdkAppId: info.sdk_app_id,
    refreshToken: info.refresh_token,
    tokenExpiresAt: info.token_expires_at,
    userSigExpiresAt: info.user_sig_expires_at,
  }
}
