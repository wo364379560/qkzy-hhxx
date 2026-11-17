<template>
  <div ref="manageRef" class="manage">
    <header
      v-if="!isUniFrameWork || currentTab === 'admin'"
      class="manage-header"
    >
      <Icon :file="backSVG" @onClick="back" />
      <div class="manage-header-content">
        {{ TUITranslateService.t(`TUIGroup.${TabName}`) }}
      </div>
      <div />
    </header>
    <main
      v-if="!currentTab || (isUniFrameWork && currentTab != 'admin')"
      class="main"
    >
      <ManageName
        class="space-top"
        :isAuthor="isOwner || isAdmin || isWorkGroup"
        :data="currentGroup"
        @update="updateProfile"
      />
      <div class="user-info space-top">
        <header class="user-info-header" @click="setCurrentTab('member')">
          <label class="user-info-header-left">
            {{ TUITranslateService.t(`TUIGroup.群成员`) }}
          </label>
          <div class="user-info-header-right">
            <span class="span">
              {{ displayedMemberCount }}
              {{ TUITranslateService.t(`TUIGroup.人`) }}
            </span>
            <Icon :file="rightIcon" />
          </div>
        </header>
        <ol class="user-info-list">
          <dl
            v-for="(item, index) in groupMemberList.slice(0, showUserNum)"
            :key="item.userID || index"
            class="user-info-list-item"
          >
            <dt
              class="user-info-list-item-main"
              @click="handleMemberProfileShow(item)"
            >
              <img
                class="avatar"
                :src="
                  item.avatar ||
                  'https://web.sdk.qcloud.com/component/TUIKit/assets/avatar_21.png'
                "
                onerror="this.onerror=null;this.src='https://web.sdk.qcloud.com/component/TUIKit/assets/avatar_21.png'"
              />
            </dt>
            <dd class="user-info-list-item-info">
              {{ item.remark || item.friendRemark || item.nick || item.userID }}
            </dd>
          </dl>
          <dl v-if="isShowAddMember" class="user-info-list-item">
            <dt
              :class="[
                'avatar',
                'member-action-avatar',
                'member-action-avatar-add',
                transferOpening && 'member-action-avatar-disabled',
                transferOpeningType === 'add' && 'member-action-avatar-loading',
              ]"
              @click="toggleMask('add')"
            >
              <Icon
                :file="addCircleIcon"
                width="18px"
                height="18px"
                :hotAreaSize="0"
              />
            </dt>
          </dl>
          <dl v-if="canDeleteAnyMember" class="user-info-list-item">
            <dt
              :class="[
                'avatar',
                'member-action-avatar',
                'member-action-avatar-remove',
                transferOpening && 'member-action-avatar-disabled',
                transferOpeningType === 'remove' && 'member-action-avatar-loading',
              ]"
              @click="toggleMask('remove')"
            >
              <Icon
                :file="minusIcon"
                width="16px"
                height="16px"
                :hotAreaSize="0"
              />
            </dt>
          </dl>
        </ol>
      </div>
      <ul class="content list space-top" @click="editLableName = ''">
        <li class="list-item" @click="setCurrentTab('notification')">
          <aside class="aside">
            <label class="label">{{
              TUITranslateService.t(`TUIGroup.群公告`)
            }}</label>
            <article class="article">
              {{ groupNoticeContent || currentGroup?.notification }}
            </article>
          </aside>
          <Icon :file="rightIcon" class="end" />
        </li>
        <li
          v-if="canManageRegularMembers"
          class="list-item"
          @click="setCurrentTab('admin')"
        >
          <label class="label">{{
            TUITranslateService.t(`TUIGroup.群管理`)
          }}</label>
          <Icon :file="rightIcon" />
        </li>
        <li class="list-item list-item-switch">
          <label class="label">{{
            TUITranslateService.t(`TUIConversation.消息免打扰`)
          }}</label>
          <Slider
            :open="Boolean(currentConversation?.isMuted)"
            @change="toggleConversationMute"
          />
        </li>
        <li class="list-item list-item-switch">
          <label class="label">{{
            TUITranslateService.t(`TUIConversation.置顶会话`)
          }}</label>
          <Slider
            :open="Boolean(currentConversation?.isPinned)"
            @change="toggleConversationPin"
          />
        </li>
        <li class="list-item list-item-switch">
          <label class="label">{{
            TUITranslateService.t(`TUIGroup.显示群成员昵称`)
          }}</label>
          <Slider
            :open="showGroupMemberNickname"
            @change="toggleGroupMemberNickname"
          />
        </li>
        <li class="list-item">
          <label class="label">{{
            TUITranslateService.t(`TUIGroup.群ID`)
          }}</label>
          <div class="groupID">
            <span class="span">{{ currentGroupID }}</span>
          </div>
        </li>
        <li class="list-item">
          <label class="label">{{
            TUITranslateService.t(`TUIGroup.群头像`)
          }}</label>
          <div
            class="group-avatar-action"
            :class="[canEditGroupAvatar && 'list-item-clickable']"
            @click.stop="openGroupAvatarUploader"
          >
            <img
              class="avatar"
              :src="
                currentGroup?.avatar ||
                'https://web.sdk.qcloud.com/im/demo/TUIkit/web/img/constomer.svg'
              "
              onerror="this.onerror=null;this.src='https://web.sdk.qcloud.com/im/demo/TUIkit/web/img/constomer.svg'"
            />
            <span
              v-if="isOwner || isAdmin || isWorkGroup"
              class="group-avatar-tip"
            >
              {{
                TUITranslateService.t(
                  `TUIGroup.${groupAvatarUploading ? "上传中" : "点击上传"}`
                )
              }}
            </span>
            <input
              ref="groupAvatarInputRef"
              class="avatar-upload-input"
              type="file"
              accept="image/*"
              @change="handleGroupAvatarSelected"
            />
            <button v-if="pendingGroupAvatar" :disabled="groupAvatarUploading" @click.stop="retryGroupAvatarSync">重试资料同步</button>
          </div>
        </li>
        <li class="list-item">
          <label class="label">{{
            TUITranslateService.t(`TUIGroup.群类型`)
          }}</label>
          <span class="span">{{ getOptionLabel(currentGroup?.type) }}</span>
        </li>
        <li class="list-item">
          <label class="label">{{
            TUITranslateService.t(`TUIGroup.加群方式`)
          }}</label>
          <span class="span">{{
            getOptionLabel(currentGroup?.joinOption)
          }}</span>
        </li>
        <li
          class="list-item"
          :class="[canEditInviteOption && 'list-item-clickable']"
          @click="openInviteOptionDialog"
        >
          <label class="label">{{
            TUITranslateService.t(`TUIGroup.邀请方式`)
          }}</label>
          <span class="span">{{
            getInviteOptionLabel(currentInviteOption)
          }}</span>
        </li>
        <li class="list-item" @click="openSelfNameCardDialog">
          <label class="label">{{
            TUITranslateService.t(`TUIGroup.我在群里的备注`)
          }}</label>
          <span class="span">{{ selfNameCardDisplay }}</span>
        </li>
      </ul>
      <ul class="footer list space-top">
        <li
          v-if="currentGroupID"
          class="list-item"
          @click.stop="reportCurrentGroup"
        >
          举报群聊
        </li>
        <li
          v-if="groupConversationID"
          class="list-item"
          @click.stop="clearHistoryDialogShow = true"
        >
          {{ TUITranslateService.t(`TUIGroup.清空消息`) }}
        </li>
        <li
          v-if="currentSelfRole === 'Owner' && groupMemberList.length > 1"
          class="list-item"
          @click.stop="toggleMask('changeOwner')"
        >
          {{ TUITranslateService.t(`TUIGroup.转让群组`) }}
        </li>
        <li
          v-if="canIDissmissGroup"
          class="list-item"
          @click.stop="dismissGroup(currentGroup)"
        >
          {{ TUITranslateService.t(`TUIGroup.解散群聊`) }}
        </li>
        <li v-else class="list-item" @click.stop="quitGroup(currentGroup)">
          {{ TUITranslateService.t(`TUIGroup.退出群组`) }}
        </li>
      </ul>
    </main>
    <ManageMember
      v-if="currentTab === 'member'"
      :self="currentGroup?.selfInfo"
      :list="groupMemberList"
      :total="displayedMemberCount"
      :has-more="hasMoreGroupMembers"
      :isShowDel="canDeleteAnyMember"
      :isShowAdd="isShowAddMember"
      :display-online-status="displayOnlineStatus"
      :user-online-status-map="userOnlineStatusMap"
      :actionLoading="transferOpening || isManageSubmitting"
      :actionLoadingType="transferOpeningType"
      :memberSearchLoading="isMemberSearching"
      @more="getMember('more')"
      @del="submit"
      @add="toggleMask('add')"
      @search="searchGroupMemberByKeyword"
      @handleMemberProfileShow="handleMemberProfileShow"
      @close="setCurrentTab('')"
    />
    <ManageProfile
      v-if="currentTab === 'profile'"
      :userInfo="currentMember"
      @close="setCurrentTab('')"
    />
    <ManageNotification
      :canPin="isOwner || isAdmin"
      :pinned="groupNoticePinned"
      :saving="groupNoticeSaving"
      @pin="updateNoticePin"
      v-if="currentTab === 'notification'"
      :isAuthor="isOwner || isAdmin || isWorkGroup"
      :data="{ ...currentGroup, notification: groupNoticeContent || currentGroup?.notification }"
      @update="updateProfile"
      @close="setCurrentTab('')"
    />
    <ManageAdmin
      v-if="currentTab === 'admin'"
      v-show="canManageRegularMembers"
      :isSetMuteTime="isSetMuteTime"
      :member="member"
      :currentGroup="currentGroup"
      :memberProfileAccessEnabled="memberProfileAccessEnabled"
      :groupVoiceCallEnabled="groupVoiceCallEnabled"
      :groupVideoCallEnabled="groupVideoCallEnabled"
      :canManageAdmins="isOwner"
      :canManageGroupCalls="isOwner"
      :canManageGroupSettings="canManageTencentGroupSettings"
      :canManageSlowMode="canSetGroupSlowMode"
      :slowModeEnabled="groupSlowModeEnabled"
      :slowModeSeconds="groupSlowModeSeconds"
      :slowModeLoading="groupSlowModeLoading"
      :actionLoading="transferOpening || isManageSubmitting"
      :actionLoadingType="transferOpeningType"
      @addAdmin="toggleMask('addAdmin')"
      @removeAdmin="toggleMask('removeAdmin')"
      @setMemberProfileAccess="setMemberProfileAccess"
      @setGroupVoiceCallAccess="setGroupVoiceCallAccess"
      @setGroupVideoCallAccess="setGroupVideoCallAccess"
      @setSlowMode="setGroupSlowMode"
      @setAllMuteTime="setAllMuteTime"
      @addMute="toggleMask('addMute')"
      @removeMute="toggleMask('removeMute')"
      @blockedAction="showNoGroupSettingPermission"
      @close="setCurrentTab('')"
    />
    <MaskLayer :show="mask" @update:show="(e) => (mask = e)">
      <Transfer
        :title="TUITranslateService.t(`TUIGroup.${transferTitle}`)"
        :list="transferList"
        :total="!isCompleted ? transferTotal : transferList.length"
        :isSearch="isSearch"
        :isRadio="isRadio"
        :isCustomItem="transferType === 'addMute'"
        :selectedList="selectedList"
        :searchPlaceholder="transferSearchPlaceholder"
        :emptyText="transferEmptyText"
        :maxSelectable="transferMaxSelectable"
        :maxSelectionMessage="transferMaxSelectionMessage"
        :submitLoading="isManageSubmitting"
        :isH5="!isPC"
        @submit="submit"
        @cancel="cancel"
        @search="handleSearchMember"
        @getMore="getMember('more')"
      >
        <template #left="{ data }">
          <div class="mute-transfer-member">
            <img
              class="mute-transfer-member-avatar"
              :src="
                data.avatar ||
                'https://web.sdk.qcloud.com/component/TUIKit/assets/avatar_21.png'
              "
              onerror="this.onerror=null;this.src='https://web.sdk.qcloud.com/component/TUIKit/assets/avatar_21.png'"
            />
            <span class="mute-transfer-member-name">
              {{ data.remark || data.friendRemark || data.nick || data.userID }}
            </span>
            <span v-if="isAdminMember(data)" class="mute-transfer-member-badge">
              {{ TUITranslateService.t(`TUIGroup.管理员`) }}
            </span>
          </div>
        </template>
        <template #right="{ data }">
          <div class="mute-transfer-member mute-transfer-member-selected">
            <img
              class="mute-transfer-member-avatar"
              :src="
                data.avatar ||
                'https://web.sdk.qcloud.com/component/TUIKit/assets/avatar_21.png'
              "
              onerror="this.onerror=null;this.src='https://web.sdk.qcloud.com/component/TUIKit/assets/avatar_21.png'"
            />
            <span class="mute-transfer-member-name">
              {{ data.remark || data.friendRemark || data.nick || data.userID }}
            </span>
            <span v-if="isAdminMember(data)" class="mute-transfer-member-badge">
              {{ TUITranslateService.t(`TUIGroup.管理员`) }}
            </span>
          </div>
        </template>
      </Transfer>
    </MaskLayer>
    <Dialog
      class="deleted-dialog"
      :title="TUITranslateService.t(`TUIGroup.删除成员`)"
      :show="delDialogShow"
      :isH5="!isPC"
      :center="true"
      :isHeaderShow="isPC"
      :submitLoading="isManageSubmitting"
      :autoCloseOnSubmit="false"
      @submit="confirmRemoveMembers"
      @update:show="(e) => (delDialogShow = e)"
    >
      <p v-if="deletedUserList.length === 1" class="del-dialog-title">
        {{ TUITranslateService.t(`TUIGroup.确定从群聊中删除该成员？`) }}
      </p>
      <p v-if="deletedUserList.length > 1" class="del-dialog-title">
        {{ TUITranslateService.t(`TUIGroup.确定从群聊中删除所选成员？`) }}
      </p>
    </Dialog>
    <Dialog
      :show="inviteOptionDialogShow"
      :title="TUITranslateService.t(`TUIGroup.邀请方式`)"
      :isH5="!isPC"
      :center="true"
      :isHeaderShow="isPC"
      @submit="submitInviteOption"
      @update:show="(e) => (inviteOptionDialogShow = e)"
    >
      <div class="setting-option-list">
        <button
          v-for="item in inviteOptionChoices"
          :key="item.value"
          type="button"
          class="setting-option-item"
          :class="[
            inviteOptionDraft === item.value && 'setting-option-item--active',
          ]"
          @click="inviteOptionDraft = item.value"
        >
          {{ item.label }}
        </button>
      </div>
    </Dialog>
    <Dialog
      :show="selfNameCardDialogShow"
      :title="TUITranslateService.t(`TUIGroup.修改我的群昵称`)"
      :isH5="!isPC"
      :center="true"
      :isHeaderShow="isPC"
      @submit="submitSelfNameCard"
      @update:show="(e) => (selfNameCardDialogShow = e)"
    >
      <input
        v-model="selfNameCardDraft"
        class="setting-input"
        type="text"
        :placeholder="TUITranslateService.t(`TUIGroup.请输入群内备注`)"
      />
    </Dialog>
    <Dialog
      :show="clearHistoryDialogShow"
      :title="TUITranslateService.t(`TUIGroup.清空消息`)"
      :isH5="!isPC"
      :center="true"
      :isHeaderShow="isPC"
      @submit="clearGroupHistory"
      @update:show="(e) => (clearHistoryDialogShow = e)"
    >
      <p class="del-dialog-title">
        {{ TUITranslateService.t(`TUIChat.确认要清空当前的聊天记录吗？`) }}
      </p>
      <p class="setting-dialog-tip">
        {{ TUITranslateService.t(`TUIChat.清空后无法恢复`) }}
      </p>
    </Dialog>
  </div>
