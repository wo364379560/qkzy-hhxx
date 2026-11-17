/**
 * 消息操作：复制 / 删除 / 逐条转发 / 合并转发 / 引用摘要
 */
import { displayAtAllText } from '@/utils/atAll'
import { formatSendError, getChat, isFriendSendError, TencentCloudChat } from './ImClient'
import { ensureC2CCanSend } from './friends'
import { largeAttachmentDescription, parseLargeAttachment } from './largeAttachment'
import {
  fileInfoFromMessage,
  isFileMessage,
  isImageMessage,
  isMergerMessage,
  isSoundMessage,
  isVideoMessage,
  mergerInfoFromMessage,
} from './mediaPayload'
import { tipPreview } from './messageTips'
import { extractRtcFromMessage, rtcBubbleText } from './rtcCall'
import type { ChatMessage } from './types'
import { encodeQuote } from './quote'
import { emojiWireText } from './emojiWire'

// eslint-disable-next-line @typescript-eslint/no-explicit-any
type AnyChat = any

export const MSG_SELECT_MAX = 30

export function messageCopyText(msg: ChatMessage): string {
  const att = parseLargeAttachment(msg)
  if (att) return largeAttachmentDescription(att)
  if (isMergerMessage(msg)) return `[聊天记录] ${mergerInfoFromMessage(msg).title || ''}`.trim()
  if (isImageMessage(msg)) return '[图片]'
  if (isSoundMessage(msg)) return '[语音]'
  if (isVideoMessage(msg)) return '[视频]'
  if (isFileMessage(msg)) {
    const name = fileInfoFromMessage(msg).name
    return name && name !== '文件' ? `[文件] ${name}` : '[文件]'
  }
  if (msg.type === 'TIMTextElem' || String(msg.type || '').includes('Text')) {
    return displayAtAllText(msg.payload?.text || '')
  }
  const rtc = extractRtcFromMessage(msg)
  if (rtc) return rtcBubbleText(rtc)
  const tip = tipPreview(msg)
  if (tip) return tip
  return msg.payload?.description || msg.payload?.text || ''
}

export function messageAbstractLine(msg: ChatMessage): string {
  const name = msg.nick || msg.from || ''
  const body = messageCopyText(msg) || '[消息]'
  return name ? `${name}: ${body}` : body
}

export async function deleteMessagesLocal(messages: ChatMessage[]) {
  if (!messages.length) return
  const chat = getChat()
  await chat.deleteMessage(messages)
}

function peerFromConversation(conversationID: string) {
  const isGroup = conversationID.startsWith('GROUP')
  return {
    isGroup,
    to: conversationID.replace(/^(C2C|GROUP)/, ''),
  }
}

async function sendCreatedWithFriendRetry(
  conversationID: string,
  create: () => unknown,
) {
  const chat = getChat()
  const { isGroup, to } = peerFromConversation(conversationID)
  if (!isGroup && to) {
    await ensureC2CCanSend(to)
  }
  try {
    await chat.sendMessage(create())
  } catch (e) {
    if (!isGroup && to && isFriendSendError(e)) {
      await ensureC2CCanSend(to)
      await new Promise((r) => window.setTimeout(r, 400))
      try {
        await chat.sendMessage(create())
        return
      } catch (e2) {
        throw new Error(formatSendError(e2))
      }
    }
    throw new Error(formatSendError(e))
  }
}

export async function forwardMessageOne(
  source: ChatMessage,
  targetConversationID: string,
) {
  const chat = getChat()
  const TYPES = (TencentCloudChat as AnyChat).TYPES
  const { isGroup, to } = peerFromConversation(targetConversationID)
  await sendCreatedWithFriendRetry(targetConversationID, () =>
    chat.createForwardMessage({
      to,
      conversationType: isGroup ? TYPES.CONV_GROUP : TYPES.CONV_C2C,
      payload: source,
    }),
  )
}

export async function forwardMessagesSeparate(
  messages: ChatMessage[],
  targetIds: string[],
) {
  const list = sortedMessages(messages)
  if (!list.length) throw new Error('没有可转发的消息')
  for (const target of targetIds) {
    for (const msg of list) {
      await forwardMessageOne(msg, target)
    }
  }
}

function sortedMessages(messages: ChatMessage[]) {
  return [...messages]
    .filter((m) => !m.isRevoked)
    .sort((a, b) => (Number(a.time) || 0) - (Number(b.time) || 0))
    .slice(0, MSG_SELECT_MAX)
}

export async function forwardMessagesMerged(
  messages: ChatMessage[],
  targetIds: string[],
  title: string,
) {
  const list = sortedMessages(messages)
  if (!list.length) throw new Error('没有可转发的消息')
  const chat = getChat()
  if (typeof chat.createMergerMessage !== 'function') {
    await forwardMessagesSeparate(list, targetIds)
    return
  }
  const TYPES = (TencentCloudChat as AnyChat).TYPES
  const abstracts = list.slice(0, 4).map(messageAbstractLine)
  for (const target of targetIds) {
    const { isGroup, to } = peerFromConversation(target)
    await sendCreatedWithFriendRetry(target, () =>
      chat.createMergerMessage({
        to,
        conversationType: isGroup ? TYPES.CONV_GROUP : TYPES.CONV_C2C,
        payload: {
          messageList: list,
          title: title || '聊天记录',
          abstractList: abstracts,
          compatibleText: '[聊天记录]',
        },
        needReadReceipt: false,
      }),
    )
  }
}

/** Emit both the native AtomicX and web/TUIKit quote envelopes. */
export function buildQuoteCloudCustomData(quote: ChatMessage): string {
  return encodeQuote(quote, emojiWireText(messageCopyText(quote)).slice(0, 100))
}

export { parseQuoteFromMessage } from './quote'
