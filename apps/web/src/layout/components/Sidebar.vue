<script setup lang="ts">
// 响应式侧栏：桌面固定，窄屏使用抽屉模式。
// 分组可折叠：默认全部收起，当前路由所在分组自动展开；图标收起模式下保持全部图标展示
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ChevronDown, ChevronsLeft, Moon, Sun } from 'lucide-vue-next'
import AppIcon from '@/components/AppIcon.vue'
import { useAppStore } from '@/stores/app'
import { usePermissionStore } from '@/stores/permission'
import type { MenuNode } from '@/api/auth'

const route = useRoute()
const router = useRouter()
const appStore = useAppStore()
const permissionStore = usePermissionStore()

const mobileOpen = computed(() => appStore.mobileSidebarOpen)

// 顶层目录 = 分组；顶层叶子菜单（如 首页）= 独立导航项
const groups = computed(() => {
  const out: Array<{ title: string; icon?: string; items: MenuNode[] }> = []
  const leafItems: MenuNode[] = []
  for (const top of permissionStore.menus) {
    if (top.hidden) continue
    const items = (top.children ?? []).filter((c) => !c.hidden)
    if (top.menuType === 'M' && items.length > 0) {
      out.push({ title: top.title, icon: top.icon, items })
    } else {
      leafItems.push(top)
    }
  }
  if (leafItems.length > 0) {
    out.unshift({ title: '', items: leafItems })
  }
  return out
})

function isActive(path: string) {
  return route.path === path || route.path.startsWith(`${path}/`)
}

function navigate(path: string) {
  appStore.setMobileSidebar(false)
  void router.push(path)
}

const collapsed = computed(() => appStore.sidebarCollapsed)

// ==================== 分组折叠（仅展开模式下生效） ====================
const openGroups = ref<Set<string>>(new Set())

function toggleGroup(title: string) {
  const next = new Set(openGroups.value)
  if (next.has(title)) {
    next.delete(title)
  } else {
    next.add(title)
  }
  openGroups.value = next
}

// 当前路由所在分组自动展开
watch(
  () => route.path,
  () => {
    for (const g of groups.value) {
      if (g.title && g.items.some((i) => isActive(i.path))) {
        if (!openGroups.value.has(g.title)) {
          const next = new Set(openGroups.value)
          next.add(g.title)
          openGroups.value = next
        }
        return
      }
    }
  },
  { immediate: true },
)
</script>

