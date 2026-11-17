<script setup lang="ts">
import { computed } from 'vue'
import type { BatchImportChatKeywordsResult } from '@/types'
import { useResponsive } from '@/composables/useResponsive'

const props = defineProps<{
  visible: boolean
  result: BatchImportChatKeywordsResult | null
}>()
const { isMobile } = useResponsive()

const emit = defineEmits<{
  (event: 'update:visible', value: boolean): void
}>()

const rows = computed(() => props.result?.results ?? [])

function closeDialog() {
  emit('update:visible', false)
}

function actionLabel(action: string) {
  switch (action) {
    case 'created':
      return '新增'
    case 'updated':
      return '更新'
    case 'skipped':
      return '跳过'
    case 'error':
      return '异常'
    default:
      return action
  }
}

function actionType(action: string) {
  switch (action) {
    case 'created':
      return 'success'
    case 'updated':
      return 'primary'
    case 'skipped':
      return 'info'
    case 'error':
      return 'danger'
    default:
      return 'info'
  }
}
</script>

<template>
  <el-dialog
    :model-value="visible"
    title="导入结果"
    :width="isMobile ? '100%' : '760px'"
    :fullscreen="isMobile"
    destroy-on-close
    @close="closeDialog"
  >
    <div class="result-shell">
      <div class="summary-grid">
        <div class="summary-card summary-created">
          <span class="summary-label">新增</span>
          <strong class="summary-value">{{ result?.created ?? 0 }}</strong>
        </div>
        <div class="summary-card summary-updated">
          <span class="summary-label">更新</span>
          <strong class="summary-value">{{ result?.updated ?? 0 }}</strong>
        </div>
        <div class="summary-card summary-skipped">
          <span class="summary-label">跳过</span>
          <strong class="summary-value">{{ result?.skipped ?? 0 }}</strong>
        </div>
      </div>

      <div class="table-scroll-x result-table-wrap">
        <el-table :data="rows" stripe size="small" max-height="420" style="width: 100%">
        <el-table-column label="行号" width="72" align="center">
          <template #default="{ row }">
            <span class="cell-line">{{ row.line || '—' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="关键词" min-width="160">
          <template #default="{ row }">
            <span class="cell-code">{{ row.keyword || '—' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="替换文本" min-width="160">
          <template #default="{ row }">
            <span class="cell-code">{{ row.replacement || '—' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="结果" width="90" align="center">
          <template #default="{ row }">
            <el-tag :type="actionType(row.action)" effect="light" round>
              {{ actionLabel(row.action) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="说明" min-width="200">
          <template #default="{ row }">
            <span class="cell-message">{{ row.message }}</span>
          </template>
        </el-table-column>
        </el-table>
      </div>
    </div>

    <template #footer>
      <div class="dialog-footer">
        <el-button type="primary" @click="closeDialog">知道了</el-button>
      </div>
    </template>
  </el-dialog>
</template>

<style scoped>
.result-shell {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.result-table-wrap {
  border-radius: 12px;
}

.summary-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
}

.summary-card {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 14px 16px;
  border-radius: 14px;
  border: 1px solid #e2e8f0;
  background: #f8fafc;
}

.summary-created {
  background: #f0fdf4;
  border-color: #bbf7d0;
}

.summary-updated {
  background: #eff6ff;
  border-color: #bfdbfe;
}

.summary-skipped {
  background: #f8fafc;
  border-color: #cbd5e1;
}

.summary-label {
  font-size: 12px;
  color: #64748b;
}

.summary-value {
  font-size: 26px;
  line-height: 1;
  color: #0f172a;
}

.cell-line {
  color: #64748b;
  font-size: 12px;
}

.cell-code {
  font-family: 'Fira Code', 'Courier New', monospace;
  font-size: 12px;
  color: #0f172a;
}

.cell-message {
  font-size: 12px;
  color: #475569;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
}

@media (max-width: 767px) {
  .summary-grid {
    grid-template-columns: 1fr;
  }

  .dialog-footer {
    justify-content: stretch;
  }

  .dialog-footer :deep(.el-button) {
    width: 100%;
  }
}
</style>
