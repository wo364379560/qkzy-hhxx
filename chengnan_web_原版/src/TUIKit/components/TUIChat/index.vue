<template>
  <div class="chat">
    <div :class="['tui-chat', !isPC && 'tui-chat-h5']">
      <div
        v-if="!currentConversationID"
        :class="['tui-chat-default', !isPC && 'tui-chat-h5-default']"
      >
        <slot />
      </div>
      <div
        v-if="currentConversationID"
        :class="['tui-chat', !isPC && 'tui-chat-h5']"
      >
        <ChatHeader
          :class="[
            'tui-chat-header',
            !isPC && 'tui-chat-H5-header',
            isUniFrameWork && 'tui-chat-uniapp-header',
          ]"
          :isGroup="isGroup"
          :peerLastOnlineLabel="peerLastOnlineLabel"
          :headerExtensionList="headerExtensionList"
          @closeChat="closeChat"
          @openGroupManagement="handleGroup"
          @openConversationProfile="openCurrentConversationContactInfo"
        />
        <GroupNoticeBar v-if="isGroup && !isNotInGroup" :groupID="groupID" />
        <GroupPinnedMessageBar
          :messages="currentGroupPinnedMessages"
          :currentIndex="currentPinnedIndex"
          @cycle="handleCyclePinnedMessage"
          @jump="handleJumpPinnedMessage"
          @unpin="handleUnpinPinnedMessage"
        />
        <Forward @toggleMultipleSelectMode="toggleMultipleSelectMode" />
        <MessageList
          ref="messageListRef"
          :class="[
            'tui-chat-message-list',
            !isPC && 'tui-chat-h5-message-list',
          ]"
          :isGroup="isGroup"
          :groupID="groupID"
          :isNotInGroup="isNotInGroup"
          :isMultipleSelectMode="isMultipleSelectMode"
          @handleEditor="handleEditor"
          @closeInputToolBar="() => changeToolbarDisplayType('none')"
          @openContactInfo="openAvatarContactInfo"
          @toggleMultipleSelectMode="toggleMultipleSelectMode"
          @groupMessagePinChanged="handleGroupMessagePinChanged"
        />
        <div
          v-if="isNotInGroup"
          :class="{
            'tui-chat-leave-group': true,
            'tui-chat-leave-group-mobile': isMobile,
          }"
        >
          {{ leaveGroupReasonText }}
        </div>
        <MultipleSelectPanel
          v-else-if="isMultipleSelectMode"
          @oneByOneForwardMessage="oneByOneForwardMessage"
          @mergeForwardMessage="mergeForwardMessage"
          @toggleMultipleSelectMode="toggleMultipleSelectMode"
        />
        <template v-else>
          <MessageInputToolbar
            v-if="isInputToolbarShow && !isCurrentUserMuted"
            :class="[
              'tui-chat-message-input-toolbar',
              !isPC && 'tui-chat-h5-message-input-toolbar',
              isUniFrameWork && 'tui-chat-uni-message-input-toolbar',
            ]"
            :displayType="inputToolbarDisplayType"
            @insertEmoji="insertEmoji"
            @changeToolbarDisplayType="changeToolbarDisplayType"
            @scrollToLatestMessage="scrollToLatestMessage"
          />
          <MessageInput
            ref="messageInputRef"
            :class="[
              'tui-chat-message-input',
              !isPC && 'tui-chat-h5-message-input',
              isUniFrameWork && 'tui-chat-uni-message-input',
              isWeChat && 'tui-chat-wx-message-input',
            ]"
            :enableAt="featureConfig.InputMention"
            :isMuted="isCurrentUserMuted"
            :muteText="mutedText"
            :placeholder="TUITranslateService.t('TUIChat.请输入消息')"
            :inputToolbarDisplayType="inputToolbarDisplayType"
            @changeToolbarDisplayType="changeToolbarDisplayType"
          />
        </template>
        <ContactInfoPopup
          :visible="showAvatarContactInfo"
          @close="closeAvatarContactInfo"
          @switchConversation="handleAvatarContactSwitchConversation"
        />
      </div>
      <!-- Group Management -->
      <div
        v-if="
          !isNotInGroup &&
          !isApp &&
          isUniFrameWork &&
          isGroup &&
          headerExtensionList.length > 0
        "
        class="group-profile"
        @click="handleGroup"
      >
        {{ headerExtensionList[0].text }}
      </div>
    </div>
  </div>