<template>
  <!-- 窄屏遮罩 -->
  <div
    v-if="mobileOpen"
    class="fixed inset-0 z-40 bg-slate-900/40 lg:hidden"
    aria-hidden="true"
    @click="appStore.setMobileSidebar(false)"
  />
  <aside
    class="fixed inset-y-0 left-0 z-50 flex w-[240px] -translate-x-full flex-col border-r border-[#dfe7ee] bg-white transition-[width,transform] duration-200 lg:translate-x-0 dark:border-slate-800 dark:bg-slate-900"
    :class="[mobileOpen ? 'translate-x-0' : '', collapsed ? 'lg:w-16' : '']"
  >
    <!-- Logo -->
    <div class="flex h-20 items-center gap-2 border-b border-slate-100 px-5 dark:border-slate-800">
      <div class="flex h-8 w-8 items-center justify-center rounded-lg bg-brand text-sm font-bold text-white">G</div>
      <div v-if="!collapsed" class="flex items-baseline gap-1.5">
        <span class="text-lg font-bold text-slate-900 dark:text-white">GoKeep</span>
        <span class="text-xs text-slate-400">v0.1.0</span>
      </div>
    </div>

    <!-- 导航 -->
    <nav class="sidebar-nav flex-1 overflow-y-auto px-[14px] py-[18px]">
      <div v-for="group in groups" :key="group.title || 'leaf'" class="mb-4">
        <!-- 无标题组（首页等独立导航项）：直接展示 -->
        <template v-if="!group.title">
          <button
            v-for="item in group.items"
            :key="item.id"
            class="mb-1.5 flex h-12 w-full items-center gap-3 rounded-[7px] px-[18px] text-[15px] font-semibold transition-colors"
            :class="
              isActive(item.path)
                ? 'bg-[#e9f8f5] text-brand-700 dark:bg-brand-500/15 dark:text-brand-300'
                : 'text-slate-700 hover:bg-slate-50 dark:text-slate-300 dark:hover:bg-slate-800'
            "
            :title="item.title"
            @click="navigate(item.path)"
          >
            <AppIcon :name="item.icon" :size="16" />
            <span v-if="!collapsed" class="truncate">{{ item.title }}</span>
          </button>
        </template>

        <!-- 图标收起模式：分组项全部平铺为图标 -->
        <template v-else-if="collapsed">
          <button
            v-for="item in group.items"
            :key="item.id"
            class="mb-1.5 flex h-12 w-full items-center justify-center rounded-[7px] transition-colors"
            :class="
              isActive(item.path)
                ? 'bg-[#e9f8f5] text-brand-700 dark:bg-brand-500/15 dark:text-brand-300'
                : 'text-slate-700 hover:bg-slate-50 dark:text-slate-300 dark:hover:bg-slate-800'
            "
            :title="item.title"
            @click="navigate(item.path)"
          >
            <AppIcon :name="item.icon" :size="16" />
          </button>
        </template>

        <!-- 可折叠分组 -->
        <template v-else>
          <button
            class="flex h-10 w-full items-center gap-2 rounded-[7px] px-1 text-[15px] font-bold text-[#5b6b82] transition-colors hover:bg-slate-50 hover:text-slate-900 dark:text-slate-400 dark:hover:bg-slate-800 dark:hover:text-slate-200"
            :aria-expanded="openGroups.has(group.title)"
            @click="toggleGroup(group.title)"
          >
            <AppIcon v-if="group.icon" :name="group.icon" :size="14" />
            <span class="truncate">{{ group.title }}</span>
            <ChevronDown :size="12" class="ml-auto shrink-0 transition-transform duration-200" :class="openGroups.has(group.title) ? 'rotate-180' : ''" />
          </button>
          <div
            class="grid transition-[grid-template-rows] duration-200"
            :class="openGroups.has(group.title) ? 'grid-rows-[1fr]' : 'grid-rows-[0fr]'"
            :aria-hidden="!openGroups.has(group.title)"
          >
            <div class="overflow-hidden">
              <button
                v-for="item in group.items"
                :key="item.id"
                class="mb-1.5 flex h-12 w-full items-center gap-3 rounded-[7px] px-[18px] text-[15px] font-semibold transition-colors"
                :class="
                  isActive(item.path)
                    ? 'bg-[#e9f8f5] text-brand-700 dark:bg-brand-500/15 dark:text-brand-300'
                    : 'text-slate-700 hover:bg-slate-50 dark:text-slate-300 dark:hover:bg-slate-800'
                "
                :title="item.title"
                @click="navigate(item.path)"
              >
                <AppIcon :name="item.icon" :size="16" />
                <span class="truncate">{{ item.title }}</span>
              </button>
            </div>
          </div>
        </template>
      </div>
    </nav>

    <!-- 底部：主题切换 + 收起 -->
    <div class="border-t border-slate-100 p-3 dark:border-slate-800">
      <button
        class="flex h-10 w-full items-center gap-3 rounded-[7px] px-3 text-sm font-medium text-slate-600 hover:bg-slate-50 dark:text-slate-300 dark:hover:bg-slate-800"
        @click="appStore.toggleTheme()"
      >
        <Sun v-if="appStore.theme === 'dark'" :size="16" />
        <Moon v-else :size="16" />
        <span v-if="!collapsed">{{ appStore.theme === 'dark' ? '浅色模式' : '深色模式' }}</span>
      </button>
      <button
        class="mt-1 hidden h-10 w-full items-center gap-3 rounded-[7px] px-3 text-sm font-medium text-slate-600 hover:bg-slate-50 lg:flex dark:text-slate-300 dark:hover:bg-slate-800"
        @click="appStore.toggleSidebar()"
      >
        <ChevronsLeft :size="16" class="transition-transform" :class="collapsed ? 'rotate-180' : ''" />
        <span v-if="!collapsed">收起</span>
      </button>
    </div>
  </aside>
</template>
