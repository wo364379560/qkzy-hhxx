import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { login as apiLogin, logout as apiLogout } from '@/api/auth'
import type { AdminUser } from '@/types'
import router from '@/router'

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string>(localStorage.getItem('admin_token') ?? '')
  const user = ref<AdminUser | null>(
    JSON.parse(localStorage.getItem('admin_user') ?? 'null')
  )

  const isLoggedIn = computed(() => !!token.value && !!user.value)
  const isSuperAdmin = computed(() => user.value?.role === 'super_admin')
  const isManager = computed(() => user.value?.role === 'admin')
  const isAssistant = computed(() => user.value?.role === 'assistant')
  const isAdmin = computed(() => isSuperAdmin.value || isManager.value)
  const canManageAdmins = computed(() => isSuperAdmin.value)
  const canManageAssistants = computed(() => isAdmin.value)
  const canManageGlobalSettings = computed(() => isSuperAdmin.value)

  async function login(username: string, password: string, turnstileToken?: string) {
    const res = await apiLogin({ username, password, turnstile_token: turnstileToken })
    const data = res.data.data

    // 检查是否需要 2FA
    if (data.require_totp_setup || data.require_totp_verify) {
      return data // 返回 2FA 相关数据
    }

    // 正常登录
    token.value = data.token
    user.value = data.user
    localStorage.setItem('admin_token', data.token)
    localStorage.setItem('admin_user', JSON.stringify(data.user))
    return data
  }

  function setAuth(newToken: string, newUser: AdminUser) {
    token.value = newToken
    user.value = newUser
    localStorage.setItem('admin_token', newToken)
    localStorage.setItem('admin_user', JSON.stringify(newUser))
  }

  function updateUser(patch: Partial<AdminUser>) {
    if (!user.value) return
    user.value = { ...user.value, ...patch }
    localStorage.setItem('admin_user', JSON.stringify(user.value))
  }

  async function logout() {
    try {
      await apiLogout()
    } catch {
      // ignore logout errors
    } finally {
      token.value = ''
      user.value = null
      localStorage.removeItem('admin_token')
      localStorage.removeItem('admin_user')
      router.push('/login')
    }
  }

  return {
    token,
    user,
    isLoggedIn,
    isSuperAdmin,
    isManager,
    isAssistant,
    isAdmin,
    canManageAdmins,
    canManageAssistants,
    canManageGlobalSettings,
    login,
    logout,
    setAuth,
    updateUser,
  }
})
