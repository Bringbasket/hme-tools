<script setup lang="ts">
import { nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { X } from 'lucide-vue-next'

defineOptions({ inheritAttrs: false })

const props = withDefaults(defineProps<{
  open: boolean
  title: string
  subtitle?: string
  busy?: boolean
  role?: 'dialog' | 'alertdialog'
  width?: 'normal' | 'wide'
}>(), { subtitle: '', busy: false, role: 'dialog', width: 'normal' })

const emit = defineEmits<{ close: [] }>()
const panel = ref<HTMLElement | null>(null)
let restoreFocus: HTMLElement | null = null

function focusable() {
  return Array.from(panel.value?.querySelectorAll<HTMLElement>(
    'button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), a[href], [tabindex]:not([tabindex="-1"])',
  ) || []).filter((element) => element.offsetParent !== null || element === document.activeElement)
}

function requestClose() {
  if (!props.busy) emit('close')
}

function onKeydown(event: KeyboardEvent) {
  if (!props.open) return
  if (event.key === 'Escape') {
    event.preventDefault()
    requestClose()
    return
  }
  if (event.key !== 'Tab') return
  const items = focusable()
  if (!items.length) {
    event.preventDefault()
    panel.value?.focus()
    return
  }
  const first = items[0]
  const last = items[items.length - 1]
  if (event.shiftKey && document.activeElement === first) {
    event.preventDefault()
    last.focus()
  } else if (!event.shiftKey && document.activeElement === last) {
    event.preventDefault()
    first.focus()
  }
}

watch(() => props.open, async (open) => {
  if (open) {
    restoreFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null
    document.addEventListener('keydown', onKeydown)
    await nextTick()
    const preferred = panel.value?.querySelector<HTMLElement>('[autofocus]')
    ;(preferred || focusable()[0] || panel.value)?.focus()
  } else {
    document.removeEventListener('keydown', onKeydown)
    restoreFocus?.focus()
    restoreFocus = null
  }
})

onBeforeUnmount(() => document.removeEventListener('keydown', onKeydown))
</script>

<template>
  <Teleport to="body">
    <div v-if="open" class="mail-dialog-backdrop" @mousedown.self="requestClose">
      <section
        :id="`${$attrs.id || 'app'}-dialog`"
        ref="panel"
        class="mail-dialog"
        :class="{ wide: width === 'wide' }"
        :role="role"
        aria-modal="true"
        tabindex="-1"
        :aria-labelledby="`${$attrs.id || 'app'}-dialog-title`"
      >
        <header class="mail-dialog-heading">
          <div>
            <h2 :id="`${$attrs.id || 'app'}-dialog-title`">{{ title }}</h2>
            <p v-if="subtitle">{{ subtitle }}</p>
          </div>
          <button type="button" class="icon-button" title="关闭" aria-label="关闭" :disabled="busy" @click="requestClose"><X :size="18" /></button>
        </header>
        <div class="mail-dialog-body"><slot /></div>
        <footer class="dialog-actions"><slot name="actions" /></footer>
      </section>
    </div>
  </Teleport>
</template>

<style scoped>
.mail-dialog-backdrop { position: fixed; inset: 0; z-index: 100; display: grid; padding: 18px; background: rgba(2, 6, 23, .58); backdrop-filter: blur(3px); place-items: center; }
.mail-dialog { display: grid; width: min(480px, 100%); max-height: min(680px, calc(100vh - 40px)); grid-template-rows: auto minmax(0, 1fr) auto; padding: 0; overflow: hidden; background: var(--surface); border: 1px solid var(--border); border-radius: 8px; box-shadow: 0 20px 48px rgba(2, 6, 23, .22); }
.mail-dialog.wide { width: min(760px, calc(100vw - 28px)); }
.mail-dialog-heading { display: flex; align-items: flex-start; justify-content: space-between; gap: 16px; padding: 19px 20px 15px; border-bottom: 1px solid var(--border-soft); }
.mail-dialog-heading h2 { margin: 5px 0 0; color: var(--text); font-size: 20px; font-weight: 750; }
.mail-dialog-heading p { margin: 5px 0 0; color: var(--muted); font-size: 12px; line-height: 1.5; overflow-wrap: anywhere; }
.mail-dialog-body { min-height: 0; padding: 18px 20px; overflow: auto; }
.dialog-actions { min-height: 66px; padding: 14px 20px; border-top: 1px solid var(--border-soft); }
@media (max-width: 620px) { .mail-dialog { width: calc(100vw - 24px); max-height: calc(100vh - 24px); } .mail-dialog-heading, .mail-dialog-body { padding-right: 16px; padding-left: 16px; } .dialog-actions { padding: 12px 16px; } }
</style>
