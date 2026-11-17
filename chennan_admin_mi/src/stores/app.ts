import { defineStore } from 'pinia'
import { ref } from 'vue'

export interface VisitedTab {
  key: string
  title: string
  path: string
  closable: boolean
}

const DASHBOARD_TAB: VisitedTab = {
  key: 'Dashboard',
  title: '仪表盘',
  path: '/dashboard',
  closable: false,
}

const VISITED_TABS_STORAGE_KEY = 'admin_visited_tabs'

function restoreVisitedTabs(): VisitedTab[] {
  try {
    const stored = JSON.parse(localStorage.getItem(VISITED_TABS_STORAGE_KEY) || '[]')
    if (!Array.isArray(stored)) return [{ ...DASHBOARD_TAB }]

    const restored = stored.reduce<VisitedTab[]>((tabs, item) => {
      if (
        !item ||
        typeof item.key !== 'string' ||
        typeof item.title !== 'string' ||
        typeof item.path !== 'string' ||
        item.key === DASHBOARD_TAB.key ||
        !/^\/(?!\/)/.test(item.path) ||
        tabs.some(tab => tab.key === item.key)
      ) {
        return tabs
      }
      tabs.push({
        key: item.key,
        title: item.title,
        path: item.path,
        closable: true,
      })
      return tabs
    }, [])

    return [{ ...DASHBOARD_TAB }, ...restored]
  } catch {
    return [{ ...DASHBOARD_TAB }]
  }
}

export const useAppStore = defineStore('app', () => {
  const loading = ref(false)
  const sidebarCollapsed = ref(false)
  const visitedTabs = ref<VisitedTab[]>(restoreVisitedTabs())

  function persistVisitedTabs() {
    localStorage.setItem(VISITED_TABS_STORAGE_KEY, JSON.stringify(visitedTabs.value))
  }

  function setLoading(val: boolean) {
    loading.value = val
  }

  function toggleSidebar() {
    sidebarCollapsed.value = !sidebarCollapsed.value
  }

  function upsertVisitedTab(tab: VisitedTab) {
    const index = visitedTabs.value.findIndex(item => item.key === tab.key)
    if (index >= 0) {
      visitedTabs.value[index] = tab.key === DASHBOARD_TAB.key
        ? { ...DASHBOARD_TAB }
        : { ...tab }
      persistVisitedTabs()
      return
    }
    visitedTabs.value.push(tab.key === DASHBOARD_TAB.key ? { ...DASHBOARD_TAB } : { ...tab })
    persistVisitedTabs()
  }

  function closeVisitedTab(key: string) {
    const tab = visitedTabs.value.find(item => item.key === key)
    if (!tab?.closable) return
    visitedTabs.value = visitedTabs.value.filter(item => item.key !== key)
    persistVisitedTabs()
  }

  function resetVisitedTabs() {
    visitedTabs.value = [{ ...DASHBOARD_TAB }]
    persistVisitedTabs()
  }

  return {
    loading,
    sidebarCollapsed,
    visitedTabs,
    setLoading,
    toggleSidebar,
    upsertVisitedTab,
    closeVisitedTab,
    resetVisitedTabs,
  }
})
