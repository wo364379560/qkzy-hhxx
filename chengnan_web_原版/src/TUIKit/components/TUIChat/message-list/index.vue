<template>
  <div class="tui-chat" :class="[isH5 ? 'tui-chat-h5' : '']">
    <!-- <JoinGroupCard v-if="isH5" /> -->
    <div id="tui-chat-main" class="tui-chat-main" @click="closeChatPop">
      <!-- Safe Tips -->
      <div v-if="isOfficial" class="tui-chat-safe-tips">
        <span>
          {{
            TUITranslateService.t(
              "TUIChat.【安全提示】本 APP 仅用于体验腾讯云即时通信 IM 产品功能，不可用于业务洽谈与拓展。请勿轻信汇款、中奖等涉及钱款的信息，勿轻易拨打陌生电话，谨防上当受骗。"
            )
          }}
        </span>
        <a @click="openComplaintLink(Link.complaint)">
          {{ TUITranslateService.t("TUIChat.点此投诉") }}
        </a>
      </div>
      <MessageGroupApplication
        v-if="isGroup"
        :key="props.groupID"
        :groupID="props.groupID"
      />
      <p v-if="adminLoadError" role="status">{{ adminLoadError }}</p>
      <!-- Message List -->
      <ul
        id="messageScrollList"
        ref="messageListRef"
        class="tui-message-list"
        :class="{ 'tui-message-list-virtual': isPC }"
        @click="onMessageListBackgroundClick"
      >
        <p
          v-if="isRenderWindowLimited"
          class="message-more"
          @click="showMoreRenderedMessages"
        >
          {{ TUITranslateService.t("TUIChat.查看更多") }}
        </p>
        <p
          v-else-if="isShowLoadMore"
          class="message-more"
          @click="getHistoryMessageList"
        >
          {{ TUITranslateService.t("TUIChat.查看更多") }}
        </p>
        <li v-if="isPC" ref="virtualTopSpacer" aria-hidden="true"
          class="message-virtual-spacer" :style="{ height: virtualRange.top + 'px' }" />
        <li
          v-for="(item, index) in renderedMessageList"
          :id="'tui-' + item.ID"
          :data-message-id="item.ID"
          :key="item.ID"
          ref="messageElementListRef"
          class="message-li"
        >
          <MessageTimestamp
            :currTime="item.time"
            :prevTime="getPreviousMessageTime(index)"
          />
          <div class="message-item">
            <MessageTip
              v-if="
                item.type === TYPES.MSG_GRP_TIP ||
                isCreateGroupCustomMessage(item) ||
                isGroupMemberInviteCustomMessage(item) ||
                isGroupCollabCustomMessage(item)
              "
              :content="getTipMessageContent(item)"
              :blinkMessageIDList="blinkMessageIDList"
              @blinkMessage="blinkMessage"
            />
            <MessageRevoked
              v-else-if="item.isRevoked"
              :isEdit="item.type === TYPES.MSG_TEXT"
              :messageItem="shallowCopyMessage(item)"
              @messageEdit="handleEdit(item)"
            />
            <MessagePlugin
              v-else-if="isPluginMessage(item)"
              :message="deepCopy(item)"
              :blinkMessageIDList="blinkMessageIDList"
              @resendMessage="resendMessage"
              @handleToggleMessageItem="handleToggleMessageItem"
              @handleH5LongPress="handleH5LongPress"
            />
            <div
              v-else
              :class="{
                'message-event-bind-div': true,
              }"
              @longpress="handleToggleMessageItem($event, item, true)"
              @click.prevent.right="handleToggleMessageItemForPC($event, item)"
              @touchstart="handleH5LongPress($event, item, 'touchstart')"
              @touchend="handleH5LongPress($event, item, 'touchend')"
              @mouseover="handleH5LongPress($event, item, 'touchend')"
            >
              <MessageBubble
                :content="item.getMessageContent()"
                :showSenderName="showGroupMemberNickname"
                :isAudioPlayed="Boolean(audioPlayedMapping[item.ID])"
                :blinkMessageIDList="blinkMessageIDList"
                :isMultipleSelectMode="isMultipleSelectMode"
                :messageItem="shallowCopyMessage(item)"
                :multipleSelectedMessageIDList="multipleSelectedMessageIDList"
                @blinkMessage="blinkMessage"
                @openContactInfo="openContactInfo"
                @resendMessage="resendMessage(item)"
                @changeSelectMessageIDList="changeSelectMessageIDList"
                @setReadReceiptPanelVisible="setReadReceiptPanelVisible"
              >
                <template #messageElement>
                  <MessageText
                    v-if="item.type === TYPES.MSG_TEXT"
                    :content="item.getMessageContent()"
                    :messageItem="item"
                    :enableURLHighlight="true"
                  />
                  <ProgressMessage
                    v-else-if="item.type === TYPES.MSG_IMAGE"
                    :content="item.getMessageContent()"
                    :messageItem="item"
                  >
                    <MessageImage
                      :content="item.getMessageContent()"
                      :messageItem="item"
                      @previewImage="handleImagePreview"
                    />
                  </ProgressMessage>
                  <ProgressMessage
                    v-else-if="item.type === TYPES.MSG_VIDEO"
                    :content="item.getMessageContent()"
                    :messageItem="item"
                  >
                    <MessageVideo
                      :content="getVideoContent(item)"
                      :messageItem="item"
                    />
                  </ProgressMessage>
                  <MessageAudio
                    v-else-if="item.type === TYPES.MSG_AUDIO"
                    :content="item.getMessageContent()"
                    :messageItem="item"
                    @setAudioPlayed="setAudioPlayed"
                  />
                  <ProgressMessage
                    v-else-if="item.type === TYPES.MSG_FILE"
                    :content="item.getMessageContent()"
                    :messageItem="item"
                  >
                    <MessageFile
                      :content="getFileContent(item)"
                      :messageItem="item"
                    />
                  </ProgressMessage>
                  <MessageRecord
                    v-else-if="item.type === TYPES.MSG_MERGER"
                    :renderData="item.payload"
                    :messageItem="item"
                  />
                  <MessageFace
                    v-else-if="item.type === TYPES.MSG_FACE"
                    :content="item.getMessageContent()"
                  />
                  <MessageLocation
                    v-else-if="item.type === TYPES.MSG_LOCATION"
                    :content="item.getMessageContent()"
                  />
                  <MessageStreamMarkdown
                    v-else-if="AiRobotManager.isRobotMessage(item)"
                    :payloadData="item.payload.data"
                    :message="item"
                    @onStreaming="scrollStreamMessageToBottom"
                  />
                  <MessageCustom
                    v-else-if="item.type === TYPES.MSG_CUSTOM"
                    :content="item.getMessageContent()"
                    :messageItem="item"
                    @openContactInfo="openContactInfo"
                  />
                </template>
                <template #TUIEmojiPlugin>
                  <TUIEmojiPlugin
                    v-if="isShowEmojiPlugin && item.reactionList.length > 0"
                    type="reaction-detail"
                    :emojiConfig="emojiConfig"
                    :message="shallowCopyMessage(item)"
                  />
                </template>
                <template #messageExtra> </template>
              </MessageBubble>
            </div>
            <!-- message tool -->
            <MessageTool
              v-if="item.ID === toggleID"
              :class="{
                'message-tool': true,
                'message-tool-out': item.flow === 'out',
                'message-tool-in': item.flow === 'in',
                'message-tool-bottom': isTopMessageDom,
              }"
              :messageItem="item"
              :isMultipleSelectMode="isMultipleSelectMode"
              :canModerateIncomingMessage="canModerateIncomingMessage"
              :groupModerationContext="getGroupMessageModerationContext(item)"
              @ready="onMessageToolReady"
              @toggleMultipleSelectMode="
                () => emits('toggleMultipleSelectMode')
              "
              @moderateGroupMember="handleModerateGroupMember"
              @groupMessagePinChanged="() => emits('groupMessagePinChanged')"
            >
              <template #TUIEmojiPlugin>
                <TUIEmojiPlugin
                  v-if="isShowEmojiPlugin"
                  :message="item"
                  :emojiConfig="emojiConfig"
                />
              </template>
            </MessageTool>
          </div>
        </li>
        <li v-if="isPC" aria-hidden="true" class="message-virtual-spacer"
          :style="{ height: virtualRange.bottom + 'px' }" />
        <li
          v-for="task in largeAttachmentUploadTasks"
          :key="task.id"
          class="message-li large-attachment-upload-li"
        >
          <LargeAttachmentUploadCard
            :task="task"
            @cancel="cancelLargeAttachmentUploadTask"
            @retry="retryLargeAttachmentUploadTask"
            @close="removeLargeAttachmentUploadTask"
          />
        </li>
      </ul>
      <ScrollButton
        ref="scrollButtonInstanceRef"
        @scrollToLatestMessage="scrollToLatestMessage"
      />
      <button
        v-if="isAtJumpVisible"
        class="tui-at-jump-chip"
        :disabled="atJumpBusy"
        @click.stop="handleAtJumpClick"
      >
        <span class="tui-at-jump-chip-icon">⌃</span>
        <span>{{ atJumpLabel }}</span>
      </button>
      <Dialog
        v-if="reSendDialogShow"
        class="resend-dialog"
        :show="reSendDialogShow"
        :isH5="!isPC"
        :center="true"
        :isHeaderShow="isPC"
        @submit="resendMessageConfirm()"
        @update:show="(e) => (reSendDialogShow = e)"
      >
        <p class="delDialog-title">
          {{ TUITranslateService.t("TUIChat.确认重发该消息？") }}
        </p>
      </Dialog>
      <ImagePreviewer
        v-if="showImagePreview && currentImagePreview"
        :currentImage="currentImagePreview"
        :imageList="imageMessageList"
        @close="onImagePreviewerClose"
      />
      <ReadReceiptPanel
        v-if="isShowReadUserStatusPanel"
        :message="Object.assign({}, readStatusMessage)"
        @setReadReceiptPanelVisible="setReadReceiptPanelVisible"
      />
    </div>
  </div>
