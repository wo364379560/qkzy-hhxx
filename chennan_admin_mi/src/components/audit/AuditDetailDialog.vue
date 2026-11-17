<script setup lang="ts">
export interface AuditDetailItem {
  label: string
  value: string
  code?: boolean
  multiline?: boolean
  onClick?: () => void
}

const props = defineProps<{
  visible: boolean
  title: string
  items: AuditDetailItem[]
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
}>()

function handleClose() {
  emit('update:visible', false)
}
</script>

<template>
  <el-dialog
    :model-value="visible"
    :title="title"
    width="720px"
    :close-on-click-modal="false"
    @close="handleClose"
  >
    <div class="detail-grid">
      <div
        v-for="item in items"
        :key="item.label"
        class="detail-item"
        :class="{ 'detail-item-full': item.multiline }"
      >
        <div class="detail-label">{{ item.label }}</div>
        <div
          class="detail-value"
          :class="{ 'detail-code': item.code, 'detail-multiline': item.multiline }"
        >
          <button
            v-if="item.onClick"
            type="button"
            class="detail-link"
            @click="item.onClick"
          >
            {{ item.value || '—' }}
          </button>
          <template v-else>
            {{ item.value || '—' }}
          </template>
        </div>
      </div>
    </div>

    <template #footer>
      <el-button @click="handleClose">关闭</el-button>
    </template>
  </el-dialog>
</template>

<style scoped>
.detail-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px 16px;
}

.detail-item {
  padding: 14px 16px;
  border: 1px solid #eef2f7;
  border-radius: 10px;
  background: #fafcff;
}

.detail-item-full {
  grid-column: 1 / -1;
}

.detail-label {
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  color: #94a3b8;
}

.detail-value {
  margin-top: 8px;
  font-size: 13px;
  line-height: 1.6;
  color: #1e293b;
  word-break: break-word;
}

.detail-code {
  font-family: 'Fira Code', 'Courier New', monospace;
}

.detail-multiline {
  white-space: pre-wrap;
}

.detail-link {
  padding: 0;
  border: none;
  background: transparent;
  color: #1677ff;
  font: inherit;
  cursor: pointer;
}

.detail-link:hover {
  color: #409eff;
  text-decoration: underline;
}
</style>
