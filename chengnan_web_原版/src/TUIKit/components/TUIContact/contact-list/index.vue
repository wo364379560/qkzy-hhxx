<template>
  <ul
    v-if="!contactSearchingStatus"
    :class="['tui-contact-list', !isPC && 'tui-contact-list-h5']"
  >
    <li
      v-for="(contactListObj, key) in sortContactListMap"
      :key="key"
      class="tui-contact-list-item"
    >
      <header
        class="tui-contact-list-item-header"
        @click="toggleCurrentContactList(key)"
      >
        <div class="tui-contact-list-item-header-left">
          <Icon
            :file="currentContactListKey === key ? downSVG : rightSVG"
            width="16px"
            height="16px"
          />
          <div>
            {{ TUITranslateService.t(`TUIContact.${contactListObj.title}`) }}
          </div>
        </div>
        <div class="tui-contact-list-item-header-right">
          <span
            v-if="contactListObj.unreadCount"
            class="tui-contact-list-item-header-right-unread"
          >
            {{ contactListObj.unreadCount }}
          </span>
        </div>
      </header>
      <ul
        :class="[
          'tui-contact-list-item-main',
          currentContactListKey === key ? '' : 'hidden',
        ]"
      >
        <template v-if="key === 'friendList'">
          <div
            v-for="(groupData, groupKey) in friendGroups"
            :key="groupKey"
          >
            <div class="tui-contact-list-group-title">
              {{ groupKey }} ({{ groupData.length }})
            </div>
            <li
              v-for="contactListItem in groupData"
              :key="contactListItem.renderKey"
              class="tui-contact-list-item-main-item"
              :class="['selected']"
              @click="selectItem(contactListItem)"
            >
              <ContactListItem
                :key="contactListItem.renderKey"
                :item="deepCopy(contactListItem)"
                :display-online-status="displayOnlineStatus"
              />
            </li>
          </div>
        </template>
        <template v-else>
          <li
            v-for="contactListItem in flatContacts(contactListObj.list)"
            :key="contactListItem.renderKey"
            class="tui-contact-list-item-main-item"
            :class="['selected']"
            @click="selectItem(contactListItem)"
          >
            <ContactListItem
              :key="contactListItem.renderKey"
              :item="deepCopy(contactListItem)"
            />
          </li>
        </template>
      </ul>
    </li>
  </ul>
  <ul v-else class="tui-contact-list">
    <li
      v-for="(item, key) in contactSearchResult"
      :key="key"
      class="tui-contact-list-item"
    >
      <div v-if="item.list[0]" class="tui-contact-search-list">
        <div class="tui-contact-search-list-title">
          {{ TUITranslateService.t(`TUIContact.${item.label}`) }}
        </div>
        <div
          v-for="(listItem, index) in item.list"
          :key="index"
          class="tui-contact-search-list-item"
          :class="['selected']"
          @click="selectItem(listItem)"
        >
          <ContactListItem :item="listItem" :display-online-status="false" />
        </div>
      </div>
    </li>
    <div v-if="isContactSearchNoResult" class="tui-contact-search-list-default">
      {{ TUITranslateService.t("TUIContact.无搜索结果") }}
    </div>
  </ul>
</template>
<script setup lang="ts">
import { captureFriendOwnerScope, synchronizeFriendOwner } from "@/features/account/friendOwnership";
import { updatePresenceLease, releasePresenceLease } from '@/features/presence/subscriptions';
const onlinePresenceLease = Symbol('online-status');

