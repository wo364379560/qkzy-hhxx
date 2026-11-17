<template>
  <div class="message-input-wrapper">
    <div
      :class="[
        'message-input-container',
        !isPC && 'message-input-container-h5',
      ]"
    >
      <MessageInputEditor
        ref="editor"
        :placeholder="props.placeholder"
        :isMuted="props.isMuted"
        :muteText="props.muteText"
        :enableInput="props.enableInput"
        :enableAt="props.enableAt"
        :enableTyping="props.enableTyping"
        :enableDragUpload="props.enableDragUpload"
        @sendMessage="sendMessage"
        @onTyping="onTyping"
        @onAt="onAt"
      />
      <MessageInputButton
        v-if="!props.isMuted && !isRobotStreaming"
        :disabled="isSubmitting"
        @sendMessage="sendMessage"
      />
      <MessageInputButton
        v-if="isRobotStreaming"
        className="break-btn"
        :hoverText="TUITranslateService.t('TUIChat.停止回答')"
        @sendMessage="sendBreakMessage"
      >
        <Icon :file="breakBtnSvg" class="icon" />
      </MessageInputButton>
      <MessageInputAt
        v-if="props.enableAt"
        ref="messageInputAtRef"
        @insertAt="insertAt"
        @onAtListOpen="onAtListOpen"
      />
    </div>
    <MessageInputQuote />
    <div
      v-if="visibleQueueJobs.length"
      class="message-outbox"
      aria-live="polite"
    >
      <div v-for="(job, index) in visibleQueueJobs" :key="index">
        <span :title="job.label">{{ job.label }}</span>
        <strong>{{ queueStatusText(job.status) }}</strong>
        <button
          v-if="job.status === 'failed'"
          type="button"
          @click="retryQueuedMessage(job)"
        >
          重试
        </button>
        <button
          v-if="job.status === 'failed' || job.status === 'unknown'"
          type="button"
          @click="dismissQueuedMessage(job)"
        >
          移除记录
        </button>
      </div>
      <small v-if="visibleQueueJobs.some((job) => job.status === 'unknown')"
        >消息结果待确认，请先核对聊天记录，勿重复发送。</small
      >
    </div>
    <Dialog
      v-if="showNotFriendDialog"
      class="not-friend-dialog"
      :show="showNotFriendDialog"
      :center="true"
      :isHeaderShow="isPC"
      @submit="goToAddFriend()"
      @update:show="(value) => (showNotFriendDialog = value)"
    >
      <div class="not-friend-dialog-content">
        <p class="not-friend-dialog-title">
          {{
            TUITranslateService.t("TUIChat.当前消息发送失败，可添加好友后重试")
          }}
        </p>
        <p class="not-friend-dialog-desc">
          {{
            TUITranslateService.t(
              "TUIChat.点击确定前往通讯录，向对方发送好友申请"
            )
          }}
        </p>
      </div>
    </Dialog>
  </div>
</template>

<script setup lang="ts">
import {
  TUIStore,
  StoreName,
  IConversationModel,
  TUITranslateService,
} from "@tencentcloud/chat-uikit-engine-lite";
import { computed, onMounted, onUnmounted, ref } from "../../../adapter-vue";
import MessageInputEditor from "./message-input-editor.vue";
import MessageInputAt from "./message-input-at/index.vue";
import MessageInputButton from "./message-input-button.vue";
import MessageInputQuote from "./message-input-quote/index.vue";
import { sendMessages, sendTyping } from "../utils/sendMessage";
import { captureMessageSendScope } from "../utils/messageSendScope";
import { transformTextWithEmojiNamesToKeys } from "../emoji-config";
import { isPC, isH5 } from "../../../utils/env";
import AiRobotManager from "../aiRobotManager";
import Icon from "../../common/Icon.vue";
import Dialog from "../../common/Dialog/index.vue";
import breakBtnSvg from "../../../assets/icon/break-btn.svg";
import { cloneDeep } from "lodash";
import { Toast, TOAST_TYPE } from "../../common/Toast/index";
import { captureTextQuote } from "../utils/textMessageQuote";
import {
  getSharedMessageQueue,
  type MessageQueueJob,
  type QueueStatus,
} from "../utils/messageQueue";

