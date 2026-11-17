<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  getInviteCodeList, batchCreateInviteCodes,
  updateInviteCode, regenerateInviteCode, deleteInviteCode,
} from '@/api/inviteCode'
import { getUserOwners } from '@/api/user'
import { useAuthStore } from '@/stores/auth'
import type { InviteCode, InviteCodeStatus, UserOwnerOption } from '@/types'
import { useResponsive } from '@/composables/useResponsive'

const authStore = useAuthStore()
const { isMobile } = useResponsive()
const isAdmin = computed(() => authStore.isAdmin)
const pageSubText = computed(() => {
  if (authStore.isSuperAdmin) return '管理全量邀请码的生成与分配'
  if (authStore.isManager) return '管理我归属范围内的邀请码与注册入口'
  return '管理我的邀请码与注册记录'
})

// ── 字段筛选 ──
interface FilterField {
  key: string; label: string; type: 'text' | 'select'
  options?: { value: string; label: string }[]
}
const FILTER_FIELDS: FilterField[] = [
  { key: 'status', label: '状态', type: 'select', options: [
    { value: 'active',  label: '有效' },
    { value: 'used',    label: '已用完' },
    { value: 'expired', label: '已过期' },
  ]},
  { key: 'owner', label: '归属', type: 'text' },
  { key: 'maxUses', label: '使用上限', type: 'select', options: [
    { value: 'unlimited', label: '不限次数' },
    { value: 'limited',   label: '有次数限制' },
  ]},
]

interface ActiveFilter { key: string; label: string; value: string; display: string }
const activeFilters = ref<ActiveFilter[]>([])
const filterWrapRef = ref<HTMLElement | null>(null)
const dropdownOpen  = ref(false)
const selectedField = ref<FilterField | null>(null)
const filterInput   = ref('')
const filterSelect  = ref('')

function openDropdown() { dropdownOpen.value = true; selectedField.value = null; filterInput.value = ''; filterSelect.value = '' }
function selectField(f: FilterField) { selectedField.value = f; filterInput.value = ''; filterSelect.value = '' }
function confirmFilter() {
  if (!selectedField.value) return
  const f = selectedField.value
  const raw = f.type === 'select' ? filterSelect.value : filterInput.value.trim()
  if (!raw) return
  const display = f.type === 'select' ? (f.options?.find(o => o.value === raw)?.label ?? raw) : raw
  activeFilters.value = activeFilters.value.filter(a => a.key !== f.key)
  activeFilters.value.push({ key: f.key, label: f.label, value: raw, display })
  dropdownOpen.value = false; selectedField.value = null
}
function removeFilter(key: string) { activeFilters.value = activeFilters.value.filter(a => a.key !== key) }
function clearAllFilters() { activeFilters.value = [] }
function onDocMousedown(e: MouseEvent) {
  if (filterWrapRef.value && !filterWrapRef.value.contains(e.target as Node)) dropdownOpen.value = false
}
onMounted(() => document.addEventListener('mousedown', onDocMousedown))
onUnmounted(() => document.removeEventListener('mousedown', onDocMousedown))

// ── 列表状态 ──
const loading    = ref(false)
const codes      = ref<InviteCode[]>([])
const total      = ref(0)
const page       = ref(1)
const pageSize   = ref(10)

const filtered = computed(() => {
  if (!activeFilters.value.length) return codes.value
  return codes.value.filter(c =>
    activeFilters.value.every(f => {
      switch (f.key) {
        case 'status':  return c.status === f.value
        case 'owner':   return String(c.ownerID).includes(f.value) || c.ownerName?.toLowerCase().includes(f.value.toLowerCase())
        case 'maxUses': return f.value === 'unlimited' ? c.maxUses <= 0 : c.maxUses > 0
        default: return true
      }
    })
  )
})

async function fetchCodes() {
  loading.value = true
  try {
    const res = await getInviteCodeList({ page: page.value, pageSize: pageSize.value })
    codes.value = res.data.data?.list ?? []
    total.value = res.data.data?.total ?? 0
  } catch (e: any) { ElMessage.error(e.message || '获取邀请码列表失败') }
  finally { loading.value = false }
}
onMounted(fetchCodes)