// @ts-nocheck
import TUIChatEngine, {
  TUITranslateService,
  TUIStore,
  StoreName,
  TUIFriendService,
  TUIUserService,
} from "@tencentcloud/chat-uikit-engine-lite";
import TUICore, { TUIConstants } from "@tencentcloud/tui-core-lite";
import {
  ref,
  computed,
  onMounted,
  onUnmounted,
  provide,
} from "../../../adapter-vue";
import downSVG from "../../../assets/icon/down-icon.svg";
import rightSVG from "../../../assets/icon/right-icon.svg";
import { isPC } from "../../../utils/env";
import Icon from "../../common/Icon.vue";
import { deepCopy } from "../../TUIChat/utils/utils";
import { sortByFirstChar } from "../utils/sortByFirstChar";
import ContactListItem from "./contact-list-item/index.vue";
import type {
  IContactList,
  IContactSearchResult,
  IBlackListUserItem,
  IUserStatus,
  IUserStatusMap,
  IContactInfoType,
} from "../../../interface";
import type {
  IGroupModel,
  IConversationModel,
  Friend,
  FriendApplication,
} from "@tencentcloud/chat-uikit-engine-lite";
const currentContactListKey = ref<keyof IContactList>("");
const currentContactInfo = ref<IContactInfoType>({} as IContactInfoType);
const contactListMap = ref<IContactList>({
  friendApplicationList: {
    key: "friendApplicationList",
    title: "新的联系人",
    list: [] as FriendApplication[],
    unreadCount: 0,
  },
  blackList: {
    key: "blackList",
    title: "黑名单",
    list: [] as IBlackListUserItem[],
  },
  groupList: {
    key: "groupList",
    title: "我的群聊",
    list: [] as IGroupModel[],
  },
  friendList: {
    key: "friendList",
    title: "我的好友",
    list: [] as Friend[],
  },
});
type RenderContact = IContactInfoType & { renderKey?: string };
const friendGroups = computed(() => sortByFirstChar<RenderContact>(contactListMap.value.friendList.list,
  item => { const friend = item as Friend; return friend.remark || friend.profile?.nick || friend.userID || ''; }).groupedList || {});
function flatContacts(value: unknown): RenderContact[] { return Array.isArray(value) ? value : []; }
const sortContactListMap = computed(() => {
  const { groupedList: groupedFriendList } = sortByFirstChar(
    contactListMap.value?.friendList?.list,
    (friend: Friend) =>
      friend.remark || friend.profile?.nick || friend.userID || ""
  );
  const { sortedList: sortedGroupList } = sortByFirstChar(
    contactListMap.value?.groupList?.list,
    (group: IGroupModel) => group.name || group.groupID || "",
    false
  );
  return {
    ...contactListMap.value,
    friendList: {
      ...contactListMap.value?.friendList,
      list: groupedFriendList,
    },
    groupList: {
      ...contactListMap.value?.groupList,
      list: sortedGroupList,
    },
  };
});
const contactSearchingStatus = ref<boolean>(false);
const contactSearchResult = ref<IContactSearchResult>();
const displayOnlineStatus = ref<boolean>(false);
const userOnlineStatusMap = ref<IUserStatusMap>({});
let subscribedFriendUserIDs: string[] = [];

const isContactSearchNoResult = computed(
  (): boolean =>
    !contactSearchResult?.value?.user?.list[0] &&
    !contactSearchResult?.value?.group?.list[0]
);

function normalizeGroupID(groupID?: string) {
  return String(groupID || "").trim();
}

function getConversationGroupID(conversation: IConversationModel) {
  const profileGroupID = normalizeGroupID(conversation?.groupProfile?.groupID);
  if (profileGroupID) {
    return profileGroupID;
  }

  const conversationID = String(conversation?.conversationID || "");
  const groupConversationPrefix = TUIChatEngine.TYPES.CONV_GROUP || "GROUP";
  return conversationID.startsWith(groupConversationPrefix)
    ? conversationID.slice(groupConversationPrefix.length)
    : "";
}

function isGroupConversation(conversation: IConversationModel) {
  const conversationID = String(conversation?.conversationID || "");
  const groupConversationPrefix = TUIChatEngine.TYPES.CONV_GROUP || "GROUP";
  return (
    conversation?.type === TUIChatEngine.TYPES.CONV_GROUP ||
    (conversation as any)?.conversationType ===
      TUIChatEngine.TYPES.CONV_GROUP ||
    conversationID.startsWith(groupConversationPrefix)
  );
}

