import request from './request'
import type { ApiResponse, Group, GroupMember, GroupNotice, PageResult } from '@/types'

export interface GroupListParams {
  page?: number
  pageSize?: number
  keyword?: string
  status?: string
  groupType?: GroupType
  ownerID?: number
}

export type GroupType = 'Work' | 'Public' | 'Meeting' | 'AVChatRoom' | 'Community'

export const GROUP_TYPE_OPTIONS: { value: GroupType; label: string; desc: string }[] = [
  { value: 'Work',        label: '好友工作群',   desc: '类似微信群，仅限群成员邀请加入' },
  { value: 'Public',      label: '普通群组',     desc: '类似 QQ 群，需群主/管理员审批' },
  { value: 'Meeting',     label: '临时会议群',   desc: '可查看入群前消息，适合音视频会议' },
  { value: 'AVChatRoom',  label: '直播群',       desc: '无人数上限，不存历史消息，适合直播弹幕' },
  { value: 'Community',   label: '社群',         desc: '最多 10 万人，支持历史消息，无需审批' },
]

export const GROUP_TYPE_CREATE_OPTIONS: { value: GroupType; label: string; desc: string }[] = [
  { value: 'Public',      label: '普通群组', desc: '类似 QQ 群，需群主/管理员审批' },
  { value: 'Community',   label: '社群',     desc: '最多 10 万人，支持历史消息，无需审批' },
]

export interface CreateGroupParams {
  name: string
  groupType?: GroupType
  ownerID?: number       // 管理员建群时指定归属人 ID
  memberIDs?: number[]
  ownerMemberID?: number  // 指定群主（本地用户 ID），不传则 TIM 默认
}

export function getGroupList(params: GroupListParams = {}) {
  return request.get<ApiResponse<PageResult<Group>>>('/groups', { params })
}

export function getGroupDetail(groupId: number) {
  return request.get<ApiResponse<Group>>(`/groups/${groupId}`)
}

export function getGroupNotice(groupId: number) {
  return request.get<ApiResponse<GroupNotice>>(`/groups/${groupId}/notice`)
}

export function saveGroupNotice(groupId: number, content: string) {
  return request.post<ApiResponse<GroupNotice>>(`/groups/${groupId}/notice`, {
    content,
  })
}

export function createGroup(params: CreateGroupParams) {
  return request.post<ApiResponse<Group>>('/groups', params)
}

export function addGroupMembers(groupId: number, memberIDs: number[]) {
  return request.post<ApiResponse<null>>(`/groups/${groupId}/members`, { memberIDs })
}

export function removeGroupMembers(groupId: number, memberIDs: number[]) {
  return request.post<ApiResponse<null>>(`/groups/${groupId}/members/remove`, { memberIDs })
}

export function dismissGroup(groupId: number) {
  return request.post<ApiResponse<null>>(`/groups/${groupId}/delete`)
}

export function getGroupMembers(groupId: number) {
  return request.get<ApiResponse<GroupMember[]>>(`/groups/${groupId}/members`)
}

export function muteGroup(groupId: number, mute: boolean) {
  return request.post<ApiResponse<null>>(`/groups/${groupId}/mute`, { mute })
}

export interface BatchMuteGroupResultDetail {
  groupID: number
  groupName: string
  success: boolean
  message: string
}

export interface BatchMuteGroupResult {
  success: number
  failed: number
  details: BatchMuteGroupResultDetail[]
}

export function batchMuteGroups(groupIDs: number[]) {
  return request.post<ApiResponse<BatchMuteGroupResult>>('/groups/batch-mute', { groupIDs })
}

export function batchUnmuteGroups(groupIDs: number[]) {
  return request.post<ApiResponse<BatchMuteGroupResult>>('/groups/batch-unmute', { groupIDs })
}

export function repairGroup(groupId: number) {
  return request.post<ApiResponse<null>>(`/groups/${groupId}/repair`)
}

export function syncGroupsPreview() {
  return request.post<ApiResponse<import('@/types').SyncGroupPreview[]>>('/groups/sync/preview')
}

export interface SyncConfirmItem {
  imGroupId: string
  action: import('@/types').SyncAction
}

export function syncGroupsConfirm(selections: SyncConfirmItem[]) {
  return request.post<ApiResponse<{ success: number; failed: number }>>('/groups/sync/confirm', { selections })
}
