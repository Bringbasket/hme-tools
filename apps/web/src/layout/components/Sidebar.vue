<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ChevronDown, ChevronsLeft, Moon, Sun } from 'lucide-vue-next'
import AppIcon from '@/components/AppIcon.vue'
import { useAppStore } from '@/stores/app'
import { usePermissionStore } from '@/stores/permission'
import { useUserStore } from '@/stores/user'
import type { MenuNode } from '@/api/auth'
import { APP_VERSION } from '@/version'
import VersionUpdate from './VersionUpdate.vue'

const route = useRoute()
const router = useRouter()
const appStore = useAppStore()
const permissionStore = usePermissionStore()
const userStore = useUserStore()

const mobileOpen = computed(() => appStore.mobileSidebarOpen)
const collapsed = computed(() => appStore.sidebarCollapsed)
const navElement = ref<HTMLElement | null>(null)
const openGroups = ref<Set<string>>(new Set())

const groups = computed(() => {
  const out: Array<{ title: string; items: MenuNode[]; collapsible: boolean }> = []
  const leafItems: MenuNode[] = []
  for (const top of permissionStore.menus) {
    if (top.hidden) continue
    const items = (top.children ?? []).filter((item) => !item.hidden)
    if (top.menuType === 'M' && items.length > 0) {
      out.push({ title: top.title, items, collapsible: true })
    } else {
      leafItems.push(top)
    }
  }
  if (leafItems.length > 0) out.unshift({ title: '工作台', items: leafItems, collapsible: false })
  return out
})

function isActive(path: string) {
  return route.path === path || route.path.startsWith(`${path}/`)
}

function navigate(path: string) {
  appStore.setMobileSidebar(false)
  void router.push(path)
}

function toggleGroup(title: string) {
  const next = new Set(openGroups.value)
  if (next.has(title)) next.delete(title)
  else next.add(title)
  openGroups.value = next
}

function openActiveGroup() {
  const activeGroup = groups.value.find((group) => group.collapsible && group.items.some((item) => isActive(item.path)))
  if (!activeGroup || openGroups.value.has(activeGroup.title)) return
  openGroups.value = new Set([...openGroups.value, activeGroup.title])
}

async function scrollActiveMenuIntoView() {
  await nextTick()
  navElement.value?.querySelector<HTMLElement>('.sidebar-item.active')?.scrollIntoView({ block: 'nearest' })
}

watch(
  [() => route.path, () => groups.value.length],
  async () => {
    openActiveGroup()
    await scrollActiveMenuIntoView()
  },
  { immediate: true },
)
onMounted(() => window.addEventListener('resize', scrollActiveMenuIntoView))
onBeforeUnmount(() => window.removeEventListener('resize', scrollActiveMenuIntoView))
</script>

<template>
  <div
    v-if="mobileOpen"
    class="fixed inset-0 z-40 bg-slate-950/35 lg:hidden"
    aria-hidden="true"
    @click="appStore.setMobileSidebar(false)"
  />
  <aside
    class="app-sidebar fixed inset-y-0 left-0 z-50 flex w-[240px] -translate-x-full flex-col border-r bg-white transition-[width,transform] duration-200 lg:translate-x-0 dark:bg-slate-900"
    :class="[mobileOpen ? 'translate-x-0' : '', collapsed ? 'is-collapsed lg:w-16' : '']"
  >
    <div class="sidebar-brand flex h-20 shrink-0 items-center gap-3 border-b px-5">
      <div class="grid h-8 w-8 shrink-0 place-items-center rounded-md bg-brand text-sm font-bold text-white">G</div>
      <div class="sidebar-copy min-w-0">
        <div class="flex min-w-0 items-center gap-2 leading-5">
          <span class="truncate text-[18px] font-bold text-slate-950 dark:text-white">GoKeep</span>
          <span class="shrink-0 font-mono text-[10px] font-semibold text-slate-400 dark:text-slate-500">v{{ APP_VERSION }}</span>
        </div>
        <div class="mt-1 text-[10px] font-semibold text-brand-700 dark:text-brand-300">管理工作台</div>
      </div>
    </div>

    <nav ref="navElement" class="sidebar-nav min-w-0 flex-1 overflow-y-auto overflow-x-hidden px-[14px] py-4" aria-label="主菜单">
      <section v-for="group in groups" :key="group.title" class="sidebar-group">
        <button
          v-if="group.collapsible"
          type="button"
          class="sidebar-group-title sidebar-group-trigger"
          :aria-expanded="openGroups.has(group.title)"
          @click="toggleGroup(group.title)"
        >
          <span class="sidebar-label truncate">{{ group.title }}</span>
          <ChevronDown :size="14" :class="openGroups.has(group.title) ? 'rotate-180' : ''" />
        </button>
        <p v-else class="sidebar-group-title"><span class="sidebar-label">{{ group.title }}</span></p>
        <div class="sidebar-group-items" :class="{ closed: group.collapsible && !openGroups.has(group.title) }">
          <button
            v-for="item in group.items"
            :key="item.id"
            type="button"
            class="sidebar-item"
            :class="{ active: isActive(item.path) }"
            :title="item.title"
            :aria-current="isActive(item.path) ? 'page' : undefined"
            @click="navigate(item.path)"
          >
            <AppIcon :name="item.icon" :size="18" />
            <span class="sidebar-label truncate">{{ item.title }}</span>
          </button>
        </div>
      </section>
    </nav>

    <div class="sidebar-footer shrink-0 border-t p-3">
      <VersionUpdate
        v-if="userStore.hasPerm('system:config:list')"
        :collapsed="collapsed"
        :can-manage="userStore.hasPerm('system:config:edit')"
      />
      <button type="button" class="sidebar-utility" :title="appStore.theme === 'dark' ? '切换到浅色模式' : '切换到深色模式'" @click="appStore.toggleTheme()">
        <Sun v-if="appStore.theme === 'dark'" :size="17" />
        <Moon v-else :size="17" />
        <span class="sidebar-label">{{ appStore.theme === 'dark' ? '浅色模式' : '深色模式' }}</span>
      </button>
      <button type="button" class="sidebar-utility mt-1 hidden lg:flex" :title="collapsed ? '展开侧栏' : '收起侧栏'" @click="appStore.toggleSidebar()">
        <ChevronsLeft :size="17" class="transition-transform" :class="collapsed ? 'rotate-180' : ''" />
        <span class="sidebar-label">收起</span>
      </button>
    </div>
  </aside>
