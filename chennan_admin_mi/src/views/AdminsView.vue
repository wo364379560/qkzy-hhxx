<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { MoreFilled } from '@element-plus/icons-vue'
import {
  createAdmin,
  deleteAdmin,
  getAdminList,
  banAdmin,
  unbanAdmin,
  resetAdminPassword,
  resetAdminTOTP,
  updateAdminProfile,
  resolveAdminLoginCooldown,
} from '@/api/admin'
import ChatQuietHoursPolicyDialog from '@/components/ChatQuietHoursPolicyDialog.vue'
import IPWhitelistRulesDialog from '@/components/IPWhitelistRulesDialog.vue'
import type { AdminUser } from '@/types'
import { useResponsive } from '@/composables/useResponsive'

interface FilterField {
  key: string
  label: string
  type: 'text' | 'select'
  options?: { value: string; label: string }[]
}

const FILTER_FIELDS: FilterField[] = [
  { key: 'username', label: '用户名', type: 'text' },
  { key: 'status', label: '账号状态', type: 'select', options: [
    { value: 'active', label: '正常' },
    { value: 'banned', label: '已禁用' },
  ]},
]
const { isMobile } = useResponsive()

interface ActiveFilter { key: string; label: string; value: string; display: string }
const activeFilters = ref<ActiveFilter[]>([])
const filterWrapRef = ref<HTMLElement | null>(null)
const dropdownOpen = ref(false)
const selectedField = ref<FilterField | null>(null)
const filterInput = ref('')
const filterSelect = ref('')

function openDropdown() { dropdownOpen.value = true; selectedField.value = null; filterInput.value = ''; filterSelect.value = '' }
function selectField(f: FilterField) { selectedField.value = f; filterInput.value = ''; filterSelect.value = '' }
function confirmFilter() {
  if (!selectedField.value) return
  const f = selectedField.value
  const raw = f.type === 'select' ? filterSelect.value : filterInput.value.trim()
  if (!raw) return
  const display = f.type === 'select' ? (f.options?.find(o => o.value === raw)?.label ?? raw) : raw
  activeFilters.value = activeFilters.value.filter(item => item.key !== f.key)
  activeFilters.value.push({ key: f.key, label: f.label, value: raw, display })
  dropdownOpen.value = false
  selectedField.value = null
}
function removeFilter(key: string) { activeFilters.value = activeFilters.value.filter(item => item.key !== key) }
function clearAllFilters() { activeFilters.value = [] }
function onDocMousedown(e: MouseEvent) {
  if (filterWrapRef.value && !filterWrapRef.value.contains(e.target as Node)) dropdownOpen.value = false
}

onMounted(() => document.addEventListener('mousedown', onDocMousedown))
onUnmounted(() => document.removeEventListener('mousedown', onDocMousedown))

const loading = ref(false)
const admins = ref<AdminUser[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)

const filtered = computed(() => {
  if (!activeFilters.value.length) return admins.value
  return admins.value.filter(admin => {
    return activeFilters.value.every(filter => {
      const value = filter.value.toLowerCase()
      switch (filter.key) {
        case 'username':
          return admin.username.toLowerCase().includes(value)
        case 'status':
          return admin.status === filter.value
        default:
          return true
      }
    })
  })
})

const showCreateModal = ref(false)
const showResetModal = ref(false)
const showEditProfileModal = ref(false)
const showIPWhitelistModal = ref(false)
const showPolicyModal = ref(false)
const creating = ref(false)
const resetting = ref(false)
const profileSaving = ref(false)
const createForm = reactive({ displayName: '', username: '', password: '' })
const resetForm = reactive({ id: 0, username: '', password: '' })
const profileForm = reactive({ id: 0, username: '', displayName: '' })
const selectedAdmin = ref<AdminUser | null>(null)
const selectedPolicyAdmin = ref<AdminUser | null>(null)

function adminDisplayName(row: AdminUser) {
  return row.displayName?.trim() || '未设置'
}

function accountStatusClass(row: AdminUser) {
  if (row.loginCooldown) return 'status-cooldown'
  return row.status === 'active' ? 'status-active' : 'status-banned'
}

function accountStatusText(row: AdminUser) {
  if (row.loginCooldown) return '冷却中'
  return row.status === 'active' ? '正常' : '已禁用'
}

function loginCooldownDesc(row: AdminUser) {
  const parts = ['清除登录错误冷却，允许该管理员立即重新登录']
  if (row.loginCooldownUntil) parts.push(`冷却至 ${row.loginCooldownUntil}`)
  return parts.join('；')
}

