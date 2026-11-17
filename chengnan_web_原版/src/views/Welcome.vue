<script setup lang="ts">
import { useRouter } from 'vue-router';
import { APP_NAME } from '@/config/runtime';

const router = useRouter();

const featureList = [
  '安全登录',
  '快速注册',
  '轻松沟通',
];
</script>

<template>
  <div class="welcome-page">
    <main class="welcome-card">
      <div class="welcome-topline" />
      <p class="welcome-kicker">WELCOME</p>
      <p class="welcome-brand">{{ APP_NAME }}</p>

      <h1 class="welcome-title">
        欢迎使用，
        <span class="welcome-title-accent">{{ APP_NAME }}</span>
      </h1>
      <p class="welcome-subtitle">
        登录或注册后，即可开始聊天，与朋友轻松保持联系。
      </p>

      <div class="welcome-features">
        <span
          v-for="item in featureList"
          :key="item"
          class="welcome-feature"
        >
          {{ item }}
        </span>
      </div>

      <div class="welcome-actions">
        <button
          class="welcome-button welcome-button-primary"
          type="button"
          @click="router.push({ name: 'login' })"
        >
          <span class="welcome-button-title">登录</span>
          <small class="welcome-button-desc">已有账号</small>
        </button>
        <button
          class="welcome-button welcome-button-secondary"
          type="button"
          @click="router.push({ name: 'register' })"
        >
          <span class="welcome-button-title">注册</span>
          <small class="welcome-button-desc">创建新账号</small>
        </button>
      </div>

      <div class="welcome-footer">
        <span class="welcome-footer-item">简单开始</span>
        <span class="welcome-footer-divider" />
        <span class="welcome-footer-item">安心沟通</span>
      </div>
    </main>
  </div>
</template>

<style scoped lang="scss">
.welcome-page {
  min-height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px 16px;
  background:
    radial-gradient(circle at top left, rgba(67, 97, 238, 0.08), transparent 34%),
    linear-gradient(180deg, #f8fafc 0%, #f3f6fb 100%);
}

.welcome-card {
  width: min(100%, 420px);
  padding: 34px 28px 28px;
  border: 1px solid rgba(148, 163, 184, 0.14);
  border-radius: 28px;
  background: rgba(255, 255, 255, 0.98);
  box-shadow:
    0 24px 60px rgba(15, 23, 42, 0.08),
    0 1px 0 rgba(255, 255, 255, 0.8) inset;
}

.welcome-topline {
  width: 64px;
  height: 4px;
  border-radius: 999px;
  background: linear-gradient(90deg, #4361ee, #7c93ff);
  margin-bottom: 18px;
}

.welcome-kicker {
  margin: 0;
  color: #64748b;
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.16em;
}

.welcome-brand {
  margin: 10px 0 0;
  color: #0f172a;
  font-size: 17px;
  font-weight: 700;
}

.welcome-title {
  margin: 26px 0 0;
  color: #0f172a;
  font-size: 42px;
  font-weight: 800;
  line-height: 0.98;
  letter-spacing: -0.04em;
}

.welcome-title-accent {
  display: block;
  color: #4361ee;
}

.welcome-subtitle {
  margin: 16px 0 0;
  color: #64748b;
  font-size: 15px;
  line-height: 1.8;
}

.welcome-features {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 20px;
}

.welcome-feature {
  display: inline-flex;
  align-items: center;
  min-height: 32px;
  padding: 0 11px;
  border-radius: 999px;
  background: #f1f5f9;
  color: #475569;
  font-size: 12px;
  font-weight: 600;
}

.welcome-actions {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
  margin-top: 30px;
}

.welcome-button {
  width: 100%;
  min-height: 68px;
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  justify-content: center;
  gap: 4px;
  padding: 0 18px;
  border: 1px solid transparent;
  border-radius: 16px;
  cursor: pointer;
  transition:
    transform 160ms ease,
    box-shadow 160ms ease,
    background-color 160ms ease,
    border-color 160ms ease;
}

.welcome-button:hover {
  transform: translateY(-2px);
}

.welcome-button:focus-visible {
  outline: 3px solid rgba(67, 97, 238, 0.2);
  outline-offset: 2px;
}

.welcome-button-primary {
  color: #ffffff;
  background: linear-gradient(135deg, #4361ee, #3451d1);
  box-shadow: 0 14px 28px rgba(67, 97, 238, 0.2);
}

.welcome-button-secondary {
  color: #1e293b;
  background: #ffffff;
  border-color: rgba(148, 163, 184, 0.2);
}

.welcome-button-title {
  font-size: 16px;
  font-weight: 700;
}

.welcome-button-desc {
  font-size: 12px;
  font-weight: 500;
  opacity: 0.74;
}

.welcome-footer {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  margin-top: 22px;
  color: #64748b;
  font-size: 13px;
}

.welcome-footer-divider {
  width: 4px;
  height: 4px;
  border-radius: 50%;
  background: rgba(100, 116, 139, 0.45);
}

@media (max-width: 640px) {
  .welcome-card {
    padding: 30px 22px;
  }

  .welcome-title {
    font-size: 36px;
  }

  .welcome-actions {
    grid-template-columns: 1fr;
  }
}

@media (prefers-reduced-motion: reduce) {
  .welcome-button {
    transition: none;
  }

  .welcome-button:hover {
    transform: none;
  }
}
</style>
