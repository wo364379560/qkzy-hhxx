<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { MoreFilled } from '@element-plus/icons-vue'
import {
  getAssistantList, createAssistant, deleteAssistant,
  generateAssistantInviteCode, updateInviteCode, regenerateInviteCode,
  banAssistant, unbanAssistant, resetAssistantPassword,
  resetAssistantTOTP, updateAssistantProfile, resolveAssistantLoginCooldown,
} from '@/api/assistant'
import ChatQuietHoursPolicyDialog from '@/components/ChatQuietHoursPolicyDialog.vue'
import type { AssistantDetail } from '@/types'
import IPWhitelistRulesDialog from '@/components/IPWhitelistRulesDialog.vue'
import { useAuthStore } from '@/stores/auth'
import { useResponsive } from '@/composables/useResponsive'

const authStore = useAuthStore()
const { isMobile } = useResponsive()
const pageSubText = computed(() => authStore.isSuperAdmin ? '全量助理账号及其邀请码一览' : '我名下助理账号及其邀请码一览')

// ── 字段筛选 ──
interface FilterField {
  key: string; label: string; type: 'text' | 'select'
  options?: { value: string; label: string }[]
}
const FILTER_FIELDS: FilterField[] = [
  { key: 'username', label: '用户名', type: 'text' },
  { key: 'status', label: '账号状态', type: 'select', options: [
    { value: 'active', label: '正常' },
    { value: 'banned', label: '已禁用' },
  ]},
  { key: 'inviteCode', label: '邀请码', type: 'text' },
  { key: 'inviteCodeStatus', label: '码状态', type: 'select', options: [
    { value: 'active',  label: '启用' },
    { value: 'used',    label: '已用完' },
    { value: 'expired', label: '已过期' },
  ]},
  { key: 'hasInviteCode', label: '有无邀请码', type: 'select', options: [
    { value: 'yes', label: '有邀请码' },
    { value: 'no',  label: '无邀请码' },
  ]},
  { key: 'inviteCodeMax', label: '使用上限', type: 'select', options: [
    { value: 'unlimited', label: '不限次数' },
    { value: 'limited',   label: '有次数限制' },
  ]},
  { key: 'userCount', label: '用户总数', type: 'select', options: [
    { value: 'has',  label: '有用户' },
    { value: 'none', label: '无用户' },
  ]},
]

interface ActiveFilter { key: string; label: string; value: string; display: string }
const activeFilters  = ref<ActiveFilter[]>([])
const filterWrapRef  = ref<HTMLElement | null>(null)
const dropdownOpen   = ref(false)
const selectedField  = ref<FilterField | null>(null)
const filterInput    = ref('')
const filterSelect   = ref('')

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

// ── 弹窗状态 ──
const showCreateModal = ref(false)
const showResetModal = ref(false)
const showEditProfileModal = ref(false)
const showEditCodeModal = ref(false)
const showIPWhitelistModal = ref(false)
const showPolicyModal = ref(false)
const loading = ref(false)
const submitting = ref(false)
const profileSaving = ref(false)
const codeRegenerating = ref(false)

const createForm = reactive({ displayName: '', username: '', password: '' })
const resetForm = reactive({ id: 0, username: '', password: '' })
const profileForm = reactive({ id: 0, username: '', displayName: '' })
const editCodeForm = reactive({
  assistantName: '',
  codeId: 0,
  code: '',
  maxUses: 0,
  status: 'active',
  codeEdited: false,
})

const assistants = ref<AssistantDetail[]>([])
const selectedAssistant = ref<AssistantDetail | null>(null)
const selectedPolicyAssistant = ref<AssistantDetail | null>(null)
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)

const filtered = computed(() => {
  if (!activeFilters.value.length) return assistants.value
  return assistants.value.filter(a => {
    return activeFilters.value.every(f => {
      const val = f.value.toLowerCase()
      switch (f.key) {
        case 'username':         return a.username?.toLowerCase().includes(val)
        case 'status':           return a.status === f.value
        case 'inviteCode':       return a.inviteCode?.toLowerCase().includes(val)
        case 'inviteCodeStatus': return a.inviteCodeStatus === f.value
        case 'hasInviteCode':    return f.value === 'yes' ? !!a.inviteCode : !a.inviteCode
        case 'inviteCodeMax':    return f.value === 'unlimited' ? a.inviteCodeMax <= 0 : a.inviteCodeMax > 0
        case 'userCount':        return f.value === 'has' ? a.userCount > 0 : a.userCount === 0
        default: return true
      }
    })
  })
})

