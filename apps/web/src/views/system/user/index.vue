<script setup lang="ts">
// 用户管理列表。
import { onMounted, reactive, ref } from 'vue'
import { Plus, RefreshCw, Search } from 'lucide-vue-next'
import Modal from '@/components/ui/Modal.vue'
import Pagination from '@/components/ui/Pagination.vue'
import ConfirmModal from '@/components/ui/ConfirmModal.vue'
import RowActions from '@/components/ui/RowActions.vue'
import type { RowAction } from '@/components/ui/RowActions.vue'
import { useUserStore } from '@/stores/user'
import { createUser, deleteUsers, listUsers, roleOptions, updateUser } from '@/api/system'
import type { SysUserRow, UserForm } from '@/api/system'

const userStore = useUserStore()

const loading = ref(false)
const rows = ref<SysUserRow[]>([])
const total = ref(0)
const query = reactive({ page: 1, pageSize: 10, keyword: '', status: '' })

const modalOpen = ref(false)
const editing = ref<SysUserRow | null>(null)
const saving = ref(false)
const formError = ref('')
const form = reactive<UserForm>({ username: '', password: '', nickname: '', phone: '', email: '', status: '0', roleIds: [], remark: '' })
const allRoles = ref<Array<{ id: number; name: string }>>([])

const confirmOpen = ref(false)
const pendingDelete = ref<SysUserRow | null>(null)

// 行操作统一使用 RowActions。
function rowActions(row: SysUserRow): RowAction[] {
  const acts: RowAction[] = []
  if (userStore.hasPerm('system:user:edit')) acts.push({ label: '编辑', onClick: () => openEdit(row) })
  if (userStore.hasPerm('system:user:remove')) acts.push({ label: '删除', danger: true, onClick: () => { pendingDelete.value = row; confirmOpen.value = true } })
  return acts
}

