<template>
  <div v-if="!isUniFrameWork" class="memeber-profile">
    <div class="memeber-profile-main">
      <img
        class="avatar"
        :src="
          userInfoManager.avatar ||
          'https://web.sdk.qcloud.com/component/TUIKit/assets/avatar_21.png'
        "
        onerror="
          this.onerror = null;
          this.src =
            'https://web.sdk.qcloud.com/component/TUIKit/assets/avatar_21.png';
        "
      />
      <ul class="list">
        <h2>
          {{ userInfoManager.nick || userInfoManager.userID }}
          <span v-if="userTag" class="member-tag">{{ userTag }}</span>
        </h2>
        <li>
          <label>ID：</label>
          <span>{{ userInfoManager.userID }}</span>
        </li>
        <li>
          <label>{{ TUITranslateService.t("TUIContact.个性签名") }}：</label>
          <span>{{ userInfoManager.selfSignature }}</span>
        </li>
      </ul>
    </div>
    <div class="memeber-profile-footer">
      <div
        v-if="showAddFriend()"
        class="button button-secondary"
        @click="handleAddFriend"
      >
        {{
          isAddFriendSubmitting
            ? "处理中..."
            : TUITranslateService.t("TUIContact.发送申请")
        }}
      </div>
      <div
        v-if="showEnter()"
        class="button"
        @click="enter(userInfoManager.userID, 'C2C')"
      >
        {{ TUITranslateService.t("TUIContact.发送消息") }}
      </div>
    </div>
  </div>
  <div v-else class="edit-h5">
    <main class="main">
      <header class="edit-h5-header">
        <aside class="left">
          <h1>{{ TUITranslateService.t(`TUIGroup.群成员`) }}</h1>
        </aside>
        <span class="close" @click="close('profile')">{{
          TUITranslateService.t(`关闭`)
        }}</span>
      </header>
      <div class="edit-h5-profile">
        <div class="memeber-profile-main">
          <Avatar class="avatar" :url="userInfoManager.avatar" size="60px" />
          <ul class="list">
            <h1>
              {{ userInfoManager.nick || userInfoManager.userID }}
              <span v-if="userTag" class="member-tag">{{ userTag }}</span>
            </h1>
            <li>
              <label>ID：</label>
              <span>{{ userInfoManager.userID }}</span>
            </li>
            <li>
              <label
                >{{ TUITranslateService.t("TUIContact.个性签名") }}：</label
              >
              <span>{{ userInfoManager.selfSignature }}</span>
            </li>
          </ul>
        </div>
        <div class="memeber-profile-footer">
          <div
            v-if="showAddFriend()"
            class="button button-secondary"
            @click="handleAddFriend"
          >
            {{
          isAddFriendSubmitting
            ? "处理中..."
            : TUITranslateService.t("TUIContact.发送申请")
        }}
          </div>
          <div
            v-if="showEnter()"
            class="button"
            @click="enter(userInfoManager.userID, 'C2C')"
          >
            {{ TUITranslateService.t("TUIContact.发送消息") }}
          </div>
        </div>
      </div>
    </main>
  </div>
</template>
<script lang="ts" setup>
// @ts-nocheck
import { ref, watch, watchEffect, computed } from "../../../adapter-vue";
import TUIChatEngine, {
  TUITranslateService,
  TUIUserService,
  TUIConversationService,
  TUIFriendService,
  TUIStore,
  StoreName,
} from "@tencentcloud/chat-uikit-engine-lite";
import { TUIGlobal } from "@tencentcloud/universal-api";
import Avatar from "../../common/Avatar/index.vue";
import { Toast, TOAST_TYPE } from "../../common/Toast/index";
import { IUserProfile } from "../../../interface";
import { isUniFrameWork } from "../../../utils/env";
import { extractUserTagFromProfile } from "../../../utils/user-tag";
import {
  resolveAddFriendErrorMessage,
  resolveAddFriendOutcome,
  enterC2CChat,
} from "../../TUIContact/utils/index";
import { withBusy } from "../../common/BusyOverlay/index";

const props = defineProps({
  userInfo: {
    type: Object,
    default: () => ({}),
  },
});

const isFriendShip = ref(false);
const friendRequestSent = ref(false);
const isAddFriendSubmitting = ref(false);
const userInfoManager = ref<IUserProfile>({});

watchEffect(() => {
  userInfoManager.value = props.userInfo;
  friendRequestSent.value = false;
});
const emits = defineEmits([
  "handleSwitchConversation",
  "close",
  "openConversation",
]);

watch(
  () => props.userInfo,
  async (newVal: any, oldVal: any) => {
    if (newVal === oldVal) return;
    const res = await TUIUserService.getUserProfile({
      userIDList: [props.userInfo.userID],
    });
    userInfoManager.value = res?.data[0];
    checkFriend();
  },
  {
    deep: true,
    immediate: true,
  },
);

