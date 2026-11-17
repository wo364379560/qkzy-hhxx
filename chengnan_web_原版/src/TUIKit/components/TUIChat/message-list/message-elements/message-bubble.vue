<template>
  <div
    :class="containerClassNameList"
    :date-message="
      AiRobotManager.isRobotMessage(messageItem)
        ? 'robotMessage'
        : messageItem.type
    "
  >
    <!-- multiple select radio -->
    <RadioSelect
      v-if="props.isMultipleSelectMode"
      class="multiple-select-radio"
      :isSelected="isMultipleSelected"
      @onChange="toggleMultipleSelect"
    />
    <div
      :class="{
        'control-reverse': message.flow === 'out',
      }"
    >
      <!-- message-bubble-container -->
      <div class="message-bubble-content">
        <div
          class="message-bubble-main-content"
          :class="[message.flow === 'in' ? '' : 'reverse']"
        >
          <div
            class="message-avatar-trigger"
            :class="[
              message.flow === 'in' && 'message-avatar-trigger-clickable',
            ]"
            @click.stop="handleAvatarClick"
            @pointerdown="startAvatarPress"
            @pointermove="moveAvatarPress"
            @pointerup="cancelAvatarPress"
            @pointercancel="cancelAvatarPress"
            @pointerleave="cancelAvatarPress"
            @contextmenu.prevent
          >
            <Avatar useSkeletonAnimation :url="message.avatar || ''" />
          </div>
          <main class="message-body" @click.stop>
            <div
              v-if="
                props.showSenderName &&
                message.flow === 'in' &&
                message.conversationType === 'GROUP' &&
                props.content.showName
              "
              class="message-body-nick-name"
            >
              <span>{{ props.content.showName }}</span>
              <span v-if="senderUserTag" class="message-body-user-tag">
                {{ senderUserTag }}
              </span>
            </div>
            <div
              :class="[
                'message-body-main',
                message.flow === 'out' && 'message-body-main-reverse',
              ]"
            >
              <div
                :class="[
                  'blink',
                  'message-body-content',
                  message.flow === 'out' ? 'content-out' : 'content-in',
                  message.hasRiskContent && 'content-has-risk',
                  isNoPadding ? 'content-no-padding' : '',
                  isNoPadding && isBlink ? 'blink-shadow' : '',
                  !isNoPadding && isBlink ? 'blink-content' : '',
                ]"
              >
                <div class="content-main">
                  <img
                    v-if="
                      (message.type === TYPES.MSG_IMAGE ||
                        message.type === TYPES.MSG_VIDEO) &&
                      message.hasRiskContent
                    "
                    :class="[
                      'message-risk-replace',
                      !isPC && 'message-risk-replace-h5',
                    ]"
                    :src="riskImageReplaceUrl"
                  />
                  <template v-else>
                    <slot name="messageElement" />
                    <slot name="TUIEmojiPlugin" />
                  </template>
                </div>
                <!-- Risk Content Tips -->
                <div
                  v-if="message.hasRiskContent"
                  class="content-has-risk-tips"
                >
                  {{ riskContentText }}
                </div>
              </div>
              <!-- audio unplay mark -->
              <div v-if="isDisplayUnplayMark" class="audio-unplay-mark" />
              <!-- Send Fail Icon -->
              <div
                v-if="message.status === 'fail' || message.hasRiskContent"
                class="message-label fail"
                @click="resendMessage()"
              >
                !
              </div>
              <!-- Loading Icon -->
              <Icon
                v-if="
                  message.status === 'unSend' &&
                  needLoadingIconMessageType.includes(message.type)
                "
                class="message-label loading-circle"
                :file="loadingIcon"
                :width="'15px'"
                :height="'15px'"
              />
              <!-- Read & Unread -->
              <ReadStatus
                class="message-label align-self-bottom"
                :message="message"
                @openReadUserPanel="openReadUserPanel"
              />
            </div>
          </main>
        </div>
        <!-- message extra area -->
        <div class="message-bubble-extra-content">
          <!-- extra: message translation -->
          <MessageTranslate
            :class="message.flow === 'out' ? 'reverse' : 'flex-row'"
            :message="message"
          />
          <!-- extra: message convert voice to text -->
          <MessageConvert
            :class="message.flow === 'out' ? 'reverse' : 'flex-row'"
            :message="message"
          />
          <!-- extra: message quote -->
          <MessageQuote
            :class="message.flow === 'out' ? 'reverse' : 'flex-row'"
            :message="message"
            @blinkMessage="blinkMessage"
            @scrollTo="scrollTo"
          />
        </div>
        <div
          v-if="messageTimeText"
          :class="[
            'message-send-time',
            message.flow === 'out'
              ? 'message-send-time-out'
              : 'message-send-time-in',
          ]"
        >
          {{ messageTimeText }}
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, toRefs, watchEffect, onUnmounted } from "../../../../adapter-vue";
import TUIChatEngine, {
  TUIStore,
  StoreName,
  TUITranslateService,
  IMessageModel,
} from "@tencentcloud/chat-uikit-engine-lite";
import Icon from "../../../common/Icon.vue";
import ReadStatus from "./read-status/index.vue";
import MessageQuote from "./message-quote/index.vue";
import Avatar from "../../../common/Avatar/index.vue";
import MessageTranslate from "./message-translate/index.vue";
import MessageConvert from "./message-convert/index.vue";
import RadioSelect from "../../../common/RadioSelect/index.vue";
import loadingIcon from "../../../../assets/icon/loading.png";
import { calculateTimestamp } from "../../utils/utils";
import { isPC } from "../../../../utils/env";
import AiRobotManager from "../../aiRobotManager";
import { ensureUserTags, getUserTag } from "../../../../utils/user-tag";