</template>

<script lang="ts" setup>
import {
  ref,
  nextTick,
  computed,
  onMounted,
  onUnmounted,
  watch,
  defineAsyncComponent,
} from "../../../adapter-vue";
import TUIChatEngine, {
  IMessageModel,
  IGroupModel,
  TUIStore,
  StoreName,
  TUITranslateService,
  TUIChatService,
  TUIGroupService,
} from "@tencentcloud/chat-uikit-engine-lite";
import TUICore, { TUIConstants } from "@tencentcloud/tui-core-lite";
import {
  outsideClick,
  getBoundingClientRect,
  getScrollInfo,
} from "@tencentcloud/universal-api";
import { TUIEmojiPlugin } from "@tencentcloud/tui-emoji-plugin";
// import { JoinGroupCard } from '@trtc/calls-uikit-vue';
import Link from "./link";
import MessageText from "./message-elements/message-text.vue";
import MessageImage from "./message-elements/message-image.vue";
import MessageAudio from "./message-elements/message-audio.vue";
import MessageRecord from "./message-elements/message-record/index.vue";
import MessageFile from "./message-elements/message-file.vue";
import MessageFace from "./message-elements/message-face.vue";
import MessageCustom from "./message-elements/message-custom.vue";
import MessageTip from "./message-elements/message-tip.vue";
import MessageBubble from "./message-elements/message-bubble.vue";
import MessageLocation from "./message-elements/message-location.vue";
import MessageTimestamp from "./message-elements/message-timestamp.vue";
import MessageVideo from "./message-elements/message-video.vue";
import MessageRevoked from "./message-tool/message-revoked.vue";
import ScrollButton from "./scroll-button/index.vue";
import LargeAttachmentUploadCard from "./large-attachment-upload-card.vue";
import {
  getCallMessageData,
  isPluginMessage,
} from "../../../plugins/plugin-components/index";
import ProgressMessage from "../../common/ProgressMessage/index.vue";
import { Toast, TOAST_TYPE } from "../../common/Toast/index";
import { withBusy } from "../../common/BusyOverlay/index";
import { normalizeExternalUrl } from "../../../utils/security";
import { emojiConfig } from "../emoji-config";
import { isPC, isH5 } from "../../../utils/env";
import chatStorage from "../utils/chatStorage";
import { throttle } from "../../../utils/lodash";
import {
  isEnabledMessageReadReceiptGlobal,
  shallowCopyMessage,
  isCreateGroupCustomMessage,
  isGroupMemberInviteCustomMessage,
  getGroupMemberInviteCustomContent,
  isGroupCollabCustomMessage,
  getGroupCollabCustomContent,
  deepCopy,
} from "../utils/utils";
import { getGroupConversationContext } from "../utils/groupConversationContext";
import {
  getConversationMessageSnapshot,
  setConversationMessageSnapshot,
} from "../utils/conversationMessageSnapshot";
import AiRobotManager from "../aiRobotManager";
import { dedupeCallSummaries } from "../utils/dedupeCallSummaries";
import { useVirtualMessages } from "../composables/useVirtualMessages";
import {
  getGroupTipOperatorID,
  isMuteRelatedGroupTip,
  isModeratorManagedGroupTip,
  isModeratorRole,
  shouldSilentlyHideRevokedMessage,
} from "@/features/group/policies";
import {
  IGroupMember,
  IGroupMessageModerationContext,
  IVideoMessageContent,
  IFileMessageContent,
} from "../../../interface";
import {
  GROUP_MEMBER_NICKNAME_VISIBILITY_EVENT,
  isGroupMemberNicknameEnabled,
} from "@/features/group/member-nickname";
import {
  cancelLargeAttachmentUploadTask,
  getLargeAttachmentUploadTasks,
  removeLargeAttachmentUploadTask,
  retryLargeAttachmentUploadTask,
} from "../utils/largeAttachmentUploadTasks";
import {
  getGroupAdmins,
  muteGroupMember,
  removeGroupMember,
  unmuteGroupMember,
} from "@/api";
import { isDoMattxCallSignalMessage } from "@/features/call/doMattxCallProtocol";

const MessageGroupApplication = defineAsyncComponent(
  () => import("./message-group-application/index.vue"),
);
const MessageTool = defineAsyncComponent(
  () => import("./message-tool/index.vue"),
);
const MessagePlugin = defineAsyncComponent(
  () => import("../../../plugins/plugin-components/message-plugin.vue"),
);
const MessageStreamMarkdown = defineAsyncComponent(
  () => import("./message-elements/message-stream-markdown/index.vue"),
);
const ReadReceiptPanel = defineAsyncComponent(
  () => import("./read-receipt-panel/index.vue"),
);
const Dialog = defineAsyncComponent(
  () => import("../../common/Dialog/index.vue"),
);
const ImagePreviewer = defineAsyncComponent(
  () => import("../../common/ImagePreviewer/index.vue"),
);

interface ScrollConfig {
  scrollToMessage?: IMessageModel;
  scrollToBottom?: boolean;
  scrollToOffset?: {
    top?: number;
    bottom?: number;
  };
}

interface AtJumpItem {
  seq?: string;
  atAll: boolean;
  msgID?: string;
}

interface IProps {
  isGroup: boolean;
  groupID: string;
  isNotInGroup: boolean;
  isMultipleSelectMode: boolean;
}

interface IEmits {
  (key: "closeInputToolBar"): void;
  (key: "toggleMultipleSelectMode"): void;
  (key: "handleEditor", message: IMessageModel, type: string): void;
  (key: "openContactInfo", contactInfo: Record<string, any>): void;
  (key: "groupMessagePinChanged"): void;
}

const emits = defineEmits<IEmits>();
const props = withDefaults(defineProps<IProps>(), {
  isGroup: false,
  groupID: "",
  isNotInGroup: false,
  isMultipleSelectMode: false,
});

let groupType: string | undefined;
let observer: IntersectionObserver | null = null;
let observerRoot: HTMLElement | null = null;
const observedReceiptElementIDSet = new Set<string>();
const receiptMessageMap = new Map<
  string,
  {
    msgDom: HTMLElement;
    msgModel: IMessageModel;
  }
>();
const sentReceiptMessageIDSet = new Set<string>();
const isOfficial = TUIStore.getData(StoreName.APP, "isOfficial");
const enabledEmojiPlugin = TUIStore.getData(
  StoreName.APP,
  "enabledEmojiPlugin"
);

const messageListRef = ref<HTMLElement>();
// The messageList displayed on the screen, not including messages where isDeleted is true
const messageList = ref<IMessageModel[]>([]);
// All messageList, including messages where isDeleted is true
const allMessageList = ref<IMessageModel[]>([]);
const multipleSelectedMessageIDList = ref<string[]>([]);
const isCompleted = ref(false);
const currentConversationID = ref("");
const currentGroupProfile = ref<IGroupModel>();
const groupMemberRoleMap = ref<Record<string, string>>({});
const groupMemberInfoMap = ref<Record<string, IGroupMember>>({});
const isModeratingGroupMember = ref(false);
const mergedAdminUserIDSet = ref<Set<string>>(new Set());
const showGroupMemberNickname = ref(false);
const currentLastMessage = ref<IMessageModel | undefined>();
const nextReqMessageID = ref();
const toggleID = ref("");
let messageMenuMouseButton = 2;
const TYPES = ref(TUIChatEngine.TYPES);
const isLongpressing = ref(false);
const messageTarget = ref<IMessageModel | undefined>();
const messageElementListRef = ref<HTMLElement[] | null>();
const blinkMessageIDList = ref<string[]>([]);
const scrollButtonInstanceRef = ref<InstanceType<typeof ScrollButton>>();
const isShowReadUserStatusPanel = ref<boolean>(false);
const readStatusMessage = ref<IMessageModel | undefined>();
const beforeHistoryGetScrollHeight = ref<number>(0);
const isTopMessageDom = ref<boolean>(false);
const audioPlayedMapping = ref<Record<string, boolean>>({});
const isHistoryMessageLoading = ref(false);
const pendingRoleRequestUserIDSet = new Set<string>();
const H5_INITIAL_RENDER_MESSAGE_COUNT = 180;
const H5_RENDER_MESSAGE_INCREMENT = 80;
const renderMessageLimit = ref(H5_INITIAL_RENDER_MESSAGE_COUNT);
let conversationRevision = 0;
let componentDisposed = false;
const virtualMessages = useVirtualMessages(messageList, messageListRef, messageElementListRef, isPC);
const virtualRange = virtualMessages.range;
const virtualTopSpacer = virtualMessages.topSpacer;
const GROUP_AT_JUMP_STORE_KEY = "groupAtJumpMap";
const DISMISSED_GROUP_AT_JUMP_STORE_KEY = "dismissedGroupAtJumpMap";
const atJumpList = ref<AtJumpItem[]>([]);
const atJumpBusy = ref(false);
const capturedAtJumpConversationID = ref("");
const dismissedAtJumpList = ref<AtJumpItem[]>([]);

// image preview
const showImagePreview = ref(false);
const currentImagePreview = ref<IMessageModel | undefined>();
const imageMessageList = computed(() =>
  messageList?.value?.filter((item: IMessageModel) => {
    return (
      !item.isRevoked &&
      !item.hasRiskContent &&
      item.type === TYPES.value.MSG_IMAGE
    );
  })
);

function getVideoContent(message: IMessageModel) {
  return message.getMessageContent() as unknown as IVideoMessageContent;
}

function getFileContent(message: IMessageModel) {
  return message.getMessageContent() as unknown as IFileMessageContent;
}

