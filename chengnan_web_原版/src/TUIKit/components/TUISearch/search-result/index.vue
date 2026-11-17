<template>
  <SearchResultLoading
    v-if="isLoading"
    :class="['search-result-loading', !isPC && 'search-result-loading-h5']"
  />
  <SearchResultDefault
    v-else-if="isSearchDefaultShow"
    :class="['search-result-default', !isPC && 'search-result-default-h5']"
  />
  <div
    v-else
    :class="[
      'tui-search-result',
      !isPC && 'tui-search-result-h5',
      isPC && isResultDetailShow && 'tui-search-result-with-border',
    ]"
  >
    <div
      v-if="
        props.searchType !== 'conversation' && (isPC || !isResultDetailShow)
      "
      class="tui-search-result-main"
    >
      <div class="tui-search-result-list">
        <div
          v-for="result in searchResult"
          :key="result.key"
          class="tui-search-result-list-item"
        >
          <div v-if="props.searchType === 'global'" class="header">
            {{ TUITranslateService.t(`TUISearch.${result.label}`) }}
          </div>
          <div class="list">
            <div
              v-for="(item, index) in result.list"
              :key="`${result.key}-${index}`"
              :class="[generateListItemClass(item)]"
            >
              <SearchResultItem
                v-if="
                  result.key === 'contact' ||
                  result.key === 'group' ||
                  item.conversation
                "
                :listItem="item"
                :type="result.key"
                displayType="info"
                :keywordList="keywordList"
                @showResultDetail="showResultDetail"
                @navigateToChatPosition="navigateToChatPosition"
              />
            </div>
          </div>
          <div
            v-if="currentSearchTabKey === 'all' || result.cursor"
            class="more"
            @click="getMoreResult(result)"
          >
            <Icon
              class="more-icon"
              :file="searchIcon"
              width="12px"
              height="12px"
            />
            <div class="more-text">
              <span>{{ TUITranslateService.t("TUISearch.查看更多") }}</span>
              <span>{{
                TUITranslateService.t(`TUISearch.${result.label}`)
              }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>
    <div
      v-if="isResultDetailShow || props.searchType === 'conversation'"
      :class="[
        'tui-search-result-detail',
        props.searchType === 'conversation' &&
          'tui-search-result-in-conversation',
      ]"
    >
      <SearchResultLoading
        v-if="isSearchDetailLoading"
        :class="['search-result-loading', !isPC && 'search-result-loading-h5']"
      />
      <div
        v-if="
          !isSearchDetailLoading &&
          isResultDetailShow &&
          props.searchType !== 'conversation'
        "
        class="tui-search-message-header"
      >
        <div class="header-content">
          <div class="header-content-count normal">
            <span>{{ searchConversationMessageTotalCount }}</span>
            <span>{{ TUITranslateService.t("TUISearch.条与") }}</span>
          </div>
          <div class="header-content-keyword">
            <span v-for="(keyword, index) in keywordList" :key="index">
              <span class="normal">"</span>
              <span class="highlight">{{ keyword }}</span>
              <span class="normal">"</span>
            </span>
          </div>
          <div class="header-content-type normal">
            <span>{{ TUITranslateService.t("TUISearch.相关的") }}</span>
            <span>{{
              TUITranslateService.t(
                `TUISearch.${
                  currentSearchTabKey === "allMessage"
                    ? "结果"
                    : searchMessageTypeList[currentSearchTabKey].label
                }`
              )
            }}</span>
          </div>
        </div>
        <div
          class="header-enter"
          @click="
            enterConversation({ conversationID: currentSearchConversationID })
          "
        >
          <span>{{ TUITranslateService.t("TUISearch.进入聊天") }}</span>
          <Icon
            class="enter-icon"
            :file="enterIcon"
            width="14px"
            height="14px"
          />
        </div>
      </div>
      <div
        v-if="
          !isSearchDetailLoading &&
          searchConversationMessageList &&
          searchConversationMessageList[0]
        "
        class="tui-search-message-list"
      >
        <template
          v-if="
            props.searchType === 'global' ||
            (currentSearchTabKey !== 'imageMessage' &&
              currentSearchTabKey !== 'fileMessage')
          "
        >
          <div
            v-for="item in searchConversationMessageList"
            :key="generateVueRenderKey(item.ID)"
            :class="['list-item']"
          >
            <SearchResultItem
              :listItem="item"
              :listItemContent="item.getMessageContent()"
              :type="currentSearchTabKey"
              :displayType="generateResultItemDisplayType()"
              :keywordList="keywordList"
              @showResultDetail="showResultDetail"
              @navigateToChatPosition="navigateToChatPosition"
            />
          </div>
        </template>
        <!-- Search within a conversation - messages such as files, pictures, and videos need to be displayed in groups according to time -->
        <template v-else>
          <div
            v-for="group in searchConversationResultGroupByDate"
            :key="generateVueRenderKey(group.date)"
            :class="[
              'list-group',
              currentSearchTabKey === 'fileMessage'
                ? 'list-group-file'
                : 'list-group-image',
            ]"
          >
            <div :class="['list-group-date']">
              {{ group.date }}
            </div>
            <div
              v-for="item in group.list"
              :key="generateVueRenderKey(item.ID)"
              :class="['list-group-item']"
            >
              <SearchResultItem
                :listItem="item"
                :listItemContent="item.getMessageContent()"
                :type="currentSearchTabKey"
                :displayType="generateResultItemDisplayType()"
                :keywordList="keywordList"
                @showResultDetail="showResultDetail"
                @navigateToChatPosition="navigateToChatPosition"
              />
            </div>
          </div>
        </template>
        <div
          v-if="searchConversationResult && searchConversationResult.cursor"
          class="more"
          @click="getMoreResultInConversation"
        >
          <Icon
            class="more-icon"
            :file="searchIcon"
            width="12px"
            height="12px"
          />
          <div class="more-text">
            {{ TUITranslateService.t("TUISearch.查看更多历史记录") }}
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
<script setup lang="ts">
import {
  ref,
  watch,
  computed,
  onMounted,
  onUnmounted,
} from "../../../adapter-vue";
import {
  default as TUIChatEngine,
  TUITranslateService,
  TUIConversationService,
  TUIStore,
  StoreName,
  IMessageModel,
} from "@tencentcloud/chat-uikit-engine-lite";
import { TUIGlobal } from "@tencentcloud/universal-api";
import SearchResultItem from "./search-result-item/index.vue";
import SearchResultDefault from "./search-result-default/index.vue";
import SearchResultLoading from "./search-result-loading/index.vue";
import {
  searchMessageTypeList,
  searchMessageTypeDefault,
} from "../search-type-list";
import Icon from "../../common/Icon.vue";
import searchIcon from "../../../assets/icon/search.svg";
import enterIcon from "../../../assets/icon/right-icon.svg";
import { enterConversation, generateSearchResultYMD, debounce } from "../utils";
import { enableSampleTaskStatus } from "../../../utils/enableSampleTaskStatus";
import { isPC, isUniFrameWork } from "../../../utils/env";
import { matchesMemberKeyword } from "../../../utils/member-search";
import {
  SEARCH_TYPE,
  ISearchInputValue,
  ISearchMessageType,
  ISearchMessageTime,
} from "../type";
import {
  ISearchCloudMessageResult,
  ISearchResultListItem,
} from "../../../interface";

