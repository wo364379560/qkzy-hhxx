<script setup lang="ts">
import { computed } from 'vue';
import { Link } from '@/utils/link';

const props = defineProps<{
  modelValue: boolean;
}>();

const emit = defineEmits<{
  (event: 'update:modelValue', value: boolean): void;
}>();

const checked = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value),
});
</script>

<template>
  <label class="privacy-field">
    <input
      v-model="checked"
      class="privacy-field-checkbox"
      type="checkbox"
    >
    <span class="privacy-field-text">
      我已阅读并同意
      <a
        :href="Link.agreement.url"
        target="_blank"
        rel="noreferrer"
      >
        《用户服务协议》
      </a>
      和
      <a
        :href="Link.privacy.url"
        target="_blank"
        rel="noreferrer"
      >
        《隐私政策》
      </a>
    </span>
  </label>
</template>

<style scoped lang="scss">
.privacy-field {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  color: #64748b;
  font-size: 13px;
  line-height: 1.5;
}

.privacy-field-checkbox {
  margin-top: 2px;
}

.privacy-field-text a {
  color: #4361ee;
  text-decoration: none;
}
</style>