</template>
<script lang="ts" setup>
import { updatePresenceLease, releasePresenceLease } from '@/features/presence/subscriptions';
const onlinePresenceLease = Symbol('online-status');

import {
  ref,
  watch,
  computed,
  watchEffect,
  onMounted,
  onUnmounted,
  nextTick,
  defineAsyncComponent,
} from "../../../adapter-vue";
import TUIChatEngine, {
  TUITranslateService,
  TUIGroupService,
  TUIFriendService,
  TUIChatService,
  TUIUserService,
  TUIStore,
  StoreName,
  IGroupModel,
  TUIConversationService,
  IConversationModel,
  GetMemberListParams,
} from "@tencentcloud/chat-uikit-engine-lite";
import { TUIGlobal, outsideClick } from "@tencentcloud/universal-api";
import ManageName from "./manage-name.vue";
import Icon from "../../common/Icon.vue";
import Slider from "../../common/Slider/index.vue";
import addCircleIcon from "../../../assets/icon/add-circle.svg";
import minusIcon from "../../../assets/icon/minus.svg";
import backSVG from "../../../assets/icon/back.svg";
import rightIcon from "../../../assets/icon/right-icon.svg";
import { Toast, TOAST_TYPE } from "../../common/Toast/index";
import { isPC, isUniFrameWork } from "../../../utils/env";
import Server from "../server";
import { enableSampleTaskStatus } from "../../../utils/enableSampleTaskStatus";
import { IFriendProfile, IGroupMember } from "../../../interface";
import type { IUserStatus, IUserStatusMap } from "../../../interface";
import { openReportDialog } from "@/features/report/dialog";
import { uploadAndSyncGroupAvatar, syncUploadedGroupAvatar } from "@/features/group/profile";
import { AvatarSyncError } from '@/features/account/avatarSync';
import {
  addGroupAdmin,
  addGroupMembers,
  filterGroupMembers,
  getConversationCustomPins,
  getGroupAdmins,
  getGroupSlowMode,
  muteGroupMember,
  pinConversationCustom,
  removeGroupAdmin,
  removeGroupMember,
  saveGroupSlowMode,
  unpinConversationCustom,
  unmuteGroupMember,
  type GroupAdminMember,
} from "@/api";
import {
  CONVERSATION_LOCAL_PINS_UPDATED_KEY,
  getConversationCustomPin,
} from "@/features/conversation/localPins";
import { announceGroupNoticeUpdate } from "@/features/group/notice";
import { getGroupNotice, saveGroupNotice, pinGroupNotice } from "@/api";
import {
  GROUP_MEMBER_PROFILE_ACCESS_ATTRIBUTE,
  GROUP_VIDEO_CALL_ATTRIBUTE,
  GROUP_VOICE_CALL_ATTRIBUTE,
  isGroupVideoCallEnabled,
  isGroupVoiceCallEnabled,
  isMemberProfileAccessEnabled,
  serializeGroupCallAccess,
  serializeMemberProfileAccess,
} from "@/features/group/policies";
import {
  isGroupMemberNicknameEnabled,
  setGroupMemberNicknameEnabled,
} from "@/features/group/member-nickname";
import { matchesMemberKeyword } from "../../../utils/member-search";
import { sortByFirstChar } from "../../TUIContact/utils/sortByFirstChar";
import { withBusy } from "../../common/BusyOverlay/index";

const MaskLayer = defineAsyncComponent(() => import("../../common/MaskLayer/index.vue"));
const Dialog = defineAsyncComponent(() => import("../../common/Dialog/index.vue"));
const Transfer = defineAsyncComponent(() => import("../../common/Transfer/index.vue"));
const ManageNotification = defineAsyncComponent(() => import("./manage-notification.vue"));
const ManageMember = defineAsyncComponent(() => import("./manage-member.vue"));
const ManageProfile = defineAsyncComponent(() => import("./manage-profile.vue"));
const ManageAdmin = defineAsyncComponent(() => import("./manage-admin.vue"));

const TUIGroupServer = Server.getInstance();
const TUIConstants = TUIGroupServer.constants;
const DEFAULT_GROUP_MEMBER_LIMIT = 2000;
const GROUP_ADD_MEMBER_BATCH_SIZE = 20;

const props = defineProps({
  groupID: {
    type: String,
    default: "",
  },
  groupCurrentTab: {
    type: String,
    default: "",
  },
});

const manageRef = ref<any>(undefined);
// type 'profile' | 'member' | 'notification' | 'admin' | ''
const currentTab = ref("");
const editLableName = ref("");
const transferType = ref("");
const mask = ref(false);
const currentGroupID = ref<string>("");
const currentMember = ref<IGroupMember>({});
const typeName = ref({
  [TUIChatEngine.TYPES.GRP_WORK]: "好友工作群",
  [TUIChatEngine.TYPES.GRP_PUBLIC]: "普通群组",
  [TUIChatEngine.TYPES.GRP_MEETING]: "临时会议群",
  [TUIChatEngine.TYPES.GRP_AVCHATROOM]: "直播群",
  [TUIChatEngine.TYPES.GRP_COMMUNITY]: "社群",
  [TUIChatEngine.TYPES.JOIN_OPTIONS_FREE_ACCESS]: "自由加入",
  [TUIChatEngine.TYPES.JOIN_OPTIONS_NEED_PERMISSION]: "需要验证",
  [TUIChatEngine.TYPES.JOIN_OPTIONS_DISABLE_APPLY]: "禁止加群",
});
const member = ref({
  admin: [] as IGroupMember[],
  member: [] as IGroupMember[],
  muteMember: [] as IGroupMember[],
});
const mergedAdminUserIDSet = ref<Set<string>>(new Set());
const friendList = ref<IFriendProfile[]>([]);
const transferList = ref<IGroupMember[]>([]);
const displayOnlineStatus = ref(false);
const userOnlineStatusMap = ref<IUserStatusMap>({});
const transferTotal = ref<number>(0);
const transferTitle = ref("");
const isSearch = ref(false);
const isRadio = ref(false);
const isTransferLoading = ref(false);
const transferOpening = ref(false);
const transferOpeningType = ref("");
const isManageSubmitting = ref(false);
const selectedList = ref([]);
const delDialogShow = ref(false);
const groupMemberList = ref<IGroupMember[]>([]);
const deletedUserList = ref([]);
const currentGroup = ref<IGroupModel>();
const groupNoticeContent = ref("");
const groupNoticePinned = ref(false);
const groupNoticeSaving = ref(false);
const currentConversation = ref<IConversationModel>();
const currentGroupAttributes = ref<Record<string, string>>({});
const currentGroupAttributesGroupID = ref("");
const currentSelfRole = ref("");
const groupIDValue = ref<string>("");
const groupMemberOffset = ref<number | string>(0);
const isCompleted = ref(false);
const syncedMemberCountGroupID = ref("");
const isLoadingCurrentGroupMembers = ref(false);
const isMemberSearching = ref(false);
let memberSearchSeq = 0;
const groupAvatarInputRef = ref<HTMLInputElement | null>(null);
const groupAvatarUploading = ref(false);
const pendingGroupAvatar = ref<{ groupID: string; owner: string; url: string } | null>(null);
let groupAvatarGeneration = 0;
let groupAvatarAbort: AbortController | null = null;
onUnmounted(() => { ++groupAvatarGeneration; groupAvatarAbort?.abort(); });
const inviteOptionDialogShow = ref(false);
const selfNameCardDialogShow = ref(false);
const clearHistoryDialogShow = ref(false);
const inviteOptionDraft = ref("");
const selfNameCardDraft = ref("");
const showGroupMemberNickname = ref(false);
const groupSlowModeEnabled = ref(false);
const groupSlowModeSeconds = ref(30);
const canSetGroupSlowMode = ref(false);
const groupSlowModeLoading = ref(false);
let subscribedGroupMemberUserIDs: string[] = [];

const isCurrentStoreGroup = () => {
  const storeGroupID = TUIStore.getData(StoreName.GRP, "currentGroupID") || "";
  return Boolean(currentGroupID.value && storeGroupID === currentGroupID.value);
};

const getCurrentGroupScopedAttributes = () => {
  if (!currentGroupID.value) {
    return {};
  }
  return currentGroupAttributesGroupID.value === currentGroupID.value
    ? currentGroupAttributes.value
    : {};
};

const memberProfileAccessEnabled = computed(() =>
  isMemberProfileAccessEnabled(getCurrentGroupScopedAttributes())
);
const groupVoiceCallEnabled = computed(() =>
  isGroupVoiceCallEnabled(getCurrentGroupScopedAttributes())
);
const groupVideoCallEnabled = computed(() =>
  isGroupVideoCallEnabled(getCurrentGroupScopedAttributes())
);
const groupConversationID = computed(() => {
  if (!currentGroupID.value) {
    return "";
  }
  return `${TUIChatEngine.TYPES.CONV_GROUP}${currentGroupID.value}`;
});
const currentInviteOption = computed(() => {
  return (
    (currentGroup.value as any)?.inviteOption ||
    currentGroup.value?.joinOption ||
    TUIChatEngine.TYPES.JOIN_OPTIONS_FREE_ACCESS
  );
});
const canEditInviteOption = computed(() => isOwner.value || isAdmin.value);
const selfNameCardDisplay = computed(() => {
  return currentGroup.value?.selfInfo?.nameCard || "-";
});
const currentUserID = computed(() => {
  return TUIStore.getData(StoreName.USER, "userProfile")?.userID || "";
});
const inviteOptionChoices = computed(() => [
  {
    value: TUIChatEngine.TYPES.JOIN_OPTIONS_DISABLE_APPLY,
    label: TUITranslateService.t("TUIGroup.禁止邀请"),
  },
  {
    value: TUIChatEngine.TYPES.JOIN_OPTIONS_NEED_PERMISSION,
    label: TUITranslateService.t("TUIGroup.需要验证"),
  },
  {
    value: TUIChatEngine.TYPES.JOIN_OPTIONS_FREE_ACCESS,
    label: TUITranslateService.t("TUIGroup.自由加入"),
  },
]);