const props = defineProps({
  searchType: {
    type: String,
    default: "global", // "global" / "conversation"
    validator(value: string) {
      return ["global", "conversation"].includes(value);
    },
  },
});

// search params
const keywordList = ref<string[]>([]);
const messageTypeList = ref<string | string[]>(
  searchMessageTypeDefault[props.searchType as SEARCH_TYPE]?.value as string[]
);
const timePosition = ref<number>(0);
const timePeriod = ref<number>(0);
// Search by "and" after splitting the whole string by space
// For example: enter "111 222", search for messages with both 111 and 222, and also include messages that strictly search for "111 222"
const keywordListMatchType = ref<string>("and");

// current search tab key
const currentSearchTabKey = ref<string>(
  searchMessageTypeDefault[props.searchType as SEARCH_TYPE]?.key
);

// search results all
const searchResult = ref<{
  [propsName: string]: {
    key: string;
    label: string;
    list: ISearchResultListItem[];
    cursor: string | null;
  };
}>({});
const searchAllMessageList = ref<ISearchResultListItem[]>([]);
const searchAllMessageTotalCount = ref<number>(0);

// search results detail
const currentSearchConversationID = ref<string>("");
const searchConversationResult = ref<ISearchCloudMessageResult>();
const searchConversationMessageList = ref<IMessageModel[]>([]);
const searchConversationMessageTotalCount = ref<number>();
const localMessageList = ref<IMessageModel[]>([]);
const conversationList = ref<any[]>([]);
const LOCAL_MESSAGE_CACHE_TTL = 60 * 1000;
const LOCAL_MESSAGE_FETCH_MAX_PAGES = 10;
const LOCAL_MESSAGE_FETCH_CONCURRENCY = 4;
const conversationMessageCache = new Map<
  string,
  { messages: IMessageModel[]; fetchedAt: number }
>();
const pendingConversationMessageRequests = new Map<
  string,
  Promise<IMessageModel[]>
