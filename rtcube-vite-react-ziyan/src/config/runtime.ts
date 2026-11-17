/** 对照 docs/IM_SPEC.md — 禁止各端私改大群/撤回/信令 key */

import { apiOrigin, refreshApiOrigins } from './domains'

export const APP_NAME = '睿之禾'

export let CHAT_API_BASE_URL =
  import.meta.env.VITE_CHAT_API_BASE_URL ||
  (import.meta.env.DEV ? '/api/v1/chat' : `${apiOrigin()}/api/v1/chat`)

export let API_BASE_URL =
  import.meta.env.VITE_API_BASE_URL ||
  (import.meta.env.DEV ? '/api' : `${apiOrigin()}/api`)

export async function initializeRuntimeEndpoints() {
  await refreshApiOrigins()
  if (!import.meta.env.DEV) {
    CHAT_API_BASE_URL = import.meta.env.VITE_CHAT_API_BASE_URL || `${apiOrigin()}/api/v1/chat`
    API_BASE_URL = import.meta.env.VITE_API_BASE_URL || `${apiOrigin()}/api`
  }
}

const envSdk = Number(import.meta.env.VITE_IM_SDK_APP_ID ?? 1600163137)
/** 初始占位 0；必须以登录回包 sdk_app_id 为准（对照 web-h5） */
export let SDK_APP_ID = Number.isFinite(envSdk) && envSdk > 0 ? envSdk : 0

export function setRuntimeSDKAppID(id: number) {
  if (Number.isInteger(id) && id > 0) {
    SDK_APP_ID = id
  }
}

export const MESSAGE_WINDOW_LIMIT = 400
/** 右侧成员列表：首次 150（SDK 单次最多 100，会拆成 100+50），之后每次 50 */
export const GROUP_MEMBER_FIRST_PAGE = 150
export const GROUP_MEMBER_PAGE_SIZE = 50
export const AT_MEMBER_PAGE_SIZE = 50
export const RTC_CALL_TIMEOUT_SEC = 45
export const RTC_BUSINESS_ID = 'rtc_call_v1' as const
export const FRIEND_GREETING = '你我已成为好友，开始聊天吧'
