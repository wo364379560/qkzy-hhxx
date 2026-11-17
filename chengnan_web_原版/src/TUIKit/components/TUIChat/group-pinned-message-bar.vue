<template>
  <div
    v-if="current?.message_id"
    class="tui-chat-group-pin"
  >
    <div
      class="tui-chat-group-pin-main"
      @click="$emit('cycle')"
    >
      <div class="tui-chat-group-pin-accent"></div>
      <div class="tui-chat-group-pin-content">
        <div class="tui-chat-group-pin-text">
          <span class="tui-chat-group-pin-label">{{ pinLabel }}</span>
          <strong>{{ current.message_sender_name || "群成员" }}：</strong>
          <span>{{ current.message_abstract || "已置顶一条消息" }}</span>
        </div>
      </div>
    </div>
    <button
      class="tui-chat-group-pin-more"
      type="button"
      title="全部置顶"
      @click.stop="panelOpen = !panelOpen"
    >
      <span :class="{ 'is-open': panelOpen }">▾</span>
    </button>
    <div
      v-if="panelOpen"
      class="tui-chat-group-pin-mask"
      @click="panelOpen = false"
    ></div>
    <div
      v-if="panelOpen"
      class="tui-chat-group-pin-panel"
    >
      <div class="tui-chat-group-pin-panel-title">置顶消息</div>
      <button
        v-for="item in messages"
        :key="item.message_id"
        class="tui-chat-group-pin-panel-item"
        :class="{ 'is-active': item.message_id === current.message_id }"
        type="button"
        @click="handleJump(item)"
      >
        <div class="tui-chat-group-pin-panel-item-main">
          <strong>{{ item.message_sender_name || "群成员" }}</strong>
          <span>{{ item.message_abstract || "已置顶一条消息" }}</span>
        </div>
        <span
          class="tui-chat-group-pin-panel-unpin"
          title="取消置顶"
          @click.stop="$emit('unpin', item)"
        >×</span>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from '../../adapter-vue';
import type { GroupMessagePin } from '@/api';

const props = defineProps<{
  messages: GroupMessagePin[];
  currentIndex: number;
}>();

const emit = defineEmits<{
  (e: 'cycle'): void;
  (e: 'jump', message: GroupMessagePin): void;
  (e: 'unpin', message: GroupMessagePin): void;
}>();

const panelOpen = ref(false);

const current = computed(() => {
  const list = props.messages || [];
  if (!list.length) return null;
  const index = Math.min(Math.max(props.currentIndex || 0, 0), list.length - 1);
  return list[index];
});

const pinLabel = computed(() => {
  const total = props.messages?.length || 0;
  if (total > 1) {
    const index = Math.min(Math.max(props.currentIndex || 0, 0), total - 1);
    return `置顶消息 ${index + 1}/${total}`;
  }
  return '置顶消息';
});

watch(
  () => props.messages.map(item => item.message_id).join(','),
  () => {
    panelOpen.value = false;
  },
);

function handleJump(item: GroupMessagePin) {
  panelOpen.value = false;
  emit('jump', item);
}
</script>

<style scoped lang="scss">
.tui-chat-group-pin {
  position: relative;
  display: flex;
  align-items: center;
  flex-direction: row;
  padding: 8px 8px 8px 16px;
  border-bottom: 1px solid #eef2f7;
  background: #ffffff;

  &-main {
    min-width: 0;
    flex: 1;
    display: flex;
    align-items: center;
    flex-direction: row;
    gap: 10px;
    cursor: pointer;
  }

  &-accent {
    width: 3px;
    min-width: 3px;
    height: 14px;
    border-radius: 999px;
    background: #e5e7eb;
  }

  &-content {
    min-width: 0;
    flex: 1;
    width: 100%;
    display: flex;
    flex-direction: row;
    align-items: center;
  }

  &-label {
    margin-right: 8px;
    color: #94a3b8;
    font-size: 11px;
    font-weight: 600;
    line-height: 16px;
    flex-shrink: 0;
  }

  &-text {
    display: block;
    min-width: 0;
    flex: 1;
    overflow: hidden;
    color: #475569;
    font-size: 12px;
    line-height: 16px;
    white-space: nowrap;
    text-overflow: ellipsis;
  }

  &-more {
    flex-shrink: 0;
    width: 28px;
    height: 28px;
    border: 0;
    background: transparent;
    color: #64748b;
    font-size: 16px;
    line-height: 1;
    cursor: pointer;

    span {
      display: inline-block;
      transition: transform 0.16s ease;
    }

    .is-open {
      transform: rotate(180deg);
    }
  }

  &-mask {
    position: fixed;
    inset: 0;
    z-index: 20;
  }

  &-panel {
    position: absolute;
    top: calc(100% - 4px);
    right: 8px;
    left: 16px;
    z-index: 21;
    max-height: 280px;
    overflow: auto;
    padding: 8px 0;
    border: 1px solid #e2e8f0;
    border-radius: 12px;
    background: #fff;
    box-shadow: 0 12px 32px rgba(15, 23, 42, 0.12);
  }

  &-panel-title {
    padding: 4px 14px 10px;
    color: #0f172a;
    font-size: 13px;
    font-weight: 700;
  }

  &-panel-item {
    display: flex;
    align-items: center;
    width: 100%;
    padding: 8px 10px 8px 14px;
    border: 0;
    background: transparent;
    text-align: left;
    cursor: pointer;

    &:hover,
    &.is-active {
      background: #f8fafc;
    }
  }

  &-panel-item-main {
    min-width: 0;
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 2px;

    strong {
      color: #334155;
      font-size: 12px;
      font-weight: 600;
    }

    span {
      overflow: hidden;
      color: #64748b;
      font-size: 12px;
      line-height: 16px;
      white-space: nowrap;
      text-overflow: ellipsis;
    }
  }

  &-panel-unpin {
    flex-shrink: 0;
    width: 24px;
    height: 24px;
    margin-left: 8px;
    border-radius: 999px;
    color: #94a3b8;
    font-size: 18px;
    line-height: 22px;
    text-align: center;

    &:hover {
      background: #fee2e2;
      color: #dc2626;
    }
  }
}
</style>