>();
let activeSearchRequestID = 0;

// search results for file messages/image and video messages, grouped by timeline
const searchConversationResultGroupByDate = ref<
  Array<{ date: string; list: IMessageModel[] }>
>([]);

// ui display control
const isResultDetailShow = ref<boolean>(false);
const isLoading = ref<boolean>(false);
const isSearchDetailLoading = ref<boolean>(false);
const isSearchDefaultShow = computed((): boolean => {
  if (isLoading.value) {
    return false;
  }
  if (props.searchType === "global") {
    if (
      !keywordList?.value?.length ||
      Object?.keys(searchResult.value)?.length
    ) {
      return false;
    } else {
      return true;
    }
  } else {
    if (searchConversationMessageList?.value?.length) {
      return false;
    } else {
      return true;
    }
  }
});

function onCurrentConversationIDUpdate(id: string) {
  props.searchType === "conversation" &&
    (currentSearchConversationID.value = id);
}

function onMessageListUpdate(list: IMessageModel[]) {
  const currentConversationID =
    TUIStore.getData(StoreName.CONV, "currentConversationID") || "";
  if (currentConversationID && Array.isArray(list)) {
    setConversationMessageCache(currentConversationID, list);
  }
  if (props.searchType === "conversation") {
    localMessageList.value = list || [];
  }
}

function onConversationListUpdate(list: any[]) {
  conversationList.value = list || [];
  pruneConversationMessageCache(list || []);
}

function onCurrentSearchInputValueUpdate(obj: ISearchInputValue) {
  if (obj?.searchType === props?.searchType) {
    keywordList.value = obj?.value ? obj.value.trim().split(/\s+/) : [];
  }
}

function onCurrentSearchMessageTypeUpdate(typeObject: ISearchMessageType) {
  if (typeObject?.searchType === props?.searchType) {
    currentSearchTabKey.value =
      typeObject?.value?.key ||
      searchMessageTypeDefault[props.searchType as SEARCH_TYPE]?.key;
    messageTypeList.value =
      typeObject?.value?.value ||
      searchMessageTypeDefault[props.searchType as SEARCH_TYPE]?.value;
  }
}

function onCurrentSearchMessageTimeUpdate(timeObject: ISearchMessageTime) {
  if (timeObject?.searchType === props?.searchType) {
    timePosition.value = timeObject?.value?.value?.timePosition;
    timePeriod.value = timeObject?.value?.value?.timePeriod;
  }
}

onMounted(() => {
  conversationList.value =
    TUIStore.getData(StoreName.CONV, "conversationList") || [];
  currentSearchConversationID.value =
    TUIStore.getData(StoreName.CONV, "currentConversationID") || "";
  if (props.searchType === "conversation") {
    localMessageList.value =
      TUIStore.getData(StoreName.CHAT, "messageList") || [];
    if (currentSearchConversationID.value && localMessageList.value.length) {
      setConversationMessageCache(
        currentSearchConversationID.value,
        localMessageList.value
      );
    }
  }
  TUIStore.watch(StoreName.CONV, {
    currentConversationID: onCurrentConversationIDUpdate,
    conversationList: onConversationListUpdate,
  });
  TUIStore.watch(StoreName.CHAT, {
    messageList: onMessageListUpdate,
  });
  TUIStore.watch(StoreName.SEARCH, {
    currentSearchInputValue: onCurrentSearchInputValueUpdate,
    currentSearchMessageType: onCurrentSearchMessageTypeUpdate,
    currentSearchMessageTime: onCurrentSearchMessageTimeUpdate,
  });
});

onUnmounted(() => {
  activeSearchRequestID += 1;
  (setMessageSearchResultListDebounce as any).cancel?.();
  TUIStore.unwatch(StoreName.CONV, {
    currentConversationID: onCurrentConversationIDUpdate,
    conversationList: onConversationListUpdate,
  });
  TUIStore.unwatch(StoreName.CHAT, {
    messageList: onMessageListUpdate,
  });
  TUIStore.unwatch(StoreName.SEARCH, {
    currentSearchInputValue: onCurrentSearchInputValueUpdate,
    currentSearchMessageType: onCurrentSearchMessageTypeUpdate,
    currentSearchMessageTime: onCurrentSearchMessageTimeUpdate,
  });
});

