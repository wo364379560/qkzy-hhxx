<script setup lang="ts">
import { computed, ref, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  getGroupList,
  createGroup,
  dismissGroup,
  getGroupMembers,
  getGroupNotice,
  muteGroup,
  batchMuteGroups,
  batchUnmuteGroups,
  repairGroup,
  saveGroupNotice,
  syncGroupsPreview,
  syncGroupsConfirm,
  GROUP_TYPE_OPTIONS,
  GROUP_TYPE_CREATE_OPTIONS,
} from '@/api/group'
import type { GroupType, SyncConfirmItem } from '@/api/group'
import { getUserList, getUserOwners } from '@/api/user'
import type { Group, GroupMember, AppUser, SyncGroupPreview, UserOwnerOption } from '@/types'
import { useAuthStore } from '@/stores/auth'
import { useResponsive } from '@/composables/useResponsive'

const router = useRouter()
const authStore = useAuthStore()
const { isMobile } = useResponsive()
const pageSubText = computed(() => {
  if (authStore.isSuperAdmin) return '管理所有 IM 群组'
  if (authStore.isManager) return '管理我归属范围内的 IM 群组'
  return '管理我名下用户所属的 IM 群组'
})

// ── 字段筛选 ──
interface FilterField {
  key: string; label: string; type: 'text' | 'select'
  paramKey?: string
  options?: { value: string; label: string }[]
}
interface ActiveFilter { key: string; paramKey: string; label: string; value: string; display: string }
const ownerOptions = ref<UserOwnerOption[]>([])

const ownerFilterOptions = computed(() =>
  ownerOptions.value.map(owner => ({
    value: String(owner.id),
    label: owner.username,
  }))
)

const ownerManagementOptions = computed(() =>
  ownerOptions.value.map(owner => {
    const isSelf = owner.id === authStore.user?.id
    const roleLabel = owner.role === 'super_admin' ? '超级管理员' : owner.role === 'admin' ? '管理员' : '助理'
    const statusSuffix = owner.status === 'active' ? '' : '（已禁用）'
    const selfSuffix = isSelf ? '（我自己）' : ''
    return {
      value: owner.id,
      label: `${roleLabel}: ${owner.username}${selfSuffix}${statusSuffix}`,
    }
  })
)

const FILTER_FIELDS = computed<FilterField[]>(() => {
  const base: FilterField[] = [
    { key: 'status', label: '状态', type: 'select', options: [
      { value: 'active',    label: '正常' },
      { value: 'dismissed', label: '已解散' },
    ]},
    { key: 'groupType', label: '群类型', type: 'select', options: GROUP_TYPE_OPTIONS.map(opt => ({
      value: opt.value,
      label: opt.label,
    }))},
  ]

  if (authStore.isAdmin) {
    base.unshift({
      key: 'ownerName',
      paramKey: 'ownerID',
      label: '归属昵称',
      type: 'select',
      options: ownerFilterOptions.value,
    })
  }

  return base
})

const activeFilters = ref<ActiveFilter[]>([])
const filterWrapRef = ref<HTMLElement | null>(null)
const dropdownOpen  = ref(false)
const selectedField = ref<FilterField | null>(null)
const filterInput   = ref('')
const filterSelect  = ref('')
const selectedFieldOptions = computed(() => {
  if (!selectedField.value) return []
  if (selectedField.value.key === 'ownerName') return ownerFilterOptions.value
  return selectedField.value.options ?? []
})

function openDropdown() {
  dropdownOpen.value = true
  selectedField.value = null
  filterInput.value = ''
  filterSelect.value = ''
  if (authStore.isAdmin && ownerOptions.value.length === 0) {
    void loadOwnerOptions()
  }
}
function selectField(f: FilterField) { selectedField.value = f; filterInput.value = ''; filterSelect.value = '' }
function confirmFilter() {
  if (!selectedField.value) return
  const f = selectedField.value
  const raw = f.type === 'select' ? filterSelect.value : filterInput.value.trim()
  if (!raw) return
  const display = f.type === 'select' ? (selectedFieldOptions.value.find(o => o.value === raw)?.label ?? raw) : raw
  activeFilters.value = activeFilters.value.filter(a => a.key !== f.key)
  activeFilters.value.push({
    key: f.key,
    paramKey: f.paramKey ?? f.key,
    label: f.label,
    value: raw,
    display,
  })
  dropdownOpen.value = false; selectedField.value = null; page.value = 1; fetchGroups()
}
function removeFilter(key: string) { activeFilters.value = activeFilters.value.filter(a => a.key !== key); page.value = 1; fetchGroups() }
function clearAllFilters() { activeFilters.value = []; page.value = 1; fetchGroups() }
function onDocMousedown(e: MouseEvent) {
  if (filterWrapRef.value && !filterWrapRef.value.contains(e.target as Node)) dropdownOpen.value = false
}
onMounted(() => document.addEventListener('mousedown', onDocMousedown))
onUnmounted(() => document.removeEventListener('mousedown', onDocMousedown))

const showCreateModal = ref(false)
const loading    = ref(false)
const submitting = ref(false)
const groupName  = ref('')
const groupType  = ref<GroupType>('Public')
const groups     = ref<Group[]>([])
const total      = ref(0)
const page       = ref(1)
const pageSize   = ref(10)
const pageSizeOptions = [10, 30, 50]
const selectedGroups = ref<Group[]>([])
const searchKeyword = ref('')

function clearSelection() {
  selectedGroups.value = []
}

// ── 归属人选择（管理员专用）──
const selectedOwnerID = ref<number | null>(null)

async function loadOwnerOptions() {
  if (!authStore.isAdmin) {
    ownerOptions.value = []
    return
  }
  try {
    const res = await getUserOwners()
    ownerOptions.value = res.data.data ?? []
  } catch (e: any) {
    ownerOptions.value = []
    ElMessage.error(e.message || '获取归属人员列表失败')
  }
}

