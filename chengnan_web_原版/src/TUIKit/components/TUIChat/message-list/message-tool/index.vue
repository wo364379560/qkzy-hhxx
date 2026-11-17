<template>
  <div
    v-if="!isAllActionItemInvalid && !messageItem.hasRiskContent"
    ref="messageToolDom"
    :class="['dialog-item', !isPC ? 'dialog-item-h5' : 'dialog-item-web']"
    @mousedown.stop
  >
    <slot
      v-if="featureConfig.EmojiReaction"
      name="TUIEmojiPlugin"
    />
    <div
      class="dialog-item-list"
      :class="!isPC ? 'dialog-item-list-h5' : 'dialog-item-list-web'"
    >
      <template v-for="(item, index) in actionItems">
        <div
          v-if="item.renderCondition()"
          :key="item.key"
          class="list-item"
          @click="getFunction(index)"
          @mousedown="beforeCopy(item.key)"
        >
          <Icon
            :file="item.key === 'pin-message'
              ? (isCurrentPinnedMessage() ? cancelIcon : selectedIcon)
              : item.iconUrl"
            :size="'15px'"
          />
          <span class="list-item-text">{{
            item.key === 'pin-message'
              ? (isCurrentPinnedMessage() ? '取消置顶' : '置顶')
              : (typeof item.getText === 'function' ? item.getText() : item.text)
          }}</span>
        </div>
      </template>
    </div>
  </div>
</template>

<script lang="ts" setup>
// @ts-nocheck
import TUIChatEngine, {
  TUIChatService,
  TUIStore,
  StoreName,
  TUITranslateService,
  IMessageModel,
} from '@tencentcloud/chat-uikit-engine-lite';
import { TUIGlobal } from '@tencentcloud/universal-api';
import { ref, watchEffect, computed, onMounted, onUnmounted } from '../../../../adapter-vue';
import Icon from '../../../common/Icon.vue';
import { Toast, TOAST_TYPE } from '../../../common/Toast/index';
import delIcon from '../../../../assets/icon/msg-del.svg';
import copyIcon from '../../../../assets/icon/msg-copy.svg';
import quoteIcon from '../../../../assets/icon/msg-quote.svg';
import revokeIcon from '../../../../assets/icon/msg-revoke.svg';
import forwardIcon from '../../../../assets/icon/msg-forward.svg';
import translateIcon from '../../../../assets/icon/translate.svg';
import multipleSelectIcon from '../../../../assets/icon/multiple-select.svg';
import convertText from '../../../../assets/icon/convertText_zh.svg';
import muteIcon from '../../../../assets/icon/mute.svg';
import minusIcon from '../../../../assets/icon/minus.svg';
import selectedIcon from '../../../../assets/icon/selected.svg';
import cancelIcon from '../../../../assets/icon/cancel.svg';
import { enableSampleTaskStatus } from '../../../../utils/enableSampleTaskStatus';
import { transformTextWithKeysToEmojiNames } from '../../emoji-config';
import { isH5, isPC, isUniFrameWork } from '../../../../utils/env';
import {
  ITranslateInfo,
  IConvertInfo,
  IGroupMessageModerationContext,
} from '../../../../interface';
import TUIChatConfig from '../../config';
import AiRobotManager from '../../aiRobotManager';
import { revokeGroupMessage } from '@/api';

// uni-app conditional compilation will not run the following code
// #ifndef APP || APP-PLUS || MP || H5
import CopyManager from '../../utils/copy';
// #endif

interface IProps {
  messageItem: IMessageModel;
  isMultipleSelectMode: boolean;
  canModerateIncomingMessage?: boolean;
  groupModerationContext?: IGroupMessageModerationContext;
}

interface IEmits {
  (key: 'ready', context: { messageID: string; element: HTMLElement }): void;
  (key: 'toggleMultipleSelectMode'): void;
  (
    key: 'moderateGroupMember',
    action: 'mute' | 'unmute' | 'kick',
    context: IGroupMessageModerationContext
  ): void;
  (key: 'groupMessagePinChanged'): void;
}

const emits = defineEmits<IEmits>();
const props = withDefaults(defineProps<IProps>(), {
  isMultipleSelectMode: false,
  messageItem: () => ({}) as IMessageModel,
  canModerateIncomingMessage: false,
});
const featureConfig = TUIChatConfig.getFeatureConfig();

