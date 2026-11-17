export const AT_MEMBER_EVENT = 'cn-at-member'

export type AtMemberPayload = {
  groupId: string
  userID: string
  nick: string
}

export function emitAtMember(payload: AtMemberPayload) {
  window.dispatchEvent(new CustomEvent(AT_MEMBER_EVENT, { detail: payload }))
}
