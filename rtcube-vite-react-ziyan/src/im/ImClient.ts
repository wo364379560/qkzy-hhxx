/**
 * 官方 Chat SDK 薄封装 — 禁止依赖 TUIKit / TUICallKit。
 * 大群：成员/消息务必走分页与窗口上限（docs/IM_SPEC.md §4 §7）。
 *
 * 腾讯侧群成员：getGroupMemberList({ groupID, offset, count })
 * - count 建议 50～100，SDK 单次上限通常 100
 * - 回包 data.offset 为下一页偏移；禁止一次拉全员
 */
// eslint-disable-next-line @typescript-eslint/no-explicit-any
import TencentCloudChat from '@tencentcloud/chat'
import TIMUploadPlugin from 'tim-upload-plugin'
import { emojiWireText } from './emojiWire'
import {
  AT_MEMBER_PAGE_SIZE,
  GROUP_MEMBER_FIRST_PAGE,
  GROUP_MEMBER_PAGE_SIZE,
  MESSAGE_WINDOW_LIMIT,
  SDK_APP_ID,
  setRuntimeSDKAppID,
} from '@/config/runtime'
import type { ChatMessage, ConversationItem, GroupMember } from './types'
import { playSound } from '@/audio/sounds'
import { displayAtAllText, isAtAllUserId, textHasAtAll } from '@/utils/atAll'