// ── 创建 ──
const showCreateModal = ref(false)
const submitting      = ref(false)
const createForm = reactive({ count: 1, maxUses: 0, expiredAt: '', ownerID: 0 })
const ownerOptions = ref<UserOwnerOption[]>([])

const ownerManagementOptions = computed(() =>
  ownerOptions.value.map(owner => {
    const isSelf = owner.id === authStore.user?.id
    const roleLabel = owner.role === 'super_admin' ? '超级管理员' : owner.role === 'admin' ? '管理员' : '助理'
    const selfSuffix = isSelf ? '（我自己）' : ''
    const statusSuffix = owner.status === 'active' ? '' : '（已禁用）'
    return {
      value: owner.id,
      label: `${roleLabel}: ${owner.username}${selfSuffix}${statusSuffix}`,
    }
  })
)

async function loadOwnerOptions() {
  if (!isAdmin.value) {
    ownerOptions.value = []
    return
  }
  try {
    const res = await getUserOwners()
    ownerOptions.value = res.data.data ?? []
  } catch { /* 静默 */ }
}

function openCreateModal() {
  createForm.count = 1; createForm.maxUses = 0; createForm.expiredAt = ''; createForm.ownerID = 0
  showCreateModal.value = true
  loadOwnerOptions()
}

async function handleCreate() {
  submitting.value = true
  try {
    await batchCreateInviteCodes({
      count: createForm.count,
      maxUses: createForm.maxUses,
      expiredAt: createForm.expiredAt || null,
      ownerID: createForm.ownerID || undefined,
    })
    ElMessage.success(`已生成 ${createForm.count} 个邀请码`)
    showCreateModal.value = false
    fetchCodes()
  } catch (e: any) { ElMessage.error(e.message || '生成失败') }
  finally { submitting.value = false }
}

// ── 编辑 ──
const showEditModal    = ref(false)
const codeRegenerating = ref(false)
const editForm = reactive({
  id: 0, code: '', maxUses: 0, status: 'active', codeEdited: false,
})

function openEditModal(row: InviteCode) {
  editForm.id = row.id
  editForm.code = row.code
  editForm.maxUses = row.maxUses <= 0 ? 0 : row.maxUses
  editForm.status = row.status
  editForm.codeEdited = false
  showEditModal.value = true
}

async function handleRegenerateCode() {
  codeRegenerating.value = true
  try {
    const res = await regenerateInviteCode(editForm.id)
    editForm.code = res.data.data?.code ?? editForm.code
    editForm.codeEdited = true
    ElMessage.success('邀请码已随机生成')
  } catch (e: any) { ElMessage.error(e.message || '生成失败') }
  finally { codeRegenerating.value = false }
}

async function handleSaveEdit() {
  if (!editForm.code.trim()) { ElMessage.warning('邀请码不能为空'); return }
  submitting.value = true
  try {
    await updateInviteCode(editForm.id, {
      maxUses: editForm.maxUses,
      status: editForm.status,
      code: editForm.codeEdited ? editForm.code.trim() : undefined,
    })
    ElMessage.success('已保存')
    showEditModal.value = false
    fetchCodes()
  } catch (e: any) { ElMessage.error(e.message || '保存失败') }
  finally { submitting.value = false }
}

// ── 删除 ──
async function handleDelete(code: InviteCode) {
  await ElMessageBox.confirm(`确认删除邀请码 "${code.code}"？`, '确认操作', { type: 'warning' })
  try { await deleteInviteCode(code.id); ElMessage.success('已删除'); fetchCodes() }
  catch (e: any) { ElMessage.error(e.message || '操作失败') }
}

// ── 工具 ──
async function copyCode(code: string) {
  await navigator.clipboard.writeText(code)
  ElMessage.success('已复制')
}
function statusType(s: InviteCodeStatus) { return { active: 'success', used: 'info', expired: 'danger' }[s] as any }
function statusLabel(s: InviteCodeStatus) { return { active: '有效', used: '已用完', expired: '已过期' }[s] ?? s }
</script>