interface IProps {
  messageItem: IMessageModel;
  content?: any;
  classNameList?: string[];
  blinkMessageIDList?: string[];
  isMultipleSelectMode?: boolean;
  isAudioPlayed?: boolean | undefined;
  multipleSelectedMessageIDList?: string[];
  showSenderName?: boolean;
}

interface IEmits {
  (e: "resendMessage"): void;
  (e: "blinkMessage", messageID: string): void;
  (
    e: "setReadReceiptPanelVisible",
    visible: boolean,
    message?: IMessageModel
  ): void;
  (
    e: "changeSelectMessageIDList",
    options: { type: "add" | "remove" | "clearAll"; messageID: string }
  ): void;
  (e: "openContactInfo", contactInfo: Record<string, any>): void;
  // Only for uniapp
  (e: "scrollTo", scrollHeight: number): void;
}

const emits = defineEmits<IEmits>();

const props = withDefaults(defineProps<IProps>(), {
  messageItem: () => ({} as IMessageModel),
  content: () => ({}),
  isAudioPlayed: false,
  blinkMessageIDList: () => [],
  classNameList: () => [],
  isMultipleSelectMode: false,
  multipleSelectedMessageIDList: () => [],
  showSenderName: true,
});

const TYPES = TUIChatEngine.TYPES;
const riskImageReplaceUrl =
  "https://web.sdk.qcloud.com/component/TUIKit/assets/has_risk_default.png";
const needLoadingIconMessageType = [
  TYPES.MSG_LOCATION,
  TYPES.MSG_TEXT,
  TYPES.MSG_CUSTOM,
  TYPES.MSG_MERGER,
  TYPES.MSG_FACE,
];

const { blinkMessageIDList, messageItem: message } = toRefs(props);

const isMultipleSelected = computed<boolean>(() => {
  return props.multipleSelectedMessageIDList.includes(message.value.ID);
});

const isDisplayUnplayMark = computed<boolean>(() => {
  return (
    message.value.flow === "in" &&
    message.value.status === "success" &&
    message.value.type === TYPES.MSG_AUDIO &&
    !props.isAudioPlayed
  );
});

const containerClassNameList = computed(() => {
  return [
    "message-bubble",
    isMultipleSelected.value ? "multiple-selected" : "",
    ...props.classNameList,
  ];
});

// When an emoji is deleted, the `reactionList` will update the corresponding emoji's `totalUserCount`.
const hasEmojiReaction = computed(() => {
  return message.value?.reactionList.some(
    (item: any) => item.totalUserCount !== 0
  );
});

const isNoPadding = computed(() => {
  return (
    !hasEmojiReaction.value &&
    [TYPES.MSG_IMAGE, TYPES.MSG_VIDEO, TYPES.MSG_MERGER].includes(
      message.value.type
    )
  );
});

