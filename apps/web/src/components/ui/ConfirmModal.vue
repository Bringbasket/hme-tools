<script setup lang="ts">
// 危险操作确认（docs/06 §5：执行前明确目标与后果）
import Modal from './Modal.vue'

const props = defineProps<{
  open: boolean
  title?: string
  message: string
  loading?: boolean
}>()

const emit = defineEmits<{ 'update:open': [value: boolean]; confirm: [] }>()

function close() {
  if (!props.loading) emit('update:open', false)
}
</script>

<template>
  <Modal :open="props.open" :title="props.title ?? '操作确认'" width-class="max-w-sm" @update:open="close">
    <p class="text-sm leading-6 text-slate-600 dark:text-slate-300">{{ props.message }}</p>
    <template #footer>
      <button class="btn-secondary h-9 px-4" :disabled="props.loading" @click="close">取消</button>
      <button class="btn-danger h-9 px-4" :disabled="props.loading" @click="emit('confirm')">
        {{ props.loading ? '处理中…' : '确定' }}
      </button>
    </template>
  </Modal>
</template>