const props = defineProps({
  placeholder: {
    type: String,
    default: "this is placeholder",
  },
  isMuted: {
    type: Boolean,
    default: true,
  },
  muteText: {
    type: String,
    default: "",
  },
  enableInput: {
    type: Boolean,
    default: true,
  },
  enableAt: {
    type: Boolean,
    default: true,
  },
  enableDragUpload: {
    type: Boolean,
    default: true,
  },
  enableTyping: {
    type: Boolean,
    default: true,
  },
});

const emit = defineEmits(["sendMessage", "resetReplyOrReference", "onTyping"]);
const editor = ref<InstanceType<typeof MessageInputEditor>>();
const messageInputAtRef = ref<InstanceType<typeof MessageInputAt>>();
const currentConversation = ref<IConversationModel>();
const robotStreamingStatus = ref<Record<string, boolean>>({});
const showNotFriendDialog = ref(false);
const notFriendTarget = ref<{
  userID: string;
  nick?: string;
  avatar?: string;
} | null>(null);
const isSubmitting = ref(false);
let composerMounted = true;
const queueJobs = ref<MessageQueueJob[]>([]);
const messageQueue = getSharedMessageQueue();
const unsubscribeQueue = messageQueue.subscribe(() => {
  queueJobs.value = messageQueue.jobs;
});

const retryQueuedMessage = (job: MessageQueueJob) => {
  if (!messageQueue.retry(job))
    Toast({ message: "当前无法重试，请稍后再试", type: TOAST_TYPE.WARNING });
};
const dismissQueuedMessage = (job: MessageQueueJob) => {
  if (
    job.status === "unknown" &&
    !window.confirm(
      "发送结果尚未确认，请先核对聊天记录。移除记录不会撤销或重新发送消息，是否继续？"
    )
  )
    return;
  if (!messageQueue.dismiss(job))
    Toast({
      message: "消息仍在发送，请等待结果确认",
      type: TOAST_TYPE.WARNING,
    });
};
const visibleQueueJobs = computed(() =>
  queueJobs.value.filter((job) => job.isActive() && job.status !== "success")
);
const queueStatusText = (status?: QueueStatus) =>
  ({
    pending: "待发送",
    sending: "发送中",
    success: "已发送",
    failed: "发送失败",
    unknown: "结果待确认",
  }[status || "pending"]);

AiRobotManager.onSteamingStatusChange((option?: Record<string, boolean>) => {
  robotStreamingStatus.value = option || {};
});

const conversationWatchOptions = {
  currentConversation: (conversation: IConversationModel) => {
    currentConversation.value = conversation;
  },
};

onMounted(() => {
  currentConversation.value = TUIStore.getData(
    StoreName.CONV,
    "currentConversation"
  );
  TUIStore.watch(StoreName.CONV, conversationWatchOptions);
});

onUnmounted(() => {
  composerMounted = false;
  TUIStore.unwatch(StoreName.CONV, conversationWatchOptions);
  unsubscribeQueue();
});

const isRobotStreaming = computed(() => {
  if (currentConversation.value?.conversationID) {
    return (
      robotStreamingStatus.value[currentConversation.value?.conversationID] ||
      false
    );
  }
  return false;
});

const onTyping = (inputContentEmpty: boolean, inputBlur: boolean) => {
  sendTyping(inputContentEmpty, inputBlur);
};

const onAt = (show: boolean) => {
  messageInputAtRef.value?.toggleAtList(show);
};

const sendBreakMessage = () => {
  AiRobotManager.sendBreakConversation(
    currentConversation.value?.conversationID || ""
  );
};

