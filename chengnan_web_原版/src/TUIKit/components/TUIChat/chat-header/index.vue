<template>
  <div :class="['chat-header', !isPC && 'chat-header-h5']">
    <div
      v-if="!isPC && isNotRoomChat"
      :class="['chat-header-back', !isPC && 'chat-header-h5-back']"
      @click="closeChat(currentConversation?.conversationID)"
    >
      <Icon :file="backSVG" />
    </div>
    <div class="chat-header-container">
      <div
        v-if="isNotRoomChat"
        :class="[
          'chat-header-content',
          !isPC && 'chat-header-h5-content',
          canOpenChatProfile && 'chat-header-content-clickable',
        ]"
        @click="openChatProfile"
      >
        <div class="chat-header-title">{{ currentConversationName }}</div>
        <div
          v-if="showConversationMeta"
          class="chat-header-meta"
        >
          <span
            v-if="conversationOnlineStatusText"
            :class="[
              'chat-header-meta-status',
              isPeerOnline && 'chat-header-meta-status-online',
            ]"
          >
            {{ conversationOnlineStatusText }}
          </span>
          <span
            v-if="currentLoginDisplay"
            class="chat-header-meta-ip"
          >
            {{ currentLoginDisplay }}
          </span>
          <span
            v-if="visiblePeerLastOnlineLabel"
            class="chat-header-meta-last-online"
          >
            {{ visiblePeerLastOnlineLabel }}
          </span>
        </div>
      </div>
      <div>
        <!-- <JoinGroupCard v-if="isPC" /> -->
      </div>
    </div>
    <div :class="['chat-header-setting', !isPC && 'chat-header-h5-setting']">
      <button
        v-if="showHeaderProfileEntry"
        type="button"
        class="chat-header-profile-entry"
        @click.stop="handlePrimaryHeaderAction"
      >
        <Icon :file="headerProfileIcon" />
      </button>
      <button
        v-for="(item, index) in extraHeaderExtensions"
        :key="index"
        type="button"
        class="chat-header-profile-entry"
        @click.stop="handleExtensions(item)"
      >
        <Icon :file="item.icon" />
      </button>
    </div>
  </div>
</template>
<script lang="ts" setup>
import { ref, onMounted, onUnmounted, withDefaults, computed } from '../../../adapter-vue';
import TUIChatEngine, {
  TUIStore,
  StoreName,
  TUITranslateService,
  IConversationModel,
} from '@tencentcloud/chat-uikit-engine-lite';
import { TUIConstants, ExtensionInfo } from '@tencentcloud/tui-core-lite';
// import { JoinGroupCard } from '@trtc/calls-uikit-vue';
import Icon from '../../common/Icon.vue';
import backSVG from '../../../assets/icon/back.svg';
import settingSVG from '../../../assets/icon/setting.svg';
import { isPC } from '../../../utils/env';
import TUIChatConfig from '../config';
import { getUserLoginIP } from '@/api';
import { isUserOnlineStatus } from '@/features/presence/invisible';
import type { IUserStatus, IUserStatusMap } from '../../../interface';

const props = withDefaults(
  defineProps<{
    headerExtensionList: ExtensionInfo[];
    isGroup?: boolean;
    peerLastOnlineLabel?: string;
  }>(),
  {
    headerExtensionList: () => ([]),
    isGroup: false,
    peerLastOnlineLabel: '',
  },
);

