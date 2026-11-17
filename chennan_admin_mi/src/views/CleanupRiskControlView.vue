<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  banCleanupCandidate,
  getCleanupCandidates,
  getCleanupStats,
  restoreCleanupUser,
  whitelistCleanupCandidate,
  type CleanupCandidate,
  type CleanupStats,
} from '@/api/operation'

const loading = ref(false)
const actionLoading = ref(false)
const rows = ref<CleanupCandidate[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const stats = ref<CleanupStats>({
  candidateCount: 0,
  bannedCount: 0,
  whitelistCount: 0,
  manualBannedCount: 0,
  restoredCount: 0,
  protectedAAccounts: 0,
  createdByNonZero: 0,
  withInviteCode: 0,
})

const filters = reactive({
  keyword: '',
  currentLoginIP: '',
})

async function fetchStats() {
  const res = await getCleanupStats()
  stats.value = res.data.data ?? stats.value
}

async function fetchRows() {
  loading.value = true
  try {
    await fetchStats()
    const res = await getCleanupCandidates({
      page: page.value,
      pageSize: pageSize.value,
      keyword: filters.keyword || undefined,
      currentLoginIP: filters.currentLoginIP || undefined,
    })
    rows.value = res.data.data?.list ?? []
    total.value = res.data.data?.total ?? 0
  } catch (error: any) {
    ElMessage.error(error.message || '加载清理风控失败')
  } finally {
    loading.value = false
  }
}

function search() {
  page.value = 1
  fetchRows()
}

async function askReason(title: string, placeholder: string) {
  const result = await ElMessageBox.prompt(placeholder, title, {
    confirmButtonText: '确认',
    cancelButtonText: '取消',
    inputType: 'textarea',
  })
  return String(result.value || '')
}

async function addWhitelist(row: CleanupCandidate) {
  try {
    const reason = await askReason('加入清理白名单', '请输入加白原因')
    actionLoading.value = true
    await whitelistCleanupCandidate(row.id, reason)
    ElMessage.success('已加入清理白名单')
    await fetchRows()
  } catch (error: any) {
    if (error !== 'cancel') ElMessage.error(error.message || '操作失败')
  } finally {
    actionLoading.value = false
  }
}

async function banCandidate(row: CleanupCandidate) {
  try {
    await ElMessageBox.confirm('该操作会把本地账号置为 banned，并调用腾讯 IM 封禁与踢下线；可在恢复入口恢复。', '确认封禁候选账号', {
      type: 'warning',
      confirmButtonText: '封禁',
      cancelButtonText: '取消',
    })
    const reason = await askReason('封禁说明', '请输入封禁原因')
    actionLoading.value = true
    await banCleanupCandidate(row.id, reason)
    ElMessage.success('已封禁')
    await fetchRows()
  } catch (error: any) {
    if (error !== 'cancel') ElMessage.error(error.message || '操作失败')
  } finally {
    actionLoading.value = false
  }
}

async function restore(row: CleanupCandidate) {
  try {
    const reason = await askReason('恢复账号', '请输入恢复原因')
    actionLoading.value = true
    await restoreCleanupUser(row.id, reason)
    ElMessage.success('已恢复')
    await fetchRows()
  } catch (error: any) {
    if (error !== 'cancel') ElMessage.error(error.message || '操作失败')
  } finally {
    actionLoading.value = false
  }
}

function statusLabel(status: string) {
  return status === 'active' ? '正常' : status === 'banned' ? '已封禁' : status === 'deleted' ? '已删除' : status
}

onMounted(fetchRows)
</script>

<template>
  <div class="cleanup-page">
    <div class="page-head">
      <div>
        <h1>清理风控</h1>
        <p>自动清理只处理普通注册、无邀请码、超过 30 分钟、created_by=0、非 a_、非白名单账号。</p>
      </div>
      <el-button @click="fetchRows">刷新</el-button>
    </div>

    <section class="stats-grid">
      <article class="stat"><span>候选</span><strong class="danger">{{ stats.candidateCount }}</strong></article>
      <article class="stat"><span>已封禁</span><strong>{{ stats.bannedCount }}</strong></article>
      <article class="stat"><span>清理白名单</span><strong>{{ stats.whitelistCount }}</strong></article>
      <article class="stat"><span>手动封禁</span><strong>{{ stats.manualBannedCount }}</strong></article>
      <article class="stat"><span>已恢复</span><strong>{{ stats.restoredCount }}</strong></article>
      <article class="stat"><span>a_ 保护</span><strong>{{ stats.protectedAAccounts }}</strong></article>
      <article class="stat"><span>后台创建保护</span><strong>{{ stats.createdByNonZero }}</strong></article>
      <article class="stat"><span>邀请码保护</span><strong>{{ stats.withInviteCode }}</strong></article>
    </section>

    <section class="panel">
      <div class="toolbar">
        <el-input v-model="filters.keyword" placeholder="用户ID / IM ID / 手机号 / 昵称 / 邀请码" clearable @keyup.enter="search" />
        <el-input v-model="filters.currentLoginIP" placeholder="当前登录 IP" clearable @keyup.enter="search" />
        <el-button type="primary" @click="search">筛选</el-button>
      </div>

      <el-table v-loading="loading" :data="rows" stripe size="small" class="risk-table">
        <el-table-column label="类型" width="112">
          <template #default="{ row }"><el-tag type="danger" effect="plain">无邀请码</el-tag></template>
        </el-table-column>
        <el-table-column label="用户" min-width="170">
          <template #default="{ row }">
            <div class="strong">{{ row.nickname || '-' }}</div>
            <div class="muted">{{ row.imUserID || '-' }}</div>
          </template>
        </el-table-column>
        <el-table-column label="账号" min-width="150">
          <template #default="{ row }">{{ row.phone || row.wananID || '-' }}</template>
        </el-table-column>
        <el-table-column label="当前登录IP" min-width="130">
          <template #default="{ row }">{{ row.currentLoginIP || '-' }}</template>
        </el-table-column>
        <el-table-column label="归属" min-width="130">
          <template #default="{ row }">
            <div>{{ row.ownerName || '-' }}</div>
            <div class="muted">{{ row.ownerRole || '-' }}</div>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="{ row }"><el-tag effect="plain">{{ statusLabel(row.status) }}</el-tag></template>
        </el-table-column>
        <el-table-column label="注册时间" min-width="160" prop="createdAt" />
        <el-table-column label="处置说明" min-width="240" prop="reason" />
        <el-table-column label="操作" width="220" fixed="right">
          <template #default="{ row }">
            <el-button size="small" :disabled="actionLoading" @click="addWhitelist(row)">加白</el-button>
            <el-button size="small" type="danger" plain :disabled="actionLoading || row.status !== 'active'" @click="banCandidate(row)">封禁</el-button>
            <el-button size="small" plain :disabled="actionLoading || row.status !== 'banned'" @click="restore(row)">恢复</el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="pagination-bar">
        <el-pagination
          v-model:current-page="page"
          v-model:page-size="pageSize"
          :total="total"
          :page-sizes="[10, 20, 50, 100]"
          layout="sizes, prev, pager, next"
          @change="fetchRows"
        />
      </div>
    </section>
  </div>
</template>

<style scoped>
.cleanup-page { display: flex; flex-direction: column; gap: 14px; }
.page-head { display: flex; justify-content: space-between; gap: 12px; align-items: flex-start; }
.page-head h1 { margin: 0 0 8px; font-size: 26px; color: #111827; }
.page-head p, .muted { color: #667085; }
.stats-grid { display: grid; grid-template-columns: repeat(8, minmax(0, 1fr)); gap: 10px; }
.stat { border: 1px solid #e6eaf2; background: #fff; border-radius: 8px; padding: 14px; }
.stat span { display: block; color: #667085; font-size: 13px; }
.stat strong { display: block; margin-top: 8px; font-size: 24px; color: #2563eb; }
.stat strong.danger { color: #dc2626; }
.panel { background: #fff; border: 1px solid #e6eaf2; border-radius: 8px; padding: 14px; }
.toolbar { display: grid; grid-template-columns: minmax(220px, 1fr) minmax(180px, 260px) 88px; gap: 10px; margin-bottom: 12px; }
.strong { font-weight: 700; color: #111827; }
.risk-table { width: 100%; }
.pagination-bar { display: flex; justify-content: flex-end; padding-top: 12px; }
@media (max-width: 1100px) { .stats-grid { grid-template-columns: repeat(4, minmax(0, 1fr)); } }
@media (max-width: 760px) {
  .page-head { flex-direction: column; }
  .stats-grid, .toolbar { grid-template-columns: 1fr; }
}
</style>
