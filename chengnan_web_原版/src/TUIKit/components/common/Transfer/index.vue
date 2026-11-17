<template>
  <div
    class="transfer"
    :class="[!isPC ? 'transfer-h5' : '', isWeChat ? 'transfer-h5-wechat' : '']"
  >
    <header v-if="!isPC" class="transfer-header transfer-h5-header">
      <div v-if="!props.isHiddenBackIcon" @click="cancel">
        <Icon class="icon" :file="backIcon" :width="'18px'" :height="'18px'" />
      </div>
      <span class="title">{{ transferTitle }}</span>
      <span class="space" />
    </header>
    <main class="main">
      <div class="left">
        <header class="transfer-header">
          <slot name="header-extra" />
          <!-- PC triggers @keyup.enter -->
          <input
            v-if="isPC && isTransferSearch"
            type="text"
            :value="searchValue"
            :placeholder="resolvedSearchPlaceholder"
            enterkeyhint="search"
            :class="[isUniFrameWork ? 'left-uniapp-input' : '']"
            @input="handleInput"
            @keyup.enter="handleInput"
          />
          <!-- not PC triggers blur -->
          <input
            v-if="!isPC && isTransferSearch"
            type="text"
            :placeholder="resolvedSearchPlaceholder"
            enterkeyhint="search"
            :value="searchValue"
            :class="[isUniFrameWork ? 'left-uniapp-input' : '']"
            @input="handleInput"
            @blur="handleInput"
            @confirm="handleInput"
          />
        </header>
        <main class="transfer-left-main">
          <ul class="transfer-list">
            <li
              v-if="optional.length > 1 && !isRadio"
              class="transfer-list-item"
              @click="selectedAll"
            >
              <Icon
                v-if="isAllOptionalSelected"
                :file="selectedIcon"
                :width="'18px'"
                :height="'18px'"
              />
              <i v-else class="icon-unselected" />
              <span class="select-all">{{
                TUITranslateService.t("component.全选")
              }}</span>
            </li>
            <li
              v-for="item in transferList"
              :key="item.userID"
              class="transfer-list-item"
              @click="selected(item)"
            >
              <Icon
                v-if="isSelected(item)"
                :file="selectedIcon"
                :class="[item.isDisabled && 'disabled']"
                :width="'18px'"
                :height="'18px'"
              />
              <i
                v-else
                :class="[item.isDisabled && 'disabled', 'icon-unselected']"
              />
              <template v-if="!isTransferCustomItem">
                <img
                  class="avatar"
                  :src="
                    item.avatar ||
                    'https://web.sdk.qcloud.com/component/TUIKit/assets/avatar_21.png'
                  "
                  onerror="this.onerror=null;this.src='https://web.sdk.qcloud.com/component/TUIKit/assets/avatar_21.png'"
                />
                <span class="name">{{
                  item.remark || item.friendRemark || item.nick || item.userID
                }}</span>
                <span v-if="item.isDisabled"
                  >（{{ TUITranslateService.t("component.已在群中") }}）</span
                >
              </template>
              <template v-else>
                <slot name="left" :data="item" />
              </template>
            </li>
            <li
              v-if="transferTotal > transferList.length"
              class="transfer-list-item more"
              @click="getMore"
            >
              {{ TUITranslateService.t("component.查看更多") }}
            </li>
          </ul>
          <div v-if="transferList.length === 0" class="transfer-empty">
            {{ resolvedEmptyText }}
          </div>
        </main>
      </div>
      <div class="right">
        <header v-if="isPC" class="transfer-header">
          {{ transferTitle }}
        </header>
        <ul v-if="resultShow" class="transfer-list">
          <p
            v-if="transferSelectedList.length > 0 && isPC"
            class="transfer-text"
          >
            {{ TUITranslateService.t("component.已选中")
            }}{{ transferSelectedList.length
            }}{{ TUITranslateService.t("component.人") }}
          </p>
          <li
            v-for="(item, index) in transferSelectedList"
            :key="index"
            class="transfer-list-item space-between"
          >
            <aside class="transfer-list-item-content">
              <template v-if="!isTransferCustomItem">
                <img
                  class="avatar"
                  :src="
                    item.avatar ||
                    'https://web.sdk.qcloud.com/component/TUIKit/assets/avatar_21.png'
                  "
                  onerror="this.onerror=null;this.src='https://web.sdk.qcloud.com/component/TUIKit/assets/avatar_21.png'"
                />
                <span v-if="isPC" class="name">{{
                  item.remark || item.friendRemark || item.nick || item.userID
                }}</span>
              </template>
              <template v-else>
                <slot name="right" :data="item" />
              </template>
            </aside>
            <span v-if="isPC" @click="selected(item)">
              <Icon :file="cancelIcon" :width="'18px'" :height="'18px'" />
            </span>
          </li>
        </ul>
        <footer class="transfer-right-footer">
          <button
            class="btn btn-cancel"
            :disabled="props.submitLoading"
            @click="cancel"
          >
            {{ TUITranslateService.t("component.取消") }}
          </button>
          <button
            v-if="transferSelectedList.length > 0"
            class="btn"
            :disabled="props.submitLoading || props.submitDisabled"
            @click="submit"
          >
            {{
              props.submitLoading
                ? "处理中..."
                : TUITranslateService.t("component.完成")
            }}
          </button>
          <button
            v-else
            class="btn btn-no"
            :disabled="props.submitLoading || props.submitDisabled"
            @click="submit"
          >
            {{
              props.submitLoading
                ? "处理中..."
                : TUITranslateService.t("component.完成")
            }}
          </button>
        </footer>
      </div>
    </main>
  </div>
