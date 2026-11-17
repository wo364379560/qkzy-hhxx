/**
 * 自研音视频 — 对齐 LingDoSpace（DoMattxCallService）
 * - IM Signaling：onlineUserOnly=true，不落历史
 * - 通话结果：单独发 av_call 自定义消息（气泡展示，非 tip）
 * - 媒体：trtc-sdk-v5
 * 同时兼容历史 rtc_call_v1 / mllx_trtc_call
 */
import { RTC_BUSINESS_ID, RTC_CALL_TIMEOUT_SEC } from '@/config/runtime'
import { getChat, TencentCloudChat, currentImUserId } from './ImClient'

// eslint-disable-next-line @typescript-eslint/no-explicit-any
type AnyChat = any

/** LingDoSpace 信令 businessID */
export const MLLX_TRTC_CALL = 'mllx_trtc_call'
/** LingDoSpace 会话摘要 businessID */
export const AV_CALL_BUSINESS = 'av_call'

export type RtcMediaType = 1 | 2
export type RtcAction =
  | 'invite'
  | 'accept'
  | 'reject'
  | 'cancel'
  | 'hangup'
  | 'no_response'

export type RtcCallPayload = {
  businessID: string
  action: RtcAction
  version: number
  mediaType: RtcMediaType
  roomId: string
  peerName?: string
  peerAvatar?: string
  durationSec?: number
  /** LingDoSpace av_call */
  actionType?: number
  summary?: string
}

let activeInviteId = ''

export function getActiveInviteId() {
  return activeInviteId
}

export function setActiveInviteId(id: string) {
  activeInviteId = id
}

export function buildRtcPayload(
  action: RtcAction,
  mediaType: RtcMediaType,
  roomId: string,
  extra?: Partial<RtcCallPayload>,
): RtcCallPayload {
  return {
    businessID: RTC_BUSINESS_ID,
    action,
    version: 1,
    mediaType,
    roomId,
    ...extra,
  }
}

/** LingDoSpace 邀请信令 data */
export function buildLingDoInviteSignal(opts: {
  mediaType: RtcMediaType
  roomId: string | number
  caller: string
  nickname?: string
  avatar?: string
  action?: string
}) {
  return {
    businessID: MLLX_TRTC_CALL,
    version: 1,
    mediaType: opts.mediaType,
    roomId: Number(opts.roomId) || 0,
    caller: opts.caller,
    nickname: opts.nickname || opts.caller,
    ...(opts.avatar ? { avatar: opts.avatar } : {}),
    ...(opts.action ? { action: opts.action } : {}),
  }
}

