<template>
  <main v-if="!isUniFrameWork" class="member">
    <div class="member-search">
      <div class="member-search-box">
        <input
          v-model.trim="searchKeyword"
          class="member-search-input"
          type="text"
          placeholder="请输入关键词"
          @keyup.enter="search"
          @input="handleSearchInput"
        />
        <button
          class="member-search-btn"
          type="button"
          :disabled="isMemberSearchLoading"
          @click="search"
        >
          <span
            v-if="isMemberSearchLoading"
            class="member-action-spinner member-search-spinner"
          />
          <span v-else>搜索</span>
        </button>
      </div>
    </div>
    <div v-if="isShowAddBtn" class="member-action-bar">
      <button
        class="member-action-btn"
        type="button"
        :disabled="isActionLoading"
        @click="add"
      >
        <span
          v-if="isAddOpening"
          class="member-action-spinner"
        />
        <Icon
          v-else
          :file="addCircleIcon"
          width="14px"
          height="14px"
          :hotAreaSize="0"
        />
        {{
          isAddOpening
            ? "加载中..."
            : TUITranslateService.t(`TUIGroup.添加成员`)
        }}
      </button>
    </div>
    <ul class="list">
      <li
        v-for="(item, index) in filteredMemberList"
        :key="item.userID || index"
        class="list-item"
      >
        <aside class="aside" @click="handleMemberProfileShow(item)">
          <div class="avatar-wrapper">
            <img
              class="avatar"
              :src="
                item.avatar ||
                'https://web.sdk.qcloud.com/component/TUIKit/assets/avatar_21.png'
              "
              onerror="this.onerror=null;this.src='https://web.sdk.qcloud.com/component/TUIKit/assets/avatar_21.png'"
            />
            <span
              v-if="displayOnlineStatus && item.userID"
              :class="[
                'online-status',
                isOnline(item) ? 'online-status-online' : 'online-status-offline',
              ]"
            />
          </div>
          <span class="name">{{ item.remark || item.friendRemark || item.nick || item.userID }}</span>
          <span v-if="resolveUserTag(item)" class="member-tag">
            {{ resolveUserTag(item) }}
          </span>
          <span>{{ handleRoleName(item) }}</span>
        </aside>
        <div
          :class="['member-delete-action', isActionLoading && 'member-delete-action-disabled']"
          @click="submit(item)"
        >
          <Icon
            v-if="canDeleteMember(item)"
            :file="delIcon"
            :width="'16px'"
            :height="'16px'"
          />
        </div>
      </li>
      <li
        v-if="hasMore && !isMemberSearchLoading"
        class="list-item"
        @click="getMore"
      >
        {{ TUITranslateService.t(`TUIGroup.查看更多`) }}
      </li>
      <li v-if="isMemberSearchLoading" class="list-item list-item-empty">
        正在搜索中...
      </li>
      <li
        v-else-if="!filteredMemberList.length"
        class="list-item list-item-empty"
      >
        {{ TUITranslateService.t(`TUIContact.无搜索结果`) }}
      </li>
    </ul>
  </main>
  <div v-else class="edit-h5">
    <main class="main">
      <header class="edit-h5-header">
        <aside class="left">
          <h1>{{ TUITranslateService.t(`TUIGroup.群成员`) }}</h1>
        </aside>
        <button
          v-if="isShowAddBtn"
          class="member-action-btn member-action-btn-h5"
          type="button"
          :disabled="isActionLoading"
          @click="add"
        >
          <span
            v-if="isAddOpening"
            class="member-action-spinner"
          />
          <Icon
            v-else
            :file="addCircleIcon"
            width="14px"
            height="14px"
            :hotAreaSize="0"
          />
          {{
            isAddOpening
              ? "加载中..."
              : TUITranslateService.t(`TUIGroup.添加成员`)
          }}
        </button>
        <span class="close" @click="close('member')">{{
          TUITranslateService.t(`关闭`)
        }}</span>
      </header>
      <div class="member">
        <div class="member-search member-search-h5">
          <div class="member-search-box">
            <input
              v-model.trim="searchKeyword"
              class="member-search-input"
              type="text"
              placeholder="请输入关键词"
              @keyup.enter="search"
              @input="handleSearchInput"
            />
            <button
              class="member-search-btn"
              type="button"
              :disabled="isMemberSearchLoading"
              @click="search"
            >
              <span
                v-if="isMemberSearchLoading"
                class="member-action-spinner member-search-spinner"
              />
              <span v-else>搜索</span>
            </button>
          </div>
        </div>
        <ul class="list list-uniapp">
          <li
            v-for="(item, index) in filteredMemberList"
            :key="item.userID || index"
            class="list-item"
          >
            <aside class="aside" @click="handleMemberProfileShow(item)">
              <div class="avatar-wrapper">
                <img
                  class="avatar"
                  :src="
                    item.avatar ||
                    'https://web.sdk.qcloud.com/component/TUIKit/assets/avatar_21.png'
                  "
                  onerror="this.onerror=null;this.src='https://web.sdk.qcloud.com/component/TUIKit/assets/avatar_21.png'"
                />
                <span
                  v-if="displayOnlineStatus && item.userID"
                  :class="[
                    'online-status',
                    isOnline(item) ? 'online-status-online' : 'online-status-offline',
                  ]"
                />
              </div>
              <span class="name">{{ item.remark || item.friendRemark || item.nick || item.userID }}</span>
              <span v-if="resolveUserTag(item)" class="member-tag">
                {{ resolveUserTag(item) }}
              </span>
              <span>{{ handleRoleName(item) }}</span>
            </aside>
            <div
              :class="['member-delete-action', isActionLoading && 'member-delete-action-disabled']"
              @click="submit(item)"
            >
              <Icon
                v-if="canDeleteMember(item)"
                :file="delIcon"
                :width="'16px'"
                :height="'16px'"
              />
            </div>
          </li>
          <li
            v-if="hasMore && !isMemberSearchLoading"
            class="list-item"
            @click="getMore"
          >
            {{ TUITranslateService.t(`TUIGroup.查看更多`) }}
          </li>
          <li
            v-if="isMemberSearchLoading"
            class="list-item list-item-empty"
          >
            正在搜索中...
          </li>
          <li
            v-else-if="!filteredMemberList.length"
            class="list-item list-item-empty"
          >
            {{ TUITranslateService.t(`TUIContact.无搜索结果`) }}
          </li>
        </ul>
      </div>
    </main>
  </div>