</template>
<script lang="ts" setup>
import GroupNoticeBar from "./group-notice-bar.vue";
import {
  ref,
  onMounted,
  onUnmounted,
  computed,
  defineAsyncComponent,
} from "../../adapter-vue";
import TUIChatEngine, {
  TUITranslateService,
  TUIConversationService,
  TUIGroupService,
  TUIStore,
  StoreName,
  IMessageModel,
  IConversationModel,
} from "@tencentcloud/chat-uikit-engine-lite";
import TUICore, {
  TUIConstants,
  ExtensionInfo,
} from "@tencentcloud/tui-core-lite";
import ContactInfoPopup from "./contact-info-popup.vue";
import GroupPinnedMessageBar from "./group-pinned-message-bar.vue";
import {
  isPC,
  isWeChat,
  isUniFrameWork,
  isMobile,
  isApp,
} from "../../utils/env";
import {
  ToolbarDisplayType,
  type IUserStatus,
  type IUserStatusMap,
} from "../../interface";
import TUIChatConfig from "./config";
import { useGroupMuteState } from "./composables/useGroupMuteState";
import { useGroupPinnedMessage } from "./composables/useGroupPinnedMessage";
import {
  GROUP_MEMBER_PROFILE_ACCESS_ATTRIBUTE,
  GROUP_VIDEO_CALL_ATTRIBUTE,
  GROUP_VOICE_CALL_ATTRIBUTE,
  isMemberProfileAccessEnabled,
} from "@/features/group/policies";
import { getUserLastOnline, type GroupMessagePin } from "@/api";
import { isUserOnlineStatus } from "@/features/presence/invisible";
import { formatLastOnlineText, resolveLastOnlineAt } from "@/features/presence/last-online";
import { Toast, TOAST_TYPE } from "../common/Toast/index";
import { preloadGroupConversationContext } from "./utils/groupConversationContext";

const ChatHeader = defineAsyncComponent(() => import("./chat-header/index.vue"));
const MessageList = defineAsyncComponent(() => import("./message-list/index.vue"));
const MessageInput = defineAsyncComponent(() => import("./message-input/index.vue"));
const MultipleSelectPanel = defineAsyncComponent(
  () => import("./mulitple-select-panel/index.vue"),
);
const Forward = defineAsyncComponent(() => import("./forward/index.vue"));
const MessageInputToolbar = defineAsyncComponent(
  () => import("./message-input-toolbar/index.vue"),
);

const emits = defineEmits(["closeChat"]);

const groupID = ref<string | undefined>(undefined);
const isGroup = ref(false);
const isNotInGroup = ref(false);
const notInGroupReason = ref<number>();
const currentConversation = ref<IConversationModel>();
const currentConversationID = ref();
const currentGroupAttributes = ref<Record<string, string>>({});
const currentGroupAttributesGroupID = ref("");
const currentGroupMemberRoleMap = ref<Record<string, string>>({});
const peerLastOnlineAt = ref("");
const peerPresenceNow = ref(Date.now());
const peerLastOnlineLabel = computed(() => formatLastOnlineText(peerLastOnlineAt.value, peerPresenceNow.value));
let peerObservedOfflineAt = 0;
let peerPresenceTimer: ReturnType<typeof setInterval> | undefined;
let peerPresenceDisposed = false;
let peerPresenceFetching = false;
let peerLastOnlineRequestSeq = 0;
const displayOnlineStatus = ref(false);
const userOnlineStatusMap = ref<IUserStatusMap>({});
let peerOnlineState: boolean | null = null;
const isMultipleSelectMode = ref(false);
const inputToolbarDisplayType = ref<ToolbarDisplayType>("none");
const showAvatarContactInfo = ref(false);
const messageInputRef = ref<any>();
const messageListRef = ref<any>();
const headerExtensionList = ref<ExtensionInfo[]>([]);
const featureConfig = TUIChatConfig.getFeatureConfig();
const {
  isCurrentUserMuted,
  resetMuteState,
  onCurrentGroupUpdate: syncMuteStateWithGroup,
  onCurrentGroupMemberListUpdate: syncMuteStateWithMemberList,
  onMessageListUpdate: syncMuteStateWithMessageList,
  handleConversationChange,
} = useGroupMuteState();
const {
  currentGroupPinnedMessage,
  currentGroupPinnedMessages,
  currentPinnedIndex,
  loadCurrentGroupPinnedMessage,
  onCurrentGroupPinnedMessageUpdate,
  cyclePinnedMessage,
  jumpToPinnedMessage,
  unpinGroupPinnedMessage,
} = useGroupPinnedMessage();

