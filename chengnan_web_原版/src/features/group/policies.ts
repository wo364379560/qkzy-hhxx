import TUIChatEngine, { IMessageModel } from '@tencentcloud/chat-uikit-engine-lite';

export const GROUP_MEMBER_PROFILE_ACCESS_ATTRIBUTE = 'inner_attr_member_profile_access';
export const GROUP_VOICE_CALL_ATTRIBUTE = 'inner_attr_group_voice_call';
export const GROUP_VIDEO_CALL_ATTRIBUTE = 'inner_attr_group_video_call';

const ENABLED_VALUES = new Set(['1', 'true', 'yes', 'on']);

export function isMemberProfileAccessEnabled(attributes?: Record<string, string> | null) {
  const rawValue = attributes?.[GROUP_MEMBER_PROFILE_ACCESS_ATTRIBUTE]?.trim().toLowerCase();
  if (!rawValue) {
    return true;
  }

  return ENABLED_VALUES.has(rawValue);
}

export function serializeMemberProfileAccess(enabled: boolean) {
  return enabled ? '1' : '0';
}

function isEnabledValue(rawValue?: string | null) {
  if (!rawValue) {
    return false;
  }
  return ENABLED_VALUES.has(rawValue.trim().toLowerCase());
}

export function isGroupVoiceCallEnabled(attributes?: Record<string, string> | null) {
  return isEnabledValue(attributes?.[GROUP_VOICE_CALL_ATTRIBUTE]);
}

export function isGroupVideoCallEnabled(attributes?: Record<string, string> | null) {
  return isEnabledValue(attributes?.[GROUP_VIDEO_CALL_ATTRIBUTE]);
}

export function serializeGroupCallAccess(enabled: boolean) {
  return enabled ? '1' : '0';
}

export function isModeratorRole(role?: string) {
  return role === TUIChatEngine.TYPES.GRP_MBR_ROLE_OWNER
    || role === TUIChatEngine.TYPES.GRP_MBR_ROLE_ADMIN;
}

export function getGroupTipOperatorID(message: IMessageModel) {
  if (message.isRevoked) {
    return message.revokerInfo?.userID || message.from || '';
  }

  return message.payload?.operatorID || message.from || '';
}

/**
 * 对齐 chixin / ShangLan：群主/管理员撤回他人消息时静默隐藏，不展示「XX 撤回了一条消息」。
 * 自己撤回自己的消息仍展示撤回提示。
 */
export function shouldSilentlyHideRevokedMessage(
  message: IMessageModel,
  isGroup: boolean
) {
  if (!message?.isRevoked) {
    return false;
  }
  if (!isGroup) {
    return false;
  }
  const senderID = String(message.from || '').trim();
  // 只用 userID，避免 revoker 昵称误判
  const revokerID =
    String(message.revokerInfo?.userID || '').trim() || senderID;
  if (senderID && revokerID === senderID) {
    return false;
  }
  return Boolean(senderID && revokerID && revokerID !== senderID);
}

export function isModeratorManagedGroupTip(message: IMessageModel) {
  if (message.type !== TUIChatEngine.TYPES.MSG_GRP_TIP) {
    return false;
  }

  const operationType = message.payload?.operationType;
  if (operationType === TUIChatEngine.TYPES.GRP_TIP_MBR_SET_ADMIN) {
    return true;
  }
  if (operationType === TUIChatEngine.TYPES.GRP_TIP_MBR_CANCELED_ADMIN) {
    return true;
  }
  if (operationType === TUIChatEngine.TYPES.GRP_TIP_MBR_PROFILE_UPDATED) {
    return true;
  }
  if (operationType === TUIChatEngine.TYPES.GRP_TIP_BAN_AVCHATROOM_MEMBER) {
    return true;
  }
  if (operationType === TUIChatEngine.TYPES.GRP_TIP_UNBAN_AVCHATROOM_MEMBER) {
    return true;
  }
  if (operationType !== TUIChatEngine.TYPES.GRP_TIP_GRP_PROFILE_UPDATED) {
    return false;
  }

  const newGroupProfile = message.payload?.newGroupProfile || {};
  return Object.prototype.hasOwnProperty.call(newGroupProfile, 'muteAllMembers');
}

export function isMuteRelatedGroupTip(message: IMessageModel) {
  if (message.type !== TUIChatEngine.TYPES.MSG_GRP_TIP) {
    return false;
  }

  const operationType = message.payload?.operationType;
  if (operationType === TUIChatEngine.TYPES.GRP_TIP_MBR_PROFILE_UPDATED) {
    return true;
  }
  if (operationType === TUIChatEngine.TYPES.GRP_TIP_BAN_AVCHATROOM_MEMBER) {
    return true;
  }
  if (operationType === TUIChatEngine.TYPES.GRP_TIP_UNBAN_AVCHATROOM_MEMBER) {
    return true;
  }
  if (operationType !== TUIChatEngine.TYPES.GRP_TIP_GRP_PROFILE_UPDATED) {
    return false;
  }

  const newGroupProfile = message.payload?.newGroupProfile || {};
  return Object.prototype.hasOwnProperty.call(newGroupProfile, 'muteAllMembers');
}