// ── 成员选择器 ──
const memberSearchText    = ref('')
const memberSearchLoading = ref(false)
const memberList          = ref<AppUser[]>([])
const selectedMembers     = ref<AppUser[]>([])
const ownerMemberID       = ref<number | null>(null)
let   searchTimer: ReturnType<typeof setTimeout> | null = null

const isSelected  = (u: AppUser) => selectedMembers.value.some(m => m.id === u.id)
function toggleMember(u: AppUser) {
  const idx = selectedMembers.value.findIndex(m => m.id === u.id)
  if (idx >= 0) selectedMembers.value.splice(idx, 1)
  else          selectedMembers.value.push(u)
}
function removeMember(u: AppUser) {
  selectedMembers.value = selectedMembers.value.filter(m => m.id !== u.id)
  if (ownerMemberID.value === u.id) ownerMemberID.value = null
}

async function loadMembers(keyword = '') {
  memberSearchLoading.value = true
  try {
    // 管理员：按选定归属人过滤；助理：后端自动过滤自己的用户
    const ownerID = authStore.isAdmin ? (selectedOwnerID.value ?? undefined) : undefined
    const res = await getUserList({ pageSize: 100, ...(keyword ? { keyword } : {}), ...(ownerID ? { ownerID } : {}) })
    memberList.value = res.data.data?.list ?? []
  } catch { /* 静默 */ }
  finally { memberSearchLoading.value = false }
}

function onSearchInput() {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => loadMembers(memberSearchText.value.trim()), 300)
}

async function openCreateModal() {
  groupName.value = ''
  groupType.value = 'Public'
  memberSearchText.value = ''
  selectedMembers.value = []
  ownerMemberID.value = null
  memberList.value = []
  selectedOwnerID.value = null

  if (authStore.isAdmin) {
    await loadOwnerOptions()
  }

  showCreateModal.value = true
  if (!authStore.isAdmin) loadMembers()
}

async function fetchGroups() {
  loading.value = true
  try {
    const res = await getGroupList(buildParams())
    groups.value = res.data.data?.list ?? []
    total.value  = res.data.data?.total ?? 0
    clearSelection()
  } catch (e: any) { ElMessage.error(e.message || '获取群组列表失败') }
  finally { loading.value = false }
}
onMounted(() => {
  if (authStore.isAdmin) {
    void loadOwnerOptions()
  }
  void fetchGroups()
})

const buildParams = () => {
  const p: Record<string, any> = { page: page.value, pageSize: pageSize.value }
  if (searchKeyword.value.trim()) p.keyword = searchKeyword.value.trim()
  for (const f of activeFilters.value) p[f.paramKey] = f.value
  return p
}

function applySearch() {
  page.value = 1
  fetchGroups()
}

function clearSearch() {
  if (!searchKeyword.value) return
  searchKeyword.value = ''
  page.value = 1
  fetchGroups()
}

async function handleCreate() {
  if (!groupName.value.trim()) { ElMessage.warning('请填写群组名称'); return }
  if (authStore.isAdmin && !selectedOwnerID.value) { ElMessage.warning('请选择归属人'); return }
  submitting.value = true
  try {
    await createGroup({
      name: groupName.value.trim(),
      groupType: groupType.value,
      ownerID: authStore.isAdmin ? (selectedOwnerID.value ?? undefined) : undefined,
      memberIDs: selectedMembers.value.map(m => m.id),
      ownerMemberID: ownerMemberID.value ?? undefined,
    })
    ElMessage.success('群组已创建')
    showCreateModal.value = false
    fetchGroups()
  } catch (e: any) { ElMessage.error(e.message || '创建失败') }
  finally { submitting.value = false }
}

async function handleMute(group: Group) {
  const next = !group.isMuted
  const label = next ? '静默（禁止所有成员发消息）' : '取消静默'
  await ElMessageBox.confirm(`确认对群组 "${group.name}" ${label}？`, '确认操作', { type: 'warning' })
  try {
    await muteGroup(group.id, next)
    ElMessage.success(next ? '已开启全员禁言' : '已取消全员禁言')
    fetchGroups()
  } catch (e: any) { ElMessage.error(e.message || '操作失败') }
}

async function handleRepair(group: Group) {
  try {
    await repairGroup(group.id)
    ElMessage.success('已同步到 IM，稍后刷新查看')
    fetchGroups()
  } catch (e: any) { ElMessage.error(e.message || '同步失败') }
}

async function handleDismiss(group: Group) {
  await ElMessageBox.confirm(`确认解散群组 "${group.name}"？`, '确认操作', { type: 'error' })
  try { await dismissGroup(group.id); ElMessage.success('群组已解散'); fetchGroups() }
  catch (e: any) { ElMessage.error(e.message || '操作失败') }
}

async function handleBatchMute(nextMute: boolean) {
  if (selectedGroups.value.length === 0) {
    ElMessage.warning('请先选择群组')
    return
  }

  const actionLabel = nextMute ? '静默' : '取消静默'
  await ElMessageBox.confirm(`确认对已选 ${selectedGroups.value.length} 个群组执行${actionLabel}？`, '确认操作', { type: 'warning' })

  try {
    const groupIDs = selectedGroups.value.map(group => group.id)
    const res = nextMute ? await batchMuteGroups(groupIDs) : await batchUnmuteGroups(groupIDs)
    const data = res.data.data
    const failedDetails = (data.details ?? [])
      .filter(item => !item.success)
      .slice(0, 3)
      .map(item => `${item.groupName || item.groupID}: ${item.message}`)
      .join('；')
    const summary = `成功 ${data.success} 个，失败 ${data.failed} 个`
    ElMessage[data.failed > 0 ? 'warning' : 'success'](`${actionLabel}完成：${summary}${failedDetails ? `；${failedDetails}` : ''}`)
    fetchGroups()
  } catch (e: any) {
    ElMessage.error(e.message || `${actionLabel}失败`)
  }
}