async function fetchAdmins() {
  loading.value = true
  try {
    const res = await getAdminList({ page: page.value, pageSize: pageSize.value })
    admins.value = res.data.data?.list ?? []
    total.value = res.data.data?.total ?? 0
  } catch (e: any) {
    ElMessage.error(e.message || '获取管理员列表失败')
  } finally {
    loading.value = false
  }
}

onMounted(fetchAdmins)

async function handleCreate() {
  if (!createForm.displayName.trim()) {
    ElMessage.warning('请填写昵称')
    return
  }
  if (createForm.username.length < 3) {
    ElMessage.warning('用户名至少 3 位')
    return
  }
  if (createForm.password.length < 8) {
    ElMessage.warning('密码至少 8 位')
    return
  }
  creating.value = true
  try {
    await createAdmin({
      username: createForm.username,
      password: createForm.password,
      display_name: createForm.displayName.trim(),
    })
    ElMessage.success('管理员已创建')
    createForm.displayName = ''
    createForm.username = ''
    createForm.password = ''
    showCreateModal.value = false
    fetchAdmins()
  } catch (e: any) {
    ElMessage.error(e.message || '创建失败')
  } finally {
    creating.value = false
  }
}

async function handleToggleBan(row: AdminUser) {
  const isBanned = row.status !== 'active'
  const action = isBanned ? '启用' : '禁用'
  await ElMessageBox.confirm(`确认${action}管理员 "${row.username}"？`, '确认操作', {
    type: isBanned ? 'info' : 'warning',
  })
  try {
    if (isBanned) await unbanAdmin(row.id)
    else await banAdmin(row.id)
    ElMessage.success(`已${action}`)
    fetchAdmins()
  } catch (e: any) {
    ElMessage.error(e.message || '操作失败')
  }
}

function openResetModal(row: AdminUser) {
  resetForm.id = row.id
  resetForm.username = row.username
  resetForm.password = ''
  showResetModal.value = true
}

function openEditProfileModal(row: AdminUser) {
  profileForm.id = row.id
  profileForm.username = row.username
  profileForm.displayName = row.displayName || ''
  showEditProfileModal.value = true
}

function openIPWhitelistModal(row: AdminUser) {
  selectedAdmin.value = row
  showIPWhitelistModal.value = true
}

function openPolicyModal(row: AdminUser) {
  selectedPolicyAdmin.value = row
  showPolicyModal.value = true
}

async function handleResetPassword() {
  if (resetForm.password.length < 8) {
    ElMessage.warning('密码至少 8 位')
    return
  }
  resetting.value = true
  try {
    await resetAdminPassword(resetForm.id, resetForm.password)
    ElMessage.success('密码已重置')
    showResetModal.value = false
  } catch (e: any) {
    ElMessage.error(e.message || '重置失败')
  } finally {
    resetting.value = false
  }
}

async function handleSaveProfile() {
  if (!profileForm.displayName.trim()) {
    ElMessage.warning('请填写昵称')
    return
  }
  profileSaving.value = true
  try {
    await updateAdminProfile(profileForm.id, { username: profileForm.username, display_name: profileForm.displayName.trim() })
    ElMessage.success('账号资料已更新')
    showEditProfileModal.value = false
    fetchAdmins()
  } catch (e: any) {
    ElMessage.error(e.message || '保存失败')
  } finally {
    profileSaving.value = false
  }
}

async function handleResetTOTP(row: AdminUser) {
  await ElMessageBox.confirm(`确认重置管理员 "${row.username}" 的双因素认证？旧绑定、恢复码和登录状态将失效；全局 2FA 开启时，下次登录必须重新绑定。`, '确认操作', { type: 'warning' })
  try {
    await resetAdminTOTP(row.id)
    ElMessage.success('已重置 2FA')
    fetchAdmins()
  } catch (e: any) {
    ElMessage.error(e.message || '重置失败')
  }
}

async function handleResolveLoginCooldown(row: AdminUser) {
  const suffix = row.loginCooldownUntil ? `当前冷却至 ${row.loginCooldownUntil}。` : ''
  await ElMessageBox.confirm(
    `确认解除管理员 "${row.username}" 的登录冷却？${suffix}`,
    '解除登录冷却',
    { type: 'warning', confirmButtonText: '解除' },
  )
  try {
    await resolveAdminLoginCooldown(row.id)
    ElMessage.success('登录冷却已解除')
    fetchAdmins()
  } catch (e: any) {
    ElMessage.error(e.message || '解除失败')
  }
}