function getTipMessageContent(message: IMessageModel) {
  if (isGroupMemberInviteCustomMessage(message)) {
    return getGroupMemberInviteCustomContent(message);
  }
  if (isGroupCollabCustomMessage(message)) {
    return getGroupCollabCustomContent(message);
  }
  const payload = message?.payload || {};
  if (message.type === TUIChatEngine.TYPES.MSG_GRP_TIP) {
    const memberNames = resolveGroupTipMemberNames(payload);
    const operatorName = resolveGroupTipOperatorName(payload);
    if (payload.operationType === TUIChatEngine.TYPES.GRP_TIP_MBR_KICKED_OUT) {
      return {
        text: memberNames.length
          ? `${operatorName} 移除了 ${memberNames.join("、")}`
          : `${operatorName} 移除了群成员`,
      };
    }
    if (
      payload.operationType === TUIChatEngine.TYPES.GRP_TIP_MBR_JOIN &&
      payload.groupJoinType === 2
    ) {
      return {
        text: memberNames.length
          ? `${operatorName} 邀请 ${memberNames.join("、")} 加入群聊`
          : `${operatorName} 邀请成员加入群聊`,
      };
    }
  }
  return message.getMessageContent();
}

function resolveGroupTipOperatorName(payload: any) {
  const operatorID = String(payload?.operatorID || "").trim();
  const operator = groupMemberInfoMap.value[operatorID] as any;
  return String(
    operator?.nameCard || operator?.nick || operatorID || "成员"
  ).trim() || "成员";
}

function resolveGroupTipMemberNames(payload: any) {
  const names = (Array.isArray(payload?.memberList) ? payload.memberList : [])
    .map((member: any) =>
      String(member?.nameCard || member?.nick || member?.userID || "").trim()
    )
    .filter(Boolean);
  if (!names.length && Array.isArray(payload?.userIDList)) {
    names.push(
      ...payload.userIDList
        .map((userID: string) => {
          const member = groupMemberInfoMap.value[userID] as any;
          return String(
            member?.nameCard || member?.nick || userID || ""
          ).trim();
        })
        .filter(Boolean)
    );
  }
  return names;
}

// resend message dialog
const reSendDialogShow = ref(false);
const resendMessageData = ref();

const isShowEmojiPlugin = computed(() => {
  const msgPopMenuExtensionList = TUICore.getExtensionList(
    TUIConstants.TUIChat.EXTENSION.MSG_POP_MENU.EXT_ID,
    {
      enabledEmojiPlugin,
    }
  );
  return msgPopMenuExtensionList.some((item) => {
    return item.text === "TUIEmojiPlugin";
  });
});

onMounted(() => {
  // Retrieve the information about whether the audio has been played from localStorage
  audioPlayedMapping.value =
    chatStorage.getChatStorage("audioPlayedMapping") || {};

  TUIStore.watch(StoreName.CHAT, {
    messageList: onMessageListUpdated,
    messageSource: onMessageSourceUpdated,
    isCompleted: isCompletedUpdated,
  });

  TUIStore.watch(StoreName.CONV, {
    currentConversationID: onCurrentConversationIDUpdated,
  });

  TUIStore.watch(StoreName.GRP, {
    currentGroup: onCurrentGroupUpdated,
    currentGroupMemberList: onCurrentGroupMemberListUpdated,
  });

  TUIStore.watch(StoreName.CUSTOM, {
    isShowMessagePopMenu: isShowMessagePopMenuUpdated,
  });
});

onMounted(() => {
  messageListRef.value?.addEventListener("scroll", handelScrollListScroll);
  messageListRef.value?.addEventListener("tui-message-jump", onVirtualMessageJump as EventListener);
  if (typeof window !== "undefined") {
    window.addEventListener(
      GROUP_MEMBER_NICKNAME_VISIBILITY_EVENT,
      onGroupMemberNicknameVisibilityUpdated as EventListener
    );
  }
});

onUnmounted(() => {
  componentDisposed = true;
  conversationRevision += 1;
  TUIStore.unwatch(StoreName.CHAT, {
    messageList: onMessageListUpdated,
    messageSource: onMessageSourceUpdated,
    isCompleted: isCompletedUpdated,
  });

  TUIStore.unwatch(StoreName.CONV, {
    currentConversationID: onCurrentConversationIDUpdated,
  });

  TUIStore.unwatch(StoreName.GRP, {
    currentGroup: onCurrentGroupUpdated,
    currentGroupMemberList: onCurrentGroupMemberListUpdated,
  });

  TUIStore.unwatch(StoreName.CUSTOM, {
    isShowMessagePopMenu: isShowMessagePopMenuUpdated,
  });

  messageListRef.value?.removeEventListener("scroll", handelScrollListScroll);
  messageListRef.value?.removeEventListener("tui-message-jump", onVirtualMessageJump as EventListener);
  handelScrollListScroll.cancel?.();
  if (typeof window !== "undefined") {
    window.removeEventListener(
      GROUP_MEMBER_NICKNAME_VISIBILITY_EVENT,
      onGroupMemberNicknameVisibilityUpdated as EventListener
    );
  }

  if (Object.keys(audioPlayedMapping.value).length > 0) {
    // Synchronize storage about whether the audio has been played when the component is unmounted
    chatStorage.setChatStorage("audioPlayedMapping", audioPlayedMapping.value);
  }

  sentReceiptMessageIDSet.clear();
  resetReadReceiptObserver();
});

const isShowLoadMore = computed(() => {
  const list = allMessageList.value || [];
  return list.length > 0 && !isCompleted.value;
});

const renderedMessageStartIndex = computed(() => {
  if (isPC) return virtualRange.value.start;
  if (!isH5) {
    return 0;
  }
  return Math.max(messageList.value.length - renderMessageLimit.value, 0);
});

const isRenderWindowLimited = computed(() => {
  return isH5 && renderedMessageStartIndex.value > 0;
});

const renderedMessageList = computed(() => {
  if (isPC) return messageList.value.slice(virtualRange.value.start, virtualRange.value.end);
  if (!isH5) {
    return messageList.value;
  }
  return messageList.value.slice(renderedMessageStartIndex.value);
});

const largeAttachmentUploadTasks = computed(() =>
  getLargeAttachmentUploadTasks(currentConversationID.value)
);

const adminLoadError = ref('');
const currentUserID = computed<string>(() => {
  return TUIStore.getData(StoreName.USER, "userProfile")?.userID || "";
});

const isAtJumpVisible = computed(() => {
  return props.isGroup && atJumpList.value.length > 0;
});

const atJumpLabel = computed(() => {
  const next = atJumpList.value[0];
  if (!next) {
    return "";
  }
  return next.atAll ? "@所有人" : "有人@我";
});

const canModerateIncomingMessage = computed<boolean>(() => {
  if (!props.isGroup || !currentGroupProfile.value) {
    return false;
  }
  if (currentGroupProfile.value.type === TUIChatEngine.TYPES.GRP_AVCHATROOM) {
    return false;
  }
  return isSelfGroupModerator();
});

watch(
  () => props.groupID,
  async (groupID) => {
    adminLoadError.value = "";
    if (!props.isGroup || !groupID) {
      currentGroupProfile.value = undefined;
      groupMemberRoleMap.value = {};
      groupMemberInfoMap.value = {};
      mergedAdminUserIDSet.value = new Set();
      showGroupMemberNickname.value = false;
      return;
    }
    showGroupMemberNickname.value = isGroupMemberNicknameEnabled(
      groupID,
      currentUserID.value
    );
    await syncCurrentGroupContext(groupID);
    captureAtJumpsFromConversation(currentConversationID.value);
  },
  {
    immediate: true,
  }
);

watch(
  () =>
    largeAttachmentUploadTasks.value
      .map((task) => `${task.id}:${task.status}`)
      .join("|"),
  async (taskSignature, previousTaskSignature) => {
    if (!taskSignature || taskSignature === previousTaskSignature) {
      return;
    }
    await nextTick();
    await scrollToPosition({ scrollToBottom: true }).catch(() => undefined);
  }
);

async function onMessageListUpdated(list: IMessageModel[] = []) {
  const revision = conversationRevision;
  const activeID = TUIStore.getData(StoreName.CONV, "currentConversationID");
  if (list?.some(message => message.conversationID !== activeID)) return;
  const oldLastMessage = currentLastMessage.value;
  allMessageList.value = list;
  void ensureModeratorOperatorRoles(list);
  void ensureInviteTipUserProfiles(list);
  const hasEmojiReaction = rebuildVisibleMessageList(list);
  enrichAtJumpsWithLoadedMessages();

  if (!messageList.value.length) {
    currentLastMessage.value = undefined;
    return;
  }
  const newLastMessage = messageList.value[messageList.value.length - 1];
  if (
    scrollButtonInstanceRef.value?.isScrollButtonVisible &&
    newLastMessage?.flow === "in" &&
    getMessageRenderSignature(oldLastMessage) !==
      getMessageRenderSignature(newLastMessage)
  ) {
    enqueueIncomingAtJump(newLastMessage);
  }
  if (messageTarget.value) {
    if (
      messageList.value?.findIndex(
        (message: IMessageModel) => message?.ID === messageTarget.value?.ID
      ) >= 0
    ) {
      const tempMessage = messageTarget.value;
      messageTarget.value = undefined;
      ensureMessageRendered(tempMessage.ID);
      await nextTick();
      await scrollToPosition({ scrollToMessage: tempMessage });
      await blinkMessage(tempMessage?.ID);
    }
  } else if (beforeHistoryGetScrollHeight.value) {
    await scrollToPosition({
      scrollToOffset: { bottom: beforeHistoryGetScrollHeight.value },
    });
    beforeHistoryGetScrollHeight.value = 0;
  } else if (
    scrollButtonInstanceRef.value?.isScrollButtonVisible &&
    newLastMessage?.flow === "in"
  ) {
    // Keep the user's reading position, but still refresh the last-message
    // marker and snapshot cache so the next conversation switch does not
    // restore stale messages.
  } else if (
    newLastMessage?.ID &&
    getMessageRenderSignature(oldLastMessage) !==
      getMessageRenderSignature(newLastMessage)
  ) {
    await scrollToPosition({ scrollToBottom: true });
  } else if (hasEmojiReaction && isCurrentListInBottomPosition()) {
    await scrollToPosition({ scrollToBottom: true });
  }
  if (componentDisposed || revision !== conversationRevision) return;
  currentLastMessage.value = Object.assign({}, newLastMessage);
  if (currentConversationID.value && allMessageList.value.length) {
    setConversationMessageSnapshot(currentConversationID.value, {
      allMessages: allMessageList.value,
      visibleMessages: messageList.value,
      lastMessage: currentLastMessage.value,
      isCompleted: isCompleted.value,
    });
  }
  if (isEnabledMessageReadReceiptGlobal()) {
    nextTick(() => bindIntersectionObserver());
  }
}