function buildGroupFromConversation(conversation: IConversationModel) {
  if (!isGroupConversation(conversation)) {
    return null;
  }

  const groupID = getConversationGroupID(conversation);
  if (!groupID) {
    return null;
  }

  const groupProfile = conversation?.groupProfile || {};
  return {
    ...groupProfile,
    groupID,
    name:
      groupProfile.name ||
      conversation?.getShowName?.() ||
      (conversation as IConversationModel & { name?: string }).name ||
      groupID,
    avatar:
      groupProfile.avatar ||
      conversation?.getAvatar?.() ||
      (conversation as IConversationModel & { avatar?: string }).avatar ||
      "",
    type: groupProfile.type || "",
  } as IGroupModel;
}

function mergeGroupInfo(baseGroup: IGroupModel, fallbackGroup: IGroupModel) {
  return {
    ...fallbackGroup,
    ...baseGroup,
    groupID: normalizeGroupID(baseGroup.groupID) || fallbackGroup.groupID,
    name: baseGroup.name || fallbackGroup.name,
    avatar: baseGroup.avatar || fallbackGroup.avatar,
    type: baseGroup.type || fallbackGroup.type,
  } as IGroupModel;
}

function mergeJoinedGroupList(
  groupList: IGroupModel[] = [],
  conversationList: IConversationModel[] = []
) {
  const groupMap = new Map<string, IGroupModel>();

  groupList.forEach((group) => {
    const groupID = normalizeGroupID(group?.groupID);
    if (groupID) {
      groupMap.set(groupID, { ...group, groupID } as IGroupModel);
    }
  });

  conversationList.forEach((conversation) => {
    const fallbackGroup = buildGroupFromConversation(conversation);
    if (!fallbackGroup) {
      return;
    }

    const currentGroup = groupMap.get(fallbackGroup.groupID);
    groupMap.set(
      fallbackGroup.groupID,
      currentGroup ? mergeGroupInfo(currentGroup, fallbackGroup) : fallbackGroup
    );
  });

  return Array.from(groupMap.values());
}

function syncJoinedGroupList(
  groupListFromStore?: IGroupModel[],
  conversationListFromStore?: IConversationModel[]
) {
  const groupList = Array.isArray(groupListFromStore)
    ? groupListFromStore
    : TUIStore.getData(StoreName.GRP, "groupList") || [];
  const conversationList = Array.isArray(conversationListFromStore)
    ? conversationListFromStore
    : TUIStore.getData(StoreName.CONV, "conversationList") || [];

  updateContactListMap(
    "groupList",
    mergeJoinedGroupList(
      groupList as IGroupModel[],
      conversationList as IConversationModel[]
    )
  );
}

function syncContactStateFromStore() {
  const isCustomerServiceEnabled = TUIStore.getData(
    StoreName.APP,
    "enabledCustomerServicePlugin"
  );
  if (isCustomerServiceEnabled) {
    onCustomerServiceCommercialPluginUpdated(Boolean(isCustomerServiceEnabled));
  }

  syncJoinedGroupList();

  const userBlacklist = TUIStore.getData(StoreName.USER, "userBlacklist");
  if (Array.isArray(userBlacklist)) {
    updateContactListMap("blackList", userBlacklist as IBlackListUserItem[]);
  }

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

  const friendList = TUIStore.getData(StoreName.FRIEND, "friendList");
  if (Array.isArray(friendList)) {
    updateContactListMap("friendList", friendList as Friend[]);
  }

  const friendApplicationList = TUIStore.getData(
    StoreName.FRIEND,
    "friendApplicationList"
  );
  if (Array.isArray(friendApplicationList)) {
    updateContactListMap(
      "friendApplicationList",
      friendApplicationList as FriendApplication[]
    );
  }

  const friendApplicationUnreadCount = TUIStore.getData(
    StoreName.FRIEND,
    "friendApplicationUnreadCount"
  );
  if (typeof friendApplicationUnreadCount === "number") {
    contactListMap.value.friendApplicationList.unreadCount =
      friendApplicationUnreadCount;
  }

  const searchingStatus = TUIStore.getData(
    StoreName.CUSTOM,
    "currentContactSearchingStatus"
  );
  if (typeof searchingStatus === "boolean") {
    contactSearchingStatus.value = searchingStatus;
  }

  const currentSearchResult = TUIStore.getData(
    StoreName.CUSTOM,
    "currentContactSearchResult"
  );
  if (currentSearchResult) {
    contactSearchResult.value = currentSearchResult as IContactSearchResult;
  }

  const currentListKey = TUIStore.getData(
    StoreName.CUSTOM,
    "currentContactListKey"
  );
  if (typeof currentListKey === "string") {
    currentContactListKey.value = currentListKey as keyof IContactList;
  }

  const currentContact = TUIStore.getData(
    StoreName.CUSTOM,
    "currentContactInfo"
  );
  if (currentContact && typeof currentContact === "object") {
    currentContactInfo.value = currentContact as IContactInfoType;
  }

  if (displayOnlineStatus.value) {
    void syncFriendOnlineStatus(
      contactListMap.value.friendList.list as Friend[]
    );
  }
}