const mutedText = computed(() =>
  TUITranslateService.t("TUIChat.您已被管理员禁言")
);

const currentUserID = computed(
  () => TUIStore.getData(StoreName.USER, "userProfile")?.userID || ""
);

onMounted(() => {
  peerPresenceTimer = setInterval(() => {
    peerPresenceNow.value = Date.now();
    if (!document.hidden && getPeerOnlineState(currentConversation.value) === false && !peerPresenceFetching) {
      void syncPeerLastOnline(currentConversation.value);
    }
  }, 30000);
  TUIStore.watch(StoreName.CONV, {
    currentConversation: onCurrentConversationUpdate,
  });
  TUIStore.watch(StoreName.GRP, {
    currentGroup: onCurrentGroupUpdate,
    currentGroupMemberList: onCurrentGroupMemberListUpdate,
  });
  TUIStore.watch(StoreName.CHAT, {
    messageList: onMessageListUpdate,
  });
  TUIStore.watch(StoreName.CUSTOM, {
    currentGroupPinnedMessage: onCurrentGroupPinnedMessageUpdate,
  });
  TUIStore.watch(StoreName.USER, {
    displayOnlineStatus: onDisplayOnlineStatusUpdated,
    userStatusList: onUserStatusListUpdated,
  });
  syncOnlineStatusFromStore();
});

onUnmounted(() => {
  peerPresenceDisposed = true;
  clearInterval(peerPresenceTimer);
  clearPeerLastOnline();
  TUIStore.unwatch(StoreName.CONV, {
    currentConversation: onCurrentConversationUpdate,
  });
  TUIStore.unwatch(StoreName.GRP, {
    currentGroup: onCurrentGroupUpdate,
    currentGroupMemberList: onCurrentGroupMemberListUpdate,
  });
  TUIStore.unwatch(StoreName.CHAT, {
    messageList: onMessageListUpdate,
  });
  TUIStore.unwatch(StoreName.CUSTOM, {
    currentGroupPinnedMessage: onCurrentGroupPinnedMessageUpdate,
  });
  TUIStore.unwatch(StoreName.USER, {
    displayOnlineStatus: onDisplayOnlineStatusUpdated,
    userStatusList: onUserStatusListUpdated,
  });
  reset();
});

const isInputToolbarShow = computed<boolean>(() => {
  return isUniFrameWork ? inputToolbarDisplayType.value !== "none" : true;
});

const leaveGroupReasonText = computed<string>(() => {
  let text = "";
  switch (notInGroupReason.value) {
    case 4:
      text = TUITranslateService.t("TUIChat.您已被管理员移出群聊");
      break;
    case 5:
      text = TUITranslateService.t("TUIChat.该群聊已被解散");
      break;
    case 8:
      text = TUITranslateService.t("TUIChat.您已退出该群聊");
      break;
    default:
      text = TUITranslateService.t("TUIChat.您已退出该群聊");
      break;
  }
  return text;
});

const reset = () => {
  TUIConversationService.switchConversation("");
  closeAvatarContactInfo();
};

