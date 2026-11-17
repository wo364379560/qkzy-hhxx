import TUIChatEngine, {
  TUIStore,
  StoreName,
} from '@tencentcloud/chat-uikit-engine-lite';
import { onMounted, onUnmounted, ref } from '../../../adapter-vue';
import { removeGroupMessagePin, type GroupMessagePin } from '@/api';
import {
  createRefreshCoordinator,
  type RefreshRequestOptions,
} from './refresh-coordinator';

const PASSIVE_PIN_REFRESH_INTERVAL_MS = 5 * 60 * 1000;

function buildMessageAbstract(message: any) {
  const types = TUIChatEngine.TYPES;
  switch (message?.type) {
    case types.MSG_TEXT:
      return message.payload?.text || '';
    case types.MSG_IMAGE:
      return '[图片]';
    case types.MSG_VIDEO:
      return '[视频]';
    case types.MSG_AUDIO:
      return '[语音]';
    case types.MSG_FILE:
      return `[文件] ${message.payload?.fileName || ''}`.trim();
    case types.MSG_FACE:
      return '[表情]';
    case types.MSG_LOCATION:
      return '[位置]';
    case types.MSG_CUSTOM:
      return '[自定义消息]';
    default:
      return '[消息]';
  }
}

function toGroupMessagePin(groupID: string, message: any): GroupMessagePin {
  return {
    group_id: 0,
    im_group_id: groupID,
    message_id: String(message?.ID || ''),
    message_seq: Number(message?.sequence || 0),
    message_sender_im_user_id: String(message?.from || ''),
    message_sender_name: String(
      message?.nameCard || message?.nick || message?.from || '',
    ),
    message_type: String(message?.type || ''),
    message_abstract: buildMessageAbstract(message),
    status: 'active',
    pinned_at: '',
  };
}