function getMessageRenderSignature(message?: IMessageModel) {
  if (!message?.ID) {
    return "";
  }

  return [
    message.ID,
    message.time || 0,
    message.status || "",
    message.isRevoked ? "1" : "0",
    message.reactionList?.length || 0,
  ].join(":");
}

function getMessageIdentity(message?: any) {
  return String(
    message?.ID ||
      message?.msgID ||
      message?.messageID ||
      message?.clientSequence ||
      message?.sequence ||
      message?.seq ||
      ""
  ).trim();
}

function getMessageTime(message?: any) {
  return Number(
    message?.time || message?.lastTime || message?.lastTimeStamp || 0
  );
}

function isMessageSnapshotFresh(
  conversationID: string,
  snapshot: NonNullable<ReturnType<typeof getConversationMessageSnapshot>>
) {
  const conversation = TUIStore.getConversationModel(conversationID) as any;
  const conversationLastMessage = conversation?.lastMessage;
  if (!conversationLastMessage) {
    return true;
  }

  const snapshotLastMessage =
    snapshot.allMessages?.[snapshot.allMessages.length - 1] ||
    snapshot.lastMessage;
  if (!snapshotLastMessage) {
    return false;
  }

  const conversationLastIdentity = getMessageIdentity(conversationLastMessage);
  const snapshotLastIdentity = getMessageIdentity(snapshotLastMessage);
  if (conversationLastIdentity && snapshotLastIdentity) {
    return conversationLastIdentity === snapshotLastIdentity;
  }

  const conversationLastTime = getMessageTime(conversationLastMessage);
  const snapshotLastTime = getMessageTime(snapshotLastMessage);
  if (conversationLastTime && snapshotLastTime) {
    return snapshotLastTime >= conversationLastTime;
  }

  return true;
}

function getMessageSenderDisplayName(message: IMessageModel) {
  return (
    message?.nameCard ||
    message?.nick ||
    message?.from ||
    message?.conversationID ||
    ""
  );
}

function rebuildVisibleMessageList(
  list: IMessageModel[] = allMessageList.value || []
) {
  let hasEmojiReaction = false;
  let visibleList = list.filter((message) => {
    if (message.reactionList?.length && !message.isDeleted) {
      hasEmojiReaction = true;
    }
    return (
      !message.isDeleted
      && !shouldHideModeratorManagedMessage(message)
      && !isDoMattxCallSignalMessage(message)
    );
  });
  visibleList = dedupeNearbyCallSummaryMessages(visibleList);

  if (AiRobotManager.isRobot(currentConversationID.value)) {
    visibleList = AiRobotManager.handleMessageList(
      visibleList,
      currentConversationID.value
    );
  }

  messageList.value = visibleList;
  return hasEmojiReaction;
}

function dedupeNearbyCallSummaryMessages(list: IMessageModel[]) {
  return dedupeCallSummaries(list, message => {
    const call = getCallMessageData(message);
    if (!call) return undefined;
    const duration = Number(call.call_end || 0);
    const action = Number(call.actionType || 0);
    const caller = String(call.caller || call.inviter || call.call_user_id || "").trim()
      || (action === 2 && duration <= 0 ? String(message.from || "").trim() : "");
    return {
      key: `${Number(call.call_type || (call.cmd === "videoCall" ? 2 : 1))}:${duration > 0 ? "duration" : `action:${action}`}`,
      duration,
      caller,
    };
  });
}

function getStoredAtJumpMap(): Record<string, AtJumpItem[]> {
  const value = TUIStore.getData(StoreName.CUSTOM, GROUP_AT_JUMP_STORE_KEY);
  return value && typeof value === "object" ? value : {};
}

function setStoredAtJumpMap(map: Record<string, AtJumpItem[]>) {
  TUIStore.update(StoreName.CUSTOM, GROUP_AT_JUMP_STORE_KEY, map);
}

function getDismissedAtJumpMap(): Record<string, AtJumpItem[]> {
  const value = TUIStore.getData(
    StoreName.CUSTOM,
    DISMISSED_GROUP_AT_JUMP_STORE_KEY
  );
  return value && typeof value === "object" ? value : {};
}

function setDismissedAtJumpMap(map: Record<string, AtJumpItem[]>) {
  TUIStore.update(StoreName.CUSTOM, DISMISSED_GROUP_AT_JUMP_STORE_KEY, map);
}

function syncDismissedAtJumpsFromStore(conversationID: string) {
  const dismissedMap = getDismissedAtJumpMap();
  dismissedAtJumpList.value = Array.isArray(dismissedMap[conversationID])
    ? dismissedMap[conversationID]
    : [];
}

function getAtJumpSeq(item: AtJumpItem) {
  return String(item?.seq || "").trim();
}

function getAtJumpMsgID(item: AtJumpItem) {
  return String(item?.msgID || "").trim();
}

function getAtJumpKey(item: AtJumpItem) {
  const seq = getAtJumpSeq(item);
  if (seq) {
    return `seq:${seq}`;
  }
  const msgID = getAtJumpMsgID(item);
  return msgID ? `id:${msgID}` : "";
}

function getMessageSeq(message?: IMessageModel) {
  const raw =
    (message as any)?.sequence ||
    (message as any)?.seq ||
    (message as any)?.messageSequence ||
    (message as any)?.payload?.sequence ||
    "";
  return String(raw || "").trim();
}

function isSameAtJumpSeq(a?: string, b?: string) {
  const left = String(a || "").trim();
  const right = String(b || "").trim();
  if (!left || !right) {
    return false;
  }
  if (left === right) {
    return true;
  }
  const leftNumber = Number(left);
  const rightNumber = Number(right);
  return (
    Number.isFinite(leftNumber) &&
    Number.isFinite(rightNumber) &&
    leftNumber === rightNumber
  );
}

function isSameAtJumpTarget(a?: AtJumpItem, b?: AtJumpItem) {
  if (!a || !b) {
    return false;
  }
  const aMsgID = getAtJumpMsgID(a);
  const bMsgID = getAtJumpMsgID(b);
  if (aMsgID && bMsgID && aMsgID === bMsgID) {
    return true;
  }
  return isSameAtJumpSeq(a.seq, b.seq);
}

function getAtUserList(message?: IMessageModel): string[] {
  const raw =
    (message as any)?.atUserList ||
    (message as any)?.groupAtUserList ||
    (message as any)?.payload?.atUserList ||
    [];
  return Array.isArray(raw) ? raw.map((item) => String(item || "")) : [];
}

function isAtAllUserID(userID: string) {
  return (
    userID === TUIChatEngine.TYPES.MSG_AT_ALL ||
    userID === "@TIM#SYSTEM" ||
    userID === "c_api"
  );
}

function normalizeAtInfoTypeList(info: any) {
  const raw =
    info?.atType ??
    info?.groupAtType ??
    info?.GroupAtType ??
    info?.atTypeArray ??
    info?.groupAtTypeArray ??
    info?.AtTypeArray ??
    info?.AtFlagList;
  const source = Array.isArray(raw) ? raw : [raw];
  return source
    .map((item) => Number(item))
    .filter((item) => Number.isFinite(item));
}

function getAtInfoSeq(info: any) {
  return String(
    info?.seq ||
      info?.sequence ||
      info?.messageSequence ||
      info?.atMsgSeq ||
      info?.AtMsgSeq ||
      info?.MsgSeq ||
      ""
  ).trim();
}

function getAtInfoMsgID(info: any) {
  return String(info?.msgID || info?.messageID || info?.ID || "").trim();
}

function getAtJumpFromAtInfo(info: any): AtJumpItem | undefined {
  const seq = getAtInfoSeq(info);
  const msgID = getAtInfoMsgID(info);
  if (!seq && !msgID) {
    return undefined;
  }
  const atTypes = normalizeAtInfoTypeList(info);
  if (atTypes.length && !atTypes.some((type) => [1, 2, 3].includes(type))) {
    return undefined;
  }
  return {
    seq,
    msgID,
    atAll: atTypes.includes(2) || atTypes.includes(3),
  };
}

function getAtJumpFromMessage(message?: IMessageModel): AtJumpItem | undefined {
  if (!props.isGroup || !message || message.flow === "out") {
    return undefined;
  }
  const seq = getMessageSeq(message);
  const msgID = message.ID;
  if (!seq && !msgID) {
    return undefined;
  }
  const atUserList = getAtUserList(message);
  if (!atUserList.length) {
    return undefined;
  }
  const atAll = atUserList.some(isAtAllUserID);
  const atMe = Boolean(
    currentUserID.value && atUserList.includes(currentUserID.value)
  );
  if (!atAll && !atMe) {
    return undefined;
  }
  return {
    seq,
    atAll: atAll && !atMe,
    msgID,
  };
}

function sortAtJumps(list: AtJumpItem[]) {
  return [...list].sort((a, b) => {
    const bSeq = Number(getAtJumpSeq(b));
    const aSeq = Number(getAtJumpSeq(a));
    if (Number.isFinite(aSeq) && Number.isFinite(bSeq) && aSeq !== bSeq) {
      return bSeq - aSeq;
    }
    return getAtJumpKey(b).localeCompare(getAtJumpKey(a));
  });
}

