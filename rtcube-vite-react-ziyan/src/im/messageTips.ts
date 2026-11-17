/**
 * 群 tip / 系统提示文案
 * - group_create：与 iOS ShangLan_GroupCreateMessageParser 对齐
 * - TIMGroupTipElem：建群后邀请、踢人、退群、公告变更等
 * - 撤回：自己撤回显示 tip；群管撤回他人由业务层静默删
 */
import {
  extractRtcFromMessage,
  rtcBubbleText,
  rtcPreviewText,
  shouldHideRtcInviteMessage,
} from './rtcCall'
import type { ChatMessage } from './types'

export { shouldHideRtcInviteMessage }

export type GroupCreatePayload = {
  businessID: 'group_create'
  cmd: number // 1 社群 · 0 普通群
  opUser?: string
  content?: string
  version?: number
  members?: Array<{ userID?: string; nick?: string }>
}

const GROUP_MEMBER_INVITE_ID = 'mllx_group_member_invite'
const FRESH_GROUP_MS = 20_000
const freshGroupAt = new Map<string, number>()

export function markGroupJustCreated(groupID: string) {
  const id = String(groupID || '').replace(/^GROUP/, '').trim()
  if (id) freshGroupAt.set(id, Date.now())
}

function isFreshlyCreatedGroup(conversationID?: string) {
  const id = String(conversationID || '').replace(/^GROUP/, '').trim()
  const at = freshGroupAt.get(id)
  return Boolean(at && Date.now() - at < FRESH_GROUP_MS)
}

function parseJsonPayload(raw: unknown): Record<string, unknown> | null {
  try {
    const data =
      typeof raw === 'string' ? (JSON.parse(raw) as Record<string, unknown>) : (raw as Record<string, unknown>)
    if (!data || typeof data !== 'object') return null
    return data
  } catch {
    return null
  }
}

export function parseGroupCreate(raw: unknown): GroupCreatePayload | null {
  try {
    const data = parseJsonPayload(raw)
    if (!data || data.businessID !== 'group_create') return null
    const cmd = Number(data.cmd ?? 0)
    const membersRaw = Array.isArray(data.members) ? data.members : []
    const members = membersRaw.flatMap((row) => {
      if (!row || typeof row !== 'object') return []
      const m = row as { userID?: string; nick?: string }
      return [
        {
          userID: typeof m.userID === 'string' ? m.userID : undefined,
          nick: typeof m.nick === 'string' ? m.nick : undefined,
        },
      ]
    })
    return {
      businessID: 'group_create',
      cmd: Number.isFinite(cmd) ? cmd : 0,
      opUser: typeof data.opUser === 'string' ? data.opUser : undefined,
      content: typeof data.content === 'string' ? data.content : undefined,
      version: typeof data.version === 'number' ? data.version : undefined,
      members,
    }
  } catch {
    return null
  }
}

function memberName(m: {
  userID?: string
  nick?: string
  nameCard?: string
  nickName?: string
}): string {
  return (
    m.nameCard?.trim() ||
    m.nick?.trim() ||
    m.nickName?.trim() ||
    m.userID ||
    '成员'
  )
}

function compactNames(names: string[], limit = 3): string {
  if (!names.length) return ''
  if (names.length <= limit) return names.join('、')
  return `${names.slice(0, limit).join('、')}等${names.length}人`
}

function isAdminPlaceholder(name: string): boolean {
  const n = name.trim().toLowerCase()
  return !n || n === 'administrator' || n === 'administrators' || n === 'admin'
}

function displayActor(name: string): string {
  return isAdminPlaceholder(name) ? '管理员' : name
}

function pickNick(raw: unknown): string {
  if (!raw || typeof raw !== 'object') return ''
  const o = raw as { nick?: string; nameCard?: string; nickName?: string; userID?: string }
  return String(o.nick || o.nameCard || o.nickName || o.userID || '').trim()
}

/** 对齐 App：businessID = mllx_group_member_invite */
export function formatGroupMemberInviteTip(msg: ChatMessage): string | null {
  const data = parseJsonPayload(msg.payload?.data)
  const ext = String(msg.payload?.extension || '')
  const isInvite =
    data?.businessID === GROUP_MEMBER_INVITE_ID || ext === GROUP_MEMBER_INVITE_ID
  if (isInvite && data) {
    const content = String(data.content || '').trim()
    if (content) return content
    const inviterName = displayActor(pickNick(data.inviter) || msg.nick || msg.from || '成员')
    const membersRaw = Array.isArray(data.members) ? data.members : []
    const names = compactNames(membersRaw.map(pickNick).filter(Boolean))
    return names ? `${inviterName} 邀请 ${names} 加入群聊` : `${inviterName} 邀请成员加入群聊`
  }
  if (isAdminInviteNotice(msg)) return '邀请成员加入群聊'
  return null
}

/** 后台 REST 拉人会下发 administrator 的「群成员邀请」文本，不当气泡 */
export function isAdminInviteNotice(msg: ChatMessage): boolean {
  const from = String(msg.from || '')
  const nick = String(msg.nick || '')
  if (!isAdminPlaceholder(from) && !isAdminPlaceholder(nick)) return false
  const text = String(msg.payload?.text || msg.payload?.description || '').trim()
  return text === '群成员邀请' || text === '邀请成员入群' || text === '邀请加入群聊'
}

