import type { ChatMessage } from './types'

export type MessageQuote = {
  messageID: string
  messageSender: string
  messageAbstract: string
  messageTime?: number
  messageSequence?: number
}

const nativeTypes: Record<string, number> = {
  TIMTextElem: 1, TIMCustomElem: 2, TIMImageElem: 3, TIMSoundElem: 4,
  TIMVideoFileElem: 5, TIMFileElem: 6, TIMLocationElem: 7,
  TIMFaceElem: 8, TIMGroupTipElem: 9, TIMRelayElem: 10,
}

export function encodeQuote(quote: ChatMessage, summary: string): string {
  const sender = quote.nick || quote.from || ''
  const messageType = nativeTypes[quote.type] || 1
  const messageTime = quote.time || 0
  const messageSequence = quote.sequence || quote.seq || quote.msgSeq || 0
  return JSON.stringify({
    messageReply: {
      messageID: quote.ID, messageSender: sender, messageAbstract: summary,
      messageType, messageTime, messageSequence, version: 1,
    },
    atomicx_quote_v1: {
      msgID: quote.ID, msgSender: sender, msgAbstract: summary,
      messageType, messageTime, messageSequence,
    },
  })
}

function record(value: unknown): Record<string, unknown> | undefined {
  return value && typeof value === 'object' && !Array.isArray(value)
    ? value as Record<string, unknown> : undefined
}

function text(value: unknown): string {
  return typeof value === 'string' ? value.trim() : ''
}

export function parseQuoteFromMessage(msg: Pick<ChatMessage, 'cloudCustomData'>): MessageQuote | null {
  try {
    const root = record(JSON.parse(msg.cloudCustomData || ''))
    if (!root) return null
    for (const key of ['messageReply', 'atomicx_quote_v1']) {
      const r = record(root[key])
      if (!r) continue
      const native = key === 'atomicx_quote_v1'
      const messageID = text(r[native ? 'msgID' : 'messageID'])
      if (!messageID) continue
      return {
        messageID,
        messageSender: text(r[native ? 'msgSender' : 'messageSender']),
        messageAbstract: text(r[native ? 'msgAbstract' : 'messageAbstract']) || '[消息]',
        messageTime: Number(r.messageTime) || undefined,
        messageSequence: Number(r.messageSequence) || undefined,
      }
    }
  } catch { /* Ordinary messages and malformed metadata have no quote. */ }
  return null
}