function extractLocalMessageSearchText(message: IMessageModel) {
  const content = message.getMessageContent?.() || {};
  switch (message.type) {
    case TUIChatEngine.TYPES.MSG_TEXT:
      return String(content.text || "");
    case TUIChatEngine.TYPES.MSG_FILE:
      return String(content.name || content.fileName || "");
    case TUIChatEngine.TYPES.MSG_IMAGE:
    case TUIChatEngine.TYPES.MSG_VIDEO:
      return String(content.name || content.fileName || content.url || "");
    case TUIChatEngine.TYPES.MSG_AUDIO:
      return String(content.second || "");
    case TUIChatEngine.TYPES.MSG_LOCATION:
      return String(content.description || content.address || "");
    case TUIChatEngine.TYPES.MSG_CUSTOM:
      return String(content.description || content.custom || "");
    case TUIChatEngine.TYPES.MSG_MERGER:
      return String(content.title || content.abstract || "");
    default:
      return "";
  }
}

function getSearchMessageShowName(message: Record<string, any>) {
  const friendList =
    (TUIStore.getData(StoreName.FRIEND, "friendList") as Array<{
      userID?: string;
      remark?: string;
    }>) || [];
  const matchedFriend = friendList.find(
    (friend) => friend.userID === message.from
  );
  return (
    matchedFriend?.remark ||
    message.nameCard ||
    message.nick ||
    message.from ||
    ""
  );
}

function getSearchMessageContent(message: Record<string, any>) {
  const payload = message?.payload || {};
  const showName = getSearchMessageShowName(message);

  switch (message?.type) {
    case TUIChatEngine.TYPES.MSG_TEXT:
      return {
        text: payload.text || "",
        showName,
      };
    case TUIChatEngine.TYPES.MSG_FILE:
      return {
        name: payload.fileName || "",
        size: payload.fileSize || "",
        url: payload.url || payload.fileUrl || payload.downloadUrl || "",
        showName,
      };
    case TUIChatEngine.TYPES.MSG_IMAGE: {
      const imageInfoArray = payload.imageInfoArray || [];
      const imageInfo =
        imageInfoArray[0] || imageInfoArray[imageInfoArray.length - 1] || {};
      return {
        url: imageInfo.url || payload.url || "",
        showName,
      };
    }
    case TUIChatEngine.TYPES.MSG_VIDEO:
      return {
        url:
          payload.snapshotUrl ||
          payload.videoUrl ||
          payload.downloadUrl ||
          payload.url ||
          "",
        name: payload.fileName || "",
        showName,
      };
    case TUIChatEngine.TYPES.MSG_AUDIO:
      return {
        second: payload.second || payload.duration || 0,
        showName,
      };
    case TUIChatEngine.TYPES.MSG_LOCATION:
      return {
        description: payload.description || payload.address || "",
        showName,
      };
    case TUIChatEngine.TYPES.MSG_CUSTOM:
      return {
        custom: payload.data || "",
        description: payload.description || "",
        showName,
      };
    case TUIChatEngine.TYPES.MSG_MERGER:
      return {
        title: payload.title || "",
        abstract: payload.abstract || "",
        showName,
      };
    default:
      return {
        showName,
      };
  }
}

function toSearchMessageModel(message: Record<string, any>) {
  const rawMessage = { ...message };
  return {
    ...rawMessage,
    getMessage: () => rawMessage,
    getMessageContent: () => getSearchMessageContent(rawMessage),
  } as unknown as IMessageModel;
}

function matchKeywords(text: string) {
  const normalizedText = text.toLowerCase();
  if (!keywordList.value.length) {
    return false;
  }
  return keywordList.value.every((keyword) =>
    normalizedText.includes(keyword.toLowerCase())
  );
}

function getConversationSearchText(conversation: any) {
  if (!conversation) {
    return "";
  }
  if (typeof conversation.getShowName === "function") {
    return String(conversation.getShowName() || "");
  }
  return String(
    conversation.showName ||
      conversation.remark ||
      conversation.userProfile?.nick ||
      conversation.groupProfile?.name ||
      conversation.nick ||
      conversation.name ||
      conversation.userID ||
      conversation.groupID ||
      conversation.conversationID ||
      ""
  );
}

function normalizeLocalContact(item: Record<string, any>) {
  const conversationID = String(item?.conversationID || "");
  const conversationShowName =
    typeof item?.getShowName === "function"
      ? item.getShowName()
      : item?.showName || item?.name || "";
  const conversationAvatar =
    typeof item?.getAvatar === "function" ? item.getAvatar() : "";

  return {
    ...item,
    userID:
      item?.userID ||
      item?.profile?.userID ||
      item?.userProfile?.userID ||
      (conversationID.startsWith("C2C") ? conversationID.slice(3) : ""),
    remark: item?.remark || item?.friendRemark || "",
    friendRemark: item?.friendRemark || item?.remark || "",
    nick:
      item?.nick ||
      item?.profile?.nick ||
      item?.userProfile?.nick ||
      conversationShowName ||
      "",
    avatar:
      item?.avatar ||
      item?.profile?.avatar ||
      item?.userProfile?.avatar ||
      conversationAvatar ||
      "",
    _wananID:
      item?._wananID ||
      item?.wananID ||
      item?.profile?.wananID ||
      item?.userProfile?.wananID ||
      "",
  };
}

