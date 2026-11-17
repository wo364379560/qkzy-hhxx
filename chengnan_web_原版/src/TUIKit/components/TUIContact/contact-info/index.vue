<template>
  <div
    v-if="
      typeof contactInfoData === 'object' && Object.keys(contactInfoData).length
    "
    :class="['tui-contact-info', !isPC && 'tui-contact-info-h5']"
  >
    <div
      v-if="!isPC"
      :class="[
        'tui-contact-info-header',
        !isPC && 'tui-contact-info-h5-header',
      ]"
    >
      <div
        :class="[
          'tui-contact-info-header-icon',
          !isPC && 'tui-contact-info-h5-header-icon',
        ]"
        @click="resetContactSearchingUIData"
      >
        <Icon :file="backSVG" />
      </div>
      <div
        :class="[
          'tui-contact-info-header-title',
          !isPC && 'tui-contact-info-h5-header-title',
        ]"
      >
        {{ TUITranslateService.t("TUIContact.添加好友/群聊") }}
      </div>
    </div>
    <div
      :class="['tui-contact-info-basic', !isPC && 'tui-contact-info-h5-basic']"
    >
      <div
        :class="[
          'tui-contact-info-basic-text',
          !isPC && 'tui-contact-info-h5-basic-text',
        ]"
      >
        <div
          :class="[
            'tui-contact-info-basic-text-name',
            !isPC && 'tui-contact-info-h5-basic-text-name',
          ]"
        >
          {{ generateContactInfoName(contactInfoData) }}
        </div>
        <div
          v-for="item in contactInfoBasicList"
          :key="item.label"
          :class="[
            'tui-contact-info-basic-text-other',
            item.copyable && 'tui-contact-info-basic-text-other-copyable',
            !isPC && 'tui-contact-info-h5-basic-text-other',
          ]"
          @click="onBasicInfoClicked(item)"
        >
          <span>
            {{
              `${TUITranslateService.t(`TUIContact.${item.label}`)}: ${
                item.data
              }`
            }}
          </span>
          <button
            v-if="item.label === '归属' && shouldShowBindOwnerButton()"
            class="tui-contact-info-owner-bind"
            :disabled="isBindingOwner"
            @click.stop="onBindOwnerClicked"
          >
            {{
              TUITranslateService.t(
                `TUIContact.${isBindingOwner ? "绑定中" : "绑定归属"}`,
              )
            }}
          </button>
          <span
            v-if="item.copyable"
            class="tui-contact-info-basic-text-other-copy"
          >
            {{ TUITranslateService.t("TUIChat.复制") }}
          </span>
        </div>
      </div>
      <img
        :class="[
          'tui-contact-info-basic-avatar',
          !isPC && 'tui-contact-info-h5-basic-avatar',
        ]"
        :src="generateAvatar(contactInfoData)"
      />
    </div>
    <div
      v-if="contactInfoMoreList[0]"
      :class="['tui-contact-info-more', !isPC && 'tui-contact-info-h5-more']"
    >
      <div
        v-for="item in contactInfoMoreList"
        :key="item.key"
        :class="[
          'tui-contact-info-more-item',
          !isPC && 'tui-contact-info-h5-more-item',
          item.labelPosition === CONTACT_INFO_LABEL_POSITION.TOP
            ? 'tui-contact-info-more-item-top'
            : 'tui-contact-info-more-item-left',
        ]"
      >
        <div
          :class="[
            'tui-contact-info-more-item-label',
            !isPC && 'tui-contact-info-h5-more-item-label',
          ]"
        >
          {{ `${TUITranslateService.t(`TUIContact.${item.label}`)}` }}
        </div>
        <div
          :class="[
            'tui-contact-info-more-item-content',
            !isPC && 'tui-contact-info-h5-more-item-content',
          ]"
        >
          <div
            v-if="!item.editing"
            :class="[
              'tui-contact-info-more-item-content-text',
              !isPC && 'tui-contact-info-h5-more-item-content-text',
            ]"
          >
            <div
              :class="[
                'tui-contact-info-more-item-content-text-data',
                !isPC && 'tui-contact-info-h5-more-item-content-text-data',
              ]"
            >
              {{ item.data }}
            </div>
            <div
              v-if="item.editable"
              :class="[
                'tui-contact-info-more-item-content-text-icon',
                !isPC && 'tui-contact-info-h5-more-item-content-text-icon',
              ]"
              @click="setEditing(item)"
            >
              <Icon :file="editSVG" width="14px" height="14px" />
            </div>
          </div>
          <div
            v-else-if="item.editType === CONTACT_INFO_MORE_EDIT_TYPE.INPUT"
            class="tui-contact-info-more-item-content-edit"
          >
            <input
              v-model="item.data"
              :class="[
                'tui-contact-info-more-item-content-input',
                !isPC && 'tui-contact-info-h5-more-item-content-input',
              ]"
              type="text"
              @keyup.enter="onContactInfoEmitSubmit(item)"
            />
            <button
              type="button"
              class="tui-contact-info-more-item-content-confirm"
              @click="onContactInfoEmitSubmit(item)"
            >
              {{ TUITranslateService.t("TUIContact.确定") }}
            </button>
          </div>
          <textarea
            v-else-if="item.editType === CONTACT_INFO_MORE_EDIT_TYPE.TEXTAREA"
            v-model="item.data"
            :class="[
              'tui-contact-info-more-item-content-textarea',
              !isPC && 'tui-contact-info-h5-more-item-content-textarea',
            ]"
            confirm-type="done"
          />
          <div
            v-else-if="item.editType === CONTACT_INFO_MORE_EDIT_TYPE.SWITCH"
            @click="onContactInfoEmitSubmit(item)"
          >
            <SwitchBar :value="item.data" />
          </div>
        </div>
      </div>
    </div>
    <div
      :class="[
        'tui-contact-info-button',
        !isPC && 'tui-contact-info-h5-button',
      ]"
    >
      <button
        v-for="item in contactInfoButtonList"
        :key="item.key"
        :disabled="Boolean(item.disabled)"
        :class="[
          'tui-contact-info-button-item',
          item.disabled && 'tui-contact-info-button-item-disabled',
          !isPC && 'tui-contact-info-h5-button-item',
          item.type === CONTACT_INFO_BUTTON_TYPE.CANCEL
            ? `tui-contact-info-button-item-cancel`
            : `tui-contact-info-button-item-submit`,
        ]"
        @click="onContactInfoButtonClicked(item)"
      >
        {{ TUITranslateService.t(`TUIContact.${item.label}`) }}
      </button>
    </div>
    <Dialog
      v-if="deleteFriendDialogShow"
      :show="deleteFriendDialogShow"
      :center="true"
      :isHeaderShow="isPC"
      :title="TUITranslateService.t('TUIContact.确认删除好友')"
      @submit="confirmDeleteFriend"
      @update:show="updateDeleteFriendDialog"
    >
      <p class="delete-friend-dialog-title">
        {{
          TUITranslateService.t(
            "TUIContact.删除后对方会从你的好友列表移除，确认删除吗？",
          )
        }}
      </p>
    </Dialog>
  </div>
