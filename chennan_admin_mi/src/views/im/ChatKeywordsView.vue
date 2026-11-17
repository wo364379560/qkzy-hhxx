<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import ChatKeywordDialog from '@/components/im/ChatKeywordDialog.vue'
import ChatKeywordImportDialog from '@/components/im/ChatKeywordImportDialog.vue'
import ChatKeywordImportResultDialog from '@/components/im/ChatKeywordImportResultDialog.vue'
import {
  createChatKeyword,
  deleteChatKeyword,
  getChatKeywordList,
  importChatKeywords,
  updateChatKeyword,
} from '@/api/chatKeyword'
import type { BatchImportChatKeywordsResult, ChatKeyword } from '@/types'
import { useResponsive } from '@/composables/useResponsive'

const loading = ref(false)
const submitting = ref(false)
const rows = ref<ChatKeyword[]>([])
const total = ref(0)
const dialogVisible = ref(false)
const importDialogVisible = ref(false)
const importResultDialogVisible = ref(false)
const dialogMode = ref<'create' | 'edit'>('create')
const currentKeyword = ref<ChatKeyword | null>(null)
const latestImportResult = ref<BatchImportChatKeywordsResult | null>(null)

const filters = reactive({
  keyword: '',
})

const pagination = reactive({
  page: 1,
  pageSize: 20,
})

const pageSizeOptions = [10, 20, 50, 100]
const { isMobile } = useResponsive()

const summaryText = computed(() => `共 ${total.value} 条关键词规则`)

async function fetchRows() {
  loading.value = true
  try {
    const res = await getChatKeywordList({
      page: pagination.page,
      pageSize: pagination.pageSize,
      keyword: filters.keyword || undefined,
    })
    rows.value = res.data.data?.list ?? []
    total.value = res.data.data?.total ?? 0
  } catch (error: any) {
    ElMessage.error(error.message || '加载关键词规则失败')
  } finally {
    loading.value = false
  }
}

function handleSearch() {
  pagination.page = 1
  void fetchRows()
}

function resetFilters() {
  filters.keyword = ''
  handleSearch()
}

function openCreateDialog() {
  dialogMode.value = 'create'
  currentKeyword.value = null
  dialogVisible.value = true
}

function openImportDialog() {
  importDialogVisible.value = true
}

function openEditDialog(row: ChatKeyword) {
  dialogMode.value = 'edit'
  currentKeyword.value = row
  dialogVisible.value = true
}

async function handleDialogSubmit(payload: { keyword: string; replacement: string; enabled: boolean }) {
  if (!payload.keyword) {
    ElMessage.warning('请输入关键词')
    return
  }

  submitting.value = true
  try {
    if (dialogMode.value === 'create') {
      await createChatKeyword(payload)
      ElMessage.success('关键词已创建')
    } else if (currentKeyword.value) {
      await updateChatKeyword(currentKeyword.value.id, payload)
      ElMessage.success('关键词已更新')
    }

    dialogVisible.value = false
    await fetchRows()
  } catch (error: any) {
    ElMessage.error(error.message || '保存关键词失败')
  } finally {
    submitting.value = false
  }
}

async function handleToggle(row: ChatKeyword, enabled: boolean) {
  try {
    await updateChatKeyword(row.id, {
      keyword: row.keyword,
      replacement: row.replacement,
      enabled,
    })
    row.enabled = enabled
    ElMessage.success(enabled ? '已启用关键词' : '已停用关键词')
  } catch (error: any) {
    row.enabled = !enabled
    ElMessage.error(error.message || '更新状态失败')
  }
}

async function handleImportSubmit(payload: { content: string; enabled: boolean }) {
  if (!payload.content.trim()) {
    ElMessage.warning('请输入导入内容')
    return
  }

  submitting.value = true
  try {
    const res = await importChatKeywords(payload)
    const summary = res.data.data
    importDialogVisible.value = false
    await fetchRows()
    latestImportResult.value = summary
    importResultDialogVisible.value = true
  } catch (error: any) {
    ElMessage.error(error.message || '导入关键词失败')
  } finally {
    submitting.value = false
  }
}

