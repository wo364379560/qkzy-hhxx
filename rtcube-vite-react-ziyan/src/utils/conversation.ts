import { parseGiftPayload } from '@/im/stickerCatalog'
import { isAdminInviteNotice, shouldHideGroupCreateNoise, tipPreview } from '@/im/messageTips'
import { largeAttachmentDescription, parseLargeAttachment } from '@/im/largeAttachment'
import type { ConversationItem } from '@/im/types'
import { displayAtAllText, isAtAllUserId, textHasAtAll } from '@/utils/atAll'

export function convTitle(c: ConversationItem): string {
  if (c.remark) return c.remark
  if (c.groupProfile?.name) return c.groupProfile.name
  if (c.userProfile?.nick) return c.userProfile.nick
  return c.groupID || c.userID || c.conversationID
}

export function convAvatar(c: ConversationItem): string {
  return c.groupProfile?.avatar || c.userProfile?.avatar || ''
}

export function convPreview(c: ConversationItem): string {
  const last = c.lastMessage
  if (!last) return ''
  const fake = {
    ID: '',
    type: last.type || '',
    conversationID: c.conversationID || '',
    from: last.from || '',
    to: '',
    flow: 'in' as const,
    time: 0,
    nick: last.nick,
    payload: last.payload,
  }
  if (shouldHideGroupCreateNoise(fake)) return ''
  if (isAdminInviteNotice(fake)) {
    return '邀请成员加入群聊'
  }
  if (last.type === 'TIMTextElem' || String(last.type || '').includes('Text')) {
    const att = parseLargeAttachment({ type: last.type, payload: last.payload })
    if (att) return largeAttachmentDescription(att)
    return displayAtAllText(last.payload?.text || '')
  }
  const tip = tipPreview({
    type: last.type,
    payload: last.payload,
    nick: last.nick,
    from: last.from,
  })
  if (tip) return tip
  if (last.type === 'TIMImageElem') return '[图片]'
  if (last.type === 'TIMSoundElem') return '[语音]'
  if (last.type === 'TIMVideoFileElem') return '[视频]'
  if (last.type === 'TIMFileElem') return '[文件]'
  if (last.type === 'TIMCustomElem') {
    const data = last.payload?.data
    const raw = typeof data === 'string' ? data : data ? JSON.stringify(data) : ''
    const att = parseLargeAttachment({ type: last.type, payload: last.payload })
    if (att) return largeAttachmentDescription(att)
    if (raw.includes('group_create')) return tip || '[建群]'
    const gift = parseGiftPayload(typeof data === 'string' ? data : data)
    if (gift) return `[动图] ${gift.giftName}`
    if (raw.includes('av_call') || raw.includes('语音通话') || raw.includes('视频通话')) {
      return tip || '[通话]'
    }
    const desc = (last.payload as { description?: string } | undefined)?.description || ''
    if (desc && !/自定义消息|custom/i.test(desc)) return desc
    return '[消息]'
  }
  return '[消息]'
}

/** 微信风格：今天 HH:mm / 昨天 HH:mm / 今年 MM-DD / 往年 YYYY-MM-DD */
export function formatConvTime(ts: number): string {
  if (!ts) return ''
  const ms = ts < 1e12 ? ts * 1000 : ts
  const d = new Date(ms)
  if (Number.isNaN(d.getTime())) return ''
  const now = new Date()
  const pad = (n: number) => String(n).padStart(2, '0')
  const hm = `${pad(d.getHours())}:${pad(d.getMinutes())}`
  const startToday = new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime()
  const startThat = new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime()
  const dayDiff = Math.round((startToday - startThat) / 86400000)
  if (dayDiff === 0) return hm
  if (dayDiff === 1) return `昨天 ${hm}`
  if (d.getFullYear() === now.getFullYear()) return `${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

export function convLastTime(c: ConversationItem): number {
  const last = c.lastMessage as { lastTime?: number; time?: number } | undefined
  const extra = c as ConversationItem & { lastTime?: number }
  return Number(last?.lastTime || last?.time || extra.lastTime || 0)
}

function readAtTypes(c: ConversationItem): number[] {
  const list = c.groupAtInfoList || []
  const out: number[] = []
  for (const item of list as Array<Record<string, unknown> | number | string>) {
    if (item == null) continue
    if (typeof item === 'number' || typeof item === 'string') {
      const n = Number(item)
      if (n > 0) out.push(n)
      continue
    }
    const values: unknown[] = [
      item.atType,
      item.groupAtType,
      item.type,
      ...((Array.isArray(item.atTypeList) ? item.atTypeList : []) as unknown[]),
    ]
    for (const v of values) {
      const n = Number(v)
      if (n > 0) {
        out.push(n)
        continue
      }
      const s = String(v || '').toUpperCase()
      if (!s) continue
      if (s.includes('ALL') && s.includes('ME')) out.push(3)
      else if (s.includes('ALL')) out.push(2)
      else if (/(AT)?_?ME\b/.test(s) || s.includes('ATME')) out.push(1)
    }
  }
  return out
}

function lastAtUserList(c: ConversationItem): string[] {
  const p = c.lastMessage?.payload
  const raw = p?.atUserList || p?.userIDList || p?.groupAtUserList || []
  return Array.isArray(raw) ? raw.map((x) => String(x)) : []
}

function lastSenderName(c: ConversationItem): string {
  const last = c.lastMessage
  return String(last?.nameCard || last?.nick || last?.from || '').trim() || '有人'
}

/** 未读群会话：@我 → `[有人@我]`；@全体 → `[@全体成员] 谁 @全体成员` */
export function convAtPreview(
  c: ConversationItem,
  self?: { userId?: string; nick?: string },
): { tag: string; text: string } | null {
  if ((c.unreadCount || 0) <= 0) return null
  if (!String(c.conversationID || '').startsWith('GROUP')) return null

  const types = readAtTypes(c)
  const text = String(c.lastMessage?.payload?.text || '')
  const atList = lastAtUserList(c)
  let hasAtMe = types.includes(1) || types.includes(3)
  let hasAtAll = types.includes(2) || types.includes(3)

  if (!hasAtAll && (atList.some(isAtAllUserId) || textHasAtAll(text))) {
    hasAtAll = true
  }
  const selfId = self?.userId || ''
  const selfNick = self?.nick || ''
  if (!hasAtMe && selfId) {
    if (atList.includes(selfId)) hasAtMe = true
    if (selfNick && text.includes(`@${selfNick}`)) hasAtMe = true
    if (text.includes(`@${selfId}`)) hasAtMe = true
  }
  if (!hasAtMe && !hasAtAll && types.length) hasAtMe = true
  if (!hasAtMe && !hasAtAll) return null

  if (hasAtAll) {
    return { tag: '[@全体成员]', text: `${lastSenderName(c)} @全体成员` }
  }
  return { tag: '[有人@我]', text: convPreview(c) }
}

export function isGroupConversation(conversationID: string) {
  return conversationID.startsWith('GROUP')
}

export function groupIdFromConversation(conversationID: string) {
  return conversationID.replace(/^GROUP/, '')
}

export function peerIdFromConversation(conversationID: string) {
  return conversationID.replace(/^C2C/, '')
}
