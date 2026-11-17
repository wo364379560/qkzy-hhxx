<template>
  <div
    v-show="isShowReadStatus"
    :class="{
      'message-label': true,
      'unread': isUseUnreadStyle,
      'finger-point': isHoverFingerPointer,
    }"
    @click="openReadUserPanel"
  >
    <span>{{ readStatusText }}</span>
  </div>
</template>

<script setup lang="ts">
import {
  computed,
  ref,
  onMounted,
  onUnmounted,
  watch,
} from '../../../../../adapter-vue';
import TUIChatEngine, {
  TUIStore,
  StoreName,
  IMessageModel,
  TUITranslateService,
} from '@tencentcloud/chat-uikit-engine-lite';
import TUIChatConfig from '../../../config';

interface IProps {
  message: IMessageModel;
}

interface IEmits {
  (e: 'openReadUserPanel'): void;
}

const emits = defineEmits<IEmits>();
const props = withDefaults(defineProps<IProps>(), {
  message: () => ({}) as IMessageModel,
});
const ReadStatus = TUIChatConfig.getFeatureConfig('ReadStatus');

enum ReadState {
  Read,
  Unread,
  AllRead,
  NotShow,
  PartiallyRead,
}

const TYPES = TUIChatEngine.TYPES;
// User-level read receipt toggle has the highest priority.
const isDisplayMessageReadReceipt = ref<boolean>(TUIStore.getData(StoreName.USER, 'displayMessageReadReceipt') !== false);
const currentMessage = ref<IMessageModel>(createMessageSnapshot(props.message));

onMounted(() => {
  TUIStore.watch(StoreName.USER, {
    displayMessageReadReceipt: onDisplayMessageReadReceiptUpdate,
  });
  TUIStore.watch(StoreName.CHAT, {
    messageList: onMessageListUpdated,
  });
  syncCurrentMessage();
});

onUnmounted(() => {
  TUIStore.unwatch(StoreName.USER, {
    displayMessageReadReceipt: onDisplayMessageReadReceiptUpdate,
  });
  TUIStore.unwatch(StoreName.CHAT, {
    messageList: onMessageListUpdated,
  });
});

watch(
  () => [
    props.message.ID,
    props.message.isPeerRead,
    props.message.readReceiptInfo?.isPeerRead,
    props.message.readReceiptInfo?.readCount,
    props.message.readReceiptInfo?.unreadCount,
  ],
  () => {
    syncCurrentMessage();
  },
  { immediate: true },
);

const message = computed<IMessageModel>(() => currentMessage.value || props.message);

const isShowReadStatus = computed<boolean>(() => {
  if (!ReadStatus) {
    return false;
  }
  if (!isDisplayMessageReadReceipt.value) {
    return false;
  }
  const {
    ID,
    type,
    flow,
    status,
    hasRiskContent,
    conversationID,
    conversationType,
    needReadReceipt = false,
  } = message.value;

  // Asynchronous message strike: Determine if there is risky content after the message has been sent
  if (hasRiskContent) {
    return false;
  }

  const { groupProfile } = TUIStore.getConversationModel(conversationID) || {};
  // AVCHATROOM and COMMUNITY chats do not display read status
  if (groupProfile?.type === TYPES.GRP_AVCHATROOM || groupProfile?.type === TYPES.GRP_COMMUNITY) {
    return false;
  }

  if (type === TYPES.MSG_CUSTOM) {
    const message = TUIStore.getMessageModel(ID);
    // If it is a signaling message, do not display the read status
    if (message?.getSignalingInfo() !== null) {
      return false;
    }
  }

  // Unsuccessful message: Received messages do not display read status
  if (flow !== 'out' || status !== 'success') {
    return false;
  }

  if (conversationType === 'GROUP') {
    return needReadReceipt;
  } else if (conversationType === 'C2C') {
    return true;
  }
  return false;
});

const readState = computed<ReadState>(() => {
  const { conversationType, needReadReceipt = false, isPeerRead = false } = message.value;
  const {
    readCount = 0,
    unreadCount = 0,
    isPeerRead: isReceiptPeerRead = false,
  } = message.value.readReceiptInfo || {};
  if (conversationType === 'C2C') {
    // Native clients may report either basic C2C reads or per-message receipts.
    return isPeerRead || isReceiptPeerRead ? ReadState.Read : ReadState.Unread;
  } else if (conversationType === 'GROUP') {
    if (needReadReceipt) {
      if (readCount === 0) {
        return ReadState.Unread;
      } else if (unreadCount === 0) {
        return ReadState.AllRead;
      } else {
        return ReadState.PartiallyRead;
      }
    } else {
      return ReadState.NotShow;
    }
  }
  return ReadState.Unread;
});

const readStatusText = computed(() => {
  const { readCount = 0 } = message.value.readReceiptInfo || {};
  switch (readState.value) {
    case ReadState.Read:
      return TUITranslateService.t('TUIChat.已读');
    case ReadState.Unread:
      return TUITranslateService.t('TUIChat.未读');
    case ReadState.AllRead:
      return TUITranslateService.t('TUIChat.全部已读');
    case ReadState.PartiallyRead:
      return `${readCount}${TUITranslateService.t('TUIChat.人已读')}`;
    default:
      return '';
  }
});

const isUseUnreadStyle = computed(() => {
  const { conversationType } = message.value;
  if (conversationType === 'C2C') {
    return readState.value !== ReadState.Read;
  } else if (conversationType === 'GROUP') {
    return readState.value !== ReadState.AllRead;
  }
  return false;
});

const isHoverFingerPointer = computed<boolean>(() => {
  return (
    message.value.needReadReceipt
    && message.value.conversationType === 'GROUP'
    && (readState.value === ReadState.PartiallyRead || readState.value === ReadState.Unread)
  );
});

function openReadUserPanel() {
  if (isHoverFingerPointer.value) {
    emits('openReadUserPanel');
  }
}

function onDisplayMessageReadReceiptUpdate(isDisplay: boolean) {
  isDisplayMessageReadReceipt.value = isDisplay;
}

function onMessageListUpdated() {
  syncCurrentMessage();
}

function syncCurrentMessage() {
  const latestMessage = props.message?.ID
    ? TUIStore.getMessageModel(props.message.ID)
    : undefined;
  currentMessage.value = createMessageSnapshot(latestMessage || props.message);
}

// Force a fresh reactive snapshot because the SDK may mutate the same message object in place.
function createMessageSnapshot(message: IMessageModel | undefined): IMessageModel {
  return {
    ...(message || {}),
    ID: message?.ID,
    type: message?.type,
    flow: message?.flow,
    status: message?.status,
    hasRiskContent: message?.hasRiskContent,
    conversationID: message?.conversationID,
    conversationType: message?.conversationType,
    needReadReceipt: message?.needReadReceipt,
    isPeerRead: message?.isPeerRead,
    readReceiptInfo: {
      ...(message?.readReceiptInfo || {}),
    },
  } as IMessageModel;
}
</script>

<style scoped lang="scss">
.message-label {
  align-self: flex-end;
  font-size: 12px;
  color: #b6b8ba;
  word-break: keep-all;
  flex: 0 0 auto;

  &.unread {
    color: #679ce1 !important;
  }
}

.finger-point {
  cursor: pointer;
  -webkit-tap-highlight-color: transparent;
}
</style>
