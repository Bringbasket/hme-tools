<script setup lang="ts">
// 菜单管理（可折叠树形表格：层级缩进 + 展开/折叠箭头）
import { computed, onMounted, reactive, ref } from 'vue'
import { ChevronDown, ChevronRight, Plus, RefreshCw } from 'lucide-vue-next'
import Modal from '@/components/ui/Modal.vue'
import ConfirmModal from '@/components/ui/ConfirmModal.vue'
import RowActions from '@/components/ui/RowActions.vue'
import type { RowAction } from '@/components/ui/RowActions.vue'
import { useUserStore } from '@/stores/user'
import { createMenu, deleteMenu, listMenus, updateMenu } from '@/api/system'
import type { MenuForm, SysMenuRow } from '@/api/system'

const userStore = useUserStore()

// 行操作统一使用 RowActions。
function rowActions(row: SysMenuRow): RowAction[] {
  const acts: RowAction[] = []
  if (row.menu_type !== 'F') acts.push({ label: '新增下级', onClick: () => openCreate(row) })
  if (userStore.hasPerm('system:menu:edit')) acts.push({ label: '编辑', onClick: () => openEdit(row) })
  if (userStore.hasPerm('system:menu:remove')) acts.push({ label: '删除', danger: true, onClick: () => { pendingDelete.value = row; confirmOpen.value = true } })
  return acts
}

const loading = ref(false)
const rows = ref<SysMenuRow[]>([])
/** 折叠状态：id → 是否折叠（默认全部展开） */
const collapsedIds = ref<Record<number, boolean>>({})

function hasChildren(id: number): boolean {
  return rows.value.some((m) => m.parent_id === id)
}

function depthOf(row: SysMenuRow): number {
  let depth = 0
  let cur: SysMenuRow | undefined = row
  while (cur && cur.parent_id !== 0) {
    depth += 1
    cur = rows.value.find((m) => m.id === cur!.parent_id)
  }
  return depth
}

function isCollapsed(id: number): boolean {
  return !!collapsedIds.value[id]
}

function toggleCollapse(id: number) {
  collapsedIds.value = { ...collapsedIds.value, [id]: !collapsedIds.value[id] }
}

function expandAll() {
  collapsedIds.value = {}
}

function collapseAll() {
  const map: Record<number, boolean> = {}
  for (const m of rows.value) {
    if (hasChildren(m.id)) map[m.id] = true
  }
  collapsedIds.value = map
}

/** 祖先链上任意一层被折叠则隐藏 */
function isVisible(row: SysMenuRow): boolean {
  let cur: SysMenuRow | undefined = row
  while (cur && cur.parent_id !== 0) {
    const parent = rows.value.find((m) => m.id === cur!.parent_id)
    if (!parent) return true
    if (isCollapsed(parent.id)) return false
    cur = parent
  }
  return true
}

/** 树序排序（DFS）：子项紧跟父项之后展示，后端按 ParentID 排序会导致子项堆在列表底部 */
function sortTree(items: SysMenuRow[]): SysMenuRow[] {
  const byParent = new Map<number, SysMenuRow[]>()
  for (const m of items) {
    const list = byParent.get(m.parent_id) ?? []
    list.push(m)
    byParent.set(m.parent_id, list)
  }
  const sorted: SysMenuRow[] = []
  const walk = (pid: number) => {
    const children = byParent.get(pid)
    if (!children) return
    children.sort((a, b) => a.order_num - b.order_num)
    for (const c of children) {
      sorted.push(c)
      walk(c.id)
    }
  }
  walk(0)
  // 兜底：孤立节点（父不存在）追加在末尾
  for (const m of items) {
    if (!sorted.includes(m)) sorted.push(m)
  }
  return sorted
}

const visibleRows = computed(() => rows.value.filter((r) => isVisible(r)))

const modalOpen = ref(false)
const editing = ref<SysMenuRow | null>(null)
const saving = ref(false)
const formError = ref('')
const form = reactive<MenuForm>({
  parentId: 0,
  name: '',
  menuType: 'C',
  path: '',
  component: '',
  perms: '',
  icon: '',
  orderNum: 0,
  visible: true,
  status: '0',
})

const confirmOpen = ref(false)
const pendingDelete = ref<SysMenuRow | null>(null)

async function load() {
  loading.value = true
  try {
    rows.value = sortTree(await listMenus())
    collapseAll() // 默认收起所有目录
  } finally {
    loading.value = false
  }
}

function openCreate(parent?: SysMenuRow) {
  editing.value = null
  Object.assign(form, {
    parentId: parent?.id ?? 0,
    name: '',
    menuType: parent ? 'C' : 'M',
    path: '',
    component: '',
    perms: '',
    icon: '',
    orderNum: 0,
    visible: true,
    status: '0',
  })
  formError.value = ''
  modalOpen.value = true
}

function openEdit(row: SysMenuRow) {
  editing.value = row
  Object.assign(form, {
    parentId: row.parent_id,
    name: row.name,
    menuType: row.menu_type,
    path: row.path,
    component: row.component ?? '',
    perms: row.perms ?? '',
    icon: row.icon ?? '',
    orderNum: row.order_num,
    visible: row.visible,
    status: row.status,
  })
  formError.value = ''
  modalOpen.value = true
}