</template>

<script lang="ts" setup>
// @ts-nocheck
import TUIChatEngine, {
  TUITranslateService,
} from "@tencentcloud/chat-uikit-engine-lite";
import { watchEffect, ref, computed } from "../../../adapter-vue";
import Icon from "../../common/Icon.vue";
import delIcon from "../../../assets/icon/del-icon.svg";
import addCircleIcon from "../../../assets/icon/add-circle.svg";
import searchIcon from "../../../assets/icon/search.svg";
import { IGroupSelfInfo, IGroupMember, IUserStatusMap } from "../../../interface";
import { isUniFrameWork } from "../../../utils/env";
import { matchesMemberKeyword } from "../../../utils/member-search";
import { ensureUserTags, getUserTag } from "../../../utils/user-tag";
import { isUserOnlineStatus } from "@/features/presence/invisible";

const props = defineProps({
  list: {
    type: Array,
    default: () => [],
  },
  total: {
    type: Number,
    default: () => 0,
  },
  hasMore: {
    type: Boolean,
    default: false,
  },
  isShowDel: {
    type: Boolean,
    default: () => false,
  },
  isShowAdd: {
    type: Boolean,
    default: () => false,
  },
  self: {
    type: Object,
    default: () => ({}),
  },
  displayOnlineStatus: {
    type: Boolean,
    default: () => false,
  },
  userOnlineStatusMap: {
    type: Object,
    default: () => ({}),
  },
  actionLoading: {
    type: Boolean,
    default: () => false,
  },
  actionLoadingType: {
    type: String,
    default: () => "",
  },
  memberSearchLoading: {
    type: Boolean,
    default: () => false,
  },
});

const totalMember = ref(0);
const hasMore = ref(false);
const memberList = ref<Array<IGroupMember>>([]);
const isShowDeleteBtn = ref(false);
const isShowAddBtn = ref(false);
const selfValue = ref<IGroupSelfInfo>({});
const searchKeyword = ref("");
const submittedSearchKeyword = ref("");
const isActionLoading = computed(() => Boolean(props.actionLoading));
const isMemberSearchLoading = computed(() => Boolean(props.memberSearchLoading));
const isAddOpening = computed(
  () => Boolean(props.actionLoading) && props.actionLoadingType === "add"
);