const canEditGroupAvatar = computed(() => {
  return Boolean(
    (isOwner.value || isAdmin.value || isWorkGroup.value) &&
      currentGroupID.value &&
      !groupAvatarUploading.value
  );
});
const groupAddMemberLimit = computed(() => {
  if (currentGroup.value?.type === TUIChatEngine.TYPES.GRP_COMMUNITY) {
    return Number.POSITIVE_INFINITY;
  }
  return DEFAULT_GROUP_MEMBER_LIMIT;
});
const displayedMemberCount = computed(() => {
  const loadedCount = groupMemberList.value.length;
  const canUseSyncedProfileCount =
    syncedMemberCountGroupID.value === currentGroupID.value;
  const profileCount = canUseSyncedProfileCount
    ? currentGroup.value?.memberCount || 0
    : 0;
  if (canUseSyncedProfileCount && profileCount > 0) {
    return Math.max(profileCount, loadedCount);
  }
  if (isCompleted.value && loadedCount > 0) {
    return loadedCount;
  }
  return Math.max(profileCount, loadedCount);
});
const hasMoreGroupMembers = computed(() => {
  if (isCompleted.value) {
    return false;
  }
  const nextOffset = String(groupMemberOffset.value || "");
  if (nextOffset && nextOffset !== "0") {
    return true;
  }
  return groupMemberList.value.length < displayedMemberCount.value;
});
const remainingAddMemberSlots = computed(() => {
  const currentCount = Math.max(
    currentGroup.value?.memberCount || 0,
    groupMemberList.value.length
  );
  if (!Number.isFinite(groupAddMemberLimit.value)) {
    return Number.POSITIVE_INFINITY;
  }
  return Math.max(groupAddMemberLimit.value - currentCount, 0);
});
const transferMaxSelectable = computed(() => {
  if (transferType.value === "add") {
    return remainingAddMemberSlots.value;
  }
  return -1;
});
const transferMaxSelectionMessage = computed(() => {
  if (transferType.value !== "add") {
    return "";
  }
  if (!Number.isFinite(remainingAddMemberSlots.value)) {
    return "";
  }
  if (remainingAddMemberSlots.value > 0) {
    return `本次最多还能添加 ${remainingAddMemberSlots.value} 人`;
  }
  return "群已满员";
});
const transferEmptyText = computed(() => {
  return isTransferLoading.value
    ? TUITranslateService.t("TUIChat.正在加载")
    : TUITranslateService.t("TUISearch.暂无搜索结果");
});
const transferSearchPlaceholder = computed(() => {
  if (transferType.value === "add") {
    return TUITranslateService.t("TUIGroup.搜索备注、昵称或userID");
  }
  return TUITranslateService.t("TUIGroup.搜索备注、昵称、群昵称或userID");
});
const searchEnabledTransferTypes = new Set([
  "add",
  "remove",
  "addAdmin",
  "removeAdmin",
  "changeOwner",
  "addMute",
  "removeMute",
]);

const getGroupMemberDisplayName = (member: IGroupMember) =>
  member?.remark ||
  member?.friendRemark ||
  member?.nick ||
  member?.nameCard ||
  member?.userID ||
  "";

const getGroupMemberRolePriority = (member: IGroupMember) => {
  switch (member?.role) {
    case TUIChatEngine.TYPES.GRP_MBR_ROLE_OWNER:
      return 0;
    case TUIChatEngine.TYPES.GRP_MBR_ROLE_ADMIN:
      return 1;
    case TUIChatEngine.TYPES.GRP_MBR_ROLE_MEMBER:
      return 2;
    default:
      return 3;
  }
};

const isPrivilegedGroupRole = (role?: string) =>
  role === TUIChatEngine.TYPES.GRP_MBR_ROLE_OWNER ||
  role === TUIChatEngine.TYPES.GRP_MBR_ROLE_ADMIN;

const canCurrentUserOpenProtectedMemberProfile = () =>
  isPrivilegedGroupRole(currentSelfRole.value || currentGroup.value?.selfInfo?.role);

const canOpenProtectedGroupMemberProfile = (member?: IGroupMember) => {
  if (!member?.userID) {
    return false;
  }

  const selfUserID = currentUserID.value || currentGroup.value?.selfInfo?.userID;
  if (member.userID === selfUserID) {
    return true;
  }

  if (memberProfileAccessEnabled.value) {
    return true;
  }

  return canCurrentUserOpenProtectedMemberProfile();
};

const sortGroupMembers = (memberList: IGroupMember[] = []) => {
  const roleBuckets = memberList.reduce<Record<number, IGroupMember[]>>(
    (result, member) => {
      const priority = getGroupMemberRolePriority(member);
      if (!result[priority]) {
        result[priority] = [];
      }
      result[priority].push(member);
      return result;
    },
    {}
  );

  return [0, 1, 2, 3].flatMap((priority) =>
    sortByFirstChar(
      roleBuckets[priority] || [],
      getGroupMemberDisplayName,
      false
    ).sortedList
  );
};

const rebuildMemberBuckets = (memberList: IGroupMember[]) => {
  member.value = {
    admin: [],
    member: [],
    muteMember: [],
  };
  Array.from(memberList).map((item: any) => {
    switch (item?.role) {
      case TUIChatEngine.TYPES.GRP_MBR_ROLE_ADMIN:
        member.value.admin.push(item);
        break;
      case TUIChatEngine.TYPES.GRP_MBR_ROLE_MEMBER:
        member.value.member.push(item);
        break;
      default:
        break;
    }
    return item;
  });
  const time: number = new Date().getTime();
  member.value.muteMember = Array.from(memberList).filter(
    (item: any) => item?.muteUntil * 1000 - time > 0
  );
};

const toGroupMemberFromAdmin = (
  admin: GroupAdminMember,
  existing?: IGroupMember
): IGroupMember => ({
  ...(existing || {}),
  userID: admin.user_id,
  nick: admin.nickname || existing?.nick || existing?.nameCard || admin.user_id,
  avatar: admin.avatar_url || existing?.avatar,
  role: TUIChatEngine.TYPES.GRP_MBR_ROLE_ADMIN,
});

const mergeBackendGroupAdmins = async (groupID: string) => {
  if (!groupID) {
    mergedAdminUserIDSet.value = new Set();
    return;
  }

  const adminResult = await getGroupAdmins(groupID);
  if (groupID !== currentGroupID.value) {
    return;
  }

  const nextAdminIDs = new Set<string>();
  const currentMemberMap = new Map<string, IGroupMember>();
  groupMemberList.value.forEach((item) => {
    if (item.userID) {
      currentMemberMap.set(item.userID, item);
    }
  });
  const adminMap = new Map<string, IGroupMember>();
  member.value.admin.forEach((item) => {
    if (item.userID) {
      adminMap.set(item.userID, item);
      nextAdminIDs.add(item.userID);
    }
  });

  (adminResult?.admins || []).forEach((admin) => {
    const userID = admin.user_id;
    if (!userID) {
      return;
    }
    nextAdminIDs.add(userID);
    adminMap.set(
      userID,
      toGroupMemberFromAdmin(
        admin,
        adminMap.get(userID) || currentMemberMap.get(userID)
      )
    );
  });

  member.value.admin = sortGroupMembers(Array.from(adminMap.values()));
  member.value.member = member.value.member.filter(
    (item) => !item.userID || !nextAdminIDs.has(item.userID)
  );
  mergedAdminUserIDSet.value = nextAdminIDs;
};

const getResponseMemberList = (imResponse: any): IGroupMember[] => {
  const data = imResponse?.data || {};
  if (Array.isArray(data.memberList)) {
    return data.memberList;
  }
  if (Array.isArray(data.groupMemberList)) {
    return data.groupMemberList;
  }
  if (Array.isArray(data)) {
    return data;
  }
  return [];
};

const applyCurrentGroupMemberList = async (memberList: IGroupMember[] = []) => {
  const nextGroupMemberList = sortGroupMembers(
    enrichGroupMembersWithFriendRemark(memberList)
  );
  groupMemberList.value = nextGroupMemberList;
  rebuildMemberBuckets(nextGroupMemberList);
  if (currentGroupID.value) {
    try {
      await mergeBackendGroupAdmins(currentGroupID.value);
    } catch (_) {
      mergedAdminUserIDSet.value = new Set(
        member.value.admin
          .map((item) => item.userID || "")
          .filter((userID) => Boolean(userID))
      );
    }
  }
  void syncGroupMemberOnlineStatus(nextGroupMemberList);
  if (mask.value) {
    applyTransferSourceList(transferType.value);
    isTransferLoading.value = false;
  }
};

function syncCurrentGroupState(patch: Record<string, any> = {}) {
  if (!currentGroup.value) {
    return;
  }
  currentGroup.value = {
    ...currentGroup.value,
    ...patch,
  } as IGroupModel;
  currentSelfRole.value =
    currentGroup.value?.selfInfo?.role || currentSelfRole.value || "";
  TUIStore.update(StoreName.GRP, "currentGroup", currentGroup.value);
}

const applyTransferSourceList = (type?: string, sourceList?: IGroupMember[]) => {
  const nextSourceList = sourceList || resolveTransferSourceListSync(type);
  transferList.value = nextSourceList;

  switch (type) {
    case "remove":
    case "changeOwner":
      transferTotal.value = Math.max(displayedMemberCount.value - 1, 0);
      break;
    case "addAdmin":
    case "removeAdmin":
      transferTotal.value = nextSourceList.length;
      break;
    case "add":
    case "addMute":
    case "removeMute":
    default:
      transferTotal.value = nextSourceList.length;
      break;
  }
};

function syncGroupUserStatusStateFromStore() {
  const onlineStatusEnabled = TUIStore.getData(
    StoreName.USER,
    "displayOnlineStatus"
  );
  if (typeof onlineStatusEnabled === "boolean") {
    displayOnlineStatus.value = onlineStatusEnabled;
  }

  const userStatusList = TUIStore.getData(StoreName.USER, "userStatusList");
  if (userStatusList instanceof Map) {
    onUserStatusListUpdated(userStatusList as Map<string, IUserStatus>);
  }
}

function onDisplayOnlineStatusUpdated(status: boolean) {
  displayOnlineStatus.value = status;
  if (!status) {
    userOnlineStatusMap.value = {};
    void unsubscribeGroupMemberOnlineStatus(subscribedGroupMemberUserIDs);
    subscribedGroupMemberUserIDs = [];
    return;
  }
  void syncGroupMemberOnlineStatus(groupMemberList.value);
}

function onUserStatusListUpdated(list: Map<string, IUserStatus>) {
  if (!list || list.size === 0) {
    userOnlineStatusMap.value = {};
    return;
  }

  userOnlineStatusMap.value = [...list.entries()].reduce(
    (result, [userID, status]) => {
      result[userID] = status;
      return result;
    },
    {} as IUserStatusMap
  );
}

function batchUserIDs(userIDList: string[]) {
  const batches: string[][] = [];
  for (let index = 0; index < userIDList.length; index += 100) {
    batches.push(userIDList.slice(index, index + 100));
  }
  return batches;
}

async function unsubscribeGroupMemberOnlineStatus(_userIDList: string[]) {
  await releasePresenceLease(onlinePresenceLease);
}

async function syncGroupMemberOnlineStatus(memberList: IGroupMember[] = []) {
  if (!displayOnlineStatus.value) { await releasePresenceLease(onlinePresenceLease); return; }
  const ids = [...new Set(memberList.map(item => String(item?.userID || '').trim()).filter(Boolean))];
  subscribedGroupMemberUserIDs = ids;
  await updatePresenceLease(onlinePresenceLease, ids, rows => {
    const next = { ...userOnlineStatusMap.value };
    rows.forEach(status => { next[status.userID] = { statusType: Number(status.statusType || 0), customStatus: String(status.customStatus || '') }; });
    userOnlineStatusMap.value = next;
  });
}

