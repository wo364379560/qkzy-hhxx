<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { getReportList, reviewReport, dismissReport } from '@/api/report'
import { useAuthStore } from '@/stores/auth'
import type { Report, ReportStatus, ReportCategory } from '@/types'
import { useResponsive } from '@/composables/useResponsive'

const authStore = useAuthStore()
const { isMobile } = useResponsive()
const isAdmin = computed(() => authStore.isAdmin)
const pageSubText = computed(() => {
  if (authStore.isSuperAdmin) return '处理全量用户举报'
  if (authStore.isManager) return '处理我归属范围内用户举报'
  return '处理我的用户举报'
})

type StatusFilter = 'all' | ReportStatus
const activeFilter = ref<StatusFilter>('all')
const loading  = ref(false)
const filters: { key: StatusFilter; label: string }[] = [
  { key: 'all',       label: '全部'   },
  { key: 'pending',   label: '待处理' },
  { key: 'reviewed',  label: '已处理' },
  { key: 'dismissed', label: '已忽略' },
]
const reports  = ref<Report[]>([])
const total    = ref(0)
const page     = ref(1)
const pageSize = ref(10)

const counts = computed(() => ({
  all:       total.value,
  pending:   reports.value.filter(r => r.status === 'pending').length,
  reviewed:  reports.value.filter(r => r.status === 'reviewed').length,
  dismissed: reports.value.filter(r => r.status === 'dismissed').length,
}))

async function fetchReports() {
  loading.value = true
  try {
    const status = activeFilter.value === 'all' ? '' : activeFilter.value
    const res = await getReportList({ page: page.value, pageSize: pageSize.value, status: status || undefined })
    reports.value = res.data.data?.list ?? []; total.value = res.data.data?.total ?? 0
  } catch (e: any) { ElMessage.error(e.message || '获取举报列表失败') }
  finally { loading.value = false }
}
onMounted(fetchReports)

function onFilterChange(key: StatusFilter) { activeFilter.value = key; page.value = 1; fetchReports() }

const categoryLabel: Record<ReportCategory, string> = { spam: '垃圾信息', abuse: '违规内容', fraud: '诈骗', other: '其他' }
function statusLabel(s: ReportStatus) { return { pending: '待处理', reviewed: '已处理', dismissed: '已忽略' }[s] }
function targetTypeLabel(type: string) {
  switch (type) {
    case 'user': return '用户'
    case 'group': return '群组'
    case 'saving_post': return '省钱社区内容'
    case 'saving_comment': return '内容评论'
    case 'community_post': return '省钱社区内容'
    case 'community_comment': return '内容评论'
    default: return type
  }
}

async function handleBan(r: Report) {
  try { await reviewReport(r.id); ElMessage.success('已标记为已处理'); fetchReports() }
  catch (e: any) { ElMessage.error(e.message || '操作失败') }
}
async function handleDismiss(r: Report) {
  try { await dismissReport(r.id); ElMessage.success('已忽略'); fetchReports() }
  catch (e: any) { ElMessage.error(e.message || '操作失败') }
}
</script>

