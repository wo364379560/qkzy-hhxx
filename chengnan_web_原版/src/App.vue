<template>
  <div id="app">
    <router-view
      :key="locale"
      :language="locale"
      @changeLanguage="changeLanguage"
    />
    <div v-if="authSession.state.recoveryStatus !== 'idle'" class="connection-notice" role="status">
      连接暂时不可用，登录状态已保留。
      <button type="button" @click="authSession.retryConnection().catch(() => {})">重试连接</button>
    </div>
    <!-- <ConferenceMainView displayMode="wake-up" /> -->
  </div>
</template>
<script setup lang="ts">
import { ref, watch } from "vue";
import { RouterView, useRouter } from "vue-router";
import { setChatUnauthorizedHandler } from "@/api";
import { useAuthSession } from "@/features/auth/session";
// import ConferenceMainView from '@tencentcloud/roomkit-web-vue3';
const router = useRouter();
const authSession = useAuthSession();
const locale = ref<string>("zh");

setChatUnauthorizedHandler(async () => {
  let recovered = false;
  try { recovered = await authSession.refreshCurrentSession({ force: true }); }
  catch { return false; }
  if (recovered) {
    return true;
  }

  if (authSession.currentSession.value) return false;
  if (router.currentRoute.value.name !== "login") {
    await router.replace({ name: "login" });
  }
  return false;
});

watch(
  () => authSession.currentSession.value?.userID,
  async (userID) => {
    if (!userID) {
      return;
    }
    const { ensureChatRuntime } = await import("@/bootstrap/chatRuntime");
    ensureChatRuntime({
      onKickedOut: async () => {
        if (router.currentRoute.value.name !== "login") {
          await authSession.clearSession();
          await router.replace({ name: "login" });
        }
      },
    });
  },
  { immediate: true }
);

function changeLanguage(language: string) {
  locale.value = language;
}
</script>

<style lang="scss">
html,
body,
#app {
  width: 100%;
  height: 100%;
  margin: 0;
}
</style>

<style scoped>.connection-notice {position:fixed;bottom:12px;left:12px;z-index:99999;background:#fff3cd;padding:12px;border-radius:8px}.connection-notice button{margin-left:12px}</style>