const closeChat = (conversationID: string) => {
  emits("closeChat", conversationID);
  reset();
};

const insertEmoji = (emojiObj: object) => {
  messageInputRef.value?.insertEmoji(emojiObj);
};

const handleEditor = (message: IMessageModel, type: string) => {
  if (!message || !type) return;
  switch (type) {
    case "reference":
      // todo
      break;
    case "reply":
      // todo
      break;
    case "reedit":
      if (message?.payload?.text) {
        messageInputRef?.value?.reEdit(message?.payload?.text);
      }
      break;
    default:
      break;
  }
};

const handleGroup = () => {
  const extension = headerExtensionList.value[0];
  if (!extension || !groupID.value) {
    return;
  }
  extension?.listener?.onClicked?.({ groupID: groupID.value });
};

function changeToolbarDisplayType(type: ToolbarDisplayType) {
  inputToolbarDisplayType.value =
    inputToolbarDisplayType.value === type ? "none" : type;
  if (inputToolbarDisplayType.value !== "none" && isUniFrameWork) {
    (globalThis as any).uni?.$emit?.("scroll-to-bottom");
  }
}

function handleGroupMessagePinChanged() {
  void loadCurrentGroupPinnedMessage(isGroup.value, groupID.value, {
    force: true,
    ensureAfterInflight: true,
  });
}

function scrollToLatestMessage() {
  messageListRef.value?.scrollToLatestMessage();
}

function handleLocatePinnedMessage(messageID?: string) {
  const targetID = messageID || currentGroupPinnedMessage.value?.message_id;
  if (!targetID) {
    return;
  }
  const targetMessageDom = document.getElementById(`tui-${targetID}`);
  if (targetMessageDom?.scrollIntoView) {
    targetMessageDom.scrollIntoView({
      behavior: "smooth",
      block: "nearest",
    });
    targetMessageDom.classList.add("group-pin-target-blink");
    window.setTimeout(() => {
      targetMessageDom.classList.remove("group-pin-target-blink");
    }, 1800);
  }
}

function handleCyclePinnedMessage() {
  const next = cyclePinnedMessage();
  handleLocatePinnedMessage(next?.message_id);
}

function handleJumpPinnedMessage(pin: GroupMessagePin) {
  const next = jumpToPinnedMessage(pin);
  handleLocatePinnedMessage(next?.message_id);
}

async function handleUnpinPinnedMessage(pin: GroupMessagePin) {
  try {
    await unpinGroupPinnedMessage(pin);
    Toast({
      message: "已取消置顶",
      type: TOAST_TYPE.SUCCESS,
    });
  } catch (error: any) {
    Toast({
      message: error?.message || "取消置顶失败",
      type: TOAST_TYPE.ERROR,
    });
  }
}

function isPrivilegedGroupRole(role?: string) {
  return (
    role === TUIChatEngine.TYPES.GRP_MBR_ROLE_OWNER ||
    role === TUIChatEngine.TYPES.GRP_MBR_ROLE_ADMIN
  );
}

function getCurrentUserGroupRole() {
  return String(
    currentConversation.value?.groupProfile?.selfInfo?.role ||
      currentGroupMemberRoleMap.value[currentUserID.value] ||
      ""
  ).trim();
}

function getConversationPeerUserID(conversation?: IConversationModel) {
  if (!conversation || conversation.type !== TUIChatEngine.TYPES.CONV_C2C) {
    return "";
  }

  const profileUserID = String(conversation.userProfile?.userID || "").trim();
  if (profileUserID) {
    return profileUserID;
  }

  const conversationID = String(conversation.conversationID || "");
  const prefix = TUIChatEngine.TYPES.CONV_C2C || "C2C";
  return conversationID.startsWith(prefix)
    ? conversationID.slice(prefix.length)
    : "";
}

function clearPeerLastOnline() {
  peerLastOnlineAt.value = "";
  peerObservedOfflineAt = 0;
  peerLastOnlineRequestSeq += 1;
  peerPresenceFetching = false;
}