</template>
<script setup lang="ts">
// @ts-nocheck
import TUIChatEngine, {
  TUIStore,
  StoreName,
  TUITranslateService,
  IGroupModel,
  Friend,
  FriendApplication,
} from "@tencentcloud/chat-uikit-engine-lite";
import { TUIGlobal } from "@tencentcloud/universal-api";
import { ref, computed, onMounted, onUnmounted, watch } from "../../../adapter-vue";
import { isPC } from "../../../utils/env";

import {
  generateAvatar,
  generateContactInfoName,
  generateContactInfoBasic,
  isFriend,
  isApplicationType,
} from "../utils/index";
import { bindFriendOwnerFromMe, getProfile, getUserBriefs } from "@/api";
import { captureFriendOwnerScope, synchronizeFriendOwner, friendOwnerSelfProfile, friendOwnerTargets } from "@/features/account/friendOwnership";
import type { ChatProfile } from "@/features/auth/types";
import {
  contactMoreInfoConfig,
  contactButtonConfig,
} from "./contact-info-config";
import CopyManager from "../../TUIChat/utils/copy";
import Icon from "../../common/Icon.vue";
import Dialog from "../../common/Dialog/index.vue";
import editSVG from "../../../assets/icon/edit.svg";
import backSVG from "../../../assets/icon/back.svg";
import SwitchBar from "../../common/SwitchBar/index.vue";
import { Toast, TOAST_TYPE } from "../../common/Toast/index";
import {
  IBlackListUserItem,
  IContactInfoMoreItem,
  IContactInfoButton,
} from "../../../interface";
import {
  CONTACT_INFO_LABEL_POSITION,
  CONTACT_INFO_MORE_EDIT_TYPE,
  CONTACT_INFO_BUTTON_TYPE,
} from "../../../constant";
import { deepCopy } from "../../TUIChat/utils/utils";