/** Web SDK TIMGroupTipElem.operationType 常用值 */
const TIP = {
  JOIN: 1,
  QUIT: 2,
  KICKED: 3,
  SET_ADMIN: 4,
  CANCEL_ADMIN: 5,
  GROUP_PROFILE: 6,
  MEMBER_PROFILE: 7,
} as const

export function formatGroupTipMessage(msg: ChatMessage): string | null {
  if (msg.type !== 'TIMGroupTipElem' && msg.type !== 'TIMGroupSystemNoticeElem') {
    return null
  }
  const p = (msg.payload || {}) as {
    operationType?: number
    groupJoinType?: number
    operatorID?: string
    operatorInfo?: { userID?: string; nick?: string; nameCard?: string; nickName?: string }
    userIDList?: string[]
    memberList?: Array<{
      userID?: string
      nick?: string
      nameCard?: string
      nickName?: string
      muteTime?: number
      muteUntil?: number
    }>
    groupProfile?: {
      groupName?: string
      notification?: string
      ownerID?: string
      muteAllMembers?: boolean | string
    }
    newGroupProfile?: {
      groupName?: string
      notification?: string
      ownerID?: string
      muteAllMembers?: boolean | string
    }
    text?: string
  }

  const rawText = String(p.text || '').trim()
  if (/创建了?\s*(社群|群组)/.test(rawText)) return null
  if (rawText && rawText !== '群成员邀请') return rawText

  const opType = Number(p.operationType || 0)
  const opName = displayActor(
    p.operatorInfo
      ? memberName({ ...p.operatorInfo, userID: p.operatorInfo.userID || p.operatorID })
      : p.operatorID || msg.nick || msg.from || '有人',
  )
  const memberRows = p.memberList || []
  const members = compactNames(memberRows.map(memberName).filter(Boolean))
  const ids = compactNames((p.userIDList || []).map((id) => id.trim()).filter(Boolean))
  const who = members || ids

  switch (opType) {
    case TIP.JOIN:
      // Web SDK 中“邀请入群”仍是 JOIN(1)，groupJoinType=2 才表示被邀请。
      if (Number(p.groupJoinType || 0) === 2) {
        return who ? `${opName}邀请了${who}进群` : `${opName}邀请成员进群`
      }
      return who ? `${who}加入了群聊` : '新成员加入了群聊'
    case TIP.QUIT:
      return who ? `${who}退出了群聊` : `${opName}退出了群聊`
    case TIP.KICKED:
      return who ? `${opName}把${who}移出了群` : `${opName}移出了成员`
    case TIP.SET_ADMIN:
      return who ? `${opName}设置${who}为管理员` : `${opName}设置了管理员`
    case TIP.CANCEL_ADMIN:
      return who ? `${opName}取消了${who}的管理员` : `${opName}取消了管理员`
    case TIP.GROUP_PROFILE: {
      const g = p.newGroupProfile || p.groupProfile || {}
      const muteAll = g.muteAllMembers
      if (muteAll !== undefined) {
        const on = muteAll === true || muteAll === 'On' || muteAll === 'on' || muteAll === '1'
        return on ? `${opName}开启了全员禁言` : `${opName}取消了全员禁言`
      }
      if (g.notification !== undefined) {
        const notice = String(g.notification || '').trim()
        return notice
          ? `${opName}更新了群公告：${notice.slice(0, 40)}${notice.length > 40 ? '…' : ''}`
          : `${opName}更新了群公告`
      }
      if (g.groupName) return `${opName}修改群名为「${g.groupName}」`
      return null
    }
    case TIP.MEMBER_PROFILE: {
      const muted = memberRows.filter((m) => Number(m.muteTime || 0) > 0)
      const unmuted = memberRows.filter(
        (m) => m.muteTime !== undefined && Number(m.muteTime || 0) <= 0,
      )
      if (muted.length) {
        const names = compactNames(muted.map(memberName).filter(Boolean)) || who
        return names ? `${opName}禁言了${names}` : `${opName}禁言了成员`
      }
      if (unmuted.length) {
        const names = compactNames(unmuted.map(memberName).filter(Boolean)) || who
        return names ? `${opName}取消了${names}的禁言` : `${opName}取消了禁言`
      }
      return `${opName}修改了群名片`
    }
    default:
      return `${opName}进行了群操作`
  }
}

export function formatGroupCreateTip(msg: ChatMessage, selfId = ''): string | null {
  const payload = parseGroupCreate(msg.payload?.data)
  if (!payload) return null
  const from = String(msg.from || payload.opUser || '')
  const actor =
    selfId && (from === selfId || payload.opUser === selfId)
      ? '你'
      : msg.nick?.trim() || payload.opUser || msg.from || '有人'
  const names = compactNames(
    (payload.members || [])
      .map((m) => String(m.nick || m.userID || '').trim())
      .filter(Boolean),
  )
  if (names) return `${actor}邀请了${names}加入群聊`
  return `${actor}邀请成员加入群聊`
}

