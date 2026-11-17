<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  getOfflineActivityList,
  createOfflineActivity,
  updateOfflineActivity,
  closeOfflineActivity,
  getOfflineActivityRegistrations,
} from '@/api/offlineActivity'
import type { OfflineActivity, OfflineActivityRegistration } from '@/types'
import { useResponsive } from '@/composables/useResponsive'

const loading = ref(false)
const submitting = ref(false)
const { isMobile } = useResponsive()
const activities = ref<OfflineActivity[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)
const statusFilter = ref('')

const showModal = ref(false)
const editingId = ref<number | null>(null)
const showRegistrationsModal = ref(false)
const registrationsLoading = ref(false)
const currentActivity = ref<OfflineActivity | null>(null)
const registrations = ref<OfflineActivityRegistration[]>([])

const form = ref({
  title: '',
  subtitle: '',
  description: '',
  city: '',
  venue: '',
  startAt: '',
  endAt: '',
  quota: 0,
  status: 'draft',
})

const filteredActivities = computed(() => {
  if (!statusFilter.value) return activities.value
  return activities.value.filter(item => item.status === statusFilter.value)
})

function statusLabel(status: string) {
  switch (status) {
    case 'draft': return '草稿'
    case 'published': return '已发布'
    case 'closed': return '已关闭'
    case 'cancelled': return '已取消'
    default: return status
  }
}

function statusClass(status: string) {
  switch (status) {
    case 'published': return 'status-active'
    case 'draft': return 'status-draft'
    case 'closed': return 'status-closed'
    case 'cancelled': return 'status-cancelled'
    default: return 'status-draft'
  }
}

function quotaText(activity: OfflineActivity) {
  return activity.quota > 0 ? `${activity.registrationCount}/${activity.quota}` : `${activity.registrationCount}/不限`
}

async function fetchActivities() {
  loading.value = true
  try {
    const res = await getOfflineActivityList({
      page: page.value,
      pageSize: pageSize.value,
      status: statusFilter.value || undefined,
    })
    activities.value = res.data.data?.list ?? []
    total.value = res.data.data?.total ?? 0
  } catch (e: any) {
    ElMessage.error(e.message || '获取线下省钱活动列表失败')
  } finally {
    loading.value = false
  }
}

function resetForm() {
  form.value = {
    title: '',
    subtitle: '',
    description: '',
    city: '',
    venue: '',
    startAt: '',
    endAt: '',
    quota: 0,
    status: 'draft',
  }
}

function openCreateModal() {
  editingId.value = null
  resetForm()
  showModal.value = true
}

function openEditModal(activity: OfflineActivity) {
  editingId.value = activity.id
  form.value = {
    title: activity.title,
    subtitle: activity.subtitle,
    description: activity.description,
    city: activity.city,
    venue: activity.venue,
    startAt: activity.startAt,
    endAt: activity.endAt,
    quota: activity.quota,
    status: activity.status,
  }
  showModal.value = true
}

async function submitForm() {
  if (!form.value.title.trim()) {
    ElMessage.warning('请填写活动标题')
    return
  }
  if (!form.value.city.trim()) {
    ElMessage.warning('请填写城市')
    return
  }
  if (!form.value.venue.trim()) {
    ElMessage.warning('请填写活动地点')
    return
  }
  if (!form.value.startAt.trim() || !form.value.endAt.trim()) {
    ElMessage.warning('请填写活动时间')
    return
  }

  submitting.value = true
  try {
    const payload = {
      title: form.value.title.trim(),
      subtitle: form.value.subtitle.trim(),
      description: form.value.description.trim(),
      city: form.value.city.trim(),
      venue: form.value.venue.trim(),
      startAt: form.value.startAt.trim(),
      endAt: form.value.endAt.trim(),
      quota: Number(form.value.quota || 0),
      status: form.value.status,
    }
    if (editingId.value) {
      await updateOfflineActivity(editingId.value, payload)
      ElMessage.success('活动已更新')
    } else {
      await createOfflineActivity(payload)
      ElMessage.success('活动已创建')
    }
    showModal.value = false
    fetchActivities()
  } catch (e: any) {
    ElMessage.error(e.message || '保存线下省钱活动失败')
  } finally {
    submitting.value = false
  }
}

async function handleClose(activity: OfflineActivity) {
  await ElMessageBox.confirm(`确认下线活动 "${activity.title}"？`, '确认操作', { type: 'warning' })
  try {
    await closeOfflineActivity(activity.id)
    ElMessage.success('活动已下线')
    fetchActivities()
  } catch (e: any) {
    ElMessage.error(e.message || '下线线下省钱活动失败')
  }
}

async function openRegistrationsModal(activity: OfflineActivity) {
  currentActivity.value = activity
  registrations.value = []
  showRegistrationsModal.value = true
  registrationsLoading.value = true
  try {
    const res = await getOfflineActivityRegistrations(activity.id)
    registrations.value = res.data.data ?? []
  } catch (e: any) {
    ElMessage.error(e.message || '获取活动报名名单失败')
  } finally {
    registrationsLoading.value = false
  }
}