export function parseRtcPayload(raw: unknown): RtcCallPayload | null {
  if (raw == null || raw === '') return null
  try {
    const obj =
      typeof raw === 'string'
        ? (JSON.parse(raw) as Record<string, unknown>)
        : (raw as Record<string, unknown>)
    let nestedMap: Record<string, unknown> = {}
    const nested = obj.data
    if (typeof nested === 'string' && nested.trim().startsWith('{')) {
      try {
        nestedMap = JSON.parse(nested) as Record<string, unknown>
      } catch {
        /* ignore */
      }
    } else if (nested && typeof nested === 'object') {
      nestedMap = nested as Record<string, unknown>
    }

    const business = String(obj.businessID || nestedMap.businessID || '')
    const summary = String(obj.summary || nestedMap.summary || '')
    const descHint = summary

    // LingDoSpace av_call 摘要
    if (business === AV_CALL_BUSINESS || business === 'rtc_call') {
      const callType = Number(obj.call_type ?? nestedMap.call_type ?? 1)
      const mediaType: RtcMediaType = callType === 2 ? 2 : 1
      const actionType = Number(obj.actionType ?? nestedMap.actionType ?? 0)
      const callEnd = Number(obj.call_end ?? nestedMap.call_end ?? 0)
      let action: RtcAction = 'invite'
      if (callEnd > 0) action = 'hangup'
      else if (actionType === 2) action = 'cancel'
      else if (actionType === 3) action = 'accept'
      else if (actionType === 4) action = 'reject'
      else if (actionType === 5) action = 'no_response'
      else if (actionType === 1) action = 'invite'
      return {
        businessID: AV_CALL_BUSINESS,
        action,
        version: 1,
        mediaType,
        roomId: '',
        actionType,
        durationSec: callEnd > 0 ? callEnd : undefined,
        summary: descHint || undefined,
      }
    }

    // LingDoSpace / 自研信令 mllx_trtc_call 或 rtc_call_v1
    if (
      business === MLLX_TRTC_CALL ||
      business === RTC_BUSINESS_ID ||
      (!business && (obj.roomId != null || nestedMap.roomId != null))
    ) {
      const roomId = String(obj.roomId ?? nestedMap.roomId ?? obj.call_id ?? '')
      const mediaType: RtcMediaType =
        Number(obj.mediaType ?? nestedMap.mediaType ?? 1) === 2 ? 2 : 1
      const actionRaw = String(obj.action ?? nestedMap.action ?? 'invite')
      let action: RtcAction = 'invite'
      if (actionRaw === 'hangup') action = 'hangup'
      else if (actionRaw === 'busy') action = 'reject'
      else if (
        actionRaw === 'accept' ||
        actionRaw === 'reject' ||
        actionRaw === 'cancel' ||
        actionRaw === 'no_response'
      ) {
        action = actionRaw
      }
      return {
        businessID: business || MLLX_TRTC_CALL,
        action,
        version: Number(obj.version ?? 1),
        mediaType,
        roomId,
        peerName: String(obj.nickname || obj.peerName || nestedMap.nickname || '') || undefined,
        peerAvatar: String(obj.avatar || obj.peerAvatar || '') || undefined,
      }
    }

    if (descHint.includes('通话') || descHint.includes('语音') || descHint.includes('视频')) {
      return {
        businessID: AV_CALL_BUSINESS,
        action: /取消/.test(descHint)
          ? 'cancel'
          : /拒绝/.test(descHint)
            ? 'reject'
            : /未接|无应答/.test(descHint)
              ? 'no_response'
              : /时长|结束/.test(descHint)
                ? 'hangup'
                : 'invite',
        version: 1,
        mediaType: /视频/.test(descHint) ? 2 : 1,
        roomId: '',
        summary: descHint.startsWith('[') ? descHint : `[${descHint}]`,
      }
    }
    return null
  } catch {
    return null
  }
}

export function extractRtcFromMessage(msg: {
  type?: string
  payload?: { data?: unknown; description?: string; extension?: string }
}): RtcCallPayload | null {
  // 禁止在渲染历史消息时调用 getSignalingInfo：SDK 会把过期邀请再次派发成来电
  const payload = msg.payload || {}
  const fromData = parseRtcPayload(payload.data)
  if (fromData) return fromData

  const desc = String(payload.description || payload.extension || '')
  if (/语音通话|视频通话|通话/.test(desc)) {
    return {
      businessID: AV_CALL_BUSINESS,
      action: /取消/.test(desc)
        ? 'cancel'
        : /拒绝/.test(desc)
          ? 'reject'
          : /未接|无应答/.test(desc)
            ? 'no_response'
            : /时长|结束/.test(desc)
              ? 'hangup'
              : 'invite',
      version: 1,
      mediaType: /视频/.test(desc) ? 2 : 1,
      roomId: '',
      summary: desc.startsWith('[') ? desc : `[${desc}]`,
    }
  }
  return null
}

