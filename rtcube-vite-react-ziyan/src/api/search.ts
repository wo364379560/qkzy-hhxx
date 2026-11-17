/**
 * 搜索加好友 / 群 — 对照 web-h5 api/search.ts
 */
import { chatGet } from './chatHttp'

export type SearchUserResult = {
  im_user_id: string
  wanan_id?: string
  nickname: string
  avatar_url?: string
  can_add_friend?: boolean
  add_blocked_reason?: string
}

export type SearchGroupResult = {
  group_id: string
  group_name: string
  avatar_url?: string
}

async function chatGetJson<T>(path: string): Promise<T | undefined> {
  return chatGet<T | undefined>(path, true)
}

export async function searchUsers(keyword: string): Promise<SearchUserResult[]> {
  const data = await chatGetJson<{ users?: SearchUserResult[] } | SearchUserResult[]>(
    `/users/search?q=${encodeURIComponent(keyword)}`,
  )
  if (Array.isArray(data)) return data
  return data?.users || []
}

export async function searchUsersExact(keyword: string): Promise<SearchUserResult[]> {
  const data = await chatGetJson<{ users?: SearchUserResult[] } | SearchUserResult[]>(
    `/users/search-exact?q=${encodeURIComponent(keyword)}`,
  )
  if (Array.isArray(data)) return data
  return data?.users || []
}

export async function searchGroupByID(groupID: string): Promise<SearchGroupResult | null> {
  const data = await chatGetJson<{ group?: SearchGroupResult | null }>(
    `/groups/search?group_id=${encodeURIComponent(groupID)}`,
  )
  return data?.group || null
}