onMounted(() => {
  TUIStore.watch(StoreName.APP, {
    enabledCustomerServicePlugin: onCustomerServiceCommercialPluginUpdated,
  });

  TUIStore.watch(StoreName.GRP, {
    groupList: onGroupListUpdated,
  });

  TUIStore.watch(StoreName.CONV, {
    conversationList: onConversationListUpdated,
  });

  TUIStore.watch(StoreName.USER, {
    userBlacklist: onUserBlacklistUpdated,
    displayOnlineStatus: onDisplayOnlineStatusUpdated,
    userStatusList: onUserStatusListUpdated,
  });

  TUIStore.watch(StoreName.FRIEND, {
    friendList: onFriendListUpdated,
    friendApplicationList: onFriendApplicationListUpdated,
    friendApplicationUnreadCount: onFriendApplicationUnreadCountUpdated,
  });

  TUIStore.watch(StoreName.CUSTOM, {
    currentContactSearchingStatus: onCurrentContactSearchingStatusUpdated,
    currentContactSearchResult: onCurrentContactSearchResultUpdated,
    currentContactListKey: onCurrentContactListKeyUpdated,
    currentContactInfo: onCurrentContactInfoUpdated,
  });

  syncContactStateFromStore();
});

onUnmounted(() => {
  TUIStore.unwatch(StoreName.APP, {
    enabledCustomerServicePlugin: onCustomerServiceCommercialPluginUpdated,
  });

  TUIStore.unwatch(StoreName.GRP, {
    groupList: onGroupListUpdated,
  });

  TUIStore.unwatch(StoreName.CONV, {
    conversationList: onConversationListUpdated,
  });

  TUIStore.unwatch(StoreName.USER, {
    userBlacklist: onUserBlacklistUpdated,
    displayOnlineStatus: onDisplayOnlineStatusUpdated,
    userStatusList: onUserStatusListUpdated,
  });

  TUIStore.unwatch(StoreName.FRIEND, {
    friendList: onFriendListUpdated,
    friendApplicationList: onFriendApplicationListUpdated,
    friendApplicationUnreadCount: onFriendApplicationUnreadCountUpdated,
  });

  TUIStore.unwatch(StoreName.CUSTOM, {
    currentContactSearchingStatus: onCurrentContactSearchingStatusUpdated,
    currentContactSearchResult: onCurrentContactSearchResultUpdated,
    currentContactListKey: onCurrentContactListKeyUpdated,
    currentContactInfo: onCurrentContactInfoUpdated,
  });

  void releasePresenceLease(onlinePresenceLease);
});

function toggleCurrentContactList(key: keyof IContactList) {
  if (currentContactListKey.value === key) {
    currentContactListKey.value = "";
    currentContactInfo.value = {} as IContactInfoType;
    TUIStore.update(StoreName.CUSTOM, "currentContactListKey", "");
    TUIStore.update(
      StoreName.CUSTOM,
      "currentContactInfo",
      {} as IContactInfoType
    );
  } else {
    currentContactListKey.value = key;
    TUIStore.update(StoreName.CUSTOM, "currentContactListKey", key);
    if (key === "friendApplicationList") {
      TUIFriendService.setFriendApplicationRead();
    }
  }
}

