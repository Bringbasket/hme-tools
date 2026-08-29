<script setup lang="ts">
import { computed, ref } from 'vue'
import { Braces, Copy, LoaderCircle, Play } from 'lucide-vue-next'
import { getToken } from '@/utils/auth'
import { mailAccountState } from '../account'
import { showToast } from '../compat'
import AsyncState from '../components/AsyncState.vue'

type Endpoint = { id: string; method: 'GET' | 'POST'; path: string; label: string; needsID?: boolean; body?: string }
type RequestState = 'idle' | 'loading' | 'success' | 'error'

const endpoints: Endpoint[] = [
  { id: 'list', method: 'GET', path: '/api/v1/mail/aliases', label: '获取邮箱列表' },
  { id: 'status', method: 'GET', path: '/api/v1/mail/session/status', label: 'Session 状态' },
  { id: 'refresh', method: 'POST', path: '/api/v1/mail/session/refresh', label: '检查 Session', body: '{}' },
  { id: 'disable', method: 'POST', path: '/api/v1/mail/aliases/{id}/disable', label: '停用邮箱', needsID: true, body: '{}' },
  { id: 'enable', method: 'POST', path: '/api/v1/mail/aliases/{id}/enable', label: '启用邮箱', needsID: true, body: '{}' },
  { id: 'delete', method: 'POST', path: '/api/v1/mail/aliases/{id}/delete', label: '删除邮箱', needsID: true, body: '{}' },
]
const selected = ref(endpoints[0])
const anonymousID = ref('')
const body = ref('')
const responseText = ref('')
const status = ref('')
const loading = ref(false)
const requestState = ref<RequestState>('idle')

const resolvedPath = computed(() => selected.value.path.replace('{id}', encodeURIComponent(anonymousID.value.trim() || '{id}')))
function buildCurl(bearer: string) {
  const lines = [
    `curl --request ${selected.value.method} '${location.origin}${resolvedPath.value}'`,
    `  --header 'Authorization: Bearer ${bearer}'`,
    `  --header 'X-Mail-Account-ID: ${mailAccountState.currentId || 'default'}'`,
  ]
  if (selected.value.method === 'POST') lines.push("  --header 'Content-Type: application/json'", `  --data '${body.value.replaceAll("'", "'\\''")}'`)
  return lines.join(' \\\n')
}
const curl = computed(() => buildCurl('YOUR_ACCESS_TOKEN'))

function choose(endpoint: Endpoint) {
  selected.value = endpoint
  body.value = endpoint.body || ''
  status.value = ''
  responseText.value = ''
  requestState.value = 'idle'
}

async function send() {
  if (selected.value.needsID && !anonymousID.value.trim()) {
    status.value = '参数不完整'
    responseText.value = '请输入 anonymousId 后再发送请求。'
    requestState.value = 'error'
    return
  }
  loading.value = true
  requestState.value = 'loading'
  status.value = ''
  responseText.value = ''
  try {
    if (selected.value.method === 'POST') JSON.parse(body.value || '{}')
    const headers: Record<string, string> = {
      Authorization: `Bearer ${getToken()}`,
      'X-Mail-Account-ID': mailAccountState.currentId || 'default',
    }
    if (selected.value.method === 'POST') headers['Content-Type'] = 'application/json'
    const result = await fetch(resolvedPath.value, { method: selected.value.method, credentials: 'same-origin', headers, body: selected.value.method === 'POST' ? (body.value || '{}') : undefined })
    status.value = `HTTP ${result.status} ${result.statusText}`
    const text = await result.text()
    requestState.value = result.ok ? 'success' : 'error'
    if (!text.trim()) {
      responseText.value = result.ok ? '请求成功，服务器未返回响应正文。' : '请求失败，服务器未返回响应正文。'
    } else {
      try { responseText.value = JSON.stringify(JSON.parse(text), null, 2) } catch { responseText.value = text }
    }
  } catch (error) {
    status.value = '请求失败'
    responseText.value = error instanceof Error ? error.message : String(error)
    requestState.value = 'error'
  }
  finally { loading.value = false }
}

async function copyCurl() {
  await navigator.clipboard.writeText(buildCurl(getToken() || 'YOUR_ACCESS_TOKEN'))
  showToast('已复制包含当前会话凭证的 cURL', 'info')
}
</script>

<template>
  <section class="page api-builder">
    <div class="page-heading"><div><h2>API 调试</h2><p>在当前登录会话中调试邮箱接口，并复制可直接运行的 cURL。</p></div></div>

    <div class="builder-layout">
      <aside class="endpoint-panel"><div class="panel-title"><span><Braces :size="17" />接口列表</span><small>{{ endpoints.length }}</small></div><button v-for="endpoint in endpoints" :key="endpoint.id" :class="{ active: selected.id === endpoint.id }" @click="choose(endpoint)"><span :class="['method', endpoint.method.toLowerCase()]">{{ endpoint.method }}</span><span>{{ endpoint.label }}</span></button></aside>
      <div class="request-workspace">
        <section class="work-section">
          <div class="section-heading"><div><h3>{{ selected.label }}</h3></div><button class="button primary" :disabled="loading || Boolean(selected.needsID && !anonymousID.trim())" @click="send"><LoaderCircle v-if="loading" :size="16" class="spin" /><Play v-else :size="16" />发送请求</button></div>
          <div class="request-line"><span :class="['method', selected.method.toLowerCase()]">{{ selected.method }}</span><code>{{ resolvedPath }}</code></div>
          <label v-if="selected.needsID" class="field"><span>anonymousId</span><input v-model="anonymousID" placeholder="邮箱记录的 anonymousId" /></label>
          <label v-if="selected.method === 'POST'" class="field"><span>JSON 请求体</span><textarea v-model="body" class="code-input" rows="8" spellcheck="false" /></label>
          <div class="code-block"><button class="icon-button copy-button" title="复制包含当前会话凭证的 cURL" aria-label="复制包含当前会话凭证的 cURL" @click="copyCurl"><Copy :size="16" /></button><pre>{{ curl }}</pre></div>
        </section>
        <section class="work-section response-section">
          <div class="section-heading"><h3>响应结果</h3><span class="response-status" :class="requestState">{{ requestState === 'loading' ? '请求中' : status || '等待请求' }}</span></div>
          <div v-if="requestState === 'idle' || requestState === 'loading'" class="response-placeholder" aria-live="polite">
            <AsyncState v-if="requestState === 'loading'" state="loading" title="正在发送请求" />
            <AsyncState v-else state="empty" title="尚未发送请求" detail="选择接口并填写必要参数后发送请求"><template #icon><Braces :size="25" /></template></AsyncState>
          </div>
          <pre v-else class="response-output" :class="requestState" aria-live="polite">{{ responseText }}</pre>
        </section>
      </div>
    </div>
  </section>
</template>

<style scoped>
.response-status.loading { color: var(--primary-text); background: var(--primary-soft); }
</style>
