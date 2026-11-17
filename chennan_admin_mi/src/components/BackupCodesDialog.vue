<script setup lang="ts">
defineProps<{
  visible: boolean
  codes: string[]
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
}>()
</script>

<template>
  <el-dialog
    :model-value="visible"
    title="备份码"
    width="500px"
    :close-on-click-modal="false"
    :close-on-press-escape="false"
    :show-close="false"
    @close="emit('update:visible', false)"
  >
    <p style="color: #ef4444; margin-bottom: 16px">⚠️ 请妥善保存这些备份码，每个备份码仅可使用一次</p>
    <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 12px; background: #f8fafc; padding: 16px; border-radius: 8px">
      <code v-for="(code, i) in codes" :key="i" style="background: white; padding: 8px; border-radius: 4px; text-align: center; font-family: 'Fira Code', monospace">{{ code }}</code>
    </div>
    <template #footer>
      <el-button type="primary" @click="emit('update:visible', false)">我已保存</el-button>
    </template>
  </el-dialog>
</template>
