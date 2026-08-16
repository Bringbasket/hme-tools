<script setup lang="ts">
// 首页工作台：欢迎信息、链路状态和快速导航。
import { onMounted, ref } from 'vue'
import { http, unwrap } from '@/api/http'
import { useAppStore } from '@/stores/app'
import { useUserStore } from '@/stores/user'

const appStore = useAppStore()
const userStore = useUserStore()

const pingResult = ref('')
const readyState = ref<{ postgres: boolean; redis: boolean } | null>(null)
const loading = ref(false)

async function check() {
  loading.value = true
  try {
    const t0 = performance.now()
    const data = await unwrap<{ message: string; time: string }>(http.get('/ping'))
    pingResult.value = `${data.message} · ${Math.round(performance.now() - t0)}ms`
    appStore.setServerStatus('ok')
  } catch {
    pingResult.value = '网关不可达'
    appStore.setServerStatus('fail')
  }
  try {
    // /readyz 在网关根路径（不在 /api/v1 下），显式覆盖 baseURL
    readyState.value = await unwrap<{ postgres: boolean; redis: boolean }>(http.get('/readyz', { baseURL: '/' }))
  } catch {
    readyState.value = null
  }
  loading.value = false
}

onMounted(check)
</script>

<template>
  <div class="grid gap-6">
    <!-- 欢迎 -->
    <section class="rounded-lg border border-[#dbe6ee] bg-white p-4 shadow-[0_1px_2px_rgba(15,23,42,0.035)] dark:border-slate-700 dark:bg-slate-900">
      <div class="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h2 class="text-base font-bold text-slate-900 dark:text-white">
            欢迎回来，{{ userStore.nickname }}
          </h2>
          <p class="mt-1 text-sm text-slate-500">
            角色：{{ userStore.roles.join(' / ') || '未分配' }}
          </p>
        </div>
        <button class="btn-primary h-9 px-4" :disabled="loading" @click="check">
          {{ loading ? '检测中…' : '重新检测链路' }}
        </button>
      </div>
    </section>

    <!-- 链路状态 -->
    <section class="rounded-lg border border-[#dbe6ee] bg-white p-4 shadow-[0_1px_2px_rgba(15,23,42,0.035)] dark:border-slate-700 dark:bg-slate-900">
      <h2 class="text-sm font-bold text-slate-900 dark:text-white">平台状态</h2>
      <div class="mt-3 grid gap-3 sm:grid-cols-3">
        <div class="flex items-center gap-3 rounded-lg bg-slate-50/70 p-3 dark:bg-slate-800/60">
          <span class="h-2.5 w-2.5 rounded-full" :class="appStore.serverStatus === 'ok' ? 'bg-emerald-500' : 'bg-red-500'" />
          <div>
            <p class="text-sm font-semibold text-slate-800 dark:text-slate-200">API 网关</p>
            <p class="text-xs text-slate-400">{{ pingResult || '—' }}</p>
          </div>
        </div>
        <div class="flex items-center gap-3 rounded-lg bg-slate-50/70 p-3 dark:bg-slate-800/60">
          <span class="h-2.5 w-2.5 rounded-full" :class="readyState?.postgres ? 'bg-emerald-500' : 'bg-red-500'" />
          <div>
            <p class="text-sm font-semibold text-slate-800 dark:text-slate-200">PostgreSQL</p>
            <p class="text-xs text-slate-400">{{ readyState?.postgres ? '已连接' : '未就绪' }}</p>
          </div>
        </div>
        <div class="flex items-center gap-3 rounded-lg bg-slate-50/70 p-3 dark:bg-slate-800/60">
          <span class="h-2.5 w-2.5 rounded-full" :class="readyState?.redis ? 'bg-emerald-500' : 'bg-red-500'" />
          <div>
            <p class="text-sm font-semibold text-slate-800 dark:text-slate-200">Redis</p>
            <p class="text-xs text-slate-400">{{ readyState?.redis ? '已连接' : '未就绪' }}</p>
          </div>
        </div>
      </div>
      <p class="mt-3 text-xs text-slate-400">
        后端未启动时显示未就绪——启动方式：<code class="rounded bg-slate-100 px-1 dark:bg-slate-800">powershell -ExecutionPolicy Bypass -File .\scripts\dev-server.ps1</code>
      </p>
    </section>
  </div>
</template>
