<template>
  <div ref="conversationListInnerDomRef" class="tui-conversation-list">
    <ActionsMenu
      v-if="isShowOverlay"
      :selectedConversation="currentConversation"
      :actionsMenuPosition="actionsMenuPosition"
      :selectedConversationDomRect="currentConversationDomRect"
      @closeConversationActionMenu="closeConversationActionMenu"
    />
    <div
      v-for="(conversation, index) in conversationList"
      :id="`convlistitem-${index}`"
      :key="index"
      :class="[
        'tui-conversation-content',
        isMobile && 'tui-conversation-content-h5 disable-select',
      ]"
    >
      <div
        :class="[
          isPC && 'isPC',
          'tui-conversation-item',
          currentConversationID === conversation.conversationID &&
            'tui-conversation-item-selected',
          conversation.isPinned && 'tui-conversation-item-pinned',
        ]"
        @click="enterConversationChat(conversation)"
        @longpress="showConversationActionMenu($event, conversation, index)"
        @contextmenu="
          showConversationActionMenu($event, conversation, index, true)
        "
      >
        <aside class="left">
          <Avatar
            useSkeletonAnimation
            :url="conversation.getAvatar()"
            size="30px"
          />
          <span
            v-if="conversation.unreadCount > 0 && !conversation.isMuted"
            class="num"
          >
            {{
              conversation.unreadCount > 99 ? "99+" : conversation.unreadCount
            }}
          </span>
          <span
            v-if="conversation.unreadCount > 0 && conversation.isMuted"
            class="num-notify"
          />
        </aside>
        <div class="content">
          <div class="content-header">
            <label class="content-header-label">
              <p class="name">
                <span class="name-text">{{ conversation.getShowName() }}</span>
              </p>
            </label>
            <div class="middle-box">
              <span
                v-if="
                  conversation.draftText &&
                  conversation.conversationID !== currentConversationID
                "
                class="middle-box-draft"
                >{{ TUITranslateService.t("TUIChat.[草稿]") }}</span
              >
              <span
                v-else-if="
                  conversation.type === 'GROUP' &&
                  conversation.groupAtInfoList &&
                  conversation.groupAtInfoList.length > 0
                "
                class="middle-box-at"
                >{{ conversation.getGroupAtInfo() }}</span
              >
              <span
                v-else-if="getConversationOnlineText(conversation)"
                :class="[
                  'conversation-online-text',
                  isConversationPeerOnline(conversation)
                    ? 'conversation-online-text-online'
                    : 'conversation-online-text-offline',
                ]"
                >{{ getConversationOnlineText(conversation) }}</span
              >
              <div class="middle-box-content">
                {{ getConversationPreviewText(conversation) }}
              </div>
            </div>
          </div>
          <div class="content-footer">
            <span class="time">{{ conversation.getLastMessage("time") }}</span>
            <Icon v-if="conversation.isMuted" :file="muteIcon" />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
// @ts-nocheck
interface IUserStatus {
  statusType: number;
  customStatus: string;
}

interface IUserStatusMap {
  [userID: string]: IUserStatus;
}

import { ref, onMounted, onUnmounted, watch } from "../../../adapter-vue";
import TUIChatEngine, {
  TUIStore,
  StoreName,
  TUIConversationService,
  TUITranslateService,
  IConversationModel,
} from "@tencentcloud/chat-uikit-engine-lite";
import {
  TUIGlobal,
  isIOS,
  addLongPressListener,
} from "@tencentcloud/universal-api";
import Icon from "../../common/Icon.vue";
import Avatar from "../../common/Avatar/index.vue";
import ActionsMenu from "../actions-menu/index.vue";
import muteIcon from "../../../assets/icon/mute.svg";
import { getConversationCallPreviewText } from "@/features/call/doMattxCallProtocol";
import { isPC, isH5, isUniFrameWork, isMobile } from "../../../utils/env";
import { isUserOnlineStatus } from "@/features/presence/invisible";
import { getConversationCustomPins, getUserLastOnline } from "@/api";
import {
  applyLocalConversationPins,
  compareLocalConversationPins,
  CONVERSATION_LOCAL_PINS_UPDATED_KEY,
  toConversationCustomPinMap,
  type ConversationCustomPinMap,
} from "@/features/conversation/localPins";

import { resolveLastOnlineAt, formatLastOnlineText } from "@/features/presence/last-online";

