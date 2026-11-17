<template>
  <BottomPopup
    :show="showAtList"
    @onClose="closeAt"
  >
    <div
      ref="MessageInputAt"
      :class="[isPC ? 'message-input-at' : 'message-input-at-h5']"
    >
      <div
        ref="dialog"
        class="member-list"
      >
        <header
          v-if="!isPC"
          class="member-list-title"
        >
          <span class="title">{{
            TUITranslateService.t("TUIChat.选择提醒的人")
          }}</span>
        </header>
        <ul
          ref="memberListBox"
          class="member-list-box"
          @scroll="handleScroll"
        >
          <li
            v-for="(item, index) in showMemberList"
            :key="index"
            ref="memberListItems"
            class="member-list-box-body"
            :class="[index === selectedIndex && 'selected']"
            @click="selectItem(index)"
          >
            <img
              class="member-list-box-body-avatar"
              :src="handleMemberAvatar(item)"
            >
            <span class="member-list-box-body-name">
              {{ handleMemberName(item) }}
            </span>
          </li>
          <li
            v-if="isLoading"
            class="member-list-box-loading"
          >
            <span class="loading-text">{{ TUITranslateService.t("TUIChat.正在加载") }}</span>
          </li>
        </ul>
      </div>
    </div>
  </BottomPopup>
</template>
<script lang="ts" setup>
import TUIChatEngine, {
  TUIStore,
  StoreName,
  TUIGroupService,
  TUITranslateService,
} from '@tencentcloud/chat-uikit-engine-lite';
import { TUIGlobal } from '@tencentcloud/universal-api';
import { onMounted, onUnmounted, ref, watch } from '../../../../adapter-vue';
import { isPC, isH5 } from '../../../../utils/env';
import BottomPopup from '../../../common/BottomPopup/index.vue';
import { resolveGroupAtDisplayName } from './at-display-name';

const emits = defineEmits(['onAtListOpen', 'insertAt']);

const MessageInputAt = ref();
const memberListItems = ref();
const memberListBox = ref();

const showAtList = ref(false);
const memberList = ref<any[]>();
const allMemberList = ref<any[]>();
const showMemberList = ref<any[]>();
const isGroup = ref(false);
const position = ref({
  left: 0,
  top: 0,
});
const selectedIndex = ref(0);
const currentConversationID = ref('');
const currentGroupID = ref('');
const memberListGroupID = ref('');
const isLoading = ref(false);
const loadingGroupID = ref('');
const isCompleted = ref(false);

const all = {
  userID: TUIChatEngine.TYPES.MSG_AT_ALL,
  nick: '所有人',
  isAll: true,
  avatar: 'https://web.sdk.qcloud.com/im/assets/images/at.svg',
};

const conversationWatchOptions = {
  currentConversationID: (id: string) => {
    if (id !== currentConversationID.value) {
      currentConversationID.value = id;
      resetMemberList();
      isGroup.value = false;
      isCompleted.value = false;
      if (currentConversationID?.value?.startsWith('GROUP')) {
        isGroup.value = true;
        const groupID = currentConversationID?.value?.substring(5);
        currentGroupID.value = groupID;
        void ensureCurrentGroupMemberList(groupID);
      } else {
        currentGroupID.value = '';
        TUIGroupService.switchGroup('');
      }
    }
  },
};

const groupWatchOptions = {
  currentGroupMemberList: (list: any[]) => {
    const storeGroupID = TUIStore.getData(StoreName.GRP, 'currentGroupID') || '';
    if (!isGroup.value || !currentGroupID.value || storeGroupID !== currentGroupID.value) {
      return;
    }
    setMemberList(list, storeGroupID);
  },
  isCompleted: (completed: boolean) => {
    const storeGroupID = TUIStore.getData(StoreName.GRP, 'currentGroupID') || '';
    if (isGroup.value && currentGroupID.value && storeGroupID !== currentGroupID.value) {
      return;
    }
    isCompleted.value = completed;
    isLoading.value = false;
  },
};

onMounted(() => {
  const activeConversationID = TUIStore.getData(
    StoreName.CONV,
    'currentConversationID',
  );
  const currentGroupMemberList = TUIStore.getData(
    StoreName.GRP,
    'currentGroupMemberList',
  );
  const completed = TUIStore.getData(StoreName.GRP, 'isCompleted');
  if (activeConversationID) {
    conversationWatchOptions.currentConversationID(activeConversationID);
  }
  const storeGroupID = TUIStore.getData(StoreName.GRP, 'currentGroupID') || '';
  if (
    Array.isArray(currentGroupMemberList)
    && currentGroupID.value
    && storeGroupID === currentGroupID.value
  ) {
    setMemberList(currentGroupMemberList, storeGroupID);
  }
  if (typeof completed === 'boolean') {
    groupWatchOptions.isCompleted(completed);
  }
  TUIStore.watch(StoreName.CONV, conversationWatchOptions);
  TUIStore.watch(StoreName.GRP, groupWatchOptions);
});