async function syncPeerLastOnline(conversation?: IConversationModel) {
  const requestSeq = ++peerLastOnlineRequestSeq;

  const userID = getConversationPeerUserID(conversation);
  if (!userID) {
    return;
  }

  try {
    peerPresenceFetching = true;
    const items = await getUserLastOnline([userID]);
    if (peerPresenceDisposed || requestSeq !== peerLastOnlineRequestSeq) {
      return;
    }
    peerLastOnlineAt.value = resolveLastOnlineAt(
      items.find((item) => item.im_user_id === userID), peerObservedOfflineAt
    );
  } catch (error) {
    console.warn("sync peer last online failed", error);
  } finally {
    if (requestSeq === peerLastOnlineRequestSeq) peerPresenceFetching = false;
  }
}

function syncOnlineStatusFromStore() {
  const onlineStatusEnabled = TUIStore.getData(
    StoreName.USER,
    "displayOnlineStatus"
  );
  if (typeof onlineStatusEnabled === "boolean") {
    displayOnlineStatus.value = onlineStatusEnabled;
  }

  const userStatusList = TUIStore.getData(StoreName.USER, "userStatusList");
  onUserStatusListUpdated(userStatusList);
}

function onDisplayOnlineStatusUpdated(status: boolean) {
  displayOnlineStatus.value = status;
  handlePeerOnlineStateChange();
}

function onUserStatusListUpdated(list: unknown) {
  if (!(list instanceof Map)) {
    userOnlineStatusMap.value = {};
    handlePeerOnlineStateChange();
    return;
  }
  const nextMap: IUserStatusMap = {};
  list.forEach((status: IUserStatus, userID: string) => {
    nextMap[userID] = status;
  });
  userOnlineStatusMap.value = nextMap;
  handlePeerOnlineStateChange();
}

function getPeerOnlineState(conversation?: IConversationModel) {
  if (!conversation || conversation.type !== TUIChatEngine.TYPES.CONV_C2C) {
    return null;
  }

  const peerUserID = getConversationPeerUserID(conversation);
  if (!peerUserID || !displayOnlineStatus.value) {
    return null;
  }

  return isUserOnlineStatus(userOnlineStatusMap.value[peerUserID]);
}

function resetPeerOnlineState(conversation?: IConversationModel) {
  peerOnlineState = getPeerOnlineState(conversation);
}

function handlePeerOnlineStateChange() {
  const nextPeerOnlineState = getPeerOnlineState(currentConversation.value);
  if (nextPeerOnlineState === null) {
    return;
  }

  if (nextPeerOnlineState === false && peerOnlineState !== false) {
    if (peerOnlineState === true) {
      peerObservedOfflineAt = Date.now();
      peerLastOnlineAt.value = new Date(peerObservedOfflineAt).toISOString();
    }
    void syncPeerLastOnline(currentConversation.value);
  } else if (nextPeerOnlineState === true && peerOnlineState !== true) {
    clearPeerLastOnline();
  }

  peerOnlineState = nextPeerOnlineState;
}

function getCurrentGroupScopedAttributes() {
  const activeGroupID = groupID.value || "";
  if (!activeGroupID) {
    return {};
  }
  return currentGroupAttributesGroupID.value === activeGroupID
    ? currentGroupAttributes.value
    : {};
}

function updateCurrentGroupMemberRoleMap(memberList: Array<Record<string, any>> = []) {
  if (!memberList.length) {
    return;
  }

  currentGroupMemberRoleMap.value = memberList.reduce(
    (result, member) => {
      const userID = String(member?.userID || "").trim();
      if (!userID) {
        return result;
      }
      const role = String(member?.role || "").trim();
      if (role) {
        result[userID] = role;
      }
      return result;
    },
    { ...currentGroupMemberRoleMap.value } as Record<string, string>
  );
}

