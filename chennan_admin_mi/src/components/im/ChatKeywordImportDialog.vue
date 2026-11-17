<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { reactive, watch } from 'vue'
import { useResponsive } from '@/composables/useResponsive'

const props = defineProps<{
  visible: boolean
  loading?: boolean
}>()
const { isMobile } = useResponsive()

const emit = defineEmits<{
  (event: 'update:visible', value: boolean): void
  (event: 'submit', payload: { content: string; enabled: boolean }): void
}>()

const form = reactive({
  content: '',
  enabled: true,
})

const importTemplate = `2
广告,禁止发送
测试词,***`

watch(
  () => props.visible,
  (visible) => {
    if (!visible) return
    form.content = ''
    form.enabled = true
  }
)

function closeDialog() {
  emit('update:visible', false)
}

function submit() {
  emit('submit', {
    content: form.content,
    enabled: form.enabled,
  })
}

async function copyTemplate() {
  try {
    await navigator.clipboard.writeText(importTemplate)
    ElMessage.success('导入模板已复制')
  } catch {
    ElMessage.error('复制模板失败')
  }
}
</script>

<template>
  <el-dialog
    :model-value="visible"
    title="批量导入关键词"
    :width="isMobile ? '100%' : '560px'"
    :fullscreen="isMobile"
    destroy-on-close
    @close="closeDialog"
  >
    <div class="dialog-body">
      <div class="dialog-toolbar">
        <el-button text type="primary" @click="copyTemplate">复制导入模板</el-button>
      </div>
      <el-form label-position="top">
        <el-form-item label="导入内容">
          <el-input
            v-model="form.content"
            type="textarea"
            :rows="10"
            resize="none"
            placeholder="一行一个关键词，或使用“关键词,替换文本”格式&#10;例如：&#10;2&#10;测试词,已屏蔽&#10;广告，禁止发送"
          />
        </el-form-item>
        <el-form-item label="导入后状态">
          <el-switch v-model="form.enabled" inline-prompt active-text="启用" inactive-text="停用" />
        </el-form-item>
      </el-form>

      <div class="dialog-tip">
        <div>支持格式：</div>
        <div>1. <span class="tip-code">关键词</span>，替换文本默认使用 <span class="tip-code">**</span></div>
        <div>2. <span class="tip-code">关键词,替换文本</span>，也支持中文逗号</div>
        <div>3. 若关键词已存在，则会自动更新其替换文本和状态</div>
      </div>
    </div>

    <template #footer>
      <div class="dialog-footer">
        <el-button @click="closeDialog">取消</el-button>
        <el-button type="primary" :loading="loading" @click="submit">开始导入</el-button>
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

.dialog-toolbar {
  display: flex;
  justify-content: flex-end;
}

.dialog-tip {
  padding: 12px;
  border-radius: 12px;
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  font-size: 12px;
  color: #64748b;
  display: grid;
  gap: 6px;
}

.tip-code {
  font-family: 'Fira Code', 'Courier New', monospace;
  color: #ef4444;
  font-weight: 600;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