function mergeAtJumpList(items: AtJumpItem[]) {
  const merged = new Map<string, AtJumpItem>();
  [...atJumpList.value, ...items].forEach((item) => {
    if (
      dismissedAtJumpList.value.some((dismissed) =>
        isSameAtJumpTarget(dismissed, item)
      )
    ) {
      return;
    }
    const key = getAtJumpKey(item);
    if (!key) {
      return;
    }
    const previous = Array.from(merged.values()).find((candidate) =>
      isSameAtJumpTarget(candidate, item)
    );
    const next = {
      ...(previous || {}),
      ...item,
      atAll: previous ? previous.atAll || item.atAll : item.atAll,
      msgID: previous?.msgID || item.msgID,
      seq: previous?.seq || item.seq,
    };
    merged.set(previous ? getAtJumpKey(previous) : key, next);
  });
  atJumpList.value = sortAtJumps(Array.from(merged.values()));
}

function captureAtJumpsFromConversation(conversationID: string) {
  if (!props.isGroup || !conversationID) {
    atJumpList.value = [];
    capturedAtJumpConversationID.value = "";
    return;
  }
  syncDismissedAtJumpsFromStore(conversationID);
  if (capturedAtJumpConversationID.value === conversationID) {
    return;
  }
  capturedAtJumpConversationID.value = conversationID;
  const storeMap = getStoredAtJumpMap();
  const stored = Array.isArray(storeMap[conversationID])
    ? storeMap[conversationID]
    : [];
  const conversation = TUIStore.getConversationModel(conversationID) as any;
  const live = Array.isArray(conversation?.groupAtInfoList)
    ? conversation.groupAtInfoList
        .map((info: any) => getAtJumpFromAtInfo(info))
        .filter(Boolean)
    : [];
  atJumpList.value = [];
  mergeAtJumpList([...(stored as AtJumpItem[]), ...(live as AtJumpItem[])]);
  enrichAtJumpsWithLoadedMessages();
}

function clearStoredAtJump(conversationID: string, target: AtJumpItem) {
  const storeMap = { ...getStoredAtJumpMap() };
  const stored = Array.isArray(storeMap[conversationID])
    ? storeMap[conversationID]
    : [];
  const next = stored.filter((item) => !isSameAtJumpTarget(item, target));
  if (next.length) {
    storeMap[conversationID] = next;
  } else {
    delete storeMap[conversationID];
  }
  setStoredAtJumpMap(storeMap);
}

function saveDismissedAtJump(conversationID: string, target: AtJumpItem) {
  if (!conversationID || !getAtJumpKey(target)) {
    return;
  }
  const dismissedMap = { ...getDismissedAtJumpMap() };
  const current = Array.isArray(dismissedMap[conversationID])
    ? dismissedMap[conversationID]
    : [];
  const next = [
    target,
    ...current.filter((item) => !isSameAtJumpTarget(item, target)),
  ].slice(0, 50);
  dismissedMap[conversationID] = next;
  dismissedAtJumpList.value = next;
  setDismissedAtJumpMap(dismissedMap);
}

function enrichAtJumpsWithLoadedMessages() {
  if (!props.isGroup || !atJumpList.value.length) {
    return;
  }
  let changed = false;
  atJumpList.value = atJumpList.value.map((item) => {
    if (item.msgID) {
      return item;
    }
    const message = messageList.value.find((candidate) =>
      isSameAtJumpSeq(getMessageSeq(candidate), item.seq)
    );
    if (!message?.ID) {
      return item;
    }
    changed = true;
    return {
      ...item,
      msgID: message.ID,
    };
  });
  if (changed) {
    atJumpList.value = sortAtJumps(atJumpList.value);
  }
}

function enqueueIncomingAtJump(message: IMessageModel) {
  const item = getAtJumpFromMessage(message);
  if (!item) {
    return;
  }
  if (
    dismissedAtJumpList.value.some((dismissed) =>
      isSameAtJumpTarget(dismissed, item)
    )
  ) {
    return;
  }
  atJumpList.value = atJumpList.value.filter(
    (candidate) => !isSameAtJumpTarget(candidate, item)
  );
  atJumpList.value = sortAtJumps([item, ...atJumpList.value]);
}

function findMessageByAtJump(target: AtJumpItem) {
  if (target.msgID) {
    const byID = messageList.value.find((message) => message.ID === target.msgID);
    if (byID) {
      return byID;
    }
  }
  return messageList.value.find((message) =>
    isSameAtJumpSeq(getMessageSeq(message), target.seq)
  );
}

async function scrollToMessageAndBlink(message: IMessageModel) {
  const revision = conversationRevision;
  ensureMessageRendered(message.ID);
  await nextTick();
  await scrollToPosition({ scrollToMessage: message });
  if (componentDisposed || revision !== conversationRevision) return;
  await blinkMessage(message.ID);
}

async function loadHistoryUntilAtJumpFound(target: AtJumpItem) {
  const revision = conversationRevision;
  let message = findMessageByAtJump(target);
  for (
    let round = 0;
    !message && !isCompleted.value && round < 10;
    round += 1
  ) {
    await getHistoryMessageList();
    if (componentDisposed || revision !== conversationRevision) return undefined;
    await nextTick();
    enrichAtJumpsWithLoadedMessages();
    message = findMessageByAtJump(target);
  }
  return message;
}

async function handleAtJumpClick() {
  if (atJumpBusy.value || !atJumpList.value.length) {
    return;
  }
  const revision = conversationRevision;
  const target = atJumpList.value[0];
  atJumpList.value = atJumpList.value.slice(1);
  atJumpBusy.value = true;
  try {
    const message = await loadHistoryUntilAtJumpFound(target);
    if (componentDisposed || revision !== conversationRevision) return;
    if (!message) {
      atJumpList.value = sortAtJumps([target, ...atJumpList.value]);
      Toast({
        message: "未找到该消息，请继续上滑加载历史",
        type: TOAST_TYPE.ERROR,
      });
      return;
    }
    await scrollToMessageAndBlink(message);
    if (componentDisposed || revision !== conversationRevision) return;
    if (currentConversationID.value) {
      saveDismissedAtJump(currentConversationID.value, {
        ...target,
        msgID: target.msgID || message.ID,
        seq: target.seq || getMessageSeq(message),
      });
      clearStoredAtJump(currentConversationID.value, target);
    }
  } finally {
    if (revision === conversationRevision) atJumpBusy.value = false;
  }
}

function onVirtualMessageJump(event: CustomEvent<{ messageID: string }>) {
  const message = messageList.value.find(item => item.ID === event.detail?.messageID);
  if (message) void scrollToMessageAndBlink(message);
}

watch(() => [virtualRange.value.start, virtualRange.value.end], () => {
  if (isPC && isEnabledMessageReadReceiptGlobal()) nextTick(() => bindIntersectionObserver());
});

function getPreviousMessageTime(renderedIndex: number) {
  const sourceIndex = renderedMessageStartIndex.value + renderedIndex;
  return sourceIndex > 0 ? messageList.value[sourceIndex - 1]?.time || 0 : 0;
}

function expandRenderLimit(increment = H5_RENDER_MESSAGE_INCREMENT) {
  if (!isH5) {
    return;
  }
  renderMessageLimit.value += increment;
}

async function showMoreRenderedMessages() {
  if (!isRenderWindowLimited.value) {
    return;
  }
  const previousScrollHeight = messageListRef.value?.scrollHeight || 0;
  expandRenderLimit();
  await nextTick();
  if (previousScrollHeight) {
    await scrollToPosition({
      scrollToOffset: { bottom: previousScrollHeight },
    }).catch(() => undefined);
  }
  if (isEnabledMessageReadReceiptGlobal()) {
    nextTick(() => bindIntersectionObserver());
  }
}

function ensureMessageRendered(messageID?: string) {
  if (isPC && messageID) {
    virtualMessages.scrollToMessage(messageID);
    return;
  }
  if (!isH5 || !messageID) {
    return;
  }
  const messageIndex = messageList.value.findIndex(
    (message) => message.ID === messageID
  );
  if (messageIndex < 0) {
    return;
  }
  const neededLimit = messageList.value.length - messageIndex;
  if (neededLimit > renderMessageLimit.value) {
    renderMessageLimit.value = neededLimit;
  }
}

function shouldHideModeratorManagedMessage(message: IMessageModel) {
  if (!props.isGroup) {
    return false;
  }

  if (message.isRevoked) {
    // 群主/管理员撤回群成员消息：整条不可见（对齐 chixin）
    return shouldSilentlyHideRevokedMessage(message, true);
  }

  if (!isModeratorManagedGroupTip(message)) {
    return false;
  }

  if (isMuteRelatedGroupTip(message)) {
    return true;
  }

  return isModeratorOperator(getGroupTipOperatorID(message));
}

function isModeratorOperator(userID: string) {
  if (!userID) {
    return false;
  }
  if (userID === currentUserID.value) {
    return isSelfGroupModerator();
  }
  if (userID === currentGroupProfile.value?.ownerID) {
    return true;
  }
  return isModeratorRole(groupMemberRoleMap.value[userID]) || isMergedAdmin(userID);
}

function isMergedAdmin(userID?: string) {
  return Boolean(userID && mergedAdminUserIDSet.value.has(userID));
}

function isSelfGroupModerator() {
  return (
    isModeratorRole(currentGroupProfile.value?.selfInfo?.role) ||
    isMergedAdmin(currentUserID.value)
  );
}

function updateGroupMemberRoleMap(
  memberList: Array<{ userID?: string; role?: string }> = []
) {
  if (!memberList.length) {
    return;
  }

  groupMemberRoleMap.value = memberList.reduce<Record<string, string>>(
    (result, member) => {
      if (member?.userID && member?.role) {
        result[member.userID] = member.role;
      }
      return result;
    },
    { ...groupMemberRoleMap.value }
  );
}

