<script setup lang="ts">
// 角色管理（含菜单授权）
import { onMounted, reactive, ref } from 'vue'
import { Plus, RefreshCw, Search } from 'lucide-vue-next'
import Modal from '@/components/ui/Modal.vue'
import Pagination from '@/components/ui/Pagination.vue'
import ConfirmModal from '@/components/ui/ConfirmModal.vue'
import RowActions from '@/components/ui/RowActions.vue'
import type { RowAction } from '@/components/ui/RowActions.vue'
import { useUserStore } from '@/stores/user'
import { createRole, deleteRoles, listMenus, listRoles, updateRole } from '@/api/system'
import type { RoleForm, SysMenuRow, SysRoleRow } from '@/api/system'

const userStore = useUserStore()

const loading = ref(false)
const rows = ref<SysRoleRow[]>([])
const total = ref(0)
const query = reactive({ page: 1, pageSize: 10, keyword: '' })

const modalOpen = ref(false)
const editing = ref<SysRoleRow | null>(null)
const saving = ref(false)
const formError = ref('')
const form = reactive<RoleForm>({ name: '', code: '', sort: 0, isAdmin: false, status: '0', remark: '', menuIds: [] })

const confirmOpen = ref(false)
const pendingDelete = ref<SysRoleRow | null>(null)

// 行操作统一使用 RowActions。
function rowActions(row: SysRoleRow): RowAction[] {
  const acts: RowAction[] = []
  if (userStore.hasPerm('system:role:edit')) acts.push({ label: '编辑', onClick: () => openEdit(row) })
  acts.push({ label: '授权', onClick: () => openAssign(row) })
  if (userStore.hasPerm('system:role:remove')) acts.push({ label: '删除', danger: true, onClick: () => { pendingDelete.value = row; confirmOpen.value = true } })
  return acts
}

// 菜单授权树
const assignOpen = ref(false)
const assignRole = ref<SysRoleRow | null>(null)
const menuTree = ref<SysMenuRow[]>([])
const checkedIds = ref<number[]>([])