function buildLocalContactResults() {
  if (
    props.searchType !== "global" ||
    currentSearchTabKey.value !== "allMessage" ||
    !keywordList.value.length
  ) {
    return [];
  }

  const friendList =
    (TUIStore.getData(StoreName.FRIEND, "friendList") as Array<
      Record<string, any>
    >) || [];
  const c2cConversationContacts = (conversationList.value || [])
    .filter((conversation) =>
      String(conversation?.conversationID || "").startsWith("C2C")
    )
    .map((conversation) => normalizeLocalContact(conversation));

  const mergedContacts = new Map<string, Record<string, any>>();
  [
    ...c2cConversationContacts,
    ...friendList.map(normalizeLocalContact),
  ].forEach((contact) => {
    const userID = String(contact?.userID || "");
    if (!userID) {
      return;
    }

    const previous = mergedContacts.get(userID) || {};
    mergedContacts.set(userID, {
      ...previous,
      ...contact,
      userID,
      remark: contact.remark || previous.remark || "",
      friendRemark: contact.friendRemark || previous.friendRemark || "",
      nick: contact.nick || previous.nick || "",
      avatar: contact.avatar || previous.avatar || "",
      _wananID: contact._wananID || previous._wananID || "",
    });
  });

  return Array.from(mergedContacts.values()).filter((contact) =>
    keywordList.value.every((keyword) => {
      const normalizedKeyword = keyword.trim().toLowerCase();
      if (!normalizedKeyword) {
        return true;
      }
      return (
        matchesMemberKeyword(contact, keyword) ||
        String(contact?._wananID || "")
          .toLowerCase()
          .includes(normalizedKeyword)
      );
    })
  );
}

function matchesSearchTarget(message: IMessageModel, conversation?: any) {
  const candidateTexts = [
    extractLocalMessageSearchText(message),
    String(message.getMessageContent?.()?.showName || ""),
    getConversationSearchText(conversation),
  ]
    .filter(Boolean)
    .map((text) => text.toLowerCase());

  if (!keywordList.value.length) {
    return false;
  }

  return keywordList.value.every((keyword) =>
    candidateTexts.some((text) => text.includes(keyword.toLowerCase()))
  );
}

function matchesTimeRange(message: IMessageModel) {
  if (!timePeriod.value || timePeriod.value <= 0) {
    return true;
  }

  const messageTime = message.time || 0;
  const endTime =
    timePosition.value && timePosition.value > 0
      ? timePosition.value
      : Math.floor(Date.now() / 1000);
  const startTime = endTime - timePeriod.value;
  return messageTime >= startTime && messageTime <= endTime;
}

function buildFilteredLocalMessages(
  sourceMessages: IMessageModel[],
  conversation?: any
) {
  const allowedMessageTypes = Array.isArray(messageTypeList.value)
    ? messageTypeList.value
    : [messageTypeList.value];
  const normalizedAllowedMessageTypes = allowedMessageTypes.map((type) =>
    String(type)
  );

  return (sourceMessages || [])
    .filter((message) => !message?.isDeleted && !message?.isRevoked)
    .filter((message) =>
      normalizedAllowedMessageTypes.includes(String(message.type))
    )
    .filter(matchesTimeRange)
    .filter((message) => matchesSearchTarget(message, conversation))
    .sort((prev, next) => (next.time || 0) - (prev.time || 0));
}

function applyFilteredConversationMessages(filteredMessages: IMessageModel[]) {
  searchConversationMessageList.value = filteredMessages;
  searchConversationMessageTotalCount.value = filteredMessages.length;
  searchConversationResult.value = {
    searchResultList: [],
    cursor: "",
    totalCount: filteredMessages.length,
  } as ISearchCloudMessageResult;

  if (
    currentSearchTabKey.value === "fileMessage" ||
    currentSearchTabKey.value === "imageMessage"
  ) {
    searchConversationResultGroupByDate.value =
      groupResultListByDate(filteredMessages);
  } else {
    searchConversationResultGroupByDate.value = [];
  }

  isLoading.value = false;
  isSearchDetailLoading.value = false;
}

function setLocalConversationSearchResult() {
  const currentConversation = (conversationList.value || []).find(
    (conversation) =>
      conversation?.conversationID === currentSearchConversationID.value
  );
  const filteredMessages = buildFilteredLocalMessages(
    localMessageList.value || [],
    currentConversation
  );

  applyFilteredConversationMessages(filteredMessages);
}