function maxUsesLabel(max: number) {
  return max <= 0 || max === -1 ? '∞' : String(max)
}

function parentAdminLabel(row: AssistantDetail) {
  if (!row.parentAdminName) return '—'
  if (row.parentAdminRole === 'super_admin') return `超级管理员 · ${row.parentAdminName}`
  if (row.parentAdminRole === 'admin') return `管理员 · ${row.parentAdminName}`
  return row.parentAdminName
}

function assistantDisplayName(row: AssistantDetail) {
  return row.displayName?.trim() || '未设置'
}

function accountStatusClass(row: AssistantDetail) {
  if (row.loginCooldown) return 'status-cooldown'
  return row.status === 'active' ? 'status-active' : 'status-banned'
}

function accountStatusText(row: AssistantDetail) {
  if (row.loginCooldown) return '冷却中'
  return row.status === 'active' ? '正常' : '已禁用'
}

function loginCooldownDesc(row: AssistantDetail) {
  const parts = ['清除登录错误冷却，允许该助理立即重新登录']
  if (row.loginCooldownUntil) parts.push(`冷却至 ${row.loginCooldownUntil}`)
  return parts.join('；')
}

async function copyCode(code: string) {
  await navigator.clipboard.writeText(code)
  ElMessage.success('已复制')
}

async function fetchAssistants() {
  loading.value = true
  try {
    const res = await getAssistantList({ page: page.value, pageSize: pageSize.value })
    assistants.value = res.data.data?.list ?? []
    total.value = res.data.data?.total ?? 0
  } catch (e: any) {
    ElMessage.error(e.message || '获取助理列表失败')
  } finally {
    loading.value = false
  }
}

onMounted(fetchAssistants)

async function handleCreate() {
  if (!createForm.username.trim() || !createForm.password) {
    ElMessage.warning('请填写用户名和密码')
    return
  }
  if (createForm.username.trim().length < 3) { ElMessage.warning('用户名至少 3 位'); return }
  if (createForm.password.length < 8) { ElMessage.warning('密码至少 8 位'); return }
  submitting.value = true
  try {
    await createAssistant({
      username: createForm.username,
      password: createForm.password,
      display_name: createForm.displayName.trim(),
    })
    ElMessage.success('助理已创建')
    showCreateModal.value = false
    createForm.displayName = ''
    createForm.username = ''
    createForm.password = ''
    fetchAssistants()
  } catch (e: any) {
    ElMessage.error(e.message || '创建失败')
  } finally {
    submitting.value = false
  }
}

async function handleGenerateCode(row: AssistantDetail) {
  try {
    await generateAssistantInviteCode(row.id)
    ElMessage.success('邀请码已生成')
    fetchAssistants()
  } catch (e: any) {
    ElMessage.error(e.message || '生成失败')
  }
}

function openEditCodeModal(row: AssistantDetail) {
  editCodeForm.assistantName = row.username
  editCodeForm.codeId = row.inviteCodeID!
  editCodeForm.code = row.inviteCode!
  editCodeForm.maxUses = row.inviteCodeMax > 0 ? row.inviteCodeMax : 0
  editCodeForm.status = row.inviteCodeStatus ?? 'active'
  editCodeForm.codeEdited = false
  showEditCodeModal.value = true
}

async function handleRegenerateCode() {
  codeRegenerating.value = true
  try {
    const res = await regenerateInviteCode(editCodeForm.codeId)
    editCodeForm.code = res.data.data?.code ?? editCodeForm.code
    editCodeForm.codeEdited = true
    ElMessage.success('邀请码已随机生成')
  } catch (e: any) {
    ElMessage.error(e.message || '生成失败')
  } finally {
    codeRegenerating.value = false
  }
}

