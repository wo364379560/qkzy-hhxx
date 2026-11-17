<template>
  <div :class="['TUI-profile', !isPC && 'TUI-profile-h5']">
    <div
      v-if="displayType !== 'setting'"
      :class="['TUI-profile-basic', !isPC && 'TUI-profile-h5-basic']"
    >
      <img
        :class="[
          'TUI-profile-basic-avatar',
          !isPC && 'TUI-profile-h5-basic-avatar',
        ]"
        :src="
          userProfile.avatar ||
          'https://web.sdk.qcloud.com/component/TUIKit/assets/avatar_21.png'
        "
      />
      <div
        :class="[
          'TUI-profile-basic-info',
          !isPC && 'TUI-profile-h5-basic-info',
        ]"
      >
        <div
          :class="[
            'TUI-profile-basic-info-nick',
            !isPC && 'TUI-profile-h5-basic-info-nick',
          ]"
        >
          {{ userProfile.nick || "-" }}
        </div>
        <div
          :class="[
            'TUI-profile-basic-info-id',
            !isPC && 'TUI-profile-h5-basic-info-id',
          ]"
        >
          <label
            :class="[
              'TUI-profile-basic-info-id-label',
              !isPC && 'TUI-profile-h5-basic-info-id-label',
            ]"
            >{{ TUITranslateService.t("Profile.账号") }}:</label
          >
          <div
            :class="[
              'TUI-profile-basic-info-id-value',
              !isPC && 'TUI-profile-h5-basic-info-id-value',
            ]"
          >
            {{
              serverProfile?.phone ||
              serverProfile?.wanan_id ||
              userProfile.userID
            }}
          </div>
        </div>
        <div
          :class="[
            'TUI-profile-basic-info-id',
            !isPC && 'TUI-profile-h5-basic-info-id',
          ]"
        >
          <label
            :class="[
              'TUI-profile-basic-info-id-label',
              !isPC && 'TUI-profile-h5-basic-info-id-label',
            ]"
            >{{ TUITranslateService.t("Profile.万安号") }}:</label
          >
          <div
            :class="[
              'TUI-profile-basic-info-id-value',
              !isPC && 'TUI-profile-h5-basic-info-id-value',
            ]"
          >
            {{
              serverProfile?.wanan_id ||
              TUITranslateService.t("Profile.暂未设置")
            }}
          </div>
        </div>
      </div>
    </div>
    <div
      v-if="
        displayType !== 'profile' && (!isPC || (showSetting && !hideSetting))
      "
      ref="settingDomRef"
      :class="['TUI-profile-setting', !isPC && 'TUI-profile-h5-setting']"
      @click.stop
    >
      <div
        v-for="item in visibleSettingList"
        :key="item.value"
        :class="[
          'TUI-profile-setting-item',
          !isPC && 'TUI-profile-h5-setting-item',
          item.value === 'exit' && 'TUI-profile-h5-setting-item-exit',
        ]"
      >
        <div
          :class="[
            'TUI-profile-setting-item-label',
            !isPC && 'TUI-profile-h5-setting-item-label',
          ]"
          @click="handleSettingListItemOnClick(item)"
        >
          <div :class="['label-left']">
            <div :class="['label-title']">
              {{ TUITranslateService.t(`Profile.${item.label}`) }}
            </div>
            <div
              v-if="
                item.children && !isPC && item.childrenShowType === 'switch'
              "
              :class="['label-desc']"
            >
              {{ item.value }}
            </div>
          </div>
          <div :class="['label-right']">
            <div
              v-if="item.displayValue"
              :class="[
                'TUI-profile-setting-item-label-value',
                !isPC && 'TUI-profile-h5-setting-item-label-value',
              ]"
            >
              {{ item.displayValue }}
            </div>
            <div
              v-if="
                !item.displayValue &&
                !isPC &&
                item.children &&
                item.selectedChild &&
                item.childrenShowType === 'bottomPopup'
              "
              :class="[
                'TUI-profile-setting-item-label-value',
                !isPC && 'TUI-profile-h5-setting-item-label-value',
              ]"
            >
              {{
                TUITranslateService.t(
                  `Profile.${item?.children[item.selectedChild]?.label}`
                )
              }}
            </div>
            <Icon
              v-if="item.children || item.showArrow"
              :file="rightArrowIcon"
              width="14px"
              height="14px"
            />
          </div>
        </div>
        <div
          v-if="item.children && isPC"
          :class="[
            'TUI-profile-setting-item-children',
            !isPC && 'TUI-profile-h5-setting-item-children',
          ]"
        >
          <div
            v-for="child in item.children"
            :key="child.value"
            :class="[
              'TUI-profile-setting-item-children-item',
              !isPC && 'TUI-profile-h5-setting-item-children-item',
            ]"
            @click="handleSettingListItemOnClick(child)"
          >
            <div
              :class="[
                'TUI-profile-setting-item-children-item-label',
                !isPC && 'TUI-profile-h5-setting-item-children-item-label',
              ]"
            >
              {{ TUITranslateService.t(`Profile.${child.label}`) }}
            </div>
            <Icon
              v-if="item.selectedChild === child.value"
              :file="selectedIcon"
              width="14px"
              height="14px"
            />
          </div>
        </div>
        <BottomPopup
          v-if="
            item.children && !isPC && item.childrenShowType === 'bottomPopup'
          "
          :show="item.showChildren"
          @onClose="item.showChildren = false"
        >
          <div
            v-for="child in item.children"
            :key="child.value"
            :class="[
              'TUI-profile-setting-item-bottom-popup',
              !isPC && 'TUI-profile-h5-setting-item-bottom-popup',
            ]"
            @click="handleSettingListItemOnClick(child)"
          >
            {{ TUITranslateService.t(`Profile.${child.label}`) }}
          </div>
        </BottomPopup>
      </div>
    </div>
    <AboutDialog
      v-if="isAboutBoxShow"
      :userProfile="userProfile"
      @closeAboutBox="closeAboutBox"
    />
    <ResetPasswordDialog
      v-if="isResetPasswordDialogShow"
      @close="closeResetPasswordDialog"
      @success="handleResetPasswordSuccess"
    />
    <InviteCodeDialog
      v-if="isInviteCodeDialogShow"
      :profile="serverProfile"
      @close="closeInviteCodeDialog"
      @updated="handleInviteCodeUpdated"
    />
    <EditProfileDialog
      v-if="isEditProfileBoxShow"
      @closeEditProfileBox="closeEditProfileBox"
    />
  </div>