watchEffect(() => {
  totalMember.value = props.total;
  hasMore.value = props.hasMore;
  isShowDeleteBtn.value = props.isShowDel;
  isShowAddBtn.value = props.isShowAdd;
  memberList.value = props.list as Array<IGroupMember>;
  selfValue.value = props.self;
  void ensureUserTags(
    memberList.value
      .map((member) => String(member.userID || ""))
      .filter((item) => item)
  );
});

const filteredMemberList = computed(() => {
  const keyword = submittedSearchKeyword.value.trim();
  if (!keyword) {
    return memberList.value;
  }

  return memberList.value.filter((member) =>
    matchesMemberKeyword(member as Record<string, any>, keyword)
  );
});

const emits = defineEmits([
  "more",
  "del",
  "add",
  "search",
  "handleMemberProfileShow",
  "close",
]);

const handleRoleName = (item: any) => {
  let name = "";
  switch (item?.role) {
    case TUIChatEngine.TYPES.GRP_MBR_ROLE_ADMIN:
      name = TUITranslateService.t("TUIGroup.管理员");
      break;
    case TUIChatEngine.TYPES.GRP_MBR_ROLE_OWNER:
      name = TUITranslateService.t("TUIGroup.群主");
      break;
  }
  if (name) {
    name = `(${name})`;
  }
  if (item.userID === selfValue.value.userID) {
    name += ` (${TUITranslateService.t("TUIGroup.我")})`;
  }
  return name;
};

const resolveUserTag = (item: any) => getUserTag(String(item?.userID || ""));

const isOnline = (item: IGroupMember) =>
  !!(
    item?.userID &&
    isUserOnlineStatus((props.userOnlineStatusMap as IUserStatusMap)?.[item.userID])
  );

const canDeleteMember = (item: IGroupMember) => {
  if (!isShowDeleteBtn.value || !item?.userID) {
    return false;
  }

  if (item.userID === selfValue.value.userID) {
    return false;
  }

  const selfRole = selfValue.value.role;
  if (selfRole === TUIChatEngine.TYPES.GRP_MBR_ROLE_OWNER) {
    return item.role !== TUIChatEngine.TYPES.GRP_MBR_ROLE_OWNER;
  }

  if (selfRole === TUIChatEngine.TYPES.GRP_MBR_ROLE_ADMIN) {
    return item.role === TUIChatEngine.TYPES.GRP_MBR_ROLE_MEMBER;
  }

  return false;
};

const getMore = () => {
  emits("more");
};

const submit = (item: any) => {
  if (isActionLoading.value) {
    return;
  }
  if (!canDeleteMember(item)) {
    return;
  }
  emits("del", [item]);
};

const add = () => {
  if (isActionLoading.value) {
    return;
  }
  emits("add");
};

const handleSearchInput = () => {
  if (!searchKeyword.value.trim()) {
    submittedSearchKeyword.value = "";
  }
};

const search = () => {
  const keyword = searchKeyword.value.trim();
  submittedSearchKeyword.value = keyword;
  emits("search", keyword);
};

const handleMemberProfileShow = (user: any) => {
  emits("handleMemberProfileShow", user);
};

const close = (tabName: string) => {
  emits("close", tabName);
};
</script>

<style lang="scss" scoped>
@import "../../../assets/styles/common";