const messageTimeText = computed(() => {
  if (
    !message.value?.time ||
    message.value?.status !== "success" ||
    message.value?.isRevoked
  ) {
    return "";
  }

  return calculateTimestamp(message.value.time * 1000);
});

watchEffect(() => {
  const from = String(message.value?.from || "").trim();
  if (
    from &&
    props.showSenderName &&
    message.value?.flow === "in" &&
    message.value?.conversationType === TYPES.CONV_GROUP
  ) {
    void ensureUserTags([from]);
  }
});

const senderUserTag = computed(() =>
  getUserTag(String(message.value?.from || ""))
);

const riskContentText = computed<string>(() => {
  let content = TUITranslateService.t("TUIChat.涉及敏感内容") + ", ";
  if (message.value.flow === "out") {
    content += TUITranslateService.t("TUIChat.发送失败");
  } else {
    content += TUITranslateService.t(
      message.value.type === TYPES.MSG_AUDIO
        ? "TUIChat.无法收听"
        : "TUIChat.无法查看"
    );
  }
  return content;
});

const isBlink = computed(() => {
  if (message.value?.ID) {
    return blinkMessageIDList?.value?.includes(message.value.ID);
  }
  return false;
});

function toggleMultipleSelect(isSelected: boolean) {
  emits("changeSelectMessageIDList", {
    type: isSelected ? "add" : "remove",
    messageID: message.value.ID,
  });
}

function resendMessage() {
  if (!message.value?.hasRiskContent) {
    emits("resendMessage");
  }
}

function blinkMessage(messageID?: string) {
  if (!messageID) {
    return;
  }
  emits("blinkMessage", messageID);
}

function scrollTo(scrollHeight: number) {
  emits("scrollTo", scrollHeight);
}

function openReadUserPanel() {
  emits("setReadReceiptPanelVisible", true, message.value);
}

let avatarPressTimer: ReturnType<typeof setTimeout> | undefined;
let avatarPressed = false;
let avatarPressPoint = { x: 0, y: 0 };
function cancelAvatarPress() { clearTimeout(avatarPressTimer); }
function moveAvatarPress(event: PointerEvent) {
  if (Math.hypot(event.clientX - avatarPressPoint.x, event.clientY - avatarPressPoint.y) > 10) cancelAvatarPress();
}
function startAvatarPress(event: PointerEvent) {
  cancelAvatarPress();
  avatarPressed = false;
  if (event.button !== 0 || message.value.conversationType !== 'GROUP' || message.value.flow !== 'in') return;
  avatarPressPoint = { x: event.clientX, y: event.clientY };
  avatarPressTimer = setTimeout(() => {
    avatarPressed = true;
    TUIStore.update(StoreName.CUSTOM, 'avatarMention', {
      conversationID: message.value.conversationID,
      id: message.value.from,
      label: (message.value as any).nameCard || message.value.nick || message.value.from,
      nonce: Date.now(),
    });
  }, 500);
}
onUnmounted(cancelAvatarPress);

function handleAvatarClick() {
  if (avatarPressed) { avatarPressed = false; return; }
  if (message.value.flow !== "in" || !message.value.from) {
    return;
  }

  const nick = message.value.nick || message.value.from;
  const avatar = message.value.avatar || "";

  emits("openContactInfo", {
    userID: message.value.from,
    nick,
    avatar,
    profile: {
      nick,
      avatar,
      selfSignature: "",
    },
  });
}
</script>

<style lang="scss" scoped>
* {
  display: flex;
  flex-direction: column;
  min-width: 0;
  box-sizing: border-box;
}

.message-body-nick-name {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  flex-wrap: wrap;
}

  .message-body-user-tag {
  display: inline-flex;
  align-items: center;
  padding: 2px 8px;
  border-radius: 999px;
  background: rgba(51, 112, 255, 0.12);
  color: #3370ff;
  font-size: 12px;
  line-height: 18px;
  font-weight: 500;
}

