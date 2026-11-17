import { getChat, TencentCloudChat } from './ImClient'

// eslint-disable-next-line @typescript-eslint/no-explicit-any
type AnyChat = any

export type UserProfile = {
  userID: string
  nick: string
  avatar: string
  selfSignature?: string
}

export async function fetchUserProfile(userID: string): Promise<UserProfile> {
  const chat = getChat() as AnyChat
  const res = await chat.getUserProfile({ userIDList: [userID] })
  const p = res?.data?.[0] || {}
  return {
    userID: p.userID || userID,
    nick: p.nick || userID,
    avatar: p.avatar || '',
    selfSignature: p.selfSignature || '',
  }
}

export async function pinConversation(conversationID: string, isPinned: boolean) {
  const chat = getChat() as AnyChat
  await chat.pinConversation({ conversationID, isPinned })
}

export async function setConversationMute(conversationID: string, mute: boolean) {
  const chat = getChat() as AnyChat
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const TYPES = (TencentCloudChat as any).TYPES
  await chat.setMessageRemindType({
    groupID: conversationID.startsWith('GROUP') ? conversationID.replace(/^GROUP/, '') : undefined,
    userIDList: conversationID.startsWith('C2C') ? [conversationID.replace(/^C2C/, '')] : undefined,
    messageRemindType: mute ? TYPES.MSG_REMIND_ACPT_NOT_NOTE : TYPES.MSG_REMIND_ACPT_AND_NOTE,
  })
}

export async function deleteFriend(userID: string) {
  const chat = getChat() as AnyChat
  await chat.deleteFriend({
    userIDList: [userID],
    type: (TencentCloudChat as AnyChat).TYPES?.SNS_DELETE_TYPE_BOTH,
  })
}

export async function addToBlacklist(userID: string) {
  const chat = getChat() as AnyChat
  await chat.addToBlacklist({ userIDList: [userID] })
}

export async function removeFromBlacklist(userID: string) {
  const chat = getChat() as AnyChat
  await chat.removeFromBlacklist({ userIDList: [userID] })
}

export async function getBlacklist(): Promise<string[]> {
  const chat = getChat() as AnyChat
  const res = await chat.getBlacklist()
  return (res?.data || []).map((x: string | { userID?: string }) =>
    typeof x === 'string' ? x : x.userID || '',
  )
}

export async function updateFriendRemark(userID: string, remark: string) {
  const chat = getChat() as AnyChat
  await chat.updateFriend({
    userID,
    remark,
  })
}

export async function getConversationProfile(conversationID: string) {
  const chat = getChat() as AnyChat
  const res = await chat.getConversationProfile({ conversationID })
  return res?.data?.conversation || null
}

export async function deleteConversation(conversationID: string) {
  const chat = getChat() as AnyChat
  if (!conversationID) return
  try {
    await chat.deleteConversation({
      conversationIDList: [conversationID],
      clearHistoryMessage: false,
    })
  } catch {
    await chat.deleteConversation(conversationID)
  }
}

export async function markConversationUnread(conversationID: string) {
  const chat = getChat() as AnyChat
  if (typeof chat.setConversationUnread === 'function') {
    await chat.setConversationUnread({ conversationID, unreadCount: 1 })
    return
  }
  throw new Error('当前 SDK 不支持标为未读')
}

export async function setConversationDraft(conversationID: string, draftText: string) {
  const chat = getChat() as AnyChat
  if (!conversationID) return
  try {
    await chat.setConversationDraft({ conversationID, draftText: draftText || '' })
  } catch {
    /* 部分会话类型可能不支持草稿 */
  }
}
