<template>
  <SelectUser
    :isRadio="selectOptions.isRadio"
    :isNeedSearch="selectOptions.isNeedSearch"
    :title="selectOptions.title"
    :userList="userList"
    :total="group.memberCount"
    :searchPlaceholder="
      TUITranslateService.t('TUIGroup.搜索备注、昵称、群昵称或userID')
    "
    @getMore="getMember"
    @search="handleSearch"
    @complete="handleSelectedResult"
  />
</template>
<script lang="ts" setup>
// @ts-nocheck
import {
  TUIGroupService,
  TUIStore,
  StoreName,
  TUITranslateService,
  IGroupModel,
} from "@tencentcloud/chat-uikit-engine-lite";
import { onMounted, onUnmounted, ref, watchEffect } from "../../../adapter-vue";
import type { IGroupMember } from "../../../interface";
import SelectUser from "../../common/SelectUser/index.vue";
import Server from "../server";
import { matchesMemberKeyword } from "../../../utils/member-search";
const TUIContactServer = Server.getInstance();
const TUIConstants = TUIContactServer.constants;

const memberList = ref<IGroupMember[]>([]);
const userList = ref<IGroupMember[]>([]);
const group = ref<IGroupModel | undefined>();
const selectOptions: any = ref({
  groupID: "",
  isRadio: false,
  isNeedSearch: false,
  title: "",
  filterUserIDList: [] as Array<string>,
});

watchEffect(() => {
  const params = TUIContactServer.getOnCallParams(
    TUIConstants.TUIContact.SERVICE.METHOD.SELECT_FRIEND
  );
  selectOptions.value.groupID = params.groupID;
  selectOptions.value.title = params.title;
  selectOptions.value.isRadio = params.isRadio;
  selectOptions.value.isNeedSearch = params.isNeedSearch;
  selectOptions.value.filterUserIDList = params.filterUserIDList;
});

const groupWatchOptions = {
  currentGroup: (data: IGroupModel) => {
    group.value = data;
  },
  currentGroupMemberList: (List: IGroupMember[]) => {
    memberList.value = List.map((item: IGroupMember) => {
      if (selectOptions.value.filterUserIDList.indexOf(item.userID) > -1) {
        item.isDisabled = true as any;
      } else {
        item.isDisabled = false as any;
      }
      return item;
    });
    userList.value = memberList.value;
  },
};

onMounted(() => {
  const currentGroup = TUIStore.getData(StoreName.GRP, "currentGroup");
  const currentGroupMemberList = TUIStore.getData(
    StoreName.GRP,
    "currentGroupMemberList"
  );
  if (currentGroup) {
    groupWatchOptions.currentGroup(currentGroup);
  }
  if (currentGroupMemberList) {
    groupWatchOptions.currentGroupMemberList(currentGroupMemberList);
  }
  TUIStore.watch(StoreName.GRP, groupWatchOptions);
});

onUnmounted(() => {
  TUIStore.unwatch(StoreName.GRP, groupWatchOptions);
});

const getMember = async () => {
  const groupID = group.value?.groupID;
  if (!groupID) {
    return;
  }
  const options = {
    groupID,
    count: 100,
    offset: memberList.value.length,
  };
  await TUIGroupService.getGroupMemberList(options);
};

const handleSelectedResult = (memberList: Array<any>) => {
  TUIStore.update(StoreName.GRP, "isShowSelectComponent", false);
  const callback = TUIContactServer.getOnCallCallback(
    TUIConstants.TUIGroup.SERVICE.METHOD.SELECT_GROUP_MEMBER
  );
  callback && callback(memberList);
};

const handleSearch = async (val: string) => {
  const keyword = val.trim();
  if (!keyword) {
    userList.value = [...memberList.value];
    return;
  }

  const results = memberList.value.filter((item: any) =>
    matchesMemberKeyword(item, keyword)
  );
  userList.value = [...results];
};
</script>