function openConversationShortcut(group: Group) {
  router.push({
    name: 'Conversations',
    query: {
      mode: 'group',
      groupID: group.imGroupId,
      name: group.name,
      groupType: group.groupType,
    },
  })
}

// ── 查看成员 ──
const showMembersModal = ref(false)
const membersLoading   = ref(false)
const currentGroup     = ref<Group | null>(null)
const memberResults    = ref<GroupMember[]>([])
const showNoticeModal  = ref(false)
const noticeLoading    = ref(false)
const noticeSaving     = ref(false)
const currentNotice    = ref('')

async function openMembersModal(group: Group) {
  currentGroup.value = group
  memberResults.value = []
  showMembersModal.value = true
  membersLoading.value = true
  try {
    const res = await getGroupMembers(group.id)
    memberResults.value = res.data.data ?? []
  } catch (e: any) { ElMessage.error(e.message || '获取成员失败') }
  finally { membersLoading.value = false }
}

async function openNoticeModal(group: Group) {
  currentGroup.value = group
  currentNotice.value = ''
  showNoticeModal.value = true
  noticeLoading.value = true
  try {
    const res = await getGroupNotice(group.id)
    currentNotice.value = res.data.data?.content ?? ''
  } catch (e: any) {
    ElMessage.error(e.message || '获取群公告失败')
  } finally {
    noticeLoading.value = false
  }
}

async function saveNotice() {
  if (!currentGroup.value) return
  noticeSaving.value = true
  try {
    await saveGroupNotice(currentGroup.value.id, currentNotice.value)
    ElMessage.success('群公告已保存')
    showNoticeModal.value = false
  } catch (e: any) {
    ElMessage.error(e.message || '保存群公告失败')
  } finally {
    noticeSaving.value = false
  }
}

function roleLabel(role: string) {
  if (role === 'Owner') return { cls: 'role-owner', text: '群主' }
  if (role === 'Admin') return { cls: 'role-admin', text: '管理员' }
  return { cls: 'role-member', text: '成员' }
}
function formatJoinTime(ts: number) {
  if (!ts) return '-'
  return new Date(ts * 1000).toLocaleDateString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit' })
}

// ── 同步群组 ──
const showSyncModal = ref(false)
const syncLoading = ref(false)
const syncSubmitting = ref(false)
const syncPreviews = ref<SyncGroupPreview[]>([])

async function openSyncModal() {
  showSyncModal.value = true
  syncPreviews.value = []
  syncLoading.value = true
  try {
    const res = await syncGroupsPreview()
    syncPreviews.value = (res.data.data ?? []).map(item => ({ ...item, action: 'skip' as const }))
  } catch (e: any) {
    ElMessage.error(e.message || '拉取群组数据失败')
    showSyncModal.value = false
  } finally {
    syncLoading.value = false
  }
}

function toggleSyncAction(item: SyncGroupPreview) {
  if (item.isNew) {
    item.action = item.action === 'skip' ? 'import' : 'skip'
  } else {
    item.action = item.action === 'skip' ? 'update' : 'skip'
  }
}

async function handleSyncConfirm() {
  const selections: SyncConfirmItem[] = syncPreviews.value
    .filter(item => item.action !== 'skip')
    .map(item => ({ imGroupId: item.imGroupId, action: item.action }))

  if (selections.length === 0) {
    ElMessage.warning('请至少选择一个群组进行同步')
    return
  }

  syncSubmitting.value = true
  try {
    const res = await syncGroupsConfirm(selections)
    const result = res.data.data
    ElMessage.success(`同步完成：成功 ${result.success} 个，失败 ${result.failed} 个`)
    showSyncModal.value = false
    fetchGroups()
  } catch (e: any) {
    ElMessage.error(e.message || '同步失败')
  } finally {
    syncSubmitting.value = false
  }
}
</script>