type IContactInfoType =
  IGroupModel | Friend | FriendApplication | IBlackListUserItem;

const emits = defineEmits(["switchConversation"]);

const contactInfoData = ref<IContactInfoType>({} as IContactInfoType);
const contactInfoBasicList = ref<
  Array<{ label: string; data: string; copyable?: boolean }>
>([]);
const contactInfoMoreList = ref<IContactInfoMoreItem[]>([]);
const contactInfoButtonList = ref<IContactInfoButton[]>([]);
const deleteFriendDialogShow = ref(false);
const pendingDeleteFriendButton = ref<IContactInfoButton | null>(null);
type ContactBusinessInfo = {
  wananID: string;
  phone: string;
  nickname: string;
  ownerID: number;
  ownerName: string;
};

const contactBusinessInfoCache = new Map<string, ContactBusinessInfo>();
let currentContactInfoRequestID = 0;
let contactBusinessOwnerScope = "";
const currentProfile = ref<ChatProfile | null>(null);
const bindingOwnerUserID = ref("");
watch(friendOwnerSelfProfile, profile=>{
 if (profile?.im_user_id===TUIChatEngine.chat?.getLoginUser()) currentProfile.value=profile;
});
watch(friendOwnerTargets, targets=>{
 const id=String((contactInfoData.value as Friend)?.userID||'');
 if (id && targets[id]) {contactBusinessInfoCache.delete(id);void hydrateBusinessInfoForUser(id,currentContactInfoRequestID)}
});


const setEditing = (item: any) => {
  item.editing = true;
};

const isGroup = computed((): boolean =>
  (contactInfoData.value as IGroupModel)?.groupID ? true : false,
);

const isApplication = computed((): boolean => {
  return isApplicationType(contactInfoData?.value);
});

// is both friend, if is group type always false
const isBothFriend = ref<boolean>(false);

// is group member, including ordinary member, admin, group owner
const isGroupMember = computed((): boolean => {
  return (contactInfoData.value as IGroupModel)?.selfInfo?.userID
    ? true
    : false;
});

// is in black list, if is group type always false
const isInBlackList = computed((): boolean => {
  return (
    !isGroup.value &&
    blackList.value?.findIndex(
      (item: IBlackListUserItem) =>
        item?.userID === (contactInfoData.value as IBlackListUserItem)?.userID,
    ) >= 0
  );
});

const blackList = ref<IBlackListUserItem[]>([]);

onMounted(() => {
  TUIStore.watch(StoreName.CUSTOM, {
    currentContactInfo: onCurrentContactInfoUpdated,
  });
  TUIStore.watch(StoreName.USER, {
    userBlacklist: onUserBlacklistUpdated,
  });
  void loadCurrentProfile();
});