async function handleSaveEditCode() {
  if (!editCodeForm.code.trim()) { ElMessage.warning('邀请码不能为空'); return }
  submitting.value = true
  try {
    await updateInviteCode(editCodeForm.codeId, {
      maxUses: editCodeForm.maxUses,
      status: editCodeForm.status,
      code: editCodeForm.codeEdited ? editCodeForm.code.trim() : undefined,
    })
    ElMessage.success('已保存')
    showEditCodeModal.value = false
    fetchAssistants()
  } catch (e: any) {
    ElMessage.error(e.message || '保存失败')
  } finally {
    submitting.value = false
  }
}

async function handleToggleBan(row: AssistantDetail) {
  const isBanned = row.status !== 'active'
  const action = isBanned ? '启用' : '禁用'
  await ElMessageBox.confirm(`确认${action}助理 "${row.username}"？`, '确认操作', {
    type: isBanned ? 'info' : 'warning',
  })
  try {
    const res = isBanned ? await unbanAssistant(row.id) : await banAssistant(row.id)
    const affectedUserCount = res.data.data?.affectedUserCount ?? 0
    ElMessage.success(
      isBanned
        ? `已${action}，名下 ${affectedUserCount} 个账户恢复可登录`
        : `已${action}，名下 ${affectedUserCount} 个账户将被禁止登录`,
    )
    fetchAssistants()
  } catch (e: any) {
    ElMessage.error(e.message || '操作失败')
  }
}

async function handleDelete(row: AssistantDetail) {
  await ElMessageBox.confirm(
    `确认删除助理 "${row.username}"？删除后不可恢复。若名下仍有关联用户、群组或邀请码，将无法删除。`,
    '删除助理',
    { type: 'warning', confirmButtonText: '删除' },
  )
  try {
    await deleteAssistant(row.id)
    ElMessage.success('助理已删除')
    fetchAssistants()
  } catch (e: any) {
    ElMessage.error(e.message || '删除失败')
  }
}

function openPolicyModal(row: AssistantDetail) {
  selectedPolicyAssistant.value = row
  showPolicyModal.value = true
}

function openEditProfileModal(row: AssistantDetail) {
  profileForm.id = row.id
  profileForm.username = row.username
  profileForm.displayName = row.displayName || ''
  showEditProfileModal.value = true
}

function handleRowAction(command: { action: string; row: AssistantDetail }) {
  const { action, row } = command
  switch (action) {
    case 'editInviteCode':
      if (row.inviteCode) openEditCodeModal(row)
      else void handleGenerateCode(row)
      return
    case 'toggleBan':
      handleToggleBan(row)
      return
    case 'resetPassword':
      openResetModal(row)
      return
    case 'editProfile':
      openEditProfileModal(row)
      return
    case 'ipWhitelist':
      openIPWhitelistModal(row)
      return
    case 'quietHours':
      openPolicyModal(row)
      return
    case 'resetTotp':
      handleResetTOTP(row)
      return
    case 'resolveLoginCooldown':
      handleResolveLoginCooldown(row)
      return
    case 'delete':
      handleDelete(row)
      return
    default:
      return
  }
}

function openResetModal(row: AssistantDetail) {
  resetForm.id = row.id
  resetForm.username = row.username
  resetForm.password = ''
  showResetModal.value = true
}

function openIPWhitelistModal(row: AssistantDetail) {
  selectedAssistant.value = row
  showIPWhitelistModal.value = true
}

async function handleResetPassword() {
  if (resetForm.password.length < 8) { ElMessage.warning('密码至少 8 位'); return }
  submitting.value = true
  try {
    await resetAssistantPassword(resetForm.id, resetForm.password)
    ElMessage.success('密码已重置')
    showResetModal.value = false
  } catch (e: any) {
    ElMessage.error(e.message || '重置失败')
  } finally {
    submitting.value = false
  }
}

async function handleSaveProfile() {
  if (!profileForm.displayName.trim()) {
    ElMessage.warning('请填写昵称')
    return
  }
  profileSaving.value = true
  try {
    await updateAssistantProfile(profileForm.id, { username: profileForm.username, display_name: profileForm.displayName.trim() })
    ElMessage.success('账号资料已更新')
    showEditProfileModal.value = false
    fetchAssistants()
  } catch (e: any) {
    ElMessage.error(e.message || '保存失败')
  } finally {
    profileSaving.value = false
  }
}

