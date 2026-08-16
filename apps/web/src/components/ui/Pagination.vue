<script setup lang="ts">
// 分页（docs/06 §6：默认 10，选项 10/20/50/100，显示范围与总数）
import { computed } from 'vue'
import { ChevronLeft, ChevronRight } from 'lucide-vue-next'

const props = withDefaults(
  defineProps<{
    page: number
    pageSize: number
    total: number
  }>(),
  { page: 1, pageSize: 10, total: 0 },
)

const emit = defineEmits<{ 'update:page': [value: number]; 'update:pageSize': [value: number] }>()

const pageSizes = [10, 20, 50, 100]
const totalPages = computed(() => Math.max(1, Math.ceil(props.total / props.pageSize)))
const rangeStart = computed(() => (props.total === 0 ? 0 : (props.page - 1) * props.pageSize + 1))
const rangeEnd = computed(() => Math.min(props.page * props.pageSize, props.total))

function go(page: number) {
  if (page < 1 || page > totalPages.value) return
  emit('update:page', page)
}
</script>

<template>
  <div class="mt-4 flex flex-wrap items-center justify-between gap-3 text-sm text-slate-500">
    <p class="text-xs">
      共 {{ total }} 条 · 第 {{ rangeStart }}-{{ rangeEnd }} 条 · 第 {{ page }} / 共 {{ totalPages }} 页
    </p>
    <div class="flex items-center gap-2">
      <select
        class="h-8 rounded-md border border-slate-200 bg-white px-2 text-xs outline-none dark:border-slate-700 dark:bg-slate-800"
        :value="pageSize"
        @change="emit('update:pageSize', Number(($event.target as HTMLSelectElement).value)); emit('update:page', 1)"
      >
        <option v-for="s in pageSizes" :key="s" :value="s">{{ s }} 条/页</option>
      </select>
      <button
        class="flex h-8 w-8 items-center justify-center rounded-md border border-slate-200 bg-white disabled:opacity-40 dark:border-slate-700 dark:bg-slate-800"
        :disabled="page <= 1"
        aria-label="上一页"
        @click="go(page - 1)"
      >
        <ChevronLeft :size="14" />
      </button>
      <button
        class="flex h-8 w-8 items-center justify-center rounded-md border border-slate-200 bg-white disabled:opacity-40 dark:border-slate-700 dark:bg-slate-800"
        :disabled="page >= totalPages"
        aria-label="下一页"
        @click="go(page + 1)"
      >
        <ChevronRight :size="14" />
      </button>
    </div>
  </div>
</template>
