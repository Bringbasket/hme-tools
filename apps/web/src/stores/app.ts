import { defineStore } from 'pinia'
import { ref } from 'vue'

/**
 * 应用级共享状态（docs/05 §4）：主题、侧栏折叠、网关连通状态
 */
export const useAppStore = defineStore('app', () => {
  const serverStatus = ref<'unknown' | 'ok' | 'fail'>('unknown')
  const theme = ref<'light' | 'dark'>('light')
  const sidebarCollapsed = ref(false)
  /** 窄屏抽屉式侧栏开关（docs/08 §3） */
  const mobileSidebarOpen = ref(false)

  function setServerStatus(status: 'unknown' | 'ok' | 'fail') {
    serverStatus.value = status
  }

  function toggleTheme() {
    theme.value = theme.value === 'light' ? 'dark' : 'light'
    document.documentElement.classList.toggle('dark', theme.value === 'dark')
    document.documentElement.dataset.theme = theme.value
  }

  function toggleSidebar() {
    sidebarCollapsed.value = !sidebarCollapsed.value
  }

  function setMobileSidebar(open: boolean) {
    mobileSidebarOpen.value = open
  }

  return {
    serverStatus,
    theme,
    sidebarCollapsed,
    mobileSidebarOpen,
    setServerStatus,
    toggleTheme,
    toggleSidebar,
    setMobileSidebar,
  }
})