/** 气泡文案（对齐 LingDoSpace doMattxCallSummary） */
export function rtcBubbleText(payload: RtcCallPayload, _isSelf = false): string {
  if (payload.summary) {
    return payload.summary.startsWith('[') ? payload.summary : `[${payload.summary}]`
  }
  const media = payload.mediaType === 2 ? '视频通话' : '语音通话'
  if (payload.durationSec && payload.durationSec > 0) {
    const m = String(Math.floor(payload.durationSec / 60)).padStart(2, '0')
    const s = String(payload.durationSec % 60).padStart(2, '0')
    return `[${media}] 通话时长 ${m}:${s}`
  }
  switch (payload.action) {
    case 'cancel':
      return `[${media}] 已取消`
    case 'accept':
      return `[${media}] 已接听`
    case 'reject':
      return `[${media}] 已拒绝`
    case 'no_response':
      return `[${media}] 未接听`
    case 'hangup':
      return `[${media}] 通话结束`
    case 'invite':
    default:
      return `[${media}]`
  }
}

export function rtcPreviewText(payload: RtcCallPayload): string {
  return rtcBubbleText(payload)
}

function isTimSignalingEnvelope(raw: unknown): boolean {
  if (raw == null || raw === '') return true
  try {
    const obj =
      typeof raw === 'string'
        ? (JSON.parse(raw) as Record<string, unknown>)
        : (raw as Record<string, unknown>)
    if (!obj || typeof obj !== 'object') return false
    if (obj.inviteID || obj.inviteId) return true
    const bid = String(obj.businessID ?? '')
    if ((bid === '1' || bid === 'av_signaling') && obj.actionType != null) return true
    if (obj.actionType != null && !obj.summary && !obj.call_type && bid !== AV_CALL_BUSINESS) {
      return true
    }
    return false
  } catch {
    return false
  }
}

/** 隐藏信令/挂断等非摘要消息；仅展示 av_call 结果气泡 */
export function shouldHideRtcInviteMessage(msg: {
  type?: string
  payload?: { data?: unknown; description?: string; extension?: string }
}): boolean {
  const type = String(msg.type || '')
  const isCustom = type.includes('Custom') || type.includes('CUSTOM')
  // 禁止 getSignalingInfo：拉历史时 SDK 会把过期邀请再派发成来电并响铃
  const rtc = extractRtcFromMessage(msg)
  if (rtc) {
    if (rtc.businessID === AV_CALL_BUSINESS || rtc.businessID === 'rtc_call') return false
    return true
  }
  if (!isCustom) return false
  const data = msg.payload?.data
  if (isTimSignalingEnvelope(data)) return true
  const desc = String(msg.payload?.description || '').trim()
  if (!desc || /自定义消息|custom|发消息/i.test(desc)) {
    if (!data || data === '' || data === '{}') return true
  }
  return false
}

/** 监听连接后挂断自定义消息（对齐 LingDoSpace doMattxOnRecvMessage） */
export function watchHangupMessages(
  onHangup: (info: { roomId: string; peer: string }) => void,
) {
  const chat = getChat()
  const EVENT = (TencentCloudChat as AnyChat).EVENT || {}
  const name = EVENT.MESSAGE_RECEIVED || 'onMessageReceived'
  const handler = (evt: AnyChat) => {
    const list = (evt?.data || []) as Array<{
      from?: string
      type?: string
      payload?: { data?: unknown }
    }>
    for (const msg of list) {
      const rtc = extractRtcFromMessage(msg)
      if (!rtc || rtc.action !== 'hangup' || !rtc.roomId) continue
      onHangup({ roomId: String(rtc.roomId), peer: String(msg.from || '') })
    }
  }
  chat.on(name, handler)
  return () => {
    try {
      chat.off(name, handler)
    } catch {
      /* ignore */
    }
  }
}

export function signalingEvents() {
  const c = getChat()
  const ts = c.TSignaling || (TencentCloudChat as AnyChat).TSignaling || {}
  return {
    NEW_INVITATION_RECEIVED: ts.NEW_INVITATION_RECEIVED || 'newInvitationReceived',
    INVITEE_ACCEPTED: ts.INVITEE_ACCEPTED || 'ts_invitee_accepted',
    INVITEE_REJECTED: ts.INVITEE_REJECTED || 'ts_invitee_rejected',
    INVITATION_CANCELLED: ts.INVITATION_CANCELLED || 'ts_invitation_cancelled',
    INVITATION_TIMEOUT: ts.INVITATION_TIMEOUT || 'ts_invitation_timeout',
  }
}

