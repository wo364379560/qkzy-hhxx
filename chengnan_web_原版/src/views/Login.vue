<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from "vue";
import { useRouter } from "vue-router";
import AuthShell from "@/components/auth/AuthShell.vue";
import PrivacyAgreementField from "@/components/auth/PrivacyAgreementField.vue";
import { WEB_TURNSTILE_SITE_KEY } from "@/config/runtime";
import {
  clearRememberedLoginCredentials,
  readRememberedLoginCredentials,
  saveRememberedLoginCredentials,
} from "@/features/auth/rememberCredentials";
import { useAuthSession } from "@/features/auth/session";
import QrcodeVue from "qrcode.vue";

declare global {
  interface Window {
    turnstile?: {
      render: (
        element: HTMLElement,
        options: {
          sitekey: string;
          action?: string;
          callback?: (token: string) => void;
          "expired-callback"?: () => void;
          "error-callback"?: () => void;
        }
      ) => string;
      reset: (widgetId?: string) => void;
      remove?: (widgetId?: string) => void;
    };
  }
}

const router = useRouter();
const authSession = useAuthSession();

const form = reactive({
  phone: "",
  password: "",
  rememberPassword: true,
  agreed: false,
});

const errorMessage = ref("");
const totpDialog = reactive({
  visible: false,
  mode: "verify" as "verify" | "setup",
  tempToken: "",
  totpSecret: "",
  totpUri: "",
  code: "",
});
const turnstileRequired = computed(() => Boolean(WEB_TURNSTILE_SITE_KEY));
const turnstileContainer = ref<HTMLElement | null>(null);
const turnstileToken = ref("");
const turnstileWidgetId = ref<string>();
const successMessage = computed(() =>
  router.currentRoute.value.query.reset === "success"
    ? "密码重置成功，请使用新密码登录"
    : ""
);

const canLoginSubmit = computed(
  () => !authSession.state.authenticating && (!turnstileRequired.value || Boolean(turnstileToken.value))
);
const canTOTPSubmit = computed(() => !authSession.state.authenticating);

let turnstileScriptPromise: Promise<void> | null = null;

function loadTurnstileScript() {
  if (window.turnstile) {
    return Promise.resolve();
  }
  if (turnstileScriptPromise) {
    return turnstileScriptPromise;
  }
  turnstileScriptPromise = new Promise((resolve, reject) => {
    const existing = document.querySelector<HTMLScriptElement>(
      'script[data-turnstile-script="true"]'
    );
    if (existing) {
      existing.addEventListener("load", () => resolve(), { once: true });
      existing.addEventListener(
        "error",
        () => reject(new Error("人机验证组件加载失败")),
        { once: true }
      );
      return;
    }
    const script = document.createElement("script");
    script.src =
      "https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit";
    script.async = true;
    script.defer = true;
    script.dataset.turnstileScript = "true";
    script.onload = () => resolve();
    script.onerror = () => reject(new Error("人机验证组件加载失败"));
    document.head.appendChild(script);
  });
  return turnstileScriptPromise;
}

async function renderTurnstile() {
  if (
    !turnstileRequired.value ||
    !turnstileContainer.value ||
    turnstileWidgetId.value
  ) {
    return;
  }
  await loadTurnstileScript();
  if (!window.turnstile || !turnstileContainer.value) {
    return;
  }
  turnstileWidgetId.value = window.turnstile.render(turnstileContainer.value, {
    sitekey: WEB_TURNSTILE_SITE_KEY,
    action: "web-login",
    callback: (token) => {
      turnstileToken.value = token;
    },
    "expired-callback": () => {
      turnstileToken.value = "";
    },
    "error-callback": () => {
      turnstileToken.value = "";
    },
  });
}

function resetTurnstile() {
  turnstileToken.value = "";
  if (turnstileRequired.value && window.turnstile && turnstileWidgetId.value) {
    window.turnstile.reset(turnstileWidgetId.value);
  }
}

function syncRememberedCredentials() {
  if (!form.rememberPassword) {
    clearRememberedLoginCredentials();
    return;
  }

  saveRememberedLoginCredentials({
    phone: form.phone.trim(),
    password: form.password,
  });
}

