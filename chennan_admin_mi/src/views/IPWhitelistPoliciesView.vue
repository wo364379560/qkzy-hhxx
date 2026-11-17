<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { getIPWhitelistOverview } from '@/api/ipWhitelist'
import IPWhitelistRulesDialog from '@/components/IPWhitelistRulesDialog.vue'
import type { IPWhitelistOverviewItem } from '@/types'
import { useAuthStore } from '@/stores/auth'
import { useResponsive } from '@/composables/useResponsive'

const loading = ref(false)
const rows = ref<IPWhitelistOverviewItem[]>([])
const total = ref(0)
const selectedAccount = ref<{ id: number; username: string; role: 'admin' | 'assistant' } | null>(null)
const showWhitelistDialog = ref(false)
const { isMobile } = useResponsive()
const authStore = useAuthStore()
const pageSubText = computed(() => authStore.isSuperAdmin
  ? '展示已启用白名单的管理员与助理策略，并可快速进入规则管理。'
  : '展示已启用白名单的助理策略，并可快速进入规则管理。')

const filters = reactive({
  keyword: '',
  status: '',
})

const pagination = reactive({
  page: 1,
  pageSize: 20,
})
const pageSizeOptions = [10, 20, 50, 100]

async function fetchRows() {
  loading.value = true
  try {
    const res = await getIPWhitelistOverview({
      page: pagination.page,
      pageSize: pagination.pageSize,
      keyword: filters.keyword || undefined,
      status: filters.status || undefined,
    })
    rows.value = res.data.data?.list ?? []
    total.value = res.data.data?.total ?? 0
  } catch (error: any) {
    ElMessage.error(error.message || '加载白名单策略失败')
  } finally {
    loading.value = false
  }
}

function handleSearch() {
  pagination.page = 1
  fetchRows()
}

function resetFilters() {
  filters.keyword = ''
  filters.status = ''
  handleSearch()
}

function openWhitelistDialog(row: IPWhitelistOverviewItem) {
  selectedAccount.value = {
    id: row.id,
    username: row.username,
    role: row.role,
  }
  showWhitelistDialog.value = true
}

function statusLabel(status: string) {
  return status === 'active' ? '正常' : status === 'banned' ? '已禁用' : status
}

function roleLabel(role: string) {
  return role === 'admin' ? '管理员' : '助理'
}

function managedLabel(row: IPWhitelistOverviewItem) {
  return row.role === 'admin' ? `名下助理 ${row.managedCount}` : `名下用户 ${row.managedCount}`
}

onMounted(fetchRows)
</script>