function isMutedRemind(remind?: string | number | boolean): boolean {
  if (remind === true) return true
  const s = String(remind || '')
  return /ACCEPT_NOT_NOTIFY|NOT_NOTIFY|Discard|MSG_REMIND_ACPT_NOT_NOTE/i.test(s)
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
type AnyChat = any

let chat: AnyChat | null = null
let ready = false
let boundSdkAppId = 0
let loggedInUser = ''

function ensureUploadPlugin(instance: AnyChat) {
  try {
    const mod = TIMUploadPlugin as { default?: unknown }
    const plugin = mod?.default ?? TIMUploadPlugin
    instance.registerPlugin?.({ 'tim-upload-plugin': plugin })
  } catch {
    /* 重复注册忽略 */
  }
}

export function getChat(sdkAppID?: number): AnyChat {
  const id = sdkAppID && sdkAppID > 0 ? sdkAppID : SDK_APP_ID
  if (!id) {
    throw new Error('SDKAppID 未就绪，请先登录拿回包 sdk_app_id')
  }
  if (!chat || boundSdkAppId !== id) {
    chat = (TencentCloudChat as AnyChat).create({ SDKAppID: id })
    ensureUploadPlugin(chat)
    chat.setLogLevel?.(1)
    boundSdkAppId = id
    ready = false
    loggedInUser = ''
  } else {
    ensureUploadPlugin(chat)
  }
  return chat
}

export function isImReady() {
  return ready
}

export function currentImUserId() {
  return loggedInUser
}

/**
 * 登录后必须等 SDK_READY（TCP/WS 真正连上）才算 ready。
 * 对照 web-h5：sdkAppID 必须用登录回包，禁止用错 AppID 建实例。
 */
export async function imLogin(userID: string, userSig: string, sdkAppId?: number) {
  if (!userID || !userSig) throw new Error('缺少 userID / userSig')
  if (!sdkAppId || sdkAppId <= 0) throw new Error('缺少 sdk_app_id，无法登录 IM')

  setRuntimeSDKAppID(sdkAppId)
  const c = getChat(sdkAppId)
  const EVENT = (TencentCloudChat as AnyChat).EVENT

  if (ready && loggedInUser === userID && c.isReady?.()) {
    return c
  }
  if (loggedInUser && loggedInUser !== userID) {
    try {
      await c.logout()
    } catch {
      /* ignore */
    }
    ready = false
    loggedInUser = ''
  }

  await new Promise<void>((resolve, reject) => {
    let settled = false
    const cleanup = () => {
      window.clearTimeout(timer)
      c.off(EVENT.SDK_READY, onReady)
    }
    const finishOk = () => {
      if (settled) return
      settled = true
      cleanup()
      ready = true
      loggedInUser = userID
      resolve()
    }
    const finishErr = (err: Error) => {
      if (settled) return
      settled = true
      cleanup()
      ready = false
      reject(err)
    }
    const onReady = () => finishOk()
    const timer = window.setTimeout(() => {
      finishErr(new Error('IM 连接超时（SDK_READY），请检查网络 / SDKAppID / UserSig'))
    }, 20000)

    c.on(EVENT.SDK_READY, onReady)

    c.login({ userID, userSig })
      .then((res: { code?: number; message?: string }) => {
        if (res?.code !== undefined && res.code !== 0) {
          finishErr(new Error(res.message || `IM 登录失败(${res.code})`))
          return
        }
        if (c.isReady?.()) finishOk()
      })
      .catch((e: unknown) => {
        finishErr(e instanceof Error ? e : new Error('IM 登录失败'))
      })
  })

  return c
}

export async function imLogout() {
  if (!chat) return
  try {
    await chat.logout()
  } finally {
    ready = false
    loggedInUser = ''
  }
}

export function onSdkReady(cb: () => void) {
  const c = getChat()
  const EVENT = (TencentCloudChat as AnyChat).EVENT
  if (ready || c.isReady?.()) {
    queueMicrotask(cb)
  }
  c.on(EVENT.SDK_READY, cb)
  return () => c.off(EVENT.SDK_READY, cb)
}

export function onConversationListUpdated(cb: (list: ConversationItem[]) => void) {
  const c = getChat()
  const EVENT = (TencentCloudChat as AnyChat).EVENT
  const handler = (event: { data: ConversationItem[] }) => cb(event.data || [])
  c.on(EVENT.CONVERSATION_LIST_UPDATED, handler)
  return () => c.off(EVENT.CONVERSATION_LIST_UPDATED, handler)
}

export function onMessageReceived(cb: (list: ChatMessage[]) => void) {
  const c = getChat()
  const EVENT = (TencentCloudChat as AnyChat).EVENT
  const handler = (event: { data: ChatMessage[] }) =>
    cb((event.data || []).map(normalizeChatMessage))
  c.on(EVENT.MESSAGE_RECEIVED, handler)
  return () => c.off(EVENT.MESSAGE_RECEIVED, handler)
}

export function onMessageRevoked(cb: (list: ChatMessage[]) => void) {
  const c = getChat()
  const EVENT = (TencentCloudChat as AnyChat).EVENT
  const evt = EVENT.MESSAGE_REVOKED || EVENT.MESSAGE_REVOKED_BY_PEER
  if (!evt) return () => {}
  const handler = (event: { data: ChatMessage[] }) => cb(event.data || [])
  c.on(evt, handler)
  return () => c.off(evt, handler)
}

/** 腾讯 IM 系统通知会话，不能当普通群打开/发消息 */
export function isHiddenConversation(item: {
  conversationID?: string
  groupID?: string
  type?: string | number
}) {
  const id = String(item.conversationID || '')
  const gid = String(item.groupID || '')
  const type = String(item.type || '')
  return (
    id === '@TIM#SYSTEM' ||
    gid === '@TIM#SYSTEM' ||
    type === '@TIM#SYSTEM' ||
    /@TIM#SYSTEM/i.test(id) ||
    /@TIM#SYSTEM/i.test(gid)
  )
}

function normalizeConversationList(raw: unknown): ConversationItem[] {
  const list = (Array.isArray(raw) ? raw : []) as ConversationItem[]
  return list
    .map((item) => {
      const conversationID = String(
        item.conversationID ||
          (item.groupID ? `GROUP${item.groupID}` : '') ||
          (item.userID ? `C2C${item.userID}` : ''),
      )
      if (!conversationID) return null
      const userID =
        item.userID ||
        (conversationID.startsWith('C2C') ? conversationID.slice(3) : undefined)
      const groupID =
        item.groupID ||
        (conversationID.startsWith('GROUP') ? conversationID.slice(5) : undefined)
      const remind = item.messageRemindType
      const isMuted = Boolean(item.isMuted) || isMutedRemind(remind)
      const mapped = {
        ...item,
        conversationID,
        userID,
        groupID,
        unreadCount: Number(item.unreadCount || 0),
        type: item.type || (conversationID.startsWith('GROUP') ? 'GROUP' : 'C2C'),
        messageRemindType: remind,
        isMuted,
        draftText: String(item.draftText || ''),
        groupAtInfoList: Array.isArray(item.groupAtInfoList) ? item.groupAtInfoList : [],
      } as ConversationItem
      return isHiddenConversation(mapped) ? null : mapped
    })
    .filter(Boolean) as ConversationItem[]
}

export async function fetchConversationList(): Promise<ConversationItem[]> {
  const c = getChat()
  const res = await c.getConversationList()
  const data = res?.data || {}
  const raw = data.conversationList ?? (Array.isArray(data) ? data : [])
  return normalizeConversationList(raw)
}

export type UserProfileLite = {
  userID: string
  nick: string
  avatar: string
}

/** 批量拉资料，补群消息头像/昵称（对照 web-h5） */
export async function fetchUserProfiles(userIDs: string[]): Promise<UserProfileLite[]> {
  if (!userIDs.length) return []
  const c = getChat()
  const res = await c.getUserProfile({ userIDList: userIDs.slice(0, 100) })
  const list = (res?.data ?? []) as AnyChat[]
  return list.map((u) => ({
    userID: String(u.userID || ''),
    nick: String(u.nick || u.userID || ''),
    avatar: String(u.avatar || ''),
  }))
}

export type FriendRow = {
  userID: string
  nick: string
  remark: string
  avatar: string
}

/** 对照 web-h5：SDK getFriendList */
export async function fetchFriendList(): Promise<FriendRow[]> {
  const c = getChat()
  const res = await c.getFriendList()
  const raw = res?.data
  const list = (Array.isArray(raw) ? raw : raw?.friendList ?? []) as AnyChat[]
  return list
    .map((f) => {
      const p = f.profile || f
      return {
        userID: String(f.userID || p.userID || ''),
        nick: String(p.nick || f.userID || ''),
        remark: String(f.remark || ''),
        avatar: String(p.avatar || ''),
      }
    })
    .filter((f) => f.userID)
}

/** 统一补齐群聊展示用的昵称/头像（SDK 字段偶发缺失） */
export function normalizeChatMessage(raw: AnyChat): ChatMessage {
  const nick =
    raw?.nick ||
    raw?.nameCard ||
    raw?.nickName ||
    raw?.fromAccountNick ||
    raw?.from ||
    ''
  const avatar = raw?.avatar || raw?.faceUrl || raw?.fromAccountAvatar || ''
  return {
    ...(raw as ChatMessage),
    cloudCustomData: raw?.cloudCustomData,
    needReadReceipt: raw?.needReadReceipt,
    readReceiptInfo: raw?.readReceiptInfo,
    nick: String(nick),
    avatar: String(avatar || ''),
    isPeerRead: Boolean(raw?.isPeerRead || raw?.readReceiptInfo?.isPeerRead),
  }
}

/** 拉取历史并裁剪窗口，防大群 OOM。SDK 单次最多 20 条，更早的靠 nextReqMessageID 翻页。 */
export async function fetchMessageList(
  conversationID: string,
  nextReqMessageID?: string,
): Promise<{ messages: ChatMessage[]; nextReqMessageID: string; isCompleted: boolean }> {
  const c = getChat()
  const res = await c.getMessageList({
    conversationID,
    nextReqMessageID: nextReqMessageID || undefined,
    count: 20,
  })
  let list: ChatMessage[] = (res?.data?.messageList ?? []).map(normalizeChatMessage)
  if (list.length > MESSAGE_WINDOW_LIMIT) {
    list = list.slice(list.length - MESSAGE_WINDOW_LIMIT)
  }
  const sdkCompleted = Boolean(res?.data?.isCompleted)
  let next = String(res?.data?.nextReqMessageID || '')
  if (!next && !sdkCompleted && list.length >= 20) {
    const oldest = list.reduce((a, b) =>
      (Number(a.time) || 0) <= (Number(b.time) || 0) ? a : b,
    )
    next = String(oldest.ID || '')
  }
  const completed = sdkCompleted || !next
  return {
    messages: list,
    nextReqMessageID: next,
    isCompleted: completed,
  }
}

export function trimMessageWindow(list: ChatMessage[]): ChatMessage[] {
  if (list.length <= MESSAGE_WINDOW_LIMIT) return list
  return list.slice(list.length - MESSAGE_WINDOW_LIMIT)
}

export async function sendTextMessage(
  conversationID: string,
  text: string,
  atUserList?: string[],
  cloudCustomData?: string,
) {
  const c = getChat()
  const TYPES = (TencentCloudChat as AnyChat).TYPES
  const isGroup = conversationID.startsWith('GROUP')
  const to = conversationID.replace(/^(C2C|GROUP)/, '')
  const atAllToken = TYPES?.MSG_AT_ALL || '__kImSDK_MesssageAtALL__'
  const body = emojiWireText(displayAtAllText(text))
  const atList = [...(atUserList || [])]
  if (isGroup && textHasAtAll(text) && !atList.some(isAtAllUserId)) {
    atList.push(atAllToken)
  }
  const ccd =
    cloudCustomData ||
    (isGroup && atList.length ? JSON.stringify({ atUserList: atList }) : undefined)
  const message = c.createTextMessage({
    to,
    conversationType: isGroup ? TYPES.CONV_GROUP : TYPES.CONV_C2C,
    payload: { text: body },
    needReadReceipt: !isGroup,
    ...(ccd ? { cloudCustomData: ccd } : {}),
  })
  if (isGroup && atList.length) {
    try {
      message.payload = {
        ...message.payload,
        text: body,
      }
      if (typeof c.createTextAtMessage === 'function') {
        const atMsg = c.createTextAtMessage({
          to,
          conversationType: TYPES.CONV_GROUP,
          payload: {
            text: body,
            atUserList: atList,
          },
          needReadReceipt: false,
          ...(cloudCustomData ? { cloudCustomData } : {}),
        })
        if (atMsg?.payload) atMsg.payload.text = body
        try {
          const res = await c.sendMessage(atMsg)
          playSound('newmsg')
          const sent = res?.data?.message as ChatMessage
          if (sent?.payload) sent.payload.text = body
          return sent
        } catch (e) {
          throw new Error(formatSendError(e))
        }
      }
    } catch (e) {
      if (e instanceof Error && e.message.includes('好友')) throw e
      /* @ 消息创建失败则走普通文本 */
    }
  }
  try {
    const res = await c.sendMessage(message)
    playSound('newmsg')
    const sent = res?.data?.message as ChatMessage
    if (sent?.payload?.text) sent.payload.text = displayAtAllText(sent.payload.text)
    return sent
  } catch (e) {
    throw new Error(formatSendError(e))
  }
}

export function isFriendSendError(err: unknown): boolean {
  const extra = err as { code?: number; message?: string }
  const raw = extra?.message || (err instanceof Error ? err.message : String(err || ''))
  const lower = raw.toLowerCase()
  const code = Number(extra?.code) || Number(/(\d{4,5})/.exec(raw)?.[1]) || 0
  return (
    code === 20009 ||
    lower.includes('not friends') ||
    lower.includes('are not friend') ||
    lower.includes('friend relation check')
  )
}

export function formatSendError(err: unknown): string {
  const extra = err as { code?: number; message?: string }
  const raw = extra?.message || (err instanceof Error ? err.message : String(err || ''))
  const lower = raw.toLowerCase()
  const code = Number(extra?.code) || Number(/(\d{4,5})/.exec(raw)?.[1]) || 0

  if (isFriendSendError(err)) {
    return '对方还不是你的好友，请先添加好友后再发消息'
  }
  if (code === 20007 || lower.includes('in blacklist') || lower.includes('in black list')) {
    return '发送失败：消息被拦截，可能已被对方拉黑'
  }
  if (code === 10017 || lower.includes('forbidden to speak') || raw.includes('被禁言')) {
    return '您已被禁言，无法发送消息'
  }
  if (
    code === 7005 ||
    raw.includes('104857600') ||
    lower.includes('file size exceeds limit') ||
    raw.includes('文件大小超出') ||
    raw.includes('最大限制是100')
  ) {
    return '发送失败，请稍后重试'
  }
  if (/[\u4e00-\u9fff]/.test(raw)) return raw.startsWith('发送') ? raw : raw
  return code ? `发送失败，请稍后重试（${code}）` : raw || '发送失败，请稍后重试'
}

export async function sendCustomJsonMessage(
  conversationID: string,
  payload: Record<string, unknown>,
  description = '',
) {
  const c = getChat()
  const TYPES = (TencentCloudChat as AnyChat).TYPES
  const isGroup = conversationID.startsWith('GROUP')
  const to = conversationID.replace(/^(C2C|GROUP)/, '')
  const message = c.createCustomMessage({
    to,
    conversationType: isGroup ? TYPES.CONV_GROUP : TYPES.CONV_C2C,
    payload: {
      data: JSON.stringify(payload),
      description: description || String(payload.businessID || ''),
      extension: String(payload.businessID || ''),
    },
    needReadReceipt: !isGroup,
  })
  try {
    const res = await c.sendMessage(message)
    playSound('newmsg')
    return res?.data?.message as ChatMessage
  } catch (e) {
    throw new Error(formatSendError(e))
  }
}

export async function setMessageRead(conversationID: string) {
  const c = getChat()
  try {
    await c.setMessageRead({ conversationID })
  } catch {
    /* ignore */
  }
}

export async function fetchGroupProfile(groupID: string) {
  const c = getChat()
  const res = await c.getGroupProfile({
    groupID,
    groupProfileFilter: [
      'GroupProfile_Name',
      'GroupProfile_FaceUrl',
      'GroupProfile_Introduction',
      'GroupProfile_Notification',
      'GroupProfile_Owner_ID',
      'GroupProfile_MemberCount',
    ],
  })
  return res?.data?.group ?? {}
}

function asRecord(v: unknown): Record<string, unknown> {
  return v && typeof v === 'object' ? (v as Record<string, unknown>) : {}
}

function strField(v: unknown): string {
  if (typeof v === 'string') return v
  if (typeof v === 'number' && Number.isFinite(v)) return String(v)
  return ''
}

function mapMemberRole(raw: string): GroupMember['role'] {
  if (/owner/i.test(raw)) return 'Owner'
  if (/admin/i.test(raw)) return 'Admin'
  return 'Member'
}

/** SDK 成员字段在不同版本可能是 userID / userId / userProfile.* */
export function normalizeGroupMember(raw: unknown): GroupMember | null {
  const m = asRecord(raw)
  const profile = asRecord(m.userProfile || m.profile || m.memberInfo)
  const userID = strField(
    m.userID || m.userId || m.memberID || profile.userID || profile.userId,
  ).trim()
  if (!userID) return null
  const nick =
    strField(
      m.nameCard ||
        m.nick ||
        m.nickName ||
        profile.nameCard ||
        profile.nick ||
        profile.nickName,
    ).trim() || userID
  const avatar = strField(
    m.avatar || m.avatarURL || m.faceUrl || profile.avatar || profile.faceUrl,
  )
  return {
    userID,
    nick,
    avatar,
    role: mapMemberRole(strField(m.role || profile.role)),
    muteUntil: strField(m.muteUntil ?? '0'),
  }
}

function readMemberList(res: unknown): unknown[] {
  const data = asRecord(asRecord(res).data)
  const list = data.memberList ?? data.groupMemberList ?? data.members
  return Array.isArray(list) ? list : []
}

/**
 * 群成员分页 — 腾讯官方分页字段：
 * request: { groupID, offset, count }
 * response.data: { memberList, offset }  // offset 为下一页起点；0 或无更多则结束
 */
export async function fetchGroupMemberPage(
  groupID: string,
  offset = 0,
  count = GROUP_MEMBER_PAGE_SIZE,
): Promise<{ members: GroupMember[]; nextOffset: number; hasMore: boolean }> {
  const safeCount = Math.min(Math.max(count, 1), 100)
  const c = getChat()
  const res = await c.getGroupMemberList({
    groupID,
    offset,
    count: safeCount,
  })
  const members = readMemberList(res)
    .map(normalizeGroupMember)
    .filter((m): m is GroupMember => Boolean(m))
  const nextOffset = Number(asRecord(asRecord(res).data).offset ?? offset + members.length)
  const hasMore = members.length >= safeCount && nextOffset > offset
  return { members, nextOffset, hasMore }
}

/** 首次多拉几页凑够 want；单次仍不超过 SDK 100 */
export async function fetchGroupMembersUntil(
  groupID: string,
  want = GROUP_MEMBER_FIRST_PAGE,
  startOffset = 0,
): Promise<{ members: GroupMember[]; nextOffset: number; hasMore: boolean }> {
  const all: GroupMember[] = []
  const seen = new Set<string>()
  let offset = startOffset
  let hasMore = true
  while (all.length < want && hasMore) {
    const page = await fetchGroupMemberPage(groupID, offset, Math.min(100, want - all.length))
    for (const m of page.members) {
      if (seen.has(m.userID)) continue
      seen.add(m.userID)
      all.push(m)
    }
    hasMore = page.hasMore
    if (!page.members.length || page.nextOffset === offset) break
    offset = page.nextOffset
  }
  return { members: all, nextOffset: offset, hasMore }
}

/** 拉已入群 userID，用于添加成员时过滤；大群最多扫 10 页。 */
export async function fetchGroupMemberIdSet(
  groupID: string,
  max = 500,
): Promise<Set<string>> {
  const ids = new Set<string>()
  let offset = 0
  for (let i = 0; i < 10 && ids.size < max; i++) {
    const page = await fetchGroupMemberPage(groupID, offset, 100)
    for (const m of page.members) {
      const id = String(m.userID || '').trim()
      if (id) ids.add(id)
    }
    if (!page.hasMore || page.nextOffset === offset) break
    offset = page.nextOffset
  }
  return ids
}

/**
 * @ 选人 / 成员搜索：优先云搜 searchGroupMembers；否则只在已分页结果上过滤，绝不拉全员。
 */
export async function searchGroupMembersPage(
  groupID: string,
  keyword: string,
  offset = 0,
  count = AT_MEMBER_PAGE_SIZE,
): Promise<{ members: GroupMember[]; nextOffset: number; hasMore: boolean }> {
  const q = keyword.trim()
  const c = getChat()

  if (q && typeof c.searchGroupMembers === 'function') {
    try {
      const res = await c.searchGroupMembers({
        groupID,
        keywordList: [q],
        // 部分版本支持分页字段；没有则只取一页
        offset,
        count: Math.min(count, 100),
      })
      const members = readMemberList(res)
        .map(normalizeGroupMember)
        .filter((m): m is GroupMember => Boolean(m))
      const nextOffset = Number(res?.data?.offset ?? offset + members.length)
      return {
        members,
        nextOffset,
        hasMore: members.length >= count && nextOffset > offset,
      }
    } catch {
      /* fallback below */
    }
  }

  // 无云搜：按页拉成员再本地过滤（仍分页，不扫全员）
  const page = await fetchGroupMemberPage(groupID, offset, count)
  if (!q) return page
  const lower = q.toLowerCase()
  const filtered = page.members.filter(
    (m) => m.nick.toLowerCase().includes(lower) || m.userID.toLowerCase().includes(lower),
  )
  return { members: filtered, nextOffset: page.nextOffset, hasMore: page.hasMore }
}

export async function updateMyImProfile(patch: {
  nick?: string
  avatar?: string
  selfSignature?: string
  gender?: number
  birthday?: number
}) {
  const c = getChat()
  await c.updateMyProfile(patch)
}

export { TencentCloudChat }
