<template>
  <SelectUser
    :isRadio="selectOptions.isRadio"
    :isNeedSearch="selectOptions.isNeedSearch"
    :title="selectOptions.title"
    :userList="userList"
    :maxSelectable="selectOptions.maxSelectable"
    :maxSelectionMessage="selectOptions.maxSelectionMessage"
    @search="handleSearch"
    @complete="handleSelectedResult"
  />
</template>
<script lang="ts" setup>
import {
  TUIFriendService,
  TUIStore,
  StoreName,
  TUITranslateService,
} from '@tencentcloud/chat-uikit-engine-lite';
import { ref, watchEffect } from '../../../adapter-vue';
import { Toast, TOAST_TYPE } from '../../common/Toast/index';
import SelectUser from '../../common/SelectUser/index.vue';
import Server from '../server';
const TUIContactServer = Server.getInstance();
const TUIConstants = TUIContactServer.constants;

const friendList = ref<any[]>([]);
const userList = ref<any[]>([]);
const selectOptions = ref({
  isRadio: false,
  isNeedSearch: false,
  title: '',
  maxSelectable: -1,
  maxSelectionMessage: '',
});

watchEffect(() => {
  const params = TUIContactServer.getOnCallParams(TUIConstants.TUIContact.SERVICE.METHOD.SELECT_FRIEND);
  selectOptions.value.title = params.title;
  selectOptions.value.isRadio = params.isRadio;
  selectOptions.value.isNeedSearch = true;
  selectOptions.value.maxSelectable = params.maxSelectable ?? -1;
  selectOptions.value.maxSelectionMessage = params.maxSelectionMessage ?? '';
  TUIFriendService.getFriendList().then((res: any) => {
    friendList.value = res.data.map((item: any) => ({
      ...item,
      ...item.profile,
      userID: item.userID || item.profile?.userID || '',
      nick: item.profile?.nick || item.nick || '',
      avatar: item.profile?.avatar || item.avatar || '',
      remark: item.remark || '',
      friendRemark: item.friendRemark || item.remark || '',
    }));
    userList.value = friendList.value;
  }).catch((err: any) => {
    console.warn('getFriendList error:', err);
  });
});

const handleSelectedResult = (memberList: Array<any>) => {
  TUIStore.update(StoreName.CUSTOM, 'isShowSelectFriendComponent', false);
  const callback = TUIContactServer.getOnCallCallback(
    TUIConstants.TUIContact.SERVICE.METHOD.SELECT_FRIEND,
  ) as ((members: Array<any>) => void) | undefined;
  callback && callback(memberList);
};

const searchFail = () => {
  Toast({
    message: TUITranslateService.t('TUISearch.暂无搜索结果'),
    type: TOAST_TYPE.ERROR,
  });
  userList.value = [...friendList.value];
};

const handleSearch = async (val: string) => {
  const keyword = val.trim();
  if (!keyword) {
    return userList.value = friendList.value;
  }

  const lowerKeyword = keyword.toLowerCase();
  const results = friendList.value.filter((item: any) => {
    const nick = (item.nick || '').toLowerCase();
    const userID = (item.userID || '').toLowerCase();
    const remark = (item.remark || item.friendRemark || '').toLowerCase();
    return nick.includes(lowerKeyword)
      || userID.includes(lowerKeyword)
      || remark.includes(lowerKeyword);
  });

  if (!results.length) {
    return searchFail();
  }
  userList.value = results;
};
</script>