onMounted(fetchActivities)
</script>

<template>
  <div class="page">
    <div class="card">
      <div class="card-header">
        <div>
          <h1 class="page-title">线下省钱活动</h1>
          <p class="page-sub">管理仅在 iOS 端展示的线下省钱活动内容与报名情况</p>
        </div>
        <div class="page-header-actions">
          <el-button type="primary" @click="openCreateModal">
            <template #icon><el-icon><Plus /></el-icon></template>
            新建活动
          </el-button>
        </div>
      </div>

      <div class="card-toolbar">
        <div class="toolbar-left">
          <div class="status-filter">
            <el-select v-model="statusFilter" placeholder="活动状态" clearable style="width: 150px" @change="() => { page = 1; fetchActivities() }">
              <el-option label="草稿" value="draft" />
              <el-option label="已发布" value="published" />
              <el-option label="已关闭" value="closed" />
              <el-option label="已取消" value="cancelled" />
            </el-select>
          </div>
        </div>
        <span class="total-count">共 {{ total }} 条</span>
      </div>

      <div class="table-scroll-x page-table-wrap">
        <el-table
          :data="filteredActivities"
          v-loading="loading"
          stripe
          size="small"
          style="width:100%"
        >
        <el-table-column label="序号" width="60" align="center">
          <template #default="{ $index }">
            <span class="cell-id">{{ (page - 1) * pageSize + $index + 1 }}</span>
          </template>
        </el-table-column>

        <el-table-column label="活动" min-width="220">
          <template #default="{ row }">
            <div class="activity-cell">
              <div class="activity-title">{{ row.title }}</div>
              <div class="activity-sub">{{ row.subtitle || '未填写副标题' }}</div>
            </div>
          </template>
        </el-table-column>

        <el-table-column v-if="!isMobile" label="城市 / 地点" min-width="180">
          <template #default="{ row }">
            <div class="location-cell">
              <div class="location-city">{{ row.city }}</div>
              <div class="location-venue">{{ row.venue }}</div>
            </div>
          </template>
        </el-table-column>

        <el-table-column v-if="!isMobile" label="开始时间" min-width="150">
          <template #default="{ row }">
            <span class="cell-code cell-muted">{{ row.startAt }}</span>
          </template>
        </el-table-column>

        <el-table-column v-if="!isMobile" label="结束时间" min-width="150">
          <template #default="{ row }">
            <span class="cell-code cell-muted">{{ row.endAt }}</span>
          </template>
        </el-table-column>

        <el-table-column label="报名 / 名额" width="110">
          <template #default="{ row }">
            <span class="quota-badge">{{ quotaText(row) }}</span>
          </template>
        </el-table-column>

        <el-table-column label="状态" width="90">
          <template #default="{ row }">
            <span class="status-badge" :class="statusClass(row.status)">{{ statusLabel(row.status) }}</span>
          </template>
        </el-table-column>

        <el-table-column label="操作" :width="isMobile ? 220 : 220" :fixed="isMobile ? false : 'right'">
          <template #default="{ row }">
            <div class="action-btns">
              <el-button size="small" type="primary" plain style="padding:2px 8px;font-size:12px" @click="openRegistrationsModal(row)">报名名单</el-button>
              <el-button size="small" plain style="padding:2px 8px;font-size:12px" @click="openEditModal(row)">编辑</el-button>
              <el-button
                v-if="row.status === 'draft' || row.status === 'published'"
                size="small"
                type="danger"
                plain
                style="padding:2px 8px;font-size:12px"
                @click="handleClose(row)"
              >
                下线
              </el-button>
            </div>
          </template>
        </el-table-column>
        </el-table>
      </div>

      <div class="pagination-bar">
        <el-pagination
          v-model:current-page="page"
          :page-size="pageSize"
          :total="total"
          layout="prev, pager, next"
          background
          small
          @current-change="(p:number) => { page = p; fetchActivities() }"
        />
      </div>
    </div>

    <el-dialog
      v-model="showModal"
      :title="editingId ? '编辑活动' : '新建活动'"
      :width="isMobile ? '100%' : '760px'"
      :fullscreen="isMobile"
      :close-on-click-modal="false"
    >
      <div class="form-grid">
        <div class="field field-span-2">
          <label class="form-label">活动标题</label>
          <el-input v-model="form.title" placeholder="请输入活动标题" />
        </div>
        <div class="field field-span-2">
          <label class="form-label">副标题</label>
          <el-input v-model="form.subtitle" placeholder="一句话描述活动定位" />
        </div>
        <div class="field field-span-2">
          <label class="form-label">活动说明</label>
          <el-input v-model="form.description" type="textarea" :rows="4" placeholder="描述活动规则、适合人群、注意事项" />
        </div>
        <div class="field">
          <label class="form-label">城市</label>
          <el-input v-model="form.city" placeholder="例如：上海" />
        </div>
        <div class="field">
          <label class="form-label">活动地点</label>
          <el-input v-model="form.venue" placeholder="例如：徐汇滨江跑道" />
        </div>
        <div class="field">
          <label class="form-label">开始时间</label>
          <el-input v-model="form.startAt" placeholder="YYYY-MM-DD HH:mm:ss" />
        </div>
        <div class="field">
          <label class="form-label">结束时间</label>
          <el-input v-model="form.endAt" placeholder="YYYY-MM-DD HH:mm:ss" />
        </div>
        <div class="field">
          <label class="form-label">人数上限</label>
          <el-input-number v-model="form.quota" :min="0" :step="1" style="width: 100%" />
        </div>
        <div class="field">
          <label class="form-label">状态</label>
          <el-select v-model="form.status" style="width: 100%">
            <el-option label="草稿" value="draft" />
            <el-option label="已发布" value="published" />
            <el-option label="已关闭" value="closed" />
            <el-option label="已取消" value="cancelled" />
          </el-select>
        </div>
      </div>
      <template #footer>
        <el-button @click="showModal = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="submitForm">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog
      v-model="showRegistrationsModal"
      :title="currentActivity ? `${currentActivity.title} · 报名名单` : '报名名单'"
      :width="isMobile ? '100%' : '720px'"
      :fullscreen="isMobile"
      :close-on-click-modal="false"
    >
      <div class="table-scroll-x registrations-table-wrap">
        <el-table :data="registrations" v-loading="registrationsLoading" stripe size="small">
        <el-table-column prop="nickname" label="昵称" min-width="160" />
        <el-table-column prop="phone" label="手机号" min-width="140" />
        <el-table-column prop="imUserID" label="IM UserID" min-width="180" />
        <el-table-column prop="joinedAt" label="报名时间" min-width="180" />
        </el-table>
      </div>
      <div v-if="!registrationsLoading && registrations.length === 0" class="empty-note">
        暂无报名用户
      </div>
    </el-dialog>
  </div>
