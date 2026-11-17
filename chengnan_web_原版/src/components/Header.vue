<template>
  <div class="header">
    <div class="left">
      <div
        v-if="showType === 'menu'"
        class="menu"
        @click="toggleMenu"
      >
        <Icon :file="menuPNG" />
        <div class="menu-guide">
          {{ TUITranslateService.t("使用指引") }}
        </div>
      </div>
      <div
        v-else-if="showType === 'logo'"
        class="logo"
      >
        <div class="logo">
          <img
            class="logo-img"
            src="../assets/image/logo.svg"
            alt=""
          >
          <div class="logo-name label-app">{{ APP_NAME }}</div>
        </div>
      </div>
    </div>
    <div class="right">
      <div class="dropdown dropdown-disabled">
        <Icon :file="globalPNG" />
        <div class="dropdown-label">
          简体中文
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { withDefaults, defineProps, defineEmits } from '../TUIKit/adapter-vue';
import { TUITranslateService } from '@tencentcloud/chat-uikit-engine-lite';
import Icon from '../TUIKit/components/common/Icon.vue';
import menuPNG from '../assets/icon/menu.png';
import globalPNG from '../assets/icon/global.png';
import { APP_NAME } from '@/config/runtime';

withDefaults(
  defineProps<{
    showType: 'menu' | 'logo';
    defaultLanguage: string;
  }>(),
  {
    showType: 'logo',
    defaultLanguage: 'zh',
  },
);
const emits = defineEmits(['toggleMenu', 'closeMenu', 'changeLanguage']);
function toggleMenu() {
  emits('toggleMenu');
}
</script>

<style scoped lang="scss">
.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: #fff;
  box-sizing: border-box;
  padding: 20px 34px;
  user-select: none;

  .left {
    .menu,
    .logo {
      display: flex;
      justify-content: center;
      align-items: center;
      cursor: pointer;
      font-size: 14px;
      font-weight: 500;
      line-height: 20px;
    }

    .menu {
      .menu-guide {
        margin-left: 15px;
      }
    }

    .logo {
      .logo-img {
        width: 32px;
        margin-right: 10px;
      }

      .logo-name {
        font-family: PingFangSC-Medium;
        font-size: 18px;
        font-weight: 600;
        display: flex;
      }

      .label-app {
        color: #0f172a;
      }
    }
  }

  .right {
    .dropdown {
      display: flex;
      flex-direction: row;
      align-items: center;
      justify-content: center;
      cursor: default;

      .dropdown-label {
        padding: 0 8px;
      }
    }
  }
}
</style>
