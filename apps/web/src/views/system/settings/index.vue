<script setup lang="ts">
// 系统设置：分组化参数配置，使用全宽主区与分节布局。
import { computed, onMounted, ref } from 'vue'
import { Database, Download, MailCheck, Plus, RefreshCw, RotateCcw, Trash2 } from 'lucide-vue-next'
import ConfirmModal from '@/components/ui/ConfirmModal.vue'
import {
  createBackup,
  deleteBackup,
  downloadBackup,
  fetchSettings,
  listBackups,
  restoreBackup,
  saveSettings,
  sendTestMail,
  testS3Connection,
} from '@/api/system'
import type { BackupRecordRow, SettingItem, SettingTab } from '@/api/system'

const tabs = ref<SettingTab[]>([])
const activeKey = ref('')
const loading = ref(true)
const saving = ref(false)
const savedMsg = ref('')
const errorMsg = ref('')

// 编辑态：key → value
const draft = ref<Record<string, string>>({})
const initial = ref<Record<string, string>>({})

// 测试邮件
const testMailTo = ref('')
const testMailSending = ref(false)
const testMailMsg = ref('')

// ==================== 数据备份 ====================
const backups = ref<BackupRecordRow[]>([])
const backupsTotal = ref(0)
const backupsLoading = ref(false)
const backupExpireDays = ref(14)
const backupCreating = ref(false)
const testS3Sending = ref(false)
const testS3Msg = ref('')
const restoreConfirmOpen = ref(false)
const deleteConfirmOpen = ref(false)
const pendingBackup = ref<BackupRecordRow | null>(null)