onUnmounted(() => {
  TUIStore.unwatch(StoreName.CUSTOM, {
    currentContactInfo: onCurrentContactInfoUpdated,
  });
  TUIStore.unwatch(StoreName.USER, {
    userBlacklist: onUserBlacklistUpdated,
  });
});

const resetContactInfoUIData = () => {
  contactInfoData.value = {} as IContactInfoType;
  contactInfoBasicList.value = [];
  contactInfoMoreList.value = [];
  contactInfoButtonList.value = [];
};

const resetContactSearchingUIData = () => {
  TUIStore.update(StoreName.CUSTOM, "currentContactInfo", {});
  TUIStore.update(StoreName.CUSTOM, "currentContactSearchingStatus", false);
  TUIGlobal?.closeSearching && TUIGlobal?.closeSearching();
};

const onContactInfoEmitSubmit = (item: any) => {
  item.editSubmitHandler &&
    item.editSubmitHandler({
      item,
      contactInfoData: contactInfoData.value,
      isBothFriend: isBothFriend.value,
      isInBlackList: isInBlackList.value,
    });
};

const onContactInfoButtonClicked = (item: any) => {
  if (item.disabled) {
    return;
  }
  if (item.key === "deleteFriend") {
    pendingDeleteFriendButton.value = item;
    deleteFriendDialogShow.value = true;
    return;
  }
  item.onClick &&
    item.onClick({
      contactInfoData: contactInfoData.value,
      contactInfoMoreList: contactInfoMoreList.value,
    });
  if (
    item.key === "enterGroupConversation" ||
    item.key === "enterC2CConversation"
  ) {
    emits("switchConversation", contactInfoData.value);
    resetContactSearchingUIData();
  }
};

async function loadCurrentProfile() {
  try {
    const scope=captureFriendOwnerScope();
    const profile=await getProfile({fresh:true});
    if (!scope.active()) return;
    currentProfile.value = profile;
    if (
      contactInfoData.value &&
      Object.keys(contactInfoData.value).length > 0
    ) {
      generateButton();
    }
  } catch (error) {
    console.warn("load current profile failed", error);
  }
}

function shouldShowBindOwnerButton() {
  if (isGroup.value || isApplication.value || isInBlackList.value) {
    return false;
  }
  const targetUserID = String((contactInfoData.value as Friend)?.userID || "");
  return Boolean(targetUserID);
}

const isBindingOwner = computed(() => {
  const targetUserID = String((contactInfoData.value as Friend)?.userID || "");
  return Boolean(targetUserID && bindingOwnerUserID.value === targetUserID);
});

async function onBindOwnerClicked() {
  const targetUserID = String((contactInfoData.value as Friend)?.userID || "");
  if (!targetUserID || bindingOwnerUserID.value || !shouldShowBindOwnerButton()) {
    return;
  }
  bindingOwnerUserID.value = targetUserID;
  try {
    const scope=captureFriendOwnerScope();
    const assignment=await synchronizeFriendOwner(targetUserID);
    if (!scope.active()) return;
    const updatedUser=friendOwnerTargets.value[targetUserID];
    if (!updatedUser) {Toast({message:'好友关系与归属待核对，请稍后重试',type:TOAST_TYPE.WARNING});return}
    const businessInfo: ContactBusinessInfo = {
      wananID: updatedUser.wanan_id || "",
      phone: updatedUser.phone || "",
      nickname: updatedUser.nickname || "",
      ownerID: Number(updatedUser.owner_id || 0),
      ownerName: updatedUser.owner_name || "",
    };
    contactBusinessInfoCache.set(targetUserID, businessInfo);
    if ((contactInfoData.value as Friend)?.userID === targetUserID) {
      contactInfoData.value = {
        ...contactInfoData.value,
        _wananID: businessInfo.wananID,
        _phone: businessInfo.phone,
        _nickname: businessInfo.nickname,
        _ownerID: businessInfo.ownerID,
        _ownerName: businessInfo.ownerName,
      };
      contactInfoBasicList.value = generateContactInfoBasic(
        contactInfoData.value,
      );
    }
    Toast({
      message: assignment?.state === "migrated" || assignment?.state === "unchanged" ? "归属已同步" : assignment?.state === "conflict" ? "好友已建立，归属冲突待管理员处理" : "好友已建立，归属待同步",
      type: assignment?.state === "migrated" || assignment?.state === "unchanged" ? TOAST_TYPE.SUCCESS : TOAST_TYPE.WARNING,
    });
  } catch (error) {
    Toast({
      message: error instanceof Error ? error.message : "归属绑定失败",
      type: TOAST_TYPE.ERROR,
    });
  } finally {
    bindingOwnerUserID.value = "";
  }
}

