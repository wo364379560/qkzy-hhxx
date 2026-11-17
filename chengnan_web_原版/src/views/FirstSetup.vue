<script setup lang="ts">
import { ref } from 'vue';
import { useRouter } from 'vue-router';
import AuthShell from '@/components/auth/AuthShell.vue';
import { useAuthSession } from '@/features/auth/session';

const router = useRouter();
const authSession = useAuthSession();

const nickname = ref('');
const errorMessage = ref('');

async function handleSubmit() {
  errorMessage.value = '';
  const trimmedNickname = nickname.value.trim();
  if (!trimmedNickname) {
    errorMessage.value = '请输入昵称';
    return;
  }
  if (trimmedNickname.length < 2) {
    errorMessage.value = '昵称至少 2 个字符';
    return;
  }

  try {
    const routeName = await authSession.completeFirstSetup(trimmedNickname);
    await router.replace({ name: routeName });
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '提交失败，请稍后重试';
  }
}
</script>

<template>
  <AuthShell
    title="完善个人资料"
    subtitle="首次进入前需要至少设置昵称，后端会同时激活账号。"
  >
    <label class="field">
      <span class="field-label">昵称</span>
      <input
        v-model="nickname"
        class="field-input"
        type="text"
        name="nickname"
        autocomplete="nickname"
        placeholder="请输入昵称"
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
      {{ authSession.state.authenticating ? '提交中...' : '完成设置，进入聊天' }}
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