</template>

<style scoped>
.page-header-actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 8px;
}

.card-toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  border-bottom: 1px solid #f0f0f0;
}

.page-table-wrap,
.registrations-table-wrap {
  border-bottom: 1px solid #f0f0f0;
}

.toolbar-left {
  display: flex;
  align-items: center;
  gap: 10px;
}

.status-filter {
  display: flex;
  align-items: center;
  gap: 8px;
}

.total-count {
  font-size: 12px;
  color: #94a3b8;
  margin-left: auto;
  white-space: nowrap;
}

.cell-id {
  font-size: 13px;
  color: #94a3b8;
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

.activity-cell {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.activity-title {
  font-size: 13px;
  font-weight: 600;
  color: #0f172a;
}

.activity-sub {
  font-size: 12px;
  color: #94a3b8;
}

.location-cell {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.location-city {
  font-size: 13px;
  font-weight: 600;
  color: #1f2937;
}

.location-venue {
  font-size: 12px;
  color: #94a3b8;
}

.quota-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 78px;
  padding: 0 10px;
  height: 24px;
  border-radius: 999px;
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  color: #475569;
  font-size: 12px;
  font-weight: 600;
}

.status-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 66px;
  height: 24px;
  padding: 0 10px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 600;
  border: 1px solid transparent;
}

.status-active {
  color: #0f766e;
  background: #ecfeff;
  border-color: #99f6e4;
}

.status-draft {
  color: #475569;
  background: #f8fafc;
  border-color: #e2e8f0;
}

.status-closed {
  color: #b45309;
  background: #fffbeb;
  border-color: #fde68a;
}

.status-cancelled {
  color: #dc2626;
  background: #fef2f2;
  border-color: #fecaca;
}

.action-btns {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}

.pagination-bar {
  display: flex;
  justify-content: flex-end;
  padding: 14px 16px 16px;
  border-top: 1px solid #f0f0f0;
}

.form-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
}

.field {
  display: grid;
  gap: 8px;
}

.field-span-2 {
  grid-column: span 2;
}

.form-label {
  font-size: 13px;
  font-weight: 600;
  color: #374151;
}

.empty-note {
  padding: 16px 0 4px;
  color: #6b7280;
  text-align: center;
}

@media (max-width: 900px) {
  .page-header-actions {
    width: 100%;
  }

  .page-header-actions :deep(.el-button) {
    width: 100%;
  }

  .card-toolbar {
    flex-direction: column;
    align-items: flex-start;
  }

  .total-count {
    margin-left: 0;
  }

  .form-grid {
    grid-template-columns: 1fr;
  }

  .field-span-2 {
    grid-column: span 1;
  }
}

@media (max-width: 767px) {
  .toolbar-left,
  .pagination-bar {
    width: 100%;
    flex-direction: column;
    align-items: stretch;
  }

  .status-filter :deep(.el-select) {
    width: 100% !important;
  }

  .action-btns {
    flex-direction: column;
  }

  .action-btns :deep(.el-button) {
    width: 100%;
    margin-left: 0;
  }
}
</style>
