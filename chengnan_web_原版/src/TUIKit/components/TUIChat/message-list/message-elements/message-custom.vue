<template>
  <div class="custom">
    <ChatAttachmentCard v-if="attachmentInfo" :attachment="attachmentInfo" />
    <SharedCardView v-else-if="sharedCard" :card="sharedCard" @openContactInfo="emit('openContactInfo', $event)" />
    <template
      v-else-if="customData.businessID === CHAT_MSG_CUSTOM_TYPE.SERVICE"
    >
      <div>
        <h1>
          <label>{{ extension.title }}</label>
          <a
            v-if="serviceHeaderUrl"
            :href="serviceHeaderUrl"
            target="_blank"
            rel="noopener noreferrer"
            >{{ extension.hyperlinks_text.key }}</a
          >
        </h1>
        <ul v-if="extension.item && extension.item.length > 0">
          <li v-for="(item, index) in extension.item" :key="index">
            <a
              v-if="isUrl(item.value)"
              :href="item.value"
              target="_blank"
              rel="noopener noreferrer"
              >{{ item.key }}</a
            >
            <p v-else>
              {{ item.key }}
            </p>
          </li>
        </ul>
        <article>{{ extension.description }}</article>
      </div>
    </template>
    <template
      v-else-if="customData.businessID === CHAT_MSG_CUSTOM_TYPE.EVALUATE"
    >
      <div class="evaluate">
        <h1>{{ TUITranslateService.t("message.custom.对本次服务评价") }}</h1>
        <ul class="evaluate-list">
          <li
            v-for="(item, index) in Math.max(Number(customData.score) || 0, 0)"
            :key="index"
            class="evaluate-list-item"
          >
            <Icon :file="star" class="file-icon" />
          </li>
        </ul>
        <article>{{ customData.comment }}</article>
      </div>
    </template>
    <template v-else-if="customData.businessID === CHAT_MSG_CUSTOM_TYPE.ORDER">
      <div class="order" @click="openLink(customData.link)">
        <img :src="customData.imageUrl" />
        <main>
          <h1>{{ customData.title }}</h1>
          <p>{{ customData.description }}</p>
          <span>{{ customData.price }}</span>
        </main>
      </div>
    </template>
    <template v-else-if="customData.businessID === CHAT_MSG_CUSTOM_TYPE.LINK">
      <div class="textLink">
        <p>{{ customData.text }}</p>
        <a
          v-if="linkMessageUrl"
          :href="linkMessageUrl"
          target="_blank"
          rel="noopener noreferrer"
          >{{ TUITranslateService.t("message.custom.查看详情>>") }}</a
        >
      </div>
    </template>
    <template v-else>
      <span class="unsupported-message">{{ fallbackCustomText }}</span>
    </template>
  </div>
</template>

<script lang="ts" setup>
import { watchEffect, ref, computed } from "../../../../adapter-vue";
import {
  TUITranslateService,
  IMessageModel,
} from "@tencentcloud/chat-uikit-engine-lite";
import { isUrl, JSONToObject } from "../../../../utils/index";
import { CHAT_MSG_CUSTOM_TYPE } from "../../../../constant";
import { ICustomMessagePayload } from "../../../../interface";
import Icon from "../../../common/Icon.vue";
import star from "../../../../assets/icon/star-light.png";
import { normalizeExternalUrl } from "../../../../utils/security";
import ChatAttachmentCard from "./chat-attachment-card.vue";
import SharedCardView from './shared-card.vue';
import { parseSharedCard } from '../../utils/sharedCard';
import type { ChatAttachmentInfo } from "../../utils/chatAttachmentMessage";
import { parseChatAttachmentCustomPayload } from "../../utils/chatAttachmentMessage";
import {
  getLongdoCallRecordText,
  isRawLongdoCallDisplayText,
  parseCallRecordDisplay,
} from "@/features/call/doMattxCallProtocol";
interface Props {
  messageItem: IMessageModel;
  content: any;
}

const props = withDefaults(defineProps<Props>(), {
  messageItem: undefined,
  content: undefined,
});
const emit = defineEmits<{ (event: 'openContactInfo', profile: Record<string, any>): void }>();
const sharedCard = computed(() => parseSharedCard(props.messageItem?.payload?.data));

const custom = ref();
const message = ref<IMessageModel>();
const extension = ref();
const customData = ref<ICustomMessagePayload>({
  businessID: "",
});
const attachmentInfo = ref<ChatAttachmentInfo | null>(null);
const serviceHeaderUrl = computed(() =>
  normalizeExternalUrl(extension.value?.hyperlinks_text?.value)
);
const linkMessageUrl = computed(() =>
  normalizeExternalUrl(customData.value?.link)
);
const fallbackCustomText = computed(() => {
  const record =
    getLongdoCallRecordText(customData.value) ||
    parseCallRecordDisplay(customData.value)?.summary;
  if (record) {
    return record;
  }
  const description = String(props.messageItem?.payload?.description || "").trim();
  if (
    description &&
    !isRawLongdoCallDisplayText(description) &&
    description !== "发起通话" &&
    description !== "[自定义消息]" &&
    description !== "当前版本不支持该消息类型"
  ) {
    return description;
  }
  return "当前版本不支持该消息类型";
});

watchEffect(() => {
  custom.value = props.content;
  message.value = props.messageItem;
  const payload = props.messageItem?.payload || {};
  customData.value = JSONToObject(payload.data) || { businessID: "" };
  attachmentInfo.value =
    parseChatAttachmentCustomPayload(customData.value) ||
    parseChatAttachmentCustomPayload(payload.data) ||
    parseChatAttachmentCustomPayload(props.content?.custom) ||
    parseChatAttachmentCustomPayload(props.content);
  if (customData.value?.businessID === CHAT_MSG_CUSTOM_TYPE.SERVICE) {
    extension.value = JSONToObject(payload.extension);
  } else {
    extension.value = undefined;
  }
});
const openLink = (url: any) => {
  const safeUrl = normalizeExternalUrl(url);
  if (!safeUrl) {
    return;
  }
  window.open(safeUrl, "_blank", "noopener,noreferrer");
};
</script>
<style lang="scss" scoped>
@import "../../../../assets/styles/common";

a {
  color: #679ce1;
}

.custom {
  font-size: 14px;

  .unsupported-message {
    display: inline-block;
    color: #6b7280;
    line-height: 1.5;
  }

  h1 {
    font-size: 14px;
    color: #000;
  }

  h1,
  a,
  p {
    font-size: 14px;
  }

  .evaluate {
    ul {
      display: flex;
      padding: 10px 0;
    }

    &-list {
      display: flex;
      flex-direction: row;

      &-item {
        padding: 0 2px;
      }
    }
  }

  .order {
    display: flex;

    main {
      padding-left: 5px;

      p {
        font-family: PingFangSC-Regular;
        width: 145px;
        line-height: 17px;
        font-size: 14px;
        color: #999;
        letter-spacing: 0;
        margin-bottom: 6px;
        word-break: break-word;
      }

      span {
        font-family: PingFangSC-Regular;
        line-height: 25px;
        color: #ff7201;
      }
    }

    img {
      width: 67px;
      height: 67px;
    }
  }
}
</style>