function updateDeleteFriendDialog(isShow: boolean) {
  deleteFriendDialogShow.value = isShow;
  if (!isShow) {
    pendingDeleteFriendButton.value = null;
  }
}

function confirmDeleteFriend() {
  const item = pendingDeleteFriendButton.value;
  pendingDeleteFriendButton.value = null;
  deleteFriendDialogShow.value = false;
  item?.onClick?.({
    contactInfoData: contactInfoData.value,
    contactInfoMoreList: contactInfoMoreList.value,
  });
}

const onBasicInfoClicked = async (item: {
  label: string;
  data: string;
  copyable?: boolean;
}) => {
  if (!item.copyable || !item.data) {
    return;
  }
  await CopyManager.copyTextOrHtml(item.data, "text");
};

const generateMoreInfo = () => {
  contactInfoMoreList.value = [];
  if (!isApplication.value) {
    if (
      isGroup.value &&
      !isGroupMember.value &&
      (contactInfoData.value as IGroupModel)?.type !==
        TUIChatEngine?.TYPES?.GRP_AVCHATROOM
    ) {
      const setWords = deepCopy(contactMoreInfoConfig.setWords);
      setWords.data = "";
      contactInfoMoreList.value.push(setWords);
    }
    if (!isGroup.value && !isInBlackList.value) {
      const setRemark = deepCopy(contactMoreInfoConfig.setRemark);
      setRemark.data = (contactInfoData.value as Friend)?.remark || "";
      setRemark.editing = false;
      contactInfoMoreList.value.push(setRemark);
    }
    if (!isGroup.value && (isBothFriend.value || isInBlackList.value)) {
      const blackListItem = deepCopy(contactMoreInfoConfig.blackList);
      blackListItem.data = isInBlackList.value || false;
      contactInfoMoreList.value.push(blackListItem);
    }
  } else {
    const displayWords = deepCopy(contactMoreInfoConfig.displayWords);
    displayWords.data =
      (contactInfoData.value as FriendApplication)?.wording || "";
    contactInfoMoreList.value.push(displayWords);
  }
};

const generateButton = () => {
  contactInfoButtonList.value = [];
  if (isInBlackList.value) {
    return;
  }
  if (isApplication.value) {
    if (
      (contactInfoData.value as FriendApplication)?.type ===
      TUIChatEngine?.TYPES?.SNS_APPLICATION_SENT_TO_ME
    ) {
      contactInfoButtonList?.value?.push(
        contactButtonConfig.refuseFriendApplication,
      );
      contactInfoButtonList?.value?.push(
        contactButtonConfig.acceptFriendApplication,
      );
    }
  } else {
    if (isGroup.value && isGroupMember.value) {
      switch ((contactInfoData.value as IGroupModel)?.selfInfo?.role) {
        case "Owner":
          contactInfoButtonList?.value?.push(contactButtonConfig.dismissGroup);
          break;
        default:
          contactInfoButtonList?.value?.push(contactButtonConfig.quitGroup);
          break;
      }
      contactInfoButtonList?.value?.push(contactButtonConfig.reportGroup);
      contactInfoButtonList?.value?.push(
        contactButtonConfig.enterGroupConversation,
      );
    } else if (!isGroup.value && isBothFriend.value) {
      contactInfoButtonList?.value?.push(contactButtonConfig.deleteFriend);
      contactInfoButtonList?.value?.push(contactButtonConfig.reportUser);
      contactInfoButtonList?.value?.push(
        contactButtonConfig.enterC2CConversation,
      );
    } else {
      if (isGroup.value) {
        contactInfoButtonList?.value?.push(contactButtonConfig.reportGroup);
        contactInfoButtonList?.value?.push(
          (contactInfoData.value as IGroupModel)?.type ===
            TUIChatEngine?.TYPES?.GRP_AVCHATROOM
            ? contactButtonConfig.joinAVChatGroup
            : contactButtonConfig.joinGroup,
        );
      } else {
        contactInfoButtonList?.value?.push(contactButtonConfig.reportUser);
        contactInfoButtonList?.value?.push(
          (contactInfoData.value as any)?._friendRequestPending
            ? contactButtonConfig.friendRequestPending
            : contactButtonConfig.addFriend,
        );
      }
    }
  }
};