const enter = async (ID: any, type: string) => {
  const name = `${type}${ID}`;
  TUIConversationService.getConversationProfile(name)
    .then((res: any) => {
      TUIConversationService.switchConversation(
        res.data.conversation.conversationID,
      ).then(() => {
        TUIStore.update(StoreName.GRP, "isShowManageComponent", false);
        if (isUniFrameWork) {
          TUIGlobal?.navigateBack();
        }
      });
    })
    .catch((err: any) => {
      console.warn("获取会话资料失败", err.code, err.msg);
    });
};
const checkFriend = async () => {
  if (!(userInfoManager.value as any).userID) return;
  TUIFriendService.checkFriend({
    userIDList: [userInfoManager.value.userID],
    type: TUIChatEngine.TYPES.SNS_CHECK_TYPE_BOTH,
  }).then((res: any) => {
    const relation = res?.data?.successUserIDList?.[0]?.relation;
    isFriendShip.value = relation === TUIChatEngine.TYPES.SNS_TYPE_BOTH_WAY;
    if (isFriendShip.value) {
      friendRequestSent.value = false;
    }
  });
};

const handleAddFriend = async () => {
  const userID = userInfoManager.value?.userID;
  if (!userID || isFriendShip.value || friendRequestSent.value || isAddFriendSubmitting.value) {
    return;
  }

  isAddFriendSubmitting.value = true;
  try {
    await withBusy("正在添加好友...", async () => {
      const result = await TUIFriendService.addFriend({
        to: userID,
        source: "AddSource_Type_Web",
        wording: "你好，我想加你为好友",
        type: TUIChatEngine.TYPES.SNS_ADD_TYPE_BOTH,
      });

      const outcome = resolveAddFriendOutcome(result);

      if (
        outcome.kind === "added" ||
        outcome.kind === "sent" ||
        outcome.kind === "already-friend"
      ) {
        isFriendShip.value = outcome.kind !== "sent";
        friendRequestSent.value = outcome.kind === "sent";
        enterC2CChat(userID);
        Toast({
          message:
            outcome.kind === "already-friend"
              ? outcome.message
              : TUITranslateService.t("TUIContact.添加好友成功"),
          type:
            outcome.kind === "already-friend"
              ? TOAST_TYPE.WARNING
              : TOAST_TYPE.SUCCESS,
        });
        return;
      }

      if (outcome.kind === "already-friend") {
        isFriendShip.value = true;
      }

      Toast({
        message: outcome.message,
        type:
          outcome.kind === "already-friend"
            ? TOAST_TYPE.WARNING
            : TOAST_TYPE.ERROR,
      });
    });
  } catch (error) {
    console.warn("add friend failed", error);
    Toast({
      message: resolveAddFriendErrorMessage(error),
      type: TOAST_TYPE.ERROR,
    });
  } finally {
    isAddFriendSubmitting.value = false;
  }
};

const showAddFriend = () => {
  return !isFriendShip.value && !friendRequestSent.value;
};

const showEnter = () => {
  return !!userInfoManager.value?.userID;
};

const close = (tabName: string) => {
  emits("close", tabName);
};

const userTag = computed(() =>
  extractUserTagFromProfile(userInfoManager.value),
);
</script>
<style lang="scss" scoped>
@import "../../../assets/styles/common";

.memeber-profile {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow-y: auto;
  overflow-x: hidden;

  &-main {
    display: flex;
    flex-direction: row;
    width: 100%;
    overflow: hidden;

    .avatar {
      width: 60px;
      height: 60px;
      border-radius: 8px;
      margin: 20px 10px 20px 20px;
    }

    .list {
      flex: 1;
      overflow: hidden;
      margin: 20px 10px;
      font-weight: 400;

      li {
        color: #999;
      }

      h1,
      li {
        overflow: hidden;
        white-space: nowrap;
        text-overflow: ellipsis;
      }

      .member-tag {
        display: inline-flex;
        align-items: center;
        margin-left: 8px;
        padding: 2px 8px;
        border-radius: 999px;
        background: rgba(51, 112, 255, 0.12);
        color: #3370ff;
        font-size: 12px;
        line-height: 18px;
        font-weight: 500;
        vertical-align: middle;
      }
    }
  }

  &-footer {
    border-top: 1px solid #f4f5f9;
    padding: 14px;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;

    .button {
      width: 100px;
      cursor: pointer;
      background-color: #006eff;
      color: #fff;
      padding: 8px 20px;
      border-radius: 4px;
      border: none;
      font-size: 14px;
      text-align: center;
      line-height: 20px;
    }
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
    width: 80vw;

    .edit-h5-header {
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

    .edit-h5-profile {
      .memeber-profile-main {
        .avatar {
          margin: 20px;
        }
      }
    }
  }
}
</style>