function unwrapInviteEvent(evt: unknown): {
  inviteID: string
  inviter: string
  data: string
} | null {
  const e = evt as AnyChat
  const body = e?.data && (e.data.inviteID || e.data.inviter) ? e.data : e
  const inviteID = String(body?.inviteID || body?.inviteId || '')
  const inviter = String(body?.inviter || body?.from || body?.inviterID || '')
  const data =
    typeof body?.data === 'string' ? body.data : body?.data ? JSON.stringify(body.data) : ''
  if (!inviteID || !inviter) return null
  return { inviteID, inviter, data }
}

export type IncomingCallInfo = {
  peer: string
  roomId: string
  mediaType: RtcMediaType
  inviteID: string
  peerName?: string
}

const seenInviteIds = new Set<string>()
try {
  const raw = sessionStorage.getItem('cn_ziyan_seen_invites')
  const arr = raw ? (JSON.parse(raw) as string[]) : []
  if (Array.isArray(arr)) arr.slice(-80).forEach((id) => seenInviteIds.add(id))
} catch {
  /* ignore */
}

function rememberInviteId(id: string) {
  if (!id) return
  seenInviteIds.add(id)
  try {
    sessionStorage.setItem(
      'cn_ziyan_seen_invites',
      JSON.stringify([...seenInviteIds].slice(-80)),
    )
  } catch {
    /* ignore */
  }
}

function inviteEventAgeSec(evt: unknown, body: AnyChat): number {
  const t = Number(
    body?.message?.time ||
      body?.time ||
      (evt as AnyChat)?.data?.message?.time ||
      (evt as AnyChat)?.time ||
      0,
  )
  if (!t) return 0
  const sec = t > 1e12 ? t / 1000 : t
  return Date.now() / 1000 - sec
}

let incomingWatchOff: (() => void) | null = null

export function watchIncomingCalls(onInvite: (info: IncomingCallInfo) => void) {
  if (incomingWatchOff) {
    incomingWatchOff()
    incomingWatchOff = null
  }
  const chat = getChat()
  const events = signalingEvents()
  const onNew = (evt: unknown) => {
    const raw = unwrapInviteEvent(evt)
    if (!raw) return
    if (seenInviteIds.has(raw.inviteID)) return
    const me = currentImUserId()
    if (me && raw.inviter === me) return
    const payload = parseRtcPayload(raw.data)
    if (!payload?.roomId) return
    if (payload.businessID === AV_CALL_BUSINESS || payload.businessID === 'rtc_call') return
    if (
      payload.action === 'hangup' ||
      payload.action === 'cancel' ||
      payload.action === 'reject' ||
      payload.action === 'accept' ||
      payload.action === 'no_response'
    ) {
      return
    }
    const age = inviteEventAgeSec(evt, (evt as AnyChat)?.data || evt)
    // 有时间戳且已过期：历史回放，不当新来电
    if (age > RTC_CALL_TIMEOUT_SEC + 8) return
    rememberInviteId(raw.inviteID)
    setActiveInviteId(raw.inviteID)
    onInvite({
      peer: raw.inviter,
      roomId: String(payload.roomId),
      mediaType: payload.mediaType,
      inviteID: raw.inviteID,
      peerName: payload.peerName,
    })
  }

  if (typeof chat.addSignalingListener !== 'function') return () => undefined
  chat.addSignalingListener(events.NEW_INVITATION_RECEIVED, onNew)
  incomingWatchOff = () => {
    try {
      chat.removeSignalingListener(events.NEW_INVITATION_RECEIVED, onNew)
    } catch {
      /* ignore */
    }
    if (incomingWatchOff) incomingWatchOff = null
  }
  return incomingWatchOff
}