const emits = defineEmits(["handleSwitchConversation", "getPassingRef"]);
const currentConversation = ref<IConversationModel>();
const currentConversationID = ref<string>();
const currentConversationDomRect = ref<DOMRect>();
const isShowOverlay = ref<boolean>(false);
const conversationList = ref<IConversationModel[]>([]);
const conversationListDomRef = ref<HTMLElement | undefined>();
const conversationListInnerDomRef = ref<HTMLElement | undefined>();
const actionsMenuPosition = ref<{
  top: number;
  left: number | undefined;
  conversationHeight: number | undefined;
}>({
  top: 0,
  left: undefined,
  conversationHeight: undefined,
});
const displayOnlineStatus = ref(false);
const userOnlineStatusMap = ref<IUserStatusMap>();
const rawConversationList = ref<IConversationModel[]>([]);
const localConversationPinMap = ref<ConversationCustomPinMap>({});

const lastSeenByUser = ref<Record<string, string>>({});
const observedOfflineByUser: Record<string, number> = {};
const presenceNow = ref(Date.now());
let presenceTimer: ReturnType<typeof setInterval> | undefined;
let presenceFetchPending = false;
let presenceDisposed = false;

// Watch only the peer IDs, not message previews, to avoid fetching on every message.
watch(() => displayOnlineStatus.value
  ? conversationList.value.map(getConversationUserID).filter(Boolean).sort().join(",")
  : "", () => { void refreshConversationLastOnline(); });

async function refreshConversationLastOnline() {
  if (presenceDisposed || presenceFetchPending || !displayOnlineStatus.value) return;
  const ids = conversationList.value.map(getConversationUserID).filter(Boolean);
  if (!ids.length) return;
  presenceFetchPending = true;
  try {
    const items = await getUserLastOnline(ids);
    if (presenceDisposed) return;
    const next = { ...lastSeenByUser.value };
    for (const item of items) {
      next[item.im_user_id] = resolveLastOnlineAt(item, observedOfflineByUser[item.im_user_id]);
    }
    lastSeenByUser.value = next;
  } catch (error) {
    console.warn("refresh conversation last online failed", error);
  } finally {
    presenceFetchPending = false;
  }
}

let lastestOpenActionsMenuTime: number | null = null;
const GROUP_AT_JUMP_STORE_KEY = "groupAtJumpMap";

onMounted(() => {
  presenceTimer = setInterval(() => {
    presenceNow.value = Date.now();
    if (!document.hidden) void refreshConversationLastOnline();
  }, 30000);
  TUIStore.watch(StoreName.CONV, {
    currentConversationID: onCurrentConversationIDUpdated,
    conversationList: onConversationListUpdated,
    currentConversation: onCurrentConversationUpdated,
  });

  TUIStore.watch(StoreName.USER, {
    displayOnlineStatus: onDisplayOnlineStatusUpdated,
    userStatusList: onUserStatusListUpdated,
  });
  TUIStore.watch(StoreName.CUSTOM, {
    [CONVERSATION_LOCAL_PINS_UPDATED_KEY]: onConversationLocalPinsUpdated,
  });

  const storeConversationList = TUIStore.getData(StoreName.CONV, "conversationList");
  if (Array.isArray(storeConversationList)) {
    onConversationListUpdated(storeConversationList);
  }
  currentConversationID.value =
    TUIStore.getData(StoreName.CONV, "currentConversationID") || "";
  syncOnlineStatusFromStore();
  void refreshLocalConversationPins();

  if (!isUniFrameWork && isIOS && !isPC) {
    addLongPressHandler();
  }
});

onUnmounted(() => {
  presenceDisposed = true;
  clearInterval(presenceTimer);
  TUIStore.unwatch(StoreName.CONV, {
    currentConversationID: onCurrentConversationIDUpdated,
    conversationList: onConversationListUpdated,
    currentConversation: onCurrentConversationUpdated,
  });

  TUIStore.unwatch(StoreName.USER, {
    displayOnlineStatus: onDisplayOnlineStatusUpdated,
    userStatusList: onUserStatusListUpdated,
  });
  TUIStore.unwatch(StoreName.CUSTOM, {
    [CONVERSATION_LOCAL_PINS_UPDATED_KEY]: onConversationLocalPinsUpdated,
  });
});

const isShowUserOnlineStatus = (conversation: IConversationModel): boolean => {
  return (
    displayOnlineStatus.value &&
    conversation.type === TUIChatEngine.TYPES.CONV_C2C
  );
};

const getConversationPreviewText = (conversation: IConversationModel): string => {
  return getConversationCallPreviewText(conversation);
};

const isConversationPeerOnline = (
  conversation: IConversationModel
): boolean => {
  const userID = getConversationUserID(conversation);
  return Boolean(
    userID && isUserOnlineStatus(userOnlineStatusMap.value?.[userID])
  );
};

