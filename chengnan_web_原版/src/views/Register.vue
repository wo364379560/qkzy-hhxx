<script setup lang="ts">
import { computed, reactive, ref } from 'vue';
import { useRouter } from 'vue-router';
import AuthShell from '@/components/auth/AuthShell.vue';
import PrivacyAgreementField from '@/components/auth/PrivacyAgreementField.vue';
import { sendSMS } from '@/api';
import { useAuthSession } from '@/features/auth/session';
import { useSmsCode } from '@/features/auth/useSmsCode';

const router = useRouter();
const authSession = useAuthSession();

const form = reactive({
  nickname: '',
  phone: '',
  code: '',
  password: '',
  inviteCode: '',
  agreed: false,
});

const errorMessage = ref('');

const sms = useSmsCode(() => form.phone, (phone) => sendSMS(phone, "register"));
const canSubmit = computed(() => !authSession.state.authenticating);

async function handleRegister() {
  errorMessage.value = '';

  if (!form.nickname.trim() || !form.phone.trim() || !form.code.trim() || !form.password) {
    errorMessage.value = '请填写所有必填项';
    return;
  }
  if (!form.agreed) {
    errorMessage.value = '请先阅读并同意隐私协议';
    return;
  }

  try {
    const routeName = await authSession.registerAccount({
      nickname: form.nickname.trim(),
      phone: form.phone.trim(),
      code: form.code.trim(),
      password: form.password,
      inviteCode: form.inviteCode.trim() || undefined,
    });
    await router.replace({ name: routeName });
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '注册失败，请稍后重试';
  }
}

async function handleRequestCode() {
  errorMessage.value = '';
  try {
    await sms.requestCode();
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '验证码发送失败';
  }
}
</script>

<template>
  <AuthShell
    title="创建账号"
    subtitle="完成注册后即可进入睿之禾，继续使用社区、记账与协作能力。"
  >
    <label class="field">
      <span class="field-label">昵称</span>
      <input
        v-model.trim="form.nickname"
        class="field-input"
        type="text"
        name="nickname"
        autocomplete="nickname"
        placeholder="请输入昵称"
      >
    </label>
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
      >
    </label>
    <div class="field-group">
      <label class="field field-grow">
        <span class="field-label">验证码</span>
        <input
          v-model.trim="form.code"
          class="field-input"
          type="text"
          name="sms_code"
          autocomplete="one-time-code"
          inputmode="numeric"
          spellcheck="false"
          placeholder="请输入验证码"
        >
      </label>
      <button
        class="sms-button"
        type="button"
        :disabled="!sms.canSend.value"
        @click="handleRequestCode"
      >
        {{ sms.countdown.value > 0 ? `${sms.countdown.value}s` : sms.sending.value ? '发送中...' : '获取验证码' }}
      </button>
    </div>
    <label class="field">
      <span class="field-label">密码</span>
      <input
        v-model="form.password"
        class="field-input"
        type="password"
        name="password"
        autocomplete="new-password"
        placeholder="请设置密码"
      >
    </label>
    <label class="field">
      <span class="field-label">邀请码</span>
      <input
        v-model.trim="form.inviteCode"
        class="field-input"
        type="text"
        name="invite_code"
        autocomplete="off"
        spellcheck="false"
        placeholder="选填"
      >
    </label>
    <PrivacyAgreementField v-model="form.agreed" />
    <p
      v-if="errorMessage"
      class="error-text"
    >
      {{ errorMessage }}
    </p>
    <button
      class="submit-button"
      type="button"
      :disabled="!canSubmit"
      @click="handleRegister"
    >
      {{ authSession.state.authenticating ? '注册中...' : '注册' }}
    </button>

    <template #footer>
      <button
        class="link-button"
        type="button"
        @click="router.push({ name: 'login' })"
      >
        已有账号？去登录
      </button>
    </template>
  </AuthShell>
</template>

<style scoped lang="scss">
.field {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.field-group {
  display: flex;
  gap: 12px;
  align-items: flex-end;
}

.field-grow {
  flex: 1;
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

.sms-button,
.submit-button,
.link-button {
  border: 0;
  cursor: pointer;
}

.sms-button {
  min-width: 118px;
  height: 44px;
  padding: 0 16px;
  border-radius: 12px;
  color: #4361ee;
  background: #eef2ff;
  font-size: 14px;
  font-weight: 600;
}

.sms-button:disabled,
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

.link-button {
  width: 100%;
  color: #4361ee;
  background: transparent;
  font-size: 14px;
  font-weight: 600;
}

.error-text {
  margin: 0;
  color: #dc2626;
  font-size: 13px;
}
</style>