const emits = defineEmits(['closeChat', 'openConversationProfile']);
const currentConversation = ref<IConversationModel>();
const currentConversationName = ref('');
const typingStatus = ref(false);
const groupID = ref('');
const peerUserID = ref('');
const currentLoginIP = ref('');
const currentLoginLocation = ref('');
const displayOnlineStatus = ref(false);
const userOnlineStatusMap = ref<IUserStatusMap>();
const isNotRoomChat = ref<boolean>(TUIChatConfig.getChatType() !== TUIConstants.TUIChat.TYPE.ROOM);
let loginIPRequestSerial = 0;
const primaryHeaderExtension = computed(() =>
  props.isGroup ? props.headerExtensionList[0] : undefined
);
const canOpenChatProfile = computed(() => {
  if (props.isGroup) {
    return Boolean(primaryHeaderExtension.value);
  }
  const conversationID = currentConversation.value?.conversationID || '';
  return Boolean(
    currentConversation.value?.userProfile?.userID
      || (conversationID.startsWith('C2C') ? conversationID.slice(3) : ''),
  );
});
const showHeaderProfileEntry = computed(() => canOpenChatProfile.value);
const headerProfileIcon = computed(() => {
  return primaryHeaderExtension.value?.icon || settingSVG;
});
const extraHeaderExtensions = computed(() => {
  if (props.isGroup) {
    return props.headerExtensionList.slice(1);
  }
  return props.headerExtensionList;
});

onMounted(() => {
  TUIStore.watch(StoreName.CONV, {
    currentConversation: onCurrentConversationUpdated,
  });

  TUIStore.watch(StoreName.CHAT, {
    typingStatus: onTypingStatusUpdated,
  });

  TUIStore.watch(StoreName.USER, {
    displayOnlineStatus: onDisplayOnlineStatusUpdated,
    userStatusList: onUserStatusListUpdated,
  });

  syncOnlineStatusFromStore();
  syncCurrentConversationFromStore();
});

onUnmounted(() => {
  TUIStore.unwatch(StoreName.CONV, {
    currentConversation: onCurrentConversationUpdated,
  });

  TUIStore.unwatch(StoreName.CHAT, {
    typingStatus: onTypingStatusUpdated,
  });

  TUIStore.unwatch(StoreName.USER, {
    displayOnlineStatus: onDisplayOnlineStatusUpdated,
    userStatusList: onUserStatusListUpdated,
  });
});

const closeChat = (conversationID: string | undefined) => {
  emits('closeChat', conversationID);
};

const handleExtensions = (item: ExtensionInfo) => {
  item.listener.onClicked?.({ groupID: groupID.value });
};

const handlePrimaryHeaderAction = () => {
  openChatProfile();
};

const openChatProfile = () => {
  if (!canOpenChatProfile.value) {
    return;
  }
  emits('openConversationProfile');
};

function syncCurrentConversationFromStore() {
  const conversation = TUIStore.getData(StoreName.CONV, 'currentConversation');
  if (conversation) {
    onCurrentConversationUpdated(conversation as IConversationModel);
  }
}

function onCurrentConversationUpdated(conversation: IConversationModel) {
  currentConversation.value = conversation;
  groupID.value = currentConversation.value?.groupProfile?.groupID || "";
  currentConversationName.value = currentConversation.value?.getShowName() || "";
  const nextPeerUserID = conversation?.type === TUIChatEngine.TYPES.CONV_C2C
    ? getConversationPeerUserID(conversation)
    : '';
  if (nextPeerUserID !== peerUserID.value) {
    peerUserID.value = nextPeerUserID;
    void loadCurrentLoginIP(nextPeerUserID);
  }
}

function onTypingStatusUpdated(status: boolean) {
  typingStatus.value = status;
  if (typingStatus.value) {
    currentConversationName.value = TUITranslateService.t('TUIChat.对方正在输入');
  } else {
    currentConversationName.value = currentConversation.value?.getShowName() || '';
  }
}

const isPeerOnline = computed(() => (
  Boolean(
    peerUserID.value &&
    displayOnlineStatus.value &&
    isUserOnlineStatus(userOnlineStatusMap.value?.[peerUserID.value])
  )
));

const conversationOnlineStatusText = computed(() => {
  if (!peerUserID.value || !displayOnlineStatus.value) {
    return '';
  }
  return isPeerOnline.value ? '在线' : '离线';
});

const currentLoginDisplay = computed(() => (
  [currentLoginIP.value, currentLoginLocation.value].filter(Boolean).join(' ')
));

const visiblePeerLastOnlineLabel = computed(() => (
  isPeerOnline.value ? '' : props.peerLastOnlineLabel
));