</template>
<script lang="ts" setup>
import { computed, ref, watch, nextTick, onMounted } from "../TUIKit/adapter-vue";
import TUIChatEngine, {
  TUITranslateService,
  TUIUserService,
  TUIStore,
  StoreName,
} from "@tencentcloud/chat-uikit-engine-lite";
import { Toast, TOAST_TYPE } from "../TUIKit/components/common/Toast/index";
import BottomPopup from "../TUIKit/components/common/BottomPopup/index.vue";
import Icon from "../TUIKit/components/common/Icon.vue";
import AboutDialog from "../components/About.vue";
import EditProfileDialog from "../components/EditProfile.vue";
import InviteCodeDialog from "../components/InviteCodeDialog.vue";
import ResetPasswordDialog from "../components/ResetPasswordDialog.vue";
import rightArrowIcon from "../TUIKit/assets/icon/right-icon.svg";
import selectedIcon from "../TUIKit/assets/icon/selected.svg";
import { IUserProfile } from "../TUIKit/interface";
import { isPC } from "../TUIKit/utils/env";
import router from "../router/index";
import { deepCopy } from "../TUIKit/components/TUIChat/utils/utils";
import { useAuthSession } from "@/features/auth/session";
import { syncCurrentUserProfile } from "@/features/account/profile";
import {
  getInvisiblePreference,
  restoreSelfInvisibleStatus,
  setSelfInvisibleStatus,
} from "@/features/presence/invisible";
import { getProfile } from "@/api";
import type { ChatProfile } from "@/features/auth/types";