async function load() {
  loading.value = true
  try {
    const data = await listUsers({ ...query, keyword: query.keyword || undefined, status: query.status || undefined })
    rows.value = data.list
    total.value = data.total
  } catch (e) {
    formError.value = e instanceof Error ? e.message : '加载失败'
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editing.value = null
  Object.assign(form, { username: '', password: '', nickname: '', phone: '', email: '', status: '0', roleIds: [], remark: '' })
  formError.value = ''
  modalOpen.value = true
}

function openEdit(row: SysUserRow) {
  editing.value = row
  Object.assign(form, {
    username: row.username,
    password: '',
    nickname: row.nickname,
    phone: row.phone ?? '',
    email: row.email ?? '',
    status: row.status,
    roleIds: [...row.roleIds],
    remark: row.remark ?? '',
  })
  formError.value = ''
  modalOpen.value = true
}

async function save() {
  if (!form.username || (!editing.value && !form.password)) {
    formError.value = '用户名必填，新建时密码必填'
    return
  }
  saving.value = true
  formError.value = ''
  try {
    if (editing.value) {
      await updateUser(editing.value.id, form)
    } else {
      await createUser(form)
    }
    modalOpen.value = false
    await load()
  } catch (e) {
    formError.value = e instanceof Error ? e.message : '保存失败'
  } finally {
    saving.value = false
  }
}

async function confirmDelete() {
  if (!pendingDelete.value) return
  try {
    await deleteUsers([pendingDelete.value.id])
    confirmOpen.value = false
    await load()
  } catch (e) {
    formError.value = e instanceof Error ? e.message : '删除失败'
  }
}

function toggleRole(id: number) {
  const idx = form.roleIds.indexOf(id)
  if (idx >= 0) form.roleIds.splice(idx, 1)
  else form.roleIds.push(id)
}

onMounted(async () => {
  await load()
  try {
    allRoles.value = await roleOptions()
  } catch {
    allRoles.value = []
  }
})
</script>

<template>
  <div>
    <!-- 工具栏 -->
    <div class="mb-4 flex flex-wrap items-center gap-3">
      <div class="flex flex-wrap items-center gap-2">
        <input v-model="query.keyword" class="input w-full sm:w-56" placeholder="用户名 / 昵称" @keyup.enter="query.page = 1; load()" />
        <select v-model="query.status" class="input w-full sm:w-32" @change="query.page = 1; load()">
          <option value="">全部状态</option>
          <option value="0">正常</option>
          <option value="1">停用</option>
        </select>
        <button class="btn-secondary h-9 px-3" @click="query.page = 1; load()"><Search :size="14" /> 搜索</button>
        <button class="btn-secondary h-9 px-3" @click="query.keyword = ''; query.status = ''; query.page = 1; load()">
          <RefreshCw :size="14" /> 重置
        </button>
      </div>
      <div class="ml-auto">
        <button v-permission="'system:user:add'" class="btn-primary h-9 px-3" @click="openCreate"><Plus :size="14" /> 新建用户</button>
      </div>
    </div>

    <!-- 表格 -->
    <div class="overflow-x-auto rounded-lg border border-[#dbe6ee] bg-white shadow-[0_1px_2px_rgba(15,23,42,0.035)] dark:border-slate-700 dark:bg-slate-900">
      <table class="data-table">
        <thead>
          <tr>
            <th>ID</th>
            <th>用户名</th>
            <th>昵称</th>
            <th>手机号</th>
            <th>角色</th>
            <th>状态</th>
            <th>创建时间</th>
            <th class="w-52">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="loading"><td colspan="8" class="py-10 text-center text-slate-400">加载中…</td></tr>
          <tr v-else-if="rows.length === 0"><td colspan="8" class="py-10 text-center text-slate-400">暂无用户，点击右上角新建</td></tr>
          <tr v-for="row in rows" v-else :key="row.id">
            <td>{{ row.id }}</td>
            <td class="font-semibold text-slate-900 dark:text-white">{{ row.username }}</td>
            <td>{{ row.nickname }}</td>
            <td>{{ row.phone || '—' }}</td>
            <td>
              <span v-for="n in row.roleNames" :key="n" class="badge badge-brand mr-1">{{ n }}</span>
            </td>
            <td>
              <span class="pill" :class="row.status === '0' ? 'bg-emerald-100 text-emerald-700' : 'bg-slate-100 text-slate-500'">
                {{ row.status === '0' ? '正常' : '停用' }}
              </span>
            </td>
            <td class="text-xs">{{ row.createdAt }}</td>
            <td>
              <RowActions :actions="rowActions(row)" />
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    <Pagination v-model:page="query.page" v-model:page-size="query.pageSize" :total="total" @update:page="load" @update:page-size="load" />

    <!-- 编辑弹窗 -->
    <Modal v-model:open="modalOpen" :title="editing ? `编辑用户 #${editing.id}` : '新建用户'" width-class="max-w-xl">
      <div class="grid gap-4">
        <div class="grid gap-4 sm:grid-cols-2">
          <label class="grid gap-1.5 text-sm font-medium text-slate-700 dark:text-slate-200">
            用户名
            <input v-model="form.username" class="input" placeholder="登录账号" />
          </label>
          <label class="grid gap-1.5 text-sm font-medium text-slate-700 dark:text-slate-200">
            密码 <span class="text-xs font-normal text-slate-400">{{ editing ? '留空不修改' : '必填' }}</span>
            <input v-model="form.password" class="input" type="password" placeholder="至少 6 位" />
          </label>
          <label class="grid gap-1.5 text-sm font-medium text-slate-700 dark:text-slate-200">
            昵称
            <input v-model="form.nickname" class="input" placeholder="显示名称" />
          </label>
          <label class="grid gap-1.5 text-sm font-medium text-slate-700 dark:text-slate-200">
            手机号
            <input v-model="form.phone" class="input" placeholder="选填" />
          </label>
          <label class="grid gap-1.5 text-sm font-medium text-slate-700 dark:text-slate-200">
            邮箱
            <input v-model="form.email" class="input" placeholder="选填" />
          </label>
          <label class="grid gap-1.5 text-sm font-medium text-slate-700 dark:text-slate-200">
            状态
            <select v-model="form.status" class="input">
              <option value="0">正常</option>
              <option value="1">停用</option>
            </select>
          </label>
        </div>
        <div>
          <p class="mb-2 text-sm font-medium text-slate-700 dark:text-slate-200">角色</p>
          <div class="flex flex-wrap gap-2">
            <label
              v-for="r in allRoles"
              :key="r.id"
              class="flex cursor-pointer items-center gap-1.5 rounded-md border border-slate-200 px-2.5 py-1.5 text-sm text-slate-600 dark:border-slate-700 dark:text-slate-300"
              :class="form.roleIds.includes(r.id) ? 'border-brand-300 bg-brand-50 text-brand-700' : ''"
            >
              <input type="checkbox" class="accent-brand-600" :checked="form.roleIds.includes(r.id)" @change="toggleRole(r.id)" />
              {{ r.name }}
            </label>
          </div>
        </div>
        <p v-if="formError" class="text-sm text-red-600">{{ formError }}</p>
      </div>
      <template #footer>
        <button class="btn-secondary h-9 px-4" :disabled="saving" @click="modalOpen = false">取消</button>
        <button class="btn-primary h-9 px-4" :disabled="saving" @click="save">{{ saving ? '保存中…' : '保存' }}</button>
      </template>
    </Modal>

    <ConfirmModal
      v-model:open="confirmOpen"
      :message="`确定删除用户「${pendingDelete?.username ?? ''}」吗？删除后其角色关联一并移除。`"
      @confirm="confirmDelete"
    />
  </div>
</template>