<template>
  <div class="page">
    <div class="card">
      <div class="card-header">
        <div>
          <h1 class="page-title">邀请码管理</h1>
          <p class="page-sub">{{ pageSubText }}</p>
        </div>
        <div class="page-header-actions">
          <el-button type="primary" @click="openCreateModal">
            <template #icon><el-icon><Plus /></el-icon></template>生成邀请码
          </el-button>
        </div>
      </div>

      <div class="card-toolbar">
        <div ref="filterWrapRef" class="filter-wrap">
          <div class="filter-box" @click="openDropdown">
            <svg class="filter-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/>
            </svg>
            <span v-for="f in activeFilters" :key="f.key" class="filter-tag">
              <span class="filter-tag-key">{{ f.label }}:</span>{{ f.display }}
              <button class="filter-tag-remove" @click.stop="removeFilter(f.key)">×</button>
            </span>
            <span v-if="activeFilters.length === 0" class="filter-placeholder">添加筛选条件…</span>
          </div>
          <div v-if="dropdownOpen" class="filter-dropdown">
            <template v-if="!selectedField">
              <div class="filter-dropdown-head">筛选条件</div>
              <ul>
                <li v-for="f in FILTER_FIELDS" :key="f.key" class="filter-field-item" @click.stop="selectField(f)">
                  <div class="filter-field-icon">
                    <svg width="12" height="12" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 4a1 1 0 011-1h16a1 1 0 011 1v2a1 1 0 01-.293.707L13 13.414V19a1 1 0 01-.553.894l-4 2A1 1 0 017 21v-7.586L3.293 6.707A1 1 0 013 6V4z"/>
                    </svg>
                  </div>
                  <span>{{ f.label }}</span>
                  <svg class="filter-chevron" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"/>
                  </svg>
                </li>
              </ul>
            </template>
            <template v-else>
              <div class="filter-value-head">
                <button class="filter-back-btn" @click.stop="selectedField = null">
                  <svg width="16" height="16" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/>
                  </svg>
                </button>
                <span>{{ selectedField.label }}</span>
              </div>
              <div class="filter-value-body">
                <input v-if="selectedField.type === 'text'" v-model="filterInput"
                  class="filter-text-input" :placeholder="`输入${selectedField.label}…`"
                  autofocus @keyup.enter.stop="confirmFilter" @click.stop />
                <ul v-else class="filter-option-list">
                  <li v-for="opt in selectedField.options" :key="opt.value"
                    class="filter-option" :class="{ selected: filterSelect === opt.value }"
                    @click.stop="filterSelect = opt.value">
                    <div class="filter-radio" :class="{ active: filterSelect === opt.value }"></div>
                    <span>{{ opt.label }}</span>
                  </li>
                </ul>
                <button class="filter-confirm-btn" @click.stop="confirmFilter">确认</button>
              </div>
            </template>
          </div>
        </div>
        <button v-if="activeFilters.length > 0" class="clear-btn" @click="clearAllFilters">清除筛选</button>
        <span class="total-count">共 {{ total }} 条</span>
      </div>

      <div class="table-scroll-x page-table-wrap">
        <el-table :data="filtered" v-loading="loading" stripe size="small" style="width:100%">
        <el-table-column label="序号" width="60" align="center">
          <template #default="{ $index }">
            <span class="cell-id">{{ (page - 1) * pageSize + $index + 1 }}</span>
          </template>
        </el-table-column>

        <el-table-column label="邀请码" min-width="180">
          <template #default="{ row }">
            <div class="code-cell">
              <span class="cell-code">{{ row.code }}</span>
              <button class="copy-btn" @click="copyCode(row.code)">
                <svg width="14" height="14" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"/>
                </svg>
              </button>
            </div>
          </template>
        </el-table-column>

        <el-table-column v-if="isAdmin && !isMobile" label="归属" min-width="130">
          <template #default="{ row }">
            <span class="cell-owner">
              <span class="owner-name">{{ row.ownerName || '—' }}</span>
              <span class="owner-id">#{{ row.ownerID }}</span>
            </span>
          </template>
        </el-table-column>

        <el-table-column label="使用次数" width="110">
          <template #default="{ row }">
            <span class="cell-code cell-muted">
              {{ row.usedCount }} / <span v-if="row.maxUses <= 0" style="font-size:17px;line-height:1;vertical-align:middle">∞</span><template v-else>{{ row.maxUses }}</template>
            </span>
          </template>
        </el-table-column>

        <el-table-column v-if="!isMobile" label="过期时间" min-width="150">
          <template #default="{ row }">
            <span class="cell-code cell-muted">{{ row.expiredAt ?? '永久有效' }}</span>
          </template>
        </el-table-column>

        <el-table-column label="状态" width="90">
          <template #default="{ row }">
            <span class="status-badge" :class="`status-${row.status}`">{{ statusLabel(row.status) }}</span>
          </template>
        </el-table-column>

        <el-table-column v-if="!isMobile" label="创建时间" min-width="155">
          <template #default="{ row }">
            <span class="cell-code cell-xs">{{ row.createdAt }}</span>
          </template>
        </el-table-column>

        <el-table-column label="操作" :width="isMobile ? 144 : 130" :fixed="isMobile ? false : 'right'">
          <template #default="{ row }">
            <div class="action-btns">
              <el-button size="small" plain style="padding:2px 8px;font-size:12px" @click="openEditModal(row)">编辑</el-button>
              <el-button size="small" type="danger" plain style="padding:2px 8px;font-size:12px" @click="handleDelete(row)">删除</el-button>
            </div>
          </template>
        </el-table-column>
        </el-table>
      </div>

      <div class="pagination-bar">
        <el-pagination v-model:current-page="page" :page-size="pageSize" :total="total"
          layout="prev, pager, next" background small
          @current-change="(p) => { page = p; fetchCodes() }" />
      </div>
    </div>

    <!-- 创建 Modal -->
    <el-dialog
      v-model="showCreateModal"
      title="生成邀请码"
      :width="isMobile ? '100%' : '380px'"
      :fullscreen="isMobile"
      :close-on-click-modal="false"
    >
      <div class="modal-body">
        <div v-if="isAdmin" class="form-field">
          <label class="form-label">归属人 <span class="label-hint">（留空则归属自己）</span></label>
          <el-select v-model="createForm.ownerID" placeholder="归属当前管理员" style="width:100%" clearable>
            <el-option v-for="owner in ownerManagementOptions" :key="owner.value" :value="owner.value" :label="owner.label" />
          </el-select>
        </div>
        <div class="form-field">
          <label class="form-label">批量数量</label>
          <el-input-number v-model="createForm.count" :min="1" :max="200" style="width:100%" />
        </div>
        <div class="form-field">
          <label class="form-label">最大使用次数</label>
          <el-input-number v-model="createForm.maxUses" :min="0" style="width:100%" />
          <span class="form-hint">0 = 不限次数</span>
        </div>
        <div class="form-field">
          <label class="form-label">到期时间 <span class="label-hint">（留空表示永不过期）</span></label>
          <el-date-picker v-model="createForm.expiredAt" type="datetime" placeholder="选择到期时间" style="width:100%" />
        </div>
      </div>
      <template #footer>
        <el-button @click="showCreateModal = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleCreate">生成</el-button>
      </template>
    </el-dialog>

    <!-- 编辑 Modal -->
    <el-dialog
      v-model="showEditModal"
      title="编辑邀请码"
      :width="isMobile ? '100%' : '460px'"
      :fullscreen="isMobile"
      :close-on-click-modal="false"
    >
      <div class="modal-body-lg">
        <div class="modal-row">
          <span class="modal-label">邀请码</span>
          <el-input v-model="editForm.code" class="code-input" placeholder="输入自定义邀请码"
            @input="editForm.codeEdited = true" />
          <el-button size="small" plain type="primary" :loading="codeRegenerating"
            style="flex-shrink:0" @click="handleRegenerateCode">
            <svg width="13" height="13" fill="none" stroke="currentColor" viewBox="0 0 24 24" style="margin-right:3px">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
            </svg>随机
          </el-button>
        </div>
        <div class="modal-row">
          <span class="modal-label">最大使用次数</span>
          <el-input-number v-model="editForm.maxUses" :min="0" :max="9999"
            controls-position="both" style="flex:1" />
          <span style="font-size:12px;color:#94a3b8;margin-left:8px">0 = 不限</span>
        </div>
        <div class="modal-row">
          <span class="modal-label">启用状态</span>
          <div class="switch-row">
            <span class="switch-label" :class="{ active: editForm.status !== 'active' }">停用</span>
            <el-switch v-model="editForm.status" active-value="active" inactive-value="expired" />
            <span class="switch-label" :class="{ active: editForm.status === 'active' }">启用</span>
          </div>
        </div>
      </div>
      <template #footer>
        <el-button @click="showEditModal = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSaveEdit">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.page-header-actions { display:flex; flex-wrap:wrap; justify-content:flex-end; gap:8px; }