onUnmounted(() => {
  TUIStore.unwatch(StoreName.CONV, conversationWatchOptions);
  TUIStore.unwatch(StoreName.GRP, groupWatchOptions);
});

// 获取正确的滚动容器
const getScrollElement = () => {
  return memberListBox.value;
};

const resetMemberList = () => {
  memberList.value = [];
  allMemberList.value = [];
  showMemberList.value = [];
  memberListGroupID.value = '';
  selectedIndex.value = 0;
  isLoading.value = false;
  loadingGroupID.value = '';
  TUIStore.update(StoreName.CUSTOM, 'memberList', memberList.value);
};

const setMemberList = (list: any[] = [], groupID = currentGroupID.value) => {
  memberList.value = Array.isArray(list) ? list : [];
  allMemberList.value = [all, ...memberList.value];
  showMemberList.value = allMemberList.value;
  memberListGroupID.value = groupID;
  selectedIndex.value = 0;
  isLoading.value = false;
  if (loadingGroupID.value === groupID) {
    loadingGroupID.value = '';
  }
  TUIStore.update(StoreName.CUSTOM, 'memberList', memberList.value);
};

const ensureCurrentGroupMemberList = async (groupID = currentGroupID.value) => {
  if (
    !groupID
    || (isLoading.value && loadingGroupID.value === groupID)
  ) {
    return;
  }

  const storeGroupID = TUIStore.getData(StoreName.GRP, 'currentGroupID') || '';
  const storeMemberList = TUIStore.getData(StoreName.GRP, 'currentGroupMemberList');
  if (
    storeGroupID === groupID
    && Array.isArray(storeMemberList)
    && storeMemberList.length > 0
  ) {
    setMemberList(storeMemberList, groupID);
    return;
  }

  try {
    isLoading.value = true;
    loadingGroupID.value = groupID;
    if (storeGroupID === groupID) {
      await TUIGroupService.getGroupMemberList({
        groupID,
        count: 50,
        offset: 0,
      });
    } else {
      await Promise.resolve(TUIGroupService.switchGroup(groupID) as any);
    }
  } catch (error) {
    if (loadingGroupID.value === groupID) {
      isLoading.value = false;
      loadingGroupID.value = '';
    }
  }
};

const handleScrollToLower = async () => {
  if (
    !isGroup.value
    || (isLoading.value && loadingGroupID.value === currentGroupID.value)
    || isCompleted.value
  ) {
    return;
  }
  await loadMoreMembers();
};

const handleScroll = async () => {
  if (
    !isGroup.value
    || (isLoading.value && loadingGroupID.value === currentGroupID.value)
    || isCompleted.value
  ) {
    return;
  }

  const scrollElement = getScrollElement();
  if (!scrollElement) {
    return;
  }

  const { scrollTop, scrollHeight, clientHeight } = scrollElement;
  // 检查是否滑动到底部（留有10px的缓冲区）
  if (scrollTop + clientHeight >= scrollHeight - 10) {
    await loadMoreMembers();
  }
};

const loadMoreMembers = async () => {
  if (
    (isLoading.value && loadingGroupID.value === currentGroupID.value)
    || isCompleted.value
    || !isGroup.value
  ) {
    return;
  }

  try {
    isLoading.value = true;
    const groupID = currentGroupID.value || currentConversationID.value?.substring(5);
    loadingGroupID.value = groupID || '';
    const storeGroupID = TUIStore.getData(StoreName.GRP, 'currentGroupID') || '';
    if (!groupID) {
      isLoading.value = false;
      loadingGroupID.value = '';
      return;
    }
    if (storeGroupID !== groupID) {
      await Promise.resolve(TUIGroupService.switchGroup(groupID) as any);
      return;
    }
    await TUIGroupService.getGroupMemberList({
      groupID,
      count: 50,
    });
    isLoading.value = false;
    if (loadingGroupID.value === groupID) {
      loadingGroupID.value = '';
    }
  } catch (error) {
    isLoading.value = false;
    loadingGroupID.value = '';
  }
};

const toggleAtList = (show: boolean) => {
  if (!isGroup.value) {
    return;
  }
  if (show && memberListGroupID.value !== currentGroupID.value) {
    resetMemberList();
    void ensureCurrentGroupMemberList();
  }
  showAtList.value = show;
  if (showAtList.value) {
    emits('onAtListOpen');
  }
};
const handleAtListPosition = (positionData: { top: number; left: number }) => {
  position.value = positionData;
};
const setCurrentSelectIndex = (index: any) => {
  selectedIndex.value = index;
  memberListItems.value?.[selectedIndex.value]?.scrollIntoView(false);
};
const setShowMemberList = (list: any) => {
  showMemberList.value = list;
};

