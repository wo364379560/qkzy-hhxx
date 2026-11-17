<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  Avatar,
  Calendar,
  ChatDotRound,
  DataBoard,
  Document,
  Histogram,
  List,
  Location,
  MoreFilled,
  Ticket,
  User,
  UserFilled,
  Warning,
} from '@element-plus/icons-vue'
import { useAuthStore } from '@/stores/auth'
import { useAppStore, type VisitedTab } from '@/stores/app'
import { getAssistantMe } from '@/api/assistant'
import type { AssistantDetail } from '@/types'
import { useResponsive } from '@/composables/useResponsive'

interface NavItem {
  to: string
  label: string
  icon: any
  visible?: boolean
}

interface NavSection {
  label: string
  items: NavItem[]
}

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const appStore = useAppStore()
const { isMobile } = useResponsive()

const collapsed = ref(false)
const isMobileNavOpen = ref(false)
const assistantMe = ref<AssistantDetail | null>(null)

const pageTitleMap: Record<string, string> = {
  Dashboard: '仪表盘',
  OperationCenter: '运营治理中心',
  CleanupRiskControl: '清理风控',
  ServerSecurity: '服务器防护',
  Admins: '管理员',
  Settings: '账号设置',
  Users: '用户管理',
  InviteCodes: '邀请码',
  IPWhitelistPolicies: '白名单策略',
  BehaviorAudits: '行为审计',
  LoginAudits: '登录审计',
  QuietHoursHitLogs: '静默时段命中日志',
  SMSSendLogs: '短信发送日志',
  UserOperationLogs: '用户操作日志',
  Reports: '举报管理',
  CommunityPosts: '省钱社区内容',
  Groups: '群组管理',
  OfflineActivities: '线下省钱活动',
  Assistants: '助理管理',
  OwnershipOperations: '好友归属处理',
  C2C: 'C2C 消息',
  ChatKeywords: '关键词屏蔽',
  Conversations: '会话列表',
  ForwardJobs: '转发任务',
}

const roleLabel = computed(() => {
  if (authStore.isSuperAdmin) return '超级管理员'
  if (authStore.isManager) return '管理员'
  return '助理'
})

const renderCollapsed = computed(() => !isMobile.value && collapsed.value)
const sidebarWidth = computed(() => (renderCollapsed.value ? '72px' : '232px'))
function routeQueryText(key: string) {
  const value = route.query[key]
  return Array.isArray(value) ? String(value[0] || '') : String(value || '')
}

function resolveCurrentPageTitle() {
  if (route.name === 'Conversations') {
    if (route.query.mode === 'group') {
      const groupName = routeQueryText('name')
      return groupName ? `群聊：${groupName}` : '会话列表'
    }
    const nickname = routeQueryText('nickname')
    return nickname ? `会话：${nickname}` : '会话列表'
  }
  return pageTitleMap[String(route.name || '')] || '管理后台'
}

const currentPageTitle = computed(resolveCurrentPageTitle)
const activeTabKey = computed(() => String(route.name || ''))
const userInitial = computed(() => authStore.user?.username?.charAt(0).toUpperCase() || 'W')