const TYPES = TUIChatEngine.TYPES;

const actionItems = ref([
  {
    key: 'open',
    text: TUITranslateService.t('TUIChat.打开'),
    iconUrl: copyIcon,
    renderCondition() {
      if (!featureConfig.DownloadFile || !message.value) return false;
      return isPC && (message.value?.type === TYPES.MSG_FILE
        || message.value.type === TYPES.MSG_VIDEO
        || message.value.type === TYPES.MSG_IMAGE);
    },
    clickEvent: openMessage,
  },
  {
    key: 'copy',
    text: TUITranslateService.t('TUIChat.复制'),
    iconUrl: copyIcon,
    renderCondition() {
      if (!featureConfig.CopyMessage || !message.value) return false;
      const isAiRobotTextMessage = AiRobotManager.isRobotMessage(message.value);
      return message.value.type === TYPES.MSG_TEXT || isAiRobotTextMessage;
    },
    clickEvent: copyMessage,
  },
  {
    key: 'revoke',
    text: TUITranslateService.t('TUIChat.撤回'),
    iconUrl: revokeIcon,
    renderCondition() {
      if (!featureConfig.RevokeMessage || !message.value) return false;
      return message.value.status === 'success'
        && (message.value.flow === 'out' || props.canModerateIncomingMessage);
    },
    clickEvent: revokeMessage,
  },
  {
    key: 'delete',
    text: TUITranslateService.t('TUIChat.删除'),
    iconUrl: delIcon,
    renderCondition() {
      if (!featureConfig.DeleteMessage || !message.value) return false;
      return message.value.status === 'success';
    },
    clickEvent: deleteMessage,
  },
  {
    key: 'mute-member',
    text: TUITranslateService.t('TUIChat.禁言'),
    getText() {
      return props.groupModerationContext?.isMuted
        ? TUITranslateService.t('TUIChat.取消禁言')
        : TUITranslateService.t('TUIChat.禁言');
    },
    iconUrl: muteIcon,
    renderCondition() {
      if (!message.value || message.value.status !== 'success') return false;
      return Boolean(
        props.groupModerationContext?.canMute ||
          props.groupModerationContext?.canUnmute
      );
    },
    clickEvent: toggleGroupMemberMute,
  },
  {
    key: 'kick-member',
    text: TUITranslateService.t('TUIChat.踢出群聊'),
    iconUrl: minusIcon,
    renderCondition() {
      if (!message.value || message.value.status !== 'success') return false;
      return Boolean(props.groupModerationContext?.canKick);
    },
    clickEvent: kickGroupMember,
  },
  {
    key: 'pin-message',
    text: '置顶',
    iconUrl: selectedIcon,
    renderCondition() {
      if (!message.value || message.value.status !== 'success') return false;
      return Boolean(
        props.canModerateIncomingMessage ||
          props.groupModerationContext
      );
    },
    clickEvent: pinGroupMessage,
  },
  {
    key: 'forward',
    text: TUITranslateService.t('TUIChat.转发'),
    iconUrl: forwardIcon,
    renderCondition() {
      if (!featureConfig.ForwardMessage || !message.value) return false;
      return message.value.status === 'success';
    },
    clickEvent: forwardSingleMessage,
  },
  {
    key: 'quote',
    text: TUITranslateService.t('TUIChat.引用'),
    iconUrl: quoteIcon,
    renderCondition() {
      if (!featureConfig.QuoteMessage || !message.value) return false;
      const _message = TUIStore.getMessageModel(message.value.ID);
      return message.value.status === 'success' && !_message.getSignalingInfo();
    },
    clickEvent: quoteMessage,
  },
  {
    key: 'translate',
    text: TUITranslateService.t('TUIChat.翻译'),
    visible: false,
    iconUrl: translateIcon,
    renderCondition() {
      if (!featureConfig.TranslateMessage || !message.value) return false;
      return message.value.status === 'success' && message.value.type === TYPES.MSG_TEXT;
    },
    clickEvent: translateMessage,
  },
  {
    key: 'convert',
    text: TUITranslateService.t('TUIChat.转文字'),
    visible: false,
    iconUrl: convertText,
    renderCondition() {
      if (!featureConfig.VoiceToText || !message.value) return false;
      return message.value.status === 'success' && message.value.type === TYPES.MSG_AUDIO;
    },
    clickEvent: convertVoiceToText,
  },
  {
    key: 'multi-select',
    text: TUITranslateService.t('TUIChat.多选'),
    iconUrl: multipleSelectIcon,
    renderCondition() {
      if (!featureConfig.MultiSelection || !message.value) return false;
      return message.value.status === 'success';
    },
    clickEvent: multipleSelectMessage,
  },
]);