function canOpenGroupMemberProfile(contactInfo: Record<string, any>) {
  if (!isGroup.value || !contactInfo?.userID) {
    return true;
  }

  const targetUserID = String(contactInfo.userID || "").trim();
  if (!targetUserID) {
    return false;
  }

  if (targetUserID === currentUserID.value) {
    return true;
  }

  if (isMemberProfileAccessEnabled(getCurrentGroupScopedAttributes())) {
    return true;
  }

  return isPrivilegedGroupRole(getCurrentUserGroupRole());
}

async function openAvatarContactInfo(contactInfo: Record<string, any>) {
  if (!canOpenGroupMemberProfile(contactInfo)) {
    Toast({
      message: TUITranslateService.t("TUIGroup.该群已关闭成员资料查看"),
      type: TOAST_TYPE.WARNING,
    });
    return;
  }

  TUIStore.update(StoreName.CUSTOM, "currentContactSearchingStatus", false);
  TUIStore.update(StoreName.CUSTOM, "currentContactListKey", "");
  TUIStore.update(StoreName.CUSTOM, "currentContactInfo", contactInfo);
  showAvatarContactInfo.value = true;
}

function openCurrentConversationContactInfo() {
  if (isGroup.value) {
    handleGroup();
    return;
  }

  const conversation = currentConversation.value;
  if (!conversation) {
    return;
  }

  const userProfile = conversation.userProfile || {};
  const conversationID = conversation.conversationID || "";
  const userID =
    userProfile.userID ||
    (conversationID.startsWith(TUIChatEngine.TYPES.CONV_C2C)
      ? conversationID.slice(TUIChatEngine.TYPES.CONV_C2C.length)
      : "");
  if (!userID) {
    return;
  }

  openAvatarContactInfo({
    ...userProfile,
    userID,
    nick: userProfile.nick || conversation.getShowName?.() || userID,
    avatar: userProfile.avatar || "",
    profile: {
      ...userProfile,
      nick: userProfile.nick || conversation.getShowName?.() || userID,
      avatar: userProfile.avatar || "",
    },
    remark: (conversation as any).remark || "",
    friendRemark: (conversation as any).friendRemark || (conversation as any).remark || "",
  });
}

function closeAvatarContactInfo() {
  showAvatarContactInfo.value = false;
  TUIStore.update(StoreName.CUSTOM, "currentContactInfo", {});
}

function handleAvatarContactSwitchConversation() {
  closeAvatarContactInfo();
}

function toggleMultipleSelectMode(visible?: boolean) {
  isMultipleSelectMode.value =
    visible === undefined ? !isMultipleSelectMode.value : visible;
}

function mergeForwardMessage() {
  messageListRef.value?.mergeForwardMessage();
}

function oneByOneForwardMessage() {
  messageListRef.value?.oneByOneForwardMessage();
}

function onCurrentGroupUpdate(group: Record<string, any>) {
  syncMuteStateWithGroup(group, groupID.value);
}

function onCurrentGroupMemberListUpdate(memberList: Array<Record<string, any>>) {
  updateCurrentGroupMemberRoleMap(memberList);
  syncMuteStateWithMemberList(memberList, groupID.value);
}

function onMessageListUpdate(messageList: IMessageModel[] = []) {
  syncMuteStateWithMessageList(messageList, {
    isGroup: isGroup.value,
    groupID: groupID.value,
    conversationID: currentConversationID.value,
  });
}