const navSections = computed<NavSection[]>(() => {
  const adminSections: NavSection[] = [
    {
      label: '主菜单',
      items: [
        { to: '/dashboard', label: '仪表盘', icon: DataBoard },
      ],
    },
    {
      label: '系统管理',
      items: [
        { to: '/admins', label: '管理员', icon: UserFilled, visible: authStore.canManageAdmins },
        { to: '/assistants', label: '助理', icon: UserFilled },
        { to: '/ownership-operations', label: '好友归属处理', icon: Document, visible: authStore.isSuperAdmin },
        { to: '/invite-codes', label: '邀请码', icon: Ticket },
      ],
    },
    {
      label: '安全中心',
      items: [
        { to: '/operation', label: '运营治理中心', icon: DataBoard, visible: authStore.isSuperAdmin },
        { to: '/operation/cleanup', label: '清理风控', icon: Warning, visible: authStore.isSuperAdmin },
        { to: '/operation/server-security', label: '服务器防护', icon: Location, visible: authStore.isSuperAdmin },
        { to: '/ip-whitelists', label: '白名单策略', icon: Document },
        { to: '/audits/access', label: '行为审计', icon: Document },
        { to: '/audits/logins', label: '登录审计', icon: Histogram },
        { to: '/audits/quiet-hours', label: '静默时段命中', icon: Histogram },
        { to: '/audits/sms-sends', label: '短信发送日志', icon: Histogram, visible: authStore.isSuperAdmin },
        { to: '/audits/user-operations', label: '用户操作日志', icon: Document, visible: authStore.isSuperAdmin },
      ],
    },
    {
      label: '数据管理',
      items: [
        { to: '/users', label: '用户', icon: Avatar },
        { to: '/community/posts', label: '省钱社区内容', icon: Document },
        { to: '/groups', label: '群组', icon: User },
        { to: '/reports', label: '举报管理', icon: Warning },
      ],
    },
    {
      label: '运营管理',
      items: [
        { to: '/offline-activities', label: '线下省钱活动', icon: Calendar },
      ],
    },
    {
      label: 'IM 工具',
      items: [
        { to: '/im/c2c', label: 'C2C 消息', icon: ChatDotRound },
        { to: '/im/keywords', label: '关键词屏蔽', icon: Document, visible: authStore.isSuperAdmin },
        { to: '/im/conversations', label: '会话列表', icon: List },
        { to: '/im/forward-jobs', label: '转发任务', icon: List },
      ],
    },
  ]

  const assistantSections: NavSection[] = [
    {
      label: '主菜单',
      items: [
        { to: '/dashboard', label: '仪表盘', icon: DataBoard },
      ],
    },
    {
      label: '安全中心',
      items: [
        { to: '/audits/access', label: '行为审计', icon: Document },
        { to: '/audits/logins', label: '登录审计', icon: Histogram },
      ],
    },
    {
      label: '数据管理',
      items: [
        { to: '/users', label: '用户', icon: Avatar },
        { to: '/community/posts', label: '省钱社区内容', icon: Document },
        { to: '/groups', label: '群组', icon: User },
        { to: '/reports', label: '举报管理', icon: Warning },
      ],
    },
    {
      label: 'IM 工具',
      items: [
        { to: '/im/c2c', label: 'C2C 消息', icon: ChatDotRound },
        { to: '/im/conversations', label: '会话列表', icon: List },
        { to: '/im/forward-jobs', label: '转发任务', icon: List },
      ],
    },
  ]

  const source = authStore.isAdmin ? adminSections : assistantSections
  return source
    .map((section) => ({
      ...section,
      items: section.items.filter((item) => item.visible !== false),
    }))
    .filter((section) => section.items.length > 0)
})

function inviteStatusText(status?: string) {
  if (status === 'expired') return '已过期'
  if (status === 'used') return '已用尽'
  return '有效'
}

function closeMobileNav() {
  isMobileNavOpen.value = false
}

function toggleSidebar() {
  if (isMobile.value) {
    isMobileNavOpen.value = !isMobileNavOpen.value
    return
  }
  collapsed.value = !collapsed.value
}

function handleNavClick() {
  if (isMobile.value) {
    closeMobileNav()
  }
}

function handleHeaderAction(action: 'settings' | 'logout') {
  if (action === 'settings') {
    router.push('/settings')
    closeMobileNav()
    return
  }
  appStore.resetVisitedTabs()
  authStore.logout()
}

function openVisitedTab(tab: VisitedTab) {
  if (tab.path !== route.fullPath) {
    void router.push(tab.path)
  }
}

function closeVisitedTab(tab: VisitedTab) {
  if (!tab.closable) return
  const tabs = appStore.visitedTabs
  const closingIndex = tabs.findIndex(item => item.key === tab.key)
  if (closingIndex < 0) return
  const isCurrent = activeTabKey.value === tab.key
  const fallback = tabs[closingIndex - 1] || tabs[closingIndex + 1]
  appStore.closeVisitedTab(tab.key)
  if (isCurrent) {
    void router.push(fallback?.path || '/dashboard')
  }
}

onMounted(async () => {
  if (!authStore.isAdmin) {
    try {
      const res = await getAssistantMe()
      assistantMe.value = res.data.data
    } catch {
      // ignore
    }
  }
})

watch(isMobile, (mobile) => {
  if (mobile) {
    collapsed.value = false
    return
  }
  isMobileNavOpen.value = false
})