.message-bubble {
  padding: 10px 15px;
  display: flex;
  flex-direction: row;
  user-select: none;
  -webkit-touch-callout: none;
  -webkit-user-select: none;
  -khtml-user-select: none;
  -moz-user-select: none;
  -ms-user-select: none;

  &.multiple-selected {
    background-color: #f0f0f0;
  }

  .multiple-select-radio {
    margin-right: 12px;
    flex: 0 0 auto;
  }

  .message-avatar-trigger {
    flex: 0 0 auto;
  }

  .message-avatar-trigger-clickable {
    cursor: pointer;
  }

  .control-reverse {
    flex: 1 1 auto;
    flex-direction: row-reverse;
  }

  .message-send-time {
    margin: 4px 44px 0;
    font-size: 12px;
    line-height: 16px;
    color: #9ca3af;

    &-in {
      align-self: flex-start;
    }

    &-out {
      align-self: flex-end;
      text-align: right;
    }
  }

  .message-bubble-main-content {
    display: flex;
    flex-direction: row;

    .message-avatar {
      display: block;
      width: 36px;
      height: 36px;
      border-radius: 5px;
      flex: 0 0 auto;
    }

    .message-body {
      display: flex;
      flex: 0 1 auto;
      flex-direction: column;
      align-items: flex-start;
      margin: 0 8px;

      .message-body-nick-name {
        display: block;
        margin-bottom: 4px;
        font-size: 12px;
        color: #999;
        max-width: 150px;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
      }

      .message-body-main {
        max-width: 100%;
        display: flex;
        flex-direction: row;
        min-width: 0;
        box-sizing: border-box;

        &-reverse {
          flex-direction: row-reverse;
        }

        .audio-unplay-mark {
          flex: 0 0 auto;
          width: 5px;
          height: 5px;
          border-radius: 50%;
          background-color: #f00;
          margin: 5px;
        }

        .message-body-content {
          display: flex;
          flex-direction: column;
          min-width: 0;
          box-sizing: border-box;
          padding: 12px;
          font-size: 14px;
          color: #000;
          letter-spacing: 0;
          word-wrap: break-word;
          word-break: break-all;
          position: relative;

          .content-main {
            box-sizing: border-box;
            display: flex;
            flex-direction: column;
            flex-shrink: 0;
            align-content: flex-start;
            border: 0 solid black;
            margin: 0;
            padding: 0;
            min-width: 0;

            .message-risk-replace {
              width: 130px;
              height: 130px;
            }
          }

          .content-has-risk-tips {
            font-size: 12px;
            color: #fa5151;
            font-family: PingFangSC-Regular;
            margin-top: 5px;
            border-top: 1px solid #e5c7c7;
            padding-top: 5px;
          }
        }

        .content-in {
          background: #fbfbfb;
          border-radius: 0 10px 10px;
        }

        .content-out {
          background: #dceafd;
          border-radius: 10px 0 10px 10px;
        }

        .content-no-padding {
          padding: 0;
          background: transparent;
          border-radius: 10px;
          overflow: hidden;
        }

        .content-no-padding.content-has-risk {
          padding: 12px;
        }

        .content-has-risk {
          background: rgba(250, 81, 81, 0.16);
        }

        .blink-shadow {
          @keyframes shadow-blink {
            50% {
              box-shadow: rgba(255, 156, 25, 1) 0 0 10px 0;
            }
          }

          box-shadow: rgba(255, 156, 25, 0) 0 0 10px 0;
          animation: shadow-blink 1s linear 3;
        }

        .blink-content {
          @keyframes reference-blink {
            50% {
              background-color: #ff9c19;
            }
          }

          animation: reference-blink 1s linear 3;
        }

        .message-label {
          align-self: flex-end;
          font-family: PingFangSC-Regular;
          font-size: 12px;
          color: #b6b8ba;
          word-break: keep-all;
          flex: 0 0 auto;
          margin: 0 8px;

          &.fail {
            width: 15px;
            height: 15px;
            border-radius: 15px;
            background: red;
            color: #fff;
            display: flex;
            justify-content: center;
            align-items: center;
            cursor: pointer;
          }

          &.loading-circle {
            opacity: 0;
            animation: circle-loading 2s linear 1s infinite;
          }

          @keyframes circle-loading {
            0% {
              transform: rotate(0);
              opacity: 1;
            }

            100% {
              opacity: 1;
              transform: rotate(360deg);
            }
          }
        }

        .align-self-bottom {
          align-self: flex-end;
        }
      }
    }
  }

  .reverse {
    display: flex;
    flex-direction: row-reverse;
    justify-content: flex-start;
  }

  .message-bubble-extra-content {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
  }
}

[date-message="robotMessage"] {
  .content-in {
    background: none !important;
    padding: 0 !important;
    border-radius: 0 10px 10px;
  }
}
</style>