<template>
  <div class="page">
    <div class="card">
      <div class="card-header">
        <div>
          <h1 class="page-title">群组管理</h1>
          <p class="page-sub">{{ pageSubText }}</p>
        </div>
        <div class="page-header-actions">
          <el-button v-if="authStore.isAdmin" plain @click="openSyncModal">
            <template #icon>
              <svg width="14" height="14" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
              </svg>
            </template>
            同步群组
          </el-button>
          <el-button type="primary" @click="openCreateModal">
            <template #icon><el-icon><Plus /></el-icon></template>新建群组
          </el-button>
        </div>
      </div>

      <div class="card-toolbar">
        <div ref="filterWrapRef" class="filter-wrap">
          <div class="filter-box">
            <svg class="filter-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/>
            </svg>
            <span v-for="f in activeFilters" :key="f.key" class="filter-tag">
              <span class="filter-tag-key">{{ f.label }}:</span>{{ f.display }}
              <button class="filter-tag-remove" @click.stop="removeFilter(f.key)">×</button>
            </span>
            <input
              v-model="searchKeyword"
              class="filter-search-input"
              type="text"
              placeholder="搜索群组名称 / IM群组ID"
              @keyup.enter="applySearch"
              @click.stop
            />
            <div class="filter-actions">
              <button v-if="searchKeyword" class="filter-inline-btn" @click.stop="clearSearch">清空</button>
              <button v-if="activeFilters.length > 0" class="filter-inline-btn" @click.stop="clearAllFilters">重置筛选</button>
              <button class="filter-inline-btn filter-inline-btn-primary" @click.stop="applySearch">搜索</button>
              <button class="filter-inline-btn" @click.stop="openDropdown">筛选{{ activeFilters.length > 0 ? `(${activeFilters.length})` : '' }}</button>
            </div>
          </div>
          <div class="filter-helper">
            搜索支持：群组名称 / IM群组ID
            <span class="filter-helper-divider">|</span>
            筛选支持：{{ authStore.isAdmin ? '归属昵称 / 状态 / 群类型' : '状态 / 群类型' }}
          </div>
          <div v-if="dropdownOpen" class="filter-dropdown">
            <template v-if="!selectedField">
              <div class="filter-dropdown-head">结构化筛选</div>
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
                  <li v-if="selectedFieldOptions.length === 0" class="filter-option filter-option-empty">
                    暂无可选项
                  </li>
                  <li v-for="opt in selectedFieldOptions" :key="opt.value"
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
        <span class="total-count">共 {{ total }} 条</span>
      </div>

      <div v-if="selectedGroups.length > 0" class="batch-toolbar">
        <span class="batch-count">已选 {{ selectedGroups.length }} 项</span>
        <div class="batch-actions">
          <el-button size="small" plain @click="handleBatchMute(true)">批量静默</el-button>
          <el-button size="small" plain @click="handleBatchMute(false)">批量取消静默</el-button>
          <el-button size="small" plain @click="clearSelection">取消选择</el-button>
        </div>
      </div>

      <div class="table-scroll-x page-table-wrap">
        <el-table
          :data="groups"
          v-loading="loading"
          stripe
          size="small"
          style="width:100%"
          @selection-change="(rows: Group[]) => { selectedGroups = rows }"
        >
        <el-table-column type="selection" width="46" align="center" />
        <el-table-column label="序号" width="60" align="center">
          <template #default="{ $index }">
            <span class="cell-id">{{ (page - 1) * pageSize + $index + 1 }}</span>
          </template>
        </el-table-column>

        <el-table-column label="群组名称" min-width="180">
          <template #default="{ row }">
            <div class="group-cell">
              <div class="group-avatar">
                <svg width="14" height="14" fill="none" stroke="currentColor" viewBox="0 0 24 24" style="color:#22c55e">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0z"/>
                </svg>
              </div>
              <span class="group-name">{{ row.name }}</span>
            </div>
          </template>
        </el-table-column>

        <el-table-column v-if="!isMobile" label="IM 群组 ID" min-width="140">
          <template #default="{ row }">
            <span class="font-code cell-muted">{{ row.imGroupId }}</span>
          </template>
        </el-table-column>

        <el-table-column v-if="!isMobile" label="归属" min-width="120">
          <template #default="{ row }">
            <span class="cell-owner">{{ row.ownerName || '-' }}</span>
          </template>
        </el-table-column>

        <el-table-column label="成员数" width="80" align="center">
          <template #default="{ row }">
            <span class="cell-count">{{ row.memberCount ?? '-' }}</span>
          </template>
        </el-table-column>

        <el-table-column v-if="!isMobile" label="群主" min-width="120">
          <template #default="{ row }">
            <span class="cell-owner">{{ row.ownerNickname || '-' }}</span>
          </template>
        </el-table-column>

        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <span class="status-badge" :class="row.status === 'active' ? 'status-active' : 'status-dismissed'">
              {{ row.status === 'active' ? '正常' : '已解散' }}
            </span>
          </template>
        </el-table-column>

        <el-table-column v-if="!isMobile" label="创建时间" min-width="160">
          <template #default="{ row }">
            <span class="font-code cell-xs">{{ row.createdAt }}</span>
          </template>
        </el-table-column>

        <el-table-column label="操作" :width="isMobile ? 230 : 290" :fixed="isMobile ? false : 'right'">
          <template #default="{ row }">
            <div class="action-btns">
              <el-button size="small" type="primary" plain style="padding:2px 8px;font-size:12px" @click="openConversationShortcut(row)">群聊</el-button>
              <el-button size="small" plain style="padding:2px 8px;font-size:12px" @click="openMembersModal(row)">成员</el-button>
              <el-button size="small" plain style="padding:2px 8px;font-size:12px" @click="openNoticeModal(row)">公告</el-button>
              <el-button
                v-if="row.status === 'active' && !row.imGroupId"
                size="small" type="warning" plain
                style="padding:2px 8px;font-size:12px"
                @click="handleRepair(row)"
              >同步</el-button>
              <el-button
                size="small" plain
                style="padding:2px 8px;font-size:12px"
                :type="row.isMuted ? 'warning' : ''"
                :disabled="row.status !== 'active'"
                @click="handleMute(row)"
              >{{ row.isMuted ? '取消静默' : '静默' }}</el-button>
              <el-button size="small" type="danger" plain
                style="padding:2px 8px;font-size:12px"
                :disabled="row.status !== 'active'"
                @click="handleDismiss(row)">解散</el-button>
            </div>
          </template>
        </el-table-column>
        </el-table>
      </div>

      <div class="pagination-bar">
        <el-pagination
          v-model:current-page="page"
          :page-size="pageSize"
          :page-sizes="pageSizeOptions"
          :total="total"
          layout="sizes, prev, pager, next"
          background
          small
          @size-change="(size: number) => { pageSize = size; page = 1; fetchGroups() }"
          @current-change="(p) => { page = p; fetchGroups() }" />
      </div>
    </div>

    <el-dialog
      v-model="showCreateModal"
      title="新建群组"
      :width="isMobile ? '100%' : '600px'"
      :fullscreen="isMobile"
      :close-on-click-modal="false"
    >
      <div class="create-body">
        <!-- 群组名称 -->
        <div class="form-field">
          <label class="form-label">群组名称 <span class="req">*</span></label>
          <el-input v-model="groupName" placeholder="请输入群组名称" />
        </div>

        <!-- 归属人（管理员专用） -->
        <div v-if="authStore.isAdmin" class="form-field">
          <label class="form-label">归属人 <span class="req">*</span></label>
          <select
            class="owner-select-field"
            :value="selectedOwnerID ?? ''"
            @change="e => {
              const v = (e.target as HTMLSelectElement).value
              selectedOwnerID = v ? Number(v) : null
              selectedMembers = []
              ownerMemberID = null
              memberList = []
              if (selectedOwnerID) loadMembers()
            }"
          >
            <option value="">— 请先选择归属人 —</option>
            <option v-for="o in ownerManagementOptions" :key="o.value" :value="o.value">{{ o.label }}</option>
          </select>
          <p v-if="!selectedOwnerID" class="form-hint">选择归属人后才能选择成员</p>
        </div>

        <!-- 群类型 -->
        <div class="form-field">
          <label class="form-label">群类型 <span class="req">*</span></label>          <div class="group-type-tags">
            <button
              v-for="opt in GROUP_TYPE_CREATE_OPTIONS" :key="opt.value"
              class="group-type-tag" :class="{ 'group-type-tag--active': groupType === opt.value }"
              :title="opt.desc"
              @click="groupType = opt.value"
            >{{ opt.label }}</button>
          </div>
          <p class="group-type-hint">{{ GROUP_TYPE_CREATE_OPTIONS.find(o => o.value === groupType)?.desc }}</p>
        </div>

        <!-- 成员选择器 -->
        <div class="form-field">
          <label class="form-label">选择初始成员 <span class="form-opt">可选</span></label>
          <div class="member-picker" :class="{ 'member-picker--disabled': authStore.isAdmin && !selectedOwnerID }">
            <div v-if="authStore.isAdmin && !selectedOwnerID" class="member-picker-placeholder">
              请先选择归属人
            </div>
            <template v-else>

            <!-- 搜索框 -->
            <div class="member-search-wrap">
              <svg class="member-search-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/>
              </svg>
              <input
                v-model="memberSearchText"
                class="member-search-input"
                placeholder="搜索昵称或手机号…"
                @input="onSearchInput"
              />
              <span v-if="memberSearchLoading" class="member-search-spin">
                <svg class="spin-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
                </svg>
              </span>
            </div>

            <!-- 用户列表 -->
            <div class="member-list">
              <div v-if="!memberSearchLoading && memberList.length === 0" class="member-empty">暂无用户</div>
              <div
                v-for="u in memberList" :key="u.id"
                class="member-item" :class="{ 'member-item--selected': isSelected(u) }"
                @click="toggleMember(u)"
              >
                <div class="member-avatar">{{ u.nickname?.charAt(0) }}</div>
                <div class="member-info">
                  <span class="member-name">{{ u.nickname }}</span>
                  <span class="member-phone">{{ u.phone }}</span>
                </div>
                <div class="member-check" :class="{ 'member-check--active': isSelected(u) }">
                  <svg v-if="isSelected(u)" width="10" height="10" viewBox="0 0 12 12" fill="currentColor">
                    <path d="M10.28 2.28L4.5 8.06 1.72 5.28a1 1 0 00-1.44 1.44l3.5 3.5a1 1 0 001.44 0l6.5-6.5a1 1 0 00-1.44-1.44z"/>
                  </svg>
                </div>
              </div>
            </div>

            <!-- 已选成员 chips + 指定群主 -->
            <div v-if="selectedMembers.length > 0" class="selected-area">
              <div class="selected-header">
                <span class="selected-label">已选 {{ selectedMembers.length }} 人</span>
                <div class="owner-pick">
                  <span class="owner-pick-label">群主</span>
                  <select class="owner-select" :value="ownerMemberID ?? ''" @change="ownerMemberID = ($event.target as HTMLSelectElement).value ? Number(($event.target as HTMLSelectElement).value) : null">
                    <option value="">不指定</option>
                    <option v-for="m in selectedMembers" :key="m.id" :value="m.id">{{ m.nickname }}</option>
                  </select>
                </div>
              </div>
              <div class="selected-chips">
                <span v-for="m in selectedMembers" :key="m.id" class="member-chip" :class="{ 'member-chip--owner': ownerMemberID === m.id }">
                  <span v-if="ownerMemberID === m.id" class="chip-crown">👑</span>
                  {{ m.nickname }}
                  <button class="chip-remove" @click.stop="removeMember(m)">×</button>
                </span>
              </div>
            </div>

            </template>
          </div>
        </div>
      </div>
      <template #footer>
        <el-button @click="showCreateModal = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleCreate">创建</el-button>
      </template>
    </el-dialog>

    <!-- 群成员对话框 -->
    <el-dialog
      v-model="showMembersModal"
      :title="currentGroup ? `${currentGroup.name} · 成员列表` : '成员列表'"
      :width="isMobile ? '100%' : '560px'"
      :fullscreen="isMobile"
    >
      <div v-loading="membersLoading" class="members-body">
        <div v-if="!membersLoading && memberResults.length === 0" class="members-empty">暂无成员数据</div>
        <table v-else class="members-table">
          <thead>
            <tr>
              <th>IM 用户 ID</th>
              <th>昵称</th>
              <th>手机号</th>
              <th>角色</th>
              <th>加入时间</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="m in memberResults" :key="m.imUserID">
              <td><span class="font-code cell-xs">{{ m.imUserID }}</span></td>
              <td><span class="m-nick">{{ m.nickname || '-' }}</span></td>
              <td><span class="font-code cell-xs">{{ m.phone || '-' }}</span></td>
              <td>
                <span class="role-badge" :class="roleLabel(m.role).cls">{{ roleLabel(m.role).text }}</span>
              </td>
              <td><span class="font-code cell-xs">{{ formatJoinTime(m.joinTime) }}</span></td>
            </tr>
          </tbody>
        </table>
      </div>
      <template #footer>
        <el-button @click="showMembersModal = false">关闭</el-button>
      </template>
    </el-dialog>

    <el-dialog
      v-model="showNoticeModal"
      :title="currentGroup ? `${currentGroup.name} · 群公告` : '群公告'"
      :width="isMobile ? '100%' : '640px'"
      :fullscreen="isMobile"
    >
      <div v-loading="noticeLoading" class="modal-body">
        <el-input
          v-model="currentNotice"
          type="textarea"
          :rows="10"
          maxlength="5000"
          show-word-limit
          placeholder="请输入完整群公告内容"
        />
      </div>
      <template #footer>
        <el-button @click="showNoticeModal = false">取消</el-button>
        <el-button type="primary" :loading="noticeSaving" @click="saveNotice">保存</el-button>
      </template>
    </el-dialog>

    <!-- 同步群组对话框 -->
    <el-dialog
      v-model="showSyncModal"
      title="同步群组"
      :width="isMobile ? '100%' : '800px'"
      :fullscreen="isMobile"
      :close-on-click-modal="false"
    >
      <div v-loading="syncLoading" class="sync-body">
        <div v-if="!syncLoading && syncPreviews.length === 0" class="sync-empty">
          未发现需要同步的群组
        </div>
        <div v-else class="sync-list">
          <div class="sync-header">
            <span class="sync-hint">从腾讯 IM 拉取到 {{ syncPreviews.length }} 个群组，请选择要同步的群组</span>
          </div>
          <div class="sync-items">
            <div
              v-for="item in syncPreviews"
              :key="item.imGroupId"
              class="sync-item"
              :class="{ 'sync-item--selected': item.action !== 'skip' }"
              @click="toggleSyncAction(item)"
            >
              <div class="sync-check">
                <div class="sync-checkbox" :class="{ 'sync-checkbox--active': item.action !== 'skip' }">
                  <svg v-if="item.action !== 'skip'" width="12" height="12" viewBox="0 0 12 12" fill="currentColor">
                    <path d="M10.28 2.28L4.5 8.06 1.72 5.28a1 1 0 00-1.44 1.44l3.5 3.5a1 1 0 001.44 0l6.5-6.5a1 1 0 00-1.44-1.44z"/>
                  </svg>
                </div>
              </div>
              <div class="sync-info">
                <div class="sync-row">
                  <span class="sync-name">{{ item.name }}</span>
                  <span class="sync-badge" :class="item.isNew ? 'sync-badge--new' : 'sync-badge--exist'">
                    {{ item.isNew ? '新群组' : '已存在' }}
                  </span>
                </div>
                <div class="sync-row">
                  <span class="sync-meta">IM ID: <span class="font-code">{{ item.imGroupId }}</span></span>
                  <span class="sync-meta">群主: {{ item.ownerNickname }}</span>
                  <span class="sync-meta">成员: {{ item.memberCount }} 人</span>
                </div>
                <div class="sync-row">
                  <span class="sync-owner">归属: {{ item.ownerName }}</span>
                </div>
              </div>
              <div class="sync-action-label">
                <span v-if="item.action === 'import'" class="action-tag action-import">导入</span>
                <span v-else-if="item.action === 'update'" class="action-tag action-update">更新</span>
                <span v-else class="action-tag action-skip">跳过</span>
              </div>
            </div>
          </div>
        </div>
      </div>
      <template #footer>
        <el-button @click="showSyncModal = false">取消</el-button>
        <el-button type="primary" :loading="syncSubmitting" @click="handleSyncConfirm">确认同步</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.page-header-actions { display:flex; flex-wrap:wrap; justify-content:flex-end; gap:8px; }