watch(
  () => route.fullPath,
  () => {
    closeMobileNav()
    const key = String(route.name || '')
    if (!key || key === 'Login') return
    appStore.upsertVisitedTab({
      key,
      title: resolveCurrentPageTitle(),
      path: route.fullPath,
      closable: key !== 'Dashboard',
    })
  },
  { immediate: true },
)
</script>

<template>
  <el-container class="layout" :class="{ 'layout-mobile': isMobile }">
    <div v-if="isMobileNavOpen" class="mobile-nav-mask" @click="closeMobileNav"></div>

    <el-aside
      :width="isMobile ? '0px' : sidebarWidth"
      class="aside"
      :class="{
        collapsed: renderCollapsed,
        'aside-mobile': isMobile,
        'aside-mobile-open': isMobileNavOpen,
      }"
    >
      <router-link class="logo" to="/dashboard" @click="handleNavClick">
        <div class="logo-icon">
          <img src="/logo.png" alt="睿之禾 logo" class="logo-image" />
        </div>
        <div v-show="!renderCollapsed" class="logo-text">
          <span class="logo-name">睿之禾</span>
          <span class="logo-sub">管理后台</span>
        </div>
      </router-link>

      <nav class="nav">
        <section v-for="section in navSections" :key="section.label" class="nav-section">
          <div v-show="!renderCollapsed" class="nav-section-label">{{ section.label }}</div>
          <router-link
            v-for="item in section.items"
            :key="item.to"
            :to="item.to"
            class="nav-item"
            @click="handleNavClick"
          >
            <el-icon><component :is="item.icon" /></el-icon>
            <span v-show="!renderCollapsed">{{ item.label }}</span>
          </router-link>
        </section>
      </nav>
    </el-aside>

    <el-container class="main-container">
      <el-header class="header" :height="isMobile ? 'auto' : '64px'">
        <div class="header-row">
          <div class="header-left">
            <button
              class="collapse-btn"
              @click="toggleSidebar"
              :title="isMobile ? '打开导航' : renderCollapsed ? '展开侧边栏' : '收起侧边栏'"
            >
              <svg width="16" height="16" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16" />
              </svg>
            </button>

            <div class="header-heading">
              <span class="header-page">{{ currentPageTitle }}</span>
              <div class="header-meta">
                <span class="header-role" :class="authStore.isAdmin ? 'role-badge-admin' : 'role-badge-assistant'">
                  {{ roleLabel }}
                </span>
              </div>
            </div>

            <div v-if="!isMobile && !authStore.isAdmin && assistantMe" class="invite-chip">
              <svg width="12" height="12" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M15 5v2m0 4v2m0 4v2M5 5a2 2 0 00-2 2v3a2 2 0 110 4v3a2 2 0 002 2h14a2 2 0 002-2v-3a2 2 0 110-4V7a2 2 0 00-2-2H5z"
                />
              </svg>
              <span v-if="assistantMe.inviteCode" class="invite-chip-code font-code">{{ assistantMe.inviteCode }}</span>
              <span v-else class="invite-chip-none">未分配</span>
              <span v-if="assistantMe.inviteCode" class="invite-chip-usage">
                {{ assistantMe.inviteCodeUsed }}/{{ assistantMe.inviteCodeMax <= 0 ? '∞' : assistantMe.inviteCodeMax }}
              </span>
              <span
                v-if="assistantMe.inviteCode"
                class="invite-chip-status"
                :class="`status-${assistantMe.inviteCodeStatus}`"
              >
                {{ inviteStatusText(assistantMe.inviteCodeStatus) }}
              </span>
            </div>
          </div>

          <div v-if="!isMobile" class="header-right">
            <button class="hdr-btn" @click="handleHeaderAction('settings')">
              <svg width="14" height="14" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"
                />
                <circle cx="12" cy="12" r="3"></circle>
              </svg>
              账号设置
            </button>
            <button class="hdr-btn hdr-btn-danger" @click="handleHeaderAction('logout')">
              <svg width="14" height="14" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1" />
              </svg>
              退出登录
            </button>

            <div class="hdr-divider"></div>

            <div class="user-info">
              <div class="user-avatar" :class="authStore.isAdmin ? 'avatar-admin' : 'avatar-assistant'">
                {{ userInitial }}
              </div>
              <div class="user-text">
                <span class="user-name">{{ authStore.user?.username }}</span>
                <span class="user-role" :class="authStore.isAdmin ? 'role-admin' : 'role-assistant'">
                  <span class="role-dot"></span>{{ roleLabel }}
                </span>
              </div>
            </div>
          </div>

          <div v-else class="header-right-mobile">
            <el-dropdown trigger="click" placement="bottom-end">
              <button class="mobile-user-trigger">
                <div class="user-avatar" :class="authStore.isAdmin ? 'avatar-admin' : 'avatar-assistant'">
                  {{ userInitial }}
                </div>
                <el-icon><MoreFilled /></el-icon>
              </button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item @click="handleHeaderAction('settings')">账号设置</el-dropdown-item>
                  <el-dropdown-item divided @click="handleHeaderAction('logout')">退出登录</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>
        </div>

        <div v-if="isMobile && !authStore.isAdmin && assistantMe" class="header-mobile-meta">
          <div class="invite-chip invite-chip-mobile">
            <svg width="12" height="12" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M15 5v2m0 4v2m0 4v2M5 5a2 2 0 00-2 2v3a2 2 0 110 4v3a2 2 0 002 2h14a2 2 0 002-2v-3a2 2 0 110-4V7a2 2 0 00-2-2H5z"
              />
            </svg>
            <span v-if="assistantMe.inviteCode" class="invite-chip-code font-code">{{ assistantMe.inviteCode }}</span>
            <span v-else class="invite-chip-none">未分配邀请码</span>
            <span
              v-if="assistantMe.inviteCode"
              class="invite-chip-status"
              :class="`status-${assistantMe.inviteCodeStatus}`"
            >
              {{ inviteStatusText(assistantMe.inviteCodeStatus) }}
            </span>
          </div>
        </div>
      </el-header>

      <nav class="visited-tabs" aria-label="已访问页面">
        <button
          v-for="tab in appStore.visitedTabs"
          :key="tab.key"
          type="button"
          class="visited-tab"
          :class="{ 'visited-tab--active': activeTabKey === tab.key }"
          :title="tab.title"
          @click="openVisitedTab(tab)"
        >
          <span class="visited-tab-title">{{ tab.title }}</span>
          <span
            v-if="tab.closable"
            class="visited-tab-close"
            role="button"
            :aria-label="`关闭${tab.title}`"
            @click.stop="closeVisitedTab(tab)"
          >
            ×
          </span>
        </button>
      </nav>

      <el-main class="main">
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<style scoped>
.layout {
  min-height: 100vh;
  background: #f5f7fa;
}