export function watchCallSignaling(handlers: {
  onAccepted?: (inviteID: string) => void
  onRejected?: (inviteID: string) => void
  onCancelled?: (inviteID: string) => void
  onTimeout?: (inviteID: string) => void
}) {
  const chat = getChat()
  const events = signalingEvents()
  const wrap = (fn: ((id: string) => void) | undefined) => (evt: AnyChat) => {
    const body = evt?.data && (evt.data.inviteID || evt.data.inviteId) ? evt.data : evt
    const id = String(body?.inviteID || body?.inviteId || '')
    if (id && fn) fn(id)
  }
  const a = wrap(handlers.onAccepted)
  const r = wrap(handlers.onRejected)
  const c = wrap(handlers.onCancelled)
  const t = wrap(handlers.onTimeout)
  if (typeof chat.addSignalingListener !== 'function') return () => undefined
  if (handlers.onAccepted) chat.addSignalingListener(events.INVITEE_ACCEPTED, a)
  if (handlers.onRejected) chat.addSignalingListener(events.INVITEE_REJECTED, r)
  if (handlers.onCancelled) chat.addSignalingListener(events.INVITATION_CANCELLED, c)
  if (handlers.onTimeout) chat.addSignalingListener(events.INVITATION_TIMEOUT, t)
  return () => {
    try {
      if (handlers.onAccepted) chat.removeSignalingListener(events.INVITEE_ACCEPTED, a)
      if (handlers.onRejected) chat.removeSignalingListener(events.INVITEE_REJECTED, r)
      if (handlers.onCancelled) chat.removeSignalingListener(events.INVITATION_CANCELLED, c)
      if (handlers.onTimeout) chat.removeSignalingListener(events.INVITATION_TIMEOUT, t)
    } catch {
      /* ignore */
    }
  }
}

/** 对齐 LingDoSpace doMattxRoomIdFromSeed */
export function genRtcRoomId(seed?: string) {
  const s = seed || `${Date.now()}_${Math.random()}`
  let h = 0
  for (let i = 0; i < s.length; i++) {
    h = (h * 31 + s.charCodeAt(i)) & 0x7fffffff
  }
  if (h <= 0) h = 1
  return String(h)
}

export async function inviteCall(opts: {
  userID: string
  mediaType: RtcMediaType
  roomId?: string
  peerName?: string
  peerAvatar?: string
}) {
  const chat = getChat()
  const selfId = currentImUserId() || ''
  const roomId = opts.roomId || genRtcRoomId(`${selfId}_${opts.userID}_${Date.now()}`)
  const signal = buildLingDoInviteSignal({
    mediaType: opts.mediaType,
    roomId,
    caller: selfId,
    nickname: opts.peerName ? undefined : selfId, // 主叫昵称应是自己；下面用 session 更好
    avatar: opts.peerAvatar,
  })
  // 主叫展示名：用本地昵称
  try {
    const nick = localStorage.getItem('cn_ziyan_nickname') || selfId
    signal.nickname = nick
  } catch {
    signal.nickname = selfId
  }

  if (typeof chat.invite !== 'function') {
    throw new Error('当前 Chat SDK 无 invite 信令，请升级 @tencentcloud/chat')
  }
  const res = await chat.invite({
    userID: opts.userID,
    data: JSON.stringify(signal),
    timeout: RTC_CALL_TIMEOUT_SEC,
    // 对齐 LingDoSpace：信令不落历史，结果靠 av_call 摘要
    onlineUserOnly: true,
  })
  const inviteID = String(res?.data?.inviteID || res?.inviteID || res?.data || '')
  if (!inviteID) throw new Error('发起通话失败：无 inviteID')
  setActiveInviteId(inviteID)
  return {
    inviteID,
    roomId,
    payload: buildRtcPayload('invite', opts.mediaType, roomId),
  }
}

