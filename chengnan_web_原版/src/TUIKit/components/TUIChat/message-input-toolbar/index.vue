<template>
  <div
    :class="[
      'message-input-toolbar',
      !isPC && 'message-input-toolbar-h5',
      isUniFrameWork && 'message-input-toolbar-uni',
    ]"
  >
    <div
      :class="[
        'message-input-toolbar-list',
        !isPC && 'message-input-toolbar-h5-list',
        isUniFrameWork && 'message-input-toolbar-uni-list',
      ]"
    >
      <EmojiPicker
        v-if="isRenderedEmojiPicker"
        ref="emojiPickerRef"
        @insertEmoji="insertEmoji"
        @dialogShowInH5="dialogShowInH5"
        @dialogCloseInH5="dialogCloseInH5"
        @changeToolbarDisplayType="(type) => emits('changeToolbarDisplayType', type)"
      />
      <ImageUpload
        v-if="featureConfig.InputImage"
        imageSourceType="album"
      />
      <FileUpload v-if="featureConfig.InputFile" />
      <VideoUpload
        v-if="featureConfig.InputVideo"
        videoSourceType="album"
      />
      <Evaluate v-if="featureConfig.InputEvaluation" />
      <Words v-if="featureConfig.InputQuickReplies" />
      <ClearHistory v-if="featureConfig.ClearHistory" />
      <template v-if="extensionListShowInStart[0]">
        <ToolbarItemContainer
          v-for="(extension, index) in extensionListShowInStart"
          :key="extension?.data?.name || index"
          :iconFile="genExtensionIcon(extension)"
          :title="genExtensionText(extension)"
          :iconWidth="isUniFrameWork ? '25px' : '20px'"
          :iconHeight="isUniFrameWork ? '25px' : '20px'"
          :needDialog="false"
          @onIconClick="onExtensionClick(extension)"
        />
      </template>
    </div>
    <div
      v-if="extensionListShowInEnd[0] && isPC"
      :class="['message-input-toolbar-list-end']"
    >
      <ToolbarItemContainer
        v-for="(extension, index) in extensionListShowInEnd"
        :key="index"
        :iconFile="genExtensionIcon(extension)"
        :title="genExtensionText(extension)"
        :iconWidth="isUniFrameWork ? '25px' : '20px'"
        :iconHeight="isUniFrameWork ? '25px' : '20px'"
        :needDialog="false"
        @onIconClick="onExtensionClick(extension)"
      />
    </div>
    <UserSelector
      ref="userSelectorRef"
      :type="selectorShowType"
      :currentConversation="currentConversation"
      :isGroup="isGroup"
      @submit="onUserSelectorSubmit"
      @cancel="onUserSelectorCancel"
    />
    <div
      v-if="isH5"
      ref="h5Dialog"
      :class="['message-input-toolbar-h5-dialog']"
    />
  </div>
</template>
<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from '../../../adapter-vue';
import TUIChatEngine, {
  IConversationModel,
  TUIStore,
  StoreName,
  TUIReportService,
} from '@tencentcloud/chat-uikit-engine-lite';
import TUICore, { ExtensionInfo, TUIConstants } from '@tencentcloud/tui-core-lite';
import EmojiPicker from './emoji-picker/index.vue';
import ImageUpload from './image-upload/index.vue';
import FileUpload from './file-upload/index.vue';
import VideoUpload from './video-upload/index.vue';
import Evaluate from './evaluate/index.vue';
import Words from './words/index.vue';
import ClearHistory from './clearHistory/index.vue';
import ToolbarItemContainer from './toolbar-item-container/index.vue';
import UserSelector from './user-selector/index.vue';
import { isPC, isH5, isUniFrameWork } from '../../../utils/env';
import TUIChatConfig from '../config';
import { enableSampleTaskStatus } from '../../../utils/enableSampleTaskStatus';
import { ToolbarDisplayType } from '../../../interface';
import AiRobotManager from '../aiRobotManager';
import { startDoMattxC2CCall } from '@/features/call/doMattxCallBridge';
import { DoMattxCallMediaType } from '@/features/call/doMattxCallProtocol';
import { Toast, TOAST_TYPE } from '../../common/Toast/index';
import voiceCallIcon from '../../../assets/icon/call-voice.svg';
import videoCallIcon from '../../../assets/icon/call-video.svg';

interface IProps {
  displayType: ToolbarDisplayType;
}
interface IEmits {
  (e: 'scrollToLatestMessage'): void;
  (e: 'changeToolbarDisplayType', type: ToolbarDisplayType): void;
  (e: 'insertEmoji', emoji: any): void
}
const props = withDefaults(defineProps<IProps>(), {
  displayType: 'none',
});