async function handleDelete(row: ChatKeyword) {
  try {
    await ElMessageBox.confirm(
      `确认删除关键词 “${row.keyword}” 吗？删除后将立即停止屏蔽。`,
      '删除关键词',
      {
        type: 'warning',
        confirmButtonText: '删除',
        cancelButtonText: '取消',
      }
    )
  } catch {
    return
  }

  try {
    await deleteChatKeyword(row.id)
    ElMessage.success('关键词已删除')
    await fetchRows()
  } catch (error: any) {
    ElMessage.error(error.message || '删除关键词失败')
  }
}

onMounted(() => {
  void fetchRows()
})
</script>

<template>
  <div class="page">
    <div class="card">
      <div class="card-header">
        <div>
          <h1 class="page-title">关键词屏蔽</h1>
          <p class="page-sub">管理全局聊天关键词，支持自定义替换文本与批量导入，命中后会在消息下发前自动替换。</p>
        </div>
        <div class="header-actions">
          <el-button @click="openImportDialog">批量导入</el-button>
          <el-button type="primary" @click="openCreateDialog">新增关键词</el-button>
        </div>
      </div>

      <div class="card-toolbar">
        <div class="toolbar-left">
          <el-input
            v-model="filters.keyword"
            placeholder="搜索关键词"
            clearable
            class="filter-keyword"
            @keyup.enter="handleSearch"
          />
          <el-button type="primary" @click="handleSearch">筛选</el-button>
          <el-button @click="resetFilters">重置</el-button>
        </div>
        <div class="toolbar-right">
          <span class="summary-text">{{ summaryText }}</span>
        </div>
      </div>

      <div v-if="isMobile" v-loading="loading" class="mobile-keyword-list">
        <article v-for="row in rows" :key="row.id" class="mobile-keyword-card">
          <div class="mobile-keyword-top">
            <div class="cell-keyword">{{ row.keyword }}</div>
            <el-switch
              v-model="row.enabled"
              inline-prompt
              active-text="启用"
              inactive-text="停用"
              @change="(value: string | number | boolean) => handleToggle(row, !!value)"
            />
          </div>
          <div class="mobile-keyword-replacement">替换为：<span class="cell-replacement">{{ row.replacement }}</span></div>
          <div class="mobile-keyword-meta">
            <span class="cell-time">创建：{{ row.createdAt }}</span>
            <span class="cell-time">更新：{{ row.updatedAt }}</span>
          </div>
          <div class="action-group">
            <el-button size="small" type="primary" plain @click="openEditDialog(row)">编辑</el-button>
            <el-button size="small" type="danger" plain @click="handleDelete(row)">删除</el-button>
          </div>
        </article>
      </div>
      <div v-else class="table-scroll-x page-table-wrap">
        <el-table :data="rows" v-loading="loading" stripe size="small" style="width: 100%">
        <el-table-column label="序号" width="70" align="center">
          <template #default="{ $index }">
            <span class="cell-id">{{ (pagination.page - 1) * pagination.pageSize + $index + 1 }}</span>
          </template>
        </el-table-column>
        <el-table-column label="关键词" min-width="220">
          <template #default="{ row }">
            <span class="cell-keyword">{{ row.keyword }}</span>
          </template>
        </el-table-column>
        <el-table-column label="替换文本" :min-width="isMobile ? 160 : 200">
          <template #default="{ row }">
            <span class="cell-replacement">{{ row.replacement }}</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="120" align="center">
          <template #default="{ row }">
            <el-switch
              v-model="row.enabled"
              inline-prompt
              active-text="启用"
              inactive-text="停用"
              @change="(value: string | number | boolean) => handleToggle(row, !!value)"
            />
          </template>
        </el-table-column>
        <el-table-column v-if="!isMobile" label="创建时间" min-width="170">
          <template #default="{ row }">
            <span class="cell-time">{{ row.createdAt }}</span>
          </template>
        </el-table-column>
        <el-table-column v-if="!isMobile" label="更新时间" min-width="170">
          <template #default="{ row }">
            <span class="cell-time">{{ row.updatedAt }}</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" :width="isMobile ? 120 : 160" :fixed="isMobile ? false : 'right'" align="center">
          <template #default="{ row }">
            <div class="action-group">
              <el-button size="small" type="primary" plain @click="openEditDialog(row)">编辑</el-button>
              <el-button size="small" type="danger" plain @click="handleDelete(row)">删除</el-button>
            </div>
          </template>
        </el-table-column>
        </el-table>
      </div>

      <div v-if="!loading && rows.length === 0" class="empty-state">
        当前暂无关键词规则
      </div>

      <div class="pagination-bar">
        <el-pagination
          v-model:current-page="pagination.page"
          :page-size="pagination.pageSize"
          :page-sizes="pageSizeOptions"
          :total="total"
          layout="sizes, prev, pager, next"
          background
          small
          @size-change="(size: number) => { pagination.pageSize = size; pagination.page = 1; fetchRows() }"
          @current-change="fetchRows"
        />
      </div>
    </div>

    <ChatKeywordDialog
      v-model:visible="dialogVisible"
      :mode="dialogMode"
      :keyword="currentKeyword"
      :loading="submitting"
      @submit="handleDialogSubmit"
    />

    <ChatKeywordImportDialog
      v-model:visible="importDialogVisible"
      :loading="submitting"
      @submit="handleImportSubmit"
    />

    <ChatKeywordImportResultDialog
      v-model:visible="importResultDialogVisible"
      :result="latestImportResult"
    />
  </div>