const props = defineProps({
  displayType: {
    type: String,
    default: "profile", // "profile"/"setting"/"all"
  },
  showSetting: {
    type: Boolean,
    default: false,
  },
});
const emits = defineEmits(["update:showSetting"]);
const settingDomRef = ref();
const authSession = useAuthSession();
const userProfile = ref<IUserProfile>({} as IUserProfile);
const serverProfile = ref<ChatProfile | null>(null);
const isAboutBoxShow = ref<boolean>(false);
const isEditProfileBoxShow = ref<boolean>(false);
const isResetPasswordDialogShow = ref<boolean>(false);
const isInviteCodeDialogShow = ref<boolean>(false);
const hideSetting = ref<boolean>(false);
const settingList = ref<{
  [propsName: string]: {
    value: string;
    label: string;
    displayValue?: string;
    showArrow?: boolean;
    onClick?: any;
    // children related
    selectedChild?: string;
    childrenShowType?: string; // "bottomPopup"/"switch"
    showChildren?: boolean;
    children?: {
      [propsName: string]: {
        value: string;
        label: string;
        onClick?: any;
      };
    };
  };
}>({
  editProfile: {
    value: "editProfile",
    label: "编辑资料",
    onClick: () => {
      openStandaloneDialog(isEditProfileBoxShow);
    },
  },
  allowType: {
    value: "allowType",
    label: "加我为好友时",
    selectedChild: "",
    childrenShowType: "bottomPopup",
    showChildren: false,
    onClick: (item: any) => {
      if (!isPC) {
        item.showChildren = true;
      }
    },
    children: {
      [TUIChatEngine.TYPES.ALLOW_TYPE_ALLOW_ANY]: {
        value: TUIChatEngine.TYPES.ALLOW_TYPE_ALLOW_ANY,
        label: "同意任何用户加好友",
        onClick: (item: any) => {
          switchAllowType(item.value);
        },
      },
      [TUIChatEngine.TYPES.ALLOW_TYPE_NEED_CONFIRM]: {
        value: TUIChatEngine.TYPES.ALLOW_TYPE_NEED_CONFIRM,
        label: "需要验证",
        onClick: (item: any) => {
          switchAllowType(item.value);
        },
      },
      [TUIChatEngine.TYPES.ALLOW_TYPE_DENY_ANY]: {
        value: TUIChatEngine.TYPES.ALLOW_TYPE_DENY_ANY,
        label: "拒绝任何人加好友",
        onClick: (item: any) => {
          switchAllowType(item.value);
        },
      },
    },
  },
  displayOnlineStatus: {
    value: "displayOnlineStatus",
    label: "显示在线状态",
    selectedChild: "userLevelOnlineStatusOpen",
    childrenShowType: "bottomPopup",
    showChildren: false,
    onClick(item: any) {
      if (!isPC) {
        item.showChildren = true;
      }
    },
    children: {
      userLevelOnlineStatusOpen: {
        value: "userLevelOnlineStatusOpen",
        label: "开启",
        onClick() {
          switchUserLevelOnlineStatus(true);
        },
      },
      userLevelOnlineStatusClose: {
        value: "userLevelOnlineStatusClose",
        label: "关闭",
        onClick() {
          switchUserLevelOnlineStatus(false);
        },
      },
    },
  },
  invisibleStatus: {
    value: "invisibleStatus",
    label: "隐身状态",
    selectedChild: "invisibleStatusOff",
    childrenShowType: "bottomPopup",
    showChildren: false,
    onClick(item: any) {
      if (!isPC) {
        item.showChildren = true;
      }
    },
    children: {
      invisibleStatusOff: {
        value: "invisibleStatusOff",
        label: "在线",
        onClick() {
          switchInvisibleStatus(false);
        },
      },
      invisibleStatusOn: {
        value: "invisibleStatusOn",
        label: "隐身",
        onClick() {
          switchInvisibleStatus(true);
        },
      },
    },
  },
  about: {
    value: "about",
    label: "隐私与安全",
    selectedChild: "",
    childrenShowType: "bottomPopup",
    showChildren: false,
    children: {},
    onClick: () => {
      openStandaloneDialog(isAboutBoxShow);
    },
  },
  resetPassword: {
    value: "resetPassword",
    label: "重置密码",
    showArrow: true,
    onClick: () => {
      openStandaloneDialog(isResetPasswordDialogShow);
    },
  },
  inviteCode: {
    value: "inviteCode",
    label: "邀请码",
    displayValue: TUITranslateService.t("Profile.未绑定"),
    showArrow: true,
    onClick: () => {
      openStandaloneDialog(isInviteCodeDialogShow);
    },
  },
  exit: {
    value: "exit",
    label: "退出登录",
    onClick: async () => {
      await authSession.logoutSession();
      router.replace({ name: "login" });
    },
  },
});

const visibleSettingList = computed(() =>
  Object.values(settingList.value).filter(
    (item) => item.value !== "displayOnlineStatus"
  )
);

const handleSettingListItemOnClick = (item: any) => {
  if (item?.onClick && typeof item?.onClick === "function") {
    item.onClick(item);
  }
};

const closeAboutBox = () => {
  isAboutBoxShow.value = false;
  emits("update:showSetting", false);
};

const closeEditProfileBox = () => {
  isEditProfileBoxShow.value = false;
  emits("update:showSetting", false);
  void refreshServerProfile();
};

const closeResetPasswordDialog = () => {
  isResetPasswordDialogShow.value = false;
  emits("update:showSetting", false);
};

const closeInviteCodeDialog = () => {
  isInviteCodeDialogShow.value = false;
  emits("update:showSetting", false);
};

const openStandaloneDialog = (target: { value: boolean }) => {
  hideSetting.value = true;
  removeClickListener(settingDomRef.value);
  target.value = true;
};

