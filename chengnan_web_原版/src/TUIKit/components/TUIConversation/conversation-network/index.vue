<template>
  <div
    v-if="isNotNetwork"
    class="network"
  >
    <i class="icon icon-error">!</i>
    <p class="network-content">
      {{
        TUITranslateService.t("TUIConversation.网络异常，请您检查网络设置")
      }}
    </p>
  </div>
</template>
<script lang="ts" setup>
import TUIChatEngine, {
  TUIStore,
  StoreName,
  TUITranslateService,
} from '@tencentcloud/chat-uikit-engine-lite';
import {
  onMounted,
  onUnmounted,
  ref,
} from '../../../adapter-vue';

const isNotNetwork = ref(false);

const userWatchOptions = {
  netStateChange: (value: string) => {
    isNotNetwork.value = (value === TUIChatEngine.TYPES.NET_STATE_DISCONNECTED);
  },
};

onMounted(() => {
  const netState = TUIStore.getData(StoreName.USER, 'netStateChange');
  if (netState) {
    userWatchOptions.netStateChange(netState);
  }
  TUIStore.watch(StoreName.USER, userWatchOptions);
});

onUnmounted(() => {
  TUIStore.unwatch(StoreName.USER, userWatchOptions);
});
</script>

<style lang="scss" scoped src="../style/index.scss"></style>
