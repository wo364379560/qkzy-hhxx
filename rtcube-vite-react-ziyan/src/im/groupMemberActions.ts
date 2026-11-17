import { chatDelete, chatPost } from '@/api/chatHttp'
import { notifyGroupMembersChanged } from './groupLive'
import { fetchUserProfiles, getChat, TencentCloudChat } from './ImClient'

function cleanIds(userIDList: string[]) {
  return userIDList.map((id) => String(id || '').trim()).filter(Boolean)
}

async function addMembersBySdk(groupID: string, userIDList: string[]) {
  const chat = getChat()
  const res = await chat.addGroupMember({
    groupID,
    userIDList,
  })
  const data = (res?.data || {}) as {
    successUserIDList?: string[]
    existedUserIDList?: string[]
    failureUserIDList?: Array<string | { userID?: string; message?: string }>
  }
  const failed = data.failureUserIDList || []
  const ok = (data.successUserIDList || []).length + (data.existedUserIDList || []).length
  if (failed.length && !ok) {
    const first = failed[0]
    const msg = typeof first === 'string' ? first : first?.message
    throw new Error(msg || '邀请失败')
  }
}

/** 邀请/添加成员 — 默认走后台（对齐 App 静默拉人），失败再 SDK */
export async function addGroupMembers(groupID: string, userIDList: string[]) {
  const ids = cleanIds(userIDList)
  if (!ids.length) return
  try {
    await chatPost(`/groups/${encodeURIComponent(groupID)}/members`, { user_ids: ids }, true)
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e)
    if (/没有群管理权限/.test(msg)) {
      await addMembersBySdk(groupID, ids)
    } else {
      throw e instanceof Error ? e : new Error(msg)
    }
  }
  const profiles = await fetchUserProfiles(ids).catch(() => [])
  notifyGroupMembersChanged(groupID, {
    added: ids.map((userID) => {
      const p = profiles.find((x) => x.userID === userID)
      return {
        userID,
        nick: p?.nick || userID,
        avatar: p?.avatar || '',
        role: 'Member' as const,
      }
    }),
  })
  void sendGroupMemberInviteTip(groupID, ids)
}

/** 对齐 App：businessID = mllx_group_member_invite，居中 tip，邀请人用当前用户 */
async function sendGroupMemberInviteTip(groupID: string, memberIDs: string[]) {
  try {
    const chat = getChat()
    const selfId = String(chat.getLoginUser?.() || '')
    const profiles = await fetchUserProfiles([selfId, ...memberIDs].filter(Boolean))
    const nameOf = (id: string) => profiles.find((p) => p.userID === id)?.nick || id
    const inviterName = nameOf(selfId) || '成员'
    const members = memberIDs.map((userID) => ({ userID, nick: nameOf(userID) }))
    const names = members.map((m) => m.nick).filter(Boolean).join('、')
    const content = names
      ? `${inviterName} 邀请 ${names} 加入群聊`
      : `${inviterName} 邀请成员加入群聊`
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const TYPES = (TencentCloudChat as any).TYPES
    const message = chat.createCustomMessage({
      to: groupID,
      conversationType: TYPES.CONV_GROUP,
      payload: {
        data: JSON.stringify({
          businessID: 'mllx_group_member_invite',
          inviter: { userID: selfId, nick: inviterName },
          members,
          content,
        }),
        description: content,
        extension: 'mllx_group_member_invite',
      },
      needReadReceipt: false,
    })
    await chat.sendMessage(message)
  } catch {
    /* 邀请已成功，tip 失败不影响入群 */
  }
}

/** 踢人 — 默认走后台 DELETE，失败再 SDK */
export async function kickGroupMembers(groupID: string, userIDList: string[]) {
  const ids = cleanIds(userIDList)
  if (!ids.length) return
  try {
    for (const userID of ids) {
      await chatDelete(
        `/groups/${encodeURIComponent(groupID)}/members/${encodeURIComponent(userID)}`,
        true,
      )
    }
  } catch {
    const chat = getChat()
    await chat.deleteGroupMember({
      groupID,
      userIDList: ids,
      reason: '',
    })
  }
  notifyGroupMembersChanged(groupID, { removedIds: ids })
}

export async function setGroupMemberMute(
  groupID: string,
  userID: string,
  muteTime: number,
) {
  const chat = getChat()
  await chat.setGroupMemberMuteTime({
    groupID,
    userID,
    muteTime,
  })
}

export async function setGroupMemberRole(
  groupID: string,
  userID: string,
  role: 'Admin' | 'Member',
) {
  const chat = getChat()
  await chat.setGroupMemberRole({ groupID, userID, role })
  notifyGroupMembersChanged(groupID, { updated: [{ userID, role }] })
}

export async function setGroupMemberNameCard(
  groupID: string,
  userID: string,
  nameCard: string,
) {
  const chat = getChat()
  const nick = nameCard.trim()
  await chat.setGroupMemberNameCard({ groupID, userID, nameCard: nick })
  notifyGroupMembersChanged(groupID, { updated: [{ userID, nick: nick || userID }] })
}
