<template>
  <Dialog
    :show="true"
    :isH5="!isPC"
    :isHeaderShow="false"
    :isFooterShow="false"
    :background="false"
    @update:show="closeCreated"
  >
    <div class="group" :class="[!isPC ? 'group-h5' : '']">
      <div class="group-box">
        <header class="group-box-header">
          <Icon
            :file="isPC ? closeIcon : backIcon"
            :class="['icon-close', isCreating && 'icon-close-disabled']"
            size="16px"
            @onClick="closeCreated"
          />
          <h1 class="group-box-header-title">
            {{ headerTitle }}
          </h1>
        </header>
        <ul v-if="!groupInfo.isEdit" class="group-list">
          <li class="group-list-item">
            <label class="group-list-item-label">{{
              TUITranslateService.t("TUIGroup.群头像")
            }}</label>
            <Avatar :url="groupInfo.profile.avatar" />
          </li>
          <ul>
            <li
              v-for="(item, index) in createInfo"
              :key="index"
              class="group-list-item"
            >
              <label class="group-list-item-label">{{ item.name }}</label>
              <input
                v-if="isPC"
                v-model="groupInfo.profile[item.key]"
                type="text"
                :placeholder="item.placeholder"
              />
              <span
                v-else
                class="group-h5-list-item-content"
                @click="edit(item.key)"
              >
                <p class="content">{{ groupInfo.profile[item.key] }}</p>
                <Icon :file="rightIcon" />
              </span>
            </li>
            <li class="group-list-introduction">
              <div class="group-list-item">
                <label class="group-list-item-label">{{
                  TUITranslateService.t("TUIGroup.群类型")
                }}</label>
                <GroupIntroduction
                  v-if="isPC"
                  :groupType="groupInfo.profile.type"
                  @selectType="selected"
                />
                <span
                  v-else
                  class="group-h5-list-item-content"
                  @click="edit('type')"
                >
                  <p class="content">{{ groupTypeDetail.label }}</p>
                  <Icon :file="rightIcon" />
                </span>
              </div>
            </li>
          </ul>
        </ul>
        <!-- Edit Group Name -->
        <div v-else class="group-list group-list-edit">
          <input
            v-if="groupInfo.groupConfig.type === 'input'"
            v-model="groupInfo.groupConfig.value"
            class="group-name-input"
            type="text"
            :placeholder="
              TUITranslateService.t(
                `TUIGroup.${groupInfo.groupConfig.placeholder}`
              )
            "
          />
          <GroupIntroduction
            v-else
            class="group-introduction-list"
            :groupType="groupInfo.groupConfig.value"
            @selectType="selected"
          />
        </div>
        <footer class="group-profile-footer">
          <button
            v-if="isPC && !groupInfo.isEdit"
            class="btn-default"
            :disabled="isCreating"
            @click="closeCreated"
          >
            {{ TUITranslateService.t("TUIGroup.取消") }}
          </button>
          <button
            class="btn-submit"
            :disabled="submitDisabledStatus"
            @click="submit"
          >
            {{ submitButtonText }}
          </button>
        </footer>
      </div>
    </div>
  </Dialog>
</template>
<script lang="ts" setup>
import TUIChatEngine, {
  TUITranslateService,
  TUIGroupService,
  TUIStore,
  StoreName,
} from "@tencentcloud/chat-uikit-engine-lite";
import { computed, reactive, ref, watchEffect } from "../../../adapter-vue";
import { isPC } from "../../../utils/env";
import Icon from "../../common/Icon.vue";
import backIcon from "../../../assets/icon/back.svg";
import closeIcon from "../../../assets/icon/icon-close.svg";
import rightIcon from "../../../assets/icon/right-icon.svg";
import GroupIntroduction from "./group-introduction/index.vue";
import {
  groupIntroConfig,
  findGroupIntroConfig,
} from "./group-introduction/config";
import Dialog from "../../common/Dialog/index.vue";
import { Toast, TOAST_TYPE } from "../../common/Toast/index";
import { withBusy } from "../../common/BusyOverlay/index";
import Avatar from "../../common/Avatar/index.vue";
import Server from "../server";
import { IUserProfile } from "../../../interface";
import { setGroupMemberNicknameEnabled } from "@/features/group/member-nickname";
const TUIGroupServer = Server.getInstance();
const TUIConstants = TUIGroupServer.constants;
const GROUP_CREATE_INITIAL_MEMBER_LIMIT = 20;
const GROUP_INVITE_BATCH_SIZE = 20;
const isCreating = ref(false);

