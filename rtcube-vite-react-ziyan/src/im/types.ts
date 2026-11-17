export type GroupMember = {
  userID: string
  nick: string
  avatar: string
  role: 'Owner' | 'Admin' | 'Member'
  muteUntil?: string
}

export type ConversationItem = {
  conversationID: string
  type: string
  userID?: string
  groupID?: string
  unreadCount: number
  lastMessage?: {
    payload?: ChatMessage['payload']
    type?: string
    nick?: string
    nameCard?: string
    from?: string
    lastTime?: number
  }
  remark?: string
  isPinned?: boolean
  /** SDK 免打扰字段 */
  messageRemindType?: string | number
  isMuted?: boolean
  draftText?: string
  groupAtInfoList?: Array<{ atType?: number; atTypeList?: number[] }>
  userProfile?: { nick?: string; avatar?: string }
  groupProfile?: { name?: string; avatar?: string; memberCount?: number; notification?: string }
}

export type ChatMessage = {
  ID: string
  type: string
  conversationID: string
  from: string
  to: string
  flow: 'in' | 'out'
  time: number
  nick?: string
  avatar?: string
  sequence?: number
  seq?: number
  msgSeq?: number
  revoker?: string
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  payload?: Record<string, any> & {
    text?: string
    data?: string
    description?: string
    extension?: string
    userIDList?: string[]
    groupAtUserList?: string[]
    operationType?: number
    groupJoinType?: number
    operatorID?: string
    operatorInfo?: { userID?: string; nick?: string; nameCard?: string; nickName?: string }
    memberList?: Array<{ userID?: string; nick?: string; nameCard?: string; nickName?: string }>
    groupProfile?: { groupName?: string; notification?: string; ownerID?: string }
    newGroupProfile?: { groupName?: string; notification?: string; ownerID?: string }
    title?: string
    abstractList?: string[]
    messageList?: ChatMessage[]
  }
  status?: string
  isRevoked?: boolean
  cloudCustomData?: string
  needReadReceipt?: boolean
  isPeerRead?: boolean
  readReceiptInfo?: { isPeerRead?: boolean }
  /** 发送中 0~1 */
  progress?: number
  /** 本地预览 blob: url，发出去立刻显示用 */
  localPreview?: string
  sendFailed?: boolean
  sendError?: string
}