.card-toolbar { display: flex; align-items: center; gap: 8px; padding: 12px 16px; border-bottom: 1px solid #f0f0f0; }
.page-table-wrap { border-bottom: 1px solid #f0f0f0; }

.filter-wrap { position: relative; flex: 1; max-width: 760px; }
.filter-box { display: flex; flex-wrap: wrap; align-items: center; gap: 6px; min-height: 40px; padding: 4px 10px; border: 1px solid #f0f0f0; border-radius: 10px; background: #fff; transition: border-color 0.15s, box-shadow 0.15s; }
.filter-box:hover { border-color: rgba(22,119,255,0.4); }
.filter-box:focus-within { border-color: rgba(22,119,255,0.6); box-shadow: 0 0 0 3px rgba(22,119,255,0.08); }
.filter-icon { width: 14px; height: 14px; color: #94a3b8; flex-shrink: 0; }
.filter-tag { display: inline-flex; align-items: center; gap: 3px; padding: 2px 8px; background: rgba(22,119,255,0.1); color: #1677ff; font-size: 11px; font-weight: 500; border-radius: 4px; }
.filter-tag-key { color: rgba(22,119,255,0.6); margin-right: 2px; }
.filter-tag-remove { margin-left: 2px; background: none; border: none; color: rgba(22,119,255,0.5); cursor: pointer; font-size: 14px; line-height: 1; padding: 0; }
.filter-tag-remove:hover { color: #1677ff; }
.filter-search-input {
  flex: 1;
  min-width: 220px;
  height: 30px;
  border: none;
  outline: none;
  background: transparent;
  color: #1e293b;
  font-size: 13px;
}
.filter-search-input::placeholder { color: #94a3b8; }
.filter-actions {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-left: auto;
}
.filter-inline-btn {
  height: 28px;
  padding: 0 10px;
  border: 1px solid #e5e7eb;
  border-radius: 999px;
  background: #fff;
  color: #64748b;
  font-size: 12px;
  cursor: pointer;
  transition: all 0.15s ease;
}
.filter-inline-btn:hover { border-color: #bfdbfe; color: #1677ff; }
.filter-inline-btn-primary {
  border-color: #1677ff;
  background: #1677ff;
  color: #fff;
}
.filter-inline-btn-primary:hover { background: #4096ff; border-color: #4096ff; color: #fff; }
.filter-helper {
  margin-top: 6px;
  padding: 0 12px;
  color: #94a3b8;
  font-size: 11px;
  line-height: 1.5;
}
.filter-helper-divider {
  margin: 0 6px;
  color: #cbd5e1;
}

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
.filter-option-empty { color: #94a3b8; cursor: default; justify-content: center; }
.filter-option-empty:hover { background: transparent; }
.filter-radio { width: 14px; height: 14px; border-radius: 50%; border: 2px solid #d1d5db; flex-shrink: 0; transition: all 0.1s; }
.filter-radio.active { border-color: #1677ff; background: #1677ff; }
.filter-confirm-btn { width: 100%; height: 32px; border-radius: 6px; background: #1677ff; color: #fff; font-size: 12px; font-weight: 500; border: none; cursor: pointer; }
.filter-confirm-btn:hover { background: #4096ff; }

.total-count { font-size: 12px; color: #94a3b8; margin-left: auto; white-space: nowrap; }

.batch-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 16px;
  border-bottom: 1px solid #f0f0f0;
  background: #fafcff;
}

.batch-count {
  font-size: 12px;
  font-weight: 500;
  color: #475569;
}

.batch-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.cell-id   { font-size: 13px; color: #94a3b8; }
.cell-muted { font-size: 12px; color: #64748b; }
.cell-xs   { font-size: 11px; color: #94a3b8; }

.group-cell   { display: flex; align-items: center; gap: 10px; }
.group-avatar { width: 28px; height: 28px; border-radius: 7px; background: #f0fdf4; display: flex; align-items: center; justify-content: center; flex-shrink: 0; }
.group-name   { font-size: 13px; font-weight: 500; color: #1e293b; }

.status-badge { display: inline-flex; align-items: center; font-size: 12px; font-weight: 500; padding: 0 9px; height: 24px; border-radius: 6px; border: 1px solid transparent; }
.status-active    { color: #52c41a; border-color: #b7eb8f; background: #f6ffed; }
.status-dismissed { color: #8c8c8c; border-color: #d9d9d9; background: #fafafa; }

.action-btns { display: flex; align-items: center; gap: 6px; }
.pagination-bar { display: flex; justify-content: flex-end; padding: 12px 16px; border-top: 1px solid #f0f0f0; }
.modal-body { display: flex; flex-direction: column; gap: 16px; }
.form-field { display: flex; flex-direction: column; gap: 6px; }
.form-label { font-size: 12px; font-weight: 500; color: #475569; }
.req  { color: #ff4d4f; margin-left: 2px; }
.form-opt { font-size: 11px; font-weight: 400; color: #94a3b8; margin-left: 4px; }

/* 群类型选择 */
.group-type-tags { display: flex; flex-wrap: wrap; gap: 6px; }
.group-type-tag { height: 28px; padding: 0 12px; border: 1px solid #e2e8f0; border-radius: 6px; background: #fff; font-size: 12px; font-weight: 500; color: #475569; cursor: pointer; transition: all 0.15s; }
.group-type-tag:hover { border-color: rgba(22,119,255,0.4); color: #1677ff; }
.group-type-tag--active { border-color: #1677ff; background: rgba(22,119,255,0.08); color: #1677ff; }
.group-type-hint { font-size: 11px; color: #94a3b8; margin: 0; }

/* 弹窗 */
.create-body { display: flex; flex-direction: column; gap: 16px; }

/* 成员选择器 */
.member-picker { border: 1px solid #f0f0f0; border-radius: 8px; overflow: hidden; }

.member-search-wrap { display: flex; align-items: center; gap: 8px; padding: 8px 12px; border-bottom: 1px solid #f0f0f0; background: #fafafa; }
.member-search-icon { width: 14px; height: 14px; color: #94a3b8; flex-shrink: 0; }
.member-search-input { flex: 1; border: none; background: transparent; outline: none; font-size: 13px; color: #1e293b; }
.member-search-input::placeholder { color: #94a3b8; }
.member-search-spin { display: flex; }
.spin-icon { width: 14px; height: 14px; color: #94a3b8; animation: spin 0.8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }

.member-list { max-height: 240px; overflow-y: auto; }
.member-empty { padding: 32px; text-align: center; font-size: 13px; color: #94a3b8; }
.member-item { display: flex; align-items: center; gap: 10px; padding: 8px 12px; cursor: pointer; transition: background 0.1s; user-select: none; }
.member-item:hover { background: #f8fafc; }
.member-item--selected { background: rgba(22,119,255,0.04); }

.member-avatar { width: 28px; height: 28px; border-radius: 7px; background: rgba(22,119,255,0.1); display: flex; align-items: center; justify-content: center; color: #1677ff; font-size: 12px; font-weight: 600; flex-shrink: 0; }
.member-info { flex: 1; display: flex; flex-direction: column; gap: 1px; min-width: 0; }
.member-name { font-size: 13px; font-weight: 500; color: #1e293b; }
.member-phone { font-size: 11px; color: #94a3b8; font-family: 'Fira Code', monospace; }

.member-check { width: 18px; height: 18px; border-radius: 50%; border: 2px solid #e2e8f0; display: flex; align-items: center; justify-content: center; flex-shrink: 0; transition: all 0.15s; }
.member-check--active { border-color: #1677ff; background: #1677ff; color: #fff; }

/* 已选区域 */
.selected-area { border-top: 1px solid #f0f0f0; padding: 10px 12px; background: #fafafa; display: flex; flex-direction: column; gap: 8px; }
.selected-header { display: flex; align-items: center; justify-content: space-between; }
.selected-label { font-size: 11px; font-weight: 600; color: #94a3b8; text-transform: uppercase; letter-spacing: 0.05em; }
.owner-pick { display: flex; align-items: center; gap: 6px; }
.owner-pick-label { font-size: 11px; font-weight: 600; color: #475569; }
.owner-select { height: 24px; padding: 0 6px; font-size: 12px; border: 1px solid #e2e8f0; border-radius: 4px; outline: none; background: #fff; color: #1e293b; cursor: pointer; }
.owner-select:focus { border-color: rgba(22,119,255,0.6); }
.owner-select-field { width: 100%; height: 36px; padding: 0 10px; font-size: 13px; border: 1px solid #e2e8f0; border-radius: 7px; outline: none; background: #fff; color: #1e293b; cursor: pointer; }
.owner-select-field:focus { border-color: rgba(22,119,255,0.6); }
.form-hint { font-size: 11px; color: #94a3b8; margin-top: 5px; }
.member-picker--disabled { opacity: 0.5; pointer-events: none; }
.member-picker-placeholder { padding: 24px; text-align: center; color: #94a3b8; font-size: 13px; }
.selected-chips { display: flex; flex-wrap: wrap; gap: 6px; }
.member-chip { display: inline-flex; align-items: center; gap: 4px; padding: 2px 8px; background: rgba(22,119,255,0.1); color: #1677ff; font-size: 12px; font-weight: 500; border-radius: 20px; }
.member-chip--owner { background: rgba(250,173,20,0.15); color: #d48806; }
.chip-crown { font-size: 11px; }
.chip-remove { background: none; border: none; color: rgba(22,119,255,0.5); cursor: pointer; font-size: 14px; line-height: 1; padding: 0; display: flex; }
.chip-remove:hover { color: #1677ff; }

/* 归属 / 成员数 */
.cell-owner { font-size: 13px; color: #475569; }
.cell-count { font-size: 13px; font-weight: 600; color: #1e293b; }

/* 成员对话框 */
.members-body { min-height: 120px; }
.members-empty { padding: 48px; text-align: center; font-size: 13px; color: #94a3b8; }
.members-table { width: 100%; border-collapse: collapse; font-size: 13px; }
.members-table th { padding: 8px 10px; text-align: left; font-size: 11px; font-weight: 600; color: #94a3b8; text-transform: uppercase; letter-spacing: 0.04em; border-bottom: 1px solid #f0f0f0; }
.members-table td { padding: 8px 10px; border-bottom: 1px solid #fafafa; vertical-align: middle; }
.m-nick { font-weight: 500; color: #1e293b; }
.role-badge { display: inline-flex; align-items: center; font-size: 11px; font-weight: 500; padding: 1px 7px; height: 20px; border-radius: 4px; border: 1px solid transparent; }
.role-owner  { color: #d48806; border-color: #ffd666; background: #fffbe6; }
.role-admin  { color: #1677ff; border-color: #91caff; background: #e6f4ff; }
.role-member { color: #8c8c8c; border-color: #d9d9d9; background: #fafafa; }

/* 同步对话框 */
.sync-body { min-height: 200px; max-height: 500px; overflow-y: auto; }
.sync-empty { padding: 48px; text-align: center; font-size: 13px; color: #94a3b8; }
.sync-list { display: flex; flex-direction: column; gap: 12px; }
.sync-header { padding: 0 0 12px; border-bottom: 1px solid #f0f0f0; }
.sync-hint { font-size: 12px; color: #64748b; }
.sync-items { display: flex; flex-direction: column; gap: 8px; }
.sync-item { display: flex; align-items: center; gap: 12px; padding: 12px; border: 1px solid #f0f0f0; border-radius: 8px; cursor: pointer; transition: all 0.15s; }
.sync-item:hover { border-color: rgba(22,119,255,0.3); background: #fafafa; }
.sync-item--selected { border-color: #1677ff; background: rgba(22,119,255,0.04); }
.sync-check { flex-shrink: 0; }
.sync-checkbox { width: 18px; height: 18px; border-radius: 4px; border: 2px solid #e2e8f0; display: flex; align-items: center; justify-content: center; transition: all 0.15s; }
.sync-checkbox--active { border-color: #1677ff; background: #1677ff; color: #fff; }
.sync-info { flex: 1; display: flex; flex-direction: column; gap: 4px; min-width: 0; }
.sync-row { display: flex; align-items: center; gap: 12px; flex-wrap: wrap; }
.sync-name { font-size: 14px; font-weight: 500; color: #1e293b; }
.sync-badge { display: inline-flex; align-items: center; font-size: 11px; font-weight: 500; padding: 1px 7px; height: 18px; border-radius: 4px; }
.sync-badge--new { color: #52c41a; border: 1px solid #b7eb8f; background: #f6ffed; }
.sync-badge--exist { color: #1677ff; border: 1px solid #91caff; background: #e6f4ff; }
.sync-meta { font-size: 12px; color: #94a3b8; }
.sync-owner { font-size: 12px; color: #64748b; }
.sync-action-label { flex-shrink: 0; }
.action-tag { display: inline-flex; align-items: center; font-size: 11px; font-weight: 500; padding: 2px 8px; height: 22px; border-radius: 4px; }
.action-import { color: #52c41a; border: 1px solid #b7eb8f; background: #f6ffed; }
.action-update { color: #faad14; border: 1px solid #ffd666; background: #fffbe6; }
.action-skip { color: #8c8c8c; border: 1px solid #d9d9d9; background: #fafafa; }

@media (max-width: 960px) {
  .page-header-actions {
    width: 100%;
  }

  .page-header-actions :deep(.el-button) {
    flex: 1 1 160px;
  }

  .card-toolbar {
    flex-direction: column;
    align-items: stretch;
  }

  .filter-wrap {
    max-width: none;
  }

  .filter-search-input {
    min-width: 140px;
  }

  .filter-helper {
    padding: 0 4px;
  }

  .filter-helper-divider {
    display: none;
  }

  .total-count {
    margin-left: 0;
  }
}

@media (max-width: 767px) {
  .batch-toolbar,
  .batch-actions,
  .pagination-bar {
    flex-direction: column;
    align-items: stretch;
  }

  .page-header-actions :deep(.el-button),
  .batch-actions :deep(.el-button) {
    width: 100%;
    margin-left: 0;
  }

  .action-btns {
    flex-direction: column;
  }

  .members-body,
  .sync-body {
    overflow-x: auto;
  }

  .members-table {
    min-width: 620px;
  }
}
</style>
