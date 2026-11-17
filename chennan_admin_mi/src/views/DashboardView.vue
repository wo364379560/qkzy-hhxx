<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { useRouter } from 'vue-router'
import { getStats, type StatsData } from '@/api/assistant'
import { getAssistantMe } from '@/api/assistant'
import type { AssistantDetail } from '@/types'

const authStore = useAuthStore()
const router = useRouter()

const loading = ref(true)
const kpi = ref<StatsData>({
  userCount: 0,
  inviteCodeCount: 0,
  groupCount: 0,
  assistantCount: 0,
  reportCount: 0,
  pendingReportCount: 0,
})

const assistantMe = ref<AssistantDetail | null>(null)
const roleLabel = computed(() => {
  if (authStore.isSuperAdmin) return '超级管理员'
  if (authStore.isManager) return '管理员'
  return '助理'
})
const dashboardScopeText = computed(() => {
  if (authStore.isSuperAdmin) return '平台全量统计'
  if (authStore.isManager) return '我归属范围内的数据统计'
  return '我的数据统计'
})

onMounted(async () => {
  try {
    const res = await getStats()
    if (res.data.data) kpi.value = res.data.data
  } finally {
    loading.value = false
  }
  if (!authStore.isAdmin) {
    try {
      const res = await getAssistantMe()
      assistantMe.value = res.data.data
    } catch {
      // ignore
    }
  }
})

// 助理视图：4 张卡；管理员视图：6 张卡
const adminCards = computed(() => [
  {
    key: 'user',
    label: '总用户数',
    value: kpi.value.userCount,
    desc: '注册用户总量',
    accent: 'blue',
    to: '/users',
    icon: 'users',
  },
  {
    key: 'ic',
    label: '邀请码',
    value: kpi.value.inviteCodeCount,
    desc: '邀请码总数',
    accent: 'purple',
    to: '/invite-codes',
    icon: 'ticket',
  },
  {
    key: 'group',
    label: '群组数',
    value: kpi.value.groupCount,
    desc: '活跃群组总量',
    accent: 'green',
    to: '/groups',
    icon: 'group',
  },
  {
    key: 'assistant',
    label: '助理数',
    value: kpi.value.assistantCount,
    desc: '助理账号总量',
    accent: 'orange',
    to: '/assistants',
    icon: 'shield',
  },
  {
    key: 'report',
    label: '举报总数',
    value: kpi.value.reportCount,
    desc: '累计收到的举报',
    accent: 'slate',
    to: '/reports',
    icon: 'flag',
  },
  {
    key: 'pending',
    label: '待处理举报',
    value: kpi.value.pendingReportCount,
    desc: '等待处理的举报',
    accent: 'red',
    to: '/reports',
    icon: 'alert',
  },
])

const assistantCards = computed(() => [
  {
    key: 'user',
    label: '我的用户',
    value: kpi.value.userCount,
    desc: '通过我注册的用户',
    accent: 'blue',
    to: '/users',
    icon: 'users',
  },
  {
    key: 'group',
    label: '我的群组',
    value: kpi.value.groupCount,
    desc: '我名下用户的群组',
    accent: 'green',
    to: '/groups',
    icon: 'group',
  },
  {
    key: 'report',
    label: '收到举报',
    value: kpi.value.reportCount,
    desc: '我名下用户的举报',
    accent: 'slate',
    to: '/reports',
    icon: 'flag',
  },
  {
    key: 'pending',
    label: '待处理举报',
    value: kpi.value.pendingReportCount,
    desc: '等待处理的举报',
    accent: 'red',
    to: '/reports',
    icon: 'alert',
  },
])

const cards = computed(() => authStore.isAdmin ? adminCards.value : assistantCards.value)

const accentStyle: Record<string, { icon: string; bar: string; bg: string }> = {
  blue:   { icon: '#3b82f6', bar: '#3b82f6', bg: '#eff6ff' },
  purple: { icon: '#a855f7', bar: '#a855f7', bg: '#faf5ff' },
  green:  { icon: '#22c55e', bar: '#22c55e', bg: '#f0fdf4' },
  orange: { icon: '#f97316', bar: '#f97316', bg: '#fff7ed' },
  slate:  { icon: '#64748b', bar: '#94a3b8', bg: '#f8fafc' },
  red:    { icon: '#ef4444', bar: '#ef4444', bg: '#fef2f2' },
}