async function handleResetTOTP(row: AssistantDetail) {
  await ElMessageBox.confirm(`确认重置助理 "${row.username}" 的双因素认证？旧绑定、恢复码和登录状态将失效；全局 2FA 开启时，下次登录必须重新绑定。`, '确认操作', { type: 'warning' })
  try {
    await resetAssistantTOTP(row.id)
    ElMessage.success('已重置 2FA')
    fetchAssistants()
  } catch (e: any) {
    ElMessage.error(e.message || '重置失败')
  }
}

async function handleResolveLoginCooldown(row: AssistantDetail) {
  const suffix = row.loginCooldownUntil ? `当前冷却至 ${row.loginCooldownUntil}。` : ''
  await ElMessageBox.confirm(
    `确认解除助理 "${row.username}" 的登录冷却？${suffix}`,
    '解除登录冷却',
    { type: 'warning', confirmButtonText: '解除' },
  )
  try {
    await resolveAssistantLoginCooldown(row.id)
    ElMessage.success('登录冷却已解除')
    fetchAssistants()
  } catch (e: any) {
    ElMessage.error(e.message || '解除失败')
  }
}
</script>

<template>
  <div class="page">

    <!-- Table Card -->
    <div class="card">
      <div class="card-header">
        <div>
          <h1 class="page-title">助理</h1>
          <p class="page-sub">{{ pageSubText }}</p>
        </div>
        <div class="page-header-actions">
          <el-button type="primary" @click="showCreateModal = true">
            <template #icon><el-icon><Plus /></el-icon></template>
            新增助理
          </el-button>
        </div>
      </div>
      <!-- Filter Toolbar -->
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
        <el-table
          :data="filtered"
          v-loading="loading"
          stripe
          size="small"
          style="width: 100%"
        >
        <el-table-column label="序号" width="60" align="center">
          <template #default="{ $index }">
            <span class="cell-id">{{ (page - 1) * pageSize + $index + 1 }}</span>
          </template>
        </el-table-column>

        <el-table-column label="昵称 / 用户名" min-width="150">
          <template #default="{ row }">
            <div class="account-cell">
              <span class="cell-username">{{ assistantDisplayName(row) }}</span>
              <span class="cell-muted cell-xs">{{ row.username }}</span>
            </div>
          </template>
        </el-table-column>

        <el-table-column v-if="!isMobile" label="归属管理员" min-width="150">
          <template #default="{ row }">
            <span v-if="row.parentAdminName" class="cell-muted">{{ parentAdminLabel(row) }}</span>
            <span v-else class="cell-empty">—</span>
          </template>
        </el-table-column>

        <el-table-column label="账号状态" width="100">
          <template #default="{ row }">
            <span class="status-badge" :class="accountStatusClass(row)" :title="row.loginCooldownUntil || ''">
              {{ accountStatusText(row) }}
            </span>
          </template>
        </el-table-column>

        <el-table-column v-if="!isMobile" label="邀请码" min-width="140">
          <template #default="{ row }">
            <span v-if="!row.inviteCode" class="cell-empty">—</span>
            <div v-else class="code-cell">
              <span class="cell-code">{{ row.inviteCode }}</span>
              <button class="copy-btn" @click="copyCode(row.inviteCode)">
                <svg width="14" height="14" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"/>
                </svg>
              </button>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="码状态" width="90">
          <template #default="{ row }">
            <span v-if="!row.inviteCodeStatus" class="cell-empty">—</span>
            <span
              v-else
              class="status-badge"
              :class="row.inviteCodeStatus === 'active' ? 'status-active' : 'status-inactive'"
            >{{ row.inviteCodeStatus === 'active' ? '启用' : row.inviteCodeStatus === 'used' ? '已用完' : '已过期' }}</span>
          </template>
        </el-table-column>

        <el-table-column v-if="!isMobile" label="已用 / 上限" width="100">
          <template #default="{ row }">
            <span v-if="!row.inviteCode" class="cell-empty">—</span>
            <span v-else class="cell-code cell-muted">
              {{ row.inviteCodeUsed }} / <span v-if="row.inviteCodeMax <= 0" style="font-size:17px !important;line-height:1;vertical-align:middle">∞</span><template v-else>{{ row.inviteCodeMax }}</template>
            </span>
          </template>
        </el-table-column>

        <el-table-column label="用户总数" width="80">
          <template #default="{ row }">
            <span class="count-bubble" :class="row.userCount > 0 ? 'count-active' : 'count-empty'">
              {{ row.userCount }}
            </span>
          </template>
        </el-table-column>

        <el-table-column v-if="!isMobile" label="2FA" width="80">
          <template #default="{ row }">
            <span class="status-badge" :class="row.totpEnabled ? 'status-active' : 'status-inactive'">
              {{ row.totpEnabled ? '已绑定' : '未绑定' }}
            </span>
          </template>
        </el-table-column>

        <el-table-column v-if="!isMobile" label="IP 白名单" width="90">
          <template #default="{ row }">
            <span class="status-badge" :class="row.ipWhitelistEnabled ? 'status-active' : 'status-inactive'">
              {{ row.ipWhitelistEnabled ? '已启用' : '未启用' }}
            </span>
          </template>
        </el-table-column>

        <el-table-column v-if="!isMobile" label="创建时间" min-width="160">
          <template #default="{ row }">
            <span class="cell-code cell-muted cell-xs">{{ row.createdAt }}</span>
          </template>
        </el-table-column>

        <el-table-column label="操作" :width="isMobile ? 184 : 360" :fixed="isMobile ? false : 'right'">
          <template #default="{ row }">
            <div class="action-btns">
              <el-button
                size="small"
                plain
                type="warning"
                style="padding: 2px 8px; font-size: 12px;"
                @click="handleToggleBan(row)"
              >{{ row.status === 'active' ? '禁用' : '启用' }}</el-button>
              <el-dropdown
                trigger="click"
                popper-class="assistant-action-dropdown"
                max-height="60vh"
                :popper-options="{ modifiers: [{ name: 'preventOverflow', options: { altAxis: true, padding: 12, tether: false } }] }"
                @command="handleRowAction"
              >
                <el-button size="small" plain class="more-btn">
                  更多
                  <el-icon class="more-btn-icon"><MoreFilled /></el-icon>
                </el-button>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item :command="{ action: 'editInviteCode', row }">
                      <div class="dropdown-item">
                        <div class="dropdown-item-title" :class="row.inviteCode ? 'dropdown-item-default' : 'dropdown-item-primary'">
                          {{ row.inviteCode ? '编辑邀请码' : '生成邀请码' }}
                        </div>
                        <div class="dropdown-item-desc">
                          {{ row.inviteCode ? '维护邀请码内容、状态和使用上限' : '为该助理生成新的邀请码' }}
                        </div>
                      </div>
                    </el-dropdown-item>
                    <el-dropdown-item :command="{ action: 'toggleBan', row }">
                      <div class="dropdown-item">
                        <div class="dropdown-item-title" :class="row.status === 'active' ? 'dropdown-item-warning' : 'dropdown-item-success'">
                          {{ row.status === 'active' ? '禁用' : '启用' }}
                        </div>
                        <div class="dropdown-item-desc">
                          {{ row.status === 'active' ? '禁止该助理继续登录和管理' : '恢复该助理正常使用' }}
                        </div>
                      </div>
                    </el-dropdown-item>
                    <el-dropdown-item :command="{ action: 'resetPassword', row }">
                      <div class="dropdown-item">
                        <div class="dropdown-item-title dropdown-item-default">重置密码</div>
                        <div class="dropdown-item-desc">为该助理设置新的后台登录密码</div>
                      </div>
                    </el-dropdown-item>
                    <el-dropdown-item v-if="row.loginCooldown" :command="{ action: 'resolveLoginCooldown', row }">
                      <div class="dropdown-item">
                        <div class="dropdown-item-title dropdown-item-primary">解除登录冷却</div>
                        <div class="dropdown-item-desc">{{ loginCooldownDesc(row) }}</div>
                      </div>
                    </el-dropdown-item>
                    <el-dropdown-item :command="{ action: 'editProfile', row }">
                      <div class="dropdown-item">
                        <div class="dropdown-item-title dropdown-item-primary">编辑账号</div>
                        <div class="dropdown-item-desc">修改登录账号或后台展示的昵称</div>
                      </div>
                    </el-dropdown-item>
                    <el-dropdown-item :command="{ action: 'ipWhitelist', row }">
                      <div class="dropdown-item">
                        <div class="dropdown-item-title dropdown-item-primary">IP白名单</div>
                        <div class="dropdown-item-desc">查看并维护该助理的白名单规则</div>
                      </div>
                    </el-dropdown-item>
                    <el-dropdown-item :command="{ action: 'quietHours', row }">
                      <div class="dropdown-item">
                        <div class="dropdown-item-title dropdown-item-primary">定时策略</div>
                        <div class="dropdown-item-desc">查看并维护该助理管理域的定时限制策略</div>
                      </div>
                    </el-dropdown-item>
                    <el-dropdown-item :command="{ action: 'resetTotp', row }" divided>
                      <div class="dropdown-item">
                        <div class="dropdown-item-title dropdown-item-danger">重置2FA</div>
                        <div class="dropdown-item-desc">清空当前绑定，要求该助理重新绑定验证器</div>
                      </div>
                    </el-dropdown-item>
                    <el-dropdown-item :command="{ action: 'delete', row }" divided>
                      <div class="dropdown-item">
                        <div class="dropdown-item-title dropdown-item-danger">删除助理</div>
                        <div class="dropdown-item-desc">仅支持删除无关联用户、群组和邀请码的助理</div>
                      </div>
                    </el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
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
          @current-change="(p) => { page = p; fetchAssistants() }"
        />
      </div>
    </div>

    <!-- 新增助理 Modal -->
    <el-dialog
      v-model="showCreateModal"
      title="新增助理"
      :width="isMobile ? '100%' : '400px'"
      :fullscreen="isMobile"
      :close-on-click-modal="false"
    >
      <div class="modal-body">
        <div class="form-field">
          <label class="form-label">昵称（选填）</label>
          <el-input v-model="createForm.displayName" placeholder="不填则使用用户名" maxlength="64" show-word-limit />
        </div>
        <div class="form-field">
          <label class="form-label">登录账号</label>
          <el-input v-model="createForm.username" placeholder="3–32 位英文字母或数字" maxlength="32" />
        </div>
        <div class="form-field">
          <label class="form-label">初始密码</label>
          <el-input v-model="createForm.password" type="password" placeholder="至少 8 位" show-password />
        </div>
      </div>
      <template #footer>
        <el-button @click="showCreateModal = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleCreate">创建</el-button>
      </template>
    </el-dialog>

    <el-dialog
      v-model="showEditProfileModal"
      title="编辑助理账号"
      :width="isMobile ? '100%' : '380px'"
      :fullscreen="isMobile"
      :close-on-click-modal="false"
    >
      <div class="modal-body">
        <div class="form-field">
          <label class="form-label">助理</label>
          <el-input v-model="profileForm.username" placeholder="3–32 位英文字母或数字" maxlength="32" autocomplete="off" />
          <p class="cell-muted" style="margin-top: 8px; font-size: 12px">修改登录账号后需重新登录；昵称可以使用中文。</p>
        </div>
        <div class="form-field">
          <label class="form-label">昵称</label>
          <el-input v-model="profileForm.displayName" placeholder="请输入昵称" maxlength="64" show-word-limit />
        </div>
      </div>
      <template #footer>
        <el-button @click="showEditProfileModal = false">取消</el-button>
        <el-button type="primary" :loading="profileSaving" @click="handleSaveProfile">保存</el-button>
      </template>
    </el-dialog>

    <IPWhitelistRulesDialog
      v-model:visible="showIPWhitelistModal"
      :account="selectedAssistant"
      subject="assistant"
      @success="fetchAssistants"
    />

    <ChatQuietHoursPolicyDialog
      v-model:visible="showPolicyModal"
      :account="selectedPolicyAssistant"
      subject="assistant"
      :editable="authStore.isSuperAdmin || authStore.isManager"
      @success="fetchAssistants"
    />

    <!-- 编辑邀请码 Modal -->
    <el-dialog
      v-model="showEditCodeModal"
      :title="`编辑「${editCodeForm.assistantName}」的邀请码`"
      :width="isMobile ? '100%' : '460px'"
      :fullscreen="isMobile"
      :close-on-click-modal="false"
    >
      <div class="modal-body-lg">
        <div class="modal-row">
          <span class="modal-label">邀请码</span>
          <el-input
            v-model="editCodeForm.code"
            class="code-input"
            placeholder="输入自定义邀请码"
            @input="editCodeForm.codeEdited = true"
          />
          <el-button
            size="small"
            plain
            type="primary"
            :loading="codeRegenerating"
            style="flex-shrink:0"
            @click="handleRegenerateCode"
          >
            <svg width="13" height="13" fill="none" stroke="currentColor" viewBox="0 0 24 24" style="margin-right:3px">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
            </svg>
            随机
          </el-button>
        </div>
        <div class="modal-row">
          <span class="modal-label">最大使用次数</span>
          <el-input-number
            v-model="editCodeForm.maxUses"
            :min="0"
            :max="9999"
            controls-position="both"
            style="flex: 1"
          />
          <span style="font-size:12px;color:#94a3b8;margin-left:8px">0 = 不限</span>
        </div>
        <div class="modal-row">
          <span class="modal-label">启用状态</span>
          <div class="switch-row">
            <span class="switch-label" :class="{ active: editCodeForm.status !== 'active' }">停用</span>
            <el-switch v-model="editCodeForm.status" active-value="active" inactive-value="expired" />
            <span class="switch-label" :class="{ active: editCodeForm.status === 'active' }">启用</span>
          </div>
        </div>
      </div>
      <template #footer>
        <el-button @click="showEditCodeModal = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSaveEditCode">保存</el-button>
      </template>
    </el-dialog>

    <!-- 重置密码 Modal -->
    <el-dialog
      v-model="showResetModal"
      :title="`重置密码 · ${resetForm.username}`"
      :width="isMobile ? '100%' : '380px'"
      :fullscreen="isMobile"
      :close-on-click-modal="false"
    >
      <div class="modal-body">
        <div class="form-field">
          <label class="form-label">新密码</label>
          <el-input v-model="resetForm.password" type="password" placeholder="至少 8 位" show-password />
        </div>
      </div>
      <template #footer>
        <el-button @click="showResetModal = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleResetPassword">确认重置</el-button>
      </template>
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