const showConversationMeta = computed(() => (
  !props.isGroup && Boolean(
    conversationOnlineStatusText.value ||
    currentLoginDisplay.value ||
    visiblePeerLastOnlineLabel.value
  )
));

function getConversationPeerUserID(conversation: IConversationModel) {
  const userID = conversation?.userProfile?.userID || '';
  if (userID) {
    return userID;
  }
  const conversationID = conversation?.conversationID || '';
  const prefix = TUIChatEngine.TYPES.CONV_C2C || 'C2C';
  return conversationID.startsWith(prefix) ? conversationID.slice(prefix.length) : '';
}

function syncOnlineStatusFromStore() {
  const onlineStatusEnabled = TUIStore.getData(
    StoreName.USER,
    'displayOnlineStatus'
  );
  if (typeof onlineStatusEnabled === 'boolean') {
    displayOnlineStatus.value = onlineStatusEnabled;
  }

  const userStatusList = TUIStore.getData(StoreName.USER, 'userStatusList');
  onUserStatusListUpdated(userStatusList);
}

function onDisplayOnlineStatusUpdated(status: boolean) {
  displayOnlineStatus.value = status;
}

function onUserStatusListUpdated(list: unknown) {
  if (!(list instanceof Map)) {
    userOnlineStatusMap.value = {};
    return;
  }
  const nextMap: IUserStatusMap = {};
  list.forEach((status: IUserStatus, userID: string) => {
    nextMap[userID] = status;
  });
  userOnlineStatusMap.value = nextMap;
}

async function loadCurrentLoginIP(targetUserID: string) {
  const requestSerial = ++loginIPRequestSerial;
  currentLoginIP.value = '';
  currentLoginLocation.value = '';
  if (!targetUserID) {
    return;
  }

  try {
    const result = await getUserLoginIP(targetUserID);
    if (requestSerial !== loginIPRequestSerial || targetUserID !== peerUserID.value) {
      return;
    }
    currentLoginIP.value = result.current_login_ip || '';
    currentLoginLocation.value = result.current_login_location || '';
  } catch (error) {
    if (requestSerial === loginIPRequestSerial) {
      currentLoginIP.value = '';
      currentLoginLocation.value = '';
    }
    console.warn('get user login ip failed', error);
  }
}

</script>
<style lang="scss" scoped>
.chat-header {
  display: flex;
  min-width: 0;
  flex-direction: row;
  align-items: center;
  justify-content: space-between;

  &-container {
    display: flex;
    min-width: 0;
    flex: 1;
    flex-direction: column;
    justify-content: flex-start;
    overflow: hidden;
  }

  &-content {
    min-width: 0;
    margin-right: 20px;
    flex: 1;
    font-size: 16px;
    line-height: 22px;
    font-family: PingFangSC-Medium;
    font-weight: 500;
    color: #000;
    letter-spacing: 0;
    overflow: hidden;
    white-space: nowrap;
    text-overflow: ellipsis;
  }

  &-title {
    overflow: hidden;
    white-space: nowrap;
    text-overflow: ellipsis;
  }

  &-meta {
    display: flex;
    min-width: 0;
    align-items: center;
    gap: 4px;
    margin-top: 2px;
    color: #8a94a6;
    font-size: 12px;
    font-weight: 400;
    line-height: 18px;
    overflow: hidden;
    white-space: nowrap;
  }

  &-meta-status {
    flex-shrink: 0;
  }

  &-meta-status-online {
    color: #10b981;
  }

  &-meta-ip {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  &-meta-last-online {
    flex-shrink: 0;
  }

  &-content-clickable {
    cursor: pointer;
  }

  &-back,
  &-setting {
    width: 27px;
    height: 27px;

    .icon {
      width: 100%;
      height: 100%;
    }
  }

  &-profile-entry {
    padding: 0;
    border: none;
    background: transparent;
    cursor: pointer;
  }
}

.chat-header-h5 {
  &-back {
    display: flex;
    justify-content: center;
    align-items: center;
  }

  &-content{
    margin: 0 20px;
    text-align: center;
  }
}
</style>
