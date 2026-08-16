<script setup lang="ts">
// 字典管理（左：字典类型，右：字典数据）
import { onMounted, reactive, ref } from 'vue'
import { Plus, RefreshCw, Search } from 'lucide-vue-next'
import Modal from '@/components/ui/Modal.vue'
import Pagination from '@/components/ui/Pagination.vue'
import ConfirmModal from '@/components/ui/ConfirmModal.vue'
import RowActions from '@/components/ui/RowActions.vue'
import type { RowAction } from '@/components/ui/RowActions.vue'
import { useUserStore } from '@/stores/user'
import {
  createDictData,
  createDictType,
  deleteDictData,
  deleteDictTypes,
  listDictData,
  listDictTypes,
  updateDictData,
  updateDictType,
} from '@/api/system'
import type { DictDataForm, DictTypeForm, SysDictDataRow, SysDictTypeRow } from '@/api/system'

const userStore = useUserStore()

// 行操作统一使用 RowActions。
function rowActions(row: SysDictDataRow): RowAction[] {
  const acts: RowAction[] = []
  if (userStore.hasPerm('system:dict:edit')) acts.push({ label: '编辑', onClick: () => openDataEdit(row) })
  if (userStore.hasPerm('system:dict:remove')) acts.push({ label: '删除', danger: true, onClick: () => { pendingData.value = row; dataConfirmOpen.value = true } })
  return acts
}

// ===== 类型 =====
const types = ref<SysDictTypeRow[]>([])
const typesTotal = ref(0)
const typeQuery = reactive({ page: 1, pageSize: 10, keyword: '' })
const typeModalOpen = ref(false)
const editingType = ref<SysDictTypeRow | null>(null)
const typeForm = reactive<DictTypeForm>({ name: '', type: '', status: '0', remark: '' })
const typeConfirmOpen = ref(false)
const pendingType = ref<SysDictTypeRow | null>(null)

// ===== 数据 =====
const selectedType = ref('')
const dataRows = ref<SysDictDataRow[]>([])
const dataTotal = ref(0)
const dataQuery = reactive({ page: 1, pageSize: 10, keyword: '' })
const dataModalOpen = ref(false)
const editingData = ref<SysDictDataRow | null>(null)
const dataForm = reactive<DictDataForm>({ sort: 0, label: '', value: '', dictType: '', status: '0', remark: '' })
const dataConfirmOpen = ref(false)
const pendingData = ref<SysDictDataRow | null>(null)

const saving = ref(false)
const formError = ref('')

async function loadTypes() {
  const data = await listDictTypes({ ...typeQuery, keyword: typeQuery.keyword || undefined })
  types.value = data.list
  typesTotal.value = data.total
}

async function loadData() {
  if (!selectedType.value) {
    dataRows.value = []
    dataTotal.value = 0
    return
  }
  const data = await listDictData({ ...dataQuery, dictType: selectedType.value, keyword: dataQuery.keyword || undefined })
  dataRows.value = data.list
  dataTotal.value = data.total
}

function selectType(t: SysDictTypeRow) {
  selectedType.value = t.type
  dataQuery.page = 1
  void loadData()
}

// 类型 CRUD
function openTypeCreate() {
  editingType.value = null
  Object.assign(typeForm, { name: '', type: '', status: '0', remark: '' })
  formError.value = ''
  typeModalOpen.value = true
}

function openTypeEdit(row: SysDictTypeRow) {
  editingType.value = row
  Object.assign(typeForm, { name: row.name, type: row.type, status: row.status, remark: row.remark ?? '' })
  formError.value = ''
  typeModalOpen.value = true
}

async function saveType() {
  if (!typeForm.name || !typeForm.type) {
    formError.value = '名称与类型必填'
    return
  }
  saving.value = true
  try {
    if (editingType.value) await updateDictType(editingType.value.id, typeForm)
    else await createDictType(typeForm)
    typeModalOpen.value = false
    await loadTypes()
  } catch (e) {
    formError.value = e instanceof Error ? e.message : '保存失败'
  } finally {
    saving.value = false
  }
}

async function confirmDeleteType() {
  if (!pendingType.value) return
  try {
    await deleteDictTypes([pendingType.value.id])
    typeConfirmOpen.value = false
    if (selectedType.value === pendingType.value.type) {
      selectedType.value = ''
      dataRows.value = []
    }
    await loadTypes()
  } catch (e) {
    formError.value = e instanceof Error ? e.message : '删除失败'
  }
}