const sendMessage = (type: string = "text") => {
  if (isSubmitting.value || (isRobotStreaming.value && type === "enter"))
    return;
  if (isRobotStreaming.value) {
    sendBreakMessage();
    return;
  }
  const conversation = currentConversation.value;
  if (
    !conversation ||
    !props.enableInput ||
    props.isMuted ||
    conversation.operationType > 0
  )
    return;
  const contentList = (editor.value?.getEditorContent() || [])
    .map((content: any) => ({
      ...content,
      payload: {
        ...content.payload,
        ...(content.type === "text"
          ? {
              text: transformTextWithEmojiNamesToKeys(
                content.payload?.text || ""
              ),
            }
          : {}),
      },
    }))
    .filter(
      (content: any) =>
        content.type !== "text" || String(content.payload.text).trim()
    );
  if (!contentList.length) return;
  if (!messageQueue.hasCapacity(contentList.length)) {
    Toast({
      message: "待发送消息较多，请稍后再发送，输入内容已保留",
      type: TOAST_TYPE.WARNING,
    });
    return;
  }
  const isActive = captureMessageSendScope();
  if (!isActive()) return;
  const target = cloneDeep(conversation);
  const quote = captureTextQuote(conversation.conversationID);
  const firstText = contentList.find((content: any) => content.type === "text");
  isSubmitting.value = true;
  try {
    const admitted = messageQueue.enqueue(
      contentList.map((content: any) => ({
        lane:
          content.type === "text" ? ("text" as const) : ("attachment" as const),
        label:
          content.type === "text"
            ? String(content.payload.text)
            : `[${content.type === "image" ? "图片" : "附件"}]`,
        isActive,
        run: async () => {
          const canSend = () => {
            if (!isActive()) return false;
            const latest =
              TUIStore.getConversationModel(target.conversationID) || target;
            return (
              !(latest.operationType > 0) &&
              (!composerMounted ||
                currentConversation.value?.conversationID !==
                  target.conversationID ||
                (props.enableInput && !props.isMuted))
            );
          };
          const result = await sendMessages(
            [content],
            target,
            isActive,
            content === firstText ? quote : "",
            canSend
          );
          if (isActive() && result.handledErrorType === "not_friend") {
            notFriendTarget.value = result.notFriendTarget || null;
            showNotFriendDialog.value = true;
          }
          return result.hasError ? result.delivery || "unknown" : "success";
        },
      }))
    );
    if (!admitted) return;
    editor.value?.resetEditor();
    if (firstText) TUIStore.update(StoreName.CHAT, "quoteMessage", {});
    emit("sendMessage");
  } finally {
    isSubmitting.value = false;
  }
};

const goToAddFriend = () => {
  const target = notFriendTarget.value;
  showNotFriendDialog.value = false;
  if (!target?.userID) {
    return;
  }

  TUIStore.update(StoreName.CUSTOM, "currentContactSearchingStatus", false);
  TUIStore.update(StoreName.CUSTOM, "currentContactListKey", "");
  TUIStore.update(StoreName.CUSTOM, "currentContactInfo", {
    userID: target.userID,
    nick: target.nick || target.userID,
    avatar: target.avatar || "",
  });
  TUIStore.update(StoreName.CUSTOM, "currentHomeNavBar", "relation");
};

const insertEmoji = (emoji: any) => {
  editor.value?.addEmoji(emoji);
};

const onAtListOpen = () => {
  if (isH5) {
    editor.value?.blur();
  }
};

const insertAt = (atInfo: any) => {
  editor?.value?.insertAt && editor?.value?.insertAt(atInfo);
};

const reEdit = (content: any) => {
  editor.value?.resetEditor();
  editor.value?.setEditorContent(content);
};

defineExpose({
  insertEmoji,
  reEdit,
});
</script>

<style lang="scss">
.message-outbox {
  max-height: 120px;
  overflow-y: auto;
  padding: 6px 12px;
  font-size: 12px;
  color: #667085;
}
.message-outbox > div {
  display: flex;
  gap: 12px;
  justify-content: space-between;
}
.message-outbox span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.message-outbox strong {
  white-space: nowrap;
  font-weight: 400;
}
.message-outbox small {
  color: #965600;
}
@import "../../../assets/styles/common";

.message-input-wrapper {
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
}

.message-input-container {
  box-sizing: border-box;
  flex: 1;
  display: flex;
  flex-direction: column;
  border: none;
  overflow: hidden;
}

.message-input-container-h5 {
  display: flex;
  flex-flow: row nowrap;
  align-items: flex-end;
}
.break-btn {
  background: none !important;
}

.not-friend-dialog-content {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-width: 280px;
}

.not-friend-dialog-title {
  margin: 0;
  font-size: 15px;
  line-height: 1.6;
  color: #1f2937;
}

.not-friend-dialog-desc {
  margin: 0;
  font-size: 13px;
  line-height: 1.6;
  color: #6b7280;
}
</style>