export async function acceptInvite(inviteID: string, _mediaType: RtcMediaType, _roomId: string) {
  const chat = getChat()
  try {
    await chat.accept({ inviteID, data: '', onlineUserOnly: true })
  } catch {
    await chat.accept({ inviteID, data: '' })
  }
  setActiveInviteId(inviteID)
}

export async function rejectInvite(inviteID: string, _mediaType: RtcMediaType, _roomId: string) {
  const chat = getChat()
  try {
    await chat.reject({ inviteID, data: '', onlineUserOnly: true })
  } catch {
    await chat.reject({ inviteID, data: '' })
  }
  if (activeInviteId === inviteID) setActiveInviteId('')
}

export async function cancelInvite(inviteID: string, _mediaType: RtcMediaType, _roomId: string) {
  const chat = getChat()
  try {
    await chat.cancel({ inviteID, data: '', onlineUserOnly: true })
  } catch {
    await chat.cancel({ inviteID, data: '' })
  }
  if (activeInviteId === inviteID) setActiveInviteId('')
}

/**
 * 发送通话结果摘要（气泡）— 对齐 LingDoSpace _doMattxSendCallTip
 * actionType: 1默认/时长 2取消 3接听 4拒绝 5未接听
 */
export async function sendAvCallSummary(opts: {
  to: string
  mediaType: RtcMediaType
  actionType: number
  callEnd?: number
}) {
  const chat = getChat()
  const TYPES = (TencentCloudChat as AnyChat).TYPES
  const mediaLabel = opts.mediaType === 2 ? '视频通话' : '语音通话'
  let label: string
  if (opts.callEnd && opts.callEnd > 0) {
    const m = String(Math.floor(opts.callEnd / 60)).padStart(2, '0')
    const s = String(opts.callEnd % 60).padStart(2, '0')
    label = `[${mediaLabel}] 通话时长 ${m}:${s}`
  } else {
    switch (opts.actionType) {
      case 2:
        label = `[${mediaLabel}] 已取消`
        break
      case 3:
        label = `[${mediaLabel}] 已接听`
        break
      case 4:
        label = `[${mediaLabel}] 已拒绝`
        break
      case 5:
        label = `[${mediaLabel}] 未接听`
        break
      default:
        label = `[${mediaLabel}]`
    }
  }
  const payload = {
    businessID: AV_CALL_BUSINESS,
    call_type: opts.mediaType,
    actionType: opts.actionType,
    cmd: opts.mediaType === 2 ? 'videoCall' : 'audioCall',
    ...(opts.callEnd && opts.callEnd > 0 ? { call_end: opts.callEnd } : {}),
    summary: label,
  }
  const message = chat.createCustomMessage({
    to: opts.to,
    conversationType: TYPES.CONV_C2C,
    payload: {
      data: JSON.stringify(payload),
      description: label,
      extension: label,
    },
  })
  const res = await chat.sendMessage(message)
  return res?.data?.message
}

/** @deprecated */
export async function acceptCall(inviteID: string, data?: string) {
  await getChat().accept({ inviteID, data: data || '' })
}
/** @deprecated */
export async function rejectCall(inviteID: string, data?: string) {
  await getChat().reject({ inviteID, data: data || '' })
}
/** @deprecated */
export async function cancelCall(inviteID: string, data?: string) {
  await getChat().cancel({ inviteID, data: data || '' })
}

export async function sendCallRecordMessage(opts: {
  to: string
  isGroup: boolean
  payload: RtcCallPayload
}) {
  const actionType =
    opts.payload.action === 'cancel'
      ? 2
      : opts.payload.action === 'accept'
        ? 3
        : opts.payload.action === 'reject'
          ? 4
          : opts.payload.action === 'no_response'
            ? 5
            : 1
  return sendAvCallSummary({
    to: opts.to,
    mediaType: opts.payload.mediaType,
    actionType,
    callEnd: opts.payload.durationSec,
  })
}

export { RTC_CALL_TIMEOUT_SEC }