async function load() {
  loading.value = true
  try {
    const data = await listRoles({ ...query, keyword: query.keyword || undefined })
    rows.value = data.list
    total.value = data.total
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editing.value = null
  Object.assign(form, { name: '', code: '', sort: 0, isAdmin: false, status: '0', remark: '', menuIds: [] })
  formError.value = ''
  modalOpen.value = true
}

function openEdit(row: SysRoleRow) {
  editing.value = row
  Object.assign(form, {
    name: row.name,
    code: row.code,
    sort: row.sort,
    isAdmin: row.is_admin,
    status: row.status,
    remark: row.remark ?? '',
    menuIds: [],
  })
  formError.value = ''
  modalOpen.value = true
}

async function save() {
  if (!form.name || !form.code) {
    formError.value = '角色名称与权限字符必填'
    return
  }
  saving.value = true
  formError.value = ''
  try {
    if (editing.value) {
      await updateRole(editing.value.id, form)
    } else {
      await createRole(form)
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
    await deleteRoles([pendingDelete.value.id])
    confirmOpen.value = false
    await load()
  } catch (e) {
    formError.value = e instanceof Error ? e.message : '删除失败'
  }
}

// 菜单授权
async function openAssign(row: SysRoleRow) {
  assignRole.value = row
  checkedIds.value = []
  try {
    menuTree.value = await listMenus()
  } catch {
    menuTree.value = []
  }
  assignOpen.value = true
}

function toggleMenu(id: number) {
  const idx = checkedIds.value.indexOf(id)
  if (idx >= 0) checkedIds.value.splice(idx, 1)
  else checkedIds.value.push(id)
}

function menuDepth(items: SysMenuRow[], id: number): number {
  let depth = 0
  let cur = items.find((m) => m.id === id)
  while (cur && cur.parent_id !== 0) {
    depth += 1
    cur = items.find((m) => m.id === cur!.parent_id)
  }
  return depth
}

async function saveAssign() {
  if (!assignRole.value) return
  saving.value = true
  try {
    await updateRole(assignRole.value.id, {
      name: assignRole.value.name,
      code: assignRole.value.code,
      sort: assignRole.value.sort,
      isAdmin: assignRole.value.is_admin,
      status: assignRole.value.status,
      remark: assignRole.value.remark ?? '',
      menuIds: checkedIds.value,
    })
    assignOpen.value = false
    await load()
  } catch (e) {
    formError.value = e instanceof Error ? e.message : '授权失败'
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>

<template>
  <div>
    <div class="mb-4 flex flex-wrap items-center gap-3">
      <div class="flex flex-wrap items-center gap-2">
        <input v-model="query.keyword" class="input w-full sm:w-56" placeholder="角色名称 / 权限字符" @keyup.enter="query.page = 1; load()" />
        <button class="btn-secondary h-9 px-3" @click="query.page = 1; load()"><Search :size="14" /> 搜索</button>
        <button class="btn-secondary h-9 px-3" @click="query.keyword = ''; query.page = 1; load()"><RefreshCw :size="14" /> 重置</button>
      </div>
      <div class="ml-auto">
        <button v-permission="'system:role:add'" class="btn-primary h-9 px-3" @click="openCreate"><Plus :size="14" /> 新建角色</button>
      </div>
    </div>

    <div class="overflow-x-auto rounded-lg border border-[#dbe6ee] bg-white shadow-[0_1px_2px_rgba(15,23,42,0.035)] dark:border-slate-700 dark:bg-slate-900">
      <table class="data-table">
        <thead>
          <tr>
            <th>ID</th>
            <th>角色名称</th>
            <th>权限字符</th>
            <th>排序</th>
            <th>类型</th>
            <th>状态</th>
            <th class="w-52">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="loading"><td colspan="7" class="py-10 text-center text-slate-400">加载中…</td></tr>
          <tr v-else-if="rows.length === 0"><td colspan="7" class="py-10 text-center text-slate-400">暂无角色</td></tr>
          <tr v-for="row in rows" v-else :key="row.id">
            <td>{{ row.id }}</td>
            <td class="font-semibold text-slate-900 dark:text-white">{{ row.name }}</td>
            <td><code class="rounded bg-slate-100 px-1.5 py-0.5 text-xs dark:bg-slate-800">{{ row.code }}</code></td>
            <td>{{ row.sort }}</td>
            <td>
              <span v-if="row.is_admin" class="badge badge-brand">超级管理员</span>
              <span v-else class="badge badge-muted">普通角色</span>
            </td>
            <td>
              <span class="pill" :class="row.status === '0' ? 'bg-emerald-100 text-emerald-700' : 'bg-slate-100 text-slate-500'">
                {{ row.status === '0' ? '正常' : '停用' }}
              </span>
            </td>
            <td>
              <RowActions :actions="rowActions(row)" />
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    <Pagination v-model:page="query.page" v-model:page-size="query.pageSize" :total="total" @update:page="load" @update:page-size="load" />

    <Modal v-model:open="modalOpen" :title="editing ? `编辑角色 #${editing.id}` : '新建角色'" width-class="max-w-lg">
      <div class="grid gap-4">
        <label class="grid gap-1.5 text-sm font-medium text-slate-700 dark:text-slate-200">
          角色名称
          <input v-model="form.name" class="input" placeholder="如 审核员" />
        </label>
        <label class="grid gap-1.5 text-sm font-medium text-slate-700 dark:text-slate-200">
          权限字符
          <input v-model="form.code" class="input" placeholder="如 auditor" />
        </label>
        <div class="grid grid-cols-2 gap-4">
          <label class="grid gap-1.5 text-sm font-medium text-slate-700 dark:text-slate-200">
            排序
            <input v-model.number="form.sort" class="input" type="number" />
          </label>
          <label class="grid gap-1.5 text-sm font-medium text-slate-700 dark:text-slate-200">
            状态
            <select v-model="form.status" class="input">
              <option value="0">正常</option>
              <option value="1">停用</option>
            </select>
          </label>
        </div>
        <label class="flex cursor-pointer items-center gap-2 text-sm font-medium text-slate-700 dark:text-slate-200">
          <input v-model="form.isAdmin" type="checkbox" class="accent-brand-600" />
          超级管理员（跳过权限校验）
        </label>
        <label class="grid gap-1.5 text-sm font-medium text-slate-700 dark:text-slate-200">
          备注
          <input v-model="form.remark" class="input" placeholder="选填" />
        </label>
        <p v-if="formError" class="text-sm text-red-600">{{ formError }}</p>
      </div>
      <template #footer>
        <button class="btn-secondary h-9 px-4" :disabled="saving" @click="modalOpen = false">取消</button>
        <button class="btn-primary h-9 px-4" :disabled="saving" @click="save">{{ saving ? '保存中…' : '保存' }}</button>
      </template>
    </Modal>

    <!-- 菜单授权 -->
    <Modal v-model:open="assignOpen" :title="`菜单授权 — ${assignRole?.name ?? ''}`" width-class="max-w-md">
      <div class="grid gap-1">
        <label
          v-for="m in menuTree"
          :key="m.id"
          class="flex cursor-pointer items-center gap-2 rounded-md px-2 py-1.5 text-sm text-slate-700 hover:bg-slate-50 dark:text-slate-200 dark:hover:bg-slate-800"
          :style="{ paddingLeft: `${12 + menuDepth(menuTree, m.id) * 20}px` }"
        >
          <input type="checkbox" class="accent-brand-600" :checked="checkedIds.includes(m.id)" @change="toggleMenu(m.id)" />
          <span :class="m.menu_type === 'M' ? 'font-semibold' : ''">{{ m.name }}</span>
        </label>
      </div>
      <template #footer>
        <button class="btn-secondary h-9 px-4" :disabled="saving" @click="assignOpen = false">取消</button>
        <button class="btn-primary h-9 px-4" :disabled="saving" @click="saveAssign">{{ saving ? '保存中…' : '保存授权' }}</button>
      </template>
    </Modal>

    <ConfirmModal
      v-model:open="confirmOpen"
      :message="`确定删除角色「${pendingDelete?.name ?? ''}」吗？`"
      @confirm="confirmDelete"
    />
  </div>
</template>