<template>
  <div class="page">
    <div class="card">
      <div class="card-header">
        <div>
          <h1 class="page-title">举报管理</h1>
          <p class="page-sub">{{ pageSubText }}</p>
        </div>
      </div>
      <div class="filter-toolbar">
        <div class="filter-tabs">
          <button v-for="f in filters" :key="f.key"
            class="filter-tab" :class="{ active: activeFilter === f.key }"
            @click="onFilterChange(f.key)">
            {{ f.label }}
            <span class="filter-count" :class="{ 'count-active': activeFilter === f.key }">{{ counts[f.key] }}</span>
          </button>
        </div>
        <span class="total-count">共 {{ total }} 条</span>
      </div>

      <div v-if="isMobile" v-loading="loading" class="mobile-report-list">
        <article v-for="row in reports" :key="row.id" class="mobile-report-card">
          <div class="mobile-report-top">
            <div class="mobile-report-people">
              <div>
                <div class="mobile-report-label">举报人</div>
                <div class="cell-name">{{ row.reporterName }}</div>
                <div class="cell-id font-code">ID: {{ row.reporterID }}</div>
              </div>
              <div>
                <div class="mobile-report-label">被举报</div>
                <div class="cell-name">{{ row.targetName }}</div>
                <div class="cell-id font-code">ID: {{ row.targetID }}</div>
              </div>
            </div>
            <span class="status-badge" :class="`status-${row.status}`">{{ statusLabel(row.status) }}</span>
          </div>

          <div class="mobile-report-meta">
            <span class="category-badge">{{ targetTypeLabel(row.targetType) }}</span>
            <span class="category-badge">{{ categoryLabel[row.category] ?? row.category }}</span>
            <span class="font-code cell-xs">{{ row.createdAt }}</span>
          </div>

          <p class="mobile-report-desc">{{ row.description || '未填写描述' }}</p>

          <div class="mobile-report-actions">
            <template v-if="row.status === 'pending'">
              <el-button size="small" type="success" plain @click="handleBan(row)">已处理</el-button>
              <el-button size="small" plain @click="handleDismiss(row)">忽略</el-button>
            </template>
          </div>
        </article>
        <div v-if="!loading && reports.length === 0" class="empty-state">当前暂无举报记录</div>
      </div>
      <div v-else class="table-scroll-x page-table-wrap">
        <el-table :data="reports" v-loading="loading" stripe size="small" style="width:100%">
        <el-table-column label="序号" width="60" align="center">
          <template #default="{ $index }"><span class="cell-xs">{{ (page - 1) * pageSize + $index + 1 }}</span></template>
        </el-table-column>
        <el-table-column label="举报人" min-width="140">
          <template #default="{ row }">
            <p class="cell-name">{{ row.reporterName }}</p>
            <p class="cell-id font-code">ID: {{ row.reporterID }}</p>
          </template>
        </el-table-column>
        <el-table-column label="被举报" min-width="140">
          <template #default="{ row }">
            <p class="cell-name">{{ row.targetName }}</p>
            <p class="cell-id font-code">ID: {{ row.targetID }}</p>
          </template>
        </el-table-column>
        <el-table-column v-if="!isMobile" label="对象类型" width="100">
          <template #default="{ row }">
            <span class="category-badge">{{ targetTypeLabel(row.targetType) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="分类" width="100">
          <template #default="{ row }">
            <span class="category-badge">{{ categoryLabel[row.category] ?? row.category }}</span>
          </template>
        </el-table-column>
        <el-table-column v-if="!isMobile" label="描述" min-width="200">
          <template #default="{ row }">
            <p class="cell-desc" :title="row.description">{{ row.description }}</p>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <span class="status-badge" :class="`status-${row.status}`">{{ statusLabel(row.status) }}</span>
          </template>
        </el-table-column>
        <el-table-column v-if="!isMobile" label="举报时间" min-width="160">
          <template #default="{ row }"><span class="font-code cell-xs">{{ row.createdAt }}</span></template>
        </el-table-column>
        <el-table-column label="操作" :width="isMobile ? 156 : 140" :fixed="isMobile ? false : 'right'">
          <template #default="{ row }">
            <template v-if="row.status === 'pending'">
              <el-button size="small" type="success" plain style="padding:2px 8px;font-size:12px" @click="handleBan(row)">已处理</el-button>
              <el-button size="small" plain style="padding:2px 8px;font-size:12px" @click="handleDismiss(row)">忽略</el-button>
            </template>
            <span v-else class="cell-empty">—</span>
          </template>
        </el-table-column>
        </el-table>
      </div>

      <div class="pagination-bar">
        <el-pagination v-model:current-page="page" :page-size="pageSize" :total="total"
          layout="prev, pager, next" background small @current-change="(p) => { page = p; fetchReports() }" />
      </div>
    </div>
  </div>
</template>

<style scoped>

.filter-toolbar { display: flex; align-items: center; padding: 10px 16px; border-bottom: 1px solid #f0f0f0; gap: 4px; }
.page-table-wrap { border-bottom: 1px solid #f0f0f0; }
.filter-tabs { display: flex; align-items: center; gap: 4px; flex: 1; }
.filter-tab {
  display: flex; align-items: center; gap: 6px;
  padding: 6px 12px; border-radius: 7px; border: 1px solid transparent;
  font-size: 12px; font-weight: 500; cursor: pointer; background: none;
  color: #64748b; transition: all 0.15s;
}
.filter-tab:hover { background: #f8fafc; color: #1e293b; }
.filter-tab.active { background: #fafafa; color: #1e293b; border-color: #f0f0f0; box-shadow: 0 1px 2px rgba(0,0,0,0.05); }
.filter-count {
  font-size: 10px; font-family: 'Fira Code', monospace;
  padding: 1px 6px; border-radius: 4px; background: #f1f5f9; color: #94a3b8;
}
.filter-count.count-active { background: rgba(22,119,255,0.1); color: #1677ff; }
.total-count { font-size: 12px; color: #94a3b8; margin-left: auto; }

.cell-name  { font-size: 13px; font-weight: 500; color: #1e293b; line-height: 1.3; }
.cell-id    { font-size: 11px; color: #94a3b8; margin-top: 1px; }
.cell-xs    { font-size: 11px; color: #94a3b8; }
.cell-desc  { font-size: 12px; color: #64748b; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; max-width: 200px; }
.cell-empty { color: #d1d5db; font-size: 12px; }

.category-badge { background: #f1f5f9; color: #475569; font-size: 12px; padding: 2px 8px; border-radius: 4px; font-weight: 500; white-space: nowrap; }

.mobile-report-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 12px;
}

.mobile-report-card {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 14px;
  border: 1px solid #e8edf4;
  border-radius: 14px;
  background: #fff;
}

.mobile-report-top {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.mobile-report-people {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.mobile-report-label {
  font-size: 11px;
  color: #94a3b8;
  margin-bottom: 4px;
}

.mobile-report-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.mobile-report-desc {
  font-size: 13px;
  line-height: 1.6;
  color: #475569;
}

.mobile-report-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.pagination-bar { display: flex; justify-content: flex-end; padding: 12px 16px; border-top: 1px solid #f0f0f0; }

.status-badge { display: inline-flex; align-items: center; font-size: 12px; font-weight: 500; padding: 0 9px; height: 24px; border-radius: 6px; border: 1px solid transparent; }
.status-pending  { color: #d46b08; border-color: #ffd591; background: #fff7e6; }
.status-reviewed { color: #52c41a; border-color: #b7eb8f; background: #f6ffed; }
.status-dismissed { color: #8c8c8c; border-color: #d9d9d9; background: #fafafa; }

@media (max-width: 767px) {
  .filter-toolbar,
  .filter-tabs,
  .pagination-bar {
    flex-direction: column;
    align-items: stretch;
  }

  .filter-tab {
    justify-content: space-between;
    min-height: 40px;
  }

  .total-count {
    margin-left: 0;
  }

  .mobile-report-people,
  .mobile-report-actions {
    grid-template-columns: 1fr;
    flex-direction: column;
  }

  .mobile-report-actions :deep(.el-button) {
    width: 100%;
    margin-left: 0;
  }
}
</style>