// 数据 CRUD
function openDataCreate() {
  if (!selectedType.value) return
  editingData.value = null
  Object.assign(dataForm, { sort: 0, label: '', value: '', dictType: selectedType.value, status: '0', remark: '' })
  formError.value = ''
  dataModalOpen.value = true
}

function openDataEdit(row: SysDictDataRow) {
  editingData.value = row
  Object.assign(dataForm, {
    sort: row.sort,
    label: row.label,
    value: row.value,
    dictType: row.dict_type,
    status: row.status,
    remark: row.remark ?? '',
  })
  formError.value = ''
  dataModalOpen.value = true
}

async function saveData() {
  if (!dataForm.label || !dataForm.value) {
    formError.value = '标签与键值必填'
    return
  }
  saving.value = true
  try {
    if (editingData.value) await updateDictData(editingData.value.id, dataForm)
    else await createDictData(dataForm)
    dataModalOpen.value = false
    await loadData()
  } catch (e) {
    formError.value = e instanceof Error ? e.message : '保存失败'
  } finally {
    saving.value = false
  }
}

async function confirmDeleteData() {
  if (!pendingData.value) return
  try {
    await deleteDictData([pendingData.value.id])
    dataConfirmOpen.value = false
    await loadData()
  } catch (e) {
    formError.value = e instanceof Error ? e.message : '删除失败'
  }
}

onMounted(loadTypes)
</script>