type QuickAction = { label: string; desc: string; to: string; icon: string; accent: string }
const adminQuickActions = computed<QuickAction[]>(() => {
  const actions: QuickAction[] = [
    { label: '批量创建用户', desc: '一次性导入多个账号', to: '/users',        icon: 'user-plus', accent: 'blue' },
    { label: '生成邀请码',   desc: '创建用户注册入口',   to: '/invite-codes', icon: 'ticket',    accent: 'purple' },
    { label: '查看举报',     desc: '处理待审核的举报',   to: '/reports',      icon: 'flag',      accent: 'red' },
    { label: '助理管理',     desc: '管理助理账号权限',   to: '/assistants',   icon: 'shield',    accent: 'orange' },
  ]
  if (authStore.isSuperAdmin) {
    actions.unshift({ label: '管理员管理', desc: '维护管理员账号与权限', to: '/admins', icon: 'shield', accent: 'slate' })
  }
  return actions
})
const assistantQuickActions: QuickAction[] = [
  { label: '批量创建用户', desc: '一次性导入多个账号', to: '/users',            icon: 'user-plus', accent: 'blue'   },
  { label: '查看举报',     desc: '处理待审核的举报',   to: '/reports',          icon: 'flag',      accent: 'red'    },
  { label: 'C2C 消息',     desc: '按归属用户发送和查询', to: '/im/c2c',        icon: 'message',   accent: 'orange' },
  { label: '查看会话',     desc: '按用户或群组查历史', to: '/im/conversations', icon: 'message',   accent: 'orange' },
]
const quickActions = computed(() => authStore.isAdmin ? adminQuickActions.value : assistantQuickActions)

const qaAccent: Record<string, { color: string; bg: string }> = {
  blue:   { color: '#3b82f6', bg: '#eff6ff' },
  purple: { color: '#a855f7', bg: '#faf5ff' },
  red:    { color: '#ef4444', bg: '#fef2f2' },
  orange: { color: '#f97316', bg: '#fff7ed' },
}
</script>