const message = ref<IMessageModel>();
const messageToolDom = ref<HTMLElement>();
const currentGroupPinnedMessageIDs = ref<string[]>([]);

// The first load is asynchronous. Notify the list only after the real root
// exists; nextTick in the parent cannot wait for an async component import.
watchEffect(() => {
  const element = messageToolDom.value;
  if (element) {
    emits('ready', { messageID: props.messageItem.ID, element });
  }
}, { flush: 'post' });

onMounted(() => {
  TUIStore.watch(StoreName.CHAT, {
    translateTextInfo: onMessageTranslationInfoUpdated,
    voiceToTextInfo: onMessageConvertInfoUpdated,
  });
  TUIStore.watch(StoreName.CUSTOM, {
    currentGroupPinnedMessageIDs: onCurrentGroupPinnedMessageIDsUpdated,
  });
});

onUnmounted(() => {
  TUIStore.unwatch(StoreName.CHAT, {
    translateTextInfo: onMessageTranslationInfoUpdated,
    voiceToTextInfo: onMessageConvertInfoUpdated,
  });
  TUIStore.unwatch(StoreName.CUSTOM, {
    currentGroupPinnedMessageIDs: onCurrentGroupPinnedMessageIDsUpdated,
  });
});

watchEffect(() => {
  message.value = TUIStore.getMessageModel(props.messageItem.ID);
  currentGroupPinnedMessageIDs.value =
    TUIStore.getData(StoreName.CUSTOM, 'currentGroupPinnedMessageIDs') || [];
});

const isAllActionItemInvalid = computed(() => {
  for (let i = 0; i < actionItems.value.length; ++i) {
    if (actionItems.value[i].renderCondition()) {
      return false;
    }
  }
  return true;
});

function getFunction(index: number) {
  // Compatible with Vue2 and WeChat Mini Program syntax, dynamic binding is not allowed.
  actionItems.value[index].clickEvent();
}

function openMessage() {
  let url = '';
  switch (message.value?.type) {
    case TUIChatEngine.TYPES.MSG_FILE:
      url = message.value.payload.fileUrl;
      break;
    case TUIChatEngine.TYPES.MSG_VIDEO:
      url = message.value.payload.remoteVideoUrl;
      break;
    case TUIChatEngine.TYPES.MSG_IMAGE:
      url = message.value.payload.imageInfoArray[0].url;
      break;
  }
  window?.open(url, '_blank');
}

async function revokeMessage() {
  if (!message.value) return;
  const messageModel =
    TUIStore.getMessageModel(message.value.ID) ||
    props.messageItem ||
    message.value;
  const isModeratorRevokeOthers =
    props.canModerateIncomingMessage &&
    messageModel?.flow === 'in' &&
    Boolean(messageModel?.conversationID?.startsWith('GROUP'));

  // 群主/管理员撤回群成员消息：走后端 REST，无 2 分钟时限（对齐 chixin）
  if (isModeratorRevokeOthers) {
    const groupID =
      String(messageModel.conversationID || '').replace(/^GROUP/, '') ||
      messageModel.to ||
      '';
    const msgSeq = Number(messageModel.sequence || 0);
    if (!groupID || !msgSeq) {
      Toast({
        message: '撤回失败',
        type: TOAST_TYPE.ERROR,
      });
      return;
    }
    try {
      await revokeGroupMessage(groupID, msgSeq);
      enableSampleTaskStatus('revokeMessage');
    } catch (error: any) {
      Toast({
        message: error?.message || '撤回失败',
        type: TOAST_TYPE.ERROR,
      });
    }
    return;
  }

  if (!messageModel?.revokeMessage) {
    return;
  }
  messageModel
    .revokeMessage()
    .then(() => {
      enableSampleTaskStatus('revokeMessage');
    })
    .catch((error: any) => {
      // 自己撤回自己的消息仍受 IM 默认时限约束
      if (error.code === 20016 || error.code === 10031) {
        Toast({
          message: TUITranslateService.t('TUIChat.已过撤回时限'),
          type: TOAST_TYPE.ERROR,
        });
      }
    });
}