async function setLocalConversationDetailResult(
  conversationID: string,
  requestID: number
) {
  const targetConversation = (conversationList.value || []).find(
    (conversation) => conversation?.conversationID === conversationID
  );
  const conversationMessages = await loadLocalMessagesForConversation(
    conversationID
  );
  const filteredMessages = buildFilteredLocalMessages(
    conversationMessages,
    targetConversation
  );

  if (requestID !== activeSearchRequestID) {
    return;
  }
  applyFilteredConversationMessages(filteredMessages);
}

function setConversationMessageCache(
  conversationID: string,
  messages: IMessageModel[]
) {
  if (!conversationID) {
    return;
  }
  conversationMessageCache.set(conversationID, {
    messages: [...messages],
    fetchedAt: Date.now(),
  });
}

function getConversationMessageCache(conversationID: string) {
  const cached = conversationMessageCache.get(conversationID);
  if (!cached) {
    return undefined;
  }
  if (Date.now() - cached.fetchedAt > LOCAL_MESSAGE_CACHE_TTL) {
    conversationMessageCache.delete(conversationID);
    return undefined;
  }
  return cached.messages;
}

function pruneConversationMessageCache(list: Array<{ conversationID?: string }>) {
  const validConversationIDSet = new Set(
    list
      .map((item) => String(item?.conversationID || ""))
      .filter(Boolean)
  );

  Array.from(conversationMessageCache.keys()).forEach((conversationID) => {
    if (!validConversationIDSet.has(conversationID)) {
      conversationMessageCache.delete(conversationID);
      pendingConversationMessageRequests.delete(conversationID);
    }
  });
}

async function loadLocalMessagesForConversation(conversationID: string) {
  const cachedMessages = getConversationMessageCache(conversationID);
  if (cachedMessages) {
    return cachedMessages;
  }

  const pendingRequest = pendingConversationMessageRequests.get(conversationID);
  if (pendingRequest) {
    return pendingRequest;
  }

  const request = (async () => {
  const allMessages: IMessageModel[] = [];
  const seenMessageIDs = new Set<string>();
  let nextReqMessageID = "";
  let pageCount = 0;
  let isCompleted = false;

    while (!isCompleted && pageCount < LOCAL_MESSAGE_FETCH_MAX_PAGES) {
      const response = await TUIChatEngine.chat.getMessageList({
        conversationID,
        nextReqMessageID,
      });
      const list = (
        (response?.data?.messageList || []) as Array<Record<string, any>>
      ).map(toSearchMessageModel);
      list.forEach((message) => {
        if (!seenMessageIDs.has(message.ID)) {
          seenMessageIDs.add(message.ID);
          allMessages.push(message);
        }
      });

      nextReqMessageID = response?.data?.nextReqMessageID || "";
      isCompleted = Boolean(response?.data?.isCompleted) || !nextReqMessageID;
      pageCount += 1;
    }

    setConversationMessageCache(conversationID, allMessages);
    return allMessages;
  })();

  pendingConversationMessageRequests.set(conversationID, request);
  try {
    return await request;
  } finally {
    pendingConversationMessageRequests.delete(conversationID);
  }
}

async function mapWithConcurrency<T, R>(
  items: T[],
  limit: number,
  task: (item: T, index: number) => Promise<R | undefined>
) {
  const result: R[] = [];
  let currentIndex = 0;

  const workers = Array.from({
    length: Math.min(limit, items.length),
  }).map(async () => {
    while (currentIndex < items.length) {
      const workerIndex = currentIndex;
      currentIndex += 1;
      const taskResult = await task(items[workerIndex], workerIndex);
      if (typeof taskResult !== "undefined") {
        result.push(taskResult);
      }
    }
  });

  await Promise.all(workers);
  return result;
}

