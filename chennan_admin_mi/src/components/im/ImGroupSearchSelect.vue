<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { searchIMGroups } from '@/api/im'
import type { IMGroupOption } from '@/types'

const props = withDefaults(defineProps<{
  label: string
  placeholder: string
  modelValue: IMGroupOption | null
  disabled?: boolean
  minKeywordLength?: number
}>(), {
  disabled: false,
  minKeywordLength: 1,
})

const emit = defineEmits<{
  'update:modelValue': [value: IMGroupOption | null]
}>()

const rootRef = ref<HTMLElement | null>(null)
const inputRef = ref<HTMLInputElement | null>(null)
const loading = ref(false)
const isOpen = ref(false)
const queryText = ref('')
const remoteOptions = ref<IMGroupOption[]>([])
const hasLoadedDefaultOptions = ref(false)

let searchTimer: ReturnType<typeof setTimeout> | null = null
let requestToken = 0

const displayOptions = computed(() => {
  const keyword = queryText.value.trim()
  if (!keyword) return remoteOptions.value
  return remoteOptions.value.filter(option => option.name.includes(keyword))
})

watch(
  () => props.modelValue,
  (value) => {
    if (!isOpen.value) {
      queryText.value = value?.name ?? ''
    }
  },
  { immediate: true, deep: true },
)

function openPanel() {
  if (props.disabled) return
  isOpen.value = true
  queryText.value = ''
  void runSearch('')
}

function closePanel() {
  isOpen.value = false
  queryText.value = props.modelValue?.name ?? ''
}

function scheduleSearch(keyword: string) {
  queryText.value = keyword
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    void runSearch(keyword)
  }, 180)
}

async function runSearch(keyword: string) {
  const normalizedKeyword = keyword.trim()
  if (props.disabled) {
    remoteOptions.value = []
    return
  }
  if (normalizedKeyword.length > 0 && normalizedKeyword.length < props.minKeywordLength) {
    remoteOptions.value = []
    return
  }

  const currentToken = ++requestToken
  remoteOptions.value = []
  loading.value = true
  try {
    const res = await searchIMGroups({ keyword: normalizedKeyword })
    if (currentToken !== requestToken) return
    remoteOptions.value = res.data.data ?? []
    if (!normalizedKeyword) {
      hasLoadedDefaultOptions.value = true
    }
  } catch (error: any) {
    if (currentToken !== requestToken) return
    remoteOptions.value = []
    if (normalizedKeyword) {
      ElMessage.error(error.message || '搜索群组失败')
    }
  } finally {
    if (currentToken === requestToken) {
      loading.value = false
    }
  }
}

function handleInput(event: Event) {
  const value = (event.target as HTMLInputElement).value
  if (!isOpen.value) {
    isOpen.value = true
  }
  scheduleSearch(value)
}

function handleSelect(option: IMGroupOption) {
  emit('update:modelValue', option)
  queryText.value = option.name
  closePanel()
}

function clearSelection() {
  emit('update:modelValue', null)
  remoteOptions.value = []
  queryText.value = ''
  hasLoadedDefaultOptions.value = false
}

function handleDocumentMousedown(event: MouseEvent) {
  if (!rootRef.value) return
  if (!rootRef.value.contains(event.target as Node)) {
    closePanel()
  }
}

onMounted(() => {
  document.addEventListener('mousedown', handleDocumentMousedown)
})

onUnmounted(() => {
  document.removeEventListener('mousedown', handleDocumentMousedown)
  if (searchTimer) clearTimeout(searchTimer)
})
</script>