const emits = defineEmits<IEmits>();
const h5Dialog = ref<HTMLElement>();
const currentConversation = ref<IConversationModel>();
const isGroup = ref<boolean>(false);
const selectorShowType = ref<string>('');
const userSelectorRef = ref();
const emojiPickerRef = ref<InstanceType<typeof EmojiPicker>>();
const currentUserSelectorExtension = ref<ExtensionInfo>();
const currentExtensionList = ref<ExtensionInfo[]>([]);
const featureConfig = ref(TUIChatConfig.getFeatureConfig());
const isRenderedEmojiPicker = ref<boolean>(true);
const currentGroupAttributes = ref<Record<string, string>>({});

onMounted(() => {
  TUIStore.watch(StoreName.CUSTOM, {
    activeConversation: onActiveConversationUpdate,
  });
  TUIStore.watch(StoreName.GRP, {
    currentGroupAttributes: onCurrentGroupAttributesUpdate,
  });
});

onUnmounted(() => {
  TUIStore.unwatch(StoreName.CUSTOM, {
    activeConversation: onActiveConversationUpdate,
  });
  TUIStore.unwatch(StoreName.GRP, {
    currentGroupAttributes: onCurrentGroupAttributesUpdate,
  });
});

watch(() => props.displayType, (newValue) => {
  if (newValue === 'none') {
    emojiPickerRef.value?.closeEmojiPicker();
  } else {
    emits('scrollToLatestMessage');
  }
});

const onActiveConversationUpdate = async (conversationID: string) => {
  if (!conversationID) {
    return;
  }
  if (conversationID !== currentConversation.value?.conversationID) {
    if (AiRobotManager.isRobot(conversationID)) {
      AiRobotManager.initAiRobotChat(conversationID);
    } else {
      TUIChatConfig.resetFeatureConfig();
    }
    featureConfig.value = TUIChatConfig.getFeatureConfig();
    isRenderedEmojiPicker.value = featureConfig.value.InputEmoji || featureConfig.value.InputStickers;
    currentConversation.value = TUIStore.getData(StoreName.CONV, 'currentConversation');
    isGroup.value = conversationID.startsWith(TUIChatEngine.TYPES.CONV_GROUP);
    if (isGroup.value) {
      const nextGroupID =
        currentConversation.value?.groupProfile?.groupID || conversationID.slice(5);
      const activeGroupID = TUIStore.getData(StoreName.GRP, 'currentGroupID');
      currentGroupAttributes.value =
        activeGroupID === nextGroupID
          ? TUIStore.getData(StoreName.GRP, 'currentGroupAttributes') || {}
          : {};
    } else {
      currentGroupAttributes.value = {};
    }
    getExtensionList();
  }
};

const getExtensionList = () => {
  const chatType = TUIChatConfig.getChatType();
  const params: Record<string, boolean | string> = { chatType };
  // Backward compatibility: When callkit does not have chatType judgment, use filterVoice and filterVideo to filter
  if (chatType === TUIConstants.TUIChat.TYPE.CUSTOMER_SERVICE) {
    params.filterVoice = true;
    params.filterVideo = true;
    enableSampleTaskStatus('customerService');
  }
  currentExtensionList.value = [
    ...TUICore.getExtensionList(TUIConstants.TUIChat.EXTENSION.INPUT_MORE.EXT_ID, params),
  ].filter((extension: ExtensionInfo) => {
    if (extension?.data?.name === 'search') {
      return featureConfig.value.MessageSearch;
    }
    if (extension?.data?.name === 'voiceCall' || extension?.data?.name === 'videoCall') {
      return false;
    }
    return true;
  });
  if (currentConversation.value?.type === TUIChatEngine.TYPES.CONV_C2C && chatType !== 'aiRobot') {
    currentExtensionList.value = [
      ...currentExtensionList.value,
      {
        icon: voiceCallIcon,
        text: '语音通话',
        data: { name: 'voiceCall' },
        weight: 0,
        listener: {},
      },
      {
        icon: videoCallIcon,
        text: '视频通话',
        data: { name: 'videoCall' },
        weight: 0,
        listener: {},
      },
    ];
  }
  reportExtension(currentExtensionList.value);
};

function onCurrentGroupAttributesUpdate(groupAttributes: Record<string, string>) {
  const currentGroupID = currentConversation.value?.groupProfile?.groupID;
  const activeGroupID = TUIStore.getData(StoreName.GRP, 'currentGroupID');
  if (!isGroup.value || !currentGroupID || activeGroupID !== currentGroupID) {
    return;
  }
  currentGroupAttributes.value = groupAttributes || {};
  getExtensionList();
}