export function useGroupPinnedMessage() {
  const currentGroupPinnedMessage = ref<GroupMessagePin | null>(null);
  const currentGroupPinnedMessages = ref<GroupMessagePin[]>([]);
  const currentPinnedIndex = ref(0);
  const nativePinnedMessageMap = new Map<string, any>();
  const pinRefreshCoordinator = createRefreshCoordinator({
    minIntervalMs: PASSIVE_PIN_REFRESH_INTERVAL_MS,
  });
  let activeGroupID = '';

  function extractPinnedMessageList(response: any): any[] {
    const data = response?.data;
    if (Array.isArray(data)) return data;
    if (Array.isArray(data?.messageList)) return data.messageList;
    if (Array.isArray(data?.pinnedMessageList)) return data.pinnedMessageList;
    return [];
  }

  function syncDisplayedPinnedMessage(pins: GroupMessagePin[]) {
    const currentId = currentGroupPinnedMessage.value?.message_id;
    const keptIndex = pins.findIndex(pin => pin.message_id === currentId);
    currentPinnedIndex.value =
      keptIndex >= 0 ? keptIndex : Math.max(pins.length - 1, 0);
    const displayedPin = pins[currentPinnedIndex.value] || null;
    currentGroupPinnedMessages.value = pins;
    currentGroupPinnedMessage.value = displayedPin;
    TUIStore.update(StoreName.CUSTOM, 'currentGroupPinnedMessage', displayedPin);
    TUIStore.update(StoreName.CUSTOM, 'currentGroupPinnedMessages', pins);
    TUIStore.update(
      StoreName.CUSTOM,
      'currentGroupPinnedMessageIDs',
      pins.map(pin => pin.message_id),
    );
    return displayedPin;
  }

  function updatePinnedMessageStore(groupID: string, messageList: any[]) {
    nativePinnedMessageMap.clear();
    const pins = (messageList || [])
      .map(message => {
        const pin = toGroupMessagePin(groupID, message);
        if (pin.message_id) {
          nativePinnedMessageMap.set(pin.message_id, message);
        }
        return pin;
      })
      .filter(pin => Boolean(pin.message_id));
    syncDisplayedPinnedMessage(pins);
  }

  async function loadCurrentGroupPinnedMessage(
    isGroup: boolean,
    groupID?: string,
    options: RefreshRequestOptions = {},
  ) {
    const nextGroupID = isGroup && groupID ? groupID : '';
    const groupChanged = nextGroupID !== activeGroupID;
    activeGroupID = nextGroupID;
    if (!activeGroupID) {
      updatePinnedMessageStore('', []);
      return;
    }

    const requestedGroupID = activeGroupID;
    await pinRefreshCoordinator.run(
      requestedGroupID,
      async () => {
        try {
          const response = await TUIChatEngine.chat.getPinnedGroupMessageList(
            requestedGroupID,
          );
          if (activeGroupID !== requestedGroupID) return;
          const nativeMessageList = extractPinnedMessageList(response);
          if (nativeMessageList.length > 0) {
            updatePinnedMessageStore(requestedGroupID, nativeMessageList);
            return;
          }

          updatePinnedMessageStore(requestedGroupID, []);
        } catch (error) {
          if (activeGroupID !== requestedGroupID) return;
          console.warn('load Tencent group pinned messages failed', error);
          updatePinnedMessageStore(requestedGroupID, []);
        }
      },
      {
        force: options.force || groupChanged,
        ensureAfterInflight: options.ensureAfterInflight,
      },
    );
  }

  function onCurrentGroupPinnedMessageUpdate(pin: GroupMessagePin | null) {
    if (!pin?.message_id) {
      return;
    }
    const pins = currentGroupPinnedMessages.value;
    const index = pins.findIndex(item => item.message_id === pin.message_id);
    if (index >= 0) {
      currentPinnedIndex.value = index;
    }
    currentGroupPinnedMessage.value = pin;
  }

  function cyclePinnedMessage() {
    const pins = currentGroupPinnedMessages.value;
    if (pins.length === 0) {
      return null;
    }
    currentPinnedIndex.value = (currentPinnedIndex.value + 1) % pins.length;
    const displayed = pins[currentPinnedIndex.value];
    currentGroupPinnedMessage.value = displayed;
    TUIStore.update(StoreName.CUSTOM, 'currentGroupPinnedMessage', displayed);
    return displayed;
  }

  function jumpToPinnedMessage(pin: GroupMessagePin) {
    const pins = currentGroupPinnedMessages.value;
    const index = pins.findIndex(item => item.message_id === pin?.message_id);
    if (index < 0) {
      return pin;
    }
    currentPinnedIndex.value = index;
    currentGroupPinnedMessage.value = pins[index];
    TUIStore.update(StoreName.CUSTOM, 'currentGroupPinnedMessage', pins[index]);
    return pins[index];
  }

  async function unpinGroupPinnedMessage(pin: GroupMessagePin) {
    if (!activeGroupID || !pin?.message_id) {
      return false;
    }
    const rawMessage =
      nativePinnedMessageMap.get(pin.message_id) ||
      TUIStore.getMessageModel(pin.message_id)?.getMessage?.();
    if (rawMessage) {
      await TUIChatEngine.chat.pinGroupMessage({
        groupID: activeGroupID,
        message: rawMessage,
        isPinned: false,
      });
    } else {
      await removeGroupMessagePin(activeGroupID);
    }
    await loadCurrentGroupPinnedMessage(true, activeGroupID, {
      force: true,
      ensureAfterInflight: true,
    });
    return true;
  }

  function onPinnedGroupMessageUpdated(event: any) {
    const data = event?.data || event;
    if (!activeGroupID) {
      return;
    }
    const eventGroupID = String(
      data?.groupID ||
        data?.groupId ||
        data?.message?.to ||
        data?.messageList?.[0]?.to ||
        '',
    );
    // 不同 Web SDK 版本的事件包装层级不一致。事件不带 groupID 时
    // 也刷新当前群，避免 App 置顶后 Web 端收到事件却被过滤。
    if (eventGroupID && eventGroupID !== activeGroupID) return;
    void loadCurrentGroupPinnedMessage(true, activeGroupID, {
      force: true,
      ensureAfterInflight: true,
    });
  }

  function refreshActiveGroupPins() {
    if (activeGroupID) {
      void loadCurrentGroupPinnedMessage(true, activeGroupID);
    }
  }

  function onVisibilityChange() {
    if (typeof document === 'undefined' || document.visibilityState === 'visible') {
      refreshActiveGroupPins();
    }
  }

  onMounted(() => {
    TUIChatEngine.chat?.on?.(
      TUIChatEngine.EVENT.PINNED_GROUP_MESSAGE_UPDATED,
      onPinnedGroupMessageUpdated,
    );
    if (typeof window !== 'undefined') {
      window.addEventListener('focus', refreshActiveGroupPins);
      document.addEventListener('visibilitychange', onVisibilityChange);
      // SDK 事件负责实时同步；页面切换和聚焦只做低频保底刷新。
    }
  });

  onUnmounted(() => {
    TUIChatEngine.chat?.off?.(
      TUIChatEngine.EVENT.PINNED_GROUP_MESSAGE_UPDATED,
      onPinnedGroupMessageUpdated,
    );
    if (typeof window !== 'undefined') {
      window.removeEventListener('focus', refreshActiveGroupPins);
      document.removeEventListener('visibilitychange', onVisibilityChange);
    }
    pinRefreshCoordinator.clear();
  });

  return {
    currentGroupPinnedMessage,
    currentGroupPinnedMessages,
    currentPinnedIndex,
    loadCurrentGroupPinnedMessage,
    onCurrentGroupPinnedMessageUpdate,
    cyclePinnedMessage,
    jumpToPinnedMessage,
    unpinGroupPinnedMessage,
  };
}
