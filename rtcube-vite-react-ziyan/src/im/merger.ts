import { getChat } from './ImClient'
import type { ChatMessage } from './types'

// eslint-disable-next-line @typescript-eslint/no-explicit-any
type AnyMsg = any

function pickList(res: AnyMsg): AnyMsg[] {
  const candidates = [
    res?.data?.messageList,
    res?.data?.message,
    res?.messageList,
    res?.data,
    res,
  ]
  for (const c of candidates) {
    if (Array.isArray(c) && c.length) return c
  }
  return []
}

export function normalizeMergerItem(raw: AnyMsg): ChatMessage {
  const from = String(raw?.from || raw?.fromAccount || '')
  return {
    ...raw,
    ID: String(raw?.ID || raw?.id || raw?.msgID || `${from}-${raw?.time || Date.now()}`),
    type: String(raw?.type || raw?.elemType || ''),
    conversationID: String(raw?.conversationID || ''),
    from,
    to: String(raw?.to || ''),
    flow: raw?.flow === 'out' ? 'out' : 'in',
    time: Number(raw?.time || raw?.timestamp || 0),
    nick: String(raw?.nick || raw?.nickName || raw?.nameCard || from || '未知'),
    avatar: String(raw?.avatar || raw?.faceUrl || raw?.faceURL || ''),
    payload: raw?.payload || {},
    cloudCustomData: raw?.cloudCustomData,
  }
}

export async function downloadMergerMessages(message: ChatMessage): Promise<ChatMessage[]> {
  const chat = getChat()
  if (typeof chat.downloadMergerMessage === 'function') {
    const attempts = [
      () => chat.downloadMergerMessage(message),
      () => chat.downloadMergerMessage({ message }),
    ]
    for (const run of attempts) {
      try {
        const list = pickList(await run())
        if (list.length) return list.map(normalizeMergerItem)
      } catch {
        /* 换一种调用方式 / 走内嵌列表 */
      }
    }
  }
  const embedded = message?.payload?.messageList
  if (Array.isArray(embedded) && embedded.length) {
    return embedded.map(normalizeMergerItem)
  }
  return []
}
