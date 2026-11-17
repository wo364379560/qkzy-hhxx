<template>
  <div
    ref="searchRootRef"
    :class="['friend-search', !isPC && 'friend-search-h5']"
    @click.stop
  >
    <div class="friend-search-header">
      <div class="friend-search-input">
        <Icon
          class="friend-search-icon"
          :file="searchIcon"
          width="14px"
          height="14px"
        />
        <input
          v-model="searchValue"
          class="friend-search-input-main"
          type="text"
          placeholder="搜索好友/群聊"
          enterkeyhint="search"
          @focus="openSearch"
          @click="openSearch"
        >
        <button
          v-if="isSearching || searchValue"
          class="friend-search-clear"
          type="button"
          @click="clearSearch"
        >
          <Icon
            :file="closeIcon"
            width="14px"
            height="14px"
          />
        </button>
      </div>
      <button
        v-if="!isPC && isSearching"
        class="friend-search-cancel"
        type="button"
        @click="closeSearch"
      >
        取消
      </button>
      <SearchMore
        v-else
        class="friend-search-more"
        searchType="conversation"
      />
    </div>
    <div
      v-if="isSearching"
      class="friend-search-panel"
    >
      <template v-if="hasSearchResult">
        <div
          v-if="filteredFriendList.length"
          class="friend-search-section-title"
        >
          好友
        </div>
        <button
          v-for="friend in filteredFriendList"
          :key="`friend-${friend.userID}`"
          class="friend-search-result"
          type="button"
          @click="handleFriendClick(friend)"
        >
          <Avatar
            :url="generateAvatar(friend)"
            size="36px"
            borderRadius="50%"
          />
          <span class="friend-search-result-info">
            <span class="friend-search-result-name">
              {{ getDisplayName(friend) }}
            </span>
            <span
              v-if="friend.userID && friend.userID !== getDisplayName(friend)"
              class="friend-search-result-id"
            >
              ID: {{ friend.userID }}
            </span>
          </span>
        </button>
        <div
          v-if="filteredGroupList.length"
          class="friend-search-section-title"
        >
          群聊
        </div>
        <button
          v-for="group in filteredGroupList"
          :key="`group-${group.groupID}`"
          class="friend-search-result"
          type="button"
          @click="handleGroupClick(group)"
        >
          <Avatar
            :url="generateAvatar(group)"
            size="36px"
            borderRadius="50%"
          />
          <span class="friend-search-result-info">
            <span class="friend-search-result-name">
              {{ getGroupDisplayName(group) }}
            </span>
            <span
              v-if="group.groupID"
              class="friend-search-result-id"
            >
              ID: {{ group.groupID }}
            </span>
          </span>
        </button>
      </template>
      <div
        v-else
        class="friend-search-empty"
      >
        {{ emptyText }}
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import {
  computed,
  onMounted,
  onUnmounted,
  ref,
} from "../TUIKit/adapter-vue";
import {
  StoreName,
  TUIFriendService,
  TUIStore,
} from "@tencentcloud/chat-uikit-engine-lite";
import type { IGroupModel } from "@tencentcloud/chat-uikit-engine-lite";
import { outsideClick } from "@tencentcloud/universal-api";
import Icon from "../TUIKit/components/common/Icon.vue";
import Avatar from "../TUIKit/components/common/Avatar/index.vue";
import SearchMore from "../TUIKit/components/TUISearch/search-more/index.vue";
import searchIcon from "../TUIKit/assets/icon/search.svg";
import closeIcon from "../TUIKit/assets/icon/input-close.svg";
import { isPC } from "../TUIKit/utils/env";
import {
  enterConversation,
  generateAvatar,
  generateName,
} from "../TUIKit/components/TUIContact/utils";
import {
  matchesMemberKeyword,
  mergeFriendProfileList,
  resolveUserDisplayName,
} from "../TUIKit/utils/member-search";

const searchRootRef = ref<HTMLElement | null>();
const searchValue = ref("");
const isSearching = ref(false);
const friendList = ref<any[]>([]);
const groupList = ref<IGroupModel[]>([]);

const getStoredFriendList = () => {
  const storedFriendList = TUIStore.getData(StoreName.FRIEND, "friendList");
  return Array.isArray(storedFriendList) ? storedFriendList : [];
};

const getStoredGroupList = () => {
  const storedGroupList = TUIStore.getData(StoreName.GRP, "groupList");
  return Array.isArray(storedGroupList) ? storedGroupList : [];
};

const applyFriendList = (sourceList: any[] = []) => {
  friendList.value = mergeFriendProfileList(sourceList, getStoredFriendList());
};

const applyGroupList = (sourceList: IGroupModel[] = []) => {
  groupList.value = sourceList.length ? sourceList : getStoredGroupList();
};

const handleFriendListUpdated = (list: any[]) => {
  friendList.value = mergeFriendProfileList(friendList.value, list || []);
};

const handleGroupListUpdated = (list: IGroupModel[]) => {
  groupList.value = list || [];
};

const matchesGroupKeyword = (group: IGroupModel, keyword: string) => {
  const normalizedKeyword = keyword.trim().toLowerCase();
  if (!normalizedKeyword) {
    return true;
  }

  return [
    group?.name,
    group?.groupID,
    group?.introduction,
    group?.notification,
  ]
    .filter(Boolean)
    .some((value) =>
      String(value).toLowerCase().includes(normalizedKeyword)
    );
};

const filteredFriendList = computed(() => {
  const keyword = searchValue.value.trim();
  if (!keyword) {
    return friendList.value;
  }

  return friendList.value.filter((item: any) =>
    matchesMemberKeyword(item, keyword)
  );
});