async function handleDelete(row: AdminUser) {
  await ElMessageBox.confirm(
    `确认删除管理员 "${row.username}"？删除后不可恢复。若名下仍有关联助理、用户、群组或邀请码，将无法删除。`,
    '删除管理员',
    { type: 'warning', confirmButtonText: '删除' },
  )
  try {
    await deleteAdmin(row.id)
    ElMessage.success('管理员已删除')
    fetchAdmins()
  } catch (e: any) {
    ElMessage.error(e.message || '删除失败')
  }
}

function handleRowAction(command: { action: string; row: AdminUser }) {
  const { action, row } = command
  switch (action) {
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
</script>

<template>
  <div class="page">
    <div class="card">
      <div class="card-header">
        <div>
          <h1 class="page-title">管理员</h1>
          <p class="page-sub">仅超级管理员可创建、启用、禁用和重置管理员账号。</p>
        </div>
        <div class="page-header-actions">
          <el-button type="primary" @click="showCreateModal = true">
            <template #icon><el-icon><Plus /></el-icon></template>
            新增管理员
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
                <input
                  v-if="selectedField.type === 'text'"
                  v-model="filterInput"
                  class="filter-text-input"
                  :placeholder="`输入${selectedField.label}…`"
                  autofocus
                  @keyup.enter.stop="confirmFilter"
                  @click.stop
                />
                <ul v-else class="filter-option-list">
                  <li
                    v-for="opt in selectedField.options"
                    :key="opt.value"
                    class="filter-option"
                    :class="{ selected: filterSelect === opt.value }"
                    @click.stop="filterSelect = opt.value"
                  >
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
        <el-table :data="filtered" v-loading="loading" stripe size="small" style="width: 100%">
        <el-table-column label="序号" width="60" align="center">
          <template #default="{ $index }">
            <span class="cell-id">{{ (page - 1) * pageSize + $index + 1 }}</span>
          </template>
        </el-table-column>

        <el-table-column label="昵称 / 用户名" min-width="150">
          <template #default="{ row }">
            <div class="account-cell">
              <span class="cell-username">{{ adminDisplayName(row) }}</span>
              <span class="cell-muted cell-xs">{{ row.username }}</span>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="账号状态" width="100">
          <template #default="{ row }">
            <span class="status-badge" :class="accountStatusClass(row)" :title="row.loginCooldownUntil || ''">
              {{ accountStatusText(row) }}
            </span>
          </template>
        </el-table-column>

        <el-table-column v-if="!isMobile" label="助理数量" width="90">
          <template #default="{ row }">
            <span class="count-badge">{{ row.assistantCount ?? 0 }}</span>
          </template>
        </el-table-column>

        <el-table-column v-if="!isMobile" label="2FA" width="90">
          <template #default="{ row }">
            <span class="status-badge" :class="row.totpEnabled ? 'status-active' : 'status-inactive'">
              {{ row.totpEnabled ? '已绑定' : '未绑定' }}
            </span>
          </template>
        </el-table-column>

        <el-table-column v-if="!isMobile" label="IP 白名单" width="96">
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

        <el-table-column label="操作" :width="isMobile ? 184 : 272" :fixed="isMobile ? false : 'right'">
          <template #default="{ row }">
            <div class="action-btns">
              <el-button
                size="small"
                plain
                type="warning"
                style="padding: 2px 8px; font-size: 12px;"
                @click="handleToggleBan(row)"
              >
                {{ row.status === 'active' ? '禁用' : '启用' }}
              </el-button>
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
                    <el-dropdown-item :command="{ action: 'toggleBan', row }">
                      <div class="dropdown-item">
                        <div class="dropdown-item-title" :class="row.status === 'active' ? 'dropdown-item-warning' : 'dropdown-item-success'">
                          {{ row.status === 'active' ? '禁用' : '启用' }}
                        </div>
                        <div class="dropdown-item-desc">
                          {{ row.status === 'active' ? '禁止该管理员继续登录后台' : '恢复该管理员正常登录权限' }}
                        </div>
                      </div>
                    </el-dropdown-item>
                    <el-dropdown-item :command="{ action: 'resetPassword', row }">
                      <div class="dropdown-item">
                        <div class="dropdown-item-title dropdown-item-default">重置密码</div>
                        <div class="dropdown-item-desc">为该管理员设置新的后台登录密码</div>
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
                        <div class="dropdown-item-desc">查看并维护该管理员的白名单规则</div>
                      </div>
                    </el-dropdown-item>
                    <el-dropdown-item :command="{ action: 'quietHours', row }">
                      <div class="dropdown-item">
                        <div class="dropdown-item-title dropdown-item-primary">定时策略</div>
                        <div class="dropdown-item-desc">查看并维护该管理员管理域的定时限制策略</div>
                      </div>
                    </el-dropdown-item>
                    <el-dropdown-item :command="{ action: 'resetTotp', row }" divided>
                      <div class="dropdown-item">
                        <div class="dropdown-item-title dropdown-item-danger">重置2FA</div>
                        <div class="dropdown-item-desc">清空当前绑定，要求该管理员重新绑定验证器</div>
                      </div>
                    </el-dropdown-item>
                    <el-dropdown-item :command="{ action: 'delete', row }" divided>
                      <div class="dropdown-item">
                        <div class="dropdown-item-title dropdown-item-danger">删除管理员</div>
                        <div class="dropdown-item-desc">仅支持删除无关联助理、用户、群组和邀请码的管理员</div>
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
          @current-change="(p) => { page = p; fetchAdmins() }"
        />
      </div>
    </div>

    <el-dialog
      v-model="showCreateModal"
      title="新增管理员"
      :width="isMobile ? '100%' : '400px'"
      :fullscreen="isMobile"
      :close-on-click-modal="false"
    >
      <div class="modal-body">
        <div class="form-field">
          <label class="form-label">昵称</label>
          <el-input v-model="createForm.displayName" placeholder="请输入昵称" maxlength="64" show-word-limit />
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
        <el-button type="primary" :loading="creating" @click="handleCreate">创建</el-button>
      </template>
    </el-dialog>

    <el-dialog
      v-model="showEditProfileModal"
      title="编辑管理员账号"
      :width="isMobile ? '100%' : '400px'"
      :fullscreen="isMobile"
      :close-on-click-modal="false"
    >
      <div class="modal-body">
        <div class="form-field">
          <label class="form-label">管理员</label>
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

    <el-dialog
      v-model="showResetModal"
      title="重置管理员密码"
      :width="isMobile ? '100%' : '400px'"
      :fullscreen="isMobile"
      :close-on-click-modal="false"
    >
      <div class="modal-body">
        <div class="form-field">
          <label class="form-label">管理员</label>
          <el-input :model-value="resetForm.username" disabled />
        </div>
        <div class="form-field">
          <label class="form-label">新密码</label>
          <el-input v-model="resetForm.password" type="password" placeholder="至少 8 位" show-password />
        </div>
      </div>
      <template #footer>
        <el-button @click="showResetModal = false">取消</el-button>
        <el-button type="primary" :loading="resetting" @click="handleResetPassword">确认</el-button>
      </template>
    </el-dialog>

    <IPWhitelistRulesDialog
      v-model:visible="showIPWhitelistModal"
      :account="selectedAdmin"
      subject="manager"
      @success="fetchAdmins"
    />

    <ChatQuietHoursPolicyDialog
      v-model:visible="showPolicyModal"
      :account="selectedPolicyAdmin"
      subject="manager"
      :editable="true"
      @success="fetchAdmins"
    />
  </div>
</template>

<style scoped>
.page { padding: 24px 28px; }
.page-header-actions { display:flex; flex-wrap:wrap; justify-content:flex-end; gap:8px; }
.card-toolbar { display:flex; align-items:center; gap:8px; padding: 14px 18px; border-bottom:1px solid #f0f0f0; }
.page-table-wrap { border-bottom: 1px solid #f0f0f0; }
.filter-wrap { position: relative; flex: 1; max-width: 520px; }
.filter-box { min-height: 40px; display:flex; flex-wrap:wrap; align-items:center; gap:6px; padding: 4px 10px; border:1px solid #f0f0f0; border-radius:10px; background:#fff; cursor:pointer; }
.filter-icon { width:14px; height:14px; color:#94a3b8; }
.filter-tag { display:inline-flex; align-items:center; gap:3px; padding:2px 8px; background:rgba(22,119,255,0.1); color:#1677ff; font-size:11px; font-weight:500; border-radius:4px; }
.filter-tag-key { color:rgba(22,119,255,0.6); margin-right:2px; }
.filter-tag-remove { margin-left:2px; background:none; border:none; color:rgba(22,119,255,0.5); cursor:pointer; font-size:14px; line-height:1; padding:0; }
.filter-placeholder { color:#94a3b8; font-size:12px; }
.filter-dropdown { position:absolute; top:calc(100% + 6px); left:0; width:288px; background:#fff; border:1px solid #f0f0f0; border-radius:10px; box-shadow:0 10px 15px -3px rgba(0,0,0,0.1); z-index:50; overflow:hidden; }
.filter-dropdown-head { font-size:10px; font-weight:600; color:#94a3b8; text-transform:uppercase; letter-spacing:0.06em; padding:12px 12px 6px; }
.filter-field-item { display:flex; align-items:center; gap:10px; padding:8px 12px; cursor:pointer; font-size:14px; color:#1e293b; transition:background 0.1s; }
.filter-field-item:hover { background:#f8fafc; }
.filter-field-icon { width:20px; height:20px; border-radius:4px; background:rgba(22,119,255,0.1); display:flex; align-items:center; justify-content:center; color:#1677ff; flex-shrink:0; }
.filter-chevron { width:14px; height:14px; color:#d1d5db; margin-left:auto; }
.filter-value-head { display:flex; align-items:center; gap:8px; padding:10px 12px 8px; border-bottom:1px solid #f0f0f0; font-size:14px; font-weight:500; color:#1e293b; }
.filter-back-btn { background:none; border:none; color:#94a3b8; cursor:pointer; display:flex; }
.filter-value-body { padding:12px; display:flex; flex-direction:column; gap:8px; }
.filter-text-input { width:100%; height:32px; padding:0 12px; font-size:14px; border:1px solid #f0f0f0; border-radius:6px; outline:none; }
.filter-option-list { display:flex; flex-direction:column; gap:2px; max-height: 240px; overflow-y: auto; padding-right: 2px; }
.filter-option { display:flex; align-items:center; gap:8px; padding:6px 8px; border-radius:6px; cursor:pointer; font-size:14px; color:#1e293b; transition:background 0.1s; }
.filter-option:hover { background:#f8fafc; }
.filter-option.selected { background:rgba(22,119,255,0.08); color:#1677ff; }
.filter-radio { width:14px; height:14px; border-radius:50%; border:2px solid #d1d5db; flex-shrink:0; transition:all 0.1s; }
.filter-radio.active { border-color:#1677ff; background:#1677ff; }
.filter-confirm-btn { width:100%; height:32px; border-radius:6px; background:#1677ff; color:#fff; font-size:12px; font-weight:500; border:none; cursor:pointer; }
.clear-btn { height: 34px; padding: 0 12px; border-radius: 8px; border: 1px solid #f0f0f0; background: #fff; color: #64748b; font-size: 12px; cursor: pointer; }
.total-count { font-size:12px; color:#94a3b8; margin-left:auto; white-space:nowrap; }
.status-badge { display:inline-flex; align-items:center; justify-content:center; min-width:64px; height:24px; padding:0 10px; border-radius:999px; font-size:12px; font-weight:500; border:1px solid transparent; }
.status-active { color:#52c41a; border-color:#b7eb8f; background:#f6ffed; }
.status-banned { color:#ff4d4f; border-color:#ffa39e; background:#fff2f0; }
.status-cooldown { color:#fa8c16; border-color:#ffd591; background:#fff7e6; }
.status-inactive { color:#8c8c8c; border-color:#d9d9d9; background:#fafafa; }
.account-cell { display:flex; flex-direction:column; gap:2px; line-height:1.35; }
.cell-username { font-size:13px; font-weight:500; color:#1e293b; }
.cell-code, .cell-xs, .cell-muted { font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace; }
.cell-muted { color:#64748b; }
.cell-xs { color:#94a3b8; font-size:11px; }
.pagination-bar { padding: 14px 18px; display:flex; justify-content:flex-end; }
.modal-body, .form-field { display:flex; flex-direction:column; gap:8px; }
.modal-body { gap:16px; }
.form-label { font-size:12px; font-weight:500; color:#475569; }
.action-btns { display:inline-flex; align-items:center; justify-content:center; gap:6px; width:100%; }
.more-btn { padding:2px 8px !important; font-size:12px !important; }
.more-btn-icon { margin-left:4px; font-size:12px; }
.dropdown-item { display:flex; flex-direction:column; gap:2px; }
.dropdown-item-title { font-size:12px; font-weight:600; line-height:1.3; }
.dropdown-item-desc { font-size:11px; line-height:1.35; color:#8c8c8c; max-width:220px; white-space:normal; }
.dropdown-item-default { color:#1f2937; }
.dropdown-item-success { color:#16a34a; }
.dropdown-item-warning { color:#d97706; }
.dropdown-item-danger { color:#dc2626; }

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
}
</style>