.main-container {
  min-width: 0;
}

.aside {
  position: relative;
  z-index: 30;
  background: #fff;
  border-right: 1px solid #eef2f7;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  transition: width 0.24s ease, transform 0.24s ease, box-shadow 0.24s ease;
}

.aside-mobile {
  position: fixed;
  top: 0;
  left: 0;
  width: min(82vw, 296px) !important;
  height: 100vh;
  transform: translateX(-100%);
  box-shadow: none;
}

.aside-mobile-open {
  transform: translateX(0);
  box-shadow: 0 20px 60px rgba(15, 23, 42, 0.24);
}

.mobile-nav-mask {
  position: fixed;
  inset: 0;
  z-index: 25;
  background: rgba(15, 23, 42, 0.42);
  backdrop-filter: blur(2px);
}

.logo {
  height: 68px;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 0 18px;
  border-bottom: 1px solid #f5f7f9;
  flex-shrink: 0;
  overflow: hidden;
  color: inherit;
  text-decoration: none;
}

.collapsed .logo {
  justify-content: center;
  padding: 0;
}

.logo-icon {
  flex-shrink: 0;
}

.logo-image {
  width: 34px;
  height: 34px;
  border-radius: 10px;
  object-fit: cover;
}

.logo-text {
  display: flex;
  flex-direction: column;
  line-height: 1.15;
  white-space: nowrap;
}

.logo-name {
  font-size: 15px;
  font-weight: 700;
  color: #1677ff;
  letter-spacing: -0.02em;
}

.logo-sub {
  margin-top: 3px;
  font-size: 11px;
  color: #94a3b8;
}