<template>
  <div class="page">
    <div class="card">
      <div class="card-header">
        <div>
          <h1 class="page-title">白名单策略</h1>
          <p class="page-sub">{{ pageSubText }}</p>
        </div>
      </div>

      <div class="card-toolbar">
        <div class="toolbar-left">
          <el-input
            v-model="filters.keyword"
            placeholder="搜索账号用户名"
            clearable
            class="filter-keyword"
            @keyup.enter="handleSearch"
          />
          <el-select v-model="filters.status" placeholder="账号状态" clearable class="filter-select">
            <el-option label="正常" value="active" />
            <el-option label="已禁用" value="banned" />
          </el-select>
          <el-button type="primary" @click="handleSearch">筛选</el-button>
          <el-button @click="resetFilters">重置</el-button>
        </div>
        <div class="toolbar-right">
          <span class="total-count">共 {{ total }} 条</span>
        </div>
      </div>

      <div v-if="isMobile" v-loading="loading" class="mobile-policy-list">
        <article v-for="row in rows" :key="row.id" class="mobile-policy-card">
          <div class="mobile-policy-top">
            <div>
              <div class="cell-username">{{ row.username }}</div>
              <div class="mobile-policy-sub">{{ roleLabel(row.role) }} · 规则 {{ row.ruleCount }} 条 · {{ managedLabel(row) }}</div>
            </div>
            <span class="status-badge" :class="row.status === 'active' ? 'status-active' : 'status-banned'">
              {{ statusLabel(row.status) }}
            </span>
          </div>

          <div class="mobile-policy-meta">
            <span class="status-badge" :class="row.ipWhitelistEnabled ? 'status-active' : 'status-inactive'">
              {{ row.ipWhitelistEnabled ? '已启用' : '未启用' }}
            </span>
            <span v-if="row.lastUpdatedAt" class="cell-code cell-xs">{{ row.lastUpdatedAt }}</span>
          </div>

          <div v-if="row.rulePreview?.length" class="preview-tags">
            <span v-for="rule in row.rulePreview" :key="rule" class="rule-tag">{{ rule }}</span>
            <span v-if="row.ruleCount > row.rulePreview.length" class="rule-more">+{{ row.ruleCount - row.rulePreview.length }}</span>
          </div>

          <div class="mobile-policy-actions">
            <el-button size="small" type="primary" plain @click="openWhitelistDialog(row)">管理规则</el-button>
          </div>
        </article>
      </div>
      <div v-else class="table-scroll-x page-table-wrap">
        <el-table :data="rows" v-loading="loading" stripe size="small" style="width: 100%">
        <el-table-column label="序号" width="60" align="center">
          <template #default="{ $index }">
            <span class="cell-id">{{ (pagination.page - 1) * pagination.pageSize + $index + 1 }}</span>
          </template>
        </el-table-column>
        <el-table-column label="账号" min-width="140">
          <template #default="{ row }">
            <span class="cell-username">{{ row.username }}</span>
          </template>
        </el-table-column>
        <el-table-column label="角色" width="90">
          <template #default="{ row }">
            <span class="status-badge" :class="row.role === 'admin' ? 'status-admin' : 'status-inactive'">
              {{ roleLabel(row.role) }}
            </span>
          </template>
        </el-table-column>
        <el-table-column label="账号状态" width="100">
          <template #default="{ row }">
            <span class="status-badge" :class="row.status === 'active' ? 'status-active' : 'status-banned'">
              {{ statusLabel(row.status) }}
            </span>
          </template>
        </el-table-column>
        <el-table-column v-if="!isMobile" label="关联规模" width="110">
          <template #default="{ row }">
            <span class="count-badge">{{ row.managedCount }}</span>
          </template>
        </el-table-column>
        <el-table-column label="白名单状态" width="120">
          <template #default="{ row }">
            <span class="status-badge" :class="row.ipWhitelistEnabled ? 'status-active' : 'status-inactive'">
              {{ row.ipWhitelistEnabled ? '已启用' : '未启用' }}
            </span>
          </template>
        </el-table-column>
        <el-table-column label="规则数" width="90">
          <template #default="{ row }">
            <span class="count-badge" :class="{ 'count-badge-active': row.ruleCount > 0 }">{{ row.ruleCount }}</span>
          </template>
        </el-table-column>
        <el-table-column v-if="!isMobile" label="规则预览" min-width="260">
          <template #default="{ row }">
            <div v-if="row.rulePreview?.length" class="preview-tags">
              <span v-for="rule in row.rulePreview" :key="rule" class="rule-tag">{{ rule }}</span>
              <span v-if="row.ruleCount > row.rulePreview.length" class="rule-more">+{{ row.ruleCount - row.rulePreview.length }}</span>
            </div>
            <span v-else class="cell-empty">—</span>
          </template>
        </el-table-column>
        <el-table-column v-if="!isMobile" label="最近更新" min-width="160">
          <template #default="{ row }">
            <span v-if="row.lastUpdatedAt" class="cell-code cell-muted cell-xs">{{ row.lastUpdatedAt }}</span>
            <span v-else class="cell-empty">—</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" :width="isMobile ? 112 : 120" :fixed="isMobile ? false : 'right'" align="center">
          <template #default="{ row }">
            <el-button size="small" type="primary" plain @click="openWhitelistDialog(row)">管理规则</el-button>
          </template>
        </el-table-column>
        </el-table>
      </div>

      <div v-if="!loading && rows.length === 0" class="empty-state">
        当前暂无已启用的白名单策略
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

    <IPWhitelistRulesDialog
      v-model:visible="showWhitelistDialog"
      :account="selectedAccount"
      :subject="selectedAccount?.role === 'admin' ? 'manager' : 'assistant'"
      @success="fetchRows"
    />
  </div>