</template>

<style scoped>
.app-sidebar,
.sidebar-brand,
.sidebar-footer {
  border-color: var(--border);
}

.sidebar-group + .sidebar-group {
  margin-top: 18px;
}

.sidebar-group-title {
  display: flex;
  width: 100%;
  height: 28px;
  align-items: center;
  justify-content: space-between;
  margin: 0;
  padding: 0 8px;
  color: var(--muted);
  background: transparent;
  border: 0;
  font-size: 12px;
  font-weight: 600;
  line-height: 28px;
  text-align: left;
}

.sidebar-group-trigger:hover {
  color: var(--text);
  background: var(--surface-hover);
}

.sidebar-group-trigger {
  height: 44px;
  padding: 0 14px;
  color: var(--text-secondary);
  border-radius: 6px;
  font-size: 15px;
  font-weight: 600;
  line-height: 44px;
}

.sidebar-group-trigger svg {
  transition: transform 140ms ease;
}

.sidebar-group-items.closed {
  display: none;
}

.sidebar-item,
.sidebar-utility {
  display: flex;
  width: 100%;
  min-width: 0;
  height: 44px;
  align-items: center;
  gap: 12px;
  padding: 0 14px;
  color: var(--text-secondary);
  background: transparent;
  border: 0;
  border-left: 3px solid transparent;
  border-radius: 6px;
  font-size: 15px;
  font-weight: 600;
  text-align: left;
  transition: color 140ms ease, background 140ms ease, border-color 140ms ease;
}

.sidebar-item:hover,
.sidebar-utility:hover {
  color: var(--text);
  background: var(--surface-hover);
}

.sidebar-item.active {
  color: var(--primary-text);
  background: var(--primary-soft);
  border-left-color: var(--primary);
}

.sidebar-item :deep(svg),
.sidebar-utility :deep(svg) {
  flex: 0 0 18px;
}

.sidebar-utility {
  padding-left: 13px;
  color: var(--text-secondary);
  font-weight: 500;
}

@media (min-width: 1024px) {
  .is-collapsed .sidebar-brand {
    justify-content: center;
    padding-right: 0;
    padding-left: 0;
  }

  .is-collapsed .sidebar-copy,
  .is-collapsed .sidebar-group-title,
  .is-collapsed .sidebar-label {
    display: none;
  }

  .is-collapsed .sidebar-nav {
    padding-right: 8px;
    padding-left: 8px;
  }

  .is-collapsed .sidebar-group + .sidebar-group {
    margin-top: 8px;
    padding-top: 8px;
    border-top: 1px solid var(--border-soft);
  }

  .is-collapsed .sidebar-group-items.closed {
    display: block;
  }

  .is-collapsed .sidebar-item,
  .is-collapsed .sidebar-utility {
    justify-content: center;
    gap: 0;
    padding: 0;
  }

  .is-collapsed .sidebar-footer {
    padding-right: 8px;
    padding-left: 8px;
  }
}
</style>
