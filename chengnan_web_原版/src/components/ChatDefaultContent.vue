<template>
  <div class="welcome">
    <div class="welcome-title">
      欢迎来到
      <img class="logo" src="../assets/image/logo.svg" alt="" />
      {{ APP_NAME }}
    </div>
    <div class="welcome-content">
      选择一个会话开始聊天，或从通讯录发起新的对话。
    </div>
  </div>
</template>
<script setup lang="ts">
import { APP_NAME } from '@/config/runtime';
</script>
<style lang="scss" scoped>
.welcome {
  width: 100%;
  height: 100%;
  box-sizing: border-box;
  padding-left: 40px;
  padding-top: 100px;
  display: flex;
  flex-direction: column;
  background: url("../assets/image/login-background.webp") no-repeat;
  background-size: cover;
  background-position-x: -17px;
  background-position-y: 173px;
  .welcome-title {
    font-size: 1.75rem;
    display: flex;
    align-items: center;
    font-family: PingFangSC-Medium;
    font-weight: 500;
    color: #000;
    .logo {
      width: 40px;
      padding-left: 0.98rem;
      padding-right: 0.98rem;
    }
  }
  .welcome-content {
    padding-top: 1.88rem;
    max-width: 393px;
    font-size: 16px;
    line-height: 24px;
    font-family: PingFangSC-Regular;
    font-weight: 400;
    color: #666;
  }
}
</style>
