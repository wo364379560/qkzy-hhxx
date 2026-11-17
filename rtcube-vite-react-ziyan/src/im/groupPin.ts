import { getChat, TencentCloudChat } from './ImClient'
import { messageCopyText } from './messageActions'
import type { ChatMessage } from './types'

export type PinnedItem = {
  id: string
  text: string
  nick: string
  raw: ChatMessage
}

export async function pinGroupMessage(
  groupID: string,
  message: ChatMessage,
  isPinned: boolean,
) {
  const chat = getChat()
  if (typeof chat.pinGroupMessage !== 'function') {
    throw new Error('当前 SDK 不支持群消息置顶')
  }
  await chat.pinGroupMessage({ groupID, message, isPinned })
}

export async function fetchPinnedGroupMessages(groupID: string): Promise<ChatMessage[]> {
  const chat = getChat()
  const fn = chat.getPinnedGroupMessageList || chat.getGroupPinnedMessageList
  if (typeof fn !== 'function') return []
  const pick = (res: { data?: { messageList?: ChatMessage[] } | ChatMessage[] }) => {
    const list = (res?.data as { messageList?: ChatMessage[] })?.messageList || res?.data || []
    return Array.isArray(list) ? (list as ChatMessage[]) : []
  }
  try {
    return pick(await fn.call(chat, groupID))
  } catch {
    try {
      return pick(await fn.call(chat, { groupID }))
    } catch {
      return []
    }
  }
}

export function toPinnedItems(list: ChatMessage[]): PinnedItem[] {
  return list
    .map((raw) => {
      const id = String(raw.ID || (raw as ChatMessage & { id?: string }).id || '')
      if (!id) return null
      return {
        id,
        nick: String(raw.nick || raw.from || ''),
        text: messageCopyText(raw) || '[消息]',
        raw,
      }
    })
    .filter((x): x is PinnedItem => !!x)
}

export function onPinnedGroupMessageUpdated(cb: (groupID: string) => void) {
  const chat = getChat()
  const EVENT = (TencentCloudChat as { EVENT?: { PINNED_GROUP_MESSAGE_UPDATED?: string } }).EVENT
  const name = EVENT?.PINNED_GROUP_MESSAGE_UPDATED || 'onPinnedGroupMessageUpdated'
  const handler = (event: { data?: { groupID?: string } }) => {
    cb(String(event?.data?.groupID || ''))
  }
  chat.on?.(name, handler)
  return () => chat.off?.(name, handler)
}