</template>

<script lang="ts" setup>
import { ref, watchEffect, computed } from "../../../adapter-vue";
import { TUITranslateService } from "@tencentcloud/chat-uikit-engine-lite";
import { ITransferListItem } from "../../../interface";
import Icon from "../Icon.vue";
import { Toast, TOAST_TYPE } from "../Toast/index";
import selectedIcon from "../../../assets/icon/selected.svg";
import backIcon from "../../../assets/icon/back.svg";
import cancelIcon from "../../../assets/icon/cancel.svg";
import { isPC, isUniFrameWork, isWeChat } from "../../../utils/env";

const props = defineProps({
  list: {
    type: Array,
    default: () => [],
  },
  selectedList: {
    type: Array,
    default: () => [],
  },
  isSearch: {
    type: Boolean,
    default: true,
  },
  isRadio: {
    type: Boolean,
    default: false,
  },
  isCustomItem: {
    type: Boolean,
    default: false,
  },
  title: {
    type: String,
    default: "",
  },
  type: {
    type: String,
    default: "",
  },
  resultShow: {
    type: Boolean,
    default: true,
  },
  total: {
    type: Number,
    default: 0,
  },
  isHiddenBackIcon: {
    type: Boolean,
    default: false,
  },
  searchPlaceholder: {
    type: String,
    default: "",
  },
  emptyText: {
    type: String,
    default: "",
  },
  maxSelectable: {
    type: Number,
    default: -1,
  },
  maxSelectionMessage: {
    type: String,
    default: "",
  },
  submitLoading: {
    type: Boolean,
    default: false,
  },
  submitDisabled: {
    type: Boolean,
    default: false,
  },
});

const transferList = ref<ITransferListItem[]>([]);
const transferTotal = ref<number>(0);
const transferSelectedList = ref<ITransferListItem[]>([]);
const isTransferSearch = ref(true);
const isTransferCustomItem = ref(false);
const transferTitle = ref("");
const searchValue = ref("");

