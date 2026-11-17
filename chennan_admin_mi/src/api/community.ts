import request from './request'
import type {
  AdminCommunityComment,
  AdminCommunityPost,
  ApiResponse,
  PageResult,
} from '@/types'

export interface CommunityPostListParams {
  page?: number
  pageSize?: number
  keyword?: string
  postType?: string
  ownerID?: number
  userID?: number
}

export interface UpdateCommunityPostParams {
  postType?: string
  title?: string
  content: string
}

export interface UpdateCommunityUserRestrictionParams {
  communityPostDisabled?: boolean
  communityCommentDisabled?: boolean
}

export function getCommunityPostList(params: CommunityPostListParams = {}) {
  return request.get<ApiResponse<PageResult<AdminCommunityPost>>>('/community/posts', { params })
}

// 新产品语义别名：省钱社区内容
export const getSavingCommunityPostList = getCommunityPostList

export function getCommunityComments(postID: number) {
  return request.get<ApiResponse<AdminCommunityComment[]>>(`/community/posts/${postID}/comments`)
}

export const getSavingCommunityComments = getCommunityComments

export function updateCommunityPost(postID: number, params: UpdateCommunityPostParams) {
  return request.post<ApiResponse<null>>(`/community/posts/${postID}/update`, params)
}

export const updateSavingCommunityPost = updateCommunityPost

export function deleteCommunityPost(postID: number) {
  return request.post<ApiResponse<null>>(`/community/posts/${postID}/delete`)
}

export const deleteSavingCommunityPost = deleteCommunityPost

export function deleteCommunityComment(commentID: number) {
  return request.post<ApiResponse<null>>(`/community/comments/${commentID}/delete`)
}

export const deleteSavingCommunityComment = deleteCommunityComment

export function updateCommunityUserRestriction(userID: number, params: UpdateCommunityUserRestrictionParams) {
  return request.post<ApiResponse<null>>(`/community/users/${userID}/restrictions`, params)
}

export const updateSavingCommunityUserRestriction = updateCommunityUserRestriction
