<!-- Used to display the search results of [Contacts]/[Groups]/[All Conversations], which is a display of user/group/conversation dimensions -->
<template>
  <div
    :class="[
      'search-result-list-item',
      !isPC && 'search-result-list-item-h5',
      'search-result-list-item-' + displayType,
      isHovering && 'hover-' + displayType,
    ]"
    @click="onResultItemClicked"
    @mouseenter="setHoverStatus(true)"
    @mouseleave="setHoverStatus(false)"
  >
    <div
      v-if="displayType === 'info' || displayType === 'bubble'"
      :class="[displayType]"
    >
      <div :class="displayType + '-left'">
        <img
          :class="displayType + '-left-avatar'"
          :src="avatarForShow || ''"
          onerror="this.onerror=null;this.src='https://web.sdk.qcloud.com/component/TUIKit/assets/avatar_21.png'"
        />
      </div>
      <div :class="[displayType + '-main']">
        <div :class="[displayType + '-main-name']">
          <span class="name-text">{{ nameForShow }}</span>
          <span
            v-if="relationStatusLabel"
            :class="['relation-badge', `relation-badge-${relationStatusTone}`]"
          >
            {{ TUITranslateService.t(`TUISearch.${relationStatusLabel}`) }}
          </span>
        </div>
        <div :class="[displayType + '-main-content']">
          <div v-if="entityMetaVisible" class="entity-meta">
            <span class="entity-meta-id">
              {{ entityIdLabel }}: {{ entityIdValue }}
            </span>
            <span v-if="entitySecondaryText" class="entity-meta-type">
              {{ entitySecondaryText }}
            </span>
          </div>
          <MessageAbstractText
            v-else-if="
              displayType === 'info' || listItem.type === TYPES.MSG_TEXT
            "
            :content="contentForShow"
            :highlightType="displayType === 'info' ? 'font' : 'background'"
            :displayType="displayType"
          />
          <MessageAbstractFile
            v-else-if="listItem.type === TYPES.MSG_FILE"
            :contentText="contentForShow"
            :messageContent="listItemContent"
            :displayType="displayType"
          />
          <div v-else-if="listItem.type === TYPES.MSG_IMAGE" />
          <div v-else-if="listItem.type === TYPES.MSG_VIDEO" />
          <MessageAbstractCustom
            v-else-if="listItem.type === TYPES.MSG_CUSTOM"
            :contentText="contentForShow"
            :message="listItem"
            :messageContent="listItemContent"
          />
          <div v-else>
            {{ getMessageAbstractType(listItem) }}
          </div>
        </div>
      </div>
      <div :class="displayType + '-right'">
        <div :class="displayType + '-right-time'">
          {{ timeForShow }}
        </div>
        <div
          v-if="displayType === 'bubble' && isHovering"
          :class="displayType + '-right-to'"
          @click.stop="navigateToChatPosition"
        >
          {{ TUITranslateService.t("TUISearch.定位到聊天位置") }}
        </div>
      </div>
    </div>
    <div v-else-if="displayType === 'file'" :class="[displayType]">
      <div :class="[displayType + '-header']">
        <img
          :class="displayType + '-header-avatar'"
          :src="avatarForShow"
          onerror="this.onerror=null;this.src='https://web.sdk.qcloud.com/component/TUIKit/assets/avatar_21.png'"
        />
        <div :class="[displayType + '-header-name']">
          {{ nameForShow }}
        </div>
        <div
          v-if="isHovering"
          :class="displayType + '-header-to'"
          @click.stop="navigateToChatPosition"
        >
          {{ TUITranslateService.t("TUISearch.定位到聊天位置") }}
        </div>
        <div :class="displayType + '-header-time'">
          {{ timeForShow }}
        </div>
      </div>
      <div :class="[displayType + '-main-content']">
        <MessageAbstractFile
          :contentText="contentForShow"
          :messageContent="listItemContent"
          displayType="bubble"
        />
      </div>
    </div>
    <div v-else-if="displayType === 'image'" :class="[displayType]">
      <div class="image-container" @click.stop="navigateToChatPosition">
        <MessageAbstractImage
          v-if="listItem.type === TYPES.MSG_IMAGE"
          :messageContent="listItemContent"
        />
        <MessageAbstractVideo
          v-else-if="listItem.type === TYPES.MSG_VIDEO"
          :messageContent="listItemContent"
        />
        <div v-if="isHovering" class="image-container-hover">
          <div class="image-container-hover-text">
            {{ TUITranslateService.t("TUISearch.定位到聊天位置") }}
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
<script setup lang="ts">
// @ts-nocheck
import TUIChatEngine, {
  TUITranslateService,
  IMessageModel,
  TUIStore,
  StoreName,
} from "@tencentcloud/chat-uikit-engine-lite";
import { ref, watchEffect, withDefaults } from "../../../../adapter-vue";
import MessageAbstractText from "./message-abstract/message-abstract-text.vue";
import MessageAbstractFile from "./message-abstract/message-abstract-file.vue";
import MessageAbstractCustom from "./message-abstract/message-abstract-custom.vue";
import MessageAbstractImage from "./message-abstract/message-abstract-image.vue";
import MessageAbstractVideo from "./message-abstract/message-abstract-video.vue";
import {
  generateSearchResultShowName,
  generateSearchResultAvatar,
  generateSearchResultShowContent,
  generateSearchResultTime,
  enterConversation,
} from "../../utils";
import {
  messageTypeAbstractMap,
  searchResultItemDisplayTypeValues,
  searchMessageTypeValues,
  IHighlightContent,
} from "../../type";
import { ISearchResultListItem } from "../../../../interface";
import { isPC } from "../../../../utils/env";
interface IProps {
  listItem: IMessageModel | ISearchResultListItem;
  listItemContent?: Record<string, unknown>;
  type: searchMessageTypeValues;
  displayType: searchResultItemDisplayTypeValues;
  keywordList: string[];
}
const props = withDefaults(defineProps<IProps>(), {
  listItem: () => ({} as IMessageModel | ISearchResultListItem),
  listItemContent: () => ({} as Record<string, unknown>),
  type: "allMessage",
  displayType: "info",
  keywordList: () => [] as string[],
});

