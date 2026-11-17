<script setup lang="ts">
import { computed } from 'vue'
import type { IMHistoryMessage } from '@/types'
import { useResponsive } from '@/composables/useResponsive'

const props = defineProps<{
  visible: boolean
  title: string
  loading: boolean
  messages: IMHistoryMessage[]
  canLoadMore?: boolean
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  'load-more': []
}>()
const { isMobile } = useResponsive()

const dialogVisible = computed({
  get: () => props.visible,
  set: (value: boolean) => emit('update:visible', value),
})

function formatTime(value: number) {
  if (!value) return '-'
  return new Date(value * 1000).toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
}
</script>

<template>
  <el-dialog
    v-model="dialogVisible"
    :width="isMobile ? '100%' : '760px'"
    :fullscreen="isMobile"
    :close-on-click-modal="false"
  >
    <template #header>
      <div class="dialog-header">
        <span class="dialog-title">{{ props.title }}</span>
      </div>
    </template>

    <div v-loading="props.loading" class="dialog-body">
      <div v-if="!props.loading && props.messages.length === 0" class="dialog-empty">
        暂无可展示的消息记录
      </div>

      <div v-else class="table-scroll-x dialog-table-wrap">
        <el-table :data="props.messages" stripe size="small" style="width: 100%">
        <el-table-column label="时间" width="180">
          <template #default="{ row }">
            <span class="font-code cell-xs">{{ formatTime(row.msgTime) }}</span>
          </template>
        </el-table-column>

        <el-table-column v-if="!isMobile" label="发送方" width="150">
          <template #default="{ row }">
            <span class="font-code cell-muted">{{ row.fromUserID }}</span>
          </template>
        </el-table-column>

        <el-table-column label="内容" min-width="260">
          <template #default="{ row }">
            <span v-if="row.isRecalled" class="cell-recalled">[ 消息已撤回 ]</span>
            <span v-else class="cell-content">{{ row.content }}</span>
          </template>
        </el-table-column>
        </el-table>
      </div>
    </div>

    <template #footer>
      <div class="dialog-footer">
        <el-button v-if="props.canLoadMore" plain @click="emit('load-more')">加载更多</el-button>
        <el-button @click="dialogVisible = false">关闭</el-button>
      </div>
    </template>
  </el-dialog>
</template>

<style scoped>
.dialog-body {
  min-height: 180px;
}

.dialog-table-wrap {
  border-bottom: 1px solid #f0f0f0;
}

.dialog-header {
  display: flex;
  align-items: center;
  gap: 8px;
}

.dialog-title {
  font-size: 15px;
  font-weight: 600;
  color: #1e293b;
}

.dialog-empty {
  padding: 48px 20px;
  text-align: center;
  color: #94a3b8;
  font-size: 13px;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

@media (max-width: 767px) {
  .dialog-footer {
    justify-content: stretch;
    flex-direction: column;
  }

  .dialog-footer :deep(.el-button) {
    width: 100%;
    margin-left: 0;
  }
}

.cell-muted {
  font-size: 12px;
  color: #64748b;
}

.cell-xs {
  font-size: 11px;
  color: #94a3b8;
}

.cell-content {
  font-size: 13px;
  color: #1e293b;
}

.cell-recalled {
  font-size: 12px;
  color: #94a3b8;
  font-style: italic;
}
</style>
