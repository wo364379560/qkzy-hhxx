import {
  BLOCK_MEMBER_PROFILE_KEYS,
  SPEAK_FREQ_ATTR_KEY,
  type JoinInviteOption,
} from '@/constants/groupIM'
import { getChat } from './ImClient'

// eslint-disable-next-line @typescript-eslint/no-explicit-any
type AnyChat = any

export type GroupDetail = {
  groupID: string
  name: string
  avatar: string
  introduction: string
  notification: string
  ownerID: string
  memberCount: number
  joinOption: JoinInviteOption
  inviteOption: JoinInviteOption
  blockMemberProfile: boolean
  muteAllMembers: boolean
  speakFreqSec: number
}

export function readBlockFlagMaybe(raw: unknown): boolean | undefined {
  if (!raw) return undefined
  if (typeof raw === 'object' && !Array.isArray(raw)) {
    const map = raw as Record<string, string>
    for (const key of BLOCK_MEMBER_PROFILE_KEYS) {
      if (map[key] === '1' || map[key] === '0') {
        // inner_attr_member_profile_access: '0' = 禁止互看
        if (key === 'inner_attr_member_profile_access') return map[key] === '0'
        return map[key] === '1'
      }
    }
  }
  if (Array.isArray(raw)) {
    for (const item of raw as Array<{ key?: string; value?: string }>) {
      if (!item?.key) continue
      if (BLOCK_MEMBER_PROFILE_KEYS.includes(item.key as (typeof BLOCK_MEMBER_PROFILE_KEYS)[number])) {
        if (item.key === 'inner_attr_member_profile_access') return item.value === '0'
        return item.value === '1'
      }
    }
  }
  return undefined
}

export function readBlockFlag(raw: unknown): boolean {
  return Boolean(readBlockFlagMaybe(raw))
}

function readAttrMap(raw: unknown): Record<string, string> {
  if (!raw) return {}
  if (typeof raw === 'object' && !Array.isArray(raw)) {
    const out: Record<string, string> = {}
    for (const [k, v] of Object.entries(raw as Record<string, unknown>)) {
      out[k] = String(v ?? '')
    }
    return out
  }
  if (Array.isArray(raw)) {
    const out: Record<string, string> = {}
    for (const item of raw as Array<{ key?: string; value?: string }>) {
      if (item?.key) out[item.key] = String(item.value ?? '')
    }
    return out
  }
  return {}
}

function readSpeakFreq(raw: unknown): number {
  const map = readAttrMap(raw)
  const n = Number(map[SPEAK_FREQ_ATTR_KEY] || 0)
  return Number.isFinite(n) && n > 0 ? n : 0
}

export async function fetchGroupDetail(groupID: string): Promise<GroupDetail> {
  const chat = getChat() as AnyChat
  const res = await chat.getGroupProfile({ groupID })
  const g = res?.data?.group || {}
  let blockMemberProfile = false
  let speakFreqSec = 0
  try {
    const attrRes = await chat.getGroupAttributes({
      groupID,
      keyList: [...BLOCK_MEMBER_PROFILE_KEYS, SPEAK_FREQ_ATTR_KEY],
    })
    const raw =
      attrRes.data?.groupAttributes ||
      attrRes.data?.groupAttributeMap ||
      attrRes.data?.groupAttributeList?.[0]?.keyValueList ||
      {}
    blockMemberProfile = readBlockFlag(raw)
    speakFreqSec = readSpeakFreq(raw)
  } catch {
    blockMemberProfile = false
    speakFreqSec = 0
  }
  return {
    groupID: g.groupID || groupID,
    name: g.name || '',
    avatar: g.avatar || '',
    introduction: g.introduction || '',
    notification: g.notification || '',
    ownerID: g.ownerID || '',
    memberCount: g.memberCount || 0,
    joinOption: g.joinOption || 'NeedPermission',
    inviteOption: g.inviteOption || 'FreeAccess',
    blockMemberProfile,
    muteAllMembers: Boolean(g.muteAllMembers),
    speakFreqSec,
  }
}

export async function updateGroupDetail(
  groupID: string,
  patch: Partial<{
    name: string
    introduction: string
    notification: string
    avatar: string
    joinOption: JoinInviteOption
    inviteOption: JoinInviteOption
    muteAllMembers: boolean
  }>,
) {
  const chat = getChat() as AnyChat
  await chat.updateGroupProfile({ groupID, ...patch })
}

export async function setSpeakFreqSec(groupID: string, sec: number) {
  const chat = getChat() as AnyChat
  const attrs = { [SPEAK_FREQ_ATTR_KEY]: String(Math.max(0, Math.floor(sec))) }
  try {
    await chat.setGroupAttributes({ groupID, groupAttributes: attrs })
  } catch {
    await chat.setGroupAttributes({
      groupID,
      groupAttributeList: Object.entries(attrs).map(([key, value]) => ({ key, value })),
    })
  }
}

export async function setBlockMemberProfile(groupID: string, blocked: boolean) {
  const chat = getChat() as AnyChat
  // 规格 inner_attr：0=禁止互看；兼容 canglian：1=禁止
  const attrs = {
    inner_attr_member_profile_access: blocked ? '0' : '1',
    canglian_block_member_profile: blocked ? '1' : '0',
  }
  try {
    await chat.setGroupAttributes({ groupID, groupAttributes: attrs })
  } catch {
    await chat.setGroupAttributes({
      groupID,
      groupAttributeList: Object.entries(attrs).map(([key, value]) => ({ key, value })),
    })
  }
}

export async function quitGroup(groupID: string) {
  const chat = getChat() as AnyChat
  await chat.quitGroup(groupID)
}

export async function dismissGroup(groupID: string) {
  const chat = getChat() as AnyChat
  await chat.dismissGroup(groupID)
}
