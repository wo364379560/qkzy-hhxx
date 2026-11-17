/**
 * 消息撤回 — 对齐 LingDoSpace / App
 *
 * 两套方法都写在本文件：
 * - 撤回接口方法（默认）：POST /messages/group/revoke
 * - 撤回 SDK 方法：chat.revokeMessage
 *
 * 入口 revokeMessageSmart 默认走接口（群管撤他人）；
 * 自己撤回仍走 SDK（后台接口只给群主/管理员，且自己撤要出 tip）。
 */
import { chatPost } from '@/api/chatHttp'
import { getChat } from './ImClient'
import type { ChatMessage } from './types'

function messageSeq(message: ChatMessage): number {
  return Number(
    (message as ChatMessage & { sequence?: number; seq?: number; msgSeq?: number }).sequence ??
      (message as ChatMessage & { seq?: number }).seq ??
      (message as ChatMessage & { msgSeq?: number }).msgSeq,
  )
}

function isOwnMessage(message: ChatMessage, selfId: string): boolean {
  const from = String(message.from || '')
  return from === selfId || message.flow === 'out'
}

function withinTwoMinutes(message: ChatMessage): boolean {
  const t = Number(message.time || 0)
  const ts = t < 1e12 ? t * 1000 : t
  return Date.now() - ts <= 120_000
}

// ---------------------------------------------------------------------------
// 撤回接口方法（默认）
// 对齐 App：DoMattxApiClient.doMattxRevokeGroupMessage
// POST /messages/group/revoke  { group_id, msg_seq }
// 群主/管理员撤回他人，无 2 分钟限制；对端静默消失，不插 tip。
// ---------------------------------------------------------------------------
export async function revokeMessageByApi(groupID: string, msgSeq: number) {
  const gid = String(groupID || '').trim()
  const seq = Number(msgSeq)
  if (!gid) throw new Error('缺少 group_id')
  if (!Number.isFinite(seq) || seq <= 0) throw new Error('缺少 msg_seq，无法撤回')
  await chatPost('/messages/group/revoke', { group_id: gid, msg_seq: seq }, true)
}

// ---------------------------------------------------------------------------
// 撤回 SDK 方法
// 对齐 App：TencentImSDKPlugin.revokeMessage
// 自己 2 分钟内撤回，会话显示「你撤回了一条消息」。
// ---------------------------------------------------------------------------
export async function revokeMessageBySdk(message: ChatMessage) {
  const chat = getChat()
  await chat.revokeMessage(message)
}

/** @deprecated 用 revokeMessageByApi */
export const revokeGroupMessageByAdmin = revokeMessageByApi
/** @deprecated 用 revokeMessageBySdk */
export const revokeOwnMessage = revokeMessageBySdk

/** 默认调用接口；自己撤回走 SDK */
export async function revokeMessageSmart(
  message: ChatMessage,
  options: { selfId: string; isGroupAdmin: boolean; groupID?: string },
): Promise<{ silent: boolean }> {
  if (isOwnMessage(message, options.selfId)) {
    if (!withinTwoMinutes(message)) {
      throw new Error('超过 2 分钟，无法撤回')
    }
    await revokeMessageBySdk(message)
    return { silent: false }
  }
  if (!options.isGroupAdmin || !options.groupID) {
    throw new Error('无权撤回他人消息')
  }
  await revokeMessageByApi(options.groupID, messageSeq(message))
  return { silent: true }
}

export function canRevokeMessage(
  message: ChatMessage,
  options: { selfId: string; isGroupAdmin: boolean },
): boolean {
  if (message.isRevoked) return false
  if (message.type === 'TIMGroupTipElem' || message.type === 'TIMGroupSystemNoticeElem') {
    return false
  }
  if (isOwnMessage(message, options.selfId)) return withinTwoMinutes(message)
  return options.isGroupAdmin
}