watchEffect(() => {
  if (props.isCustomItem) {
    transferList.value = (props.list as ITransferListItem[]).filter(
      (item: any) => {
        const conversationID = item?.conversationID;
        if (typeof conversationID !== "string") {
          return true;
        }
        return conversationID.indexOf("@TIM#SYSTEM") < 0;
      }
    );
  } else {
    transferList.value = props.list as ITransferListItem[];
  }
  transferTotal.value = props.total ? props.total : props.list.length;
  transferSelectedList.value = (
    props.selectedList && props.selectedList.length > 0
      ? props.selectedList
      : transferSelectedList.value
  ) as any;
  isTransferSearch.value = props.isSearch;
  isTransferCustomItem.value = props.isCustomItem;
  transferTitle.value = props.title;
});

const emit = defineEmits(["search", "submit", "cancel", "getMore"]);

const optional = computed(() =>
  transferList.value.filter((item: any) => !item.isDisabled)
);

const getItemKey = (item: any) =>
  item?.userID || item?.conversationID || item?.groupID || item?.ID || "";

const isSameItem = (sourceItem: any, targetItem: any) => {
  const sourceKey = getItemKey(sourceItem);
  const targetKey = getItemKey(targetItem);
  if (sourceKey && targetKey) {
    return sourceKey === targetKey;
  }
  return sourceItem === targetItem;
};

const findSelectedIndex = (item: any) =>
  transferSelectedList.value.findIndex((selectedItem: any) =>
    isSameItem(selectedItem, item)
  );

const isSelected = (item: any) => findSelectedIndex(item) > -1;

const isAllOptionalSelected = computed(
  () =>
    optional.value.length > 0 &&
    (optional.value.every((item: any) => isSelected(item)) ||
      (props.maxSelectable > 0 && transferSelectedList.value.length >= props.maxSelectable))
);

const resolvedSearchPlaceholder = computed(
  () =>
    props.searchPlaceholder || TUITranslateService.t("component.请输入userID")
);

const resolvedEmptyText = computed(
  () => props.emptyText || TUITranslateService.t("TUISearch.暂无搜索结果")
);
const maxSelectableCount = computed(() =>
  props.maxSelectable >= 0 ? props.maxSelectable : Number.POSITIVE_INFINITY
);

const notifyMaxSelectionReached = () => {
  Toast({
    message: props.maxSelectionMessage || "已达到选择上限",
    type: TOAST_TYPE.WARNING,
  });
};

const handleInput = (e: any) => {
  searchValue.value = e?.target?.value || e?.detail?.value || "";
  emit("search", searchValue.value);
};
const selected = (item: any) => {
  if (props.submitLoading || props.submitDisabled) {
    return;
  }
  if (item.isDisabled) {
    return;
  }
  let list: ITransferListItem[] = transferSelectedList.value;
  const index: number = findSelectedIndex(item);
  if (index > -1) {
    return transferSelectedList.value.splice(index, 1);
  }
  if (props.isRadio) {
    list = [];
  }
  if (list.length >= maxSelectableCount.value) {
    notifyMaxSelectionReached();
    return;
  }
  list.push(item);
  transferSelectedList.value = list;
};

const selectedAll = () => {
  if (props.submitLoading || props.submitDisabled) {
    return;
  }
  if (isAllOptionalSelected.value) {
    transferSelectedList.value = transferSelectedList.value.filter(
      (selectedItem: any) =>
        !optional.value.some((item: any) => isSameItem(selectedItem, item))
    );
    return;
  }

  const nextItems = optional.value.filter((item: any) => !isSelected(item));
  const available = Math.max(0, maxSelectableCount.value - transferSelectedList.value.length);
  transferSelectedList.value = [...transferSelectedList.value, ...nextItems.slice(0, available)];
  if (nextItems.length > available) notifyMaxSelectionReached();
};

const submit = () => {
  if (props.submitLoading || props.submitDisabled) {
    return;
  }
  emit("submit", transferSelectedList.value);
  searchValue.value = "";
};

const cancel = () => {
  if (props.submitLoading) {
    return;
  }
  emit("cancel");
  searchValue.value = "";
};

const getMore = () => {
  emit("getMore");
};
</script>

<style lang="scss" scoped src="./style/transfer.scss"></style>
