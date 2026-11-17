/**
 * 好友申请 / 群列表 / 群申请 — 对照 web-h5 + Android 通讯录
 */
import { FRIEND_GREETING } from '@/config/runtime'
import { fetchFriendList, getChat, sendTextMessage, TencentCloudChat } from './ImClient'
import { getConversationProfile } from './profileActions'

// eslint-disable-next-line @typescript-eslint/no-explicit-any
type AnyChat = any

export type FriendApplication = {
  userID: string
  nick: string
  avatar: string
  wording: string
  type: string
}

export type GroupApplication = {
  groupID: string
  groupName: string
  applicant: string
  nick: string
  avatar: string
  wording: string
  applicationType: number
}

export type JoinedGroup = {
  groupID: string
  name: string
  avatar: string
  memberCount: number
}

export async function fetchFriendApplications(): Promise<FriendApplication[]> {
  const c = getChat()
  const res = await c.getFriendApplicationList()
  const list = (res?.data?.friendApplicationList ?? res?.data ?? []) as AnyChat[]
  return list
    .map((a) => ({
      userID: String(a.userID || a.from || ''),
      nick: String(a.nick || a.userID || ''),
      avatar: String(a.avatar || ''),
      wording: String(a.wording || a.addWording || ''),
      type: String(a.type || ''),
    }))
    .filter((a) => a.userID)
}

export async function acceptFriendApplication(userID: string) {
  const c = getChat()
  const TYPES = (TencentCloudChat as AnyChat).TYPES
  await c.acceptFriendApplication({
    userID,
    type: TYPES?.SNS_APPLICATION_AGREE_AND_ADD || 'SNS_APPLICATION_AGREE_AND_ADD',
  })
  // 删除后再加好友时，TIM 关系有短暂不同步，立刻发消息会 20009
  await new Promise((r) => window.setTimeout(r, 400))
  try {
    await sendTextMessage(`C2C${userID}`, FRIEND_GREETING)
  } catch {
    /* ignore */
  }
}

export async function refuseFriendApplication(userID: string) {
  const c = getChat()
  await c.refuseFriendApplication({ userID })
}

export async function fetchJoinedGroups(): Promise<JoinedGroup[]> {
  const c = getChat()
  const res = await c.getGroupList()
  const list = (res?.data?.groupList ?? []) as AnyChat[]
  return list.map((g) => ({
    groupID: String(g.groupID || ''),
    name: String(g.name || g.groupID || ''),
    avatar: String(g.avatar || ''),
    memberCount: Number(g.memberCount || 0),
  })).filter((g) => g.groupID)
}

/** 入群申请（新的朋友 · 群） */
export async function fetchGroupApplications(): Promise<GroupApplication[]> {
  const c = getChat()
  try {
    const res = await c.getGroupApplicationList?.()
    const list = (res?.data?.groupApplicationList ?? res?.data ?? []) as AnyChat[]
    return list
      .map((a) => ({
        groupID: String(a.groupID || ''),
        groupName: String(a.groupName || a.groupID || ''),
        applicant: String(a.applicant || a.userID || a.from || ''),
        nick: String(a.nick || a.applicantNick || a.userID || ''),
        avatar: String(a.avatar || ''),
        wording: String(a.wording || a.applyMessage || ''),
        applicationType: Number(a.applicationType ?? a.type ?? 0),
      }))
      .filter((a) => a.groupID && a.applicant)
  } catch {
    return []
  }
}

export async function handleGroupApplication(
  application: GroupApplication,
  handleAction: 'Agree' | 'Reject',
) {
  const c = getChat()
  const TYPES = (TencentCloudChat as AnyChat).TYPES
  await c.handleGroupApplication({
    handleAction:
      handleAction === 'Agree'
        ? TYPES?.GRP_APPLICATION_HANDLE_ACTION_AGREE || 'Agree'
        : TYPES?.GRP_APPLICATION_HANDLE_ACTION_REJECT || 'Reject',
    application: {
      groupID: application.groupID,
      from: application.applicant,
      applicationType: application.applicationType,
    },
  })
}

export type FriendRelation = 'none' | 'one' | 'both'

export async function checkFriendRelation(userID: string): Promise<FriendRelation> {
  const id = String(userID || '').trim()
  if (!id) return 'none'
  const c = getChat()
  try {
    const TYPES = (TencentCloudChat as AnyChat).TYPES
    const res = await c.checkFriend({
      userIDList: [id],
      type: TYPES?.SNS_CHECK_TYPE_BOTH ?? 'SNS_CHECK_TYPE_BOTH',
    })
    const row = res?.data?.successUserIDList?.[0] || res?.data?.[0]
    return parseFriendRelation(row?.relation ?? row?.resultType ?? row?.type)
  } catch {
    const friends = await fetchFriendList()
    return friends.some((f) => f.userID === id) ? 'both' : 'none'
  }
}

