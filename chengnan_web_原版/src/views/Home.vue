<!-- eslint-disable vue/multi-word-component-names -->
<template>
  <div
    :id="isPC ? 'preloadedImages' : ''"
    :class="['home', isH5 && 'home-h5']"
  >
    <ReportDialogHost />
    <DoMattxCallOverlay />
    <StandardCallKit v-if="callIntegrationState.loaded" />
    <div v-if="callIntegrationState.status === 'loading' || callIntegrationState.status === 'error'" class="call-connection-status" role="status">
      {{ callIntegrationState.status === 'loading' ? '通话连接中，聊天可正常使用' : '通话暂时不可用，聊天可正常使用' }}
      <button v-if="callIntegrationState.status === 'error'" @click="retryCallIntegration().catch(() => {})">重试通话连接</button>
    </div>
    <div class="home-container">
      <div
        v-if="isPC"
        class="home-header"
      >
        <Header
          showType="logo"
          :defaultLanguage="locale"
          @changeLanguage="changeLanguage"
        />
      </div>
      <div class="home-main">
        <div class="home-TUIKit">
          <div
            v-if="isPC || !currentConversationID"
            class="home-TUIKit-navbar"
          >
            <NavBar
              v-model:currentNavBar="currentNavBar"
              v-model:isSettingShow="isSettingShow"
            >
              <template #profile>
                <Profile display-type="profile" />
              </template>
              <template #setting>
                <Profile
                  v-model:showSetting="isSettingShow"
                  display-type="setting"
                />
              </template>
            </NavBar>
          </div>
          <div
            v-if="isPC"
            class="home-TUIKit-main"
          >
            <div
              v-show="currentNavBar === 'message'"
              class="home-TUIKit-main"
            >
              <div class="home-conversation">
                <FriendSearch />
                <TUIConversation :show-header="false" />
              </div>
              <div class="home-chat">
                <TUIChat>
                  <ChatDefaultContent />
                </TUIChat>
                <TUIGroup class="chat-aside" />
                <TUISearch
                  class="chat-aside"
                  searchType="conversation"
                />
              </div>
              <TUIContact display-type="selectFriend" />
            </div>
            <div
              v-show="currentNavBar === 'relation'"
              class="home-TUIKit-main"
            >
              <TUIContact
                display-type="contactList"
                @switchConversation="currentNavBar = 'message'"
              />
            </div>
          </div>
          <div
            v-else-if="isH5"
            class="home-TUIKit-main"
          >
            <div
              v-if="!currentConversationID"
              class="home-TUIKit-main"
            >
              <div
                v-show="currentNavBar === 'message'"
                class="home-TUIKit-main"
              >
                <FriendSearch />
                <TUIConversation :show-header="false" />
                <TUIContact display-type="selectFriend" />
              </div>
              <div
                v-show="currentNavBar === 'relation'"
                class="home-TUIKit-main"
              >
                <TUIContact
                  display-type="contactList"
                  @switchConversation="currentNavBar = 'message'"
                />
              </div>
              <div
                v-show="currentNavBar === 'profile'"
                class="home-TUIKit-main"
              >
                <Profile display-type="all" />
              </div>
            </div>
            <TUIChat v-else />
            <TUIGroup class="chat-popup" />
            <TUISearch
              class="chat-popup"
              searchType="conversation"
            />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted, defineAsyncComponent } from '../TUIKit/adapter-vue';
import { TUIStore, StoreName } from '@tencentcloud/chat-uikit-engine-lite';
import { isPC, isH5 } from '../TUIKit/utils/env';
import { callIntegrationState, retryCallIntegration } from '@/bootstrap/callRuntime';
import { ensureDoMattxCallBridge } from '@/features/call/doMattxCallBridge';
import { bindCallHistoryOwner } from '@/features/call/localCallHistory';
import { useAuthSession } from '@/features/auth/session';

import StandardCallKit from '@/components/AsyncTUICallKit';
const DoMattxCallOverlay = defineAsyncComponent(() => import('@/features/call/DoMattxCallOverlay.vue'));
const TUIChat = defineAsyncComponent(() => import('../TUIKit/components/TUIChat'));
const TUIConversation = defineAsyncComponent(() => import('../TUIKit/components/TUIConversation'));
const TUIContact = defineAsyncComponent(() => import('../TUIKit/components/TUIContact'));
const TUIGroup = defineAsyncComponent(() => import('../TUIKit/components/TUIGroup'));
const TUISearch = defineAsyncComponent(() => import('../TUIKit/components/TUISearch'));
const Header = defineAsyncComponent(() => import('../components/Header.vue'));
const NavBar = defineAsyncComponent(() => import('../components/NavBar.vue'));
const FriendSearch = defineAsyncComponent(() => import('../components/FriendSearch.vue'));
const Profile = defineAsyncComponent(() => import('./Profile.vue'));
const ChatDefaultContent = defineAsyncComponent(() => import('../components/ChatDefaultContent.vue'));
const ReportDialogHost = defineAsyncComponent(() => import('../components/ReportDialogHost.vue'));
const { currentSession } = useAuthSession();

const props = withDefaults(
  defineProps<{
    language: string;
  }>(),
  {
    language: 'zh',
  },
);
const emits = defineEmits(['changeLanguage']);

const locale = ref<string>(props.language);
const currentNavBar = ref<string>('message');
const currentConversationID = ref<string>('');
const isSettingShow = ref<boolean>(false);

function changeLanguage(language: string) {
  emits('changeLanguage', language);
}

const onCurrentConversationIDUpdated = (id: string) => {
  currentConversationID.value = id;
};

const onCurrentHomeNavBarUpdated = (nav: string) => {
  if (nav === 'message' || nav === 'relation') {
    currentNavBar.value = nav;
  }
};

watch(
  () => currentSession.value?.userID || '',
  (userID) => {
    bindCallHistoryOwner(userID);
  },
  { immediate: true },
);

onMounted(() => {
  ensureDoMattxCallBridge();
  bindCallHistoryOwner(currentSession.value?.userID || '');

  TUIStore.watch(StoreName.CONV, {
    currentConversationID: onCurrentConversationIDUpdated,
  });

  TUIStore.watch(StoreName.CUSTOM, {
    currentHomeNavBar: onCurrentHomeNavBarUpdated,
  });
});

onUnmounted(() => {
  TUIStore.unwatch(StoreName.CONV, {
    currentConversationID: onCurrentConversationIDUpdated,
  });

  TUIStore.unwatch(StoreName.CUSTOM, {
    currentHomeNavBar: onCurrentHomeNavBarUpdated,
  });
});
</script>

<style lang="scss">
@import "../styles/home";
</style>

<style scoped>.call-connection-status{position:fixed;right:12px;bottom:12px;z-index:50;padding:8px 12px;background:#fff7df;border-radius:6px;font-size:13px}.call-connection-status button{margin-left:8px}</style>
