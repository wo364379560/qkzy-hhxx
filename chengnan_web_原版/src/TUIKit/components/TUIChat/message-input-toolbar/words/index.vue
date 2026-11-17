<template>
  <ToolbarItemContainer
    ref="container"
    :iconFile="wordsIcon"
    title="常用语"
    :needBottomPopup="true"
    :iconWidth="isUniFrameWork ? '26px' : '20px'"
    :iconHeight="isUniFrameWork ? '26px' : '20px'"
    @onDialogShow="onDialogShow"
    @onDialogClose="onDialogClose"
  >
    <div :class="['words', !isPC && 'words-h5']">
      <div :class="['words-header', !isPC && 'words-h5-header']">
        <span :class="['words-header-title', !isPC && 'words-h5-header-title']">
          {{ TUITranslateService.t("Words.常用语-快捷回复工具") }}
        </span>
        <div :class="['words-header-actions', !isPC && 'words-h5-header-actions']">
          <button
            v-if="isEditMode"
            type="button"
            class="words-header-action"
            @click="addWord"
          >
            新增
          </button>
          <button
            v-if="isEditMode"
            type="button"
            class="words-header-action"
            @click="saveWords"
          >
            保存
          </button>
          <button
            v-if="isEditMode"
            type="button"
            class="words-header-action"
            @click="cancelEdit"
          >
            {{ TUITranslateService.t("TUIChat.取消") }}
          </button>
          <button
            v-else
            type="button"
            class="words-header-action"
            @click="startEdit"
          >
            编辑
          </button>
          <button
            v-if="!isPC"
            type="button"
            :class="['words-header-action', 'words-header-close', !isPC && 'words-h5-header-close']"
            @click="closeDialog"
          >
            {{ TUITranslateService.t("TUIChat.关闭") }}
          </button>
        </div>
      </div>
      <ul
        v-if="displayWords.length"
        :class="['words-list', !isPC && 'words-h5-list']"
      >
        <li
          v-for="(item, index) in displayWords"
          :key="index"
          :class="['words-list-item', !isPC && 'words-h5-list-item']"
          @click="selectWord(item)"
        >
          <template v-if="isEditMode">
            <input
              :value="item"
              type="text"
              class="words-input"
              placeholder="请输入常用语"
              @click.stop
              @input="updateWord(index, ($event.target as HTMLInputElement).value)"
            >
            <button
              type="button"
              class="words-delete"
              @click.stop="deleteWord(index)"
            >
              {{ TUITranslateService.t("TUIChat.删除") }}
            </button>
          </template>
          <template v-else>
            {{ resolveWordText(item) }}
          </template>
        </li>
      </ul>
      <div v-else class="words-empty">
        {{ isEditMode ? "暂无常用语，点击新增" : "暂无常用语" }}
      </div>
    </div>
  </ToolbarItemContainer>
</template>
<script setup lang="ts">
import {
  TUITranslateService,
  TUIChatEngine,
  TUIStore,
  StoreName,
  IConversationModel,
  SendMessageParams,
  TUIChatService,
} from '@tencentcloud/chat-uikit-engine-lite';
import { computed, ref, onMounted, onUnmounted } from '../../../../adapter-vue';
import ToolbarItemContainer from '../toolbar-item-container/index.vue';
import wordsIconLight from '../../../../assets/icon/words-light.svg';
import wordsIconDark from '../../../../assets/icon/words-dark.svg';
import { wordsList } from '../../utils/wordsList';
import { isEnabledMessageReadReceiptGlobal } from '../../utils/utils';
import { isPC, isUniFrameWork } from '../../../../utils/env';
import TUIChatConfig from '../../config';
import OfflinePushInfoManager from '../../offlinePushInfoManager';

const wordsIcon = TUIChatConfig.getTheme() === 'dark' ? wordsIconDark : wordsIconLight;
const emits = defineEmits(['onDialogPopupShowOrHide']);
const currentConversation = ref<IConversationModel>();
const container = ref();
const isEditMode = ref(false);
const wordsValueList = ref<string[]>([]);
const editingWordsValueList = ref<string[]>([]);
const COMMON_WORDS_STORAGE_KEY = 'tui-chat-common-words';
const defaultWordValueList = wordsList.map(item => item.value);
const defaultWordValueSet = new Set(defaultWordValueList);

