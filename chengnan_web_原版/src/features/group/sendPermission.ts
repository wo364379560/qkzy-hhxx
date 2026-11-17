export interface GroupSendState {
  allMuted: boolean;
  role: string;
  muteUntil?: number;
}

export function isGroupSendForbidden(state: GroupSendState, now = Date.now()) {
  const until = Number(state.muteUntil || 0);
  const personalMute = (until > 100_000_000_000 ? until : until * 1000) > now;
  const moderator = state.role === "Owner" || state.role === "Admin";
  return personalMute || (state.allMuted && !moderator);
}

/** Validate the destination group, including when forwarding from another chat. */
export function createGroupSendGuard(load: (groupID: string) => Promise<GroupSendState>) {
  return async (conversationID: string) => {
    if (!conversationID.startsWith("GROUP")) return;
    const groupID = conversationID.slice("GROUP".length);
    if (!groupID) throw new Error("群聊信息无效");
    const state = await load(groupID);
    if (isGroupSendForbidden(state)) {
      throw Object.assign(new Error("您在该群已被禁言，暂时无法发送或转发消息"), { code: 10017 });
    }
  };
}
