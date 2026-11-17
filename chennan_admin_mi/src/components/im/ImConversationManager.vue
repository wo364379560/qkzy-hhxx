<script setup lang="ts">
import { ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { deleteConversation, exportC2CPairHistory, exportUserC2CHistory, listConversations, queryC2CHistory, queryGroupHistory } from '@/api/im'
import ImConversationMessagesDialog from '@/components/im/ImConversationMessagesDialog.vue'
import ImGroupSearchSelect from '@/components/im/ImGroupSearchSelect.vue'
import ImUserSearchSelect from '@/components/im/ImUserSearchSelect.vue'
import { useAuthStore } from '@/stores/auth'
import type { Conversation, IMGroupOption, IMHistoryMessage, IMUserOption } from '@/types'
import { useResponsive } from '@/composables/useResponsive'

const queryMode = ref<'user' | 'group'>('user')
const route = useRoute()
const authStore = useAuthStore()
const { isMobile } = useResponsive()
const selectedUser = ref<IMUserOption | null>(null)
const selectedGroup = ref<IMGroupOption | null>(null)
const loading = ref(false)
const exportLoading = ref(false)
const hasQueriedUser = ref(false)
const hasQueriedGroup = ref(false)
const conversations = ref<Conversation[]>([])
const groupMessages = ref<IMHistoryMessage[]>([])
const groupHistoryFinished = ref(false)
const groupHistoryNextSeq = ref(0)

const dialogVisible = ref(false)
const dialogLoading = ref(false)
const dialogTitle = ref('')
const dialogMessages = ref<IMHistoryMessage[]>([])
const dialogCanLoadMore = ref(false)

const selectedConversation = ref<Conversation | null>(null)
const nextLastMsgKey = ref('')
const nextMaxTime = ref(0)
const nextMsgSeq = ref(0)

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

async function fetchConversations() {
  if (!selectedUser.value?.imUserID) {
    ElMessage.warning('请先从昵称列表中选择用户')
    return
  }

  hasQueriedUser.value = true
  loading.value = true
  try {
    const res = await listConversations({ fromUserID: selectedUser.value.imUserID, limit: 50 })
    conversations.value = res.data.data ?? []
  } catch (error: any) {
    ElMessage.error(error.message || '获取会话列表失败')
  } finally {
    loading.value = false
  }
}

function resetResults() {
  hasQueriedUser.value = false
  hasQueriedGroup.value = false
  conversations.value = []
  groupMessages.value = []
  groupHistoryFinished.value = false
  groupHistoryNextSeq.value = 0
  dialogVisible.value = false
  dialogMessages.value = []
  dialogTitle.value = ''
  dialogCanLoadMore.value = false
  selectedConversation.value = null
  nextLastMsgKey.value = ''
  nextMaxTime.value = 0
  nextMsgSeq.value = 0
}

function switchMode(mode: 'user' | 'group') {
  if (queryMode.value === mode) return
  queryMode.value = mode
  selectedUser.value = null
  selectedGroup.value = null
  resetResults()
}

async function fetchGroupHistory(loadMore = false) {
  if (!selectedGroup.value?.imGroupId) {
    ElMessage.warning('请先从群组列表中选择群组')
    return
  }

  if (!loadMore) {
    hasQueriedGroup.value = true
  }
  loading.value = true
  try {
    const res = await queryGroupHistory({
      groupID: selectedGroup.value.imGroupId,
      operatorUserID: queryMode.value === 'user' ? selectedUser.value?.imUserID : undefined,
      pageSize: 50,
      lastMsgSeq: loadMore && groupHistoryNextSeq.value > 0 ? groupHistoryNextSeq.value : undefined,
    })
    const data = res.data.data
    groupMessages.value = loadMore ? groupMessages.value.concat(data.list) : data.list
    groupHistoryFinished.value = data.isFinished
    groupHistoryNextSeq.value = data.nextMsgSeq || 0
  } catch (error: any) {
    ElMessage.error(error.message || '获取群聊记录失败')
  } finally {
    loading.value = false
  }
}

async function openConversation(row: Conversation, loadMore = false) {
  selectedConversation.value = row
  dialogTitle.value = `${row.type} 会话 · ${row.peerLabel || row.peerID}`
  dialogVisible.value = true
  dialogLoading.value = true

  try {
    if (row.type === 'C2C') {
      const res = await queryC2CHistory({
        operatorUserID: selectedUser.value?.imUserID || '',
        peerUserID: row.peerID,
        pageSize: 20,
        maxTime: loadMore && nextMaxTime.value > 0 ? nextMaxTime.value : undefined,
        minTime: loadMore && nextMaxTime.value > 0 ? nextMaxTime.value - 7 * 24 * 3600 : undefined,
        lastMsgKey: loadMore ? nextLastMsgKey.value || undefined : undefined,
      })
      const data = res.data.data
      dialogMessages.value = loadMore ? dialogMessages.value.concat(data.list) : data.list
      nextLastMsgKey.value = data.nextLastMsgKey || ''
      nextMaxTime.value = data.nextMaxTime || 0
      dialogCanLoadMore.value = !data.complete && !!data.nextLastMsgKey
    } else {
      const res = await queryGroupHistory({
        groupID: row.peerID,
        operatorUserID: selectedUser.value?.imUserID || undefined,
        pageSize: 50,
        lastMsgSeq: loadMore && nextMsgSeq.value > 0 ? nextMsgSeq.value : undefined,
      })
      const data = res.data.data
      dialogMessages.value = loadMore ? dialogMessages.value.concat(data.list) : data.list
      nextMsgSeq.value = data.nextMsgSeq || 0
      dialogCanLoadMore.value = !data.isFinished && !!data.nextMsgSeq
    }
  } catch (error: any) {
    dialogVisible.value = false
    ElMessage.error(error.message || '获取会话详情失败')
  } finally {
    dialogLoading.value = false
  }
}

async function handleLoadMore() {
  if (!selectedConversation.value) return
  await openConversation(selectedConversation.value, true)
}

async function handleDelete(row: Conversation) {
  await ElMessageBox.confirm(
    `确认删除会话 "${row.peerLabel || row.peerID}"？删除后将同时清理漫游消息。`,
    '确认操作',
    { type: 'warning' },
  )

  try {
    await deleteConversation({
      fromUserID: selectedUser.value?.imUserID || '',
      type: row.type,
      peerID: row.peerID,
      clearRamble: true,
    })
    ElMessage.success('会话已删除')
    await fetchConversations()
  } catch (error: any) {
    ElMessage.error(error.message || '删除会话失败')
  }
}

async function handleQuickExportConversation(row: Conversation) {
  if (row.type !== 'C2C') {
    return
  }
  if (!selectedUser.value?.imUserID) {
    ElMessage.warning('请先从昵称列表中选择用户')
    return
  }

  const operatorLabel = selectedUser.value.nickname || selectedUser.value.imUserID
  const peerLabel = row.peerLabel || row.peerID

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
      operatorUserID: selectedUser.value.imUserID,
      peerUserID: row.peerID,
    })
    ElMessage.success('已开始下载该会话最近30天聊天记录')
  } catch (error: any) {
    ElMessage.error(error.message || '导出聊天记录失败')
  } finally {
    exportLoading.value = false
  }
}

