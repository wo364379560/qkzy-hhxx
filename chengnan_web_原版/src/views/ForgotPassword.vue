<script setup lang="ts">
import { computed, reactive, ref } from "vue";
import { useRouter } from "vue-router";
import AuthShell from "@/components/auth/AuthShell.vue";
import { resetPassword, sendSMS } from "@/api";
import { useSmsCode } from "@/features/auth/useSmsCode";
import { useSingleFlight } from "@/composables/useSingleFlight";

const router = useRouter();

const form = reactive({
  phone: "",
  code: "",
  password: "",
});

const errorMessage = ref("");
const resetFlight = useSingleFlight();
const submitting = resetFlight.running;

const sms = useSmsCode(() => form.phone, (phone) => sendSMS(phone, "reset_password"));
const canSubmit = computed(() => !submitting.value);

async function handleRequestCode() {
  errorMessage.value = "";
  try {
    await sms.requestCode();
  } catch (error) {
    errorMessage.value =
      error instanceof Error ? error.message : "验证码发送失败";
  }
}

async function handleSubmit() {
  errorMessage.value = "";

  if (!form.phone.trim() || !form.code.trim()) {
    errorMessage.value = "请填写完整信息";
    return;
  }
  if (form.password.length < 8) {
    errorMessage.value = "密码至少 8 位";
    return;
  }

  try {
    await resetFlight.run(async () => {
      await resetPassword(form.phone.trim(), form.code.trim(), form.password);
      await router.replace({
        name: "login",
        query: { reset: "success" },
      });
    });
  } catch (error) {
    errorMessage.value =
      error instanceof Error ? error.message : "重置失败，请稍后重试";
  }
}
</script>

<template>
  <AuthShell title="忘记密码" subtitle="通过手机验证码重置密码。">
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
        />
      </label>
      <button
        class="sms-button"
        type="button"
        :disabled="!sms.canSend.value"
        @click="handleRequestCode"
      >
        {{
          sms.countdown.value > 0
            ? `${sms.countdown.value}s`
            : sms.sending.value
            ? "发送中..."
            : "发送验证码"
        }}
      </button>
    </div>

    <label class="field">
      <span class="field-label">新密码</span>
      <input
        v-model="form.password"
        class="field-input"
        type="password"
        name="new_password"
        autocomplete="new-password"
        placeholder="请输入新密码（至少 8 位）"
      />
    </label>

    <p v-if="errorMessage" class="error-text">
      {{ errorMessage }}
    </p>

    <button
      class="submit-button"
      type="button"
      :disabled="!canSubmit"
      @click="handleSubmit"
    >
      {{ submitting ? "提交中..." : "重置密码" }}
    </button>

    <template #footer>
      <button
        class="link-button"
        type="button"
        @click="router.push({ name: 'login' })"
      >
        返回登录
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
