<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { searchIMUsers } from '@/api/im'
import type { IMUserOption } from '@/types'

type ModelValue = IMUserOption | IMUserOption[] | null

const props = withDefaults(defineProps<{
  label: string
  placeholder: string
  modelValue: ModelValue
  multiple?: boolean
  disabled?: boolean
  minKeywordLength?: number
}>(), {
  multiple: false,
  disabled: false,
  minKeywordLength: 1,
})

const emit = defineEmits<{
  'update:modelValue': [value: ModelValue]
}>()

const rootRef = ref<HTMLElement | null>(null)
const inputRef = ref<HTMLInputElement | null>(null)
const loading = ref(false)
const isOpen = ref(false)
const queryText = ref('')
const remoteOptions = ref<IMUserOption[]>([])
const hasLoadedDefaultOptions = ref(false)

let searchTimer: ReturnType<typeof setTimeout> | null = null
let requestToken = 0

const selectedOptions = computed<IMUserOption[]>(() => {
  if (Array.isArray(props.modelValue)) return props.modelValue
  return props.modelValue ? [props.modelValue] : []
})

const selectedSingleUser = computed(() => {
  if (Array.isArray(props.modelValue)) return null
  return props.modelValue
})

const inputDisplayValue = computed(() => {
  if (props.multiple) return queryText.value
  if (isOpen.value) return queryText.value
  return queryText.value || selectedSingleUser.value?.nickname || ''
})

const displayOptions = computed(() => {
  const keyword = queryText.value.trim()
  if (!keyword) return remoteOptions.value
  return remoteOptions.value.filter(option => option.nickname.includes(keyword))
})

const selectedCountText = computed(() => {
  if (!props.multiple || selectedOptions.value.length === 0) return ''
  return `已选择 ${selectedOptions.value.length} 个用户`
})

watch(
  () => props.modelValue,
  (value) => {
    if (Array.isArray(value)) {
      queryText.value = ''
      return
    }

    if (!isOpen.value) {
      queryText.value = value?.nickname ?? ''
    }
  },
  { immediate: true, deep: true },
)

function isSelected(option: IMUserOption) {
  return selectedOptions.value.some(item => item.imUserID === option.imUserID)
}

function openPanel() {
  if (props.disabled) return
  isOpen.value = true

  if (props.multiple) {
    if (!hasLoadedDefaultOptions.value) {
      void runSearch('')
    }
    return
  }

  queryText.value = ''
  void runSearch('')
}

function closePanel() {
  isOpen.value = false
  if (!props.multiple) {
    queryText.value = selectedSingleUser.value?.nickname ?? ''
  }
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
    const res = await searchIMUsers({ keyword: normalizedKeyword })
    if (currentToken !== requestToken) return
    remoteOptions.value = res.data.data ?? []
    if (!normalizedKeyword) {
      hasLoadedDefaultOptions.value = true
    }
  } catch (error: any) {
    if (currentToken !== requestToken) return
    remoteOptions.value = []
    if (normalizedKeyword) {
      ElMessage.error(error.message || '搜索用户失败')
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

function handleSelect(option: IMUserOption) {
  if (props.multiple) {
    const current = Array.isArray(props.modelValue) ? props.modelValue : []
    if (isSelected(option)) {
      emit('update:modelValue', current.filter(item => item.imUserID !== option.imUserID))
    } else {
      emit('update:modelValue', [...current, option])
    }
    queryText.value = ''
    void runSearch('')
    inputRef.value?.focus()
    return
  }

  emit('update:modelValue', option)
  queryText.value = option.nickname
  closePanel()
}

function removeSelected(option: IMUserOption) {
  if (props.multiple) {
    const current = Array.isArray(props.modelValue) ? props.modelValue : []
    emit('update:modelValue', current.filter(item => item.imUserID !== option.imUserID))
    return
  }

  emit('update:modelValue', null)
  queryText.value = ''
  isOpen.value = false
}

function clearSelection() {
  emit('update:modelValue', props.multiple ? [] : null)
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
      <p v-if="selectedSingleUser" class="selected-meta">
        <span class="selected-label">{{ selectedSingleUser.nickname }}</span>
        <span class="meta-divider">|</span>
        <span>{{ selectedSingleUser.phone || '无手机号' }}</span>
        <span class="meta-divider">|</span>
        <span class="font-code">{{ selectedSingleUser.imUserID }}</span>
      </p>

      <p v-else-if="selectedCountText" class="selected-meta">
        {{ selectedCountText }}
      </p>
    </div>

    <div class="search-box" :class="{ 'search-box--open': isOpen, 'search-box--multiple': props.multiple }" @click="openPanel">
      <div v-if="props.multiple && selectedOptions.length > 0" class="selected-chips">
        <span
          v-for="option in selectedOptions"
          :key="option.imUserID"
          class="selected-chip"
        >
          {{ option.nickname }}
          <button class="chip-remove" @click.stop="removeSelected(option)">×</button>
        </span>
      </div>

      <input
        ref="inputRef"
        class="search-input"
        :disabled="props.disabled"
        :placeholder="props.placeholder"
        :value="inputDisplayValue"
        @focus="openPanel"
        @input="handleInput"
      />

      <button v-if="selectedOptions.length > 0 || queryText" class="clear-btn" @click.stop="clearSelection">×</button>
    </div>

    <div v-if="isOpen" class="search-panel">
      <div v-if="loading" class="search-state">正在按昵称模糊搜索用户…</div>
      <template v-else>
        <div v-if="displayOptions.length > 0" class="option-list">
          <button
            v-for="option in displayOptions"
            :key="option.imUserID"
            class="option-item"
            :class="{ 'option-item--selected': isSelected(option) }"
            @click="handleSelect(option)"
          >
            <span class="option-title">{{ option.nickname }}</span>
            <span class="option-pill option-pill-id option-cell-id font-code">{{ option.imUserID }}</span>
            <span class="option-pill option-pill-phone option-cell-phone">{{ option.phone || '无手机号' }}</span>
          </button>
        </div>

        <div v-else-if="queryText.trim()" class="search-state search-state-empty">
          未找到匹配昵称，请尝试输入更多昵称片段
        </div>

        <div v-else class="search-state">
          暂无待选昵称，可直接输入昵称片段进行模糊搜索
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

.search-box--multiple {
  min-height: 36px;
  height: auto;
  padding: 4px 10px;
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

.selected-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.selected-chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  height: 26px;
  padding: 0 8px;
  border-radius: 999px;
  background: rgba(64, 158, 255, 0.12);
  color: #1677ff;
  font-size: 12px;
  font-weight: 500;
}

.chip-remove {
  border: none;
  background: none;
  color: rgba(22, 119, 255, 0.6);
  cursor: pointer;
  font-size: 14px;
  line-height: 1;
  padding: 0;
}

.chip-remove:hover {
  color: #1677ff;
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
  grid-template-columns: minmax(0, 1fr) 140px 88px;
  align-items: center;
  column-gap: 6px;
  width: 100%;
  padding: 7px 10px;
  cursor: pointer;
  text-align: left;
}

.option-item:hover,
.option-item--selected {
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

.option-pill-phone {
  color: #64748b;
  background: #f5f7fa;
}

.option-cell-id,
.option-cell-phone {
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
  .option-pill-phone {
    width: fit-content;
  }
}
</style>