function updateGroupMemberInfoMap(memberList: IGroupMember[] = []) {
  if (!memberList.length) {
    return;
  }

  groupMemberInfoMap.value = memberList.reduce<Record<string, IGroupMember>>(
    (result, member) => {
      if (member?.userID) {
        result[member.userID] = {
          ...(result[member.userID] || {}),
          ...member,
        };
      }
      return result;
    },
    { ...groupMemberInfoMap.value }
  );
}

async function syncCurrentGroupContext(groupID: string) {
  const revision = conversationRevision;
  try {
    const context = await getGroupConversationContext(
      groupID,
      currentUserID.value,
    );
    if (componentDisposed || revision !== conversationRevision || groupID !== props.groupID) return;
    const group = context.group;
    const selfMember = context.selfMember;
    if (!group) {
      return;
    }
    currentGroupProfile.value = {
      ...group,
      selfInfo: {
        ...(group.selfInfo || {}),
        userID: group.selfInfo?.userID || currentUserID.value,
        role: group.selfInfo?.role || selfMember?.role,
      },
    };
    const selfInfo = currentGroupProfile.value.selfInfo;
    updateGroupMemberRoleMap([
      {
        userID: selfInfo?.userID,
        role: selfInfo?.role,
      },
      {
        userID: currentGroupProfile.value.ownerID,
        role: TUIChatEngine.TYPES.GRP_MBR_ROLE_OWNER,
      },
    ]);
    updateGroupMemberInfoMap([
      {
        userID: selfInfo?.userID,
        role: selfInfo?.role,
      },
      {
        userID: currentGroupProfile.value.ownerID,
        role: TUIChatEngine.TYPES.GRP_MBR_ROLE_OWNER,
      },
    ]);
    void getGroupAdmins(groupID).then((adminResult) => {
      if (componentDisposed || revision !== conversationRevision || groupID !== props.groupID) {
        return;
      }
      const adminIDs = new Set<string>();
      (adminResult?.admins || []).forEach((admin) => {
        const userID = admin?.user_id;
        if (!userID) {
          return;
        }
        adminIDs.add(userID);
      });
      mergedAdminUserIDSet.value = adminIDs;
      const adminMemberList = Array.from(adminIDs).map((userID) => ({
        userID,
        role: TUIChatEngine.TYPES.GRP_MBR_ROLE_ADMIN,
      }));
      updateGroupMemberRoleMap(adminMemberList);
      updateGroupMemberInfoMap(adminMemberList as IGroupMember[]);
      adminLoadError.value = "";
      rebuildVisibleMessageList();
    }).catch(() => {
      if (componentDisposed || revision !== conversationRevision) return;
      mergedAdminUserIDSet.value = new Set();
      adminLoadError.value = "管理员资料暂时加载失败，普通聊天仍可使用";
    });
  } catch (error) {
    console.warn("syncCurrentGroupContext failed:", error);
  }

  if (componentDisposed || revision !== conversationRevision) return;
  rebuildVisibleMessageList();
}

async function ensureModeratorOperatorRoles(list: IMessageModel[]) {
  const revision = conversationRevision;
  if (!props.isGroup || !props.groupID) {
    return;
  }

  const userIDList = Array.from(
    new Set(
      list
        .filter(
          (message) => message.isRevoked || isModeratorManagedGroupTip(message)
        )
        .map((message) => getGroupTipOperatorID(message))
        .filter(
          (userID) =>
            Boolean(userID) &&
            userID !== currentUserID.value &&
            userID !== currentGroupProfile.value?.ownerID &&
            !groupMemberRoleMap.value[userID] &&
            !pendingRoleRequestUserIDSet.has(userID)
        )
    )
  );

  if (!userIDList.length) {
    return;
  }

  userIDList.forEach((userID) => pendingRoleRequestUserIDSet.add(userID));
  try {
    const res = await TUIGroupService.getGroupMemberProfile({
      groupID: props.groupID,
      userIDList,
    });
    if (componentDisposed || revision !== conversationRevision) return;
    updateGroupMemberRoleMap(res?.data?.memberList || []);
    updateGroupMemberInfoMap(res?.data?.memberList || []);
    rebuildVisibleMessageList();
    if (currentConversationID.value && allMessageList.value.length) {
      setConversationMessageSnapshot(currentConversationID.value, {
        allMessages: allMessageList.value,
        visibleMessages: messageList.value,
        lastMessage: currentLastMessage.value,
        isCompleted: isCompleted.value,
      });
    }
  } catch (error) {
    console.warn("ensureModeratorOperatorRoles failed:", error);
  } finally {
    if (revision === conversationRevision) userIDList.forEach((userID) => pendingRoleRequestUserIDSet.delete(userID));
  }
}

async function ensureInviteTipUserProfiles(list: IMessageModel[]) {
  const revision = conversationRevision;
  if (!props.isGroup || !props.groupID) {
    return;
  }
  const userIDList = Array.from(
    new Set(
      list
        .filter(
          (message) =>
            message.type === TUIChatEngine.TYPES.MSG_GRP_TIP &&
            message.payload?.operationType ===
              TUIChatEngine.TYPES.GRP_TIP_MBR_JOIN &&
            message.payload?.groupJoinType === 2
        )
        .flatMap((message) => [
          message.payload?.operatorID,
          ...(message.payload?.userIDList || []),
        ])
        .map((userID) => String(userID || "").trim())
        .filter(
          (userID) =>
            Boolean(userID) &&
            !groupMemberInfoMap.value[userID] &&
            !pendingRoleRequestUserIDSet.has(userID)
        )
    )
  );
  if (!userIDList.length) {
    return;
  }

  userIDList.forEach((userID) => pendingRoleRequestUserIDSet.add(userID));
  try {
    const res = await TUIGroupService.getGroupMemberProfile({
      groupID: props.groupID,
      userIDList,
    });
    if (componentDisposed || revision !== conversationRevision) return;
    updateGroupMemberInfoMap(res?.data?.memberList || []);
    rebuildVisibleMessageList();
  } catch (error) {
    console.warn("ensureInviteTipUserProfiles failed:", error);
  } finally {
    if (revision === conversationRevision) userIDList.forEach((userID) => pendingRoleRequestUserIDSet.delete(userID));
  }
}

function isCurrentListInBottomPosition() {
  return (
    messageListRef.value &&
    typeof messageListRef.value.scrollTop === "number" &&
    typeof messageListRef.value.scrollHeight === "number" &&
    typeof messageListRef.value.clientHeight === "number" &&
    Math.ceil(
      messageListRef.value.scrollTop + messageListRef.value.clientHeight
    ) >= messageListRef.value.scrollHeight
  );
}

async function scrollStreamMessageToBottom() {
  await scrollToPosition({ scrollToBottom: true });
}

async function scrollToPosition(config: ScrollConfig = {}): Promise<void> {
  const revision = conversationRevision;
  if (isPC) {
    if (config.scrollToBottom) virtualMessages.scrollToBottom();
    else if (config.scrollToMessage) virtualMessages.scrollToMessage(config.scrollToMessage.ID);
    await nextTick();
  }
  return new Promise((resolve) => {
    requestAnimationFrame(() => {
      if (componentDisposed || revision !== conversationRevision || !messageListRef.value) {
        resolve();
        return;
      }
      const container = messageListRef.value;
      if (config.scrollToBottom) {
        container!.scrollTop = container!.scrollHeight;
      } else if (config.scrollToMessage) {
        const targetMessageDom = messageElementListRef.value?.find(
          (dom: HTMLElement) => dom?.id === `tui-${config.scrollToMessage?.ID}`
        );
        if (targetMessageDom?.scrollIntoView) {
          targetMessageDom.scrollIntoView({
            behavior: isPC ? "auto" : "smooth",
            block: "nearest",
          });
        }
      } else if (config.scrollToOffset) {
        if (config.scrollToOffset?.top) {
          container!.scrollTop = config.scrollToOffset.top;
        } else if (config.scrollToOffset?.bottom) {
          container!.scrollTop =
            container!.scrollHeight - config.scrollToOffset.bottom;
        }
      }
      resolve();
    });
  });
}

async function onMessageSourceUpdated(message: IMessageModel) {
  if (message && message.conversationID !== currentConversationID.value) return;
  // messageSource change has two cases
  // 1. messageSource change -> cache miss -> messageList change,
  // 2. messageSource change -> cache hit -> messageList not change
  // Only the second case needs to add scrollToTarget when listening here
  messageTarget.value = message;
  if (messageTarget.value) {
    if (
      messageList.value?.findIndex(
        (message: IMessageModel) => message?.ID === messageTarget.value?.ID
      ) >= 0
    ) {
      const tempMessage = messageTarget.value;
      messageTarget.value = undefined;
      ensureMessageRendered(tempMessage.ID);
      await nextTick();
      await scrollToPosition({ scrollToMessage: tempMessage });
      await blinkMessage(tempMessage?.ID);
    }
  }
}

function isCompletedUpdated(flag: boolean) {
  isCompleted.value = flag;
  if (currentConversationID.value && allMessageList.value.length) {
    setConversationMessageSnapshot(currentConversationID.value, {
      allMessages: allMessageList.value,
      visibleMessages: messageList.value,
      lastMessage: currentLastMessage.value,
      isCompleted: isCompleted.value,
    });
  }
}

function onCurrentGroupUpdated(group: IGroupModel) {
  if (!group || group.groupID !== props.groupID) {
    return;
  }

  currentGroupProfile.value = group;
  updateGroupMemberRoleMap([
    {
      userID: group.ownerID,
      role: TUIChatEngine.TYPES.GRP_MBR_ROLE_OWNER,
    },
    {
      userID: group.selfInfo?.userID,
      role: group.selfInfo?.role,
    },
  ]);
  updateGroupMemberInfoMap([
    {
      userID: group.ownerID,
      role: TUIChatEngine.TYPES.GRP_MBR_ROLE_OWNER,
    },
    {
      userID: group.selfInfo?.userID,
      role: group.selfInfo?.role,
    },
  ]);
  rebuildVisibleMessageList();
}