function reportCurrentGroup() {
  if (!currentGroupID.value) {
    return;
  }
  openReportDialog({
    targetId: currentGroupID.value,
    targetType: "group",
    targetName: currentGroup.value?.name || currentGroupID.value,
  });
}

function openGroupAvatarUploader() {
  if (!canEditGroupAvatar.value) {
    return;
  }
  groupAvatarInputRef.value?.click();
}

async function handleGroupAvatarSelected(event: Event) {
  const target = event.target as HTMLInputElement;
  const file = target.files?.[0];
  if (!file || !currentGroupID.value) {
    target.value = "";
    return;
  }

  groupAvatarUploading.value = true;
  const groupID = currentGroupID.value;
  const owner = currentUserID.value;
  const generation = ++groupAvatarGeneration;
  const controller = new AbortController();
  groupAvatarAbort?.abort();
  groupAvatarAbort = controller;
  pendingGroupAvatar.value = null;
  const active = () => generation === groupAvatarGeneration && groupID === currentGroupID.value && owner === currentUserID.value;
  try {
    const avatarURL = await uploadAndSyncGroupAvatar(
      groupID,
      file, active, controller.signal
    );
    if (!active()) return;
    if (currentGroup.value) {
      currentGroup.value.avatar = avatarURL;
    }
    Toast({
      message: TUITranslateService.t("TUIGroup.群头像修改成功"),
      type: TOAST_TYPE.SUCCESS,
    });
  } catch (error: any) {
    if (!active() || controller.signal.aborted) return;
    if (error instanceof AvatarSyncError) pendingGroupAvatar.value = { groupID, owner, url: error.avatarURL };
    const errorSuffix = error?.message ? `: ${error.message}` : "";
    Toast({
      message: `${TUITranslateService.t(
        "TUIGroup.群头像修改失败"
      )}${errorSuffix}`,
      type: TOAST_TYPE.ERROR,
    });
  } finally {
    if (active()) { groupAvatarUploading.value = false; groupAvatarAbort = null; }
    target.value = "";
  }
}

async function retryGroupAvatarSync() {
  const pending = pendingGroupAvatar.value;
  if (!pending) return;
  const generation = ++groupAvatarGeneration;
  const active = () => generation === groupAvatarGeneration && pending.groupID === currentGroupID.value && pending.owner === currentUserID.value;
  if (!active()) { pendingGroupAvatar.value = null; return; }
  groupAvatarUploading.value = true;
  try {
    await syncUploadedGroupAvatar(pending.groupID, pending.url, active);
    if (!active()) return;
    if (currentGroup.value) currentGroup.value.avatar = pending.url;
    pendingGroupAvatar.value = null;
    Toast({ message: '群头像资料已同步', type: TOAST_TYPE.SUCCESS });
  } catch (error: any) { if (active()) Toast({ message: error.message, type: TOAST_TYPE.ERROR }); }
  finally { if (active()) groupAvatarUploading.value = false; }
}

function resetGroupSlowModeState() {
  groupSlowModeEnabled.value = false;
  groupSlowModeSeconds.value = 30;
  canSetGroupSlowMode.value = false;
  groupSlowModeLoading.value = false;
}

async function loadGroupSlowMode(groupID: string) {
  if (!groupID) {
    resetGroupSlowModeState();
    return;
  }
  try {
    const result = await getGroupSlowMode(groupID);
    if (groupID !== currentGroupID.value) {
      return;
    }
    groupSlowModeEnabled.value = Boolean(result?.slow_mode_enabled);
    groupSlowModeSeconds.value = Number(result?.slow_mode_seconds || 30);
    canSetGroupSlowMode.value = Boolean(result?.can_set_slow_mode);
  } catch (error) {
    if (groupID !== currentGroupID.value) {
      return;
    }
    resetGroupSlowModeState();
    console.warn("loadGroupSlowMode failed:", error);
  }
}

async function setGroupSlowMode(payload: { enabled: boolean; seconds: number }) {
  const groupID = currentGroupID.value;
  if (!groupID || !canSetGroupSlowMode.value || groupSlowModeLoading.value) {
    return;
  }
  const seconds = Math.min(Math.max(Number(payload.seconds || 30), 5), 3600);
  const previousEnabled = groupSlowModeEnabled.value;
  const previousSeconds = groupSlowModeSeconds.value;
  groupSlowModeEnabled.value = Boolean(payload.enabled);
  groupSlowModeSeconds.value = seconds;
  groupSlowModeLoading.value = true;
  try {
    const result = await saveGroupSlowMode(groupID, {
      enabled: Boolean(payload.enabled),
      seconds,
    });
    if (groupID !== currentGroupID.value) {
      return;
    }
    groupSlowModeEnabled.value = Boolean(result?.slow_mode_enabled);
    groupSlowModeSeconds.value = Number(result?.slow_mode_seconds || seconds);
    canSetGroupSlowMode.value = Boolean(result?.can_set_slow_mode);
    Toast({
      message: groupSlowModeEnabled.value
        ? "发言频率限制已开启"
        : "发言频率限制已关闭",
      type: TOAST_TYPE.SUCCESS,
    });
  } catch (error: any) {
    groupSlowModeEnabled.value = previousEnabled;
    groupSlowModeSeconds.value = previousSeconds;
    Toast({
      message: `发言频率限制设置失败${error?.message ? `：${error.message}` : ""}`,
      type: TOAST_TYPE.ERROR,
    });
    await loadGroupSlowMode(groupID);
  } finally {
    if (groupID === currentGroupID.value) {
      groupSlowModeLoading.value = false;
    }
  }
}

const onGroupStoreUpdated = {
  currentGroup: (group: IGroupModel) => {
    const groupID = (group as any)?.groupID || "";
    if (groupID && groupID === currentGroupID.value) {
      currentGroup.value = group;
      currentSelfRole.value = currentGroup.value?.selfInfo?.role || "";
    }
  },
  currentGroupAttributes: (groupAttributes: Record<string, string>) => {
    if (!isCurrentStoreGroup()) {
      return;
    }
    currentGroupAttributes.value = groupAttributes || {};
    currentGroupAttributesGroupID.value = currentGroupID.value;
  },
  currentGroupMemberList: (memberList: IGroupMember[]) => {
    if (!isCurrentStoreGroup() || isLoadingCurrentGroupMembers.value) {
      return;
    }
    void applyCurrentGroupMemberList(memberList);
  },
  offset: (offset: number | string) => {
    if (!isCurrentStoreGroup()) {
      return;
    }
    groupMemberOffset.value = offset;
  },
  isCompleted: (data: boolean) => {
    if (!isCurrentStoreGroup()) {
      return;
    }
    isCompleted.value = data;
  },
};

const onConversationStoreUpdated = {
  currentConversation: (conversation: IConversationModel) => {
    groupIDValue.value = conversation?.groupProfile?.groupID;
    if (conversation?.conversationID === groupConversationID.value) {
      currentConversation.value = conversation;
      void syncCurrentConversationLocalPin();
    }
  },
};

const onCustomStoreUpdated = {
  [CONVERSATION_LOCAL_PINS_UPDATED_KEY]: () => {
    void syncCurrentConversationLocalPin();
  },
};

onMounted(() => {
  nextTick(() => {
    if (manageRef.value && !isUniFrameWork) {
      outsideClick.listen({
        domRefs: manageRef.value,
        handler: handleCompleteManage,
      });
    }
  });
  TUIStore.watch(StoreName.USER, {
    displayOnlineStatus: onDisplayOnlineStatusUpdated,
    userStatusList: onUserStatusListUpdated,
  });
  TUIStore.watch(StoreName.GRP, onGroupStoreUpdated);
  TUIStore.watch(StoreName.CONV, onConversationStoreUpdated);
  TUIStore.watch(StoreName.CUSTOM, onCustomStoreUpdated);
  syncGroupUserStatusStateFromStore();
});

onUnmounted(() => {
  TUIStore.unwatch(StoreName.USER, {
    displayOnlineStatus: onDisplayOnlineStatusUpdated,
    userStatusList: onUserStatusListUpdated,
  });
  TUIStore.unwatch(StoreName.GRP, onGroupStoreUpdated);
  TUIStore.unwatch(StoreName.CONV, onConversationStoreUpdated);
  TUIStore.unwatch(StoreName.CUSTOM, onCustomStoreUpdated);
  void releasePresenceLease(onlinePresenceLease);
});

watchEffect(() => {
  const params = TUIGroupServer.getOnCallParams(
    TUIConstants.TUIGroup.SERVICE.METHOD.OPEN_GROUP_MANAGEMENT
  );
  currentGroupID.value = groupIDValue.value || params?.groupID || "";
  currentTab.value = props.groupCurrentTab;
});

watch(
  () => currentGroupID.value,
  async (groupID) => {
    ++groupAvatarGeneration;
    groupAvatarAbort?.abort();
    groupAvatarAbort = null;
    pendingGroupAvatar.value = null;
    groupAvatarUploading.value = false;
    groupMemberList.value = [];
    rebuildMemberBuckets([]);
    groupMemberOffset.value = 0;
    isCompleted.value = false;
    syncedMemberCountGroupID.value = "";

    if (!groupID) {
      currentGroupAttributes.value = {};
      currentGroupAttributesGroupID.value = "";
      currentConversation.value = undefined;
      currentGroup.value = undefined;
      currentSelfRole.value = "";
      showGroupMemberNickname.value = false;
      resetGroupSlowModeState();
      return;
    }
    currentGroup.value = undefined;
    currentSelfRole.value = "";
    currentGroupAttributes.value = {};
    currentGroupAttributesGroupID.value = groupID;
    resetGroupSlowModeState();
    await Promise.all([
      loadGroupAttributes(groupID),
      loadConversationProfile(groupID),
      loadGroupSlowMode(groupID),
    ]);
    if (groupID !== currentGroupID.value) {
      return;
    }
    await refreshGroupMembersState(false, true);
    if (groupID !== currentGroupID.value) {
      return;
    }
    showGroupMemberNickname.value = isGroupMemberNicknameEnabled(
      groupID,
      currentUserID.value
    );
    if (!friendList.value.length) {
      void ensureFriendListLoaded();
    }
  },
  {
    immediate: true,
  }
);

const TabName = computed(() => {
  let name = "";
  switch (currentTab.value) {
    case "notification":
      name = "群公告";
      break;
    case "member":
      name = "群成员";
      break;
    case "profile":
      name = "群成员";
      break;
    case "admin":
      name = "群管理";
      break;
    default:
      name = "群聊详情";
      break;
  }
  return name;
});

const isOwner = computed(() => {
  // Determine whether the person is the group owner/administrator
  const userRole = currentGroup.value?.selfInfo?.role;
  return userRole === TUIChatEngine.TYPES.GRP_MBR_ROLE_OWNER;
});

const isAdmin = computed(() => {
  const userRole = currentGroup.value?.selfInfo?.role;
  return userRole === TUIChatEngine.TYPES.GRP_MBR_ROLE_ADMIN;
});

const canManageTencentGroupSettings = computed(() => {
  return isOwner.value || isAdmin.value;
});

const isBusinessAdmin = computed(() => {
  const selfUserID =
    currentGroup.value?.selfInfo?.userID || currentUserID.value || "";
  return Boolean(selfUserID && mergedAdminUserIDSet.value.has(selfUserID));
});

const canManageRegularMembers = computed(() => {
  return isOwner.value || isAdmin.value || isBusinessAdmin.value;
});

const isWorkGroup = computed(() => {
  return currentGroup.value?.type === TUIChatEngine.TYPES.GRP_WORK;
});

const isSetMuteTime = computed(() => {
  if (isWorkGroup.value || !canManageRegularMembers.value) {
    return false;
  }
  return true;
});

const canDelMember = computed(() => {
  const groupType = currentGroup?.value?.type;
  const isAVChatRoom = groupType === TUIChatEngine.TYPES.GRP_AVCHATROOM;
  if (isAVChatRoom) {
    return false;
  }
  return true;
});

const canDeleteGroupMember = (memberItem: IGroupMember) => {
  if (!memberItem?.userID || memberItem.userID === currentUserID.value) {
    return false;
  }

  if (isOwner.value) {
    return memberItem.role !== TUIChatEngine.TYPES.GRP_MBR_ROLE_OWNER;
  }

  if (isAdmin.value || isBusinessAdmin.value) {
    return (
      memberItem.role === TUIChatEngine.TYPES.GRP_MBR_ROLE_MEMBER &&
      !mergedAdminUserIDSet.value.has(memberItem.userID)
    );
  }

  return false;
};

