<script setup lang="ts">
// 操作日志（只读）
import { onMounted, reactive, ref } from 'vue'
import { RefreshCw, Search } from 'lucide-vue-next'
import Pagination from '@/components/ui/Pagination.vue'
import { listOperLogs } from '@/api/system'
import type { SysOperLogRow } from '@/api/system'

const loading = ref(false)
const rows = ref<SysOperLogRow[]>([])
const total = ref(0)
const query = reactive({ page: 1, pageSize:10, title: '', operator: '', status: '' })

async function load() {
  loading.value = true
  try {
    const data = await listOperLogs({
      ...query,
      title: query.title || undefined,
      operator: query.operator || undefined,
      status: query.status || undefined,
    })
    rows.value = data.list
    total.value = data.total
  } finally {
    loading.value = false
  }
}

const bizNames: Record<string, string> = {
  INSERT: '新增',
  UPDATE: '修改',
  DELETE: '删除',
  OTHER: '其他',
}

onMounted(load)
</script>

<template>
  <div>
    <div class="mb-4 flex flex-wrap items-center gap-3">
      <div class="flex flex-wrap items-center gap-2">
        <input v-model="query.title" class="input w-full sm:w-44" placeholder="模块标题" @keyup.enter="query.page = 1; load()" />
        <input v-model="query.operator" class="input w-full sm:w-44" placeholder="操作人" @keyup.enter="query.page = 1; load()" />
        <select v-model="query.status" class="input w-full sm:w-28" @change="query.page = 1; load()">
          <option value="">全部结果</option>
          <option value="ok">成功</option>
          <option value="fail">失败</option>
        </select>
        <button class="btn-secondary h-9 px-3" @click="query.page = 1; load()"><Search :size="14" /> 搜索</button>
        <button class="btn-secondary h-9 px-3" @click="query.title = ''; query.operator = ''; query.status = ''; query.page = 1; load()">
          <RefreshCw :size="14" /> 重置
        </button>
      </div>
    </div>

    <div class="overflow-x-auto rounded-lg border border-[#dbe6ee] bg-white shadow-[0_1px_2px_rgba(15,23,42,0.035)] dark:border-slate-700 dark:bg-slate-900">
      <table class="data-table">
        <thead>
          <tr>
            <th>模块</th>
            <th>类型</th>
            <th>请求</th>
            <th>操作人</th>
            <th>IP</th>
            <th>结果</th>
            <th>耗时</th>
            <th>时间</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="loading"><td colspan="8" class="py-10 text-center text-slate-400">加载中…</td></tr>
          <tr v-else-if="rows.length === 0"><td colspan="8" class="py-10 text-center text-slate-400">暂无日志</td></tr>
          <tr v-for="row in rows" v-else :key="row.id">
            <td class="font-semibold text-slate-900 dark:text-white">{{ row.title }}</td>
            <td>
              <span class="badge" :class="row.business_type === 'DELETE' ? 'badge-danger' : row.business_type === 'INSERT' ? 'badge-success' : 'badge-muted'">
                {{ bizNames[row.business_type] ?? row.business_type }}
              </span>
            </td>
            <td>
              <div>
                <span class="rounded bg-slate-100 px-1.5 py-0.5 text-[11px] font-bold text-slate-500 dark:bg-slate-800 dark:text-slate-400">{{ row.method }}</span>
                <code class="ml-1.5 text-xs">{{ row.path }}</code>
              </div>
            </td>
            <td>{{ row.operator_name || '—' }}</td>
            <td class="text-xs">{{ row.ip }}</td>
            <td>
              <span class="pill" :class="row.status_code < 400 ? 'bg-emerald-100 text-emerald-700' : 'bg-red-100 text-red-700'">
                {{ row.status_code }}
              </span>
            </td>
            <td class="text-xs">{{ row.duration_ms }}ms</td>
            <td class="text-xs">{{ row.created_at }}</td>
          </tr>
        </tbody>
      </table>
    </div>
    <Pagination v-model:page="query.page" v-model:page-size="query.pageSize" :total="total" @update:page="load" @update:page-size="load" />
  </div>
</template>