const emits = defineEmits(["showResultDetail", "navigateToChatPosition"]);
const TYPES = ref(TUIChatEngine.TYPES);

const avatarForShow = ref<string>("");
const nameForShow = ref<string>("");
const contentForShow = ref<IHighlightContent[]>([]);
const timeForShow = ref<string>("");
const relationStatusLabel = ref("");
const relationStatusTone = ref<"positive" | "neutral">("neutral");
const entityMetaVisible = ref(false);
const entityIdLabel = ref("");
const entityIdValue = ref("");
const entitySecondaryText = ref("");

const isHovering = ref<boolean>(false);

watchEffect(() => {
  const currentItem = props.listItem as any;
  avatarForShow.value = generateSearchResultAvatar(props.listItem);
  nameForShow.value = generateSearchResultShowName(
    props.listItem,
    props?.listItemContent
  );
  contentForShow.value = generateSearchResultShowContent(
    props.listItem,
    props.type,
    props.keywordList as string[],
    props?.displayType === "info"
  );
  timeForShow.value = (props.listItem as IMessageModel)?.time
    ? generateSearchResultTime((props.listItem as IMessageModel)?.time * 1000)
    : "";
  relationStatusLabel.value = currentItem?._relationStatusLabel || "";
  relationStatusTone.value = currentItem?._relationStatusTone || "neutral";

  entityMetaVisible.value = props.type === "contact" || props.type === "group";
  if (props.type === "contact") {
    const wananID = currentItem?._wananID || currentItem?.wananID || "";
    entityIdLabel.value = TUITranslateService.t("TUIContact.ID");
    entityIdValue.value = currentItem?.userID || "";
    entitySecondaryText.value = wananID
      ? `${TUITranslateService.t("TUIContact.万安号")}: ${wananID}`
      : "";
  } else if (props.type === "group") {
    entityIdLabel.value = TUITranslateService.t("TUIContact.群ID");
    entityIdValue.value = currentItem?.groupID || "";
    entitySecondaryText.value = currentItem?.type
      ? `${TUITranslateService.t("TUIContact.群类型")}: ${currentItem.type}`
      : "";
  } else {
    entityIdLabel.value = "";
    entityIdValue.value = "";
    entitySecondaryText.value = "";
  }
});

const onResultItemClicked = () => {
  if (props.type === "contact") {
    enterConversation(props.listItem as Record<string, string>);
    return;
  }

  if (props.type === "group") {
    TUIStore.update(StoreName.CUSTOM, "currentContactInfo", props.listItem);
    TUIStore.update(StoreName.CUSTOM, "currentContactSearchingStatus", false);
    TUIStore.update(StoreName.CUSTOM, "currentHomeNavBar", "relation");
    TUIStore.update(StoreName.SEARCH, "currentSearchingStatus", {
      isSearching: false,
      searchType: "global",
    });
  } else {
    if (
      props.displayType === "info" &&
      !(props.listItem as IMessageModel)?.ID
    ) {
      emits("showResultDetail", true, props.type, props.listItem);
    } else {
      navigateToChatPosition();
    }
  }
};

const setHoverStatus = (status: boolean) => {
  isHovering.value = status;
};

const navigateToChatPosition = () => {
  emits("navigateToChatPosition", props.listItem);
};

const getMessageAbstractType = (
  message: IMessageModel | ISearchResultListItem
) => {
  return message?.type
    ? TUITranslateService.t(`TUISearch.${messageTypeAbstractMap[message.type]}`)
    : TUITranslateService.t(`TUISearch.[合并消息]`);
};
</script>
<style lang="scss" scoped src="./style/index.scss"></style>
