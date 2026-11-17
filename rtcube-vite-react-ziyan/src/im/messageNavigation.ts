import { fetchMessageList, getChat, normalizeChatMessage } from './ImClient'
import type { MessageQuote } from './quote'
import type { ChatMessage } from './types'

/** Locate older quotes without retaining every page or relying on the 400-row UI window. */
export async function findMessageContext(
  conversationID: string,
  messageID: string,
  quote: MessageQuote | undefined,
  cancelled: () => boolean,
): Promise<ChatMessage[] | null> {
  const chat = getChat()
  const cached = chat.findMessage?.(messageID)
  const local = cached?.conversationID === conversationID ? normalizeChatMessage(cached) : undefined
  const sequence = local?.sequence || local?.seq || local?.msgSeq || quote?.messageSequence
  const time = local?.time || quote?.messageTime
  const anchor = conversationID.startsWith('GROUP') ? (sequence ? { sequence } : null) : (time ? { time } : null)
  if (anchor && typeof chat.getMessageListHopping === 'function') {
    try {
      // Fetch both sides so the target remains included at SDK page boundaries.
      const context = new Map<string, ChatMessage>()
      for (const direction of [0, 1]) {
        if (cancelled()) return null
        const res = await chat.getMessageListHopping({ conversationID, ...anchor, direction, count: 15 })
        for (const raw of res?.data?.messageList || []) {
          const m = normalizeChatMessage(raw)
          if (m.conversationID === conversationID) context.set(m.ID, m)
        }
      }
      if (cancelled()) return null
      if (context.has(messageID)) return [...context.values()].sort((a, b) => a.time - b.time)
    } catch { /* Older SDKs/quotes can still be located through ordinary pagination. */ }
  }
  if (local) return [local]

  let cursor = ''
  let newer: ChatMessage[] = []
  const visited = new Set<string>()
  while (!cancelled()) {
    const page = await fetchMessageList(conversationID, cursor || undefined)
    if (cancelled()) return null
    const context = [...page.messages, ...newer]
    if (page.messages.some((m) => m.ID === messageID)) return context
    newer = context.slice(0, 20)
    if (page.isCompleted || !page.nextReqMessageID || visited.has(page.nextReqMessageID)) return null
    cursor = page.nextReqMessageID
    visited.add(cursor)
  }
  return null
}