const canDeleteAnyMember = computed(() => {
  if (!canDelMember.value) {
    return false;
  }

  return groupMemberList.value.some((memberItem) => canDeleteGroupMember(memberItem));
});

function getOptionLabel(option?: string) {
  if (!option) {
    return "-";
  }
  const key = (typeName.value as Record<string, string>)[option];
  return key ? TUITranslateService.t(`TUIGroup.${key}`) : option;
}

function getInviteOptionLabel(option?: string) {
  if (option === TUIChatEngine.TYPES.JOIN_OPTIONS_DISABLE_APPLY) {
    return TUITranslateService.t("TUIGroup.禁止邀请");
  }
  if (option === TUIChatEngine.TYPES.JOIN_OPTIONS_NEED_PERMISSION) {
    return TUITranslateService.t("TUIGroup.需要验证");
  }
  if (option === TUIChatEngine.TYPES.JOIN_OPTIONS_FREE_ACCESS) {
    return TUITranslateService.t("TUIGroup.自由加入");
  }
  return "-";
}

const showNoGroupSettingPermission = () => {
  Toast({
    message: "当前账户暂无权限操作",
    type: TOAST_TYPE.WARNING,
  });
};

async function updateNoticePin(pinned: boolean) {
  const id = currentGroup.value?.groupID;
  if (!id || groupNoticeSaving.value || !(isOwner.value || isAdmin.value)) return;
  const fallbackContent = groupNoticeContent.value || currentGroup.value?.notification || "";
  groupNoticeSaving.value = true;
  try {
    const stored = await getGroupNotice(id);
    if (stored.source === "fallback") {
      await saveGroupNotice(id, fallbackContent);
    }
    const notice = await pinGroupNotice(id, pinned);
    announceGroupNoticeUpdate(id, notice);
    if (id === currentGroup.value?.groupID) groupNoticePinned.value = notice.pinned;
  } catch (error: any) {
    Toast({ message: error?.message || "公告置顶设置失败", type: TOAST_TYPE.ERROR });
  } finally { groupNoticeSaving.value = false; }
}

const updateProfile = async (newGroupProfile: any) => {
  if (!currentGroup.value?.groupID) {
    return;
  }
  const id = currentGroup.value.groupID;
  const { key, value } = newGroupProfile;
  if (key === "notification") {
    try {
      const notice = await saveGroupNotice(id, value);
      announceGroupNoticeUpdate(id, notice);
      await TUIGroupService.updateGroupProfile({
        groupID: id,
        notification: notice.summary,
      });
      if (id !== currentGroup.value?.groupID) return true;
      groupNoticePinned.value = Boolean(notice.pinned);
      groupNoticeContent.value = notice.content;
      currentGroup.value = {
        ...currentGroup.value,
        notification: notice.summary,
      };
      editLableName.value = "";
      return true;
    } catch (error: any) {
      Toast({
        message: error?.message,
        type: TOAST_TYPE.ERROR,
      });
      return false;
    }
  }
  const options: any = {
    groupID: currentGroup.value.groupID,
    [key]: value,
  };
  try {
    const res = await TUIGroupService.updateGroupProfile(options);
    currentGroup.value = res.data.group;
    editLableName.value = "";
    return true;
  } catch (error: any) {
    Toast({
      message: error?.message,
      type: TOAST_TYPE.ERROR,
    });
    return false;
  }
};

const setCurrentTab = (tabName: string) => {
  currentTab.value = tabName;
  editLableName.value = "";
  if (currentTab.value === "member") {
    transferType.value = "remove";
  }
  if (!currentTab.value) {
    transferType.value = "";
  }
  if (tabName === "admin") {
    getAllGroupMember({
      groupID: currentGroupID.value,
      count: 100,
      offset: groupMemberOffset.value,
    });
  }
};

const cancel = () => {
  toggleMask();
};

const resolveTransferSourceListSync = (type?: string) => {
  switch (type) {
    case "add":
      return transferList.value;
    case "remove":
      return enrichGroupMembersWithFriendRemark(groupMemberList.value).filter((item: IGroupMember) =>
        canDeleteGroupMember(item)
      );
    case "changeOwner":
      return enrichGroupMembersWithFriendRemark(groupMemberList.value).filter(
        (item: IGroupMember) =>
          item.userID !== currentGroup?.value?.selfInfo?.userID
      );
    case "addAdmin":
      return enrichGroupMembersWithFriendRemark(member.value.member).filter(
        (item: IGroupMember) =>
          !item.userID || !mergedAdminUserIDSet.value.has(item.userID)
      );
    case "removeAdmin":
      return enrichGroupMembersWithFriendRemark(member.value.admin);
    case "addMute":
      return getMuteCandidateMembers();
    case "removeMute":
      return enrichGroupMembersWithFriendRemark(member.value.muteMember);
    default:
      return transferList.value;
  }
};

const toggleMask = async (type?: string) => {
  if (!type) {
    mask.value = false;
    isTransferLoading.value = false;
    return;
  }
  if (transferOpening.value) {
    return;
  }

  if (type === "add" && remainingAddMemberSlots.value <= 0) {
    Toast({
      message: transferMaxSelectionMessage.value,
      type: TOAST_TYPE.WARNING,
    });
    return;
  }
  if ((type === "addAdmin" || type === "removeAdmin") && !isOwner.value) {
    Toast({
      message: "仅群主可以设置管理员",
      type: TOAST_TYPE.WARNING,
    });
    return;
  }
  switch (type) {
    case "add":
      isRadio.value = false;
      transferTitle.value = "添加成员";
      break;
    case "remove":
      isRadio.value = false;
      transferTitle.value = "删除成员";
      break;
    case "addAdmin":
      isRadio.value = false;
      transferTitle.value = "新增管理员";
      break;
    case "removeAdmin":
      isRadio.value = true;
      transferTitle.value = "移除管理员";
      break;
    case "changeOwner":
      isRadio.value = true;
      transferTitle.value = "转让群组";
      break;
    case "addMute":
      isRadio.value = true;
      transferTitle.value = "新增禁言用户";
      break;
    case "removeMute":
      isRadio.value = true;
      transferTitle.value = "移除禁言用户";
      break;
    default:
      break;
  }
  isSearch.value = type ? searchEnabledTransferTypes.has(type) : false;
  selectedList.value = [];
  type && (transferType.value = type);
  transferOpening.value = true;
  transferOpeningType.value = type;
  isTransferLoading.value = true;
  try {
    await updateTransferList(type);
    mask.value = true;
  } catch (error: any) {
    mask.value = false;
    Toast({
      message: error?.message || "成员列表加载失败，请稍后重试",
      type: TOAST_TYPE.ERROR,
    });
  } finally {
    isTransferLoading.value = false;
    transferOpening.value = false;
    transferOpeningType.value = "";
  }
};

const resolveTransferSourceList = async (type?: string) => {
  switch (type) {
    case "add":
      return await getFriendsNotInGroup();
    case "remove":
    case "changeOwner":
      await ensureFriendListLoaded();
      if (type === "remove") {
        return enrichGroupMembersWithFriendRemark(groupMemberList.value).filter((item: IGroupMember) =>
          canDeleteGroupMember(item)
        );
      }
      return enrichGroupMembersWithFriendRemark(groupMemberList.value).filter(
        (item: IGroupMember) =>
          item.userID !== currentGroup?.value?.selfInfo?.userID
      );
    case "addAdmin":
      await ensureFriendListLoaded();
      return enrichGroupMembersWithFriendRemark(member.value.member).filter(
        (item: IGroupMember) =>
          !item.userID || !mergedAdminUserIDSet.value.has(item.userID)
      );
    case "removeAdmin":
      await ensureFriendListLoaded();
      return enrichGroupMembersWithFriendRemark(member.value.admin);
    case "addMute":
      await ensureFriendListLoaded();
      return getMuteCandidateMembers();
    case "removeMute":
      await ensureFriendListLoaded();
      return enrichGroupMembersWithFriendRemark(member.value.muteMember);
    default:
      return transferList.value;
  }
};

const updateTransferList = async (type?: string) => {
  const shouldLoadAllMembers = ["remove", "changeOwner", "addMute", "addAdmin"].includes(
    type || ""
  );

  if (shouldLoadAllMembers) {
    await refreshGroupMembersState(true, true);
  } else if (type === "add") {
    await ensureFriendListLoaded();
  } else if (type === "removeAdmin" || type === "removeMute") {
    await ensureFriendListLoaded();
  }

  const sourceList = await resolveTransferSourceList(type);
  applyTransferSourceList(type, sourceList);
};

const getMuteCandidateMembers = () => {
  const currentUserID = currentGroup?.value?.selfInfo?.userID;
  const currentUserRole = currentGroup.value?.selfInfo?.role;

  let candidates = member.value.member as Array<IGroupMember>;
  if (currentUserRole === TUIChatEngine.TYPES.GRP_MBR_ROLE_OWNER) {
    // Owner can individually mute both regular members and admins.
    candidates = enrichGroupMembersWithFriendRemark(groupMemberList.value).filter(
      (item: IGroupMember) => item.userID !== currentUserID
    );
  }

  return enrichGroupMembersWithFriendRemark(candidates).filter(
    (item: IGroupMember) => member.value.muteMember.indexOf(item) < 0
  );
};

const isAdminMember = (memberItem: IGroupMember) =>
  memberItem.role === TUIChatEngine.TYPES.GRP_MBR_ROLE_ADMIN;

const normalizeFriendListItem = (item: any) => ({
  ...item,
  ...item?.profile,
  userID: item?.userID || item?.profile?.userID || "",
  nick: item?.profile?.nick || item?.nick || "",
  avatar: item?.profile?.avatar || item?.avatar || "",
  remark: item?.remark || "",
  friendRemark: item?.friendRemark || item?.remark || "",
});

async function ensureFriendListLoaded() {
  if (friendList.value.length === 0) {
    const imResponse = await TUIFriendService.getFriendList();
    friendList.value = imResponse.data.map(normalizeFriendListItem);
    const nextGroupMemberList = sortGroupMembers(
      enrichGroupMembersWithFriendRemark(groupMemberList.value)
    );
    groupMemberList.value = nextGroupMemberList;
    rebuildMemberBuckets(nextGroupMemberList);
    if (currentGroupID.value) {
      await mergeBackendGroupAdmins(currentGroupID.value);
    }
    if (mask.value && transferType.value !== "add") {
      applyTransferSourceList(transferType.value);
    }
  }
  return friendList.value;
}

const enrichGroupMembersWithFriendRemark = (memberList: IGroupMember[] = []) => {
  if (!memberList.length || !friendList.value.length) {
    return memberList;
  }

  const friendInfoMap = new Map(
    friendList.value.map((item: any) => [item.userID, item])
  );

  return memberList.map((member) => {
    const matchedFriend = friendInfoMap.get(member.userID || "");
    if (!matchedFriend) {
      return member;
    }
    return {
      ...member,
      avatar: member.avatar || matchedFriend.avatar,
      nick: member.nick || matchedFriend.nick,
      remark: matchedFriend.remark || "",
      friendRemark: matchedFriend.friendRemark || matchedFriend.remark || "",
    };
  });
};

const getFriendsNotInGroup = async () => {
  await ensureFriendListLoaded();
  const groupID = currentGroupID.value;
  const friendUserIDs = normalizeUserIDList(
    friendList.value.map((item: any) => item.userID)
  );
  if (!groupID || friendUserIDs.length === 0) {
    return [];
  }
  const result = await filterGroupMembers(groupID, friendUserIDs);
  if (groupID !== currentGroupID.value) {
    return [];
  }
  const notInGroupSet = new Set(result.not_in_group || []);
  return friendList.value.filter((item: any) => notInGroupSet.has(item.userID));
};

const canIDissmissGroup = computed(() => {
  const userRole = currentGroup?.value?.selfInfo?.role;
  const groupType = currentGroup?.value?.type;

  return (
    userRole === TUIChatEngine.TYPES.GRP_MBR_ROLE_OWNER &&
    groupType !== TUIChatEngine.TYPES.GRP_WORK
  );
});

const isShowAddMember = computed(() => {
  const groupType = currentGroup?.value?.type;
  if (!groupType) {
    return false;
  }
  if (groupType === TUIChatEngine.TYPES.GRP_AVCHATROOM) {
    return false;
  }
  if (
    currentInviteOption.value === TUIChatEngine.TYPES.JOIN_OPTIONS_DISABLE_APPLY
  ) {
    return false;
  }
  return (
    canManageRegularMembers.value ||
    groupType === TUIChatEngine.TYPES.GRP_WORK
  );
});

