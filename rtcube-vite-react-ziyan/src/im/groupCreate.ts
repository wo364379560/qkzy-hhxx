/**
 * 建群：默认社群 Community，可选普通群 Public。
 * 建群后发 group_create 自定义 tip（对齐 iOS ShangLan_IMManager）。
 */
import { routedFetch } from '@/api/transport'
import { getLocalSession, getToken } from '@/api/session'
import { normalizeGroupId } from '@/utils/groupId'
import { getChat, TencentCloudChat } from './ImClient'
import { markGroupJustCreated } from './messageTips'
import type { ChatMessage } from './types'

export type CreateGroupKind = 'community' | 'public'

export type CreateGroupMember = {
  userID: string
  nick?: string
}

export type CreateGroupParams = {
  name: string
  groupId?: string
  kind?: CreateGroupKind // 默认 community
  memberIDList?: string[]
  members?: CreateGroupMember[]
  inviterName?: string
}

export async function createGroupWithTip(params: CreateGroupParams): Promise<{
  groupID: string
  conversationID: string
  tipMessage?: ChatMessage
  warning?: string
}> {
  const chat = getChat()
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const TYPES = (TencentCloudChat as any).TYPES
  const kind: CreateGroupKind = params.kind || 'community'
  const isCommunity = kind === 'community'
  // 社群：勿传 9 位数字；留空由 SDK 分配 @TGS#…，或自定义必须以 @TGS# 开头
  const customGroupID = normalizeGroupId(params.groupId, kind)
  const name = params.name.trim()
  if (!name) throw new Error('请输入群名称')

  const type = isCommunity
    ? TYPES?.GRP_COMMUNITY || 'Community'
    : TYPES?.GRP_PUBLIC || 'Public'

  const createOpts: Record<string, unknown> = {
    name,
    type,
    joinOption: isCommunity
      ? TYPES?.JOIN_OPTIONS_FREE_ACCESS || 'FreeAccess'
      : TYPES?.JOIN_OPTIONS_NEED_PERMISSION || 'NeedPermission',
    inviteOption: TYPES?.JOIN_OPTIONS_FREE_ACCESS || 'FreeAccess',
  }
  if (customGroupID) {
    createOpts.groupID = customGroupID
  }

  const ids = [...new Set((params.memberIDList || []).filter(Boolean))]
  if (ids.length > 100) throw new Error('最多选择 100 位好友')
  if (ids.length) createOpts.memberList = ids.slice(0, 20).map((userID) => ({ userID }))

  // 社群不开启话题（与 iOS isSupportTopic=false 对齐）
  if (isCommunity) {
    createOpts.isSupportTopic = false
  }

  let res: { data?: { group?: { groupID?: string }; groupID?: string } }
  try {
    res = await chat.createGroup(createOpts)
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e)
    if (/非法|invalid|group.?id/i.test(msg) && isCommunity) {
      throw new Error('社群群号非法：须以 @TGS# 开头，或留空由系统自动分配')
    }
    throw e instanceof Error ? e : new Error(msg || '创建失败')
  }

  const groupID = String(
    res?.data?.group?.groupID || res?.data?.groupID || customGroupID || '',
  )
  if (!groupID) throw new Error('创建成功但未返回群号')
  markGroupJustCreated(groupID)
  let warning = ''
  const invitedIds = new Set(ids.slice(0, 20))
  for (let index = 20; index < ids.length; index += 20) {
    try {
      const result = await chat.addGroupMember({ groupID, userIDList: ids.slice(index, index + 20) })
      const data = result?.data || {}
      for (const id of [...(data.successUserIDList || []), ...(data.existedUserIDList || [])]) invitedIds.add(id)
      if (data.failureUserIDList?.length || data.pendingUserIDList?.length) {
        warning = '群已创建，部分好友未能加入，请在群设置中查看并继续邀请。'
      }
    } catch (error) {
      warning = `群已创建，追加成员中断，请在群设置中继续邀请。${error instanceof Error ? error.message : ''}`
      break
    }
  }

  try {
    await syncGroupFromApp(groupID)
  } catch {
    /* 同步失败不阻断 */
  }

  const members =
    params.members?.length
      ? params.members
      : (params.memberIDList || []).map((userID) => ({ userID }))
  let tipMessage: ChatMessage | undefined
  try {
    tipMessage = await sendGroupCreateTip(groupID, isCommunity, {
      members: members.filter(m => invitedIds.has(m.userID)),
      inviterName: params.inviterName,
    })
  } catch {
    // 群已创建：提示消息被限流/实名策略拦截不能让用户再次建群。
    warning ||= '群已创建，但建群提示消息未发送成功。'
  }
  return {
    groupID,
    conversationID: `GROUP${groupID}`,
    tipMessage,
    warning,
  }
}

async function sendGroupCreateTip(
  groupID: string,
  isCommunity: boolean,
  extra?: { members?: CreateGroupMember[]; inviterName?: string },
) {
  const chat = getChat()
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const TYPES = (TencentCloudChat as any).TYPES
  const opUser =
    extra?.inviterName ||
    getLocalSession()?.userId ||
    chat.getLoginUser?.() ||
    ''
  const members = (extra?.members || []).filter((m) => m.userID)
  const names = members
    .map((m) => String(m.nick || m.userID).trim())
    .filter(Boolean)
  const content = names.length
    ? `邀请了${names.length > 3 ? `${names.slice(0, 3).join('、')}等${names.length}人` : names.join('、')}加入群聊`
    : isCommunity
      ? '创建社群'
      : '创建群组'
  const payload = {
    version: 4,
    cmd: isCommunity ? 1 : 0,
    businessID: 'group_create',
    opUser,
    content,
    members,
  }
  const message = chat.createCustomMessage({
    to: groupID,
    conversationType: TYPES.CONV_GROUP,
    payload: {
      data: JSON.stringify(payload),
      description: content,
      extension: 'group_create',
    },
  })
  const res = await chat.sendMessage(message)
  return res?.data?.message as ChatMessage | undefined
}

async function syncGroupFromApp(groupId: string) {
  const token = getToken()
  const res = await routedFetch('/im/sync/group/from/app', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    body: JSON.stringify({ group_id: groupId }),
  }, '/api')
  const json = (await res.json().catch(() => ({}))) as { code?: number; message?: string }
  if (json.code !== 200 && json.code !== 0) {
    console.warn('syncGroupFromApp:', json.message)
  }
}
