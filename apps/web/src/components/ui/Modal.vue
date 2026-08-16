<script setup lang="ts">
// 弹窗原语（docs/08 §2 ResponsiveDialog：≥md 居中弹窗，<md 全屏）
import { nextTick, ref, watch } from 'vue'
import { DialogContent, DialogDescription, DialogOverlay, DialogPortal, DialogRoot, DialogTitle } from 'radix-vue'
import { X } from 'lucide-vue-next'

const props = withDefaults(
  defineProps<{
    open: boolean
    title: string
    widthClass?: string
    description?: string
  }>(),
  { widthClass: 'max-w-lg', description: '' },
)

const emit = defineEmits<{ 'update:open': [value: boolean] }>()
const bodyRef = ref<HTMLElement | null>(null)

watch(
  () => props.open,
  async (open) => {
    if (!open) return
    await nextTick()
    bodyRef.value?.scrollTo({ top: 0 })
  },
)

function close() {
  emit('update:open', false)
}
</script>

<template>
  <DialogRoot :open="props.open" @update:open="emit('update:open', $event)">
    <DialogPortal>
      <DialogOverlay class="fixed inset-0 z-40 bg-slate-900/40" />
      <DialogContent
        class="fixed inset-0 z-50 flex flex-col overflow-hidden bg-white outline-none max-md:rounded-none md:inset-auto md:left-1/2 md:top-1/2 md:h-auto md:max-h-[85vh] md:w-full md:-translate-x-1/2 md:-translate-y-1/2 md:rounded-lg md:shadow-floating dark:bg-slate-900"
        :class="props.widthClass"
      >
        <div class="flex items-center justify-between border-b border-slate-100 px-6 py-4 dark:border-slate-800">
          <div>
            <DialogTitle class="text-base font-bold text-slate-900 dark:text-white">{{ props.title }}</DialogTitle>
            <DialogDescription :class="props.description ? 'mt-0.5 text-xs text-slate-400' : 'sr-only'">
              {{ props.description || `${props.title}对话框` }}
            </DialogDescription>
          </div>
          <button
            class="flex h-8 w-8 items-center justify-center rounded-md text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-800"
            aria-label="关闭"
            @click="close"
          >
            <X :size="16" />
          </button>
        </div>
        <div ref="bodyRef" class="flex-1 overflow-y-auto px-6 py-4">
          <slot />
        </div>
        <div v-if="$slots.footer" class="flex items-center justify-end gap-3 border-t border-slate-100 px-6 py-3 dark:border-slate-800">
          <slot name="footer" />
        </div>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>
