import { chatGet, chatPost } from './chatHttp'

export type GroupNotice = {
  group_id: string
  content: string
  updated_at?: string
  updated_by?: string
}

export async function getGroupNotice(groupID: string) {
  return chatGet<GroupNotice>(`/groups/notice?group_id=${encodeURIComponent(groupID)}`)
}

export async function saveGroupNotice(groupID: string, content: string) {
  return chatPost<GroupNotice>('/groups/notice', { group_id: groupID, content })
}