function onCurrentConversationUpdate(conversation: IConversationModel) {
  currentConversation.value = conversation;
  if (conversation?.operationType > 0) {
    headerExtensionList.value = [];
    isNotInGroup.value = true;
    notInGroupReason.value = conversation?.operationType;
  } else {
    isNotInGroup.value = false;
    notInGroupReason.value = undefined;
  }
  // return when currentConversation is null
  if (!conversation) {
    closeAvatarContactInfo();
    groupID.value = undefined;
    clearPeerLastOnline();
    peerOnlineState = null;
    currentGroupAttributes.value = {};
    currentGroupAttributesGroupID.value = "";
    currentGroupMemberRoleMap.value = {};
    resetMuteState();
    return;
  }

  // Ignore repeated store updates for the same conversation.
  if (currentConversationID.value === conversation?.conversationID) {
    return;
  }

  closeAvatarContactInfo();

  let conversationType = TUIChatEngine.TYPES.CONV_C2C;
  const conversationID = conversation.conversationID;
  const groupState = handleConversationChange(conversation);
  isGroup.value = groupState.isGroup;
  groupID.value = groupState.groupID;
  if (groupState.isGroup) {
    clearPeerLastOnline();
    peerOnlineState = null;
    const nextGroupID = groupState.groupID || "";
    conversationType = TUIChatEngine.TYPES.CONV_GROUP;
    currentGroupAttributes.value = {};
    currentGroupAttributesGroupID.value = nextGroupID;
    TUIStore.update(StoreName.GRP, "currentGroupID", nextGroupID);
    TUIStore.update(StoreName.GRP, "currentGroupAttributes", {});
    currentGroupMemberRoleMap.value = {};
    if (conversation.groupProfile?.ownerID) {
      currentGroupMemberRoleMap.value[conversation.groupProfile.ownerID] =
        TUIChatEngine.TYPES.GRP_MBR_ROLE_OWNER;
    }
    if (conversation.groupProfile?.selfInfo?.userID && conversation.groupProfile?.selfInfo?.role) {
      currentGroupMemberRoleMap.value[conversation.groupProfile.selfInfo.userID] =
        conversation.groupProfile.selfInfo.role;
    }
    preloadGroupConversationContext(nextGroupID, currentUserID.value);
    void loadCurrentGroupAttributes(nextGroupID);
    void loadCurrentGroupPinnedMessage(true, nextGroupID);
  } else {
    currentGroupAttributes.value = {};
    currentGroupAttributesGroupID.value = "";
    TUIStore.update(StoreName.GRP, "currentGroupID", "");
    TUIStore.update(StoreName.GRP, "currentGroupAttributes", {});
    currentGroupMemberRoleMap.value = {};
    clearPeerLastOnline();
    resetPeerOnlineState(conversation);
    if (peerOnlineState === false) {
      void syncPeerLastOnline(conversation);
    }
    void loadCurrentGroupPinnedMessage(false);
  }

  headerExtensionList.value = [];
  isMultipleSelectMode.value = false;
  // Initialize chatType
  TUIChatConfig.setChatType(conversationType);
  // While converstaion change success, notify callkit and roomkit、or other components.
  TUICore.notifyEvent(
    TUIConstants.TUIChat.EVENT.CHAT_STATE_CHANGED,
    TUIConstants.TUIChat.EVENT_SUB_KEY.CHAT_OPENED,
    { groupID: groupID.value }
  );
  // The TUICustomerServicePlugin plugin determines if the current conversation is a customer service conversation, then sets chatType and activates the conversation.
  TUICore.callService({
    serviceName: TUIConstants.TUICustomerServicePlugin.SERVICE.NAME,
    method:
      TUIConstants.TUICustomerServicePlugin.SERVICE.METHOD.ACTIVE_CONVERSATION,
    params: { conversationID: conversationID },
  });
  // When open chat in room, close main chat ui and reset theme.
  if (TUIChatConfig.getChatType() === TUIConstants.TUIChat.TYPE.ROOM) {
    if (
      TUIChatConfig.getFeatureConfig(
        TUIConstants.TUIChat.FEATURE.InputVoice
      ) === true
    ) {
      TUIChatConfig.setTheme("light");
      currentConversationID.value = "";
      return;
    }
  }
  // Get chat header extensions
  if (TUIChatConfig.getChatType() === TUIConstants.TUIChat.TYPE.GROUP) {
    headerExtensionList.value = TUICore.getExtensionList(
      TUIConstants.TUIChat.EXTENSION.CHAT_HEADER.EXT_ID
    );
  }
  TUIStore.update(StoreName.CUSTOM, "activeConversation", conversationID);
  currentConversationID.value = conversationID;
}