function deleteMessage() {
  if (!message.value) return;
  if (AiRobotManager.isStreamingMessage(message.value as IMessageModel)) {
    const message = TUITranslateService.t('TUIChat.回答输出中，请稍后或点击停止回答');
    return Toast({
      message,
      type: TOAST_TYPE.NORMAL,
    });
  }
  const messageModel = TUIStore.getMessageModel(message.value.ID);
  messageModel.deleteMessage();
}

function toggleGroupMemberMute() {
  if (!props.groupModerationContext) {
    return;
  }
  emits(
    'moderateGroupMember',
    props.groupModerationContext.isMuted ? 'unmute' : 'mute',
    props.groupModerationContext
  );
}

function kickGroupMember() {
  if (!props.groupModerationContext) {
    return;
  }
  emits('moderateGroupMember', 'kick', props.groupModerationContext);
}

async function copyMessage() {
  if (AiRobotManager.isStreamingMessage(message.value as IMessageModel)) {
    const message = TUITranslateService.t('TUIChat.回答输出中，请稍后或点击停止回答');
    return Toast({
      message,
      type: TOAST_TYPE.NORMAL,
    });
  }
  const isAiRobotText = AiRobotManager.getRobotRenderText(message.value as IMessageModel);
  const text = isAiRobotText ? isAiRobotText : message.value?.payload.text;

  if (isUniFrameWork) {
    TUIGlobal?.setClipboardData({
      data: transformTextWithKeysToEmojiNames(text),
    });
  } else {
    // uni-app conditional compilation will not run the following code
    // #ifndef APP || APP-PLUS || MP || H5
    CopyManager.copySelection(text);
    // #endif
  }
}

function beforeCopy(key: string) {
  // only pc support copy selection or copy full message text
  // uni-app and h5 only support copy full message text
  if (key !== 'copy' || isH5) {
    return;
  }

  // uni-app conditional compilation will not run the following code
  // #ifndef APP || APP-PLUS || MP || H5
  CopyManager.saveCurrentSelection();
  // #endif
}

function forwardSingleMessage() {
  if (!message.value) return;
  if (AiRobotManager.isStreamingMessage(message.value as IMessageModel)) {
    const message = TUITranslateService.t('TUIChat.回答输出中，请稍后或点击停止回答');
    return Toast({
      message,
      type: TOAST_TYPE.NORMAL,
    });
  }
  TUIStore.update(StoreName.CUSTOM, 'singleForwardMessageID', message.value.ID);
}

function quoteMessage() {
  if (!message.value) return;
  TUIChatService.quoteMessage(getQuoteMessageWithGroupSenderName(message.value));
}

function getQuoteMessageWithGroupSenderName(sourceMessage: IMessageModel) {
  if (!isGroupMessage(sourceMessage)) {
    return sourceMessage;
  }

  const senderName = resolveGroupQuoteSenderName(sourceMessage);
  if (!senderName || senderName === sourceMessage.nick) {
    return sourceMessage;
  }

  return {
    ...sourceMessage,
    nick: senderName,
  };
}

function isGroupMessage(sourceMessage: IMessageModel) {
  return (
    sourceMessage.conversationType === TYPES.CONV_GROUP ||
    sourceMessage.conversationID?.startsWith(TYPES.CONV_GROUP)
  );
}

function resolveGroupQuoteSenderName(sourceMessage: IMessageModel) {
  const memberList =
    TUIStore.getData(StoreName.GRP, 'currentGroupMemberList') || [];
  const matchedMember = Array.isArray(memberList)
    ? memberList.find((member: any) => member?.userID === sourceMessage.from)
    : undefined;

  return (
    matchedMember?.nameCard ||
    matchedMember?.nick ||
    sourceMessage.nameCard ||
    sourceMessage.from ||
    ''
  );
}

function translateMessage() {
  const enable = TUIStore.getData(StoreName.APP, 'enabledTranslationPlugin');
  if (!enable) {
    Toast({
      message: TUITranslateService.t('TUIChat.请开通翻译功能'),
      type: TOAST_TYPE.WARNING,
    });
    return;
  }

  if (!message.value) return;
  const index = actionItems.value.findIndex(item => item.key === 'translate');
  TUIStore.update(StoreName.CHAT, 'translateTextInfo', {
    conversationID: message.value.conversationID,
    messageID: message.value.ID,
    visible: !actionItems.value[index].visible,
  });
}