/* ── Toolbar ── */
.card-toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  border-bottom: 1px solid #f0f0f0;
}

.page-table-wrap {
  border-bottom: 1px solid #f0f0f0;
}

.filter-wrap { position: relative; flex: 1; max-width: 560px; }
.filter-box {
  display: flex; flex-wrap: wrap; align-items: center; gap: 6px;
  min-height: 32px; padding: 4px 12px;
  border: 1px solid #f0f0f0; border-radius: 8px; background: #fff; cursor: text; transition: border-color 0.15s;
}
.filter-box:hover { border-color: rgba(22,119,255,0.4); }
.filter-icon { width: 14px; height: 14px; color: #94a3b8; flex-shrink: 0; }
.filter-tag {
  display: inline-flex; align-items: center; gap: 3px;
  padding: 2px 8px; background: rgba(22,119,255,0.1); color: #1677ff;
  font-size: 11px; font-weight: 500; border-radius: 4px;
}
.filter-tag-key { color: rgba(22,119,255,0.6); margin-right: 2px; }
.filter-tag-remove { margin-left: 2px; background: none; border: none; color: rgba(22,119,255,0.5); cursor: pointer; font-size: 14px; line-height: 1; padding: 0; }
.filter-tag-remove:hover { color: #1677ff; }
.filter-placeholder { color: #94a3b8; font-size: 13px; user-select: none; }

.filter-dropdown {
  position: absolute; top: calc(100% + 6px); left: 0; width: 288px;
  background: #fff; border: 1px solid #f0f0f0; border-radius: 10px;
  box-shadow: 0 10px 15px -3px rgba(0,0,0,0.1); z-index: 50; overflow: hidden;
}
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
.cell-id {
  font-size: 13px;
  color: #94a3b8;
}
.account-cell {
  display: flex;
  flex-direction: column;
  gap: 2px;
  line-height: 1.35;
}
.cell-username {
  font-size: 13px;
  font-weight: 500;
  color: #1677ff;
}
.cell-code {
  font-family: 'Fira Code', 'Courier New', monospace;
  font-size: 13px;
  color: #1e293b;
}
.cell-muted { color: #64748b; }
.cell-xs { font-size: 12px; color: #94a3b8; }
.cell-empty { color: #d1d5db; }
.inf-symbol { font-size: 18px; line-height: 1; vertical-align: middle; }

/* ── Status Badge ── */
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
.status-active  { color: #52c41a; border-color: #b7eb8f; background: #f6ffed; }
.status-banned  { color: #ff4d4f; border-color: #ffa39e; background: #fff2f0; }
.status-cooldown { color: #fa8c16; border-color: #ffd591; background: #fff7e6; }
.status-inactive { color: #8c8c8c; border-color: #d9d9d9; background: #fafafa; }

/* ── Invite Code Cell ── */
.code-cell {
  display: flex;
  align-items: center;
  gap: 6px;
}
.copy-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  border: none;
  background: none;
  color: #d1d5db;
  cursor: pointer;
  transition: color 0.15s;
  padding: 0;
  flex-shrink: 0;
}
.copy-btn:hover { color: #1677ff; }

/* ── User Count Bubble ── */
.count-bubble {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border-radius: 9999px;
  font-size: 12px;
  font-weight: 600;
}
.count-active { background: #e6f4ff; color: #1677ff; }
.count-empty  { background: #f1f5f9; color: #94a3b8; }

/* ── Action Buttons ── */
.action-btns {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  width: 100%;
}

.more-btn {
  padding: 2px 8px !important;
  font-size: 12px !important;
}

.more-btn-icon {
  margin-left: 4px;
  font-size: 12px;
}

.dropdown-item {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.dropdown-item-title {
  font-size: 12px;
  font-weight: 600;
  line-height: 1.3;
}

.dropdown-item-desc {
  font-size: 11px;
  line-height: 1.35;
  color: #94a3b8;
}

.dropdown-item-primary { color: #1677ff; }
.dropdown-item-default { color: #475569; }
.dropdown-item-warning { color: #d46b08; }
.dropdown-item-success { color: #389e0d; }
.dropdown-item-danger { color: #ff4d4f; }

:deep(.assistant-action-dropdown .el-dropdown-menu__item) {
  height: auto;
  min-height: 0;
  padding: 8px 12px;
  line-height: 1.35;
  align-items: flex-start;
}

:deep(.assistant-action-dropdown .el-dropdown-menu__item.is-divided) {
  margin-top: 4px;
}

/* ── Pagination ── */
.pagination-bar {
  display: flex;
  justify-content: flex-end;
  padding: 12px 16px;
  border-top: 1px solid #f0f0f0;
}

/* ── Modal ── */
.modal-body {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding-top: 4px;
}
.modal-body-lg {
  display: flex;
  flex-direction: column;
  gap: 20px;
  padding: 8px 0;
}
.form-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.form-label {
  font-size: 12px;
  font-weight: 500;
  color: #475569;
}
.modal-row {
  display: flex;
  align-items: center;
  gap: 16px;
}
.modal-label {
  font-size: 13px;
  color: #64748b;
  width: 80px;
  flex-shrink: 0;
}
.modal-code {
  flex: 1;
  font-size: 14px;
  letter-spacing: 0.05em;
}
.code-input {
  flex: 1;
  font-family: 'Fira Code', 'Courier New', monospace;
}
.switch-row {
  display: flex;
  align-items: center;
  gap: 12px;
}
.switch-label {
  font-size: 13px;
  color: #94a3b8;
  transition: color 0.15s;
}
.switch-label.active {
  color: #1677ff;
  font-weight: 500;
}

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

  .more-btn {
    width: 100%;
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