function onUserBlacklistUpdated(userBlacklist: IBlackListUserItem[]) {
  blackList.value = userBlacklist;
}

function getContactIdentity(
  contactInfo: Partial<IContactInfoType> | undefined,
) {
  if (!contactInfo) {
    return "";
  }
  if ((contactInfo as IGroupModel)?.groupID) {
    return `group:${(contactInfo as IGroupModel).groupID}`;
  }
  if ((contactInfo as Friend)?.userID) {
    return `user:${(contactInfo as Friend).userID}:${contactInfo.type || ""}`;
  }
  return "";
}

function isSameContactInfo(
  previousContactInfo: Partial<IContactInfoType> | undefined,
  nextContactInfo: Partial<IContactInfoType> | undefined,
) {
  return (
    getContactIdentity(previousContactInfo) ===
      getContactIdentity(nextContactInfo) &&
    String((previousContactInfo as Friend)?.remark || "") ===
      String((nextContactInfo as Friend)?.remark || "") &&
    Number((previousContactInfo as Friend)?.addTime || 0) ===
      Number((nextContactInfo as Friend)?.addTime || 0) &&
    Boolean((previousContactInfo as any)?._friendshipConfirmed) ===
      Boolean((nextContactInfo as any)?._friendshipConfirmed) &&
    Boolean((previousContactInfo as any)?._friendRequestPending) ===
      Boolean((nextContactInfo as any)?._friendRequestPending) &&
    String((previousContactInfo as FriendApplication)?.wording || "") ===
      String((nextContactInfo as FriendApplication)?.wording || "")
  );
}

function rebuildContactInfoPanels(contactInfo: IContactInfoType) {
  if ((contactInfo as any)?.infoKeyList) {
    contactInfoMoreList.value = (contactInfo as any).infoKeyList.map(
      (key: string) => deepCopy((contactMoreInfoConfig as any)[key]),
    );
  } else {
    generateMoreInfo();
  }

  if ((contactInfo as any)?.btnKeyList) {
    contactInfoButtonList.value = (contactInfo as any).btnKeyList.map(
      (key: string) => deepCopy((contactButtonConfig as any)[key]),
    );
  } else {
    generateButton();
  }
}

function resolveLocalFriendState(contactInfo: IContactInfoType) {
  const userID = (contactInfo as Friend)?.userID || "";
  if ((contactInfo as IGroupModel)?.groupID || !userID) {
    return false;
  }
  if ((contactInfo as Friend)?.addTime) {
    return true;
  }
  if ((contactInfo as any)?._friendshipConfirmed) {
    return true;
  }
  const friendList = TUIStore.getData(StoreName.FRIEND, "friendList") || [];
  return (
    Array.isArray(friendList) &&
    friendList.some((friend: any) => friend?.userID === userID)
  );
}