function selectItem(item: any) {
  currentContactInfo.value = item;
  // For a result in the search list, before viewing the contactInfo details,
  // it is necessary to update the data for the "already in the group list/already in the friend list" situation to obtain more detailed information
  if (contactSearchingStatus.value) {
    let targetListItem;
    if ((currentContactInfo.value as Friend)?.userID) {
      targetListItem = contactListMap.value?.friendList?.list?.find(
        (item: IContactInfoType) =>
          (item as Friend)?.userID ===
          (currentContactInfo.value as Friend)?.userID
      );
    } else if ((currentContactInfo.value as IGroupModel)?.groupID) {
      targetListItem = contactListMap.value?.groupList?.list?.find(
        (item: IContactInfoType) =>
          (item as IGroupModel)?.groupID ===
          (currentContactInfo.value as IGroupModel)?.groupID
      );
    }
    if (targetListItem) {
      currentContactInfo.value = {
        ...targetListItem,
        _wananID: (currentContactInfo.value as any)?._wananID || "",
        _phone: (currentContactInfo.value as any)?._phone || "",
      };
    }
  }
  TUIStore.update(
    StoreName.CUSTOM,
    "currentContactInfo",
    currentContactInfo.value
  );
}

function onDisplayOnlineStatusUpdated(status: boolean) {
  displayOnlineStatus.value = status;
  if (!status) {
    userOnlineStatusMap.value = {};
    void unsubscribeFriendOnlineStatus(subscribedFriendUserIDs);
    subscribedFriendUserIDs = [];
    return;
  }
  void syncFriendOnlineStatus(contactListMap.value.friendList.list as Friend[]);
}

function onUserStatusListUpdated(list: Map<string, IUserStatus>) {
  if (!list || list.size === 0) {
    return;
  }

  userOnlineStatusMap.value = [...list.entries()].reduce(
    (result, [userID, status]) => {
      result[userID] = status;
      return result;
    },
    { ...userOnlineStatusMap.value }
  );
}

function onCustomerServiceCommercialPluginUpdated(isEnabled: boolean) {
  if (!isEnabled) {
    return;
  }

  // After the customer purchases the customer service plug-in,
  // the engine updates the enabledCustomerServicePlugin to true through the commercial capability bit.
  const contactListExtensionID =
    TUIConstants.TUIContact.EXTENSION.CONTACT_LIST.EXT_ID;
  const tuiContactExtensionList = TUICore.getExtensionList(
    contactListExtensionID
  );

  const customerData = tuiContactExtensionList.find((extension: any) => {
    const { name, accountList = [] } = extension.data || {};
    return name === "customer" && accountList.length > 0;
  });

  if (customerData) {
    const { data, text } = customerData;
    const { accountList } = (data || {}) as { accountList: string[] };

    TUIUserService.getUserProfile({ userIDList: accountList })
      .then((res) => {
        if (res.data.length > 0) {
          const customerList = {
            title: text,
            list: res.data.map((item: any, index: number) => ({
              ...item,
              renderKey: generateRenderKey("customerList", item, index),
              infoKeyList: [],
              btnKeyList: ["enterC2CConversation"],
            })),
            key: "customerList",
          };
          contactListMap.value = { ...contactListMap.value, customerList };
        }
      })
      .catch(() => {});
  }
}

function onGroupListUpdated(groupList: IGroupModel[]) {
  syncJoinedGroupList(groupList);
}

function onConversationListUpdated(conversationList: IConversationModel[]) {
  syncJoinedGroupList(undefined, conversationList);
}

function onUserBlacklistUpdated(userBlacklist: IBlackListUserItem[]) {
  updateContactListMap("blackList", userBlacklist);
}

function onFriendApplicationUnreadCountUpdated(
  friendApplicationUnreadCount: number
) {
  contactListMap.value.friendApplicationList.unreadCount =
    friendApplicationUnreadCount;
}