const groupInfo = reactive<any>({
  profile: {
    groupID: "",
    name: "",
    type: groupIntroConfig[0].type,
    avatar: groupIntroConfig[0].icon,
    introduction: "",
    notification: "",
    // joinOption: '',
    memberList: [],
    isSupportTopic: false,
  },
  groupConfig: {
    title: "",
    value: "",
    key: "",
    type: "",
    placeholder: "",
  },
  isEdit: false,
});

watchEffect(() => {
  const params = TUIGroupServer.getOnCallParams(
    TUIConstants.TUIGroup.SERVICE.METHOD.CREATE_GROUP
  );
  groupInfo.profile.memberList = params.memberList;
  groupInfo.groupConfig.title = params.title;
});

const groupTypeDetail = computed(() => {
  return findGroupIntroConfig(groupInfo.profile.type);
});

const headerTitle = computed(() => {
  let name = "添加群聊";
  if (groupInfo.isEdit) {
    name = groupInfo.groupConfig.title;
  }
  return TUITranslateService.t(`TUIGroup.${name}`);
});

const createInfo = computed(() => {
  const groupNameInput = {
    name: TUITranslateService.t("TUIGroup.群名称"),
    key: "name",
    placeholder: TUITranslateService.t("TUIGroup.请输入群名称"),
  };
  const groupIDInput = {
    name: `${TUITranslateService.t("TUIGroup.群ID")}(${TUITranslateService.t(
      "TUIGroup.选填"
    )})`,
    key: "groupID",
    placeholder: TUITranslateService.t("TUIGroup.请输入群ID"),
  };
  return groupInfo.profile.type === TUIChatEngine.TYPES.GRP_COMMUNITY
    ? [groupNameInput]
    : [groupNameInput, groupIDInput];
});

const submitDisabledStatus = computed(() => {
  return isCreating.value || (groupInfo.profile.name === "" && !groupInfo.isEdit);
});

const submitButtonText = computed(() => {
  if (isCreating.value && !groupInfo.isEdit) {
    return TUITranslateService.t("TUIGroup.创建中");
  }
  return TUITranslateService.t("TUIGroup.确认");
});

const selected = (type: any) => {
  if (groupInfo.profile.type !== type) {
    groupInfo.profile.type = type;
    groupInfo.profile.avatar = findGroupIntroConfig(type).icon;
    if (groupInfo.isEdit) {
      groupInfo.groupConfig.value = type;
    }
  }
};