function onCurrentGroupMemberListUpdated(memberList: any[]) {
  updateGroupMemberRoleMap(memberList || []);
  updateGroupMemberInfoMap(memberList || []);
  rebuildVisibleMessageList();
}

function onGroupMemberNicknameVisibilityUpdated(event: Event) {
  const detail = (event as CustomEvent).detail || {};
  if (detail?.conversationID !== currentConversationID.value) {
    return;
  }
  if (typeof detail?.enabled === "boolean") {
    showGroupMemberNickname.value = detail.enabled;
  }
}

function isShowMessagePopMenuUpdated(isShow: boolean) {
  if (!isShow) {
    toggleID.value = "";
  }
}

const onCurrentConversationIDUpdated = (conversationID: string) => {
  conversationRevision += 1;
  isHistoryMessageLoading.value = false;
  beforeHistoryGetScrollHeight.value = 0;
  nextReqMessageID.value = undefined;
  messageTarget.value = undefined;
  toggleID.value = "";
  multipleSelectedMessageIDList.value = [];
  currentGroupProfile.value = undefined;
  mergedAdminUserIDSet.value = new Set();
  groupMemberRoleMap.value = {};
  groupMemberInfoMap.value = {};
  pendingRoleRequestUserIDSet.clear();
  virtualMessages.reset();
  currentConversationID.value = conversationID;
  atJumpList.value = [];
  atJumpBusy.value = false;
  capturedAtJumpConversationID.value = "";
  syncDismissedAtJumpsFromStore(conversationID);
  renderMessageLimit.value = H5_INITIAL_RENDER_MESSAGE_COUNT;
  resetReadReceiptObserver();
  sentReceiptMessageIDSet.clear();
  if (!currentConversationID.value) {
    messageList.value = [];
    allMessageList.value = [];
    currentLastMessage.value = undefined;
    isCompleted.value = false;
  } else {
    const cachedSnapshot = getConversationMessageSnapshot(conversationID);
    if (
      cachedSnapshot &&
      isMessageSnapshotFresh(conversationID, cachedSnapshot)
    ) {
      allMessageList.value = cachedSnapshot.allMessages;
      messageList.value = cachedSnapshot.visibleMessages;
      currentLastMessage.value = cachedSnapshot.lastMessage;
      isCompleted.value = cachedSnapshot.isCompleted;
      nextTick(() => {
        void scrollToPosition({ scrollToBottom: true }).catch(() => undefined);
      });
    } else {
      messageList.value = [];
      allMessageList.value = [];
      currentLastMessage.value = undefined;
      isCompleted.value = false;
    }
  }
  if (isEnabledMessageReadReceiptGlobal()) {
    const { groupProfile } =
      TUIStore.getConversationModel(conversationID) || {};
    groupType = groupProfile?.type;
  }
  captureAtJumpsFromConversation(conversationID);
  if (Object.keys(audioPlayedMapping.value).length > 0) {
    // Synchronize storage about whether the audio has been played when converstaion switched
    chatStorage.setChatStorage("audioPlayedMapping", audioPlayedMapping.value);
  }
};

async function ensureMessageTargetModerationContext(message: IMessageModel) {
  const revision = conversationRevision;
  const targetUserID = message?.from || "";
  if (
    !props.isGroup ||
    !props.groupID ||
    !targetUserID ||
    targetUserID === currentUserID.value ||
    pendingRoleRequestUserIDSet.has(targetUserID) ||
    groupMemberInfoMap.value[targetUserID]?.role
  ) {
    return;
  }

  pendingRoleRequestUserIDSet.add(targetUserID);
  try {
    const res = await TUIGroupService.getGroupMemberProfile({
      groupID: props.groupID,
      userIDList: [targetUserID],
    });
    if (componentDisposed || revision !== conversationRevision) return;
    updateGroupMemberRoleMap(res?.data?.memberList || []);
    updateGroupMemberInfoMap(res?.data?.memberList || []);
    rebuildVisibleMessageList();
  } finally {
    if (revision === conversationRevision) pendingRoleRequestUserIDSet.delete(targetUserID);
  }
}

function getGroupMessageModerationContext(
  message: IMessageModel
): IGroupMessageModerationContext | undefined {
  if (
    !props.isGroup ||
    !props.groupID ||
    !canModerateIncomingMessage.value ||
    !message ||
    message.flow !== "in" ||
    message.type === TYPES.value.MSG_GRP_TIP ||
    message.type === TYPES.value.MSG_GRP_SYS_NOTICE
  ) {
    return undefined;
  }

  const targetUserID = message.from || "";
  if (!targetUserID || targetUserID === currentUserID.value) {
    return undefined;
  }

  const selfRole = currentGroupProfile.value?.selfInfo?.role;
  const isOwner = selfRole === TUIChatEngine.TYPES.GRP_MBR_ROLE_OWNER;
  const isAdmin =
    selfRole === TUIChatEngine.TYPES.GRP_MBR_ROLE_ADMIN ||
    isMergedAdmin(currentUserID.value);
  const targetInfo = groupMemberInfoMap.value[targetUserID];
  const targetRole =
    targetUserID === currentGroupProfile.value?.ownerID
      ? TUIChatEngine.TYPES.GRP_MBR_ROLE_OWNER
      : targetInfo?.role || groupMemberRoleMap.value[targetUserID] || "";
  const isMuted = Number(targetInfo?.muteUntil || 0) * 1000 > Date.now();

  const canKick =
    currentGroupProfile.value?.type !== TUIChatEngine.TYPES.GRP_AVCHATROOM &&
    (isOwner
      ? targetRole !== TUIChatEngine.TYPES.GRP_MBR_ROLE_OWNER
      : isAdmin
        ? targetRole === TUIChatEngine.TYPES.GRP_MBR_ROLE_MEMBER
        : false);

  const canToggleMute =
    currentGroupProfile.value?.type !== TUIChatEngine.TYPES.GRP_WORK &&
    (isOwner
      ? targetRole !== TUIChatEngine.TYPES.GRP_MBR_ROLE_OWNER
      : isAdmin
        ? targetRole === TUIChatEngine.TYPES.GRP_MBR_ROLE_MEMBER
        : false);

  if (!canKick && !canToggleMute) {
    return undefined;
  }

  return {
    groupID: props.groupID,
    targetUserID,
    targetDisplayName: getMessageSenderDisplayName(message),
    targetRole,
    isMuted,
    canKick,
    canMute: canToggleMute && !isMuted,
    canUnmute: canToggleMute && isMuted,
  };
}

async function handleModerateGroupMember(
  action: "mute" | "unmute" | "kick",
  context: IGroupMessageModerationContext
) {
  if (!props.groupID || !context?.targetUserID || isModeratingGroupMember.value) {
    return;
  }

  const busyText =
    action === "kick" ? "正在移出成员..." : "正在更新禁言状态...";

  isModeratingGroupMember.value = true;
  try {
    await withBusy(busyText, async () => {
      if (action === "kick") {
        await removeGroupMember(props.groupID, context.targetUserID);
        delete groupMemberRoleMap.value[context.targetUserID];
        delete groupMemberInfoMap.value[context.targetUserID];
        Toast({
          message: TUITranslateService.t("TUIChat.已踢出群聊"),
          type: TOAST_TYPE.SUCCESS,
        });
        return;
      }

      if (action === "mute") {
        await muteGroupMember(props.groupID, context.targetUserID);
      } else {
        await unmuteGroupMember(props.groupID, context.targetUserID);
      }
      const muteTime = action === "mute" ? 60 * 60 * 24 * 45 : 0;
      updateGroupMemberInfoMap([
        {
          ...(groupMemberInfoMap.value[context.targetUserID] || {}),
          userID: context.targetUserID,
          role:
            context.targetRole || groupMemberRoleMap.value[context.targetUserID],
          muteUntil:
            action === "mute"
              ? String(Math.floor(Date.now() / 1000) + muteTime)
              : "0",
        },
      ]);
      Toast({
        message: TUITranslateService.t(
          action === "mute" ? "TUIGroup.禁言设置成功" : "TUIGroup.取消禁言成功"
        ),
        type: TOAST_TYPE.SUCCESS,
      });
    });
  } catch (error: any) {
    Toast({
      message: error?.message || TUITranslateService.t("TUIChat.操作失败"),
      type: TOAST_TYPE.ERROR,
    });
  } finally {
    isModeratingGroupMember.value = false;
  }
}

const getHistoryMessageList = async () => {
  if (isHistoryMessageLoading.value || isCompleted.value) {
    return;
  }

  const revision = conversationRevision;
  isHistoryMessageLoading.value = true;
  // After getting the historical messages, keep the scroll bar in the original position
  beforeHistoryGetScrollHeight.value = isPC ? 0 : (messageListRef.value?.scrollHeight || 0);
  expandRenderLimit();

  try {
    const res: any = await TUIChatService.getMessageList();
    if (componentDisposed || revision !== conversationRevision || res.stale) return;
    const { nextReqMessageID: ID } = res.data;
    nextReqMessageID.value = ID;
  } catch (error) {
    if (revision === conversationRevision) console.warn("get history message list failed:", error);
  } finally {
    if (revision === conversationRevision) isHistoryMessageLoading.value = false;
  }
};

const openComplaintLink = (type: any) => {
  const safeUrl = normalizeExternalUrl(type?.url);
  if (!safeUrl) {
    return;
  }
  window.open(safeUrl, "_blank", "noopener,noreferrer");
};

const handleImagePreview = (message: IMessageModel) => {
  if (
    showImagePreview.value ||
    currentImagePreview.value ||
    isLongpressing.value
  ) {
    return;
  }
  showImagePreview.value = true;
  currentImagePreview.value = message;
};

