<template>
  <div
    ref="searchMoreRef"
    :class="['tui-search-more', !isPC && 'tui-search-more-h5']"
  >
    <template v-if="isPC && props.searchType === 'global'">
      <button
        v-for="(extension, index) in extensionList"
        :key="index"
        type="button"
        class="tui-search-more-inline-item"
        :title="extension.text"
        @click="handleMenu(extension)"
      >
        <Icon
          v-if="extension.icon"
          class="tui-search-more-inline-item-icon"
          :file="extension.icon"
        />
        <span class="tui-search-more-inline-item-title">
          {{ getInlineTitle(extension) }}
        </span>
      </button>
    </template>
    <div
      v-else
      class="more"
      @click="toggleMore()"
    >
      <Icon
        class="more-icon"
        :file="searchMoreSVG"
        :width="isPC ? '28px' : '34px'"
        :height="isPC ? '28px' : '34px'"
      />
    </div>
    <ul
      v-if="isListShow"
      class="tui-search-more-list"
    >
      <li
        v-for="(extension, index) in extensionList"
        :key="index"
        class="list-item"
        @click="handleMenu(extension)"
      >
        <Icon
          v-if="extension.icon"
          class="list-item-icon"
          :file="extension.icon"
        />
        <div class="list-item-title">
          {{ extension.text }}
        </div>
      </li>
    </ul>
  </div>
</template>
<script lang="ts" setup>
import { ref, watch, onMounted } from '../../../adapter-vue';
import { TUIStore, StoreName } from '@tencentcloud/chat-uikit-engine-lite';
import TUICore, { ExtensionInfo, TUIConstants } from '@tencentcloud/tui-core-lite';
import { outsideClick } from '@tencentcloud/universal-api';
import Icon from '../../common/Icon.vue';
import searchMoreSVG from '../../../assets/icon/search-more.svg';
import { isPC, isUniFrameWork } from '../../../utils/env';
import TUIConversationServer from '../../TUIConversation/server';

const props = defineProps({
  searchType: {
    type: String,
    default: 'global', // "global" / "conversation"
    validator(value: string) {
      return ['global', 'conversation'].includes(value);
    },
  },
});

const searchMoreRef = ref<HTMLElement | null>();
const isListShow = ref<boolean>(false);
const toggleMore = () => {
  isListShow.value = !isListShow.value;
  if (!isUniFrameWork && isListShow.value) {
    outsideClick.listen({
      domRefs: searchMoreRef.value,
      handler: closeSearchMore,
    });
  }
};
const extensionList = ref<ExtensionInfo[]>([]);

const getInlineTitle = (item: ExtensionInfo) => {
  if (item?.text?.includes("单聊")) {
    return "单聊";
  }
  if (item?.text?.includes("群聊")) {
    return "群聊";
  }
  return item?.text || "";
};

const handleMenu = (item: ExtensionInfo) => {
  const { listener = { onClicked: () => { } } } = item;
  listener?.onClicked?.(item);
  closeSearchMore();
};

const closeSearchMore = () => {
  isListShow.value = false;
};

onMounted(() => {
  TUIConversationServer.getInstance();
  // extensions
  extensionList.value = [
    ...TUICore.getExtensionList(TUIConstants.TUISearch.EXTENSION.SEARCH_MORE.EXT_ID),
  ];
  // hide conversation header
  TUICore.callService({
    serviceName: TUIConstants.TUIConversation.SERVICE.NAME,
    method: TUIConstants.TUIConversation.SERVICE.METHOD.HIDE_CONVERSATION_HEADER,
    params: {},
  });
});

watch(
  () => isListShow.value,
  () => {
    if (isListShow.value) {
      TUIStore.update(StoreName.SEARCH, 'currentSearchingStatus', {
        isSearching: false,
        searchType: props.searchType,
      });
    }
  },
);
</script>

<style lang="scss" scoped>
.tui-search-more {
  display: flex;
  flex-direction: row;
  align-items: center;
  gap: 8px;
  position: relative;
  flex-shrink: 0;

  .more {
    width: 28px;
    height: 28px;
    margin-right: 6px;
  }

  &-inline-item {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    height: 34px;
    padding: 0 10px;
    border: 1px solid #dfe3ea;
    border-radius: 8px;
    background: #fff;
    color: #1f2329;
    cursor: pointer;
    white-space: nowrap;

    &:hover {
      border-color: #3370ff;
      color: #3370ff;
      background: #f7fbff;
    }

    &-icon {
      width: 16px;
      height: 16px;
      flex: 0 0 auto;
    }

    &-title {
      font-size: 12px;
      line-height: 16px;
      font-weight: 500;
    }
  }

  &-list {
    margin: 10px 0;
    position: absolute;
    list-style: none;
    cursor: pointer;
    right: 6px;
    top: 20px;
    z-index: 1000;
    background: #fff;
    display: flex;
    flex-direction: column;
    justify-content: center;
    align-items: flex-start;
    padding: 0;
    box-shadow: rgba(0,0,0,0.16) 0 3px 6px, rgba(0,0,0,0.23) 0 3px 6px;

    .list-item {
      display: flex;
      flex-direction: row;
      justify-content: space-between;
      align-items: center;
      height: 40px;
      padding: 0 10px;

      &-icon {
        margin-right: 2px;
      }

      &-title {
        font-size: 14px;
        text-wrap: nowrap;
        word-break: keep-all;
      }
    }
  }
}

.tui-search-more-h5{
  gap: 0;

  .more{
    width: 34px;
    height: 34px;
  }
}
</style>