<template>
  <div class="page">

    <!-- Welcome Row -->
    <div class="card">
      <div class="card-header">
        <div class="welcome-row">
          <h1 class="welcome-title">
            你好，<span class="welcome-name">{{ authStore.user?.username }}</span>
          </h1>
          <div class="welcome-badge" :class="authStore.isAdmin ? 'badge-admin' : 'badge-assistant'">
            <span class="badge-dot"></span>
            {{ roleLabel }}
          </div>
        </div>
        <p class="welcome-sub">以下是你的系统概况</p>
      </div>
    </div>

    <!-- KPI Grid -->
    <div class="card">
      <div class="card-header">
        <div>
          <p class="page-title">数据概览</p>
          <p class="page-sub">{{ dashboardScopeText }}</p>
        </div>
      </div>
      <div class="kpi-grid-wrap">
        <div class="kpi-grid" :class="authStore.isAdmin ? 'kpi-grid-6' : 'kpi-grid-4'">
          <div
            v-for="card in cards"
            :key="card.key"
            class="kpi-card"
            :class="{ 'kpi-card-alert': card.accent === 'red' && !loading && card.value > 0 }"
            style="cursor: pointer"
            @click="router.push(card.to)"
          >
            <div class="kpi-top">
              <div class="kpi-icon-wrap" :style="{ background: accentStyle[card.accent].bg, color: accentStyle[card.accent].icon }">
                <svg v-if="card.icon === 'users'" width="18" height="18" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                    d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857
                       M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857
                       m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0z"/>
                </svg>
                <svg v-if="card.icon === 'ticket'" width="18" height="18" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                    d="M15 5v2m0 4v2m0 4v2M5 5a2 2 0 00-2 2v3a2 2 0 110 4v3a2 2 0 002 2h14a2 2 0 002-2v-3a2 2 0 110-4V7a2 2 0 00-2-2H5z"/>
                </svg>
                <svg v-if="card.icon === 'group'" width="18" height="18" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                    d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857
                       M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857
                       m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0z
                       m6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z"/>
                </svg>
                <svg v-if="card.icon === 'shield'" width="18" height="18" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                    d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944
                       a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9
                       c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622
                       0-1.042-.133-2.052-.382-3.016z"/>
                </svg>
                <svg v-if="card.icon === 'flag'" width="18" height="18" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                    d="M3 21v-4m0 0V5a2 2 0 012-2h6.5l1 1H21l-3 6 3 6h-8.5l-1-1H5a2 2 0 00-2 2zm9-13.5V9"/>
                </svg>
                <svg v-if="card.icon === 'alert'" width="18" height="18" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                    d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4
                       c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"/>
                </svg>
              </div>
              <span
                v-if="card.accent === 'red' && !loading && card.value > 0"
                class="kpi-alert-badge"
              >需处理</span>
            </div>

            <div v-if="loading" class="kpi-skeleton"></div>
            <p v-else class="kpi-value font-code">{{ card.value.toLocaleString() }}</p>
            <p class="kpi-label">{{ card.label }}</p>
            <p class="kpi-desc">{{ card.desc }}</p>
            <div class="kpi-bar">
              <div
                class="kpi-bar-fill"
                :style="{
                  width: loading ? '0%' : `${Math.min(100, Math.max(4, card.value / Math.max(...cards.map(c => c.value), 1) * 80))}%`,
                  background: accentStyle[card.accent].bar
                }"
              ></div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 助理邀请码详情 -->
    <div v-if="!authStore.isAdmin" class="card">
      <div class="card-header">
        <div>
          <p class="page-title">我的邀请码</p>
          <p class="page-sub">由管理员分配，用于邀请用户注册</p>
        </div>
      </div>
      <div class="ic-wrap">
        <div v-if="!assistantMe" class="ic-loading">加载中...</div>
        <div v-else-if="!assistantMe.inviteCode" class="ic-empty">
          <svg width="32" height="32" fill="none" stroke="#d1d5db" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5"
              d="M15 5v2m0 4v2m0 4v2M5 5a2 2 0 00-2 2v3a2 2 0 110 4v3a2 2 0 002 2h14a2 2 0 002-2v-3a2 2 0 110-4V7a2 2 0 00-2-2H5z"/>
          </svg>
          <p>暂未分配邀请码</p>
          <p class="ic-empty-sub">请联系管理员为你生成邀请码</p>
        </div>
        <div v-else class="ic-card">
          <div class="ic-main">
            <div class="ic-icon-wrap">
              <svg width="20" height="20" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M15 5v2m0 4v2m0 4v2M5 5a2 2 0 00-2 2v3a2 2 0 110 4v3a2 2 0 002 2h14a2 2 0 002-2v-3a2 2 0 110-4V7a2 2 0 00-2-2H5z"/>
              </svg>
            </div>
            <div class="ic-info">
              <span class="ic-code font-code">{{ assistantMe.inviteCode }}</span>
              <span
                class="ic-status"
                :class="`ic-status-${assistantMe.inviteCodeStatus}`"
              >{{ assistantMe.inviteCodeStatus === 'active' ? '有效' : assistantMe.inviteCodeStatus === 'expired' ? '已过期' : '已用尽' }}</span>
            </div>
          </div>
          <div class="ic-stats">
            <div class="ic-stat">
              <span class="ic-stat-label">已使用</span>
              <span class="ic-stat-value font-code">{{ assistantMe.inviteCodeUsed }}</span>
            </div>
            <div class="ic-stat-div"></div>
            <div class="ic-stat">
              <span class="ic-stat-label">上限</span>
              <span class="ic-stat-value font-code">{{ assistantMe.inviteCodeMax <= 0 ? '不限' : assistantMe.inviteCodeMax }}</span>
            </div>
            <div class="ic-stat-div"></div>
            <div class="ic-stat">
              <span class="ic-stat-label">到期</span>
              <span class="ic-stat-value font-code">{{ assistantMe.inviteCodeExpiredAt ?? '永不' }}</span>
            </div>
          </div>
          <div class="ic-bar-wrap">
            <div class="ic-bar">
              <div
                class="ic-bar-fill"
                :style="{
                  width: assistantMe.inviteCodeMax > 0
                    ? `${Math.min(100, assistantMe.inviteCodeUsed / assistantMe.inviteCodeMax * 100)}%`
                    : '0%'
                }"
              ></div>
            </div>
            <span class="ic-bar-label">
              {{ assistantMe.inviteCodeMax > 0
                  ? `${Math.round(assistantMe.inviteCodeUsed / assistantMe.inviteCodeMax * 100)}% 已使用`
                  : '不限使用次数' }}
            </span>
          </div>
        </div>
      </div>
    </div>

    <!-- Quick Actions -->
    <div class="card">
      <div class="card-header">
        <div>
          <p class="page-title">快捷操作</p>
          <p class="page-sub">常用功能入口</p>
        </div>
      </div>
      <div class="qa-wrap">
        <div class="qa-grid" :class="authStore.isAdmin ? 'qa-grid-4' : 'qa-grid-2'">
          <button
            v-for="action in quickActions"
            :key="action.to"
            class="qa-card"
            :class="`qa-${action.accent}`"
            @click="router.push(action.to)"
          >
            <div class="qa-icon-wrap" :style="{ background: qaAccent[action.accent]?.bg, color: qaAccent[action.accent]?.color }">
              <svg v-if="action.icon === 'user-plus'" width="18" height="18" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M18 9v3m0 0v3m0-3h3m-3 0h-3m-2-5a4 4 0 11-8 0 4 4 0 018 0zM3 20a6 6 0 0112 0v1H3v-1z"/>
              </svg>
              <svg v-if="action.icon === 'ticket'" width="18" height="18" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M15 5v2m0 4v2m0 4v2M5 5a2 2 0 00-2 2v3a2 2 0 110 4v3a2 2 0 002 2h14a2 2 0 002-2v-3a2 2 0 110-4V7a2 2 0 00-2-2H5z"/>
              </svg>
              <svg v-if="action.icon === 'flag'" width="18" height="18" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M3 21v-4m0 0V5a2 2 0 012-2h6.5l1 1H21l-3 6 3 6h-8.5l-1-1H5a2 2 0 00-2 2zm9-13.5V9"/>
              </svg>
              <svg v-if="action.icon === 'shield'" width="18" height="18" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944
                     a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9
                     c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622
                     0-1.042-.133-2.052-.382-3.016z"/>
              </svg>
              <svg v-if="action.icon === 'message'" width="18" height="18" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M8 10h8m-8 4h5m-9 6l2.4-2.4A2 2 0 016.8 17H19a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2h1.8a2 2 0 011.4.6L10 21z"/>
              </svg>
            </div>
            <div class="qa-text">
              <p class="qa-label">{{ action.label }}</p>
              <p class="qa-desc">{{ action.desc }}</p>
            </div>
            <svg class="qa-arrow" width="16" height="16" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"/>
            </svg>
          </button>
        </div>
      </div>
    </div>

  </div>