async function save() {
  if (!form.name || !form.path) {
    formError.value = '菜单名称与路由地址必填'
    return
  }
  saving.value = true
  formError.value = ''
  try {
    if (editing.value) {
      await updateMenu(editing.value.id, form)
    } else {
      await createMenu(form)
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
    await deleteMenu(pendingDelete.value.id)
    confirmOpen.value = false
    await load()
  } catch (e) {
    formError.value = e instanceof Error ? e.message : '删除失败'
  }
}

onMounted(load)
</script>

<template>
  <div>
    <div class="mb-4 flex flex-wrap items-center gap-3">
      <button class="btn-secondary h-9 px-3" @click="load()"><RefreshCw :size="14" /> 刷新</button>
      <button class="btn-secondary h-9 px-3" @click="expandAll()">展开全部</button>
      <button class="btn-secondary h-9 px-3" @click="collapseAll()">折叠全部</button>
      <div class="ml-auto">
        <button v-permission="'system:menu:add'" class="btn-primary h-9 px-3" @click="openCreate()"><Plus :size="14" /> 新建菜单</button>
      </div>
    </div>

    <div class="overflow-x-auto rounded-lg border border-[#dbe6ee] bg-white shadow-[0_1px_2px_rgba(15,23,42,0.035)] dark:border-slate-700 dark:bg-slate-900">
      <table class="data-table">
        <thead>
          <tr>
            <th>菜单名称</th>
            <th>类型</th>
            <th>路由地址</th>
            <th>组件路径</th>
            <th>权限标识</th>
            <th>图标</th>
            <th>排序</th>
            <th>状态</th>
            <th class="w-52">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="loading"><td colspan="9" class="py-10 text-center text-slate-400">加载中…</td></tr>
          <tr v-else-if="rows.length === 0"><td colspan="9" class="py-10 text-center text-slate-400">暂无菜单</td></tr>
          <tr v-for="row in visibleRows" v-else :key="row.id">
            <td>
              <span
                class="inline-flex items-center gap-1 font-semibold text-slate-900 dark:text-white"
                :style="{ paddingLeft: `${depthOf(row) * 22}px` }"
              >
                <button
                  v-if="hasChildren(row.id)"
                  class="flex h-6 w-6 shrink-0 items-center justify-center rounded text-slate-400 hover:bg-slate-100 hover:text-slate-600 dark:hover:bg-slate-800"
                  :aria-label="isCollapsed(row.id) ? '展开下级' : '折叠下级'"
                  @click="toggleCollapse(row.id)"
                >
                  <ChevronRight v-if="isCollapsed(row.id)" :size="14" />
                  <ChevronDown v-else :size="14" />
                </button>
                <span v-else class="inline-block h-6 w-6 shrink-0" aria-hidden="true" />
                {{ row.name }}
              </span>
            </td>
            <td>
              <span class="badge" :class="row.menu_type === 'M' ? 'badge-brand' : row.menu_type === 'F' ? 'badge-warning' : 'badge-muted'">
                {{ row.menu_type === 'M' ? '目录' : row.menu_type === 'F' ? '按钮' : '菜单' }}
              </span>
            </td>
            <td><code class="text-xs">{{ row.path }}</code></td>
            <td class="text-xs">{{ row.component || '—' }}</td>
            <td class="text-xs">{{ row.perms || '—' }}</td>
            <td class="text-xs">{{ row.icon || '—' }}</td>
            <td>{{ row.order_num }}</td>
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

    <Modal v-model:open="modalOpen" :title="editing ? `编辑菜单 #${editing.id}` : '新建菜单'" width-class="max-w-xl">
      <div class="grid gap-4 sm:grid-cols-2">
        <label class="grid gap-1.5 text-sm font-medium text-slate-700 dark:text-slate-200">
          菜单名称
          <input v-model="form.name" class="input" />
        </label>
        <label class="grid gap-1.5 text-sm font-medium text-slate-700 dark:text-slate-200">
          类型
          <select v-model="form.menuType" class="input">
            <option value="M">目录</option>
            <option value="C">菜单</option>
            <option value="F">按钮</option>
          </select>
        </label>
        <label class="grid gap-1.5 text-sm font-medium text-slate-700 dark:text-slate-200">
          路由地址
          <input v-model="form.path" class="input" placeholder="/system/xxx" />
        </label>
        <label class="grid gap-1.5 text-sm font-medium text-slate-700 dark:text-slate-200">
          组件路径
          <input v-model="form.component" class="input" placeholder="system/xxx/index" />
        </label>
        <label class="grid gap-1.5 text-sm font-medium text-slate-700 dark:text-slate-200">
          权限标识
          <input v-model="form.perms" class="input" placeholder="system:xxx:list" />
        </label>
        <label class="grid gap-1.5 text-sm font-medium text-slate-700 dark:text-slate-200">
          图标名
          <input v-model="form.icon" class="input" placeholder="users" />
        </label>
        <label class="grid gap-1.5 text-sm font-medium text-slate-700 dark:text-slate-200">
          排序
          <input v-model.number="form.orderNum" class="input" type="number" />
        </label>
        <label class="grid gap-1.5 text-sm font-medium text-slate-700 dark:text-slate-200">
          状态
          <select v-model="form.status" class="input">
            <option value="0">正常</option>
            <option value="1">停用</option>
          </select>
        </label>
        <label class="flex cursor-pointer items-center gap-2 text-sm font-medium text-slate-700 dark:text-slate-200">
          <input v-model="form.visible" type="checkbox" class="accent-brand-600" />
          显示在侧栏
        </label>
        <p v-if="formError" class="text-sm text-red-600 sm:col-span-2">{{ formError }}</p>
      </div>
      <template #footer>
        <button class="btn-secondary h-9 px-4" :disabled="saving" @click="modalOpen = false">取消</button>
        <button class="btn-primary h-9 px-4" :disabled="saving" @click="save">{{ saving ? '保存中…' : '保存' }}</button>
      </template>
    </Modal>

    <ConfirmModal
      v-model:open="confirmOpen"
      :message="`确定删除菜单「${pendingDelete?.name ?? ''}」吗？其下级菜单与角色授权将一并移除。`"
      @confirm="confirmDelete"
    />
  </div>
</template>