const displayWords = computed(() => (isEditMode.value ? editingWordsValueList.value : wordsValueList.value));

const onCurrentConversationUpdated = (conversation: IConversationModel) => {
  currentConversation.value = conversation;
};

onMounted(() => {
  initializeWords();
  TUIStore.watch(StoreName.CONV, {
    currentConversation: onCurrentConversationUpdated,
  });
});

onUnmounted(() => {
  TUIStore.unwatch(StoreName.CONV, {
    currentConversation: onCurrentConversationUpdated,
  });
});

const selectWord = (word: string) => {
  if (isEditMode.value) {
    return;
  }

  const text = resolveWordText(word).trim();
  if (!text) {
    return;
  }
  const conversation = currentConversation.value;
  if (!conversation) {
    return;
  }

  const options = {
    to:
      conversation?.groupProfile?.groupID
      || conversation?.userProfile?.userID,
    conversationType: conversation?.type,
    payload: {
      text,
    },
    needReadReceipt: isEnabledMessageReadReceiptGlobal(),
  } as SendMessageParams;
  TUIChatService.sendTextMessage(options, {
    offlinePushInfo: OfflinePushInfoManager.create({
      conversation,
      payload: options.payload,
      messageType: TUIChatEngine.TYPES.MSG_TEXT,
    }),
  });
  // close dialog after submit evaluate
  container?.value?.toggleDialogDisplay(false);
};

const closeDialog = () => {
  cancelEdit();
  container?.value?.toggleDialogDisplay(false);
};

const onDialogShow = () => {
  emits('onDialogPopupShowOrHide', true);
};

const onDialogClose = () => {
  cancelEdit();
  emits('onDialogPopupShowOrHide', false);
};

const startEdit = () => {
  isEditMode.value = true;
  editingWordsValueList.value = [...wordsValueList.value];
};

const cancelEdit = () => {
  isEditMode.value = false;
  editingWordsValueList.value = [...wordsValueList.value];
};

const saveWords = () => {
  wordsValueList.value = sanitizeWords(editingWordsValueList.value);
  persistWords(wordsValueList.value);
  cancelEdit();
};

const addWord = () => {
  editingWordsValueList.value = editingWordsValueList.value.concat('');
};

const updateWord = (index: number, value: string) => {
  editingWordsValueList.value[index] = value;
};

const deleteWord = (index: number) => {
  editingWordsValueList.value = editingWordsValueList.value.filter((_, currentIndex) => currentIndex !== index);
};

function initializeWords() {
  wordsValueList.value = getStoredWords();
  editingWordsValueList.value = [...wordsValueList.value];
}

function getStoredWords() {
  if (typeof window === 'undefined') {
    return [...defaultWordValueList];
  }

  try {
    const rawValue = window.localStorage.getItem(COMMON_WORDS_STORAGE_KEY);
    if (rawValue === null) {
      return [...defaultWordValueList];
    }

    const parsedValue = JSON.parse(rawValue);
    if (Array.isArray(parsedValue)) {
      return sanitizeWords(parsedValue);
    }
  } catch (error) {
    console.warn('load common words failed:', error);
  }

  return [...defaultWordValueList];
}

function persistWords(words: string[]) {
  if (typeof window === 'undefined') {
    return;
  }

  window.localStorage.setItem(
    COMMON_WORDS_STORAGE_KEY,
    JSON.stringify(words),
  );
}

function sanitizeWords(words: unknown[]) {
  return words
    .map(word => String(word || '').trim())
    .filter(Boolean);
}

function resolveWordText(word: string) {
  if (defaultWordValueSet.has(word)) {
    return TUITranslateService.t(`Words.${word}`);
  }
  return word;
}
</script>
<style scoped lang="scss" src="./style/index.scss"></style>
