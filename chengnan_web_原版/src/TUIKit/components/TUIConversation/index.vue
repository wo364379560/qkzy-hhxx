<template>
  <div
    class="tui-conversation"
    @click="handleClickConv"
  >
    <ConversationHeader
      v-if="props.showHeader && isShowConversationHeader"
      ref="headerRef"
    />
    <ConversationNetwork />
    <ConversationList @handleSwitchConversation="handleSwitchConversation" />
  </div>
</template>
<script lang="ts" setup>
import { TUIStore, StoreName } from '@tencentcloud/chat-uikit-engine-lite';
import { onMounted, onUnmounted, ref, withDefaults } from '../../adapter-vue';
import ConversationList from './conversation-list/index.vue';
import ConversationHeader from './conversation-header/index.vue';
import ConversationNetwork from './conversation-network/index.vue';

const props = withDefaults(
  defineProps<{
    showHeader?: boolean;
  }>(),
  {
    showHeader: true,
  },
);
const emits = defineEmits(['handleSwitchConversation']);

const totalUnreadCount = ref(0);
const headerRef = ref<typeof ConversationHeader>();
const isShowConversationHeader = ref<boolean>(true);

const conversationWatchOptions = {
  totalUnreadCount: (count: number) => {
    totalUnreadCount.value = count;
  },
};

const customWatchOptions = {
  isShowConversationHeader: (showStatus: boolean) => {
    isShowConversationHeader.value = showStatus !== false;
  },
};

onMounted(() => {
  const unreadCount = TUIStore.getData(StoreName.CONV, 'totalUnreadCount');
  if (typeof unreadCount === 'number') {
    conversationWatchOptions.totalUnreadCount(unreadCount);
  }
  const showHeader = TUIStore.getData(
    StoreName.CUSTOM,
    'isShowConversationHeader',
  );
  if (typeof showHeader === 'boolean') {
    customWatchOptions.isShowConversationHeader(showHeader);
  }
  TUIStore.watch(StoreName.CONV, conversationWatchOptions);
  TUIStore.watch(StoreName.CUSTOM, customWatchOptions);
});

onUnmounted(() => {
  TUIStore.unwatch(StoreName.CONV, conversationWatchOptions);
  TUIStore.unwatch(StoreName.CUSTOM, customWatchOptions);
});

const handleSwitchConversation = (conversationID: string) => {
  emits('handleSwitchConversation', conversationID);
};

const handleClickConv = () => {
  headerRef?.value?.closeChildren();
};
</script>

<style lang="scss" scoped src="./style/index.scss"></style>