async function hydrateBusinessInfoForUser(userID: string, requestID: number) {
  const ownerScope=captureFriendOwnerScope();
  if (!ownerScope.active()) return;
  if (ownerScope.key!==contactBusinessOwnerScope) {contactBusinessInfoCache.clear();contactBusinessOwnerScope=ownerScope.key}

  if (!userID || isGroup.value) {
    return;
  }

  const applyBusinessInfo = (businessInfo: ContactBusinessInfo) => {
    if (
      ownerScope.active() && requestID === currentContactInfoRequestID &&
      (contactInfoData.value as Friend)?.userID === userID
    ) {
      contactInfoData.value = {
        ...contactInfoData.value,
        _wananID: businessInfo.wananID,
        _phone: businessInfo.phone,
        _nickname: businessInfo.nickname,
        _ownerID: businessInfo.ownerID,
        _ownerName: businessInfo.ownerName,
      };
      contactInfoBasicList.value = generateContactInfoBasic(
        contactInfoData.value,
      );
      rebuildContactInfoPanels(contactInfoData.value);
    }
  };

  const cachedBusinessInfo = contactBusinessInfoCache.get(userID);
  if (cachedBusinessInfo) {
    applyBusinessInfo(cachedBusinessInfo);
    if (
      cachedBusinessInfo.phone &&
      cachedBusinessInfo.wananID &&
      cachedBusinessInfo.ownerID !== undefined
    ) {
      // Ownership can change independently of the static contact details.
    }
  }

  try {
    const briefs = await getUserBriefs([userID]);
    const exactUser = briefs.find((item) => item.im_user_id === userID);
    if (!exactUser) return;

    const businessInfo = {
      wananID: exactUser.wanan_id || "",
      phone: exactUser.phone || "",
      nickname: exactUser.nickname || "",
      ownerID: Number(exactUser.owner_id || 0),
      ownerName: exactUser.owner_state === "unavailable" ? "归属不可确认" : exactUser.owner_name || "",
    };
    contactBusinessInfoCache.set(userID, businessInfo);
    applyBusinessInfo(businessInfo);
  } catch (error) {
    console.warn("hydrate contact business info failed", error);
  }
}

async function onCurrentContactInfoUpdated(contactInfo: IContactInfoType) {
  if (isSameContactInfo(contactInfoData.value, contactInfo)) {
    return;
  }
  currentContactInfoRequestID += 1;
  const requestID = currentContactInfoRequestID;
  resetContactInfoUIData();
  // deep clone
  contactInfoData.value = deepCopy(contactInfo) || {};
  if (
    !contactInfoData.value ||
    Object.keys(contactInfoData.value)?.length === 0
  ) {
    return;
  }

  const currentUserID = (contactInfoData.value as Friend)?.userID || "";
  if (currentUserID) {
    const cachedBusinessInfo = contactBusinessInfoCache.get(currentUserID);
    if (cachedBusinessInfo) {
      contactInfoData.value = {
        ...contactInfoData.value,
        _wananID: cachedBusinessInfo.wananID,
        _phone: cachedBusinessInfo.phone,
        _nickname: cachedBusinessInfo.nickname,
        _ownerID: cachedBusinessInfo.ownerID,
        _ownerName: cachedBusinessInfo.ownerName,
      };
    }
  }

  contactInfoBasicList.value = generateContactInfoBasic(contactInfoData.value);
  void hydrateBusinessInfoForUser(currentUserID, requestID);

  isBothFriend.value = resolveLocalFriendState(contactInfoData.value);
  rebuildContactInfoPanels(contactInfoData.value);

  if (
    !currentUserID ||
    isGroup.value ||
    isApplication.value ||
    isBothFriend.value
  ) {
    return;
  }

  const latestFriendState = await isFriend(contactInfoData.value);
  if (requestID !== currentContactInfoRequestID) {
    return;
  }
  if (latestFriendState !== isBothFriend.value) {
    isBothFriend.value = latestFriendState;
    rebuildContactInfoPanels(contactInfoData.value);
  }
}
</script>
<style lang="scss" scoped src="./style/index.scss"></style>
