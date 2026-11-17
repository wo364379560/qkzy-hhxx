import { readBlockFlagMaybe } from './groupActions'
import { getChat, TencentCloudChat } from './ImClient'
import type { GroupMember } from './types'
import { groupIdFromConversation } from '@/utils/conversation'

export type GroupLiveEvent = {
  groupID: string
  muteAllMembers?: boolean
  blockMemberProfile?: boolean
  membersChanged?: boolean
  added?: Array<{ userID: string; nick?: string; avatar?: string; role?: 'Owner' | 'Admin' | 'Member' }>
  removedIds?: string[]
  updated?: Array<{ userID: string; nick?: string; avatar?: string; role?: 'Owner' | 'Admin' | 'Member' }>
}

const localLive = new Set<(e: GroupLiveEvent) => void>()

/** 本端加/踢：只带变更的人，列表做增量，禁止整表重拉 */
export function notifyGroupMembersChanged(
  groupID: string,
  patch?: Pick<GroupLiveEvent, 'added' | 'removedIds' | 'updated'>,
) {
  const id = gidOf(groupID)
  if (!id) return
  const ev: GroupLiveEvent = {
    groupID: id,
    membersChanged: true,
    added: patch?.added,
    removedIds: patch?.removedIds,
    updated: patch?.updated,
  }
  localLive.forEach((cb) => cb(ev))
}

/** 列表增量：只增删变更的人，不重拉全页 */
export function applyMemberLivePatch(
  prev: GroupMember[],
  e: GroupLiveEvent,
): { next: GroupMember[]; addedCount: number; removedCount: number } {
  const seen = new Set(prev.map((m) => m.userID))
  let next = prev
  let removedCount = 0
  if (e.removedIds?.length) {
    const rm = new Set(e.removedIds)
    next = next.filter((m) => {
      if (!rm.has(m.userID)) return true
      removedCount += 1
      return false
    })
  }
  const extra: GroupMember[] = []
  for (const a of e.added || []) {
    const userID = String(a.userID || '').trim()
    if (!userID || seen.has(userID)) continue
    seen.add(userID)
    extra.push({
      userID,
      nick: a.nick || userID,
      avatar: a.avatar || '',
      role: a.role || 'Member',
    })
  }
  if (extra.length) next = [...next, ...extra]
  if (e.updated?.length) {
    const map = new Map(e.updated.map((u) => [u.userID, u]))
    next = next.map((m) => {
      const u = map.get(m.userID)
      if (!u) return m
      return {
        ...m,
        nick: u.nick ?? m.nick,
        avatar: u.avatar ?? m.avatar,
        role: u.role ?? m.role,
      }
    })
  }
  return { next, addedCount: extra.length, removedCount }
}

function parseMuteAllFlag(raw: unknown): boolean | undefined {
  if (raw === undefined || raw === null || raw === '') return undefined
  if (raw === true || raw === 1 || raw === '1') return true
  if (raw === false || raw === 0 || raw === '0') return false
  const s = String(raw).toLowerCase()
  if (s === 'on' || s === 'true' || s === 'yes') return true
  if (s === 'off' || s === 'false' || s === 'no') return false
  return undefined
}

function gidOf(raw: unknown): string {
  const id = String(raw || '')
  if (id.startsWith('GROUP')) return groupIdFromConversation(id)
  return id
}

export function muteAllFromGroupTip(msg: unknown): boolean | undefined {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const m = msg as any
  const type = String(m?.type || '')
  if (!type.includes('GroupTip') && !type.includes('GROUP_TIP') && !type.includes('GroupSystem')) {
    return undefined
  }
  const p = m?.payload || {}
  const g = p.newGroupProfile || p.groupProfile || {}
  const fromFlag = parseMuteAllFlag(g.muteAllMembers)
  if (fromFlag !== undefined) return fromFlag
  const text = String(p.text || '')
  if (text.includes('开启了全员禁言')) return true
  if (text.includes('取消了全员禁言')) return false
  return undefined
}

function emitFromMessage(
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  msg: any,
  cb: (e: GroupLiveEvent) => void,
) {
  const type = String(msg?.type || '')
  if (!type.includes('GroupTip') && !type.includes('GROUP_TIP') && !type.includes('GroupSystem')) {
    return
  }
  const p = msg.payload || {}
  const groupID = gidOf(p.groupID || msg.conversationID)
  if (!groupID) return
  const muteAllMembers = muteAllFromGroupTip(msg)
  const op = Number(p.operationType || 0)
  const ids = [
    ...((p.userIDList as string[]) || []),
    ...(((p.memberList as Array<{ userID?: string }>) || []).map((m) => m.userID || '')),
  ]
    .map((id) => String(id || '').trim())
    .filter(Boolean)
  const membersChanged = op === 1 || op === 2 || op === 3
  if (muteAllMembers !== undefined || op === 6 || op === 7 || membersChanged) {
    cb({
      groupID,
      muteAllMembers,
      membersChanged,
      added: op === 1 ? ids.map((userID) => ({ userID })) : undefined,
      removedIds: op === 2 || op === 3 ? ids : undefined,
    })
  }
}

export function onGroupLiveChange(cb: (e: GroupLiveEvent) => void): () => void {
  localLive.add(cb)
  let chat: ReturnType<typeof getChat>
  try {
    chat = getChat()
  } catch {
    return () => {
      localLive.delete(cb)
    }
  }
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const EVENT = (TencentCloudChat as any).EVENT || {}
  const offs: Array<() => void> = []

  const onMsg = (event: { data?: unknown[] }) => {
    ;(event.data || []).forEach((msg) => emitFromMessage(msg, cb))
  }
  chat.on(EVENT.MESSAGE_RECEIVED, onMsg)
  offs.push(() => chat.off(EVENT.MESSAGE_RECEIVED, onMsg))

  const attrEvts = [
    EVENT.GROUP_ATTRIBUTES_UPDATED,
    EVENT.GROUP_ATTRIBUTE_UPDATED,
  ].filter(Boolean)
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const onAttr = (event: { data?: any }) => {
    const rows = Array.isArray(event.data) ? event.data : [event.data]
    rows.forEach((row) => {
      const groupID = gidOf(row?.groupID)
      if (!groupID) return
      const raw = row?.groupAttributes || row?.groupAttributeMap || row
      cb({
        groupID,
        blockMemberProfile: readBlockFlagMaybe(raw),
      })
    })
  }
  attrEvts.forEach((evt) => {
    chat.on(evt, onAttr)
    offs.push(() => chat.off(evt, onAttr))
  })

  const listEvt = EVENT.GROUP_LIST_UPDATED
  if (listEvt) {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const onList = (event: { data?: any[] }) => {
      ;(event.data || []).forEach((g) => {
        const groupID = gidOf(g?.groupID)
        if (!groupID) return
        cb({
          groupID,
          muteAllMembers: parseMuteAllFlag(g?.muteAllMembers),
        })
      })
    }
    chat.on(listEvt, onList)
    offs.push(() => chat.off(listEvt, onList))
  }

  const memberEvt = EVENT.GROUP_MEMBER_LIST_UPDATED
  if (memberEvt) {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const onMembers = (event: { data?: any }) => {
      const rows = Array.isArray(event.data) ? event.data : [event.data]
      rows.forEach((row) => {
        const groupID = gidOf(row?.groupID)
        if (groupID) cb({ groupID, membersChanged: true })
      })
    }
    chat.on(memberEvt, onMembers)
    offs.push(() => chat.off(memberEvt, onMembers))
  }

  return () => {
    localLive.delete(cb)
    offs.forEach((fn) => fn())
  }
}