const onImagePreviewerClose = () => {
  showImagePreview.value = false;
  currentImagePreview.value = undefined;
};

// toggle message
const handleToggleMessageItem = (
  e: any,
  message: IMessageModel,
  isLongpress = false
) => {
  if (props.isMultipleSelectMode || props.isNotInGroup) {
    return;
  }
  if (isLongpress) {
    isLongpressing.value = true;
  }
  toggleID.value = message.ID;
  void ensureMessageTargetModerationContext(message);
  filterTopMessageDom(e.target);
};

const handleToggleMessageItemForPC = (
  e: MouseEvent,
  message: IMessageModel
) => {
  if (props.isMultipleSelectMode || props.isNotInGroup) {
    return;
  }
  if (isPC) {
    messageMenuMouseButton = e.button;
    toggleID.value = message.ID;
    void ensureMessageTargetModerationContext(message);
    nextTick(() => {
      if (componentDisposed || toggleID.value !== message.ID) return;
      filterTopMessageDom(e.target);
    });
  }
};

function onMessageToolReady({ messageID, element }: {
  messageID: string;
  element: HTMLElement;
}) {
  // An import may finish after the user closes the menu or switches messages.
  // Never register an empty node or a late node belonging to another message.
  if (!isPC || componentDisposed || toggleID.value !== messageID) return;
  const messageRow = messageElementListRef.value?.find(
    (dom: HTMLElement) => dom?.id === `tui-${messageID}`
  );
  if (!messageRow?.contains(element)) return;
  outsideClick.listen({
    domRefs: messageRow,
    ignoreDomRefs: element,
    handler: closeChatPop,
    button: messageMenuMouseButton,
  });
}

function filterTopMessageDom(toggleMessageElement: any) {
  const chatElement = document.getElementById("tui-chat-main");
  if (!chatElement || !toggleMessageElement) {
    isTopMessageDom.value = false;
    return;
  }
  const safeTop = 160;
  const messageElementRect = toggleMessageElement.getBoundingClientRect();
  const ChatElementRect = chatElement.getBoundingClientRect();
  isTopMessageDom.value =
    messageElementRect.top - ChatElementRect.top < safeTop ? true : false;
}

// h5 long press
let timer: ReturnType<typeof setTimeout> | undefined;
const handleH5LongPress = (e: any, message: IMessageModel, type: string) => {
  if (props.isMultipleSelectMode || props.isNotInGroup) {
    return;
  }
  if (!isH5) return;
  function longPressHandler() {
    clearTimeout(timer);
    handleToggleMessageItem(e, message);
  }
  function touchStartHandler() {
    timer = setTimeout(longPressHandler, 500);
  }
  function touchEndHandler() {
    clearTimeout(timer);
  }
  switch (type) {
    case "touchstart":
      touchStartHandler();
      break;
    case "touchend":
      touchEndHandler();
      setTimeout(() => {
        isLongpressing.value = false;
      }, 200);
      break;
  }
};

// re-edit message
const handleEdit = (message: IMessageModel) => {
  emits("handleEditor", message, "reedit");
};

const resendMessage = (message: IMessageModel) => {
  reSendDialogShow.value = true;
  resendMessageData.value = message;
};

const resendMessageConfirm = () => {
  reSendDialogShow.value = !reSendDialogShow.value;
  const messageModel = resendMessageData.value;
  messageModel.resendMessage();
};

function blinkMessage(messageID: string): Promise<void> {
  return new Promise((resolve) => {
    const index = blinkMessageIDList.value.indexOf(messageID);
    if (index >= 0) {
      resolve();
      return;
    }
    if (index < 0) {
      blinkMessageIDList.value.push(messageID);
      const timer = setTimeout(() => {
        blinkMessageIDList.value.splice(
          blinkMessageIDList.value.indexOf(messageID),
          1
        );
        clearTimeout(timer);
        resolve();
      }, 3000);
    }
  });
}

async function scrollToLatestMessage() {
  await scrollToPosition({ scrollToBottom: true });
}

const handelScrollListScroll = throttle(
  function (e: Event) {
    scrollButtonInstanceRef.value?.judgeScrollOverOneScreen(e);
  },
  150,
  { leading: true }
);

async function bindIntersectionObserver() {
  if (
    !renderedMessageList.value ||
    !messageListRef.value ||
    renderedMessageList.value.length === 0
  ) {
    return;
  }

  if (
    groupType === TYPES.value.GRP_AVCHATROOM ||
    groupType === TYPES.value.GRP_COMMUNITY
  ) {
    // AVCHATROOM and COMMUNITY chats do not monitor read receipts for messages.
    return;
  }

  const messageMap = new Map(
    renderedMessageList.value.map((message: IMessageModel) => [
      `tui-${message.ID}`,
      message,
    ])
  );

  if (!observer || observerRoot !== messageListRef.value) {
    resetReadReceiptObserver();
    observerRoot = messageListRef.value;
    observer = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          const { isIntersecting, target } = entry;
          if (!isIntersecting) {
            return;
          }
          const receiptMessage = receiptMessageMap.get(target.id);
          if (!receiptMessage) {
            return;
          }
          const { msgDom, msgModel } = receiptMessage;
          if (
            !msgModel.readReceiptInfo?.isPeerRead &&
            !sentReceiptMessageIDSet.has(msgModel.ID)
          ) {
            TUIChatService.sendMessageReadReceipt([msgModel]);
            sentReceiptMessageIDSet.add(msgModel.ID);
          }
          observer?.unobserve(msgDom);
          observedReceiptElementIDSet.delete(target.id);
          receiptMessageMap.delete(target.id);
        });
      },
      {
        root: messageListRef.value,
        threshold: 0.7,
      }
    );
  }

  const messageElementList = Array.isArray(messageElementListRef.value)
    ? messageElementListRef.value
    : [];
  const currentElementIDSet = new Set(
    messageElementList.map((messageElement) => messageElement.id)
  );
  Array.from(observedReceiptElementIDSet).forEach((elementID) => {
    if (currentElementIDSet.has(elementID)) {
      return;
    }
    const receiptMessage = receiptMessageMap.get(elementID);
    if (receiptMessage?.msgDom) {
      observer?.unobserve(receiptMessage.msgDom);
    }
    observedReceiptElementIDSet.delete(elementID);
    receiptMessageMap.delete(elementID);
  });
  messageElementList.forEach((messageElement) => {
    const matchingMessage = messageMap.get(messageElement.id);
    if (
      matchingMessage &&
      matchingMessage.needReadReceipt &&
      matchingMessage.flow === "in" &&
      !matchingMessage.readReceiptInfo?.isPeerRead &&
      !sentReceiptMessageIDSet.has(matchingMessage.ID) &&
      !observedReceiptElementIDSet.has(messageElement.id)
    ) {
      receiptMessageMap.set(messageElement.id, {
        msgDom: messageElement,
        msgModel: matchingMessage,
      });
      observedReceiptElementIDSet.add(messageElement.id);
      observer?.observe(messageElement);
    }
  });
}

function resetReadReceiptObserver() {
  observer?.disconnect();
  observer = null;
  observerRoot = null;
  observedReceiptElementIDSet.clear();
  receiptMessageMap.clear();
}

// eslint-disable-next-line @typescript-eslint/no-unused-vars
const isSignalingMessage = (message: IMessageModel) => {
  return (
    message?.type === TYPES.value.MSG_CUSTOM && message?.getSignalingInfo()
  );
};

function setReadReceiptPanelVisible(visible: boolean, message?: IMessageModel) {
  if (visible && props.isNotInGroup) {
    return;
  }
  if (!visible) {
    readStatusMessage.value = undefined;
  } else {
    readStatusMessage.value = message;
  }
  isShowReadUserStatusPanel.value = visible;
}

function closeChatPop() {
  toggleID.value = "";
}

function onMessageListBackgroundClick() {
  emits("closeInputToolBar");
}

watch(
  () => props.isMultipleSelectMode,
  (newValue) => {
    if (!newValue) {
      changeSelectMessageIDList({
        type: "clearAll",
        messageID: "",
      });
    }
  }
);

function changeSelectMessageIDList({
  type,
  messageID,
}: {
  type: "add" | "remove" | "clearAll";
  messageID: string;
}) {
  // TODO need to delete this
  if (type === "clearAll") {
    multipleSelectedMessageIDList.value = [];
  } else if (
    type === "add" &&
    !multipleSelectedMessageIDList.value.includes(messageID)
  ) {
    multipleSelectedMessageIDList.value.push(messageID);
  } else if (type === "remove") {
    multipleSelectedMessageIDList.value =
      multipleSelectedMessageIDList.value.filter((id) => id !== messageID);
  }
}

function mergeForwardMessage() {
  TUIStore.update(StoreName.CUSTOM, "multipleForwardMessageID", {
    isMergeForward: true,
    messageIDList: multipleSelectedMessageIDList.value,
  });
}

function oneByOneForwardMessage() {
  TUIStore.update(StoreName.CUSTOM, "multipleForwardMessageID", {
    isMergeForward: false,
    messageIDList: multipleSelectedMessageIDList.value,
  });
}

function setAudioPlayed(messageID: string) {
  audioPlayedMapping.value = {
    ...audioPlayedMapping.value,
    [messageID]: true,
  };
}

function openContactInfo(contactInfo: Record<string, any>) {
  emits("openContactInfo", contactInfo);
}

defineExpose({
  oneByOneForwardMessage,
  mergeForwardMessage,
  scrollToLatestMessage,
});
</script>

<style lang="scss" scoped src="./style/index.scss"></style>
<style>
.tui-message-list-virtual { overflow-anchor: none; }
.tui-message-list-virtual > .message-li,
.tui-message-list-virtual > .message-more,
.tui-message-list-virtual > .message-virtual-spacer { flex-shrink: 0; }
.message-virtual-spacer { margin: 0; padding: 0; border: 0; pointer-events: none; }
.row-reverse {
  flex-direction: row-reverse;
}
</style>