TUIGlobal.toggleAtList = toggleAtList;
TUIGlobal.handleAtListPosition = handleAtListPosition;
TUIGlobal.setCurrentSelectIndex = setCurrentSelectIndex;
TUIGlobal.setShowMemberList = setShowMemberList;

onUnmounted(() => {
  if (TUIGlobal.toggleAtList === toggleAtList) {
    TUIGlobal.toggleAtList = undefined;
  }
  if (TUIGlobal.handleAtListPosition === handleAtListPosition) {
    TUIGlobal.handleAtListPosition = undefined;
  }
  if (TUIGlobal.setCurrentSelectIndex === setCurrentSelectIndex) {
    TUIGlobal.setCurrentSelectIndex = undefined;
  }
  if (TUIGlobal.setShowMemberList === setShowMemberList) {
    TUIGlobal.setShowMemberList = undefined;
  }
});

defineExpose({
  toggleAtList,
});

watch(
  () => [position.value, MessageInputAt?.value],
  () => {
    if (isH5 || !MessageInputAt?.value || !MessageInputAt?.value?.style) {
      return;
    }
    MessageInputAt.value.style.left = position.value.left + 'px';
    MessageInputAt.value.style.top
      = position.value.top - MessageInputAt.value.clientHeight + 'px';
  },
);

const closeAt = () => {
  showAtList.value = false;
  showMemberList.value = allMemberList.value;
  position.value = {
    left: 0,
    top: 0,
  };
};

const selectItem = (index: number) => {
  if (isPC && TUIGlobal.selectItem) {
    TUIGlobal.selectItem(index);
  } else {
    if (showMemberList?.value?.length) {
      const item = showMemberList?.value[index];
      emits('insertAt', {
        id: (item as any)?.userID,
        label: resolveGroupAtDisplayName(item) || (item as any)?.userID,
      });
    }
  }
  closeAt();
};

const handleMemberAvatar = (item: any) => {
  return (
    (item as any)?.avatar
    || 'https://web.sdk.qcloud.com/component/TUIKit/assets/avatar_21.png'
  );
};

const handleMemberName = (item: any) => {
  return resolveGroupAtDisplayName(item) || (item as any)?.userID;
};
</script>
<style scoped lang="scss">
@import "../../../../assets/styles/common";

.message-input-at {
  position: fixed;
  max-width: 15rem;
  max-height: 200px;
  overflow: hidden;
  background: #fff;
  box-shadow: 0 0.06rem 0.63rem 0 rgba(2,16,43,0.15);
  border-radius: 0.13rem;
}

.member-list-box {
  max-height: 200px;
  overflow-y: auto;

  &-header {
    height: 2.5rem;
    padding-top: 5px;
    cursor: pointer;

    &:hover {
      background: rgba(0,110,255,0.1);
    }
  }

  span {
    font-family: PingFangSC-Regular;
    font-weight: 400;
    font-size: 12px;
    color: #000;
    letter-spacing: 0;
    padding: 5px;
  }

  &-body {
    height: 30px;
    cursor: pointer;
    display: flex;
    align-items: center;

    .selected,
    &:hover {
      background: rgba(0,110,255,0.1);
    }

    &-name {
      overflow: hidden;
      white-space: nowrap;
      word-wrap: break-word;
      word-break: break-all;
      text-overflow: ellipsis;
    }

    &-avatar {
      width: 20px;
      height: 20px;
      padding-left: 10px;
    }
  }

  &-loading {
    height: 30px;
    display: flex;
    align-items: center;
    justify-content: center;
    color: #999;
    font-size: 12px;

    .loading-text {
      padding: 0;
    }
  }

  .selected {
    background: rgba(0,110,255,0.1);
  }
}

.message-input-at-h5 {
  .member-list {
    height: auto;
    max-height: 500px;
    width: 100%;
    max-width: 100%;
    background: white;
    border-radius: 12px 12px 0 0;
    display: flex;
    flex-direction: column;
    overflow: hidden;

    &-title {
      height: fit-content;
      width: calc(100% - 30px);
      text-align: center;
      vertical-align: middle;
      padding: 15px;

      .title {
        vertical-align: middle;
        display: inline-block;
        font-size: 16px;
      }

      .close {
        vertical-align: middle;
        position: absolute;
        right: 10px;
        display: inline-block;
      }
    }

    &-box {
      flex: 1;
      overflow-y: scroll;
      max-height: 200px;

      &-body {
        padding: 10px;

        img {
          width: 26px;
          height: 26px;
        }

        span {
          font-size: 14px;
        }
      }

      &-loading {
        padding: 10px;
        text-align: center;
        color: #999;
        font-size: 14px;

        .loading-text {
          padding: 0;
        }
      }
    }
  }
}
</style>
