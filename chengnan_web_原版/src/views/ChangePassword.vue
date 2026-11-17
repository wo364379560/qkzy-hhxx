<script setup lang="ts">
import { reactive, ref } from 'vue';
import { useRouter } from 'vue-router';
import AuthShell from '@/components/auth/AuthShell.vue';
import { useAuthSession } from '@/features/auth/session';

const router = useRouter();
const authSession = useAuthSession();

const form = reactive({
  password: '',
  confirmPassword: '',
});

const errorMessage = ref('');

async function handleSubmit() {
  errorMessage.value = '';

  if (form.password.length < 8) {
    errorMessage.value = '密码至少 8 位';
    return;
  }
  if (form.password !== form.confirmPassword) {
    errorMessage.value = '两次输入的密码不一致';
    return;
  }

  try {
    const routeName = await authSession.completePasswordChange(form.password);
    await router.replace({ name: routeName });
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '修改密码失败';
  }
}
</script>

<template>
  <AuthShell
    title="修改初始密码"
    subtitle="你的账号由管理员创建，首次登录需要设置专属密码。"
  >
    <label class="field">
      <span class="field-label">新密码</span>
      <input
        v-model="form.password"
        class="field-input"
        type="password"
        name="new_password"
        autocomplete="new-password"
        placeholder="至少 8 位"
      >
    </label>
    <label class="field">
      <span class="field-label">确认密码</span>
      <input
        v-model="form.confirmPassword"
        class="field-input"
        type="password"
        name="confirm_password"
        autocomplete="new-password"
        placeholder="请再次输入新密码"
      >
    </label>
    <p
      v-if="errorMessage"
      class="error-text"
    >
      {{ errorMessage }}
    </p>
    <button
      class="submit-button"
      type="button"
      :disabled="authSession.state.authenticating"
      @click="handleSubmit"
    >
      {{ authSession.state.authenticating ? '提交中...' : '确认修改' }}
    </button>
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

.submit-button {
  width: 100%;
  height: 48px;
  border: 0;
  border-radius: 14px;
  color: #ffffff;
  background: linear-gradient(135deg, #4361ee, #3451d1);
  font-size: 15px;
  font-weight: 600;
  cursor: pointer;
}

.submit-button:disabled {
  cursor: not-allowed;
  opacity: 0.6;
}

.error-text {
  margin: 0;
  color: #dc2626;
  font-size: 13px;
}
</style>