const handleResetPasswordSuccess = () => {
  Toast({
    message: TUITranslateService.t("Profile.重置密码成功"),
    type: TOAST_TYPE.SUCCESS,
  });
};

const handleInviteCodeUpdated = (profile: ChatProfile) => {
  serverProfile.value = profile;
  syncInviteCodeDisplay(profile);
  Toast({
    message: TUITranslateService.t("Profile.绑定邀请码成功"),
    type: TOAST_TYPE.SUCCESS,
  });
};

const updateMyProfile = (props: object) => {
  syncCurrentUserProfile(props as any)
    .then(() => {
      void refreshServerProfile();
      Toast({
        message: "更新用户资料成功",
        type: TOAST_TYPE.SUCCESS,
      });
    })
    .catch((err: any) => {
      console.warn("更新用户资料失败", err);
      Toast({
        message: "更新用户资料失败",
        type: TOAST_TYPE.ERROR,
      });
    });
};

const refreshServerProfile = async () => {
  try {
    serverProfile.value = await getProfile();
    syncInviteCodeDisplay(serverProfile.value);
  } catch (error) {
    console.warn("获取服务端资料失败", error);
  }
};

function syncInviteCodeDisplay(profile: ChatProfile | null) {
  const inviteCode = profile?.invite_code?.trim?.() || "";
  settingList.value.inviteCode.displayValue =
    inviteCode || TUITranslateService.t("Profile.未绑定");
}

function syncInvisibleStatusDisplay(enabled: boolean) {
  settingList.value.invisibleStatus.selectedChild = enabled
    ? "invisibleStatusOn"
    : "invisibleStatusOff";
}

TUIStore.watch(StoreName.USER, {
  userProfile: (userProfileData: IUserProfile) => {
    userProfile.value = deepCopy(userProfileData);
    if (userProfile?.value?.allowType) {
      settingList.value.allowType.selectedChild = userProfile?.value?.allowType;
    }
  },
  displayOnlineStatus(isOnlineStatusDisplay: boolean) {
    settingList.value.displayOnlineStatus.selectedChild = isOnlineStatusDisplay
      ? "userLevelOnlineStatusOpen"
      : "userLevelOnlineStatusClose";
  },
});

// pc click outside
let clickOutside = false;
let clickInner = false;
const onClickOutside = (component: any) => {
  document.addEventListener("mousedown", onClickDocument);
  component?.addEventListener &&
    component?.addEventListener("mousedown", onClickTarget);
};
const onClickDocument = () => {
  clickOutside = true;
  if (!clickInner && clickOutside) {
    emits("update:showSetting", false);
    removeClickListener(settingDomRef.value);
  }
  clickOutside = false;
  clickInner = false;
};
const onClickTarget = () => {
  clickInner = true;
};
const removeClickListener = (component: any) => {
  document.removeEventListener("mousedown", onClickDocument);
  component?.removeEventListener &&
    component?.removeEventListener("mousedown", onClickTarget);
};

watch(
  () => props.showSetting,
  (newVal: any, oldVal: any) => {
    if (isPC && newVal && !oldVal) {
      nextTick(() => {
        onClickOutside(settingDomRef.value);
      });
    }
  },
  {
    immediate: true,
  }
);

onMounted(() => {
  const invisiblePreference = getInvisiblePreference();
  syncInvisibleStatusDisplay(invisiblePreference);
  if (invisiblePreference) {
    void restoreSelfInvisibleStatus().catch((error) => {
      console.warn("restore invisible status failed", error);
    });
  }
  // get user profile
  TUIUserService.getUserProfile().then((res: any) => {
    userProfile.value = res.data;
  });
  void refreshServerProfile();
});

function switchAllowType(value: string) {
  updateMyProfile({ allowType: value });
  settingList.value.allowType.showChildren = false;
}

function switchUserLevelOnlineStatus(status: boolean) {
  TUIUserService.switchUserStatus({ displayOnlineStatus: status });
  settingList.value.displayOnlineStatus.showChildren = false;
}

async function switchInvisibleStatus(status: boolean) {
  settingList.value.invisibleStatus.showChildren = false;
  try {
    await setSelfInvisibleStatus(status);
    syncInvisibleStatusDisplay(status);
    Toast({
      message: TUITranslateService.t(
        status ? "Profile.已开启隐身状态" : "Profile.已关闭隐身状态"
      ),
      type: TOAST_TYPE.SUCCESS,
    });
  } catch (error) {
    console.warn("switch invisible status failed", error);
    Toast({
      message: TUITranslateService.t("Profile.切换隐身状态失败"),
      type: TOAST_TYPE.ERROR,
    });
  }
}

</script>

<style lang="scss" scoped>
@import "../styles/profile";
</style>