function fmtSize(bytes: number) {
  if (!bytes) return '—'
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`
}

const backupStatusCls = (s: string) =>
  s === 'completed' ? 'bg-emerald-100 text-emerald-700' : s === 'failed' ? 'bg-red-100 text-red-600' : 'bg-amber-100 text-amber-700'
const backupStatusLabel = (s: string) => (s === 'completed' ? '已完成' : s === 'failed' ? '失败' : '进行中')
const backupTriggerLabel = (s: string) => (s === 'scheduled' ? '定时' : '手动')

async function loadBackups() {
  backupsLoading.value = true
  try {
    const data = await listBackups({ page: 1, pageSize: 50 })
    backups.value = data.list
    backupsTotal.value = data.total
  } catch (e) {
    errorMsg.value = e instanceof Error ? e.message : '备份记录加载失败'
  } finally {
    backupsLoading.value = false
  }
}

async function handleTestS3() {
  testS3Sending.value = true
  testS3Msg.value = ''
  try {
    const res = await testS3Connection()
    testS3Msg.value = res.message
  } catch (e) {
    testS3Msg.value = e instanceof Error ? e.message : '连接失败'
  } finally {
    testS3Sending.value = false
  }
}

async function handleCreateBackup() {
  backupCreating.value = true
  errorMsg.value = ''
  try {
    await createBackup(backupExpireDays.value || 0)
    void loadBackups()
    // 异步执行，稍后再刷一次状态
    setTimeout(() => void loadBackups(), 3000)
  } catch (e) {
    errorMsg.value = e instanceof Error ? e.message : '创建备份失败'
  } finally {
    backupCreating.value = false
  }
}

function askRestore(row: BackupRecordRow) {
  pendingBackup.value = row
  restoreConfirmOpen.value = true
}

function askDelete(row: BackupRecordRow) {
  pendingBackup.value = row
  deleteConfirmOpen.value = true
}

async function confirmRestore() {
  if (!pendingBackup.value) return
  try {
    await restoreBackup(pendingBackup.value.id)
    restoreConfirmOpen.value = false
  } catch (e) {
    errorMsg.value = e instanceof Error ? e.message : '恢复失败'
  }
}

async function confirmDelete() {
  if (!pendingBackup.value) return
  try {
    await deleteBackup(pendingBackup.value.id)
    deleteConfirmOpen.value = false
    void loadBackups()
  } catch (e) {
    errorMsg.value = e instanceof Error ? e.message : '删除失败'
  }
}

async function handleDownload(row: BackupRecordRow) {
  try {
    await downloadBackup(row.id, row.fileName)
  } catch (e) {
    errorMsg.value = e instanceof Error ? e.message : '下载失败'
  }
}

const activeTab = computed(() => tabs.value.find((t) => t.key === activeKey.value) ?? tabs.value[0])

async function load() {
  loading.value = true
  try {
    tabs.value = await fetchSettings()
    activeKey.value = tabs.value[0]?.key ?? ''
    const values: Record<string, string> = {}
    for (const tab of tabs.value) {
      for (const sec of tab.sections) {
        for (const item of sec.items) values[item.key] = item.value
      }
    }
    draft.value = { ...values }
    initial.value = { ...values }
    if (tabs.value.some((t) => t.key === 'backup')) void loadBackups()
  } catch (e) {
    errorMsg.value = e instanceof Error ? e.message : '设置加载失败'
  } finally {
    loading.value = false
  }
}

function setItem(key: string, value: string) {
  draft.value[key] = value
}

function isSwitch(item: SettingItem) {
  return item.type === 'switch'
}

function switchValue(item: SettingItem) {
  return draft.value[item.key] === 'true'
}

function toggleSwitch(item: SettingItem) {
  draft.value[item.key] = switchValue(item) ? 'false' : 'true'
}

/** 只提交当前 Tab 内变化过的值 */
async function handleSave() {
  if (!activeTab.value) return
  const changed: Record<string, string> = {}
  for (const sec of activeTab.value.sections) {
    for (const item of sec.items) {
      if (draft.value[item.key] !== initial.value[item.key]) {
        changed[item.key] = draft.value[item.key]
      }
    }
  }
  if (Object.keys(changed).length === 0) {
    savedMsg.value = '没有需要保存的修改'
    return
  }
  saving.value = true
  savedMsg.value = ''
  errorMsg.value = ''
  try {
    await saveSettings(changed)
    for (const key of Object.keys(changed)) {
      const item = activeTab.value.sections.flatMap((s) => s.items).find((i) => i.key === key)
      if (item?.type === 'password') continue
      initial.value[key] = draft.value[key]
    }
    savedMsg.value = '保存成功'
    void load()
  } catch (e) {
    errorMsg.value = e instanceof Error ? e.message : '保存失败'
  } finally {
    saving.value = false
  }
}

async function handleTestMail() {
  if (!testMailTo.value) {
    testMailMsg.value = '请输入收件人邮箱'
    return
  }
  testMailSending.value = true
  testMailMsg.value = ''
  try {
    const res = await sendTestMail(testMailTo.value)
    testMailMsg.value = res.message
  } catch (e) {
    testMailMsg.value = e instanceof Error ? e.message : '发送失败'
  } finally {
    testMailSending.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="grid gap-6">
    <!-- 工具栏：分段标签 + 页面级主操作（docs/06 §2.4 工具栏区域） -->
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div class="flex gap-1 rounded-lg bg-gray-100 p-1 dark:bg-slate-800">
        <button
          v-for="tab in tabs"
          :key="tab.key"
          class="rounded-lg px-4 py-1.5 text-sm font-medium transition-colors"
          :class="activeKey === tab.key ? 'bg-white text-slate-900 shadow-soft-sm dark:bg-slate-700 dark:text-white' : 'text-slate-600 hover:text-slate-900 dark:text-slate-300'"
          @click="activeKey = tab.key"
        >
          {{ tab.title }}
        </button>
      </div>
      <div class="flex items-center gap-3">
        <p v-if="savedMsg" class="text-sm text-green-600">{{ savedMsg }}</p>
        <p v-if="errorMsg" class="text-sm text-red-600">{{ errorMsg }}</p>
        <button class="btn-primary h-9 px-4" type="button" :disabled="saving || loading" @click="handleSave">
          {{ saving ? '保存中…' : '保存设置' }}
        </button>
      </div>
    </div>

    <div v-if="loading" class="py-16 text-center text-sm text-slate-400">设置加载中…</div>

    <!-- 单一视觉主区（docs/06 §2：一个页面只允许一个视觉主区域；分节用标题与分隔线） -->
    <div
      v-else-if="activeTab"
      class="rounded-lg border border-[#dbe6ee] bg-white shadow-[0_1px_2px_rgba(15,23,42,0.035)] dark:border-slate-700 dark:bg-slate-900"
    >
      <section v-for="(sec, si) in activeTab.sections" :key="sec.title" :class="si > 0 ? 'border-t border-[#e5eef5] dark:border-slate-700' : ''">
        <div class="px-5 pb-1 pt-4">
          <h2 class="text-sm font-semibold text-slate-900 dark:text-white">{{ sec.title }}</h2>
          <p v-if="sec.description" class="mt-1 text-xs text-slate-500 dark:text-slate-400">{{ sec.description }}</p>
        </div>
        <div class="mt-2 grid grid-cols-1 gap-x-4 gap-y-5 px-5 py-4 sm:grid-cols-2">
          <div v-for="item in sec.items" :key="item.key" class="min-w-0">
            <label class="mb-1.5 block text-sm text-slate-800 dark:text-slate-200">{{ item.label }}</label>
            <!-- 开关 -->
            <button
              v-if="isSwitch(item)"
              type="button"
              role="switch"
              :aria-checked="switchValue(item)"
              class="relative h-6 w-11 rounded-full transition-colors"
              :class="switchValue(item) ? 'bg-brand-500' : 'bg-slate-300 dark:bg-slate-600'"
              @click="toggleSwitch(item)"
            >
              <span
                class="absolute top-0.5 h-5 w-5 rounded-full bg-white shadow transition-all"
                :class="switchValue(item) ? 'left-[22px]' : 'left-0.5'"
              />
            </button>
            <!-- 输入框 -->
            <input
              v-else
              :type="item.type === 'password' ? 'password' : item.type === 'number' ? 'number' : 'text'"
              :value="draft[item.key] ?? ''"
              :placeholder="item.placeholder || ''"
              class="input h-9 w-full"
              @input="setItem(item.key, ($event.target as HTMLInputElement).value)"
            />
            <p v-if="item.hint" class="mt-1.5 text-xs leading-5 text-slate-500 dark:text-slate-400">{{ item.hint }}</p>
          </div>
        </div>
      </section>

      <!-- 测试邮件（仅邮件设置 Tab，同一主区内续段） -->
      <section v-if="activeTab.key === 'mail'" class="border-t border-[#e5eef5] dark:border-slate-700">
        <div class="px-5 pb-1 pt-4">
          <h2 class="text-sm font-semibold text-slate-900 dark:text-white">发送测试邮件</h2>
          <p class="mt-1 text-xs text-slate-500 dark:text-slate-400">用当前保存的 SMTP 配置发送一封测试邮件，验证配置是否正确</p>
        </div>
        <div class="mt-2 px-5 pb-4">
          <div class="flex items-center gap-3">
            <input v-model="testMailTo" class="input h-9 max-w-sm flex-1" type="email" placeholder="test@example.com" />
            <button class="btn-secondary h-9 shrink-0 px-3" type="button" :disabled="testMailSending" @click="handleTestMail">
              <MailCheck :size="15" class="mr-1.5 inline" />
              {{ testMailSending ? '发送中…' : '发送测试邮件' }}
            </button>
          </div>
          <p v-if="testMailMsg" class="mt-2 text-sm" :class="testMailMsg.includes('成功') || testMailMsg.includes('已发送') ? 'text-green-600' : 'text-red-600'">
            {{ testMailMsg }}
          </p>
        </div>
      </section>

      <!-- 数据备份：S3 测试连接 -->
      <section v-if="activeTab.key === 'backup'" class="border-t border-[#e5eef5] dark:border-slate-700">
        <div class="px-5 pb-1 pt-4">
          <h2 class="text-sm font-semibold text-slate-900 dark:text-white">S3 测试连接</h2>
          <p class="mt-1 text-xs text-slate-500 dark:text-slate-400">使用上方保存的 S3 配置测试连通性（先点“保存设置”再测试）</p>
        </div>
        <div class="mt-2 px-5 pb-4">
          <button class="btn-secondary h-9 px-3" type="button" :disabled="testS3Sending" @click="handleTestS3">
            <Database :size="15" class="mr-1.5 inline" />
            {{ testS3Sending ? '测试中…' : '测试连接' }}
          </button>
          <p v-if="testS3Msg" class="mt-2 text-sm" :class="testS3Msg.includes('成功') ? 'text-green-600' : 'text-red-600'">{{ testS3Msg }}</p>
        </div>
      </section>

      <!-- 数据备份：备份记录 -->
      <section v-if="activeTab.key === 'backup'" class="border-t border-[#e5eef5] dark:border-slate-700">
        <div class="flex flex-wrap items-center justify-between gap-3 px-5 pb-1 pt-4">
          <div>
            <h2 class="text-sm font-semibold text-slate-900 dark:text-white">备份记录</h2>
            <p class="mt-1 text-xs text-slate-500 dark:text-slate-400">创建手动备份并管理已有备份记录</p>
          </div>
          <div class="flex items-center gap-3">
            <label class="flex items-center gap-2 text-sm text-slate-600 dark:text-slate-300">
              过期天数
              <input v-model.number="backupExpireDays" class="input h-9 w-20" type="number" min="0" />
            </label>
            <button class="btn-primary h-9 px-3" type="button" :disabled="backupCreating" @click="handleCreateBackup">
              <Plus :size="14" class="mr-1 inline" /> {{ backupCreating ? '创建中…' : '创建备份' }}
            </button>
            <button class="btn-secondary h-9 px-3" type="button" @click="loadBackups">
              <RefreshCw :size="14" class="mr-1 inline" /> 刷新
            </button>
          </div>
        </div>
        <div class="mt-2 overflow-x-auto px-5 pb-5">
          <table class="data-table rounded-lg border border-[#e5eef5] dark:border-slate-700">
            <thead>
              <tr>
                <th>ID</th>
                <th>状态</th>
                <th>文件名</th>
                <th>大小</th>
                <th>分卷数</th>
                <th>过期时间</th>
                <th>触发方式</th>
                <th>开始时间</th>
                <th class="w-56">操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="backupsLoading"><td colspan="9" class="py-10 text-center text-slate-400">加载中…</td></tr>
              <tr v-else-if="backups.length === 0"><td colspan="9" class="py-10 text-center text-slate-400">暂无备份记录，点击右上角创建</td></tr>
              <tr v-for="row in backups" v-else :key="row.id">
                <td><code class="text-xs">{{ row.recordKey }}</code></td>
                <td><span class="pill" :class="backupStatusCls(row.status)">{{ backupStatusLabel(row.status) }}</span></td>
                <td class="max-w-64 truncate font-mono text-xs" :title="row.fileName">{{ row.fileName }}</td>
                <td class="text-xs">{{ fmtSize(row.sizeBytes) }}</td>
                <td class="text-xs">{{ row.parts }}</td>
                <td class="text-xs">{{ row.expireAt || '—' }}</td>
                <td class="text-xs">{{ backupTriggerLabel(row.triggerType) }}</td>
                <td class="text-xs">{{ row.startedAt }}</td>
                <td>
                  <div class="flex flex-wrap gap-2">
                    <button class="btn-secondary btn-sm" :disabled="row.status !== 'completed'" @click="handleDownload(row)">
                      <Download :size="12" class="mr-1 inline" /> 下载
                    </button>
                    <button class="btn-secondary btn-sm" :disabled="row.status !== 'completed'" @click="askRestore(row)">
                      <RotateCcw :size="12" class="mr-1 inline" /> 恢复
                    </button>
                    <button class="btn-secondary btn-sm !border-red-200 !text-red-600" @click="askDelete(row)">
                      <Trash2 :size="12" class="mr-1 inline" /> 删除
                    </button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
          <p v-if="backupsTotal" class="mt-2 text-xs text-slate-400">共 {{ backupsTotal }} 条备份记录</p>
        </div>
      </section>
    </div>

    <!-- 恢复/删除确认 -->
    <ConfirmModal
      v-model:open="restoreConfirmOpen"
      :message="`确定用备份「${pendingBackup?.fileName ?? ''}」恢复数据库吗？此操作会清空当前全部数据并重放备份，不可撤销。`"
      @confirm="confirmRestore"
    />
    <ConfirmModal
      v-model:open="deleteConfirmOpen"
      :message="`确定删除备份「${pendingBackup?.fileName ?? ''}」吗？S3 对象与记录将一并删除。`"
      @confirm="confirmDelete"
    />
  </div>
</template>