async function handleSubmit() {
  errorMessage.value = "";

  if (!form.phone.trim()) {
    errorMessage.value = "请输入手机号";
    return;
  }
  if (!form.password) {
    errorMessage.value = "请输入密码";
    return;
  }
  if (!form.agreed) {
    errorMessage.value = "请先阅读并同意隐私协议";
    return;
  }
  if (turnstileRequired.value && !turnstileToken.value) {
    errorMessage.value = "请先完成人机验证";
    return;
  }

  try {
    const loginResult = await authSession.loginWithPassword(
      form.phone.trim(),
      form.password,
      turnstileToken.value || undefined
    );
    if (typeof loginResult !== "string") {
      if (loginResult.type === "totp" || loginResult.type === "totp_setup") {
        totpDialog.visible = true;
        totpDialog.mode = loginResult.type === "totp_setup" ? "setup" : "verify";
        totpDialog.tempToken = loginResult.tempToken;
        totpDialog.totpSecret =
          loginResult.type === "totp_setup" ? loginResult.totpSecret : "";
        totpDialog.totpUri =
          loginResult.type === "totp_setup" ? loginResult.totpUri : "";
        totpDialog.code = "";
        resetTurnstile();
        return;
      }
      return;
    }
    syncRememberedCredentials();
    await router.replace({ name: loginResult });
  } catch (error) {
    const message =
      error instanceof Error ? error.message : "登录失败，请稍后重试";
    errorMessage.value = message;

    if (message.includes("未设置密码")) {
      errorMessage.value = "该账号未设置密码，请通过忘记密码重置密码";
    }
    resetTurnstile();
  }
}

async function handleTOTPSubmit() {
  errorMessage.value = "";
  const code = totpDialog.code.trim();
  if (!code) {
    errorMessage.value = "请输入 Google 验证码";
    return;
  }
  try {
    const routeName = await authSession.completeTOTPLogin(
      totpDialog.tempToken,
      code,
      totpDialog.mode
    );
    syncRememberedCredentials();
    totpDialog.visible = false;
    await router.replace({ name: routeName });
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : "验证失败，请稍后重试";
  }
}

onMounted(async () => {
  const rememberedCredentials = readRememberedLoginCredentials();
  if (rememberedCredentials) {
    form.phone = rememberedCredentials.phone;
    form.password = rememberedCredentials.password;
  }

  if (!turnstileRequired.value) {
    return;
  }
  await nextTick();
  try {
    await renderTurnstile();
  } catch (error) {
    errorMessage.value =
      error instanceof Error ? error.message : "人机验证组件加载失败";
  }
});

onBeforeUnmount(() => {
  if (window.turnstile?.remove && turnstileWidgetId.value) {
    window.turnstile.remove(turnstileWidgetId.value);
  }
});

watch(
  () => form.rememberPassword,
  (rememberPassword) => {
    if (!rememberPassword) {
      clearRememberedLoginCredentials();
    }
  }
);
</script>

<template>
  <AuthShell
    title="登录"
    subtitle="使用手机号和密码登录。"
  >
    <label class="field">
      <span class="field-label">手机号</span>
      <input
        v-model.trim="form.phone"
        class="field-input"
        type="tel"
        name="phone"
        autocomplete="tel"
        inputmode="tel"
        spellcheck="false"
        placeholder="请输入手机号"
      />
    </label>

    <label class="field">
      <span class="field-label">密码</span>
      <input
        v-model="form.password"
        class="field-input"
        type="password"
        name="password"
        autocomplete="current-password"
        placeholder="请输入密码"
      />
    </label>

    <label class="remember-field">
      <input
        v-model="form.rememberPassword"
        class="remember-field-checkbox"
        type="checkbox"
      />
      <span>记住密码</span>
    </label>

    <PrivacyAgreementField v-model="form.agreed" />

    <div v-if="turnstileRequired" class="turnstile-box">
      <div ref="turnstileContainer"></div>
    </div>

    <p v-if="errorMessage" class="error-text">
      {{ errorMessage }}
    </p>
    <p v-else-if="successMessage" class="success-text">
      {{ successMessage }}
    </p>

    <button
      class="submit-button"
      type="button"
      :disabled="!canLoginSubmit"
      @click="handleSubmit"
    >
      {{ authSession.state.authenticating ? "登录中..." : "登录" }}
    </button>

    <template #footer>
      <div class="footer-links">
        <button
          class="link-button"
          type="button"
          @click="router.push({ name: 'forgot-password' })"
        >
          忘记密码
        </button>
        <button
          class="link-button"
          type="button"
          @click="router.push({ name: 'register' })"
        >
          没有账号？去注册
        </button>
      </div>
    </template>

    <div v-if="totpDialog.visible" class="totp-mask">
      <div class="totp-dialog" :class="{ 'totp-dialog-setup': totpDialog.mode === 'setup' }">
        <div class="totp-title">
          {{ totpDialog.mode === "setup" ? "绑定 Google 验证器" : "Google 验证" }}
        </div>
        <p class="totp-desc">
          {{
            totpDialog.mode === "setup"
              ? "管理员已开启该账号的 Google 验证。请用验证器扫描二维码，再输入 6 位动态验证码完成绑定。"
              : "请输入验证器中的 6 位动态验证码。"
          }}
        </p>
        <div v-if="totpDialog.mode === 'setup' && totpDialog.totpUri" class="totp-setup">
          <div class="totp-qrcode">
            <QrcodeVue :value="totpDialog.totpUri" :size="180" level="H" />
          </div>
          <div class="totp-secret">{{ totpDialog.totpSecret }}</div>
        </div>
        <label class="field">
          <span class="field-label">验证码</span>
          <input
            v-model.trim="totpDialog.code"
            class="field-input"
            type="text"
            inputmode="numeric"
            maxlength="6"
            autocomplete="one-time-code"
            placeholder="请输入 6 位验证码"
            @keyup.enter="handleTOTPSubmit"
          />
        </label>
        <div class="totp-actions">
          <button class="totp-cancel" type="button" @click="totpDialog.visible = false">
            取消
          </button>
          <button class="totp-confirm" type="button" :disabled="!canTOTPSubmit" @click="handleTOTPSubmit">
            {{ authSession.state.authenticating ? "验证中..." : totpDialog.mode === "setup" ? "确认绑定" : "确认" }}
          </button>
        </div>
      </div>
    </div>
  </AuthShell>
