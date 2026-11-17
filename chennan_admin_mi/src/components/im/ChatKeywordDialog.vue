<script setup lang="ts">
import { computed, reactive, watch } from 'vue'
import type { ChatKeyword } from '@/types'
import { useResponsive } from '@/composables/useResponsive'

const props = defineProps<{
  visible: boolean
  mode: 'create' | 'edit'
  keyword?: ChatKeyword | null
  loading?: boolean
}>()
const { isMobile } = useResponsive()

const emit = defineEmits<{
  (event: 'update:visible', value: boolean): void
  (event: 'submit', payload: { keyword: string; replacement: string; enabled: boolean }): void
}>()

const form = reactive({
  keyword: '',
  replacement: '**',
  enabled: true,
})

const dialogTitle = computed(() => props.mode === 'create' ? '新增关键词' : '编辑关键词')

watch(
  () => [props.visible, props.keyword, props.mode] as const,
  ([visible, keyword, mode]) => {
    if (!visible) return
    form.keyword = keyword?.keyword ?? ''
    form.replacement = keyword?.replacement ?? '**'
    form.enabled = mode === 'create' ? true : (keyword?.enabled ?? true)
  },
  { immediate: true }
)

function closeDialog() {
  emit('update:visible', false)
}

function submit() {
  emit('submit', {
    keyword: form.keyword.trim(),
    replacement: form.replacement.trim(),
    enabled: form.enabled,
  })
}
</script>

<template>
  <el-dialog
    :model-value="visible"
    :title="dialogTitle"
    :width="isMobile ? '100%' : '420px'"
    :fullscreen="isMobile"
    destroy-on-close
    @close="closeDialog"
  >
    <div class="dialog-body">
      <el-form label-position="top">
        <el-form-item label="关键词">
          <el-input
            v-model="form.keyword"
            maxlength="64"
            placeholder="例如：2"
            clearable
            @keyup.enter="submit"
          />
        </el-form-item>
        <el-form-item label="替换文本">
          <el-input
            v-model="form.replacement"
            maxlength="64"
            placeholder="默认 **"
            clearable
            @keyup.enter="submit"
          />
        </el-form-item>
        <el-form-item label="状态">
          <el-switch v-model="form.enabled" inline-prompt active-text="启用" inactive-text="停用" />
        </el-form-item>
      </el-form>
      <div class="dialog-tip">
        命中关键词后，文本消息中的对应内容会被替换为你设置的文本；若留空，则默认使用 <span class="tip-mask">**</span>。
      </div>
    </div>

    <template #footer>
      <div class="dialog-footer">
        <el-button @click="closeDialog">取消</el-button>
        <el-button type="primary" :loading="loading" @click="submit">
          {{ mode === 'create' ? '创建' : '保存' }}
        </el-button>
      </div>
    </template>
  </el-dialog>
</template>

<style scoped>
.dialog-body {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.dialog-tip {
  padding: 10px 12px;
  border-radius: 10px;
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  font-size: 12px;
  color: #64748b;
}

.tip-mask {
  font-family: 'Fira Code', 'Courier New', monospace;
  color: #ef4444;
  font-weight: 700;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
