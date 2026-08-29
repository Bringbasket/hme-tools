<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { CheckCircle2, CircleAlert, Download, LoaderCircle, PackageCheck, RefreshCw } from 'lucide-vue-next'
import { checkSystemVersion, getSystemVersion, requestSystemUpdate, type SystemVersion } from '@/api/version'
import { APP_REVISION, APP_VERSION } from '@/version'
import { errorMessage } from '@/views/mail/compat'

const props = defineProps<{
  collapsed: boolean
  canManage: boolean
}>()

const root = ref<HTMLElement | null>(null)
const open = ref(false)
const loading = ref(true)
const checking = ref(false)
const updating = ref(false)
const error = ref('')
const version = ref<SystemVersion | null>(null)
let pollTimer: number | undefined

const checkActive = computed(() => Boolean(
  version.value
  && !version.value.canRequestUpdate
  && version.value.action === 'check'
  && ['check_queued', 'checking'].includes(version.value.state),
))
const updateActive = computed(() => Boolean(
  version.value
  && !version.value.canRequestUpdate
  && version.value.action === 'update'
  && ['update_queued', 'updating', 'restarting'].includes(version.value.state),
))
const busy = computed(() => loading.value || checking.value || updating.value || checkActive.value || updateActive.value)
const currentVersion = computed(() => version.value?.currentVersion || APP_VERSION)
const currentRevision = computed(() => version.value?.currentRevision || APP_REVISION)
const latestBuild = computed(() => version.value?.latestVersion || version.value?.latestRevision?.slice(0, 8) || '可用构建')
const summary = computed(() => {
  if (error.value) return { kind: 'error', title: '操作失败', detail: error.value }
  if (loading.value && !version.value) return { kind: 'busy', title: '正在读取版本信息', detail: '正在连接版本服务' }
  if (checking.value || checkActive.value) return { kind: 'busy', title: '正在检查更新', detail: '正在查询最新可用构建' }
  if (updating.value || updateActive.value) {
    return {
      kind: 'busy',
      title: '更新进行中',
      detail: version.value?.state === 'update_queued' ? '更新请求已提交，等待宿主机执行' : '正在拉取并部署新版本',
    }
  }
  if (version.value?.state === 'error') return { kind: 'error', title: '版本任务失败', detail: version.value.error || '请重新检查更新' }
  if (version.value?.updateAvailable) return { kind: 'available', title: '发现新版本', detail: `最新版本 v${latestBuild.value}` }
  if (version.value?.state === 'success') return { kind: 'success', title: '更新完成', detail: '服务已经切换到最新构建' }
  if (version.value?.state === 'up_to_date') return { kind: 'success', title: '已是最新版本', detail: '当前没有可用更新' }
  return { kind: 'neutral', title: '尚未检查更新', detail: props.canManage ? '点击检查图标查询最新版本' : '当前账号没有检查更新权限' }
})

function isActiveTask() {
  return checkActive.value || updateActive.value
}

function stopPolling() {
  if (pollTimer !== undefined) {
    window.clearInterval(pollTimer)
    pollTimer = undefined
  }
}

function startPolling() {
  if (pollTimer !== undefined) return
  pollTimer = window.setInterval(() => { void loadVersion(true) }, 2000)
}

async function loadVersion(silent = false) {
  if (!silent) loading.value = true
  try {
    version.value = await getSystemVersion()
    error.value = ''
    if (isActiveTask()) startPolling()
    else stopPolling()
  } catch (reason) {
    // The service is expected to be briefly unavailable while a requested
    // container restart is in progress. Keep its last task state visible.
    if (!silent || !isActiveTask()) error.value = errorMessage(reason)
  } finally {
    if (!silent) loading.value = false
  }
}

async function checkForUpdates() {
  if (busy.value || !props.canManage) return
  checking.value = true
  error.value = ''
  try {
    version.value = await checkSystemVersion()
    startPolling()
  } catch (reason) {
    error.value = errorMessage(reason)
  } finally {
    checking.value = false
  }
}

async function updateNow() {
  if (busy.value || !props.canManage || !version.value?.updateAvailable) return
  updating.value = true
  error.value = ''
  try {
    version.value = await requestSystemUpdate()
    startPolling()
  } catch (reason) {
    error.value = errorMessage(reason)
  } finally {
    updating.value = false
  }
}

function toggle() {
  open.value = !open.value
}

function onPointerDown(event: PointerEvent) {
  if (open.value && root.value && event.target instanceof Node && !root.value.contains(event.target)) open.value = false
}

function onKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') open.value = false
}

onMounted(() => {
  document.addEventListener('pointerdown', onPointerDown)
  document.addEventListener('keydown', onKeydown)
  void loadVersion()
})

onBeforeUnmount(() => {
  document.removeEventListener('pointerdown', onPointerDown)
  document.removeEventListener('keydown', onKeydown)
  stopPolling()
})
</script>

<template>
  <div ref="root" class="version-update">
    <button
      type="button"
      class="version-trigger"
      :class="{ available: version?.updateAvailable }"
      title="版本与更新"
      aria-haspopup="dialog"
      :aria-expanded="open"
      @click="toggle"
    >
      <LoaderCircle v-if="loading" :size="17" class="spin" />
      <PackageCheck v-else :size="17" />
      <span v-if="!collapsed" class="version-trigger-copy">
        <span>版本与更新</span>
      </span>
      <span v-if="version?.updateAvailable" class="update-indicator" aria-label="有可用更新" />
    </button>

    <section v-if="open" class="version-panel" role="dialog" aria-label="版本与更新">
      <header class="version-panel-header">
        <div>
          <span>当前版本</span>
          <strong>v{{ currentVersion }}</strong>
        </div>
        <button
          type="button"
          class="version-icon-button"
          title="检查更新"
          aria-label="检查更新"
          :disabled="busy || !canManage"
          @click="checkForUpdates"
        >
          <RefreshCw :size="16" :class="{ spin: checking || checkActive }" />
        </button>
      </header>

      <div class="version-summary" :class="summary.kind" role="status" aria-live="polite">
        <span class="version-summary-icon">
          <LoaderCircle v-if="summary.kind === 'busy'" :size="17" class="spin" />
          <Download v-else-if="summary.kind === 'available'" :size="17" />
          <CircleAlert v-else-if="summary.kind === 'error'" :size="17" />
          <CheckCircle2 v-else-if="summary.kind === 'success'" :size="17" />
          <PackageCheck v-else :size="17" />
        </span>
        <span class="version-summary-copy"><strong>{{ summary.title }}</strong><small>{{ summary.detail }}</small></span>
      </div>

      <dl class="version-details">
        <div><dt>应用版本</dt><dd>v{{ currentVersion }}</dd></div>
        <div v-if="currentRevision"><dt>构建标识</dt><dd :title="currentRevision">{{ currentRevision.slice(0, 12) }}</dd></div>
        <div v-if="version?.updateAvailable && version.latestVersion"><dt>最新版本</dt><dd>v{{ version.latestVersion }}</dd></div>
        <div v-if="version?.updateAvailable && version.latestRevision"><dt>最新构建</dt><dd :title="version.latestRevision">{{ version.latestRevision.slice(0, 12) }}</dd></div>
      </dl>

      <button
        v-if="version?.updateAvailable"
        type="button"
        class="version-update-button"
        :disabled="busy || !canManage || !version.canRequestUpdate"
        @click="updateNow"
      >
        <LoaderCircle v-if="updating || updateActive" :size="16" class="spin" />
        <Download v-else :size="16" />
        立即更新
      </button>
      <p v-else-if="!canManage" class="version-permission">当前账号仅可查看版本状态</p>
      <a v-if="version?.repositoryUrl" class="version-repository" :href="version.repositoryUrl" target="_blank" rel="noopener noreferrer">查看项目仓库</a>
    </section>
  </div>
</template>