</template>

<style scoped lang="scss">
.field {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.field-label {
  color: #334155;
  font-size: 14px;
  font-weight: 600;
}

.field-input {
  width: 100%;
  height: 44px;
  box-sizing: border-box;
  padding: 0 14px;
  border: 1px solid #dbe2ee;
  border-radius: 12px;
  font-size: 14px;
}

.remember-field {
  display: inline-flex;
  align-items: center;
  align-self: flex-start;
  gap: 8px;
  color: #475569;
  font-size: 14px;
  line-height: 20px;
  cursor: pointer;
  user-select: none;
}

.remember-field-checkbox {
  width: 16px;
  height: 16px;
  margin: 0;
  accent-color: #4361ee;
  cursor: pointer;
}

.submit-button,
.link-button {
  border: 0;
  cursor: pointer;
}

.submit-button:disabled {
  cursor: not-allowed;
  opacity: 0.6;
}

.submit-button {
  width: 100%;
  height: 48px;
  border-radius: 14px;
  color: #ffffff;
  background: linear-gradient(135deg, #4361ee, #3451d1);
  font-size: 15px;
  font-weight: 600;
}

.turnstile-box {
  min-height: 65px;
  display: flex;
  justify-content: center;
}

.link-button {
  color: #4361ee;
  background: transparent;
  font-size: 14px;
  font-weight: 600;
}

.footer-links {
  display: flex;
  justify-content: space-between;
  gap: 12px;
}

.error-text {
  margin: 0;
  color: #dc2626;
  font-size: 13px;
}

.success-text {
  margin: 0;
  color: #16a34a;
  font-size: 13px;
}

.totp-mask {
  position: fixed;
  inset: 0;
  z-index: 1000;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
  background: rgba(15, 23, 42, 0.46);
}

.totp-dialog {
  width: min(100%, 380px);
  padding: 22px;
  border-radius: 16px;
  background: #ffffff;
  box-shadow: 0 24px 70px rgba(15, 23, 42, 0.2);
}

.totp-dialog-setup {
  width: min(100%, 420px);
}

.totp-setup {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  margin: 0 0 16px;
}

.totp-qrcode {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 12px;
  border-radius: 12px;
  background: #f8fafc;
}

.totp-secret {
  max-width: 100%;
  word-break: break-all;
  color: #475569;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 12px;
  line-height: 1.5;
  text-align: center;
}

.totp-title {
  color: #111827;
  font-size: 18px;
  font-weight: 700;
}

.totp-desc {
  margin: 8px 0 18px;
  color: #64748b;
  font-size: 14px;
  line-height: 1.6;
}

.totp-actions {
  display: flex;
  gap: 10px;
  margin-top: 20px;
}

.totp-cancel,
.totp-confirm {
  flex: 1;
  height: 42px;
  border: 0;
  border-radius: 12px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
}

.totp-cancel {
  background: #f1f5f9;
  color: #475569;
}

.totp-confirm {
  background: #4361ee;
  color: #ffffff;
}
</style>