<template>
  <div class="grid gap-6 lg:grid-cols-[320px_1fr]">
    <!-- 左：字典类型 -->
    <section class="rounded-lg border border-[#dbe6ee] bg-white p-4 shadow-[0_1px_2px_rgba(15,23,42,0.035)] dark:border-slate-700 dark:bg-slate-900">
      <div class="mb-3 flex items-center gap-2">
        <input v-model="typeQuery.keyword" class="input h-8 w-full" placeholder="搜索字典类型" @keyup.enter="typeQuery.page = 1; loadTypes()" />
        <button class="btn-secondary h-8 w-9 shrink-0" @click="typeQuery.page = 1; loadTypes()"><Search :size="13" /></button>
        <button v-permission="'system:dict:add'" class="btn-primary h-8 w-9 shrink-0" @click="openTypeCreate"><Plus :size="13" /></button>
      </div>
      <div class="grid gap-1">
        <button
          v-for="t in types"
          :key="t.id"
          class="flex items-center justify-between rounded-md px-3 py-2.5 text-left text-sm transition-colors"
          :class="selectedType === t.type ? 'bg-[#e9f8f5] text-brand-700 dark:bg-brand-500/15 dark:text-brand-300' : 'text-slate-700 hover:bg-slate-50 dark:text-slate-300 dark:hover:bg-slate-800'"
          @click="selectType(t)"
        >
          <span class="truncate">{{ t.name }}</span>
          <span class="ml-2 flex shrink-0 items-center gap-1">
            <button
              v-permission="'system:dict:edit'"
              class="rounded px-1.5 py-0.5 text-xs hover:bg-white dark:hover:bg-slate-700"
              @click.stop="openTypeEdit(t)"
            >
              编辑
            </button>
            <button
              v-permission="'system:dict:remove'"
              class="rounded px-1.5 py-0.5 text-xs text-red-500 hover:bg-white dark:hover:bg-slate-700"
              @click.stop="pendingType = t; typeConfirmOpen = true"
            >
              删
            </button>
          </span>
        </button>
        <p v-if="types.length === 0" class="py-6 text-center text-xs text-slate-400">暂无字典类型</p>
      </div>
      <Pagination v-model:page="typeQuery.page" v-model:page-size="typeQuery.pageSize" :total="typesTotal" @update:page="loadTypes" @update:page-size="loadTypes" />
    </section>

    <!-- 右：字典数据 -->
    <section class="rounded-lg border border-[#dbe6ee] bg-white p-4 shadow-[0_1px_2px_rgba(15,23,42,0.035)] dark:border-slate-700 dark:bg-slate-900">
      <div class="mb-3 flex items-center justify-between gap-2">
        <h3 class="text-sm font-bold text-slate-900 dark:text-white">
          {{ selectedType ? `字典数据 · ${selectedType}` : '请选择左侧字典类型' }}
        </h3>
        <div class="flex items-center gap-2">
          <input
            v-model="dataQuery.keyword"
            class="input h-8 w-full sm:w-44"
            placeholder="标签 / 键值"
            :disabled="!selectedType"
            @keyup.enter="dataQuery.page = 1; loadData()"
          />
          <button class="btn-secondary h-8 px-2.5" :disabled="!selectedType" @click="dataQuery.page = 1; loadData()"><RefreshCw :size="13" /></button>
          <button v-permission="'system:dict:add'" class="btn-primary h-8 px-2.5" :disabled="!selectedType" @click="openDataCreate">
            <Plus :size="13" /> 新增
          </button>
        </div>
      </div>
      <div class="overflow-x-auto">
        <table class="data-table">
          <thead>
            <tr>
              <th>排序</th>
              <th>标签</th>
              <th>键值</th>
              <th>状态</th>
              <th class="w-52">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="!selectedType"><td colspan="5" class="py-10 text-center text-slate-400">先选择字典类型</td></tr>
            <tr v-else-if="dataRows.length === 0"><td colspan="5" class="py-10 text-center text-slate-400">暂无数据</td></tr>
            <tr v-for="row in dataRows" v-else :key="row.id">
              <td>{{ row.sort }}</td>
              <td class="font-semibold text-slate-900 dark:text-white">{{ row.label }}</td>
              <td><code class="rounded bg-slate-100 px-1.5 py-0.5 text-xs dark:bg-slate-800">{{ row.value }}</code></td>
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
      <Pagination v-model:page="dataQuery.page" v-model:page-size="dataQuery.pageSize" :total="dataTotal" @update:page="loadData" @update:page-size="loadData" />
    </section>

    <!-- 类型弹窗 -->
    <Modal v-model:open="typeModalOpen" :title="editingType ? `编辑字典类型 #${editingType.id}` : '新建字典类型'" width-class="max-w-md">
      <div class="grid gap-4">
        <label class="grid gap-1.5 text-sm font-medium text-slate-700 dark:text-slate-200">
          字典名称
          <input v-model="typeForm.name" class="input" />
        </label>
        <label class="grid gap-1.5 text-sm font-medium text-slate-700 dark:text-slate-200">
          字典类型
          <input v-model="typeForm.type" class="input" placeholder="sys_xxx" />
        </label>
        <label class="grid gap-1.5 text-sm font-medium text-slate-700 dark:text-slate-200">
          状态
          <select v-model="typeForm.status" class="input">
            <option value="0">正常</option>
            <option value="1">停用</option>
          </select>
        </label>
        <p v-if="formError" class="text-sm text-red-600">{{ formError }}</p>
      </div>
      <template #footer>
        <button class="btn-secondary h-9 px-4" :disabled="saving" @click="typeModalOpen = false">取消</button>
        <button class="btn-primary h-9 px-4" :disabled="saving" @click="saveType">{{ saving ? '保存中…' : '保存' }}</button>
      </template>
    </Modal>

    <!-- 数据弹窗 -->
    <Modal v-model:open="dataModalOpen" :title="editingData ? `编辑字典数据 #${editingData.id}` : '新增字典数据'" width-class="max-w-md">
      <div class="grid gap-4">
        <div class="grid grid-cols-2 gap-4">
          <label class="grid gap-1.5 text-sm font-medium text-slate-700 dark:text-slate-200">
            排序
            <input v-model.number="dataForm.sort" class="input" type="number" />
          </label>
          <label class="grid gap-1.5 text-sm font-medium text-slate-700 dark:text-slate-200">
            状态
            <select v-model="dataForm.status" class="input">
              <option value="0">正常</option>
              <option value="1">停用</option>
            </select>
          </label>
        </div>
        <label class="grid gap-1.5 text-sm font-medium text-slate-700 dark:text-slate-200">
          标签
          <input v-model="dataForm.label" class="input" />
        </label>
        <label class="grid gap-1.5 text-sm font-medium text-slate-700 dark:text-slate-200">
          键值
          <input v-model="dataForm.value" class="input" />
        </label>
        <p v-if="formError" class="text-sm text-red-600">{{ formError }}</p>
      </div>
      <template #footer>
        <button class="btn-secondary h-9 px-4" :disabled="saving" @click="dataModalOpen = false">取消</button>
        <button class="btn-primary h-9 px-4" :disabled="saving" @click="saveData">{{ saving ? '保存中…' : '保存' }}</button>
      </template>
    </Modal>

    <ConfirmModal v-model:open="typeConfirmOpen" :message="`确定删除字典类型「${pendingType?.name ?? ''}」吗？`" @confirm="confirmDeleteType" />
    <ConfirmModal v-model:open="dataConfirmOpen" :message="`确定删除字典数据「${pendingData?.label ?? ''}」吗？`" @confirm="confirmDeleteData" />
  </div>
</template>
