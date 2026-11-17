/**
 * 用户在线状态 — TIM getUserStatus / USER_STATUS_UPDATED
 */
import { getChat, TencentCloudChat } from './ImClient'

// eslint-disable-next-line @typescript-eslint/no-explicit-any
type AnyChat = any

export type OnlineMap = Record<string, boolean>

export function isMutedRemind(remind?: string | number | boolean): boolean {
  if (remind === true) return true
  const s = String(remind || '')
  return /ACCEPT_NOT_NOTIFY|NOT_NOTIFY|Discard|MSG_REMIND_ACPT_NOT_NOTE/i.test(s)
}

/** 批量查询在线；1=在线 */
export async function fetchUserOnlineMap(userIDs: string[]): Promise<OnlineMap> {
  const ids = Array.from(new Set(userIDs.filter(Boolean))).slice(0, 100)
  if (!ids.length) return {}
  const c = getChat()
  try {
    await c.subscribeUserStatus?.({ userIDList: ids })
  } catch {
    /* 订阅失败仍尝试 get */
  }
  try {
    const res = await c.getUserStatus({ userIDList: ids })
    const success = (res?.data?.successUserList || res?.data?.successList || []) as AnyChat[]
    const TYPES = (TencentCloudChat as AnyChat).TYPES
    const onlineCode = TYPES?.USER_STATUS_ONLINE ?? 1
    const map: OnlineMap = {}
    for (const row of success) {
      const id = String(row.userID || '')
      if (!id) continue
      const status = Number(row.statusType ?? row.status ?? 0)
      map[id] = status === onlineCode || status === 1
    }
    // 未返回的默认离线
    for (const id of ids) {
      if (map[id] === undefined) map[id] = false
    }
    return map
  } catch {
    return Object.fromEntries(ids.map((id) => [id, false]))
  }
}

export function onUserStatusUpdated(cb: (map: OnlineMap) => void) {
  const c = getChat()
  const EVENT = (TencentCloudChat as AnyChat).EVENT
  const TYPES = (TencentCloudChat as AnyChat).TYPES
  const onlineCode = TYPES?.USER_STATUS_ONLINE ?? 1
  const evt = EVENT.USER_STATUS_UPDATED
  if (!evt) return () => {}
  const handler = (event: { data?: AnyChat[] }) => {
    const list = event.data || []
    const map: OnlineMap = {}
    for (const row of list) {
      const id = String(row.userID || '')
      if (!id) continue
      const status = Number(row.statusType ?? row.status ?? 0)
      map[id] = status === onlineCode || status === 1
    }
    if (Object.keys(map).length) cb(map)
  }
  c.on(evt, handler)
  return () => c.off(evt, handler)
}