.card-toolbar { display: flex; align-items: center; gap: 8px; padding: 12px 16px; border-bottom: 1px solid #f0f0f0; }
.page-table-wrap { border-bottom: 1px solid #f0f0f0; }

.filter-wrap { position: relative; flex: 1; max-width: 560px; }
.filter-box {
  display: flex; flex-wrap: wrap; align-items: center; gap: 6px;
  min-height: 32px; padding: 4px 12px;
  border: 1px solid #f0f0f0; border-radius: 8px; background: #fff; cursor: text; transition: border-color 0.15s;
}
.filter-box:hover { border-color: rgba(22,119,255,0.4); }
.filter-icon { width: 14px; height: 14px; color: #94a3b8; flex-shrink: 0; }
.filter-tag { display: inline-flex; align-items: center; gap: 3px; padding: 2px 8px; background: rgba(22,119,255,0.1); color: #1677ff; font-size: 11px; font-weight: 500; border-radius: 4px; }
.filter-tag-key { color: rgba(22,119,255,0.6); margin-right: 2px; }
.filter-tag-remove { margin-left: 2px; background: none; border: none; color: rgba(22,119,255,0.5); cursor: pointer; font-size: 14px; line-height: 1; padding: 0; }
.filter-tag-remove:hover { color: #1677ff; }
.filter-placeholder { color: #94a3b8; font-size: 13px; user-select: none; }

.filter-dropdown { position: absolute; top: calc(100% + 6px); left: 0; width: 288px; background: #fff; border: 1px solid #f0f0f0; border-radius: 10px; box-shadow: 0 10px 15px -3px rgba(0,0,0,0.1); z-index: 50; overflow: hidden; }
.filter-dropdown-head { font-size: 10px; font-weight: 600; color: #94a3b8; text-transform: uppercase; letter-spacing: 0.06em; padding: 12px 12px 6px; }
.filter-field-item { display: flex; align-items: center; gap: 10px; padding: 8px 12px; cursor: pointer; font-size: 14px; color: #1e293b; transition: background 0.1s; }
.filter-field-item:hover { background: #f8fafc; }
.filter-field-icon { width: 20px; height: 20px; border-radius: 4px; background: rgba(22,119,255,0.1); display: flex; align-items: center; justify-content: center; color: #1677ff; flex-shrink: 0; }
.filter-chevron { width: 14px; height: 14px; color: #d1d5db; margin-left: auto; }
.filter-value-head { display: flex; align-items: center; gap: 8px; padding: 10px 12px 8px; border-bottom: 1px solid #f0f0f0; font-size: 14px; font-weight: 500; color: #1e293b; }
.filter-back-btn { background: none; border: none; color: #94a3b8; cursor: pointer; display: flex; }
.filter-back-btn:hover { color: #64748b; }
.filter-value-body { padding: 12px; display: flex; flex-direction: column; gap: 8px; }
.filter-text-input { width: 100%; height: 32px; padding: 0 12px; font-size: 14px; border: 1px solid #f0f0f0; border-radius: 6px; outline: none; }
.filter-text-input:focus { border-color: rgba(22,119,255,0.6); }
.filter-option-list { display: flex; flex-direction: column; gap: 2px; max-height: 240px; overflow-y: auto; padding-right: 2px; }
.filter-option { display: flex; align-items: center; gap: 8px; padding: 6px 8px; border-radius: 6px; cursor: pointer; font-size: 14px; color: #1e293b; transition: background 0.1s; }
.filter-option:hover { background: #f8fafc; }
.filter-option.selected { background: rgba(22,119,255,0.08); color: #1677ff; }
.filter-radio { width: 14px; height: 14px; border-radius: 50%; border: 2px solid #d1d5db; flex-shrink: 0; transition: all 0.1s; }
.filter-radio.active { border-color: #1677ff; background: #1677ff; }
.filter-confirm-btn { width: 100%; height: 32px; border-radius: 6px; background: #1677ff; color: #fff; font-size: 12px; font-weight: 500; border: none; cursor: pointer; }
.filter-confirm-btn:hover { background: #4096ff; }

.clear-btn { font-size: 12px; color: #94a3b8; background: none; border: none; cursor: pointer; white-space: nowrap; }
.clear-btn:hover { color: #64748b; }
.total-count { font-size: 12px; color: #94a3b8; margin-left: auto; white-space: nowrap; }

/* ── Table Cells ── */
.cell-id   { font-size: 13px; color: #94a3b8; }
.cell-code { font-family: 'Fira Code', 'Courier New', monospace; font-size: 13px; color: #1e293b; }
.cell-muted { color: #64748b; }
.cell-xs   { font-size: 12px; color: #94a3b8; }

.code-cell { display: flex; align-items: center; gap: 6px; }
.copy-btn {
  display: flex; align-items: center; justify-content: center;
  width: 20px; height: 20px; border: none; background: none;
  color: #d1d5db; cursor: pointer; transition: color 0.15s; padding: 0; flex-shrink: 0;
}
.copy-btn:hover { color: #1677ff; }

.cell-owner { display: inline-flex; align-items: center; gap: 5px; }
.owner-id   { font-family: 'Fira Code', 'Courier New', monospace; font-size: 12px; color: #94a3b8; }
.owner-name { font-size: 13px; font-weight: 500; color: #1677ff; }

/* ── Status Badge ── */
.status-badge {
  display: inline-flex; align-items: center;
  font-size: 12px; font-weight: 500; padding: 0 9px; height: 24px;
  border-radius: 6px; border: 1px solid transparent;
}
.status-active  { color: #52c41a; border-color: #b7eb8f; background: #f6ffed; }
.status-used    { color: #8c8c8c; border-color: #d9d9d9; background: #fafafa; }
.status-expired { color: #ff4d4f; border-color: #ffa39e; background: #fff2f0; }

.action-btns { display: flex; align-items: center; gap: 6px; }
.pagination-bar { display: flex; justify-content: flex-end; padding: 12px 16px; border-top: 1px solid #f0f0f0; }

/* ── Modal ── */
.modal-body { display: flex; flex-direction: column; gap: 16px; padding-top: 4px; }
.modal-body-lg { display: flex; flex-direction: column; gap: 20px; padding: 8px 0; }
.form-field { display: flex; flex-direction: column; gap: 6px; }
.form-label { font-size: 12px; font-weight: 500; color: #475569; }
.form-hint  { font-size: 12px; color: #94a3b8; margin-top: 2px; }
.label-hint { font-size: 11px; color: #94a3b8; font-weight: 400; }
.modal-row  { display: flex; align-items: center; gap: 16px; }
.modal-label { font-size: 13px; color: #64748b; width: 80px; flex-shrink: 0; }
.code-input { flex: 1; font-family: 'Fira Code', 'Courier New', monospace; }
.switch-row { display: flex; align-items: center; gap: 12px; }
.switch-label { font-size: 13px; color: #94a3b8; transition: color 0.15s; }
.switch-label.active { color: #1677ff; font-weight: 500; }

@media (max-width: 767px) {
  .page-header-actions {
    width: 100%;
  }

  .page-header-actions :deep(.el-button) {
    width: 100%;
  }

  .card-toolbar,
  .pagination-bar {
    flex-direction: column;
    align-items: stretch;
  }

  .filter-wrap {
    max-width: none;
  }

  .clear-btn {
    width: 100%;
    min-height: 40px;
  }

  .total-count {
    margin-left: 0;
  }

  .action-btns {
    flex-direction: column;
  }

  .modal-row {
    flex-direction: column;
    align-items: stretch;
  }

  .modal-label {
    width: auto;
  }
}
</style>
