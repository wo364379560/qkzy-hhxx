<template>
  <div
    :class="[
      'tui-search-input-container',
      !isPC && 'tui-search-input-container-h5',
      props.searchType === 'global'
        ? 'tui-search-input-container-global'
        : 'tui-search-input-container-conversation',
      props.inline && 'tui-search-input-container-inline',
    ]"
  >
    <div :class="['tui-search-input', !isPC && 'tui-search-input-h5']">
      <div class="tui-search-input-left">
        <Icon class="icon" :file="searchIcon" width="14px" height="14px" />
      </div>
      <input
        v-model="searchValueModel"
        class="tui-search-input-main"
        type="text"
        :placeholder="resolvedPlaceholder"
        :focus="false"
        enterkeyhint="search"
        @blur="onBlur"
        @input="handleInput"
        @keyup.enter="search"
        @confirm="search"
        @click.stop.prevent="onSearchInputClick"
      />
      <div
        v-if="searchingStatus"
        class="tui-search-input-right"
        @click="endSearching"
      >
        <Icon class="icon" :file="closeIcon" width="14px" height="14px" />
      </div>
    </div>
    <div
      v-if="!isPC && searchingStatus && props.searchType === 'global'"
      :class="[
        'tui-search-input-cancel',
        !isPC && 'tui-search-input-h5-cancel',
      ]"
      @click="endSearching"
    >
      {{ TUITranslateService.t("TUISearch.取消") }}
    </div>
  </div>
</template>
<script lang="ts" setup>
import { ref, onMounted, onUnmounted, computed } from "../../../adapter-vue";
import {
  TUIStore,
  StoreName,
  TUITranslateService,
} from "@tencentcloud/chat-uikit-engine-lite";
import { TUIGlobal } from "@tencentcloud/universal-api";
import Icon from "../../common/Icon.vue";
import searchIcon from "../../../assets/icon/search.svg";
import closeIcon from "../../../assets/icon/input-close.svg";
import { isPC } from "../../../utils/env";
import { ISearchInputValue, ISearchingStatus } from "../type";
const props = defineProps({
  placeholder: {
    type: String,
    default: () => TUITranslateService.t("TUISearch.搜索"),
  },
  searchType: {
    type: String,
    default: "global", // "global" / "conversation"
    validator(value: string) {
      return ["global", "conversation"].includes(value);
    },
  },
  inline: {
    type: Boolean,
    default: false,
  },
});

const searchValueModel = ref<string>("");
const currentSearchInputValue = ref<string>("");
const searchingStatus = ref<boolean>(false);
let searchTimer: ReturnType<typeof setTimeout> | null = null;
const resolvedPlaceholder = computed(() => {
  if (props.placeholder) {
    return props.placeholder;
  }
  return props.searchType === "conversation"
    ? TUITranslateService.t("TUISearch.聊天记录")
    : TUITranslateService.t("TUISearch.搜索");
});

function onCurrentSearchInputValueChange(obj: ISearchInputValue) {
  if (obj?.searchType === props?.searchType) {
    currentSearchInputValue.value = obj?.value;
    searchValueModel.value = obj?.value;
  }
}

function onCurrentSearchingStatusChange(obj: ISearchingStatus) {
  if (obj?.searchType === props?.searchType) {
    searchingStatus.value = obj?.isSearching;
  }
}

onMounted(() => {
  TUIStore.watch(StoreName.SEARCH, {
    currentSearchInputValue: onCurrentSearchInputValueChange,
    currentSearchingStatus: onCurrentSearchingStatusChange,
  });
});

onUnmounted(() => {
  TUIStore.unwatch(StoreName.SEARCH, {
    currentSearchInputValue: onCurrentSearchInputValueChange,
    currentSearchingStatus: onCurrentSearchingStatusChange,
  });
  if (searchTimer) {
    clearTimeout(searchTimer);
    searchTimer = null;
  }
});

const submitSearch = () => {
  if (searchValueModel.value === currentSearchInputValue.value) {
    return;
  }
  TUIStore.update(StoreName.SEARCH, "currentSearchInputValue", {
    value: searchValueModel.value,
    searchType: props.searchType,
  });
};

const handleInput = () => {
  if (searchTimer) {
    clearTimeout(searchTimer);
  }
  searchTimer = setTimeout(() => {
    submitSearch();
    searchTimer = null;
  }, 300);
};

const search = () => {
  if (searchTimer) {
    clearTimeout(searchTimer);
    searchTimer = null;
  }
  submitSearch();
};

const endSearching = () => {
  searchingStatus.value = false;
  TUIStore.update(StoreName.SEARCH, "currentSearchingStatus", {
    isSearching: false,
    searchType: props.searchType,
  });
  TUIStore.update(StoreName.SEARCH, "currentSearchInputValue", {
    value: "",
    searchType: props.searchType,
  });
};

const onSearchInputClick = () => {
  TUIStore.update(StoreName.SEARCH, "currentSearchingStatus", {
    isSearching: true,
    searchType: props.searchType,
  });
};

const onBlur = () => {
  TUIGlobal?.hideKeyboard?.();
};
</script>
<style lang="scss" scoped>
.tui-search-input-container {
  display: flex;
  flex-direction: row;
  box-sizing: border-box;

  &-global {
    flex: 1;
  }

  .tui-search-input {
    flex: 1;
    display: flex;
    flex-direction: row;
    width: calc(100% - 20px);
    margin: 10px;
    background: #ededed;
    justify-content: center;
    align-items: center;
    height: 28px;
    border-radius: 4px;

    &-main {
      flex: 1;
      background: transparent;
      border: none;
      caret-color: #007aff;
      font-size: 14px;

      &:focus {
        border: none;
        outline: none;
      }

      &::placeholder {
        color: #666;
        font-size: 12px;
      }
    }

    &-left,
    &-right {
      display: flex;
      width: 14px;
      height: 14px;
      padding: 0 7px;
    }
  }

  &-inline {
    width: 100%;
    min-width: 0;

    .tui-search-input {
      width: 100%;
      margin: 0;
      height: 36px;
      border-radius: 10px;

      &-left,
      &-right {
        width: 16px;
        height: 16px;
        padding: 0 10px;
        align-items: center;
        justify-content: center;
        flex: 0 0 auto;
      }

      &-main {
        min-width: 0;
        padding: 0;
      }
    }
  }
}

.tui-search-input-container-h5 {
  .tui-search-input-h5 {
    height: 34px;
  }

  .tui-search-input-cancel {
    display: flex;
    flex-direction: column;
    justify-content: center;
    align-items: center;
    color: #007aff;
    font-size: 16px;
    padding: 7px 10px 7px 3px;
    font-family: "PingFang SC", sans-serif;
  }
}
</style>