async function setLocalGlobalSearchResult(requestID: number) {
  const localContactResults = buildLocalContactResults();
  if (localContactResults.length > 0) {
    searchResult.value = {
      ...searchResult.value,
      contact: {
        key: "contact",
        label: "联系人",
        list: localContactResults as ISearchResultListItem[],
        cursor: null,
      },
    };
  } else {
    delete searchResult.value.contact;
  }

  const conversationCandidates = (conversationList.value || []).filter(
    (conversation) => conversation?.conversationID
  );
  const allowedMessageTypes = Array.isArray(messageTypeList.value)
    ? messageTypeList.value
    : [messageTypeList.value];
  const normalizedAllowedMessageTypes = allowedMessageTypes.map((type) =>
    String(type)
  );
  const groupedResults = await mapWithConcurrency(
    conversationCandidates,
    LOCAL_MESSAGE_FETCH_CONCURRENCY,
    async (conversation) => {
      if (requestID !== activeSearchRequestID) {
        return undefined;
      }

      let messages: IMessageModel[] = [];
      try {
        messages = await loadLocalMessagesForConversation(
          conversation.conversationID
        );
      } catch (error) {
        console.warn(
          "load local messages for search failed",
          conversation.conversationID,
          error
        );
        return undefined;
      }

      if (requestID !== activeSearchRequestID) {
        return undefined;
      }

      const matchedMessages = messages
        .filter((message) => !message?.isDeleted && !message?.isRevoked)
        .filter((message) =>
          normalizedAllowedMessageTypes.includes(String(message.type))
        )
        .filter(matchesTimeRange)
        .filter((message) => matchesSearchTarget(message, conversation))
        .sort((prev, next) => (next.time || 0) - (prev.time || 0));

      if (!matchedMessages.length) {
        return undefined;
      }

      return {
        conversationID: conversation.conversationID,
        messageCount: matchedMessages.length,
        messageList: matchedMessages,
        conversation,
      };
    }
  ) as Array<{
    conversationID: string;
    messageCount: number;
    messageList: IMessageModel[];
    conversation: any;
  }>;

  if (requestID !== activeSearchRequestID) {
    return;
  }

  groupedResults.sort((prev, next) => {
    const prevTime = prev.messageList[0]?.time || 0;
    const nextTime = next.messageList[0]?.time || 0;
    return nextTime - prevTime;
  });

  searchAllMessageList.value = groupedResults as ISearchResultListItem[];
  searchAllMessageTotalCount.value = groupedResults.reduce(
    (total, item) => total + item.messageCount,
    0
  );

  const key = currentSearchTabKey.value;
  if (
    groupedResults.length &&
    currentSearchTabKey.value !== "contact" &&
    currentSearchTabKey.value !== "group"
  ) {
    searchResult.value = Object.assign({}, searchResult.value, {
      [key]: {
        key,
        label: searchMessageTypeList[key].label,
        list:
          currentSearchTabKey.value === "allMessage"
            ? groupedResults.slice(0, 3)
            : groupedResults,
        cursor: null,
      },
    });
  } else {
    delete searchResult.value[key];
  }

  isLoading.value = false;
  isSearchDetailLoading.value = false;
}

async function setMessageSearchResult(option?: {
  conversationID?: string | undefined;
}) {
  activeSearchRequestID += 1;
  const requestID = activeSearchRequestID;

  if (props.searchType === "conversation") {
    setLocalConversationSearchResult();
    return;
  }

  if (option?.conversationID) {
    try {
      await setLocalConversationDetailResult(option.conversationID, requestID);
    } catch (error) {
      console.warn(
        "load local conversation detail for search failed",
        option.conversationID,
        error
      );
      if (requestID === activeSearchRequestID) {
        applyFilteredConversationMessages([]);
      }
    }
    return;
  }

  try {
    await setLocalGlobalSearchResult(requestID);
  } catch (error) {
    console.warn("search local messages failed", error);
    if (requestID === activeSearchRequestID) {
      searchResult.value = {};
      searchAllMessageList.value = [];
      searchAllMessageTotalCount.value = 0;
    }
  } finally {
    if (requestID === activeSearchRequestID) {
      isLoading.value = false;
      isSearchDetailLoading.value = false;
    }
  }
}

const setMessageSearchResultListDebounce = debounce(
  setMessageSearchResult,
  500
);

const resetSearchResult = () => {
  activeSearchRequestID += 1;
  searchResult.value = {};
  searchConversationResult.value = {} as ISearchCloudMessageResult;
  searchConversationMessageList.value = [];
  searchConversationResultGroupByDate.value = [];
};

watch(
  () => [
    keywordList.value,
    currentSearchTabKey.value,
    timePosition.value,
    timePeriod.value,
  ],
  (newValue, oldValue) => {
    if (newValue === oldValue) {
      return;
    }
    if (!keywordList?.value?.length) {
      resetSearchResult();
      return;
    }
    isLoading.value = true;
    if (props.searchType === "conversation") {
      resetSearchResult();
      setLocalConversationSearchResult();
    } else {
      if (
        oldValue &&
        oldValue[1] === "all" &&
        newValue &&
        newValue[1] === "allMessage"
      ) {
        searchResult?.value["allMessage"]?.list &&
          (searchResult.value["allMessage"].list = searchAllMessageList?.value);
        Object?.keys(searchResult?.value)?.forEach((key: string) => {
          if (key !== "allMessage") {
            delete searchResult?.value[key];
          }
        });
        isLoading.value = false;
        return;
      } else {
        isResultDetailShow.value = false;
        resetSearchResult();
      }
      setMessageSearchResultListDebounce();
    }
  },
  { immediate: true }
);