const filteredGroupList = computed(() => {
  const keyword = searchValue.value.trim();
  if (!keyword) {
    return groupList.value;
  }

  return groupList.value.filter((item: IGroupModel) =>
    matchesGroupKeyword(item, keyword)
  );
});

const hasSearchResult = computed(() =>
  filteredFriendList.value.length > 0 || filteredGroupList.value.length > 0
);

const emptyText = computed(() =>
  friendList.value.length || groupList.value.length
    ? "暂无搜索结果"
    : "暂无好友或群聊"
);

const openSearch = () => {
  if (isSearching.value) {
    return;
  }

  isSearching.value = true;
  if (isPC && searchRootRef.value) {
    outsideClick.listen({
      domRefs: searchRootRef.value,
      handler: closeSearch,
    });
  }
};

const closeSearch = () => {
  isSearching.value = false;
};

const clearSearch = () => {
  searchValue.value = "";
  closeSearch();
};

const getDisplayName = (friend: any) =>
  resolveUserDisplayName(friend) || generateName(friend) || friend?.userID || "";

const getGroupDisplayName = (group: IGroupModel) =>
  generateName(group) || group?.groupID || "";

const handleFriendClick = (friend: any) => {
  if (!friend?.userID) {
    return;
  }

  TUIStore.update(StoreName.CUSTOM, "currentHomeNavBar", "message");
  enterConversation({ userID: friend.userID });
  searchValue.value = "";
  closeSearch();
};

const handleGroupClick = (group: IGroupModel) => {
  if (!group?.groupID) {
    return;
  }

  TUIStore.update(StoreName.CUSTOM, "currentHomeNavBar", "message");
  enterConversation({ groupID: group.groupID });
  searchValue.value = "";
  closeSearch();
};

onMounted(() => {
  applyFriendList();
  applyGroupList();

  TUIStore.watch(StoreName.FRIEND, {
    friendList: handleFriendListUpdated,
  });
  TUIStore.watch(StoreName.GRP, {
    groupList: handleGroupListUpdated,
  });

  TUIFriendService.getFriendList()
    .then((res: any) => {
      applyFriendList(Array.isArray(res?.data) ? res.data : []);
    })
    .catch((error: any) => {
      console.warn("getFriendList error:", error);
    });
});

onUnmounted(() => {
  TUIStore.unwatch(StoreName.FRIEND, {
    friendList: handleFriendListUpdated,
  });
  TUIStore.unwatch(StoreName.GRP, {
    groupList: handleGroupListUpdated,
  });
});
</script>

<style scoped lang="scss">
.friend-search {
  position: relative;
  z-index: 20;
  width: 100%;
  background: #fff;
  box-sizing: border-box;

  * {
    box-sizing: border-box;
  }

  &-header {
    display: flex;
    align-items: center;
    width: 100%;
    min-height: 48px;
    padding: 10px;
    gap: 10px;
  }

  &-input {
    flex: 1;
    display: flex;
    align-items: center;
    min-width: 0;
    height: 36px;
    background: #ededed;
    border-radius: 6px;
  }

  &-icon {
    flex: 0 0 auto;
    margin: 0 7px;
  }

  &-input-main {
    flex: 1;
    min-width: 0;
    border: none;
    outline: none;
    background: transparent;
    caret-color: #007aff;
    color: #111;
    font-size: 14px;

    &::placeholder {
      color: #666;
      font-size: 13px;
    }
  }

  &-clear,
  &-cancel,
  &-result {
    border: none;
    background: transparent;
    padding: 0;
    font: inherit;
    color: inherit;
    cursor: pointer;
  }

  &-clear {
    flex: 0 0 auto;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 28px;
    height: 28px;
  }

  &-cancel {
    flex: 0 0 auto;
    padding: 7px 0 7px 3px;
    color: #007aff;
    font-size: 16px;
  }

  &-more {
    flex: 0 0 auto;

    :deep(.more) {
      display: flex;
      align-items: center;
      justify-content: center;
      width: 36px;
      height: 36px;
      margin-right: 0;
      border-radius: 6px;
      background: #f4f6fb;
      cursor: pointer;
    }
  }

  &-panel {
    position: absolute;
    top: 58px;
    left: 10px;
    right: 10px;
    z-index: 100;
    max-height: min(520px, calc(100vh - 180px));
    overflow-y: auto;
    padding: 6px 0;
    background: #fff;
    border: 1px solid #eee;
    border-radius: 6px;
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.14);
  }

  &-result {
    display: flex;
    align-items: center;
    width: 100%;
    min-height: 62px;
    padding: 8px 14px;
    text-align: left;

    &:hover {
      background: #f5f7fb;
    }
  }

  &-section-title {
    padding: 8px 14px 4px;
    color: #999;
    font-size: 13px;
    line-height: 18px;
  }

  &-result-info {
    display: flex;
    flex: 1;
    flex-direction: column;
    min-width: 0;
    margin-left: 10px;
  }

  &-result-name,
  &-result-id {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  &-result-name {
    color: #111;
    font-size: 15px;
    line-height: 22px;
  }

  &-result-id {
    margin-top: 2px;
    color: #999;
    font-size: 12px;
    line-height: 18px;
  }

  &-empty {
    padding: 24px 12px;
    color: #999;
    font-size: 13px;
    line-height: 20px;
    text-align: center;
  }
}

.friend-search-h5 {
  .friend-search-header {
    min-height: 54px;
  }

  .friend-search-input {
    height: 34px;
  }

  .friend-search-panel {
    top: 54px;
    max-height: calc(100vh - 160px);
  }
}
</style>
