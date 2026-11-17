import { getChat, TencentCloudChat } from './ImClient'
import type { ChatMessage } from './types'

type ReceiptMessage = Partial<ChatMessage> & { id?: string; messageID?: string; userID?: string; groupID?: string }
export type PeerReadUpdate = { messageID: string; conversationID?: string; isPeerRead: boolean }

function msgId(m: ReceiptMessage) {
  return String(m?.ID || m?.id || m?.messageID || '')
}

export function isMessagePeerRead(raw: Partial<ChatMessage> | undefined) {
  return raw?.isPeerRead === true || raw?.readReceiptInfo?.isPeerRead === true
}

function sdkMessage(m: ChatMessage): ChatMessage {
  return getChat().findMessage?.(m.ID) || m
}

function batchesByConversation(messages: ChatMessage[]) {
  const groups = new Map<string, ChatMessage[]>()
  for (const m of messages) {
    const list = groups.get(m.conversationID) || []
    list.push(sdkMessage(m))
    groups.set(m.conversationID, list)
  }
  return [...groups.values()].flatMap((list) => {
    const batches: ChatMessage[][] = []
    for (let i = 0; i < list.length; i += 30) batches.push(list.slice(i, i + 30))
    return batches
  })
}

export async function sendReadReceipts(incoming: ChatMessage[]) {
  const c = getChat()
  if (typeof c.sendMessageReadReceipt !== 'function') return
  const self = String(c.getLoginUser?.() || '')
  const seen = new Set<string>()
  const targets = incoming.filter((m) => {
    if (!m || m.isRevoked || m.from === self || m.flow === 'out') return false
    if (!m.needReadReceipt) return false
    const id = msgId(m)
    if (!id || seen.has(id)) return false
    seen.add(id)
    return true
  })
  for (const batch of batchesByConversation(targets)) {
    try { await c.sendMessageReadReceipt(batch) } catch { /* Optional receipt service. */ }
  }
}

export async function hydratePeerRead(messages: ChatMessage[]) {
  const c = getChat()
  const self = String(c.getLoginUser?.() || '')
  const mine = messages.filter((m) => (m.from === self || m.flow === 'out') && !m.isRevoked && m.needReadReceipt)
  const updates = new Map<string, ChatMessage>()
  if (typeof c.getMessageReadReceiptList === 'function') {
    for (const batch of batchesByConversation(mine)) {
      try {
        const res = await c.getMessageReadReceiptList(batch)
        for (const m of res?.data?.messageList || []) updates.set(msgId(m), m)
      } catch { /* Keep the basic C2C read state when advanced receipts are unavailable. */ }
    }
  }
  return messages.map((m) => {
    const updated = updates.get(m.ID) || sdkMessage(m)
    return {
      ...m,
      readReceiptInfo: updated.readReceiptInfo || m.readReceiptInfo,
      isPeerRead: isMessagePeerRead(m) || isMessagePeerRead(updated),
    }
  })
}

export function applyPeerReadUpdates(messages: ChatMessage[], updates: PeerReadUpdate[]) {
  return messages.map((m) => {
    if (!m.conversationID.startsWith('C2C') || m.flow !== 'out') return m
    const read = isMessagePeerRead(m) || isMessagePeerRead(sdkMessage(m)) || updates.some((u) =>
      u.isPeerRead && u.messageID === m.ID && (!u.conversationID || u.conversationID === m.conversationID),
    )
    return read && !m.isPeerRead ? { ...m, isPeerRead: true } : m
  })
}

export function onMessageReadByPeer(cb: (updates: PeerReadUpdate[]) => void) {
  const c = getChat()
  const events = TencentCloudChat.EVENT
  const handlers = [
    { name: events.MESSAGE_READ_BY_PEER || 'onMessageReadByPeer', legacy: true },
    { name: events.MESSAGE_READ_RECEIPT_RECEIVED || 'onMessageReadReceiptReceived', legacy: false },
  ].map(({ name, legacy }) => {
    const handler = (event: { data?: unknown }) => {
      const list: ReceiptMessage[] = Array.isArray(event?.data) ? event.data : []
      cb(list.filter((m) => m && msgId(m)).map((m) => ({
        messageID: msgId(m),
        conversationID: m.conversationID || (m.userID ? `C2C${m.userID}` : m.groupID ? `GROUP${m.groupID}` : undefined),
        isPeerRead: legacy || isMessagePeerRead(m),
      })))
    }
    c.on?.(name, handler)
    return { name, handler }
  })
  return () => handlers.forEach(({ name, handler }) => c.off?.(name, handler))
}
