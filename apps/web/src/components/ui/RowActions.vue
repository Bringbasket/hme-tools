<script setup lang="ts">
// 列表行操作按钮组：固定三按钮位，长动作收进「更多」下拉。
// 下拉菜单 Teleport 到 body 并 fixed 定位，避免被表格 overflow 容器裁剪
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { ChevronDown } from 'lucide-vue-next'

export interface RowAction {
  label: string
  danger?: boolean
  warning?: boolean
  disabled?: boolean
  onClick: () => void
}

const props = defineProps<{ actions: RowAction[] }>()

const moreOpen = ref(false)
const rootRef = ref<HTMLElement | null>(null)
const menuStyle = ref<{ top: string; left: string } | null>(null)

// 展示规则：
// - 前 2 个动作固定展示；
// - 第 3 个动作：两字标签继续展示（最多 3 个按钮位）；
// - 其余（含长文案）一律进「更多」下拉。
const visible = computed(() => props.actions.slice(0, 2))
const third = computed<RowAction | null>(() => {
  const a = props.actions[2]
  if (a && a.label.length <= 2) return a
  return null
})
const overflow = computed(() => {
  if (third.value) return props.actions.slice(3)
  return props.actions.slice(2)
})
// 只有 ≤2 个动作时也渲染「更多」占位，避免不同数据行宽度跳动。
const showMoreButton = computed(() => overflow.value.length > 0 || props.actions.length <= 2)

const MENU_WIDTH = 128 // w-32

function close() {
  moreOpen.value = false
  menuStyle.value = null
}

function toggleMore(e: MouseEvent) {
  if (moreOpen.value) {
    close()
    return
  }
  const rect = (e.currentTarget as HTMLElement).getBoundingClientRect()
  menuStyle.value = {
    top: `${Math.min(rect.bottom + 4, window.innerHeight - 140)}px`,
    left: `${Math.max(8, rect.right - MENU_WIDTH)}px`,
  }
  moreOpen.value = true
}

function onDocClick(e: MouseEvent) {
  if (rootRef.value && !rootRef.value.contains(e.target as Node)) close()
}

function onScroll() {
  // 滚动后按钮位置失效，直接收起
  if (moreOpen.value) close()
}

function run(action: RowAction) {
  close()
  if (!action.disabled) action.onClick()
}

onMounted(() => {
  document.addEventListener('click', onDocClick)
  document.addEventListener('scroll', onScroll, true)
  window.addEventListener('resize', close)
})
onBeforeUnmount(() => {
  document.removeEventListener('click', onDocClick)
  document.removeEventListener('scroll', onScroll, true)
  window.removeEventListener('resize', close)
})
</script>

<template>
  <div ref="rootRef" class="relative flex items-center gap-2">
    <!-- 前两个固定动作 -->
    <button
      v-for="(a, i) in visible"
      :key="`v-${i}-${a.label}`"
      type="button"
      class="btn-secondary btn-sm w-14 shrink-0 justify-center px-0"
      :class="a.danger ? '!border-red-200 !text-red-600' : a.warning ? '!border-amber-200 !text-amber-600' : ''"
      :disabled="a.disabled"
      @click="a.onClick"
    >
      {{ a.label }}
    </button>

    <!-- 第三个动作（两字） -->
    <button
      v-if="third"
      type="button"
      class="btn-secondary btn-sm w-14 shrink-0 justify-center px-0"
      :class="third.danger ? '!border-red-200 !text-red-600' : third.warning ? '!border-amber-200 !text-amber-600' : ''"
      :disabled="third.disabled"
      @click="third.onClick"
    >
      {{ third.label }}
    </button>

    <!-- 更多按钮 -->
    <button
      v-if="showMoreButton"
      type="button"
      class="btn-secondary btn-sm w-14 shrink-0 justify-center px-0"
      :class="moreOpen ? 'bg-brand-50' : ''"
      aria-haspopup="menu"
      :aria-expanded="moreOpen"
      @click="toggleMore"
    >
      更多
      <ChevronDown :size="12" class="ml-0.5" :class="moreOpen ? 'rotate-180' : ''" />
    </button>

    <!-- 更多下拉（Teleport 到 body，fixed 定位，不被表格容器裁剪） -->
    <Teleport to="body">
      <Transition name="fade">
        <div
          v-if="moreOpen && menuStyle"
          class="fixed z-[70] w-32 overflow-hidden rounded-lg border border-[#dbe6ee] bg-white py-1 shadow-[0_8px_24px_rgba(15,23,42,0.12)] dark:border-slate-700 dark:bg-slate-800"
          :style="{ top: menuStyle.top, left: menuStyle.left }"
          role="menu"
        >
          <template v-if="overflow.length > 0">
            <button
              v-for="a in overflow"
              :key="a.label"
              type="button"
              role="menuitem"
              class="block w-full px-3 py-2 text-left text-sm transition-colors hover:bg-slate-50 dark:hover:bg-slate-700"
              :class="a.danger ? 'text-red-600' : a.warning ? 'text-amber-600' : 'text-slate-700 dark:text-slate-200'"
              :disabled="a.disabled"
              @click="run(a)"
            >
              {{ a.label }}
            </button>
          </template>
          <p v-else class="px-3 py-2 text-sm text-slate-400">暂无更多操作</p>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.12s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