const showUserNum = computed(() => {
  let num = 3;
  if (!isShowAddMember.value) {
    num += 1;
  }
  if (currentGroup?.value?.selfInfo?.role !== "Owner") {
    num += 1;
  }
  return num;
});

const getAllGroupMember = async (
  options: GetMemberListParams,
  force = false
): Promise<IGroupMember[]> => {
  const isNotCurrentGroup = options.groupID !== currentGroupID.value;
  const isNotAdminTab = !force && currentTab.value !== "admin";
  const isAlreadyCompleted = !force && isCompleted.value;
  const isOverMemberLimit =
    !force && groupMemberList.value.length >= DEFAULT_GROUP_MEMBER_LIMIT;
  if (
    isNotCurrentGroup ||
    isAlreadyCompleted ||
    isNotAdminTab ||
    isOverMemberLimit
  ) {
    return [];
  }
  try {
    const imResponse = await TUIGroupService.getGroupMemberList(options);
    const memberList = getResponseMemberList(imResponse);
    const { offset = 0 } = imResponse.data || {};
    if (options.groupID !== currentGroupID.value) {
      return [];
    }
    if (offset) {
      const nextMemberList = await getAllGroupMember(
        {
          groupID: currentGroupID.value,
          count: 100,
          offset: offset,
        },
        force
      );
      return [...memberList, ...nextMemberList];
    }
    return memberList;
  } catch (error: any) {
    console.error("getGroupMemberList error:", error);
    return [];
  }
};

const getMember = async (type?: string) => {
  const groupID = currentGroupID.value;
  const options = {
    groupID,
    count: 100,
    offset: type && type === "more" ? groupMemberOffset.value : 0,
  };
  isLoadingCurrentGroupMembers.value = true;
  try {
    const imResponse = await TUIGroupService.getGroupMemberList(options);
    if (groupID !== currentGroupID.value) {
      return;
    }
    const memberList = getResponseMemberList(imResponse);
    const nextMemberList =
      type === "more" ? [...groupMemberList.value, ...memberList] : memberList;
    await applyCurrentGroupMemberList(nextMemberList);
    const nextOffset = imResponse?.data?.offset || 0;
    groupMemberOffset.value = nextOffset;
    isCompleted.value = !nextOffset;
  } finally {
    window.setTimeout(() => {
      if (groupID === currentGroupID.value) {
        isLoadingCurrentGroupMembers.value = false;
      }
    }, 0);
  }
};

const hasMatchedGroupMember = (keyword: string) =>
  groupMemberList.value.some((member) =>
    matchesMemberKeyword(member as Record<string, any>, keyword)
  );

const hasNextGroupMemberPage = (offset: number | string) => {
  const normalizedOffset = String(offset || "");
  return Boolean(normalizedOffset && normalizedOffset !== "0");
};

const mergeGroupMemberPage = (
  currentList: IGroupMember[] = [],
  nextPage: IGroupMember[] = []
) => {
  const merged = new Map<string, IGroupMember>();
  [...currentList, ...nextPage].forEach((memberItem) => {
    const userID = String(memberItem?.userID || "").trim();
    if (!userID) {
      return;
    }
    merged.set(userID, {
      ...(merged.get(userID) || {}),
      ...memberItem,
    });
  });
  return Array.from(merged.values());
};

const searchGroupMemberByKeyword = async (keyword: string) => {
  const normalizedKeyword = keyword.trim();
  const searchSeq = ++memberSearchSeq;
  if (!normalizedKeyword) {
    isMemberSearching.value = false;
    return;
  }
  if (hasMatchedGroupMember(normalizedKeyword) || isCompleted.value) {
    return;
  }

  const groupID = currentGroupID.value;
  isMemberSearching.value = true;
  try {
    while (
      searchSeq === memberSearchSeq &&
      groupID === currentGroupID.value &&
      !isCompleted.value &&
      hasNextGroupMemberPage(groupMemberOffset.value)
    ) {
      const memberResponse = await TUIGroupService.getGroupMemberList({
        groupID,
        count: 100,
        offset: groupMemberOffset.value,
      });
      if (searchSeq !== memberSearchSeq || groupID !== currentGroupID.value) {
        return;
      }

      const pageMemberList = getResponseMemberList(memberResponse);
      const nextMemberList = mergeGroupMemberPage(
        groupMemberList.value,
        pageMemberList
      );
      await applyCurrentGroupMemberList(nextMemberList);

      const nextOffset = memberResponse?.data?.offset || 0;
      groupMemberOffset.value = nextOffset;
      isCompleted.value = !hasNextGroupMemberPage(nextOffset);

      if (hasMatchedGroupMember(normalizedKeyword)) {
        return;
      }
    }
  } finally {
    if (searchSeq === memberSearchSeq && groupID === currentGroupID.value) {
      isMemberSearching.value = false;
    }
  }
};

async function loadConversationProfile(groupID: string) {
  try {
    const res = await TUIConversationService.getConversationProfile(
      `${TUIChatEngine.TYPES.CONV_GROUP}${groupID}`
    );
    if (groupID !== currentGroupID.value) return;
    currentConversation.value = res?.data?.conversation;
    await syncCurrentConversationLocalPin();
  } catch (error) {
    currentConversation.value = undefined;
  }

  try {
    const notice = await getGroupNotice(groupID);
    if (groupID !== currentGroupID.value) return;
    groupNoticePinned.value = Boolean(notice?.pinned);
    groupNoticeContent.value = notice?.content || "";
  } catch (_) {
    if (groupID !== currentGroupID.value) return;
    groupNoticeContent.value = "";
    groupNoticePinned.value = false;
  }
}

async function syncCurrentConversationLocalPin() {
  const conversationID = groupConversationID.value;
  if (!conversationID || !currentConversation.value) {
    return;
  }
  try {
    const pin = getConversationCustomPin(
      await getConversationCustomPins(),
      conversationID
    );
    if (conversationID !== groupConversationID.value || !currentConversation.value) {
      return;
    }
    (currentConversation.value as any).isPinned = Boolean(pin);
    (currentConversation.value as any).__localPinSortOrder = pin?.pinOrder || 0;
  } catch (_) {
    if (currentConversation.value) {
      (currentConversation.value as any).isPinned = false;
      (currentConversation.value as any).__localPinSortOrder = 0;
    }
  }
}

async function refreshGroupMembersState(
  loadAll = currentTab.value === "admin",
  forceAll = false,
  excludedUserIDList: any = [],
  maxMemberCount?: number,
  minMemberCount?: number,
  optimisticUserIDList: string[] = []
) {
  if (!currentGroupID.value) {
    return;
  }

  if (forceAll) {
    isCompleted.value = false;
    groupMemberOffset.value = 0;
  }

  const groupID = currentGroupID.value;
  const groupResponse = await TUIGroupService.getGroupProfile({ groupID });
  if (groupID !== currentGroupID.value) {
    return;
  }
  if (groupResponse?.data?.group) {
    currentGroup.value = groupResponse.data.group;
    currentSelfRole.value = groupResponse.data.group?.selfInfo?.role;
    syncedMemberCountGroupID.value = groupID;
    clampGroupMemberCount(maxMemberCount);
    liftGroupMemberCount(minMemberCount);
  }
  try {
    const notice = await getGroupNotice(groupID);
    if (groupID !== currentGroupID.value) {
      return;
    }
    groupNoticeContent.value = notice?.content || "";
  } catch (_) {
    if (groupID !== currentGroupID.value) {
      return;
    }
    groupNoticeContent.value = "";
  }

  isLoadingCurrentGroupMembers.value = true;
  try {
    const memberResponse = await TUIGroupService.getGroupMemberList({
      groupID,
      count: 100,
      offset: 0,
    });
    if (groupID !== currentGroupID.value) {
      return;
    }
    const excludedUserIDSet = new Set(normalizeUserIDList(excludedUserIDList));
    const filterExcludedMembers = (memberList: IGroupMember[] = []) =>
      excludedUserIDSet.size
        ? memberList.filter((item) => !excludedUserIDSet.has(item?.userID || ""))
        : memberList;
    let nextMemberList = filterExcludedMembers(getResponseMemberList(memberResponse));
    const nextOffset = memberResponse?.data?.offset || 0;
    if (loadAll && nextOffset) {
      const restMemberList = await getAllGroupMember(
        {
          groupID,
          count: 100,
          offset: nextOffset,
        },
        forceAll
      );
      if (groupID !== currentGroupID.value) {
        return;
      }
      nextMemberList = [...nextMemberList, ...filterExcludedMembers(restMemberList)];
    }
    nextMemberList = mergeOptimisticGroupMembers(nextMemberList, optimisticUserIDList);
    await applyCurrentGroupMemberList(nextMemberList);
    liftGroupMemberCount(minMemberCount);
    groupMemberOffset.value = loadAll ? 0 : nextOffset;
    isCompleted.value = loadAll ? true : !nextOffset;
  } finally {
    window.setTimeout(() => {
      if (groupID === currentGroupID.value) {
        isLoadingCurrentGroupMembers.value = false;
      }
    }, 0);
  }
}

const handleMemberProfileShow = (user: any) => {
  if (!canOpenProtectedGroupMemberProfile(user)) {
    Toast({
      message: TUITranslateService.t("TUIGroup.该群已关闭成员资料查看"),
      type: TOAST_TYPE.WARNING,
    });
    return;
  }
  currentMember.value = user;
  setCurrentTab("profile");
};

async function loadGroupAttributes(groupID: string) {
  const requestedGroupID = groupID;
  try {
    const res = await TUIGroupService.getGroupAttributes({
      groupID: requestedGroupID,
      keyList: [
        GROUP_MEMBER_PROFILE_ACCESS_ATTRIBUTE,
        GROUP_VOICE_CALL_ATTRIBUTE,
        GROUP_VIDEO_CALL_ATTRIBUTE,
      ],
    });
    if (requestedGroupID !== currentGroupID.value) {
      return;
    }
    currentGroupAttributes.value = res?.data?.groupAttributes || {};
    currentGroupAttributesGroupID.value = requestedGroupID;
    TUIStore.update(StoreName.GRP, "currentGroupID", requestedGroupID);
    TUIStore.update(StoreName.GRP, "currentGroupAttributes", currentGroupAttributes.value);
  } catch (error) {
    if (requestedGroupID !== currentGroupID.value) {
      return;
    }
    currentGroupAttributes.value = {};
    currentGroupAttributesGroupID.value = requestedGroupID;
    TUIStore.update(StoreName.GRP, "currentGroupID", requestedGroupID);
    TUIStore.update(StoreName.GRP, "currentGroupAttributes", {});
  }
}

async function setMemberProfileAccess(value: boolean) {
  if (!canManageTencentGroupSettings.value) {
    showNoGroupSettingPermission();
    return;
  }
  if (!currentGroupID.value) {
    return;
  }

  try {
    await TUIGroupService.setGroupAttributes({
      groupID: currentGroupID.value,
      groupAttributes: {
        [GROUP_MEMBER_PROFILE_ACCESS_ATTRIBUTE]:
          serializeMemberProfileAccess(value),
      },
    });
    const nextAttributes = {
      ...getCurrentGroupScopedAttributes(),
      [GROUP_MEMBER_PROFILE_ACCESS_ATTRIBUTE]:
        serializeMemberProfileAccess(value),
    };
    currentGroupAttributes.value = nextAttributes;
    currentGroupAttributesGroupID.value = currentGroupID.value;
    TUIStore.update(StoreName.GRP, "currentGroupID", currentGroupID.value);
    TUIStore.update(StoreName.GRP, "currentGroupAttributes", nextAttributes);
    Toast({
      message: TUITranslateService.t(
        value
          ? "TUIGroup.已开启群成员资料查看"
          : "TUIGroup.已关闭群成员资料查看"
      ),
      type: TOAST_TYPE.SUCCESS,
    });
  } catch (error: any) {
    Toast({
      message:
        error?.message ||
        TUITranslateService.t("TUIGroup.成员资料查看设置失败"),
      type: TOAST_TYPE.ERROR,
    });
  }
}

async function setGroupCallAttribute(attributeKey: string, enabled: boolean) {
  if (!currentGroupID.value) {
    return;
  }

  const nextAttributes = {
    ...getCurrentGroupScopedAttributes(),
    [attributeKey]: serializeGroupCallAccess(enabled),
  };

  await TUIGroupService.setGroupAttributes({
    groupID: currentGroupID.value,
    groupAttributes: {
      [attributeKey]: serializeGroupCallAccess(enabled),
    },
  });

  currentGroupAttributes.value = nextAttributes;
  currentGroupAttributesGroupID.value = currentGroupID.value;
  TUIStore.update(StoreName.GRP, "currentGroupID", currentGroupID.value);
  TUIStore.update(StoreName.GRP, "currentGroupAttributes", nextAttributes);
}

