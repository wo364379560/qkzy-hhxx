<template>
  <Dialog
    :show="true"
    :title="TUITranslateService.t('Profile.邀请码')"
    :isFooterShow="false"
    :center="true"
    @update:show="closeDialog"
  >
    <div class="invite-code-dialog">
      <p class="invite-code-dialog__desc">
        {{
          isLocked
            ? TUITranslateService.t("Profile.当前账号已绑定邀请码后不可再修改")
            : TUITranslateService.t("Profile.没有邀请码的用户可绑定一次邀请码切换归属")
        }}
      </p>
      <label class="invite-code-dialog__field">
        <span class="invite-code-dialog__label">
          {{ TUITranslateService.t("Profile.邀请码") }}
        </span>
        <input
          v-model.trim="inviteCode"
          :disabled="isLocked || submitting"
          type="text"
          autocomplete="off"
          :placeholder="TUITranslateService.t('Profile.请输入邀请码')"
        />
      </label>
      <div class="invite-code-dialog__note">
        <template v-if="isLocked">
          {{
            TUITranslateService.t("Profile.当前邀请码") +
            "：" +
            (profile?.invite_code || TUITranslateService.t("Profile.暂未设置"))
          }}
        </template>
        <template v-else>
          {{ TUITranslateService.t("Profile.绑定成功后将立即切换归属且后续不能再次修改") }}
        </template>
      </div>
      <p
        v-if="errorMessage"
        class="invite-code-dialog__error"
      >
        {{ errorMessage }}
      </p>
      <div class="invite-code-dialog__actions">
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
          :disabled="isLocked || submitting"
          @click="submit"
        >
          {{
            isLocked
              ? TUITranslateService.t("Profile.已绑定不可修改")
              : submitting
                ? TUITranslateService.t("Profile.提交中")
                : TUITranslateService.t("Profile.确认绑定")
          }}
        </button>
      </div>
    </div>
  </Dialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { TUITranslateService } from "@tencentcloud/chat-uikit-engine-lite";
import Dialog from "@/TUIKit/components/common/Dialog/index.vue";
import { bindInviteCode } from "@/api";
import type { ChatProfile } from "@/features/auth/types";
import { useSingleFlight } from "@/composables/useSingleFlight";

const props = defineProps<{
  profile: ChatProfile | null;
}>();

const emit = defineEmits<{
  (e: "close"): void;
  (e: "updated", profile: ChatProfile): void;
}>();

const inviteCode = ref("");
const submitFlight = useSingleFlight();
const submitting = submitFlight.running;
const errorMessage = ref("");

const isLocked = computed(() => {
  if (!props.profile) {
    return false;
  }
  return Boolean(props.profile.invite_code) || !props.profile.can_bind_invite_code;
});

watch(
  () => props.profile,
  (profile) => {
    inviteCode.value = profile?.invite_code || "";
  },
  { immediate: true }
);

function closeDialog() {
  emit("close");
}

async function submit() {
  if (isLocked.value) {
    return;
  }
  if (!inviteCode.value) {
    errorMessage.value = TUITranslateService.t("Profile.请输入邀请码");
    return;
  }

  errorMessage.value = "";
  try {
    await submitFlight.run(async () => {
      const profile = await bindInviteCode(inviteCode.value);
      emit("updated", profile);
      emit("close");
    });
  } catch (error) {
    errorMessage.value =
      error instanceof Error
        ? error.message
        : TUITranslateService.t("Profile.绑定邀请码失败");
  }
}
</script>

<style scoped lang="scss">
.invite-code-dialog {
  width: min(420px, 80vw);
  padding: 8px 8px 4px;
}

.invite-code-dialog__desc {
  margin: 0 0 16px;
  color: #666;
  font-size: 14px;
  line-height: 1.6;
}

.invite-code-dialog__field {
  display: block;
  margin-bottom: 14px;
}

.invite-code-dialog__label {
  display: block;
  margin-bottom: 6px;
  color: #222;
  font-size: 13px;
  font-weight: 600;
}

.invite-code-dialog__field input {
  width: 100%;
  border: 1px solid #d8dde8;
  border-radius: 10px;
  padding: 10px 12px;
  font-size: 14px;
  outline: none;
  box-sizing: border-box;
}

.invite-code-dialog__field input:focus {
  border-color: #00b894;
}

.invite-code-dialog__field input:disabled {
  background: #f5f6f8;
  color: #888;
  cursor: not-allowed;
}

.invite-code-dialog__note {
  border-radius: 10px;
  background: #f5f7fb;
  padding: 12px 14px;
  color: #666;
  font-size: 13px;
  line-height: 1.6;
}

.invite-code-dialog__error {
  margin: 12px 0 0;
  color: #e54545;
  font-size: 13px;
}

.invite-code-dialog__actions {
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
  background: #00b894;
  color: #fff;
}

.btn-primary:disabled {
  background: #a8ded0;
  cursor: not-allowed;
}
</style>
