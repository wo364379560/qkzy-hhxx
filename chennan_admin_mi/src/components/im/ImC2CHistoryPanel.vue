<script setup lang="ts">
import { reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { exportC2CPairHistory, queryC2CHistory, withdrawC2CMessage } from '@/api/im'
import ImUserSearchSelect from '@/components/im/ImUserSearchSelect.vue'
import type { C2CHistoryResult, IMHistoryMessage, IMUserOption } from '@/types'
import { useResponsive } from '@/composables/useResponsive'

const loading = ref(false)
const exportLoading = ref(false)
const withdrawingKey = ref('')
const { isMobile } = useResponsive()
const query = reactive({
  operatorUser: null as IMUserOption | null,
  peerUser: null as IMUserOption | null,
})
const history = ref<C2CHistoryResult | null>(null)
const rows = ref<IMHistoryMessage[]>([])

function nowUnix() {
  return Math.floor(Date.now() / 1000)
}

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

async function fetchHistory(loadMore = false) {
  if (!query.operatorUser?.imUserID || !query.peerUser?.imUserID) {
    ElMessage.warning('请先从昵称列表中选择双方用户')
    return
  }

  loading.value = true
  try {
    const baseMaxTime = history.value?.nextMaxTime && loadMore ? history.value.nextMaxTime : nowUnix()
    const res = await queryC2CHistory({
      operatorUserID: query.operatorUser.imUserID,
      peerUserID: query.peerUser.imUserID,
      pageSize: 50,
      minTime: baseMaxTime - 7 * 24 * 3600,
      maxTime: baseMaxTime,
      lastMsgKey: loadMore ? history.value?.nextLastMsgKey : undefined,
    })
    history.value = res.data.data
    rows.value = loadMore ? rows.value.concat(history.value.list) : history.value.list
  } catch (error: any) {
    ElMessage.error(error.message || '查询消息失败')
  } finally {
    loading.value = false
  }
}

async function handleWithdraw(row: IMHistoryMessage) {
  if (!row.msgKey || row.isRecalled) return
  withdrawingKey.value = row.msgKey
  try {
    await withdrawC2CMessage({
      fromUserID: row.fromUserID,
      toUserID: row.toUserID || query.peerUser?.imUserID || '',
      msgKey: row.msgKey,
    })
    // 先做本地回显，避免腾讯侧状态同步存在短暂延迟时页面看起来“没变化”。
    row.isRecalled = true
    row.content = ''
    ElMessage.success('撤回成功')
    setTimeout(() => {
      void fetchHistory(false)
    }, 800)
  } catch (error: any) {
    ElMessage.error(error.message || '撤回失败')
  } finally {
    withdrawingKey.value = ''
  }
}

async function handleExport() {
  if (!query.operatorUser?.imUserID || !query.peerUser?.imUserID) {
    ElMessage.warning('请先从昵称列表中选择双方用户')
    return
  }

  const operatorLabel = query.operatorUser.nickname || query.operatorUser.imUserID
  const peerLabel = query.peerUser.nickname || query.peerUser.imUserID

  try {
    await ElMessageBox.confirm(
      `确认导出「${operatorLabel}」与「${peerLabel}」最近 30 天的单聊记录？`,
      '确认导出',
      {
        type: 'warning',
        confirmButtonText: '确认导出',
        cancelButtonText: '取消',
      },
    )
  } catch {
    return
  }

  exportLoading.value = true
  try {
    await exportC2CPairHistory({
      operatorUserID: query.operatorUser.imUserID,
      peerUserID: query.peerUser.imUserID,
    })
    ElMessage.success('已开始下载最近30天聊天记录')
  } catch (error: any) {
    ElMessage.error(error.message || '导出聊天记录失败')
  } finally {
    exportLoading.value = false
  }
}
</script>

<template>
  <div class="panel">
    <div class="card query-card">
      <div class="query-grid">
        <ImUserSearchSelect
          v-model="query.operatorUser"
          label="用户 A"
          placeholder="输入昵称模糊搜索用户 A"
        />

        <ImUserSearchSelect
          v-model="query.peerUser"
          label="用户 B"
          placeholder="输入昵称模糊搜索用户 B"
        />

        <div class="query-action">
          <el-button type="primary" :loading="loading" @click="fetchHistory(false)">
            <template #icon><el-icon><Search /></el-icon></template>
            查询消息
          </el-button>
          <el-button plain :loading="exportLoading" @click="handleExport">
            导出最近30天
          </el-button>
        </div>
      </div>

      <p class="query-hint">点击输入框可先查看待选昵称，也支持继续模糊搜索；选中双方用户后可查询最近 7 天消息，并继续向前加载更多记录。导出按钮仅导出最近 30 天聊天记录。</p>
    </div>

    <div class="card result-card">
      <div class="result-head">
        <div>
          <p class="section-title">消息记录</p>
          <p class="section-sub">仅支持真实消息查询与撤回，不支持单条物理删除。</p>
        </div>

        <el-button
          v-if="history && !history.complete && history.nextLastMsgKey"
          plain
          :loading="loading"
          @click="fetchHistory(true)"
        >
          加载更多
        </el-button>
      </div>

      <div v-if="!loading && rows.length === 0" class="empty-state">
        先搜索并选择双方用户，再点击查询消息
      </div>

      <div v-else class="table-scroll-x result-table-wrap">
        <el-table :data="rows" v-loading="loading" stripe size="small" style="width: 100%">
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

        <el-table-column label="内容" min-width="280">
          <template #default="{ row }">
            <span v-if="row.isRecalled" class="cell-recalled">[ 消息已撤回 ]</span>
            <span v-else class="cell-content">{{ row.content }}</span>
          </template>
        </el-table-column>

        <el-table-column label="操作" :width="isMobile ? 100 : 120" :fixed="isMobile ? false : 'right'">
          <template #default="{ row }">
            <el-button
              v-if="!row.isRecalled && row.msgKey"
              size="small"
              type="warning"
              plain
              :loading="withdrawingKey === row.msgKey"
              @click="handleWithdraw(row)"
            >
              撤回
            </el-button>
            <span v-else class="cell-empty">—</span>
          </template>
        </el-table-column>
        </el-table>
      </div>
    </div>
  </div>
</template>

<style scoped>
.panel {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.query-card,
.result-card {
  padding: 20px 22px;
}

.query-card {
  overflow: visible;
}

.query-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr)) auto;
  gap: 12px;
}

.form-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.form-label {
  margin: 0;
  font-size: 12px;
  font-weight: 500;
  color: #475569;
}

.query-action {
  display: flex;
  align-items: end;
  gap: 12px;
}

.query-hint {
  margin: 12px 0 0;
  font-size: 12px;
  color: #94a3b8;
}

.result-head {
  margin-bottom: 12px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.result-table-wrap {
  border-bottom: 1px solid #f0f0f0;
}

.section-title {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
  color: #1e293b;
}

.section-sub {
  margin: 4px 0 0;
  font-size: 12px;
  color: #94a3b8;
}

.empty-state {
  padding: 52px 20px;
  text-align: center;
  color: #94a3b8;
  font-size: 13px;
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

.cell-empty {
  color: #d1d5db;
  font-size: 12px;
}

@media (max-width: 900px) {
  .query-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 767px) {
  .query-card,
  .result-card {
    padding: 16px 14px;
  }

  .result-head {
    flex-direction: column;
    align-items: stretch;
  }
}
</style>