async function setGroupVoiceCallAccess(value: boolean) {
  if (!isOwner.value) {
    showNoGroupSettingPermission();
    return;
  }
  try {
    await setGroupCallAttribute(GROUP_VOICE_CALL_ATTRIBUTE, value);
    Toast({
      message: TUITranslateService.t(
        value ? "TUIGroup.群语音已开启" : "TUIGroup.群语音已关闭"
      ),
      type: TOAST_TYPE.SUCCESS,
    });
  } catch (error: any) {
    Toast({
      message:
        error?.message || TUITranslateService.t("TUIGroup.群语音设置失败"),
      type: TOAST_TYPE.ERROR,
    });
  }
}

async function setGroupVideoCallAccess(value: boolean) {
  if (!isOwner.value) {
    showNoGroupSettingPermission();
    return;
  }
  try {
    await setGroupCallAttribute(GROUP_VIDEO_CALL_ATTRIBUTE, value);
    Toast({
      message: TUITranslateService.t(
        value ? "TUIGroup.群视频已开启" : "TUIGroup.群视频已关闭"
      ),
      type: TOAST_TYPE.SUCCESS,
    });
  } catch (error: any) {
    Toast({
      message:
        error?.message || TUITranslateService.t("TUIGroup.群视频设置失败"),
      type: TOAST_TYPE.ERROR,
    });
  }
}

async function toggleConversationMute(nextValue: boolean) {
  if (!currentGroupID.value) {
    return;
  }
  if (Boolean(currentConversation.value?.isMuted) === nextValue) {
    return;
  }
  try {
    await TUIConversationService.muteConversation({
      groupID: currentGroupID.value,
      messageRemindType: nextValue
        ? TUIChatEngine.TYPES.MSG_REMIND_ACPT_NOT_NOTE
        : TUIChatEngine.TYPES.MSG_REMIND_ACPT_AND_NOTE,
    });
    await loadConversationProfile(currentGroupID.value);
  } catch (error: any) {
    Toast({
      message:
        error?.message || TUITranslateService.t("TUIConversation.消息免打扰"),
      type: TOAST_TYPE.ERROR,
    });
  }
}

function toggleGroupMemberNickname(nextValue: boolean) {
  if (!currentGroupID.value) {
    return;
  }
  showGroupMemberNickname.value = nextValue;
  setGroupMemberNicknameEnabled(
    currentGroupID.value,
    currentUserID.value,
    nextValue
  );
}

async function toggleConversationPin(nextValue: boolean) {
  if (!groupConversationID.value) {
    return;
  }
  if (Boolean(currentConversation.value?.isPinned) === nextValue) {
    return;
  }
  try {
    const pinOrder = Date.now();
    if (nextValue) {
      await pinConversationCustom({
        conversationID: groupConversationID.value,
        pinOrder,
      });
    } else {
      await unpinConversationCustom({
        conversationID: groupConversationID.value,
      });
    }
    if (currentConversation.value) {
      (currentConversation.value as any).isPinned = nextValue;
      (currentConversation.value as any).__localPinSortOrder = nextValue
        ? pinOrder
        : 0;
    }
    TUIStore.update(StoreName.CUSTOM, CONVERSATION_LOCAL_PINS_UPDATED_KEY, Date.now());
  } catch (error: any) {
    Toast({
      message:
        error?.message || "置顶失败，请稍后重试",
      type: TOAST_TYPE.ERROR,
    });
  }
}

function openInviteOptionDialog() {
  if (!canEditInviteOption.value) {
    return;
  }
  inviteOptionDraft.value = currentInviteOption.value;
  inviteOptionDialogShow.value = true;
}

async function submitInviteOption() {
  if (!currentGroupID.value) {
    return;
  }
  try {
    const res = await TUIGroupService.updateGroupProfile({
      groupID: currentGroupID.value,
      inviteOption: inviteOptionDraft.value as any,
    });
    currentGroup.value = res?.data?.group || currentGroup.value;
    Toast({
      message: TUITranslateService.t("TUIGroup.邀请方式更新成功"),
      type: TOAST_TYPE.SUCCESS,
    });
  } catch (error: any) {
    Toast({
      message:
        error?.message || TUITranslateService.t("TUIGroup.邀请方式更新失败"),
      type: TOAST_TYPE.ERROR,
    });
  }
}

function openSelfNameCardDialog() {
  selfNameCardDraft.value = currentGroup.value?.selfInfo?.nameCard || "";
  selfNameCardDialogShow.value = true;
}

async function submitSelfNameCard() {
  if (!currentGroupID.value) {
    return;
  }
  try {
    await TUIGroupService.setGroupMemberNameCard({
      groupID: currentGroupID.value,
      nameCard: selfNameCardDraft.value,
    });
    if (currentGroup.value?.selfInfo) {
      currentGroup.value.selfInfo.nameCard = selfNameCardDraft.value;
    }
    await refreshGroupMembersState(false);
    Toast({
      message: TUITranslateService.t("TUIGroup.群内备注更新成功"),
      type: TOAST_TYPE.SUCCESS,
    });
  } catch (error: any) {
    Toast({
      message:
        error?.message || TUITranslateService.t("TUIGroup.群内备注更新失败"),
      type: TOAST_TYPE.ERROR,
    });
  }
}

async function clearGroupHistory() {
  if (!groupConversationID.value) {
    return;
  }
  try {
    await TUIChatService.clearHistoryMessage(groupConversationID.value);
    Toast({
      message: TUITranslateService.t("TUIGroup.消息已清空"),
      type: TOAST_TYPE.SUCCESS,
    });
  } catch (error: any) {
    Toast({
      message: error?.message || TUITranslateService.t("TUIGroup.清空消息"),
      type: TOAST_TYPE.ERROR,
    });
  }
}

const submit = async (userList: any) => {
  if (isManageSubmitting.value) {
    return;
  }
  if (transferType.value === "remove") {
    deletedUserList.value = userList;
    delDialogShow.value = !delDialogShow.value;
    mask.value = false;
    return;
  }

  isManageSubmitting.value = true;
  try {
    const success = await handleManage(userList, transferType.value);
    if (success) {
      selectedList.value = [];
      mask.value = false;
    }
  } finally {
    isManageSubmitting.value = false;
  }
};

const confirmRemoveMembers = async () => {
  if (isManageSubmitting.value) {
    return;
  }
  isManageSubmitting.value = true;
  try {
    const success = await handleManage(deletedUserList.value, "remove");
    if (success) {
      delDialogShow.value = false;
      deletedUserList.value = [];
    }
  } finally {
    isManageSubmitting.value = false;
  }
};

const dismissGroup = async (group: any) => {
  await TUIGroupService.dismissGroup(group.groupID);
  enableSampleTaskStatus("dismissGroup");
  Toast({
    message: TUITranslateService.t("TUIGroup.群组解散成功"),
    type: TOAST_TYPE.SUCCESS,
  });
  clearGroupInfo();
};

const clearGroupInfo = () => {
  if (isUniFrameWork) {
    TUIGlobal?.switchTab({
      url: "/TUIKit/components/TUIConversation/index",
    });
  } else {
    handleCompleteManage();
    TUIConversationService.switchConversation("");
  }
};

const setAllMuteTime = async (value: boolean) => {
  if (!canManageTencentGroupSettings.value) {
    showNoGroupSettingPermission();
    return;
  }
  // Set up a muting time for all members
  const success = await updateProfile({ key: "muteAllMembers", value });
  if (!success) {
    return;
  }
  if (value) {
    enableSampleTaskStatus("muteGroup");
    Toast({
      message: TUITranslateService.t("TUIGroup.禁言设置成功"),
      type: TOAST_TYPE.SUCCESS,
    });
  } else {
    Toast({
      message: TUITranslateService.t("TUIGroup.取消禁言成功"),
      type: TOAST_TYPE.SUCCESS,
    });
  }
};

const handleSearchMember = async (value: string) => {
  const keyword = value.trim();
  const sourceList = await resolveTransferSourceList(transferType.value);
  if (!keyword) {
    transferList.value = sourceList;
    transferTotal.value = sourceList.length;
    return;
  }

  const filteredList = (sourceList as Array<Record<string, any>>).filter((item: any) =>
    matchesMemberKeyword(item, keyword)
  );
  transferList.value = filteredList;
  transferTotal.value = filteredList.length;
};

type AdminBatchResult = {
  successCount: number;
  failedCount: number;
  firstErrorMessage: string;
};

const getAdminActionLabel = (mode: "add" | "remove") =>
  mode === "add" ? "添加管理员" : "移除管理员";

const getAdminSuccessMessage = (mode: "add" | "remove", count: number) => {
  const label = getAdminActionLabel(mode);
  if (count <= 1) {
    return `${label}成功`;
  }
  return `成功${mode === "add" ? "添加" : "移除"} ${count} 个管理员`;
};

const getAdminFailureMessage = (
  mode: "add" | "remove",
  result: AdminBatchResult
) => {
  const label = getAdminActionLabel(mode);
  const reason = result.firstErrorMessage ? `：${result.firstErrorMessage}` : "";
  if (result.successCount > 0 && result.failedCount > 0) {
    return `成功${mode === "add" ? "添加" : "移除"} ${result.successCount} 个，失败 ${result.failedCount} 个${reason}`;
  }
  return `${label}失败${reason}`;
};

const showAdminBatchResultToast = (
  mode: "add" | "remove",
  result: AdminBatchResult
) => {
  if (result.failedCount > 0) {
    Toast({
      message: getAdminFailureMessage(mode, result),
      type: result.successCount > 0 ? TOAST_TYPE.WARNING : TOAST_TYPE.ERROR,
    });
    return result.successCount > 0;
  }

  if (result.successCount === 0) {
    Toast({
      message: "请选择要操作的用户",
      type: TOAST_TYPE.WARNING,
    });
    return false;
  }

  Toast({
    message: getAdminSuccessMessage(mode, result.successCount),
    type: TOAST_TYPE.SUCCESS,
  });
  return true;
};

const handleManage = async (userList: any, type: any): Promise<boolean> => {
  const busyText =
    type === "add"
      ? "正在拉人进群..."
      : type === "remove"
        ? "正在移出成员..."
        : type === "addAdmin" || type === "removeAdmin"
          ? "正在更新管理员..."
          : type === "changeOwner"
            ? "正在转让群主..."
            : "处理中...";

  return withBusy(busyText, async () => {
  const managedUserList = Array.isArray(userList) ? userList : [];
  const userIDList: any = [];
  managedUserList.map((item: any) => {
    userIDList.push(item.userID);
    return item;
  });
  try {
    switch (type) {
      case "add":
        await addMember(userIDList);
        break;
      case "remove":
        await deleteGroupMember(userIDList);
        break;
      case "addAdmin":
        if (userIDList.length === 0) {
          Toast({
            message: "请选择要添加的管理员",
            type: TOAST_TYPE.WARNING,
          });
          return false;
        }
        return showAdminBatchResultToast(
          "add",
          await handleAdminBatch(managedUserList, "add")
        );
      case "removeAdmin":
        if (userIDList.length === 0) {
          Toast({
            message: "请选择要移除的管理员",
            type: TOAST_TYPE.WARNING,
          });
          return false;
        }
        return showAdminBatchResultToast(
          "remove",
          await handleAdminBatch(managedUserList, "remove")
        );
      case "changeOwner":
        await changeOwner(userIDList[0]);
        break;
      case "addMute":
        await setMemberMuteTime(userIDList[0], "add");
        break;
      case "removeMute":
        await setMemberMuteTime(userIDList[0], "remove");
        break;
      default:
        break;
    }
  } catch (error: any) {
    Toast({
      message: resolveManageGroupErrorMessage(error, type),
      type: TOAST_TYPE.ERROR,
    });
    return false;
  }
  return true;
  });
};

const resolveManageGroupErrorMessage = (error: any, type: string) => {
  const rawMessage = String(error?.message || "").trim();
  const normalizedMessage = rawMessage.toLowerCase();

  if (
    rawMessage.includes('"code":10016') ||
    normalizedMessage.includes("before callback failed and refuse")
  ) {
    if (type === "add") {
      return "当前处于系统静默时段或被系统规则拒绝，暂时不能邀请成员入群，请稍后再试";
    }
    return "当前处于系统静默时段或被系统规则拒绝，请稍后再试";
  }

  if (
    normalizedMessage.includes("this group does not enable invite") ||
    rawMessage.includes('"code":10007')
  ) {
    return TUITranslateService.t(
      "TUIGroup.当前群未开启邀请权限，请先修改邀请方式"
    );
  }

  if (
    type === "add" &&
    (normalizedMessage.includes("too many group members") ||
      normalizedMessage.includes("max=100"))
  ) {
    return TUITranslateService.t("TUIGroup.单次添加成员过多，请分批操作");
  }

  return rawMessage || "操作失败";
};