function reportExtension(extensionList:ExtensionInfo[]){
  extensionList.forEach((extension: ExtensionInfo)=>{
    const _name = extension?.data?.name;
    if(_name === 'voiceCall'){
      TUIReportService.reportFeature(203, 'voice-call');
    } else if (_name === 'videoCall') {
      TUIReportService.reportFeature(203, 'video-call');
    } else if(_name === 'quickRoom'){
      TUIReportService.reportFeature(204);
    }
  });
}

const extensionListShowInStart = computed<ExtensionInfo[]>(() => {
  if (isPC) {
    const extensionList = currentExtensionList.value.filter((extension: ExtensionInfo) => extension?.data?.name !== 'search');
    return extensionList;
  }
  return currentExtensionList.value;
});

const extensionListShowInEnd = computed<ExtensionInfo[]>(() => {
  if (isPC) {
    const searchExtension = currentExtensionList.value.find(extension => extension?.data?.name === 'search');
    return searchExtension ? [searchExtension] : [];
  }
  return [];
});

// handle extensions onclick
const onExtensionClick = (extension: ExtensionInfo) => {
  switch (extension?.data?.name) {
    case 'voiceCall':
      onCallExtensionClicked(extension, 1);
      break;
    case 'videoCall':
      onCallExtensionClicked(extension, 2);
      break;
    default:
      (extension as any)?.listener?.onClicked(currentConversation.value?._conversation);
      break;
  }
};

const onCallExtensionClicked = async (extension: ExtensionInfo, callType: number) => {
  selectorShowType.value = extension?.data?.name;
  if (isGroup.value) {
    Toast({
      message: '暂不支持群通话',
      type: TOAST_TYPE.WARNING,
    });
    return;
  }
  if (currentConversation?.value?.type === TUIChatEngine.TYPES.CONV_C2C) {
    const peerUserID = currentConversation?.value?.conversationID?.slice(3) || '';
    const userProfile = currentConversation?.value?.userProfile;
    try {
      await startDoMattxC2CCall(
        peerUserID,
        callType === DoMattxCallMediaType.Video
          ? DoMattxCallMediaType.Video
          : DoMattxCallMediaType.Audio,
        userProfile?.nick || peerUserID,
        userProfile?.avatar,
      );
    } catch (error: any) {
      Toast({
        message: error?.message || '发起通话失败',
        type: TOAST_TYPE.ERROR,
      });
    }
  }
};

const genExtensionIcon = (extension: any) => {
  return extension?.icon;
};

const genExtensionText = (extension: any) => {
  return extension?.text;
};

const onUserSelectorSubmit = () => {
  Toast({
    message: '暂不支持群通话',
    type: TOAST_TYPE.WARNING,
  });
  currentUserSelectorExtension.value = undefined;
};

const onUserSelectorCancel = () => {
  currentUserSelectorExtension.value = undefined;
};

const insertEmoji = (emojiObj: object) => {
  emits('insertEmoji', emojiObj);
};

const dialogShowInH5 = (dialogDom: any) => {
  if (!isH5) {
    return;
  }
  h5Dialog.value?.appendChild && h5Dialog.value?.appendChild(dialogDom);
};

const dialogCloseInH5 = (dialogDom: any) => {
  if (!isH5) {
    return;
  }
  if (dialogDom) {
    h5Dialog.value?.removeChild && h5Dialog.value?.removeChild(dialogDom);
  }
};
</script>
<style lang="scss" scoped>
@import "../../../assets/styles/common";

.message-input-toolbar {
  border-top: 1px solid #f4f5f9;
  width: 100%;
  max-width: 100%;
  display: flex;
  flex-direction: row;
  justify-content: space-between;
  user-select: none;
  -webkit-touch-callout: none;
  -webkit-user-select: none;
  -khtml-user-select: none;
  -moz-user-select: none;
  -ms-user-select: none;

  &-list {
    display: flex;
    flex-direction: row;
    align-items: center;

    .extension-list {
      list-style: none;
      display: flex;

      &-item {
        width: 20px;
        height: 20px;
        padding: 12px 10px 1px;
        cursor: pointer;
      }
    }
  }
}

.message-input-toolbar-h5 {
  padding: 5px 10px;
  box-sizing: border-box;
  flex-direction: column;
}

.message-input-toolbar-uni {
  background-color: #ebf0f6;
  flex-direction: column;

  &-list {
    flex: 1;
    display: grid;
    grid-template-columns: repeat(4, 25%);
    grid-template-rows: repeat(2, 100px);
  }
}
</style>
