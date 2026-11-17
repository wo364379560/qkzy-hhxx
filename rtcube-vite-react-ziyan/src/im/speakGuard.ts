import { fetchGroupDetail } from './groupActions'
import { getChat } from './ImClient'

export type SpeakBlock = {
  blocked: boolean
  reason: string
  muteAll: boolean
  personalMuted: boolean
  isManager: boolean
}

function muteStillActive(raw: unknown): boolean {
  const n = Number(raw || 0)
  if (!Number.isFinite(n) || n <= 0) return false
  if (n < 1e9) return true
  const ms = n < 1e12 ? n * 1000 : n
  return ms > Date.now()
}

export function applyMuteAll(prev: SpeakBlock, muteAll: boolean): SpeakBlock {
  const blocked = prev.personalMuted || (muteAll && !prev.isManager)
  const reason = prev.personalMuted ? '你已被禁言' : muteAll && !prev.isManager ? '全员禁言中' : ''
  return { ...prev, muteAll, blocked, reason }
}

export async function fetchSpeakBlock(groupId: string, selfId: string): Promise<SpeakBlock> {
  const empty: SpeakBlock = {
    blocked: false,
    reason: '',
    muteAll: false,
    personalMuted: false,
    isManager: false,
  }
  if (!groupId || !selfId) return empty
  try {
    const d = await fetchGroupDetail(groupId)
    let isManager = Boolean(selfId && d.ownerID === selfId)
    let personalMuted = false
    try {
      const res = await getChat().getGroupMemberProfile({
        groupID: groupId,
        userIDList: [selfId],
      })
      const m = res?.data?.memberList?.[0] || {}
      const role = String(m.role || '')
      if (role === 'Admin' || role === 'Owner') isManager = true
      personalMuted = muteStillActive(m.muteUntil ?? m.muteTime ?? m.muteUntilTime)
    } catch {
      /* ignore */
    }
    const muteAll = Boolean(d.muteAllMembers)
    const blocked = personalMuted || (muteAll && !isManager)
    const reason = personalMuted ? '你已被禁言' : muteAll && !isManager ? '全员禁言中' : ''
    return { blocked, reason, muteAll, personalMuted, isManager }
  } catch {
    return empty
  }
}