</template>

<style scoped>
/* ── Welcome ── */
.welcome-row {
  display: flex;
  align-items: center;
  gap: 10px;
}
.welcome-title {
  font-size: 22px;
  font-weight: 600;
  color: #1e293b;
  line-height: 1.3;
}
.welcome-name { color: #1677ff; }
.welcome-sub {
  font-size: 13px;
  color: #94a3b8;
  margin-top: 3px;
}
.welcome-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  font-weight: 500;
  padding: 5px 12px;
  border-radius: 9999px;
  border: 1px solid transparent;
}
.badge-admin     { color: #1677ff; background: #eff6ff; border-color: #bfdbfe; }
.badge-assistant { color: #9333ea; background: #faf5ff; border-color: #e9d5ff; }
.badge-dot { width: 6px; height: 6px; border-radius: 50%; background: currentColor; }

/* ── KPI Grid ── */
.kpi-grid-wrap { padding: 20px; }
.kpi-grid {
  display: grid;
  gap: 14px;
}
.kpi-grid-6 { grid-template-columns: repeat(3, 1fr); }
.kpi-grid-4 { grid-template-columns: repeat(2, 1fr); }
@media (min-width: 1100px) {
  .kpi-grid-6 { grid-template-columns: repeat(6, 1fr); }
  .kpi-grid-4 { grid-template-columns: repeat(4, 1fr); }
}
@media (max-width: 640px) {
  .kpi-grid-6, .kpi-grid-4 { grid-template-columns: repeat(2, 1fr); }
}

.kpi-card {
  background: #f8fafc;
  border: 1px solid #f0f2f7;
  border-radius: 10px;
  padding: 16px;
  transition: box-shadow 0.2s, transform 0.15s, border-color 0.15s;
}
.kpi-card:hover {
  box-shadow: 0 4px 14px rgba(0,0,0,0.07);
  transform: translateY(-1px);
  background: #fff;
}
.kpi-card-alert {
  border-color: #fecaca;
  background: linear-gradient(135deg, #fff 60%, #fff5f5 100%);
}
.kpi-card-alert:hover { border-color: #f87171; }

.kpi-top {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  margin-bottom: 14px;
}
.kpi-icon-wrap {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 34px;
  height: 34px;
  border-radius: 8px;
}
.kpi-alert-badge {
  font-size: 10px;
  font-weight: 600;
  color: #ef4444;
  background: #fee2e2;
  border: 1px solid #fecaca;
  padding: 2px 7px;
  border-radius: 9999px;
  animation: pulse-badge 2s infinite;
}
@keyframes pulse-badge {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.6; }
}

.kpi-skeleton {
  width: 60px; height: 28px;
  border-radius: 6px;
  background: linear-gradient(90deg, #f1f5f9 25%, #e2e8f0 50%, #f1f5f9 75%);
  background-size: 200% 100%;
  animation: shimmer 1.2s infinite;
  margin-bottom: 4px;
}
@keyframes shimmer {
  0%   { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}

.kpi-value {
  font-size: 26px;
  font-weight: 700;
  color: #1e293b;
  line-height: 1.1;
  letter-spacing: -0.02em;
}
.kpi-label {
  font-size: 12px;
  font-weight: 600;
  color: #475569;
  margin-top: 4px;
}
.kpi-desc {
  font-size: 11px;
  color: #94a3b8;
  margin-top: 2px;
}

.kpi-bar {
  margin-top: 12px;
  height: 3px;
  background: #f1f5f9;
  border-radius: 9999px;
  overflow: hidden;
}
.kpi-bar-fill {
  height: 100%;
  border-radius: 9999px;
  transition: width 0.9s cubic-bezier(0.34, 1.56, 0.64, 1);
}

/* ── Quick Actions ── */
.qa-wrap { padding: 20px; }
.qa-grid { display: grid; gap: 12px; }
.qa-grid-4 { grid-template-columns: repeat(2, 1fr); }
.qa-grid-2 { grid-template-columns: repeat(1, 1fr); }
@media (min-width: 768px) {
  .qa-grid-4 { grid-template-columns: repeat(4, 1fr); }
  .qa-grid-2 { grid-template-columns: repeat(2, 1fr); }
}

/* ── Invite Code Card ── */
.ic-wrap { padding: 20px; }
.ic-loading { color: #94a3b8; font-size: 13px; }
.ic-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 32px 0;
  color: #9ca3af;
  font-size: 13px;
}
.ic-empty-sub { font-size: 12px; color: #cbd5e1; }

.ic-card {
  background: linear-gradient(135deg, #f5f0ff 0%, #ede9fe 100%);
  border: 1px solid #e9d5ff;
  border-radius: 12px;
  padding: 20px 24px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.ic-main { display: flex; align-items: center; gap: 14px; }
.ic-icon-wrap {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  border-radius: 10px;
  background: rgba(147,51,234,0.12);
  color: #7c3aed;
  flex-shrink: 0;
}
.ic-info { display: flex; align-items: center; gap: 10px; }
.ic-code {
  font-size: 20px;
  font-weight: 700;
  color: #4c1d95;
  letter-spacing: 0.08em;
}
.ic-status {
  font-size: 11px;
  font-weight: 600;
  padding: 2px 8px;
  border-radius: 9999px;
}
.ic-status-active  { color: #16a34a; background: #dcfce7; }
.ic-status-expired { color: #dc2626; background: #fee2e2; }
.ic-status-used    { color: #92400e; background: #fef3c7; }

.ic-stats { display: flex; align-items: center; gap: 0; }
.ic-stat { display: flex; flex-direction: column; align-items: center; gap: 2px; flex: 1; }
.ic-stat-label { font-size: 11px; color: #7c3aed; }
.ic-stat-value { font-size: 16px; font-weight: 700; color: #4c1d95; }
.ic-stat-div { width: 1px; height: 32px; background: #ddd6fe; }

.ic-bar-wrap { display: flex; align-items: center; gap: 10px; }
.ic-bar { flex: 1; height: 6px; background: #ddd6fe; border-radius: 9999px; overflow: hidden; }
.ic-bar-fill { height: 100%; background: #9333ea; border-radius: 9999px; transition: width 0.9s cubic-bezier(0.34, 1.56, 0.64, 1); }
.ic-bar-label { font-size: 11px; color: #7c3aed; white-space: nowrap; }

.qa-card {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 16px;
  background: #f8fafc;
  border: 1px solid #f0f2f7;
  border-radius: 10px;
  text-align: left;
  cursor: pointer;
  transition: box-shadow 0.15s, border-color 0.15s, transform 0.15s, background 0.15s;
}
.qa-card:hover {
  box-shadow: 0 4px 12px rgba(0,0,0,0.07);
  transform: translateY(-1px);
  background: #fff;
}
.qa-blue:hover   { border-color: #bfdbfe; }
.qa-purple:hover { border-color: #e9d5ff; }
.qa-red:hover    { border-color: #fecaca; }
.qa-orange:hover { border-color: #fed7aa; }

.qa-icon-wrap {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border-radius: 8px;
  flex-shrink: 0;
}
.qa-text { flex: 1; min-width: 0; }
.qa-label {
  font-size: 13px;
  font-weight: 500;
  color: #1e293b;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.qa-desc {
  font-size: 12px;
  color: #94a3b8;
  margin-top: 1px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.qa-arrow { color: #cbd5e1; flex-shrink: 0; }
.qa-card:hover .qa-arrow { color: #94a3b8; }
</style>