function convertVoiceToText() {
  const enable = TUIStore.getData(StoreName.APP, 'enabledVoiceToText');
  if (!enable) {
    Toast({
      message: TUITranslateService.t('TUIChat.请开通语音转文字功能'),
      type: '',
    });
    return;
  }

  if (!message.value) return;
  const index = actionItems.value.findIndex(item => item.key === 'convert');
  TUIStore.update(StoreName.CHAT, 'voiceToTextInfo', {
    conversationID: message.value.conversationID,
    messageID: message.value.ID,
    visible: !actionItems.value[index].visible,
  });
}

function multipleSelectMessage() {
  emits('toggleMultipleSelectMode');
}

async function pinGroupMessage() {
  if (!message.value?.conversationID?.startsWith('GROUP')) {
    return;
  }
  const groupID = message.value.conversationID.slice(5);
  try {
    const rawMessage = message.value.getMessage?.();
    if (!rawMessage) {
      throw new Error('无法获取消息对象');
    }
    await TUIChatEngine.chat.pinGroupMessage({
      groupID,
      message: rawMessage,
      isPinned: !isCurrentPinnedMessage(),
    });
    emits('groupMessagePinChanged');
  } catch (error: any) {
    Toast({
      message: error?.message || '置顶消息失败',
      type: TOAST_TYPE.ERROR,
    });
  }
}

function isCurrentPinnedMessage() {
  return Boolean(
    message.value?.ID &&
      currentGroupPinnedMessageIDs.value.includes(message.value.ID),
  );
}

function onCurrentGroupPinnedMessageIDsUpdated(value: string[]) {
  currentGroupPinnedMessageIDs.value = Array.isArray(value) ? value : [];
}

function onMessageTranslationInfoUpdated(info: Map<string, ITranslateInfo[]>) {
  if (info === undefined) return;
  const translationInfoList = info.get(props.messageItem.conversationID) || [];
  const idx = actionItems.value.findIndex(item => item.key === 'translate');
  for (let i = 0; i < translationInfoList.length; ++i) {
    const { messageID, visible } = translationInfoList[i];
    if (messageID === props.messageItem.ID) {
      actionItems.value[idx].text = TUITranslateService.t(visible ? 'TUIChat.隐藏' : 'TUIChat.翻译');
      actionItems.value[idx].visible = !!visible;
      return;
    }
  }
  actionItems.value[idx].text = TUITranslateService.t('TUIChat.翻译');
}

function onMessageConvertInfoUpdated(info: Map<string, IConvertInfo[]>) {
  if (info === undefined) return;
  const convertInfoList = info.get(props.messageItem.conversationID) || [];
  const idx = actionItems.value.findIndex(item => item.key === 'convert');
  for (let i = 0; i < convertInfoList.length; ++i) {
    const { messageID, visible } = convertInfoList[i];
    if (messageID === props.messageItem.ID) {
      actionItems.value[idx].text = TUITranslateService.t(visible ? 'TUIChat.隐藏' : 'TUIChat.转文字');
      actionItems.value[idx].visible = !!visible;
      return;
    }
  }
  actionItems.value[idx].text = TUITranslateService.t('TUIChat.转文字');
}

defineExpose({
  messageToolDom,
});
</script>

<style lang="scss" scoped>
@import "../../../../assets/styles/common";

.dialog-item-web {
  background: #fff;
  border-radius: 8px;
  border: 1px solid #e0e0e0;
  padding: 12px 0;

  .dialog-item-list {
    display: flex;
    align-items: baseline;
    white-space: nowrap;
    flex-wrap: wrap;
    max-width: 280px;

    .list-item {
      padding: 4px 12px;
      display: flex;
      flex-direction: row;
      align-items: center;

      .list-item-text {
        padding-left: 4px;
        font-size: 12px;
        line-height: 17px;
        color: #000;
      }
    }
  }
}

.dialog-item-h5 {
  @extend .dialog-item-web;

  padding: 0;

  .dialog-item-list {
    margin: 10px;
    white-space: nowrap;
    flex-wrap: wrap;
    max-width: 280px;

    .list-item {
      padding: 0 8px;
      display: flex;
      flex-direction: column;
      align-items: center;
      color: #4f4f4f;

      .list-item-text {
        padding-left: 0;
        color: #000;
      }
    }
  }
}
</style>