const normalizeUserIDList = (userIDList: any[] = []) =>
  Array.from(
    new Set(
      userIDList
        .map((userID: any) => String(userID || "").trim())
        .filter(Boolean)
    )
  );

const buildGroupMemberFromFriend = (userID: string): IGroupMember => {
  const friend = friendList.value.find((item: any) => item.userID === userID);
  return {
    ...(friend || {}),
    userID,
    role: TUIChatEngine.TYPES.GRP_MBR_ROLE_MEMBER,
    nick: friend?.nick || userID,
    avatar: friend?.avatar || "",
    remark: friend?.remark || "",
    friendRemark: friend?.friendRemark || friend?.remark || "",
  } as IGroupMember;
};

const mergeOptimisticGroupMembers = (
  memberList: IGroupMember[] = [],
  userIDs: string[] = []
) => {
  const existingUserIDSet = new Set(
    memberList.map((memberItem: IGroupMember) => memberItem.userID).filter(Boolean)
  );
  const appendedMembers = normalizeUserIDList(userIDs)
    .filter((userID) => !existingUserIDSet.has(userID))
    .map(buildGroupMemberFromFriend);
  return appendedMembers.length > 0
    ? [...memberList, ...appendedMembers]
    : memberList;
};

const appendAddedGroupMembers = async (userIDs: string[], newlyAdded: string[]) => {
  const addedUserIDs = normalizeUserIDList(userIDs);
  if (addedUserIDs.length === 0) {
    return 0;
  }
  const existingUserIDSet = new Set(
    groupMemberList.value
      .map((memberItem: IGroupMember) => memberItem.userID)
      .filter(Boolean)
  );
  const newMembers = addedUserIDs
    .filter((userID) => !existingUserIDSet.has(userID))
    .map(buildGroupMemberFromFriend);
  if (newMembers.length > 0) {
    await applyCurrentGroupMemberList([...groupMemberList.value, ...newMembers]);
  }
  const addedSet = new Set(newlyAdded);
  syncGroupMemberCount(newMembers.filter(member => addedSet.has(member.userID)).length);
  const addedUserIDSet = new Set(addedUserIDs);
  transferList.value = transferList.value.filter(
    (item: IGroupMember) => !addedUserIDSet.has(item.userID || "")
  );
  transferTotal.value = transferList.value.length;
  return newMembers.length;
};

const syncGroupMemberCount = (delta = 0) => {
  if (!currentGroup.value) {
    return;
  }
  const currentProfileCount = Number(currentGroup.value.memberCount || 0);
  syncCurrentGroupState({
    memberCount: Math.max(currentProfileCount + delta, groupMemberList.value.length),
  });
  syncedMemberCountGroupID.value = currentGroupID.value;
};

const clampGroupMemberCount = (maxMemberCount?: number) => {
  if (!currentGroup.value || typeof maxMemberCount !== "number") {
    return;
  }
  syncCurrentGroupState({
    memberCount: Math.max(
      0,
      Math.min(Number(currentGroup.value.memberCount || 0), maxMemberCount)
    ),
  });
  syncedMemberCountGroupID.value = currentGroupID.value;
};

const liftGroupMemberCount = (minMemberCount?: number) => {
  if (!currentGroup.value || typeof minMemberCount !== "number") {
    return;
  }
  syncCurrentGroupState({
    memberCount: Math.max(Number(currentGroup.value.memberCount || 0), minMemberCount),
  });
  syncedMemberCountGroupID.value = currentGroupID.value;
};

const removeMembersFromLocalState = async (userIDList: any[] = []) => {
  const removedUserIDSet = new Set(normalizeUserIDList(userIDList));
  if (removedUserIDSet.size === 0) {
    return 0;
  }
  const previousMemberList = groupMemberList.value;
  const nextMemberList = previousMemberList.filter(
    (item: IGroupMember) => !removedUserIDSet.has(item?.userID || "")
  );
  const removedCount = previousMemberList.length - nextMemberList.length;
  if (removedCount > 0) {
    await applyCurrentGroupMemberList(nextMemberList);
    TUIStore.update(StoreName.GRP, "currentGroupMemberList", groupMemberList.value);
  }
  return removedCount;
};

type GroupAddMemberResult = {
  added: string[];
  already_members: string[];
  failed: Array<{ account: string; result: number; reason?: string }>;
};

const createEmptyAddMemberResult = (): GroupAddMemberResult => ({
  added: [],
  already_members: [],
  failed: [],
});

const mergeAddMemberResult = (
  target: GroupAddMemberResult,
  source: Partial<GroupAddMemberResult> = {}
) => {
  target.added.push(...(source.added || []));
  target.already_members.push(...(source.already_members || []));
  target.failed.push(...(source.failed || []));
};

const getGroupAddFailedReason = (
  item: { result?: number; reason?: string } = {}
) => {
  const reason = item.reason?.trim();
  if (reason) {
    return reason;
  }
  switch (item.result) {
    case -1:
      return "不能邀请不同归属用户入群";
    case 2:
      return "该用户可加入的群数量已达腾讯限制，无法邀请入群";
    default:
      return item.result != null
        ? `添加失败，错误码 ${item.result}`
        : "添加失败";
  }
};

const getGroupAddFailedReasonText = (result: GroupAddMemberResult) => {
  const reasons = Array.from(
    new Set((result.failed || []).map(getGroupAddFailedReason).filter(Boolean))
  );
  if (reasons.length === 0) {
    return "";
  }
  return `，原因：${reasons.slice(0, 2).join("；")}${
    reasons.length > 2 ? "等" : ""
  }`;
};

const showAddMemberResultToast = (result: GroupAddMemberResult) => {
  const addedCount = result.added?.length || 0;
  const alreadyCount = result.already_members?.length || 0;
  const failedCount = result.failed?.length || 0;
  const messageParts: string[] = [];
  if (addedCount > 0) {
    messageParts.push(`成功添加 ${addedCount} 人`);
  }
  if (alreadyCount > 0) {
    messageParts.push(`${alreadyCount} 人已在群中`);
  }
  if (failedCount > 0) {
    const reason = getGroupAddFailedReasonText(result);
    messageParts.push(`${failedCount} 人添加失败${reason}`);
    Toast({
      message: messageParts.join("，"),
      type: addedCount > 0 || alreadyCount > 0 ? TOAST_TYPE.WARNING : TOAST_TYPE.ERROR,
    });
    return;
  }
  if (messageParts.length > 0) {
    Toast({
      message: messageParts.join("，"),
      type: addedCount > 0 ? TOAST_TYPE.SUCCESS : TOAST_TYPE.WARNING,
    });
  }
};

const addMember = async (userIDList: any) => {
  const groupID = currentGroupID.value;
  const pendingUserIDList = normalizeUserIDList(userIDList);
  if (!groupID || pendingUserIDList.length === 0) {
    return;
  }

  const result = createEmptyAddMemberResult();
  for (
    let index = 0;
    index < pendingUserIDList.length;
    index += GROUP_ADD_MEMBER_BATCH_SIZE
  ) {
    const batch = pendingUserIDList.slice(index, index + GROUP_ADD_MEMBER_BATCH_SIZE);
    const batchResult = await addGroupMembers(groupID, batch);
    if (groupID !== currentGroupID.value) {
      return;
    }
    mergeAddMemberResult(result, batchResult);
  }
  const successfulUserIDs = normalizeUserIDList([
    ...result.added,
    ...result.already_members,
  ]);
  if (successfulUserIDs.length > 0) {
    await appendAddedGroupMembers(successfulUserIDs, result.added);
  }
  showAddMemberResultToast(result);
};

const changeOwner = async (userID: any) => {
  const options: any = {
    groupID: currentGroupID.value,
    newOwnerID: userID,
  };
  await TUIGroupService.changeGroupOwner(options);
  await refreshGroupMembersState();
};

const setMemberMuteTime = async (userID: string, type?: string) => {
  const groupID = currentGroupID.value;
  if (!groupID || !userID) {
    return;
  }
  if (type === "add") {
    await muteGroupMember(groupID, userID);
  } else {
    await unmuteGroupMember(groupID, userID);
  }
  await refreshGroupMembersState();
};

const handleAdmin = async (user: any) => {
  if (!isOwner.value) {
    Toast({
      message: "仅群主可以设置管理员",
      type: TOAST_TYPE.WARNING,
    });
    return;
  }
  if (!currentGroupID.value || !user?.userID) {
    return;
  }
  const isRemove = mergedAdminUserIDSet.value.has(user.userID);
  try {
    if (isRemove) {
      await removeGroupAdmin(currentGroupID.value, user.userID);
    } else {
      await addGroupAdmin(currentGroupID.value, user.userID);
    }
    Toast({
      message: isRemove ? "移除管理员成功" : "添加管理员成功",
      type: TOAST_TYPE.SUCCESS,
    });
    await refreshGroupMembersState(true, true);
  } catch (error: any) {
    Toast({
      message: `${isRemove ? "移除管理员失败" : "添加管理员失败"}${error?.message ? `：${error.message}` : ""}`,
      type: TOAST_TYPE.ERROR,
    });
  }
};

const handleAdminBatch = async (
  userList: any[],
  mode: "add" | "remove"
): Promise<AdminBatchResult> => {
  const managedUserList = Array.isArray(userList) ? userList : [];
  if (!isOwner.value) {
    Toast({
      message: "仅群主可以设置管理员",
      type: TOAST_TYPE.WARNING,
    });
    return {
      successCount: 0,
      failedCount: managedUserList.length || 1,
      firstErrorMessage: "仅群主可以设置管理员",
    };
  }
  const groupID = currentGroupID.value;
  if (!groupID) {
    return {
      successCount: 0,
      failedCount: managedUserList.length || 1,
      firstErrorMessage: "群组不存在",
    };
  }
  let successCount = 0;
  let failedCount = 0;
  let firstErrorMessage = "";
  for (const user of managedUserList) {
    if (!user?.userID) {
      continue;
    }
    try {
      if (mode === "add") {
        await addGroupAdmin(groupID, user.userID);
      } else {
        await removeGroupAdmin(groupID, user.userID);
      }
      successCount += 1;
    } catch (error: any) {
      failedCount += 1;
      if (!firstErrorMessage) {
        firstErrorMessage = error?.message || "操作失败";
      }
    }
  }
  if (successCount > 0) {
    await refreshGroupMembersState(true, true);
    if (mask.value) {
      await updateTransferList(transferType.value);
    }
  }
  return {
    successCount,
    failedCount,
    firstErrorMessage,
  };
};

const deleteGroupMember = async (userIDList: any) => {
  const groupID = currentGroupID.value;
  const pendingUserIDList = normalizeUserIDList(userIDList);
  if (!groupID || pendingUserIDList.length === 0) {
    return;
  }
  const previousMemberCount = currentGroup.value?.memberCount;
  for (const userID of pendingUserIDList) {
    await removeGroupMember(groupID, userID);
  }
  const removedCount = await removeMembersFromLocalState(pendingUserIDList);
  const nextMaxMemberCount =
    typeof previousMemberCount === "number"
      ? previousMemberCount - Math.max(removedCount, pendingUserIDList.length)
      : undefined;
  clampGroupMemberCount(nextMaxMemberCount);
  await refreshGroupMembersState(
    currentTab.value === "admin",
    true,
    pendingUserIDList,
    nextMaxMemberCount
  );
  Toast({
    message: `已移出 ${pendingUserIDList.length} 名成员`,
    type: TOAST_TYPE.SUCCESS,
  });
};

const quitGroup = async (group: any) => {
  await TUIGroupService.quitGroup(group.groupID);
  clearGroupInfo();
};

const back = () => {
  if (currentTab.value) {
    setCurrentTab("");
  } else {
    handleCompleteManage();
  }
};

const handleCompleteManage = () => {
  TUIStore.update(StoreName.GRP, "isShowManageComponent", false);
  const callback = TUIGroupServer.getOnCallCallback(
    TUIConstants.TUIGroup.SERVICE.METHOD.OPEN_GROUP_MANAGEMENT
  );
  TUIGroupServer.clearOnCallParams(
    TUIConstants.TUIGroup.SERVICE.METHOD.OPEN_GROUP_MANAGEMENT
  );
  callback && callback();
};
</script>
<style lang="scss" scoped src="./style/index.scss"></style>
