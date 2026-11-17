import TUIChatEngine, {
  IMessageModel,
  IConversationModel,
  TUIStore,
  StoreName,
} from '@tencentcloud/chat-uikit-engine-lite';
import { computed, ref } from '../../../adapter-vue';
import { getGroupConversationContext } from '../utils/groupConversationContext';

export function useGroupMuteState() {
  const currentUserID = ref('');
  const currentSelfRole = ref('');
  const currentSelfMuteUntil = ref(0);
  const currentGroupMuteAllMembers = ref(false);
  const lastMuteStateSyncSignature = ref('');

  const isPrivilegedGroupMember = computed(
    () =>
      currentSelfRole.value === TUIChatEngine.TYPES.GRP_MBR_ROLE_OWNER
      || currentSelfRole.value === TUIChatEngine.TYPES.GRP_MBR_ROLE_ADMIN,
  );

  const isCurrentUserMuted = computed(() => {
    if (currentGroupMuteAllMembers.value && !isPrivilegedGroupMember.value) {
      return true;
    }
    return currentSelfMuteUntil.value * 1000 > Date.now();
  });

  function ensureCurrentUserID() {
    if (!currentUserID.value) {
      currentUserID.value = TUIStore.getData(StoreName.USER, 'userProfile')?.userID || '';
    }
  }

  function resetMuteState() {
    currentSelfRole.value = '';
    currentSelfMuteUntil.value = 0;
    currentGroupMuteAllMembers.value = false;
    lastMuteStateSyncSignature.value = '';
  }

  function isMuteRelatedGroupTip(message?: IMessageModel) {
    if (!message || message.type !== TUIChatEngine.TYPES.MSG_GRP_TIP) {
      return false;
    }

    const operationType = message.payload?.operationType;
    if (
      operationType === TUIChatEngine.TYPES.GRP_TIP_MBR_PROFILE_UPDATED
      || operationType === TUIChatEngine.TYPES.GRP_TIP_BAN_AVCHATROOM_MEMBER
      || operationType === TUIChatEngine.TYPES.GRP_TIP_UNBAN_AVCHATROOM_MEMBER
    ) {
      return true;
    }

    if (operationType !== TUIChatEngine.TYPES.GRP_TIP_GRP_PROFILE_UPDATED) {
      return false;
    }

    const newGroupProfile = message.payload?.newGroupProfile || {};
    return Object.prototype.hasOwnProperty.call(newGroupProfile, 'muteAllMembers');
  }

  function updateMuteStateFromGroupProfile(group?: Record<string, any>) {
    if (!group) {
      return;
    }
    currentGroupMuteAllMembers.value = Boolean(group?.muteAllMembers);
    currentSelfRole.value = String(group?.selfInfo?.role || currentSelfRole.value || '') || '';
  }

  function updateMuteStateFromMemberList(memberList: Array<Record<string, any>> = []) {
    ensureCurrentUserID();
    if (!currentUserID.value || !Array.isArray(memberList)) {
      return;
    }

    const selfMember = memberList.find(
      (member) => member?.userID === currentUserID.value,
    );
    if (!selfMember) {
      return;
    }

    currentSelfMuteUntil.value = Number(selfMember?.muteUntil || 0);
    currentSelfRole.value = String(selfMember?.role || currentSelfRole.value || '') || '';
  }

  async function syncGroupMuteState(
    targetGroupID?: string,
    options: {
      force?: boolean;
    } = {},
  ) {
    ensureCurrentUserID();
    if (!targetGroupID || !currentUserID.value) {
      resetMuteState();
      return;
    }

    try {
      const context = await getGroupConversationContext(
        targetGroupID,
        currentUserID.value,
        options,
      );
      updateMuteStateFromGroupProfile(context.group);
      updateMuteStateFromMemberList(context.selfMember ? [context.selfMember] : []);
    } catch (error) {
      console.warn('syncGroupMuteState failed:', error);
    }
  }

  function onCurrentGroupUpdate(group: Record<string, any>, groupID?: string) {
    if (!groupID || group?.groupID !== groupID) {
      return;
    }
    updateMuteStateFromGroupProfile(group);
  }

  function onCurrentGroupMemberListUpdate(memberList: Array<Record<string, any>>, groupID?: string) {
    if (!groupID) {
      return;
    }
    updateMuteStateFromMemberList(memberList || []);
  }

  function onMessageListUpdate(
    messageList: IMessageModel[] = [],
    options: {
      isGroup: boolean;
      groupID?: string;
      conversationID?: string;
    },
  ) {
    if (!options.isGroup || !options.groupID || !messageList.length) {
      return;
    }

    const lastMessage = messageList[messageList.length - 1];
    if (
      !lastMessage
      || lastMessage.conversationID !== options.conversationID
      || !isMuteRelatedGroupTip(lastMessage)
    ) {
      return;
    }

    const signature = `${lastMessage.ID || ''}:${lastMessage.time || 0}:${lastMessage.payload?.operationType || ''}`;
    if (lastMuteStateSyncSignature.value === signature) {
      return;
    }

    lastMuteStateSyncSignature.value = signature;
    void syncGroupMuteState(options.groupID, { force: true });
  }

  function handleConversationChange(conversation?: IConversationModel) {
    const conversationID = conversation?.conversationID || '';
    if (!conversationID.startsWith(TUIChatEngine.TYPES.CONV_GROUP)) {
      resetMuteState();
      return {
        isGroup: false,
        groupID: undefined,
      };
    }

    const groupID = conversationID.replace(TUIChatEngine.TYPES.CONV_GROUP, '');
    void syncGroupMuteState(groupID);
    return {
      isGroup: true,
      groupID,
    };
  }

  return {
    isCurrentUserMuted,
    resetMuteState,
    onCurrentGroupUpdate,
    onCurrentGroupMemberListUpdate,
    onMessageListUpdate,
    handleConversationChange,
  };
}
