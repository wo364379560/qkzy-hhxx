<template>
  <div
    v-if="visible"
    :class="[
      'tui-chat-contact-popup',
      !isPC && 'tui-chat-contact-popup-mobile',
    ]"
    @click.self="closePopup"
  >
    <div class="tui-chat-contact-popup-shell" @click.stop>
      <button
        class="tui-chat-contact-popup-close"
        type="button"
        aria-label="关闭"
        @touchstart.stop.prevent
        @touchend.stop.prevent="closePopup"
        @mousedown.stop.prevent
        @mouseup.stop.prevent
        @click.stop.prevent="closePopup"
      >
        <Icon :file="closeIcon" width="10px" height="10px" />
      </button>
      <div class="tui-chat-contact-popup-panel">
        <ContactInfo @switchConversation="$emit('switchConversation')" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import ContactInfo from '../TUIContact/contact-info/index.vue';
import Icon from '../common/Icon.vue';
import closeIcon from '../../assets/icon/icon-close.svg';
import { isPC } from '../../utils/env';

defineProps<{
  visible: boolean;
}>();

const emits = defineEmits(['close', 'switchConversation']);

function closePopup() {
  emits('close');
}
</script>

<style scoped lang="scss">
.tui-chat-contact-popup {
  position: absolute;
  inset: 0;
  z-index: 2000;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  box-sizing: border-box;
  background: rgba(15, 23, 42, 0.18);

  &-shell {
    position: relative;
    width: min(520px, 100%);
    height: min(640px, 100%);
    display: flex;
    align-items: stretch;
    justify-content: stretch;
    overflow: visible;
  }

  &-panel {
    position: relative;
    width: 100%;
    height: 100%;
    background: #f7f8fa;
    border-radius: 14px;
    overflow: hidden;
    box-shadow: 0 18px 48px rgba(15, 23, 42, 0.18);
  }

  &-close {
    position: absolute;
    top: 10px;
    right: 10px;
    z-index: 3;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 28px;
    height: 28px;
    padding: 0;
    border: 1px solid rgba(148, 163, 184, 0.22);
    border-radius: 999px;
    background: rgba(255, 255, 255, 0.92);
    box-shadow: 0 10px 24px rgba(15, 23, 42, 0.12);
    backdrop-filter: blur(6px);
    cursor: pointer;
    transition: transform 0.18s ease, box-shadow 0.18s ease,
      background-color 0.18s ease;

    &:hover {
      background: #ffffff;
      box-shadow: 0 12px 28px rgba(15, 23, 42, 0.16);
      transform: translateY(-1px);
    }

    &:active {
      transform: translateY(0);
      box-shadow: 0 8px 18px rgba(15, 23, 42, 0.14);
    }
  }
}

.tui-chat-contact-popup-mobile {
  padding: 0;

  .tui-chat-contact-popup-shell,
  .tui-chat-contact-popup-panel {
    width: 100%;
    height: 100%;
  }

  .tui-chat-contact-popup-panel {
    border-radius: 0;
  }
}
</style>
