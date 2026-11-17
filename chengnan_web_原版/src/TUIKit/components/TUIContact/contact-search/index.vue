<template>
  <div :class="['tui-contact-search', !isPC && 'tui-contact-search-h5']">
    <div
      v-if="!isSearching || !isPC"
      :class="[
        'tui-contact-search-header',
        !isPC && 'tui-contact-search-h5-header',
        isSearching && 'tui-contact-searching-h5-header',
      ]"
      @click="onChangeContactSearchingStatus(true)"
    >
      <div
        :class="[
          'tui-contact-search-header-icon',
          !isPC && 'tui-contact-search-h5-header-icon',
        ]"
        @click.stop="onChangeContactSearchingStatus(!isSearching)"
      >
        <Icon
          :file="isSearching ? backSVG : addSVG"
          :width="isSearching ? '20px' : '14px'"
          :height="isSearching ? '20px' : '14px'"
        />
      </div>

      <div
        :class="[
          'tui-contact-search-header-title',
          !isPC && 'tui-contact-search-h5-header-title',
        ]"
      >
        {{ TUITranslateService.t("TUIContact.添加好友/群聊") }}
      </div>
    </div>
    <div
      v-if="isSearching"
      :class="[
        'tui-contact-search-main',
        !isPC && 'tui-contact-search-h5-main',
      ]"
    >
      <input
        v-model="searchValue"
        class="tui-contact-search-main-input"
        type="text"
        :placeholder="searchingPlaceholder"
        enterkeyhint="search"
        @keyup.enter="search"
        @blur="search"
        @confirm="search"
      />
      <div class="tui-contact-search-main-cancel" @click="isSearching = false">
        {{ TUITranslateService.t("取消") }}
      </div>
    </div>
  </div>
</template>
<script setup lang="ts">
import { ref, watch } from "../../../adapter-vue";
import {
  TUITranslateService,
  TUIStore,
  StoreName,
} from "@tencentcloud/chat-uikit-engine-lite";
import { TUIGlobal } from "@tencentcloud/universal-api";
import { searchGroupByID, searchUsersExact } from "@/api";
import Icon from "../../common/Icon.vue";
import addSVG from "../../../assets/icon/add.svg";
import backSVG from "../../../assets/icon/back.svg";
import { isPC } from "../../../utils/env";
import { IContactSearchResult } from "../../../interface";

const searchingPlaceholder = "手机号、信号或群ID";
const isSearching = ref<boolean>(false);
const searchValue = ref<string>("");
const searchResult = ref<IContactSearchResult>({
  user: {
    label: "联系人",
    list: [],
  },
  group: {
    label: "群聊",
    list: [],
  },
});
const isSearchRequestPending = ref(false);
let currentSearchRequestID = 0;
let currentSearchingKeyword = "";
let lastCompletedKeyword = "";

const onChangeContactSearchingStatus = (status: boolean) => {
  isSearching.value = status;
};

const syncSearchResultToStore = () => {
  TUIStore.update(
    StoreName.CUSTOM,
    "currentContactSearchResult",
    searchResult.value
  );
};

const resetSearchResult = () => {
  searchResult.value = {
    user: {
      label: "联系人",
      list: [],
    },
    group: {
      label: "群聊",
      list: [],
    },
  };
  syncSearchResultToStore();
};
syncSearchResultToStore();

const search = async () => {
  const keyword = searchValue.value.trim();
  if (!keyword) {
    return;
  }
  if (
    isSearchRequestPending.value &&
    currentSearchingKeyword === keyword
  ) {
    return;
  }
  if (lastCompletedKeyword === keyword) {
    syncSearchResultToStore();
    return;
  }

  currentSearchRequestID += 1;
  const requestID = currentSearchRequestID;
  currentSearchingKeyword = keyword;
  isSearchRequestPending.value = true;

  const [userResponse, groupResponse] = await Promise.all([
    searchUsersExact(keyword)
      .then((users) => ({
        data: users.map((item) => ({
          userID: item.im_user_id,
          nick: item.nickname,
          avatar: item.avatar_url || "",
          _wananID: item.wanan_id || "",
          _phone: item.phone || "",
          _canAddFriend: item.can_add_friend !== false,
          _addBlockedReason: item.add_blocked_reason || "",
        })),
        error: null,
      }))
      .catch((error: any) => ({ data: [], error })),
    searchGroupByID(keyword)
      .then((group) => ({
        data: group
          ? [
              {
                groupID: group.group_id,
                name: group.group_name,
                avatar: group.avatar_url || "",
                type: group.group_type || "",
                isJoinedGroup: false,
              },
            ]
          : [],
        error: null,
      }))
      .catch((error: any) => ({ data: [], error })),
  ]);

  if (requestID !== currentSearchRequestID) {
    return;
  }

  searchResult.value = {
    user: {
      label: "联系人",
      list: userResponse.data,
    },
    group: {
      label: "群聊",
      list: groupResponse.data,
    },
  };
  syncSearchResultToStore();
  lastCompletedKeyword = keyword;
  isSearchRequestPending.value = false;

  if (userResponse.error) {
    console.warn("search user error", userResponse.error);
  }
  if (groupResponse.error) {
    console.warn("search group error", groupResponse.error);
  }
};

watch(
  () => isSearching.value,
  () => {
    TUIStore.update(
      StoreName.CUSTOM,
      "currentContactSearchingStatus",
      isSearching.value
    );
    if (isSearching.value) {
      searchValue.value = "";
      lastCompletedKeyword = "";
      currentSearchingKeyword = "";
      currentSearchRequestID += 1;
      isSearchRequestPending.value = false;
      resetSearchResult();
    }
  },
  {
    immediate: true,
  }
);

TUIGlobal.updateContactSearch = search;
TUIGlobal.closeSearching = () => {
  isSearching.value = false;
};
</script>
<style lang="scss" scoped>
.tui-contact-search {
  position: sticky;
  top: 0;
  z-index: 1;
  padding: 12px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: #fff;
  border-bottom: 1px solid #f4f5f9;
  flex-direction: column;

  &-header,
  &-main {
    width: 100%;
    height: 30px;
    display: flex;
    flex-direction: row;
    align-items: center;
  }

  &-header {
    user-select: none;
    cursor: pointer;

    &-icon {
      padding-right: 10px;
    }

    &-title {
      font-size: 14px;
    }
  }

  &-main {
    display: flex;
    flex-direction: row;
    flex: 1;
    justify-content: center;
    align-items: center;
    width: 100%;

    &-input {
      flex: 1;
      font-size: 14px;
      border-radius: 5px;
      padding: 7px;
      border: 1px solid #ddd;
    }

    &-input:focus {
      outline: none;
      border: 1px solid #006eff;
    }

    &-cancel {
      padding-left: 10px;
      user-select: none;
      cursor: pointer;
    }
  }

  &-h5 {
    &-header {
      width: 100%;
    }
  }
}

.tui-contact-searching-h5-header {
  padding-bottom: 10px;
  display: flex;
  flex-direction: row;

  .tui-contact-search-h5-header-title {
    flex: 1;
    text-align: center;
    font-weight: 500;
    font-size: 14px;
    margin-right: 30px;
  }
}
</style>