export async function checkIsFriend(userID: string): Promise<boolean> {
  const rel = await checkFriendRelation(userID)
  return rel === 'both' || rel === 'one'
}

/** SDK：CheckResult_Type_NoRelation / AWithB / BWithA / BothWay。code=0 只表示查询成功，不是好友。 */
function parseFriendRelation(relation: unknown): FriendRelation {
  const r = String(relation || '').toUpperCase().replace(/[_-]/g, '')
  if (!r || r.includes('NORELATION')) return 'none'
  if (r.includes('BOTHWAY')) return 'both'
  if (r.includes('AWITHB') || r.includes('BWITHA')) return 'one'
  return 'none'
}

/**
 * 删除后再加好友时，TIM 好友关系可能短暂不同步，转发/合并会报 20009。
 * 发送前刷新关系并确保 C2C 会话存在。
 */
export async function ensureC2CCanSend(userID: string): Promise<void> {
  const id = String(userID || '').trim()
  if (!id) throw new Error('无效的会话对象')
  let rel = await checkFriendRelation(id)
  if (rel === 'none') {
    const friends = await fetchFriendList()
    if (friends.some((f) => f.userID === id)) rel = 'both'
  }
  if (rel === 'none') {
    throw new Error('对方还不是你的好友，请先添加好友后再发消息')
  }
  try {
    await getConversationProfile(`C2C${id}`)
  } catch {
    /* 会话会在首次发送时创建 */
  }
}

export async function addFriend(userID: string, wording = '你好，我想加你为好友') {
  const c = getChat()
  const TYPES = (TencentCloudChat as AnyChat).TYPES
  try {
    const res = await c.addFriend({
      to: userID,
      source: 'AddSource_Type_Web',
      wording,
      type: TYPES?.SNS_ADD_TYPE_BOTH ?? 'SNS_ADD_TYPE_BOTH',
    })
    const code = Number(res?.code ?? res?.data?.code ?? 0)
    const info = String(res?.data?.message || res?.message || '')
    if (code && code !== 0 && code !== 30539 && code !== 30515) {
      throw Object.assign(new Error(info || `添加失败（${code}）`), { code })
    }
  } catch (e) {
    const msg = formatFriendAddError(e)
    if (msg.includes('已发送') || msg.includes('已经是好友')) return
    throw new Error(msg)
  }
}

export function formatFriendAddError(err: unknown): string {
  const extra = err as { code?: number; message?: string }
  const raw = extra?.message || (err instanceof Error ? err.message : String(err || ''))
  const lower = raw.toLowerCase()
  const code = Number(extra?.code) || Number(/(\d{4,5})/.exec(raw)?.[1]) || 0

  if (
    code === 38000 ||
    lower.includes('cross-owner') ||
    raw.includes('其他服务空间') ||
    raw.includes('跨归属')
  ) {
    return '当前账号属于其他服务空间无法直接添加。'
  }
  if (code === 10016 || lower.includes('before callback failed')) {
    if (lower.includes('cross-owner') || raw.includes('跨归属')) {
      return '当前账号属于其他服务空间无法直接添加。'
    }
    return '当前处于系统静默时段或被系统规则拒绝，暂时不能添加好友，请稍后再试'
  }
  if (code === 30539) return '好友请求已发送，等待对方确认'
  if (code === 30515 || lower.includes('already friend') || lower.includes('friend_exist')) {
    return '你们已经是好友了'
  }
  if (code === 30525 || lower.includes('in black list') || lower.includes('in blacklist')) {
    return '添加失败：对方在黑名单中，请先移出黑名单后再添加'
  }
  if (code === 30010 || code === 30541) return '添加失败：你的好友数量已达上限'
  if (code === 30014 || code === 30540) return '添加失败：对方好友数量已达上限'
  if (code === 30003 || lower.includes('user not found')) return '添加失败：用户不存在'
  if (code === 30004) return '添加失败：对方设置了不允许加好友'
  if (/[\u4e00-\u9fff]/.test(raw)) return raw.startsWith('添加') ? raw : raw
  return code ? `添加失败，请稍后重试（${code}）` : '添加失败，请稍后重试'
}

export async function fetchBlacklistProfiles(): Promise<
  Array<{ userID: string; nick: string; avatar: string }>
> {
  const c = getChat()
  const res = await c.getBlacklist()
  const ids = ((res?.data || []) as Array<string | { userID?: string }>)
    .map((x) => (typeof x === 'string' ? x : x.userID || ''))
    .filter(Boolean)
  if (!ids.length) return []
  const profiles = await c.getUserProfile({ userIDList: ids.slice(0, 100) })
  const list = (profiles?.data ?? []) as AnyChat[]
  const map = Object.fromEntries(
    list.map((u) => [
      String(u.userID || ''),
      { userID: String(u.userID || ''), nick: String(u.nick || u.userID || ''), avatar: String(u.avatar || '') },
    ]),
  )
  return ids.map((id) => map[id] || { userID: id, nick: id, avatar: '' })
}