</template>

<style scoped>
.header-actions {
  display: inline-flex;
  flex-wrap: wrap;
  gap: 10px;
}

.page-table-wrap {
  border-bottom: 1px solid #f0f0f0;
}

.card-toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  border-bottom: 1px solid #f0f0f0;
}

.toolbar-left {
  display: flex;
  align-items: center;
  gap: 12px;
  flex: 1;
}

.toolbar-right {
  display: flex;
  align-items: center;
  white-space: nowrap;
}

.filter-keyword {
  width: 240px;
}

.summary-text {
  font-size: 12px;
  color: #94a3b8;
}

.cell-id {
  font-size: 13px;
  color: #94a3b8;
}

.cell-keyword {
  font-family: 'Fira Code', 'Courier New', monospace;
  font-size: 13px;
  font-weight: 600;
  color: #0f172a;
}

.cell-replacement {
  font-family: 'Fira Code', 'Courier New', monospace;
  font-size: 13px;
  color: #475569;
}

.cell-time {
  font-size: 12px;
  color: #64748b;
}

.mobile-keyword-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 12px;
}

.mobile-keyword-card {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 14px;
  border: 1px solid #e8edf4;
  border-radius: 14px;
  background: #fff;
}

.mobile-keyword-top,
.mobile-keyword-meta {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.mobile-keyword-replacement {
  font-size: 13px;
  color: #475569;
}

.action-group {
  display: inline-flex;
  gap: 8px;
}

@media (max-width: 767px) {
  .header-actions,
  .card-toolbar,
  .toolbar-left,
  .pagination-bar {
    flex-direction: column;
    align-items: stretch;
  }

  .header-actions :deep(.el-button),
  .toolbar-left :deep(.el-button) {
    width: 100%;
    margin-left: 0;
  }

  .filter-keyword {
    width: 100%;
  }

  .toolbar-right {
    width: 100%;
    justify-content: flex-start;
  }

  .action-group {
    flex-direction: column;
    width: 100%;
  }

  .mobile-keyword-top,
  .mobile-keyword-meta {
    flex-direction: column;
    align-items: stretch;
  }
}
</style>
