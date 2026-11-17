<template>
  <Dialog
    :show="true"
    :title="TUITranslateService.t('Profile.重置密码')"
    :isFooterShow="false"
    :center="true"
    @update:show="closeDialog"
  >
    <div class="reset-password-dialog">
      <p class="reset-password-dialog__desc">
        {{ TUITranslateService.t("Profile.请输入当前密码并设置一个新的登录密码") }}
      </p>
      <label class="reset-password-dialog__field">
        <span class="reset-password-dialog__label">
          {{ TUITranslateService.t("Profile.当前密码") }}
        </span>
        <input
          v-model="form.currentPassword"
          type="password"
          autocomplete="current-password"
          :placeholder="TUITranslateService.t('Profile.请输入当前密码')"
        />
      </label>
      <label class="reset-password-dialog__field">
        <span class="reset-password-dialog__label">
          {{ TUITranslateService.t("Profile.新密码") }}
        </span>
        <input
          v-model="form.newPassword"
          type="password"
          autocomplete="new-password"
          :placeholder="TUITranslateService.t('Profile.请输入新密码')"
        />
      </label>
      <label class="reset-password-dialog__field">
        <span class="reset-password-dialog__label">
          {{ TUITranslateService.t("Profile.确认新密码") }}
        </span>
        <input
          v-model="form.confirmPassword"
          type="password"
          autocomplete="new-password"
          :placeholder="TUITranslateService.t('Profile.请再次输入新密码')"
        />
      </label>
      <p
        v-if="errorMessage"
        class="reset-password-dialog__error"
      >
        {{ errorMessage }}
      </p>
      <div class="reset-password-dialog__actions">
        <button
          class="btn btn-cancel"
          type="button"
          @click="closeDialog"
        >
          {{ TUITranslateService.t("Profile.取消") }}
        </button>
        <button
          class="btn btn-primary"
          type="button"
          :disabled="submitting"
          @click="submit"
        >
          {{
            submitting
              ? TUITranslateService.t("Profile.提交中")
              : TUITranslateService.t("Profile.确认重置")
          }}
        </button>
      </div>
    </div>
  </Dialog>
</template>

<script setup lang="ts">
import { reactive, ref } from "vue";
import { TUITranslateService } from "@tencentcloud/chat-uikit-engine-lite";
import Dialog from "@/TUIKit/components/common/Dialog/index.vue";
import { updatePassword } from "@/api";
import { useSingleFlight } from "@/composables/useSingleFlight";

const emit = defineEmits<{
  (e: "close"): void;
  (e: "success"): void;
}>();

const form = reactive({
  currentPassword: "",
  newPassword: "",
  confirmPassword: "",
});

const errorMessage = ref("");
const submitFlight = useSingleFlight();
const submitting = submitFlight.running;

function closeDialog() {
  emit("close");
}

async function submit() {
  if (!form.currentPassword.trim()) {
    errorMessage.value = TUITranslateService.t("Profile.请输入当前密码");
    return;
  }
  if (form.newPassword.length < 8) {
    errorMessage.value = TUITranslateService.t("Profile.密码至少8位");
    return;
  }
  if (form.newPassword !== form.confirmPassword) {
    errorMessage.value = TUITranslateService.t("Profile.两次输入的密码不一致");
    return;
  }

  errorMessage.value = "";
  try {
    await submitFlight.run(async () => {
      await updatePassword(form.currentPassword, form.newPassword);
      emit("success");
      emit("close");
    });
  } catch (error) {
    errorMessage.value =
      error instanceof Error
        ? error.message
        : TUITranslateService.t("Profile.重置密码失败");
  }
}
</script>

<style scoped lang="scss">
.reset-password-dialog {
  width: min(420px, 80vw);
  padding: 8px 8px 4px;
}

.reset-password-dialog__desc {
  margin: 0 0 16px;
  color: #666;
  font-size: 14px;
  line-height: 1.6;
}

.reset-password-dialog__field {
  display: block;
  margin-bottom: 14px;
}

.reset-password-dialog__label {
  display: block;
  margin-bottom: 6px;
  color: #222;
  font-size: 13px;
  font-weight: 600;
}

.reset-password-dialog__field input {
  width: 100%;
  border: 1px solid #d8dde8;
  border-radius: 10px;
  padding: 10px 12px;
  font-size: 14px;
  outline: none;
  box-sizing: border-box;
}

.reset-password-dialog__field input:focus {
  border-color: #147aff;
}

.reset-password-dialog__error {
  margin: 0 0 12px;
  color: #e54545;
  font-size: 13px;
}

.reset-password-dialog__actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 20px;
}

.btn {
  min-width: 96px;
  border: none;
  border-radius: 999px;
  padding: 10px 18px;
  font-size: 14px;
  cursor: pointer;
}

.btn-cancel {
  background: #f2f4f8;
  color: #333;
}

.btn-primary {
  background: #147aff;
  color: #fff;
}

.btn-primary:disabled {
  background: #a8cfff;
  cursor: not-allowed;
}
</style>