<style scoped>
.version-update { position: relative; min-width: 0; }
.version-trigger { display: flex; width: 100%; min-width: 0; height: 44px; align-items: center; gap: 12px; padding: 0 13px; color: var(--text-secondary); background: transparent; border: 0; border-radius: 6px; font-size: 13px; text-align: left; }
.version-trigger:hover { color: var(--text); background: var(--surface-hover); }
.version-trigger.available { color: var(--warning); background: var(--warning-soft); }
.version-trigger > svg { flex: 0 0 17px; }
.version-trigger-copy { display: grid; min-width: 0; line-height: 1.15; }
.version-trigger-copy > span { overflow: hidden; color: var(--text-secondary); font-size: 13px; font-weight: 600; text-overflow: ellipsis; white-space: nowrap; }
.update-indicator { flex: 0 0 7px; width: 7px; height: 7px; margin-left: auto; background: var(--warning); border-radius: 50%; box-shadow: 0 0 0 3px color-mix(in srgb, var(--warning) 15%, transparent); }
.version-panel { position: absolute; z-index: 80; bottom: calc(100% + 8px); left: 0; width: min(300px, calc(100vw - 24px)); padding: 12px; color: var(--text); background: var(--surface); border: 1px solid var(--border); border-radius: 8px; box-shadow: 0 16px 38px rgba(15, 23, 42, 0.16); }
.version-panel-header { display: flex; min-height: 42px; align-items: flex-start; justify-content: space-between; gap: 12px; padding-bottom: 11px; border-bottom: 1px solid var(--border-soft); }
.version-panel-header > div { display: grid; gap: 3px; }
.version-panel-header span { color: var(--muted); font-size: 11px; }
.version-panel-header strong { color: var(--text); font-size: 17px; line-height: 1.15; }
.version-icon-button { display: grid; flex: 0 0 34px; width: 34px; height: 34px; padding: 0; color: var(--muted); background: transparent; border: 1px solid transparent; border-radius: 6px; place-items: center; }
.version-icon-button:hover:not(:disabled) { color: var(--text); background: var(--surface-hover); border-color: var(--border); }
.version-summary { display: grid; min-height: 54px; grid-template-columns: 30px minmax(0, 1fr); align-items: center; gap: 9px; margin: 12px 0; padding: 9px; color: #047857; background: #ecfdf5; border: 1px solid #d1fae5; border-radius: 7px; }
.version-summary.neutral { color: var(--muted); background: var(--surface-subtle); border-color: var(--border-soft); }
.version-summary.available { color: #b45309; background: #fffbeb; border-color: #fde68a; }
.version-summary.busy { color: #1d4ed8; background: #eff6ff; border-color: #dbeafe; }
.version-summary.error { color: var(--danger); background: var(--danger-soft); border-color: color-mix(in srgb, var(--danger) 20%, var(--border)); }
.version-summary-icon { display: grid; width: 30px; height: 30px; background: color-mix(in srgb, currentColor 12%, transparent); border-radius: 6px; place-items: center; }
.version-summary-copy { display: grid; min-width: 0; gap: 3px; }
.version-summary-copy strong { font-size: 12px; font-weight: 700; }
.version-summary-copy small { color: currentColor; font-size: 11px; line-height: 1.35; opacity: .78; overflow-wrap: anywhere; }
.version-details { display: grid; gap: 7px; margin: 0 0 12px; }
.version-details > div { display: flex; min-width: 0; align-items: center; justify-content: space-between; gap: 12px; color: var(--muted); font-size: 11px; }
.version-details dt, .version-details dd { min-width: 0; margin: 0; }
.version-details dd { max-width: 156px; overflow: hidden; color: var(--text); font-family: ui-monospace, SFMono-Regular, Consolas, monospace; font-size: 11px; font-weight: 600; text-overflow: ellipsis; white-space: nowrap; }
.version-update-button { display: inline-flex; width: 100%; height: 36px; align-items: center; justify-content: center; gap: 7px; color: #ffffff; background: #0d9488; border: 1px solid #0d9488; border-radius: 6px; font-size: 13px; font-weight: 650; }
.version-update-button:hover:not(:disabled) { background: #0f766e; border-color: #0f766e; }
.version-permission { margin: 0; padding: 10px; color: var(--muted); background: var(--surface-subtle); border: 1px solid var(--border-soft); border-radius: 6px; font-size: 11px; line-height: 1.4; }
.version-repository { display: block; margin-top: 10px; overflow: hidden; color: var(--muted); font-size: 11px; text-align: center; text-decoration: none; text-overflow: ellipsis; white-space: nowrap; }
.version-repository:hover { color: var(--primary-text); }
.spin { animation: version-spin 800ms linear infinite; }
.dark .version-summary { color: #6ee7b7; background: rgba(16, 185, 129, .12); border-color: rgba(16, 185, 129, .2); }
.dark .version-summary.available { color: #fbbf24; background: rgba(245, 158, 11, .12); border-color: rgba(245, 158, 11, .2); }
.dark .version-summary.busy { color: #93c5fd; background: rgba(37, 99, 235, .14); border-color: rgba(37, 99, 235, .25); }
@keyframes version-spin { to { transform: rotate(360deg); } }
@media (min-width: 1024px) { .version-update :deep(.sidebar-label) { display: none; } }
@media (max-width: 360px) { .version-panel { width: calc(100vw - 24px); } }
@media (prefers-reduced-motion: reduce) { .spin { animation: none; } }
</style>