const getConversationOnlineText = (conversation: IConversationModel): string => {
  if (!isShowUserOnlineStatus(conversation)) {
    return "";
  }
  if (isConversationPeerOnline(conversation)) return "在线";
  const id = getConversationUserID(conversation);
  return formatLastOnlineText(lastSeenByUser.value[id], presenceNow.value) || "离线";
};

function getConversationUserID(conversation: IConversationModel): string {
  if (conversation.type !== TUIChatEngine.TYPES.CONV_C2C) {
    return "";
  }
  const profileUserID = String(conversation.userProfile?.userID || "").trim();
  if (profileUserID) {
    return profileUserID;
  }

  const conversationID = String(conversation.conversationID || "");
  const prefix = TUIChatEngine.TYPES.CONV_C2C || "C2C";
  return conversationID.startsWith(prefix)
    ? conversationID.slice(prefix.length)
    : "";
}

function syncOnlineStatusFromStore() {
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

const showConversationActionMenu = (
  event: Event,
  conversation: IConversationModel,
  index: number,
  isContextMenuEvent?: boolean
) => {
  if (isContextMenuEvent) {
    event.preventDefault();
    if (isUniFrameWork) {
      return;
    }
  }
  currentConversation.value = conversation;
  lastestOpenActionsMenuTime = Date.now();
  getActionsMenuPosition(event, index);
};

const closeConversationActionMenu = () => {
  // Prevent continuous triggering of overlay tap events
  if (
    lastestOpenActionsMenuTime &&
    Date.now() - lastestOpenActionsMenuTime > 300
  ) {
    currentConversation.value = undefined;
    isShowOverlay.value = false;
  }
};

const getActionsMenuPosition = (event: Event, index: number) => {
  if (isUniFrameWork) {
    if (typeof conversationListDomRef.value === "undefined") {
      emits("getPassingRef", conversationListDomRef);
    }
    const query = TUIGlobal?.createSelectorQuery().in(
      conversationListDomRef.value
    );
    query
      .select(`#convlistitem-${index}`)
      .boundingClientRect((data) => {
        if (data) {
          actionsMenuPosition.value = {
            // The uni-page-head of uni-h5 is not considered a member of the viewport, so the height of the head is manually increased.
            top: data.bottom + (isH5 ? 44 : 0),
            // @ts-expect-error in uniapp event has touches property
            left: event.touches[0].pageX,
            conversationHeight: data.height,
          };
          isShowOverlay.value = true;
        }
      })
      .exec();
  } else {
    const rect =
      (
        (event.currentTarget || event.target) as HTMLElement
      )?.getBoundingClientRect() || {};
    if (rect) {
      actionsMenuPosition.value = {
        top: rect.bottom,
        left: isPC ? (event as MouseEvent).clientX : undefined,
        conversationHeight: rect.height,
      };
    }
    isShowOverlay.value = true;
  }
};

const enterConversationChat = (conversation: IConversationModel) => {
  const conversationID = conversation?.conversationID || "";
  cacheGroupAtJumpsBeforeSwitch(conversation);
  emits("handleSwitchConversation", conversationID);
  TUIConversationService.switchConversation(conversationID);
};

function normalizeAtInfoTypeList(info: any) {
  const raw =
    info?.atType ??
    info?.groupAtType ??
    info?.GroupAtType ??
    info?.atTypeArray ??
    info?.groupAtTypeArray ??
    info?.AtTypeArray ??
    info?.AtFlagList;
  const source = Array.isArray(raw) ? raw : [raw];
  return source
    .map((item) => Number(item))
    .filter((item) => Number.isFinite(item));
}

function getAtInfoSeq(info: any) {
  return String(
    info?.seq ||
      info?.sequence ||
      info?.messageSequence ||
      info?.atMsgSeq ||
      info?.AtMsgSeq ||
      info?.MsgSeq ||
      ""
  ).trim();
}

function getAtInfoMsgID(info: any) {
  return String(info?.msgID || info?.messageID || info?.ID || "").trim();
}

function getAtJumpFromAtInfo(info: any) {
  const seq = getAtInfoSeq(info);
  const msgID = getAtInfoMsgID(info);
  if (!seq && !msgID) {
    return undefined;
  }
  const atTypes = normalizeAtInfoTypeList(info);
  if (atTypes.length && !atTypes.some((type) => [1, 2, 3].includes(type))) {
    return undefined;
  }
  return {
    seq,
    msgID,
    atAll: atTypes.includes(2) || atTypes.includes(3),
  };
}

function cacheGroupAtJumpsBeforeSwitch(conversation: IConversationModel) {
  if (
    conversation?.type !== TUIChatEngine.TYPES.CONV_GROUP ||
    !conversation.conversationID
  ) {
    return;
  }

  const groupAtInfoList = Array.isArray(conversation.groupAtInfoList)
    ? conversation.groupAtInfoList
    : [];
  const jumpList = groupAtInfoList
    .map((info: any) => getAtJumpFromAtInfo(info))
    .filter(Boolean);

  const storeMap = {
    ...(TUIStore.getData(StoreName.CUSTOM, GROUP_AT_JUMP_STORE_KEY) || {}),
  };
  if (jumpList.length) {
    storeMap[conversation.conversationID] = jumpList;
  } else {
    delete storeMap[conversation.conversationID];
  }
  TUIStore.update(StoreName.CUSTOM, GROUP_AT_JUMP_STORE_KEY, storeMap);
}

function addLongPressHandler() {
  if (!conversationListInnerDomRef.value) {
    return;
  }
  addLongPressListener({
    element: conversationListInnerDomRef.value,
    onLongPress: (event, target) => {
      const index = (
        Array.from(conversationListInnerDomRef.value!.children) as HTMLElement[]
      ).indexOf(target!);
      showConversationActionMenu(event, conversationList.value[index], index);
    },
    options: {
      eventDelegation: {
        subSelector: ".tui-conversation-content",
      },
    },
  });
}

function onCurrentConversationUpdated(conversation: IConversationModel) {
  currentConversation.value = conversation;
}

function onConversationListUpdated(list: IConversationModel[]) {
  rawConversationList.value = list || [];
  applyConversationPinsToList();
}

function sortConversationList(list: IConversationModel[]) {
  const nextList = [...list];
  return nextList.sort((a, b) => {
    const pinCompare = compareLocalConversationPins(a, b);
    if (pinCompare !== 0) {
      return pinCompare;
    }

    const aHasDraft = Boolean(a.draftText);
    const bHasDraft = Boolean(b.draftText);
    if (aHasDraft !== bHasDraft) {
      return aHasDraft ? -1 : 1;
    }

    const aAt = a.groupAtInfoList?.length ?? 0;
    const bAt = b.groupAtInfoList?.length ?? 0;
    if (aAt !== bAt) {
      return bAt - aAt;
    }

    const aTime = Number(
      a.lastMessage?.lastTime || a.lastMessage?.lastTimeStamp || 0
    );
    const bTime = Number(
      b.lastMessage?.lastTime || b.lastMessage?.lastTimeStamp || 0
    );
    if (aTime !== bTime) {
      return bTime - aTime;
    }

    return (a.conversationID || "").localeCompare(b.conversationID || "");
  });
}

function onCurrentConversationIDUpdated(id: string) {
  currentConversationID.value = id;
}

function onDisplayOnlineStatusUpdated(status: boolean) {
  displayOnlineStatus.value = status;
}

function onUserStatusListUpdated(list: Map<string, IUserStatus>) {
  if (!(list instanceof Map)) {
    return;
  }
  let changedToOffline = false;
  list.forEach((status, id) => {
    if (isUserOnlineStatus(status)) {
      delete observedOfflineByUser[id];
    } else if (isUserOnlineStatus(userOnlineStatusMap.value?.[id])) {
      observedOfflineByUser[id] = Date.now();
      lastSeenByUser.value[id] = new Date(observedOfflineByUser[id]).toISOString();
      changedToOffline = true;
    }
  });
  if (changedToOffline) void refreshConversationLastOnline();
  userOnlineStatusMap.value = [...list.entries()].reduce(
    (obj, [key, value]) => {
      obj[key] = value;
      return obj;
    },
    {} as IUserStatusMap
  );
}

async function refreshLocalConversationPins() {
  try {
    localConversationPinMap.value = toConversationCustomPinMap(
      await getConversationCustomPins()
    );
  } catch {
    localConversationPinMap.value = {};
  }
  applyConversationPinsToList();
}

function applyConversationPinsToList() {
  conversationList.value = sortConversationList(
    applyLocalConversationPins(
      rawConversationList.value,
      localConversationPinMap.value
    )
  );
}

function onConversationLocalPinsUpdated() {
  void refreshLocalConversationPins();
}

// Expose to the parent component and close actionsMenu when a sliding event is detected
defineExpose({ closeChildren: closeConversationActionMenu });
</script>

<style lang="scss" scoped src="./style/index.scss"></style>
<style lang="scss" scoped>
.disable-select {
  -webkit-touch-callout: none;
  -webkit-user-select: none;
  -khtml-user-select: none;
  -moz-user-select: none;
  -ms-user-select: none;
  user-select: none;
}
</style>