.nav {
  flex: 1;
  padding: 12px 10px 14px;
  overflow-y: auto;
  overflow-x: hidden;
}

.collapsed .nav {
  padding: 12px 8px 14px;
}

.nav-section + .nav-section {
  margin-top: 10px;
}

.nav-section-label {
  margin: 0 0 4px;
  padding: 0 8px;
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: #b0b7c3;
  white-space: nowrap;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 42px;
  padding: 9px 10px;
  border-radius: 10px;
  color: #64748b;
  font-size: 13.5px;
  font-weight: 500;
  text-decoration: none;
  transition: all 0.18s ease;
  cursor: pointer;
  margin-bottom: 2px;
  white-space: nowrap;
  overflow: hidden;
}

.collapsed .nav-item {
  justify-content: center;
  padding: 9px 0;
}

.nav-item:hover {
  background: #f0f6ff;
  color: #1677ff;
}

.nav-item.router-link-active {
  background: #e6f4ff;
  color: #1677ff;
  font-weight: 600;
}

.nav-item .el-icon {
  font-size: 16px;
  flex-shrink: 0;
}

.header {
  background: rgba(255, 255, 255, 0.92);
  border-bottom: 1px solid #eef2f7;
  backdrop-filter: blur(10px);
  padding: calc(10px + var(--app-safe-top)) calc(20px + var(--app-safe-right)) 10px calc(20px + var(--app-safe-left));
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 10px;
}

.header-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  min-width: 0;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
  flex: 1;
}

.collapse-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border: 1px solid #dbe4f0;
  border-radius: 10px;
  background: #fff;
  color: #64748b;
  cursor: pointer;
  transition: all 0.18s ease;
  flex-shrink: 0;
}

.collapse-btn:hover {
  border-color: #1677ff;
  color: #1677ff;
  background: #f0f7ff;
}

.header-heading {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.header-page {
  font-size: 18px;
  font-weight: 700;
  color: #0f172a;
  letter-spacing: -0.02em;
}

.header-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.header-role {
  display: inline-flex;
  align-items: center;
  min-height: 24px;
  padding: 0 10px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 700;
}

.role-badge-admin {
  color: #1677ff;
  background: rgba(22, 119, 255, 0.1);
}

.role-badge-assistant {
  color: #7c3aed;
  background: rgba(124, 58, 237, 0.1);
}

.header-right {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.header-right-mobile {
  flex-shrink: 0;
}

.mobile-user-trigger {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-height: 40px;
  padding: 0 10px 0 6px;
  border: 1px solid #dbe4f0;
  border-radius: 999px;
  background: #fff;
  color: #475569;
  cursor: pointer;
}

.hdr-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 34px;
  padding: 0 12px;
  border: 1px solid #e2e8f0;
  border-radius: 10px;
  background: #fff;
  color: #475569;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.18s ease;
  white-space: nowrap;
}

.hdr-btn:hover {
  border-color: #1677ff;
  color: #1677ff;
  background: rgba(22, 119, 255, 0.04);
}

.hdr-btn-danger {
  color: #ef4444;
  border-color: #fecaca;
}

.hdr-btn-danger:hover {
  border-color: #ef4444;
  background: rgba(239, 68, 68, 0.05);
  color: #ef4444;
}

.hdr-divider {
  width: 1px;
  height: 24px;
  background: #e9eef5;
  margin: 0 4px;
}

.invite-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 30px;
  padding: 0 10px;
  background: #f5f0ff;
  border: 1px solid #e9d5ff;
  border-radius: 10px;
  color: #7c3aed;
  font-size: 12px;
  white-space: nowrap;
}

.invite-chip-mobile {
  width: 100%;
  justify-content: flex-start;
  overflow: hidden;
}

.invite-chip-code {
  font-size: 12px;
  font-weight: 700;
  color: #5b21b6;
}

.invite-chip-none {
  color: #8b5cf6;
}

.invite-chip-usage {
  padding: 1px 6px;
  border-radius: 999px;
  font-size: 11px;
  color: #9333ea;
  background: rgba(147, 51, 234, 0.1);
}

.invite-chip-status {
  padding: 1px 6px;
  border-radius: 999px;
  font-size: 10px;
  font-weight: 700;
}