/** 建群瞬间 SDK 会连发「更新群资料 / 创建社群 / 入群」，只保留一条邀请 tip */
export function shouldHideGroupCreateNoise(msg: ChatMessage): boolean {
  if (parseGroupCreate(msg.payload?.data)) return false
  const desc = String(msg.payload?.description || '').trim()
  if (/^创建(社群|群组)$/.test(desc)) return true
  const type = String(msg.type || '')
  const isTip = type === 'TIMGroupTipElem' || type === 'TIMGroupSystemNoticeElem' || type.includes('GRP_TIP')
  if (!isTip) return false
  const p = (msg.payload || {}) as {
    operationType?: number
    text?: string
    groupProfile?: { groupName?: string; notification?: string; muteAllMembers?: boolean | string }
    newGroupProfile?: { groupName?: string; notification?: string; muteAllMembers?: boolean | string }
  }
  const rawText = String(p.text || '').trim()
  if (/创建了?\s*(社群|群组)/.test(rawText)) return true
  const opType = Number(p.operationType || 0)
  if (opType === TIP.GROUP_PROFILE) {
    const g = p.newGroupProfile || p.groupProfile || {}
    if (g.groupName || g.notification !== undefined || g.muteAllMembers !== undefined) return false
    return true
  }
  if (opType === TIP.JOIN && isFreshlyCreatedGroup(msg.conversationID)) return true
  return false
}

export function shouldHideDuplicateGroupCreate(msg: ChatMessage, list: ChatMessage[]): boolean {
  const mine = parseGroupCreate(msg.payload?.data)
  if (!mine) return false
  const creates = list.filter((m) => parseGroupCreate(m.payload?.data))
  if (creates.length <= 1) return false
  const withMembers = creates.filter((m) => (parseGroupCreate(m.payload?.data)?.members || []).length > 0)
  const keep = withMembers[withMembers.length - 1] || creates[creates.length - 1]
  return Boolean(keep && msg.ID !== keep.ID)
}

export function formatRevokeTip(msg: ChatMessage, selfId: string): string {
  const revoker = String((msg as ChatMessage & { revoker?: string }).revoker || msg.from || '')
  const from = String(msg.from || '')
  const revokerName = !revoker || revoker === selfId ? '你' : msg.nick && revoker === from ? msg.nick : revoker
  const targetName = !from || from === selfId ? '你' : msg.nick || from
  if (revoker && from && revoker !== from) {
    return `${revokerName}撤回了${targetName}一条消息`
  }
  if (revoker === selfId || from === selfId || msg.flow === 'out') {
    return '你撤回了一条消息'
  }
  return `${revokerName}撤回了一条消息`
}

/** 居中 tip 文案；非 tip 返回 null（通话 av_call 走气泡，不当 tip） */
export function resolveTipText(msg: ChatMessage, selfId: string): string | null {
  if (msg.isRevoked) return formatRevokeTip(msg, selfId)

  const createTip = formatGroupCreateTip(msg, selfId)
  if (createTip) return createTip

  if (shouldHideGroupCreateNoise(msg)) return null

  const inviteTip = formatGroupMemberInviteTip(msg)
  if (inviteTip) return inviteTip

  const groupTip = formatGroupTipMessage(msg)
  if (groupTip) return groupTip

  // TIMCustomElem：建群 tip 已在上面处理；av_call / 礼物等走气泡
  return null
}

export function isCenteredTipMessage(msg: ChatMessage): boolean {
  if (shouldHideGroupCreateNoise(msg)) return false
  if (msg.isRevoked) return true
  if (msg.type === 'TIMGroupTipElem' || msg.type === 'TIMGroupSystemNoticeElem') return true
  if (formatGroupMemberInviteTip(msg)) return true
  if (isAdminInviteNotice(msg)) return true
  if (msg.type === 'TIMCustomElem' || String(msg.type || '').includes('Custom')) {
    if (parseGroupCreate(msg.payload?.data)) return true
    // 通话摘要对齐 LingDoSpace：普通聊天气泡，非居中 tip
    if (extractRtcFromMessage(msg)) return false
  }
  return false
}

export function tipPreview(msg: {
  type?: string
  payload?: ChatMessage['payload']
  nick?: string
  from?: string
}): string {
  const fake = {
    ID: '',
    type: msg.type || '',
    conversationID: '',
    from: msg.from || '',
    to: '',
    flow: 'in' as const,
    time: 0,
    nick: msg.nick,
    payload: msg.payload,
  }
  const tip = resolveTipText(fake, '')
  if (tip) return tip
  if (msg.type === 'TIMCustomElem' || String(msg.type || '').includes('Custom')) {
    const rtc = extractRtcFromMessage(fake)
    if (rtc) {
      if (rtc.action === 'invite') return rtcPreviewText(rtc)
      return rtcBubbleText(rtc, false)
    }
    if (parseGroupCreate(msg.payload?.data)) return '[建群]'
    const desc = msg.payload?.description || ''
    if (desc && !/自定义消息|custom/i.test(desc)) return desc
    return '[消息]'
  }
  return ''
}