</template>

<style scoped>
.card-toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  border-bottom: 1px solid #f0f0f0;
}

.page-table-wrap {
  border-bottom: 1px solid #f0f0f0;
}

.toolbar-left {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
  flex: 1;
}

.toolbar-right {
  display: flex;
  align-items: center;
  gap: 12px;
  white-space: nowrap;
}

.filter-keyword {
  width: 240px;
}

.filter-select {
  width: 140px;
}

.total-count {
  font-size: 12px;
  color: #94a3b8;
}

.status-admin {
  color: #1677ff;
  border-color: #91caff;
  background: #e6f4ff;
}

.cell-id {
  font-size: 13px;
  color: #94a3b8;
}

.cell-username {
  font-size: 13px;
  font-weight: 600;
  color: #1677ff;
}

.cell-code {
  font-family: 'Fira Code', 'Courier New', monospace;
  font-size: 13px;
  color: #1e293b;
}

.cell-muted {
  font-size: 12px;
  color: #64748b;
}

.cell-xs {
  font-size: 11px;
  color: #94a3b8;
}

.cell-empty {
  color: #d1d5db;
}

.status-badge {
  display: inline-flex;
  align-items: center;
  font-size: 12px;
  font-weight: 500;
  padding: 0 9px;
  height: 24px;
  border-radius: 6px;
  border: 1px solid transparent;
}

.status-active {
  color: #52c41a;
  border-color: #b7eb8f;
  background: #f6ffed;
}

.status-banned {
  color: #ff4d4f;
  border-color: #ffa39e;
  background: #fff2f0;
}

.status-inactive {
  color: #8c8c8c;
  border-color: #d9d9d9;
  background: #fafafa;
}

.count-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 28px;
  height: 24px;
  padding: 0 8px;
  border-radius: 999px;
  background: #f8fafc;
  color: #64748b;
  font-size: 12px;
  font-weight: 600;
}

.count-badge-active {
  background: #ecfeff;
  color: #0f766e;
}

.preview-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.rule-tag {
  display: inline-flex;
  align-items: center;
  padding: 2px 8px;
  border-radius: 999px;
  background: #f8fafc;
  color: #475569;
  font-size: 11px;
  font-weight: 500;
  font-family: 'Fira Code', 'Courier New', monospace;
}

.rule-more {
  display: inline-flex;
  align-items: center;
  padding: 2px 8px;
  border-radius: 999px;
  background: rgba(22,119,255,0.1);
  color: #1677ff;
  font-size: 11px;
  font-weight: 600;
}

.mobile-policy-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 12px;
}

.mobile-policy-card {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 14px;
  border: 1px solid #e8edf4;
  border-radius: 14px;
  background: #fff;
}

.mobile-policy-top,
.mobile-policy-meta {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.mobile-policy-sub {
  margin-top: 4px;
  font-size: 12px;
  color: #64748b;
}

.mobile-policy-actions {
  display: flex;
}

.pagination-bar {
  display: flex;
  justify-content: flex-end;
  padding: 12px 16px;
  border-top: 1px solid #f0f0f0;
}

.empty-state {
  padding: 36px 16px 20px;
  text-align: center;
  font-size: 13px;
  color: #94a3b8;
}

@media (max-width: 767px) {
  .card-toolbar,
  .toolbar-left,
  .pagination-bar {
    flex-direction: column;
    align-items: stretch;
  }

  .filter-keyword,
  .filter-select {
    width: 100%;
  }

  .toolbar-right {
    width: 100%;
    justify-content: flex-start;
  }

  .mobile-policy-top,
  .mobile-policy-meta,
  .mobile-policy-actions {
    flex-direction: column;
    align-items: stretch;
  }

  .mobile-policy-actions :deep(.el-button) {
    width: 100%;
    margin-left: 0;
  }
}
</style>