async function handleExportUserHistory() {
  if (!selectedUser.value?.imUserID) {
    ElMessage.warning('请先从昵称列表中选择用户')
    return
  }

  const userLabel = selectedUser.value.nickname || selectedUser.value.imUserID

  try {
    await ElMessageBox.confirm(
      `确认导出「${userLabel}」最近 30 天内全部单聊聊天记录？`,
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
    await exportUserC2CHistory({
      userID: selectedUser.value.imUserID,
    })
    ElMessage.success('已开始下载该用户最近30天全部单聊记录')
  } catch (error: any) {
    ElMessage.error(error.message || '导出聊天记录失败')
  } finally {
    exportLoading.value = false
  }
}

async function applyRouteQuery() {
  const mode = route.query.mode === 'group' ? 'group' : 'user'
  queryMode.value = mode
  resetResults()

  if (mode === 'group') {
    const groupID = String(route.query.groupID || route.query.groupId || '')
    if (!groupID) {
      selectedGroup.value = null
      return
    }
    selectedGroup.value = {
      id: 0,
      name: String(route.query.name || ''),
      imGroupId: groupID,
      groupType: String(route.query.groupType || ''),
    }
    await fetchGroupHistory(false)
    return
  }

  const userID = String(route.query.userID || route.query.imUserID || '')
  if (!userID) {
    selectedUser.value = null
    return
  }
  selectedUser.value = {
    id: 0,
    nickname: String(route.query.nickname || ''),
    phone: String(route.query.phone || ''),
    imUserID: userID,
  }
  await fetchConversations()
}

watch(
  () => route.fullPath,
  () => {
    void applyRouteQuery()
  },
  { immediate: true },
)
</script>

<template>
  <div class="page">
    <div class="card query-card">
      <div class="mode-switch">
        <button class="mode-btn" :class="{ 'mode-btn--active': queryMode === 'user' }" @click="switchMode('user')">用户</button>
        <button class="mode-btn" :class="{ 'mode-btn--active': queryMode === 'group' }" @click="switchMode('group')">群组</button>
      </div>

      <div class="query-bar">
        <ImUserSearchSelect
          v-if="queryMode === 'user'"
          v-model="selectedUser"
          label="目标用户"
          placeholder="输入昵称模糊搜索用户"
        />
        <ImGroupSearchSelect
          v-else
          v-model="selectedGroup"
          label="目标群组"
          placeholder="输入群组名称模糊搜索群组"
        />

        <div class="query-action">
          <el-button type="primary" :loading="loading" @click="queryMode === 'user' ? fetchConversations() : fetchGroupHistory(false)">
            <template #icon><el-icon><Search /></el-icon></template>
            {{ queryMode === 'user' ? '查询会话' : '查询群聊记录' }}
          </el-button>
          <el-button
            v-if="queryMode === 'user'"
            plain
            :loading="exportLoading"
            @click="handleExportUserHistory"
          >
            导出最近30天单聊记录
          </el-button>
        </div>
      </div>
      <p class="query-hint">
        {{
          queryMode === 'user'
            ? '点击输入框可先查看待选昵称，也可以继续输入昵称片段做模糊搜索，然后直接点选用户。导出按钮仅导出该用户最近 30 天全部单聊记录。'
            : '点击输入框可先查看待选群组，也可以继续输入群组名称片段做模糊搜索，然后直接点选群组。'
        }}
      </p>
    </div>

    <div class="card list-card">
      <div class="list-head">
        <div>
          <p class="section-title">{{ queryMode === 'user' ? '会话列表' : '群聊记录' }}</p>
          <p class="section-sub">
            {{
              queryMode === 'user'
                ? '默认展示最近 50 条会话，支持查看最近消息和删除单个会话。'
              : '直接展示该群最近 50 条聊天记录，并支持继续向前加载更多历史消息。'
            }}
          </p>
        </div>
        <el-button
          v-if="queryMode === 'group' && !groupHistoryFinished && groupHistoryNextSeq"
          plain
          :loading="loading"
          @click="fetchGroupHistory(true)"
        >
          加载更多
        </el-button>
      </div>

      <div v-if="queryMode === 'user' && !loading && conversations.length === 0" class="empty-state">
        {{ hasQueriedUser ? '未查询到会话数据' : '先搜索并选择用户，再点击查询会话' }}
      </div>
      <div v-else-if="queryMode === 'group' && !loading && groupMessages.length === 0" class="empty-state">
        {{ hasQueriedGroup ? '未查询到群聊记录' : '先搜索并选择群组，再点击查询群聊记录' }}
      </div>

      <div v-else-if="queryMode === 'user'" class="table-scroll-x result-table-wrap">
        <el-table :data="conversations" v-loading="loading" stripe size="small" style="width: 100%">
        <el-table-column label="类型" width="100">
          <template #default="{ row }">
            <el-tag :type="row.type === 'C2C' ? 'primary' : 'success'" size="small" effect="light" round>
              {{ row.type }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column label="会话对象" min-width="180">
          <template #default="{ row }">
            <div class="peer-cell">
              <span class="peer-label">{{ row.peerLabel || row.peerID }}</span>
              <span class="font-code peer-id">{{ row.peerID }}</span>
            </div>
          </template>
        </el-table-column>

        <el-table-column v-if="!isMobile" label="置顶" width="80" align="center">
          <template #default="{ row }">
            <span class="cell-muted">{{ row.topFlag ? '是' : '否' }}</span>
          </template>
        </el-table-column>

        <el-table-column v-if="!isMobile" label="最后消息时间" min-width="180">
          <template #default="{ row }">
            <span class="font-code cell-xs">{{ formatTime(row.lastMsgTime) }}</span>
          </template>
        </el-table-column>

        <el-table-column label="操作" :width="isMobile ? 220 : 250" :fixed="isMobile ? false : 'right'">
          <template #default="{ row }">
            <div class="action-btns">
              <el-button size="small" type="primary" plain @click="openConversation(row)">查看</el-button>
              <el-tooltip
                v-if="row.type === 'C2C'"
                content="导出该会话最近30天聊天记录"
                placement="top"
              >
                <el-button
                  size="small"
                  plain
                  :loading="exportLoading"
                  @click="handleQuickExportConversation(row)"
                >
                  <template #icon><el-icon><Download /></el-icon></template>
                  导出聊天
                </el-button>
              </el-tooltip>
              <el-button size="small" type="danger" plain @click="handleDelete(row)">删除</el-button>
            </div>
          </template>
        </el-table-column>
        </el-table>
      </div>

      <div v-else class="table-scroll-x result-table-wrap">
        <el-table :data="groupMessages" v-loading="loading" stripe size="small" style="width: 100%">
        <el-table-column label="时间" width="180">
          <template #default="{ row }">
            <span class="font-code cell-xs">{{ formatTime(row.msgTime) }}</span>
          </template>
        </el-table-column>

        <el-table-column v-if="!isMobile" label="发送方" width="160">
          <template #default="{ row }">
            <span class="font-code cell-muted">{{ row.fromUserID }}</span>
          </template>
        </el-table-column>

        <el-table-column label="内容" min-width="320">
          <template #default="{ row }">
            <span v-if="row.isRecalled" class="cell-recalled">[ 消息已撤回 ]</span>
            <span v-else class="cell-content">{{ row.content }}</span>
          </template>
        </el-table-column>
        </el-table>
      </div>
    </div>

    <ImConversationMessagesDialog
      v-if="queryMode === 'user'"
      v-model:visible="dialogVisible"
      :title="dialogTitle"
      :loading="dialogLoading"
      :messages="dialogMessages"
      :can-load-more="dialogCanLoadMore"
      @load-more="handleLoadMore"
    />
  </div>
</template>

<style scoped>
.page {
  padding: 24px 28px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.query-card,
.list-card {
  padding: 20px 22px;
}

.query-card {
  overflow: visible;
}

.mode-switch {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 14px;
}

.mode-btn {
  border: 1px solid #dcdfe6;
  background: #fff;
  color: #475569;
  height: 32px;
  padding: 0 14px;
  border-radius: 8px;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s ease;
}

.mode-btn--active {
  border-color: #409eff;
  background: rgba(64, 158, 255, 0.12);
  color: #1677ff;
}

.query-bar {
  display: grid;
  grid-template-columns: minmax(0, 420px) auto;
  gap: 12px;
  align-items: end;
}

.query-field {
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
  align-items: center;
  align-self: end;
  gap: 12px;
}

.query-action :deep(.el-button) {
  height: 36px;
  padding: 0 16px;
}

.query-hint {
  margin: 12px 0 0;
  font-size: 12px;
  color: #94a3b8;
}

.list-head {
  margin-bottom: 12px;
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

.result-table-wrap {
  border-bottom: 1px solid #f0f0f0;
}

.peer-cell {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.peer-label {
  font-size: 13px;
  font-weight: 500;
  color: #1e293b;
}

.peer-id {
  font-size: 11px;
  color: #94a3b8;
}

.cell-muted {
  font-size: 12px;
  color: #64748b;
}

.cell-xs {
  font-size: 11px;
  color: #94a3b8;
}

.action-btns {
  display: flex;
  gap: 8px;
}

@media (max-width: 900px) {
  .query-bar {
    grid-template-columns: 1fr;
  }

  .query-action {
    padding-top: 0;
  }
}

@media (max-width: 767px) {
  .page {
    padding: 0;
  }

  .query-card,
  .list-card {
    padding: 16px 14px;
  }

  .mode-switch {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .mode-btn {
    width: 100%;
  }

  .list-head {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .query-action :deep(.el-button) {
    width: 100%;
  }

  .action-btns {
    flex-direction: column;
  }
}
</style>