const createGroup = async (options: any) => {
  if (isCreating.value) {
    return;
  }
  const originalMembers = Array.isArray(options.memberList)
    ? [...new Set<string>(options.memberList.map((item: IUserProfile) => item.userID).filter(Boolean))].map(userID => ({ userID }))
    : [];
  if (
    originalMembers.length > 100
  ) {
    Toast({
      message: '创建群聊时最多可选择 100 位好友',
      type: TOAST_TYPE.WARNING,
    });
    return;
  }

  const initialMembers = originalMembers.slice(0, GROUP_CREATE_INITIAL_MEMBER_LIMIT);
  const remainingMembers = originalMembers.slice(GROUP_CREATE_INITIAL_MEMBER_LIMIT);

  const requestOptions = {
    ...options,
    inviteOption: TUIChatEngine.TYPES.JOIN_OPTIONS_FREE_ACCESS,
    memberList: initialMembers,
  };

  try {
    isCreating.value = true;
    await withBusy("正在创建群聊...", async () => {
    if (requestOptions.type === TUIChatEngine.TYPES.GRP_COMMUNITY) {
      delete requestOptions.groupID;
    }
    const res = await TUIGroupService.createGroup(requestOptions);
    const createdGroupID = res.data.group.groupID;
    if (createdGroupID) {
      TUIStore.update(StoreName.GRP, "currentGroupID", createdGroupID);
      TUIStore.update(StoreName.GRP, "currentGroup", res.data.group);
      TUIStore.update(StoreName.GRP, "currentGroupAttributes", {});
      TUIStore.update(StoreName.GRP, "currentGroupMemberList", []);
      TUIStore.update(StoreName.GRP, "offset", 0);
      TUIStore.update(StoreName.GRP, "isCompleted", false);
      void TUIGroupService.getGroupMemberList({
        groupID: createdGroupID,
        count: GROUP_CREATE_INITIAL_MEMBER_LIMIT,
        offset: 0,
      });
    }
    const currentUserID =
      TUIStore.getData(StoreName.USER, "userProfile")?.userID || "";
    if (createdGroupID) {
      setGroupMemberNicknameEnabled(
        createdGroupID,
        currentUserID || undefined,
        true
      );
    }
    let importWarning = "";
    if (createdGroupID && remainingMembers.length > 0) {
      importWarning = await inviteRemainingMembers(
        createdGroupID,
        remainingMembers
      );
    }
    const { type } = res.data.group;
    if (type === TUIChatEngine.TYPES.GRP_AVCHATROOM) {
      await TUIGroupService.joinGroup({
        groupID: res.data.group.groupID,
        applyMessage: "",
      });
    }
    handleCompleteCreate(res.data.group);
    Toast({
      message: importWarning || TUITranslateService.t("TUIGroup.群组创建成功"),
      type: importWarning ? TOAST_TYPE.WARNING : TOAST_TYPE.SUCCESS,
    });
    });
  } catch (err: any) {
    Toast({
      message: err.message,
      type: TOAST_TYPE.ERROR,
    });
  } finally {
    isCreating.value = false;
  }
};

const inviteRemainingMembers = async (
  groupID: string,
  remainingMembers: Array<{ userID: string }>
) => {
  for (
    let index = 0;
    index < remainingMembers.length;
    index += GROUP_INVITE_BATCH_SIZE
  ) {
    const batch = remainingMembers
      .slice(index, index + GROUP_INVITE_BATCH_SIZE)
      .map((item) => item.userID);
    try {
      const response: any = await TUIGroupService.addGroupMember({
        groupID,
        userIDList: batch,
      });
      if (response?.data?.failureUserIDList?.length || response?.data?.pendingUserIDList?.length) {
        return "群已创建，部分好友未能加入，请在群设置中查看并继续邀请。";
      }
    } catch (error: any) {
      return (
        "群已创建，但追加成员失败，请在群设置中继续添加。" + (error?.message || "")
      );
    }
  }
  return "";
};

const submit = () => {
  if (isCreating.value) {
    return;
  }
  const { profile } = groupInfo;
  if (groupInfo.isEdit) {
    groupInfo.profile[groupInfo.groupConfig.key] = groupInfo.groupConfig.value;
    return (groupInfo.isEdit = !groupInfo.isEdit);
  } else {
    createGroup({ ...profile });
  }
};

const closeCreated = () => {
  if (isCreating.value) {
    return;
  }
  if (groupInfo.isEdit) {
    return (groupInfo.isEdit = !groupInfo.isEdit);
  }
  handleCompleteCreate(null);
};

const edit = (label: string) => {
  groupInfo.isEdit = !groupInfo.isEdit;
  groupInfo.groupConfig.key = label;
  groupInfo.groupConfig.value = (groupInfo.profile as any)[label];
  switch (label) {
    case "name":
      groupInfo.groupConfig.title = "设置群名称";
      groupInfo.groupConfig.placeholder = "请输入群名称";
      groupInfo.groupConfig.type = "input";
      break;
    case "groupID":
      groupInfo.groupConfig.title = "设置群ID";
      groupInfo.groupConfig.placeholder = "请输入群ID";
      groupInfo.groupConfig.type = "input";
      break;
    case "type":
      groupInfo.groupConfig.title = "选择群类型";
      groupInfo.groupConfig.type = "select";
      break;
  }
};

const handleCompleteCreate = (group: any) => {
  TUIStore.update(StoreName.GRP, "isShowCreateComponent", false);
  const callback = TUIGroupServer.getOnCallCallback(
    TUIConstants.TUIGroup.SERVICE.METHOD.CREATE_GROUP
  );
  callback && callback(group);
};
</script>
<style lang="scss" scoped src="./style/index.scss"></style>