watch(
  () => localMessageList.value,
  () => {
    if (props.searchType === "conversation" && keywordList.value.length) {
      setLocalConversationSearchResult();
    }
  },
  { deep: true }
);

const getMoreResult = (result: {
  key: string;
  label: string;
  list: ISearchResultListItem[];
  cursor: string | null;
}) => {
  if (isLoading.value || isSearchDetailLoading.value) {
    return;
  }

  if (currentSearchTabKey.value === "all") {
    // View more at this time: Switch to the result corresponding to the corresponding result to display the full search results of its type
    TUIStore.update(StoreName.SEARCH, "currentSearchMessageType", {
      value: searchMessageTypeList[result.key],
      searchType: props.searchType,
    });
  } else {
    void setMessageSearchResult();
  }
};

const getMoreResultInConversation = () => {
  if (isSearchDetailLoading.value) {
    return;
  }

  void setMessageSearchResult({
    conversationID: currentSearchConversationID?.value,
  });
};

const showResultDetail = (
  isShow: boolean,
  targetType?: string,
  targetResult?: IMessageModel | ISearchResultListItem
) => {
  if (isSearchDetailLoading.value) {
    return;
  }

  isResultDetailShow.value = isShow;
  if (targetType) {
    TUIStore.update(StoreName.SEARCH, "currentSearchMessageType", {
      value: searchMessageTypeList[targetType],
      searchType: props.searchType,
    });
  }
  currentSearchConversationID.value =
    (targetResult as ISearchResultListItem)?.conversation?.conversationID || "";
  searchConversationMessageTotalCount.value = (
    targetResult as ISearchResultListItem
  )?.messageCount;
  if (targetResult) {
    isSearchDetailLoading.value = true;
    setMessageSearchResultListDebounce({
      conversationID: currentSearchConversationID.value,
    });
  }
};

const generateListItemClass = (item: ISearchResultListItem): string[] => {
  return currentSearchConversationID.value ===
    item?.conversation?.conversationID
    ? ["list-item", "list-item-selected"]
    : ["list-item"];
};

const generateResultItemDisplayType = (): string => {
  if (
    props.searchType === "conversation" &&
    currentSearchTabKey.value === "fileMessage"
  ) {
    return "file";
  } else if (
    props.searchType === "conversation" &&
    currentSearchTabKey.value === "imageMessage"
  ) {
    return "image";
  } else if (isPC) {
    return "bubble";
  } else {
    return "info";
  }
};

const groupResultListByDate = (
  messageList: IMessageModel[]
): Array<{ date: string; list: IMessageModel[] }> => {
  const result: Array<{ date: string; list: IMessageModel[] }> = [];
  if (!messageList?.length) {
    return result;
  } else if (messageList?.length === 1) {
    result.push({
      date: generateSearchResultYMD(messageList[0]?.time),
      list: messageList,
    });
    return result;
  }
  let prevYMD = "";
  let currYMD = "";
  for (let i = 0; i < messageList?.length; i++) {
    currYMD = generateSearchResultYMD(messageList[i]?.time);
    if (prevYMD !== currYMD) {
      result.push({ date: currYMD, list: [messageList[i]] });
    } else {
      result[result?.length - 1]?.list?.push(messageList[i]);
    }
    prevYMD = currYMD;
  }
  return result;
};

const navigateToChatPosition = (message: IMessageModel) => {
  if (props.searchType === "global") {
    // Global search, close the search container
    TUIStore.update(StoreName.SEARCH, "currentSearchingStatus", {
      isSearching: false,
      searchType: props.searchType,
    });
    // switch conversation
    TUIConversationService.switchConversation(message?.conversationID).then(
      () => {
        TUIStore.update(StoreName.CHAT, "messageSource", message);
        isUniFrameWork &&
          TUIGlobal?.navigateTo({
            url: "/TUIKit/components/TUIChat/index",
          });
      }
    );
  } else if (props.searchType === "conversation") {
    // Search in conversation, close the search container
    TUIStore.update(StoreName.SEARCH, "isShowInConversationSearch", false);
    TUIStore.update(StoreName.CHAT, "messageSource", message);
    isUniFrameWork && TUIGlobal?.navigateBack();
  }
};

const generateVueRenderKey = (value: string): string => {
  return `${currentSearchTabKey.value}-${value}`;
};
</script>
<style lang="scss" scoped src="./style/index.scss"></style>