<template>
  <div ref="rootRef" class="search-select" :class="{ 'search-select--disabled': props.disabled }">
    <div class="field-header">
      <label class="form-label">{{ props.label }}</label>
      <p v-if="props.modelValue" class="selected-meta">
        <span class="selected-label">{{ props.modelValue.name }}</span>
        <span class="meta-divider">|</span>
        <span>{{ props.modelValue.groupType || '—' }}</span>
        <span class="meta-divider">|</span>
        <span class="font-code">{{ props.modelValue.imGroupId }}</span>
      </p>
    </div>

    <div class="search-box" :class="{ 'search-box--open': isOpen }" @click="openPanel">
      <input
        ref="inputRef"
        class="search-input"
        :disabled="props.disabled"
        :placeholder="props.placeholder"
        :value="isOpen ? queryText : (queryText || props.modelValue?.name || '')"
        @focus="openPanel"
        @input="handleInput"
      />

      <button v-if="props.modelValue || queryText" class="clear-btn" @click.stop="clearSelection">×</button>
    </div>

    <div v-if="isOpen" class="search-panel">
      <div v-if="loading" class="search-state">正在按群组名称模糊搜索群组…</div>
      <template v-else>
        <div v-if="displayOptions.length > 0" class="option-list">
          <button
            v-for="option in displayOptions"
            :key="option.imGroupId"
            class="option-item"
            @click="handleSelect(option)"
          >
            <span class="option-title">{{ option.name }}</span>
            <span class="option-pill option-pill-id option-cell-id font-code">{{ option.imGroupId }}</span>
            <span class="option-pill option-pill-type option-cell-type">{{ option.groupType || '—' }}</span>
          </button>
        </div>

        <div v-else-if="queryText.trim()" class="search-state search-state-empty">
          未找到匹配群组，请尝试输入更多群组名称片段
        </div>

        <div v-else class="search-state">
          暂无待选群组，可直接输入群组名称片段进行模糊搜索
        </div>
      </template>
    </div>
  </div>
</template>

<style scoped>
.search-select {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.field-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.search-select--disabled {
  opacity: 0.6;
}

.form-label {
  margin: 0;
  font-size: 12px;
  font-weight: 500;
  color: #475569;
}

.search-box {
  height: 36px;
  border: 1px solid #dcdfe6;
  border-radius: 8px;
  background: #fff;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 10px;
  box-sizing: border-box;
  transition: border-color 0.15s, box-shadow 0.15s;
}

.search-box--open {
  border-color: #409eff;
  box-shadow: 0 0 0 2px rgba(64, 158, 255, 0.12);
}

.search-input {
  flex: 1;
  height: 100%;
  border: none;
  outline: none;
  font-size: 14px;
  line-height: 20px;
  color: #1e293b;
  background: transparent;
  min-width: 0;
}

.search-input::placeholder {
  color: #94a3b8;
}

.clear-btn {
  border: none;
  background: none;
  color: #94a3b8;
  cursor: pointer;
  font-size: 18px;
  line-height: 1;
  padding: 0 4px;
}

.clear-btn:hover {
  color: #64748b;
}

.search-panel {
  position: absolute;
  top: calc(100% + 6px);
  left: 0;
  right: 0;
  max-height: 280px;
  overflow-y: auto;
  border: 1px solid #e5e7eb;
  border-radius: 10px;
  background: #fff;
  box-shadow: 0 10px 24px rgba(15, 23, 42, 0.12);
  z-index: 30;
}

.option-list {
  display: flex;
  flex-direction: column;
}

.option-item {
  border: none;
  background: #fff;
  display: grid;
  grid-template-columns: minmax(0, 1fr) 160px 84px;
  align-items: center;
  column-gap: 6px;
  width: 100%;
  padding: 7px 10px;
  cursor: pointer;
  text-align: left;
}

.option-item:hover {
  background: #f5f9ff;
}

.option-title {
  font-size: 12px;
  font-weight: 500;
  color: #1e293b;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  min-width: 0;
}

.option-pill {
  display: inline-flex;
  align-items: center;
  justify-content: flex-start;
  height: 18px;
  padding: 0 4px;
  border-radius: 6px;
  font-size: 10px;
  line-height: 1;
  white-space: nowrap;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
}

.option-pill-id {
  color: #1677ff;
  background: rgba(64, 158, 255, 0.12);
}

.option-pill-type {
  color: #64748b;
  background: #f5f7fa;
}

.option-cell-id,
.option-cell-type {
  justify-self: start;
  width: auto;
}

.search-state {
  padding: 14px 12px;
  font-size: 12px;
  color: #94a3b8;
}

.search-state-empty {
  color: #d46b08;
}

.selected-meta {
  margin: 0;
  font-size: 12px;
  color: #94a3b8;
  flex-shrink: 0;
  white-space: nowrap;
}

.selected-label {
  color: #475569;
  font-weight: 500;
}

.meta-divider {
  margin: 0 6px;
}

@media (max-width: 767px) {
  .field-header {
    flex-direction: column;
    align-items: flex-start;
  }

  .selected-meta {
    width: 100%;
    white-space: normal;
    line-height: 1.5;
  }

  .option-item {
    grid-template-columns: 1fr;
    row-gap: 6px;
  }

  .option-pill-id,
  .option-pill-type {
    width: fit-content;
  }
}
</style>