async function loadCurrentGroupAttributes(nextGroupID?: string) {
  const requestedGroupID = nextGroupID || "";
  if (!requestedGroupID) {
    currentGroupAttributes.value = {};
    currentGroupAttributesGroupID.value = "";
    TUIStore.update(StoreName.GRP, "currentGroupID", "");
    TUIStore.update(StoreName.GRP, "currentGroupAttributes", {});
    return;
  }

  try {
    const res = await TUIGroupService.getGroupAttributes({
      groupID: requestedGroupID,
      keyList: [
        GROUP_MEMBER_PROFILE_ACCESS_ATTRIBUTE,
        GROUP_VOICE_CALL_ATTRIBUTE,
        GROUP_VIDEO_CALL_ATTRIBUTE,
      ],
    });
    if (requestedGroupID !== groupID.value) {
      return;
    }
    const nextAttributes = res?.data?.groupAttributes || {};
    currentGroupAttributes.value = nextAttributes;
    currentGroupAttributesGroupID.value = requestedGroupID;
    TUIStore.update(StoreName.GRP, "currentGroupID", requestedGroupID);
    TUIStore.update(StoreName.GRP, "currentGroupAttributes", nextAttributes);
  } catch (error) {
    if (requestedGroupID !== groupID.value) {
      return;
    }
    currentGroupAttributes.value = {};
    currentGroupAttributesGroupID.value = requestedGroupID;
    TUIStore.update(StoreName.GRP, "currentGroupID", requestedGroupID);
    TUIStore.update(StoreName.GRP, "currentGroupAttributes", {});
  }
}
</script>

<style scoped lang="scss" src="./style/index.scss"></style>
<style scoped lang="scss">
.tui-chat-contact-popup {
  position: absolute;
  inset: 0;
  z-index: 2000;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  box-sizing: border-box;
  background: rgba(15, 23, 42, 0.18);

  &-panel {
    position: relative;
    width: min(520px, 100%);
    height: min(640px, 100%);
    background: #f7f8fa;
    border-radius: 14px;
    overflow: hidden;
    box-shadow: 0 18px 48px rgba(15, 23, 42, 0.18);
  }

  &-close {
    position: absolute;
    top: 14px;
    right: 14px;
    z-index: 2;
    width: 28px;
    height: 28px;
    border: none;
    border-radius: 999px;
    background: rgba(15, 23, 42, 0.08);
    color: #475569;
    cursor: pointer;
    font-size: 20px;
    line-height: 1;
  }
}

.tui-chat-contact-popup-mobile {
  padding: 0;

  .tui-chat-contact-popup-panel {
    width: 100%;
    height: 100%;
    border-radius: 0;
  }
}

.tui-chat-group-pin {
  display: flex;
  align-items: center;
  flex-direction: row;
  padding: 8px 16px;
  border-bottom: 1px solid #eef2f7;
  background: #ffffff;

  &-main {
    min-width: 0;
    flex: 1;
    display: flex;
    align-items: center;
    flex-direction: row;
    gap: 10px;
    cursor: pointer;
  }

  &-accent {
    width: 3px;
    min-width: 3px;
    height: 14px;
    border-radius: 999px;
    background: #e5e7eb;
  }

  &-content {
    min-width: 0;
    flex: 1;
    width: 100%;
    display: flex;
    flex-direction: row;
    align-items: center;
  }

  &-label {
    margin-right: 8px;
    color: #94a3b8;
    font-size: 11px;
    font-weight: 600;
    line-height: 16px;
    flex-shrink: 0;
  }

  &-text {
    display: inline-flex;
    align-items: center;
    flex-direction: row;
    gap: 0;
    min-width: 0;
    flex: 1;
    overflow: hidden;
    color: #475569;
    font-size: 12px;
    line-height: 16px;
    white-space: nowrap;
    text-overflow: ellipsis;
  }
}

:deep(.group-pin-target-blink) {
  animation: groupPinBlink 1.8s ease;
}

@keyframes groupPinBlink {
  0% {
    background-color: rgba(249, 115, 22, 0.18);
  }
  100% {
    background-color: transparent;
  }
}
</style>