.member {
  flex: 1;
  min-height: 0;
  background: #fff;
  overflow-y: auto;
  overflow-x: hidden;

  .member-search {
    padding: 12px 13px 0;
    background: #f4f5f9;

    &-h5 {
      padding: 12px 0 0;
      background: transparent;
    }

    &-box {
      display: flex;
      align-items: center;
      gap: 8px;
      width: 100%;
    }

    &-input {
      flex: 1 1 auto;
      min-width: 0;
      box-sizing: border-box;
      border: 1px solid #dfe3ea;
      border-radius: 10px;
      background: #fff;
      color: #1f2329;
      font-size: 14px;
      line-height: 20px;
      padding: 10px 12px;
      outline: none;

      &:focus {
        border-color: #3370ff;
      }
    }

    &-btn {
      flex: 0 0 72px;
      width: 72px;
      height: 42px;
      border: 1px solid #147aff;
      border-radius: 10px;
      background: #147aff;
      color: #fff;
      font-size: 14px;
      font-weight: 500;
      display: inline-flex;
      align-items: center;
      justify-content: center;
      cursor: pointer;
      white-space: nowrap;

      &:disabled {
        cursor: not-allowed;
        opacity: 0.7;
      }
    }

    &-spinner {
      border-color: rgba(255, 255, 255, 0.45);
      border-top-color: #fff;
    }
  }

  .member-action-bar {
    display: flex;
    justify-content: flex-end;
    padding: 12px 13px 0;
    background: #f4f5f9;
  }

  .list {
    display: flex;
    flex-direction: column;
    background: #f4f5f9;
    padding-top: 22px;

    &-uniapp {
      background: none;
    }

    &-item {
      padding: 13px;
      display: flex;
      justify-content: space-between;
      align-items: center;
      background: #fff;
      font-size: 14px;
      overflow: hidden;
      cursor: pointer;

      &-empty {
        justify-content: center;
        color: #888;
        cursor: default;
      }

      &:hover {
        background: #f1f2f6;
      }

      .aside {
        display: flex;
        align-items: center;
        width: 100%;
        overflow: hidden;

        .name {
          margin-left: 8px;
          font-weight: 400;
          font-size: 14px;
          color: #000;
          flex: 1;
          overflow: hidden;
          white-space: nowrap;
          text-overflow: ellipsis;
        }

        .member-tag {
          flex: 0 0 auto;
          margin-left: 8px;
          padding: 2px 8px;
          border-radius: 999px;
          background: rgba(51, 112, 255, 0.12);
          color: #3370ff;
          font-size: 12px;
          line-height: 18px;
          font-weight: 500;
        }
      }
    }
  }
}

.avatar {
  width: 36px;
  height: 36px;
  border-radius: 4px;
}

.avatar-wrapper {
  position: relative;
  width: 36px;
  height: 36px;
  flex: 0 0 auto;
}

.online-status {
  box-sizing: border-box;
  position: absolute;
  width: 10px;
  height: 10px;
  left: 30px;
  top: 30px;
  border: 2px solid #fff;
  box-shadow: 0 0 4px rgba(0, 0, 0, 0.1);
  border-radius: 50%;

  &-online {
    background: #29cc85;
  }

  &-offline {
    background: #a4a4a4;
  }
}

.member-action-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  min-width: 92px;
  border: 1px solid #3370ff;
  background: #fff;
  color: #3370ff;
  border-radius: 999px;
  font-size: 12px;
  line-height: 20px;
  padding: 4px 12px;
  cursor: pointer;

  &:disabled {
    cursor: not-allowed;
    opacity: 0.65;
  }
}

.member-action-btn-h5 {
  margin-left: auto;
  margin-right: 12px;
}

.member-action-spinner {
  width: 12px;
  height: 12px;
  border: 2px solid rgba(51, 112, 255, 0.25);
  border-top-color: #3370ff;
  border-radius: 50%;
  animation: member-action-spin 0.8s linear infinite;
}

.member-delete-action {
  flex: 0 0 auto;

  &-disabled {
    cursor: not-allowed;
    opacity: 0.45;
    pointer-events: none;
  }
}

@keyframes member-action-spin {
  to {
    transform: rotate(360deg);
  }
}

.edit-h5 {
  position: fixed;
  width: 100%;
  height: 100%;
  top: 0;
  left: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: flex-end;
  z-index: 1;

  .main {
    background: #fff;
    flex: 1;
    padding: 18px;
    border-radius: 12px 12px 0 0;
    overflow: scroll;
    height: 50%;
    width: 80vw;
  }

  &-header {
    display: flex;
    align-items: center;
    justify-content: space-between;

    .close {
      font-family: PingFangSC-Regular;
      font-weight: 400;
      font-size: 18px;
      color: #3370ff;
      letter-spacing: 0;
      line-height: 27px;
    }
  }

  &-footer {
    display: flex;

    .btn {
      flex: 1;
      border: none;
      background: #147aff;
      border-radius: 5px;
      font-family: PingFangSC-Regular;
      font-weight: 400;
      font-size: 16px;
      color: #fff;
      letter-spacing: 0;
      line-height: 27px;
      padding: 8px 0;

      &:disabled {
        opacity: 0.3;
      }
    }
  }
}
</style>