let ownerFriendScope = '';
let knownOwnerFriends: Set<string> | null = null;
function onFriendListUpdated(friendList: Friend[]) {
  const scope=captureFriendOwnerScope();
  const ids=new Set(friendList.map(item=>String(item.userID||'')).filter(Boolean));
  if (scope.active() && ownerFriendScope===scope.key && knownOwnerFriends) {
    [...ids].filter(id=>!knownOwnerFriends!.has(id)).slice(0,16).forEach(id=>{void synchronizeFriendOwner(id)});
  }
  ownerFriendScope=scope.key;knownOwnerFriends=ids;

  updateContactListMap("friendList", friendList);
  if (displayOnlineStatus.value) {
    void syncFriendOnlineStatus(friendList);
  }
}

function onFriendApplicationListUpdated(
  friendApplicationList: FriendApplication[]
) {
  updateContactListMap("friendApplicationList", friendApplicationList);
}

function updateContactListMap(
  key: keyof IContactList,
  list: IContactInfoType[]
) {
  contactListMap.value[key].list = list;
  contactListMap.value[key].list.map(
    (item: IContactInfoType, index: number) =>
      Object.assign(item, { renderKey: generateRenderKey(key, item, index) })
  );
  updateCurrentContactInfoFromList(contactListMap.value[key].list, key);
}

function updateCurrentContactInfoFromList(
  list: IContactInfoType[],
  type: keyof IContactList
) {
  if (
    !(currentContactInfo.value as Friend)?.userID &&
    !(currentContactInfo.value as IGroupModel)?.groupID
  ) {
    return;
  }
  if (type === currentContactListKey.value || contactSearchingStatus.value) {
    currentContactInfo.value =
      list?.find(
        (item: any) =>
          (item?.groupID &&
            item?.groupID ===
              (currentContactInfo.value as IGroupModel)?.groupID) ||
          (item?.userID &&
            item?.userID === (currentContactInfo.value as Friend)?.userID)
      ) || ({} as IContactInfoType);
    TUIStore.update(
      StoreName.CUSTOM,
      "currentContactInfo",
      currentContactInfo.value
    );
  }
}

function generateRenderKey(
  contactListMapKey: keyof IContactList,
  contactInfo: IContactInfoType,
  index: number
) {
  return `${contactListMapKey}-${
    (contactInfo as Friend).userID ||
    (contactInfo as IGroupModel).groupID ||
    `index${index}`
  }`;
}

function onCurrentContactSearchResultUpdated(
  searchResult: IContactSearchResult
) {
  contactSearchResult.value = searchResult;
}

function onCurrentContactSearchingStatusUpdated(searchingStatus: boolean) {
  contactSearchingStatus.value = searchingStatus;
  TUIStore.update(
    StoreName.CUSTOM,
    "currentContactInfo",
    {} as IContactInfoType
  );
  TUIStore.update(StoreName.CUSTOM, "currentContactListKey", "");
}

function onCurrentContactInfoUpdated(contactInfo: IContactInfoType) {
  currentContactInfo.value = contactInfo;
}

function onCurrentContactListKeyUpdated(contactListKey: string) {
  currentContactListKey.value = contactListKey;
}

provide("userOnlineStatusMap", userOnlineStatusMap);

function batchUserIDs(userIDList: string[]) {
  const batches: string[][] = [];
  for (let index = 0; index < userIDList.length; index += 100) {
    batches.push(userIDList.slice(index, index + 100));
  }
  return batches;
}

async function unsubscribeFriendOnlineStatus(_userIDList: string[]) {
  await releasePresenceLease(onlinePresenceLease);
}

async function syncFriendOnlineStatus(friendList: Friend[] = []) {
  if (!displayOnlineStatus.value) { await releasePresenceLease(onlinePresenceLease); return; }
  const ids = [...new Set(friendList.map(item => String(item?.userID || '').trim()).filter(Boolean))];
  subscribedFriendUserIDs = ids;
  await updatePresenceLease(onlinePresenceLease, ids, rows => {
    const next = { ...userOnlineStatusMap.value };
    rows.forEach(status => { next[status.userID] = { statusType: Number(status.statusType || 0), customStatus: String(status.customStatus || '') }; });
    userOnlineStatusMap.value = next;
  });
}

</script>

<style lang="scss" scoped src="./style/index.scss"></style>
