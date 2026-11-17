<template>
  <div
    v-if="isOverlayShow"
    ref="overlayDomRef"
    class="overlay-container"
    :style="{
      position: props.isFullScreen ? 'fixed' : 'absolute',
      zIndex: props.zIndex,
      pointerEvents: props.visible ? 'auto' : 'none',
    }"
  >
    <div
      v-if="props.useMask"
      :class="{
        'overlay-mask': true,
        'fade-in': props.visible,
      }"
      :style="{
        backgroundColor: props.maskColor,
      }"
      @click="onOverlayClick"
      @touchstart.prevent.stop="onOverlayClick"
    />
    <div
      :class="{
        'overlay-content': true,
        'full-screen': props.isFullScreen,
      }"
      @click="onContentClick"
      @touchstart.prevent.stop="onContentClick"
    >
      <slot />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, withDefaults, onUnmounted } from '../../../adapter-vue';
export interface IOverlayProps {
  visible?: boolean;
  zIndex?: number | undefined;
  useMask?: boolean | undefined;
  maskColor?: string | undefined;
  isFullScreen?: boolean | undefined;
  width?: string;
  height?: string;
}

const emits = defineEmits(['onOverlayClick']);

const props = withDefaults(defineProps<IOverlayProps>(), {
  visible: true,
  zIndex: 9999,
  useMask: true,
  isFullScreen: true,
  maskColor: 'rgba(0, 0, 0, 0.6)',
  width: 'auto',
  height: 'auto',
});

const overlayDomRef = ref<HTMLElement>();
const isOverlayShow = ref<boolean>(props.visible);
let hideTimer: ReturnType<typeof setTimeout> | undefined;

function cancelPendingHide() {
  if (hideTimer !== undefined) {
    clearTimeout(hideTimer);
    hideTimer = undefined;
  }
}

watch(() => props.visible, (visible: boolean) => {
  // A previous close (including the initial hidden state) must never hide a
  // newly opened dialog while its parent still believes it is visible.
  cancelPendingHide();
  if (visible) {
    isOverlayShow.value = true;
  } else if (isOverlayShow.value) {
    hideTimer = setTimeout(() => {
      isOverlayShow.value = false;
      hideTimer = undefined;
    }, 150);
  }
}, {
  immediate: true,
});

onUnmounted(cancelPendingHide);

function onOverlayClick() {
  emits('onOverlayClick');
}

function onContentClick(event: Event) {
  if (event.target === event.currentTarget) {
    onOverlayClick();
  }
}

defineExpose({
  overlayDomRef,
});
</script>

<style scoped lang="scss">
.overlay-container {
  position: fixed;
  top: 0;
  bottom: 0;
  left: 0;
  right: 0;
  z-index: 9999;
  display: flex;
  align-items: center;
  justify-content: center;

  .overlay-mask {
    z-index: -1;
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background-color: rgba(0, 0, 0, 0.6);
    opacity: 0;
    transition: opacity 0.15s linear;
    animation: fade-in 0.15s linear;
  }

  .full-screen{
    width: 100%;
    height: 100%;
    display: flex;
    justify-content: center;
    align-items: center;
  }
}

.overlay-mask.fade-in {
  opacity: 1;
}

@keyframes fade-in {
  0% {
    opacity: 0;
  }

  100% {
    opacity: 1;
  }
}
</style>