.status-active { color: #16a34a; background: #dcfce7; }
.status-expired { color: #dc2626; background: #fee2e2; }
.status-used { color: #92400e; background: #fef3c7; }

.header-mobile-meta {
  display: flex;
  align-items: center;
}

.user-info {
  display: flex;
  align-items: center;
  gap: 9px;
}

.user-avatar {
  width: 32px;
  height: 32px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 12px;
  font-weight: 700;
  flex-shrink: 0;
}

.avatar-admin {
  background: linear-gradient(135deg, #1677ff 0%, #60a5fa 100%);
}

.avatar-assistant {
  background: linear-gradient(135deg, #9333ea 0%, #c084fc 100%);
}

.user-text {
  display: flex;
  flex-direction: column;
  gap: 4px;
  line-height: 1;
}

.user-name {
  font-size: 13px;
  font-weight: 700;
  color: #1e293b;
}

.user-role {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  font-weight: 700;
}

.role-dot {
  width: 5px;
  height: 5px;
  border-radius: 999px;
  flex-shrink: 0;
}

.role-admin {
  color: #1677ff;
}

.role-admin .role-dot {
  background: #1677ff;
}

.role-assistant {
  color: #9333ea;
}

.role-assistant .role-dot {
  background: #9333ea;
}

.visited-tabs {
  display: flex;
  align-items: flex-end;
  flex-shrink: 0;
  gap: 6px;
  min-width: 0;
  min-height: 44px;
  padding: 6px calc(20px + var(--app-safe-right)) 0 calc(20px + var(--app-safe-left));
  overflow-x: auto;
  overflow-y: hidden;
  border-bottom: 1px solid #dfe6ef;
  background: #f8fafc;
}

.visited-tabs::-webkit-scrollbar {
  height: 4px;
}

.visited-tabs::-webkit-scrollbar-thumb {
  border-radius: 999px;
  background: #cbd5e1;
}

.visited-tab {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  flex: 0 0 auto;
  max-width: 220px;
  min-height: 34px;
  padding: 0 12px;
  border: 1px solid #dbe4f0;
  border-bottom: 0;
  border-radius: 9px 9px 0 0;
  background: #eef2f7;
  color: #64748b;
  font-size: 13px;
  cursor: pointer;
  transition: color 0.18s ease, background 0.18s ease, border-color 0.18s ease;
}

.visited-tab:hover {
  color: #1677ff;
  background: #fff;
}

.visited-tab--active {
  border-color: #bdd7ff;
  background: #fff;
  color: #1677ff;
  font-weight: 600;
}

.visited-tab-title {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.visited-tab-close {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: 0 0 auto;
  width: 18px;
  height: 18px;
  border-radius: 50%;
  color: #94a3b8;
  font-size: 16px;
  font-weight: 400;
  line-height: 1;
}

.visited-tab-close:hover {
  background: #fee2e2;
  color: #dc2626;
}

.main {
  padding: 0;
  overflow-y: auto;
  background: #eef0f5;
}

.main::-webkit-scrollbar {
  width: 6px;
}

.main::-webkit-scrollbar-track {
  background: transparent;
}

.main::-webkit-scrollbar-thumb {
  background: #d1d9e6;
  border-radius: 999px;
}

@media (max-width: 1199px) {
  .header {
    padding-right: calc(16px + var(--app-safe-right));
    padding-left: calc(16px + var(--app-safe-left));
  }

  .header-page {
    font-size: 17px;
  }
}

@media (max-width: 767px) {
  .header {
    padding: calc(8px + var(--app-safe-top)) calc(12px + var(--app-safe-right)) 10px calc(12px + var(--app-safe-left));
    gap: 8px;
  }

  .header-row {
    align-items: flex-start;
  }

  .header-left {
    gap: 10px;
  }

  .header-page {
    font-size: 16px;
    line-height: 1.2;
  }

  .header-role {
    min-height: 22px;
    padding: 0 8px;
    font-size: 11px;
  }

  .visited-tabs {
    min-height: 42px;
    padding-right: calc(12px + var(--app-safe-right));
    padding-left: calc(12px + var(--app-safe-left));
  }

  .nav {
    padding-bottom: calc(18px + var(--app-safe-bottom));
  }

  .nav-item {
    min-height: 44px;
    font-size: 14px;
  }
}
</style>
