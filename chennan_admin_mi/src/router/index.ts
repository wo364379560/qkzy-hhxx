import { createRouter, createWebHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/LoginView.vue'),
    meta: { requiresAuth: false }
  },
  {
    path: '/',
    component: () => import('@/layouts/AdminLayout.vue'),
    meta: { requiresAuth: true },
    redirect: '/dashboard',
    children: [
      { path: 'dashboard',       name: 'Dashboard',    component: () => import('@/views/DashboardView.vue') },
      { path: 'admins',          name: 'Admins',       component: () => import('@/views/AdminsView.vue'), meta: { superAdminOnly: true } },
      { path: 'settings',        name: 'Settings',     component: () => import('@/views/SettingsView.vue') },
      { path: 'operation',        name: 'OperationCenter', component: () => import('@/views/OperationCenterView.vue'), meta: { superAdminOnly: true } },
      { path: 'operation/cleanup', name: 'CleanupRiskControl', component: () => import('@/views/CleanupRiskControlView.vue'), meta: { superAdminOnly: true } },
      { path: 'operation/server-security', name: 'ServerSecurity', component: () => import('@/views/ServerSecurityView.vue'), meta: { superAdminOnly: true } },
      { path: 'ownership-operations', name: 'OwnershipOperations', component: () => import('@/views/OwnershipOperationsView.vue'), meta: { superAdminOnly: true } },
      { path: 'users',           name: 'Users',        component: () => import('@/views/UsersView.vue') },
      { path: 'invite-codes',    name: 'InviteCodes',  component: () => import('@/views/InviteCodesView.vue'), meta: { adminOnly: true } },
      { path: 'assistants/ip-whitelist', redirect: '/ip-whitelists' },
      { path: 'ip-whitelists',   name: 'IPWhitelistPolicies', component: () => import('@/views/IPWhitelistPoliciesView.vue'), meta: { adminOnly: true } },
      { path: 'audits',          redirect: '/audits/access' },
      { path: 'audits/access',   name: 'BehaviorAudits', component: () => import('@/views/BehaviorAuditView.vue') },
      { path: 'audits/logins',   name: 'LoginAudits', component: () => import('@/views/LoginAuditView.vue') },
      { path: 'audits/quiet-hours', name: 'QuietHoursHitLogs', component: () => import('@/views/QuietHoursHitLogsView.vue') },
      { path: 'audits/sms-sends', name: 'SMSSendLogs', component: () => import('@/views/SMSSendLogsView.vue') },
      { path: 'audits/user-operations', name: 'UserOperationLogs', component: () => import('@/views/UserOperationLogsView.vue'), meta: { superAdminOnly: true } },
      { path: 'reports',         name: 'Reports',      component: () => import('@/views/ReportsView.vue') },
      { path: 'community/posts', name: 'CommunityPosts', component: () => import('@/views/CommunityPostsView.vue') },
      { path: 'groups',          name: 'Groups',       component: () => import('@/views/GroupsView.vue') },
      { path: 'offline-activities', name: 'OfflineActivities', component: () => import('@/views/OfflineActivitiesView.vue'), meta: { adminOnly: true } },
      { path: 'assistants',      name: 'Assistants',   component: () => import('@/views/AssistantsView.vue'),  meta: { adminOnly: true } },
      { path: 'im/c2c',          name: 'C2C',          component: () => import('@/views/im/C2CView.vue') },
      { path: 'im/keywords',     name: 'ChatKeywords', component: () => import('@/views/im/ChatKeywordsView.vue'), meta: { superAdminOnly: true } },
      { path: 'im/conversations', name: 'Conversations', component: () => import('@/views/im/ConversationView.vue') },

      { path: 'im/forward-jobs', name: 'ForwardJobs', component: () => import('@/views/im/ForwardJobsView.vue') },

      // 旧路由重定向，兼容已有书签
      { path: 'my/users',        redirect: '/users' },
      { path: 'my/invite-codes', redirect: '/dashboard' },
      { path: 'my/reports',      redirect: '/reports' },
    ]
  },
  { path: '/:pathMatch(.*)*', redirect: '/dashboard' }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

router.beforeEach((to, _from, next) => {
  const authStore = useAuthStore()
  const requiresAuth = to.matched.some(r => r.meta.requiresAuth !== false)

  if (requiresAuth && !authStore.isLoggedIn) return next('/login')
  if (to.meta.superAdminOnly && !authStore.isSuperAdmin) return next('/dashboard')
  if (to.meta.adminOnly && !authStore.isAdmin) return next('/dashboard')
  if (to.path === '/login' && authStore.isLoggedIn) return next('/dashboard')

  next()
})

export default router
