<script setup lang="ts">
// 顶栏：移动菜单、页面标题和用户菜单。
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { LogOut, Menu } from 'lucide-vue-next'
import { DropdownMenuContent, DropdownMenuItem, DropdownMenuRoot, DropdownMenuTrigger } from 'radix-vue'
import { useAppStore } from '@/stores/app'
import { usePermissionStore } from '@/stores/permission'
import { useUserStore } from '@/stores/user'

const route = useRoute()
const router = useRouter()
const appStore = useAppStore()
const userStore = useUserStore()
const permissionStore = usePermissionStore()

const open = ref(false)
const pageTitle = computed(() => (route.meta.title as string) || '')
const headerTitle = computed(() => {
  for (const menu of permissionStore.menus) {
    const inGroup = menu.children?.some((item) => route.path === item.path || route.path.startsWith(`${item.path}/`))
    if (inGroup) return menu.title
    if (route.path === menu.path || route.path.startsWith(`${menu.path}/`)) return menu.title
  }
  return pageTitle.value || 'GoKeep'
})

async function handleLogout() {
  open.value = false
  await userStore.logout()
  permissionStore.reset()
  void router.push('/login')
}
</script>

<template>
  <header
    class="fixed inset-x-0 top-0 z-30 flex h-20 items-center justify-between border-b border-slate-200/50 bg-white/95 px-5 backdrop-blur transition-[left] duration-200 dark:border-slate-800 dark:bg-slate-900/95"
    :class="appStore.sidebarCollapsed ? 'lg:left-16' : 'lg:left-[240px]'"
  >
    <div class="flex items-center gap-3">
      <button
        class="flex h-9 w-9 items-center justify-center rounded-md text-slate-600 hover:bg-slate-100 lg:hidden dark:text-slate-300 dark:hover:bg-slate-800"
        aria-label="打开菜单"
        @click="appStore.setMobileSidebar(true)"
      >
        <Menu :size="18" />
      </button>
      <h1 class="text-lg font-semibold text-slate-900 dark:text-white">{{ headerTitle }}</h1>
    </div>

    <DropdownMenuRoot v-model:open="open">
      <DropdownMenuTrigger
        class="flex h-10 max-w-[190px] items-center gap-2 rounded-md border border-transparent bg-transparent px-2 text-sm font-medium text-slate-700 outline-none hover:border-slate-200 hover:bg-slate-50 dark:text-slate-200 dark:hover:border-slate-700 dark:hover:bg-slate-800"
      >
        <span class="flex h-6 w-6 items-center justify-center rounded-full bg-brand-100 text-xs font-bold text-brand-700">
          {{ userStore.nickname.slice(0, 1) || 'U' }}
        </span>
        <span class="max-w-24 truncate">{{ userStore.nickname }}</span>
      </DropdownMenuTrigger>
      <DropdownMenuContent
        align="end"
        :side-offset="8"
        class="z-50 min-w-40 rounded-lg border border-slate-200 bg-white p-1 shadow-lg dark:border-slate-700 dark:bg-slate-800"
      >
        <DropdownMenuItem
          class="flex cursor-pointer items-center gap-2 rounded-md px-3 py-2 text-sm text-red-600 outline-none hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-500/10"
          @select="handleLogout"
        >
          <LogOut :size="15" />
          退出登录
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenuRoot>
  </header>
</template>
