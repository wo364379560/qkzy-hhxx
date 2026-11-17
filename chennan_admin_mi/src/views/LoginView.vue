<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '@/stores/auth'
import TOTPBindDialog from '@/components/TOTPBindDialog.vue'
import TOTPVerifyDialog from '@/components/TOTPVerifyDialog.vue'
import BackupCodesDialog from '@/components/BackupCodesDialog.vue'
import type { AdminUser } from '@/types'

declare global {
  interface Window {
    turnstile?: {
      render: (
        element: HTMLElement,
        options: {
          sitekey: string
          action?: string
          callback?: (token: string) => void
          'expired-callback'?: () => void
          'error-callback'?: () => void
        }
      ) => string
      reset: (widgetId?: string) => void
      remove?: (widgetId?: string) => void
    }
  }
}

const router = useRouter()
const authStore = useAuthStore()
const loading = ref(false)
const form = reactive({ username: '', password: '' })
const turnstileSiteKey = import.meta.env.VITE_TURNSTILE_SITE_KEY || ''
const turnstileRequired = computed(() => Boolean(turnstileSiteKey))
const turnstileContainer = ref<HTMLElement | null>(null)
const turnstileToken = ref('')
const turnstileWidgetId = ref<string>()

const showBindDialog = ref(false)
const showVerifyDialog = ref(false)
const showBackupCodesDialog = ref(false)
const totpUri = ref('')
const totpSecret = ref('')
const tempToken = ref('')
const backupCodes = ref<string[]>([])
const pendingBindAuth = ref<{ token: string; user: AdminUser } | null>(null)

let turnstileScriptPromise: Promise<void> | null = null

function loadTurnstileScript() {
  if (window.turnstile) {
    return Promise.resolve()
  }
  if (turnstileScriptPromise) {
    return turnstileScriptPromise
  }
  turnstileScriptPromise = new Promise((resolve, reject) => {
    const existing = document.querySelector<HTMLScriptElement>('script[data-turnstile-script="true"]')
    if (existing) {
      existing.addEventListener('load', () => resolve(), { once: true })
      existing.addEventListener('error', () => reject(new Error('人机验证组件加载失败')), { once: true })
      return
    }
    const script = document.createElement('script')
    script.src = 'https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit'
    script.async = true
    script.defer = true
    script.dataset.turnstileScript = 'true'
    script.onload = () => resolve()
    script.onerror = () => reject(new Error('人机验证组件加载失败'))
    document.head.appendChild(script)
  })
  return turnstileScriptPromise
}

async function renderTurnstile() {
  if (!turnstileRequired.value || !turnstileContainer.value || turnstileWidgetId.value) {
    return
  }
  await loadTurnstileScript()
  if (!window.turnstile || !turnstileContainer.value) {
    return
  }
  turnstileWidgetId.value = window.turnstile.render(turnstileContainer.value, {
    sitekey: turnstileSiteKey,
    action: 'admin-login',
    callback: (token) => {
      turnstileToken.value = token
    },
    'expired-callback': () => {
      turnstileToken.value = ''
    },
    'error-callback': () => {
      turnstileToken.value = ''
    },
  })
}

function resetTurnstile() {
  turnstileToken.value = ''
  if (turnstileRequired.value && window.turnstile && turnstileWidgetId.value) {
    window.turnstile.reset(turnstileWidgetId.value)
  }
}

async function handleLogin() {
  if (loading.value || showBindDialog.value || showVerifyDialog.value || showBackupCodesDialog.value) return
  if (!form.username || !form.password) {
    ElMessage.error('请输入用户名和密码')
    return
  }
  if (turnstileRequired.value && !turnstileToken.value) {
    ElMessage.error('请先完成人机验证')
    return
  }
  loading.value = true
  try {
    const data = await authStore.login(form.username, form.password, turnstileToken.value || undefined)
    if (data.require_totp_setup) {
      totpUri.value = data.totp_uri
      totpSecret.value = data.totp_secret
      tempToken.value = data.temp_token
      showBindDialog.value = true
      resetTurnstile()
    } else if (data.require_totp_verify) {
      tempToken.value = data.temp_token
      showVerifyDialog.value = true
      resetTurnstile()
    } else {
      router.push('/dashboard')
    }
  } catch (e: any) {
    ElMessage.error(e.message || '登录失败')
    resetTurnstile()
  } finally {
    loading.value = false
  }
}

function handleBindSuccess(data: { token: string; user: AdminUser; backupCodes: string[] }) {
  pendingBindAuth.value = { token: data.token, user: data.user }
  backupCodes.value = data.backupCodes
  showBackupCodesDialog.value = true
}

function handleBackupCodesClose() {
  if (!pendingBindAuth.value) return
  authStore.setAuth(pendingBindAuth.value.token, pendingBindAuth.value.user)
  pendingBindAuth.value = null
  router.push('/dashboard')
}

function handleVerifySuccess(data: { token: string; user: AdminUser }) {
  authStore.setAuth(data.token, data.user)
  router.push('/dashboard')
}

onMounted(async () => {
  if (!turnstileRequired.value) {
    return
  }
  await nextTick()
  try {
    await renderTurnstile()
  } catch (error: any) {
    ElMessage.error(error?.message || '人机验证组件加载失败')
  }
})

onBeforeUnmount(() => {
  if (window.turnstile?.remove && turnstileWidgetId.value) {
    window.turnstile.remove(turnstileWidgetId.value)
  }
})
</script>

<template>
  <div class="login-page">
    <div class="login-box">
      <!-- Header -->
      <div class="login-header">
        <div class="login-logo">
          <img src="/logo.png" alt="睿之禾 logo" class="login-logo-image" />
        </div>
        <h1 class="login-title">睿之禾</h1>
        <p class="login-sub">管理后台</p>
      </div>

      <!-- Card -->
      <div class="login-card">
        <div class="form-item">
          <label class="form-label">用户名</label>
          <input
            v-model="form.username"
            type="text"
            placeholder="请输入用户名"
            class="input"
            @keyup.enter="handleLogin"
          />
        </div>

        <div class="form-item">
          <label class="form-label">密码</label>
          <input
            v-model="form.password"
            type="password"
            placeholder="请输入密码"
            class="input"
            @keyup.enter="handleLogin"
          />
        </div>

        <div v-if="turnstileRequired" class="turnstile-box">
          <div ref="turnstileContainer"></div>
        </div>

        <button
          @click="handleLogin"
          :disabled="loading || showBindDialog || showVerifyDialog || showBackupCodesDialog || (turnstileRequired && !turnstileToken)"
          class="btn-primary login-btn"
        >
          <svg v-if="loading" class="spin-icon animate-spin" fill="none" viewBox="0 0 24 24">
            <circle style="opacity:0.25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
            <path style="opacity:0.75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/>
          </svg>
          {{ loading ? '登录中...' : '登录' }}
        </button>
      </div>
    </div>

    <TOTPBindDialog
      v-model:visible="showBindDialog"
      :totp-uri="totpUri"
      :totp-secret="totpSecret"
      :temp-token="tempToken"
      @success="handleBindSuccess"
    />

    <TOTPVerifyDialog
      v-model:visible="showVerifyDialog"
      :temp-token="tempToken"
      @success="handleVerifySuccess"
    />

    <BackupCodesDialog
      v-model:visible="showBackupCodesDialog"
      :codes="backupCodes"
      @update:visible="(val) => { if (!val) handleBackupCodesClose() }"
    />
  </div>
</template>

<style scoped>
.login-page {
  min-height: 100vh;
  background: #f5f7fa;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px;
}

.login-box {
  width: 100%;
  max-width: 380px;
}

.login-header {
  text-align: center;
  margin-bottom: 24px;
}

.login-logo {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 58px;
  height: 58px;
  border-radius: 16px;
  background: #fff;
  border: 1px solid #e2e8f0;
  box-shadow: 0 10px 24px rgba(15, 23, 42, 0.08);
  margin-bottom: 12px;
  overflow: hidden;
}

.login-logo-image {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.login-title {
  font-size: 20px;
  font-weight: 700;
  color: #1d2433;
  letter-spacing: -0.02em;
}

.login-sub {
  font-size: 13px;
  color: #94a3b8;
  margin-top: 4px;
}

.login-card {
  background: #fff;
  border: 1px solid #edf0f5;
  border-radius: 10px;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.05);
  padding: 28px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.form-item {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.form-label {
  font-size: 12px;
  font-weight: 500;
  color: #475569;
}

.login-btn {
  margin-top: 4px;
  gap: 8px;
}

.turnstile-box {
  min-height: 65px;
  display: flex;
  justify-content: center;
}

.spin-icon {
  width: 16px;
  height: 16px;
  flex-shrink: 0;
}
</style>
