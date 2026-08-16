// 系统管理接口（与后端字段命名一致：ent 实体为 snake_case，自定义视图为 camelCase）
// 注意：ent 默认把 int64 字段序列化为 JSON 字符串（保留精度），这里统一归一化为 number
import { http, unwrap } from './http'
import type { PageResult } from './http'

const num = (v: unknown): number => (typeof v === 'string' ? Number(v) : Number(v ?? 0))

// ==================== 用户 ====================

export interface SysUserRow {
  id: number
  username: string
  nickname: string
  phone?: string | null
  email?: string | null
  status: string
  roleIds: number[]
  roleNames: string[]
  remark?: string | null
  createdAt: string
}

export interface UserForm {
  username: string
  password: string
  nickname: string
  phone?: string
  email?: string
  status: string
  roleIds: number[]
  remark?: string
}

export const listUsers = (params: { page: number; pageSize: number; keyword?: string; status?: string }) =>
  unwrap<PageResult<SysUserRow>>(http.get('/system/users', { params }))

export const createUser = (data: UserForm) => unwrap<{ id: number }>(http.post('/system/users', data))

export const updateUser = (id: number, data: UserForm) => unwrap<null>(http.put(`/system/users/${id}`, data))

export const deleteUsers = (ids: number[]) => unwrap<null>(http.delete('/system/users', { params: { ids: ids.join(',') } }))

// ==================== 角色 ====================

export interface SysRoleRow {
  id: number
  name: string
  code: string
  sort: number
  is_admin: boolean
  status: string
  remark?: string | null
}

export interface RoleForm {
  name: string
  code: string
  sort: number
  isAdmin: boolean
  status: string
  remark?: string
  menuIds: number[]
}

const mapRole = (r: SysRoleRow): SysRoleRow => ({ ...r, id: num(r.id), sort: num(r.sort) })

export const listRoles = async (params: { page: number; pageSize: number; keyword?: string }) => {
  const data = await unwrap<PageResult<SysRoleRow>>(http.get('/system/roles', { params }))
  return { ...data, list: data.list.map(mapRole) }
}

export const roleOptions = async () => {
  const data = await unwrap<Array<{ id: number; name: string; code: string }>>(http.get('/system/roles/options'))
  return data.map((r) => ({ ...r, id: num(r.id) }))
}

export const createRole = (data: RoleForm) => unwrap<{ id: number }>(http.post('/system/roles', data))

export const updateRole = (id: number, data: RoleForm) => unwrap<null>(http.put(`/system/roles/${id}`, data))

export const deleteRoles = (ids: number[]) => unwrap<null>(http.delete('/system/roles', { params: { ids: ids.join(',') } }))

// ==================== 菜单 ====================

export interface SysMenuRow {
  id: number
  parent_id: number
  name: string
  menu_type: string
  path: string
  component?: string | null
  perms?: string | null
  icon?: string | null
  order_num: number
  visible: boolean
  status: string
}

export interface MenuForm {
  parentId: number
  name: string
  menuType: string
  path: string
  component?: string
  perms?: string
  icon?: string
  orderNum: number
  visible: boolean
  status: string
}

const mapMenu = (m: SysMenuRow): SysMenuRow => ({ ...m, id: num(m.id), parent_id: num(m.parent_id), order_num: num(m.order_num) })

export const listMenus = async () => {
  const data = await unwrap<SysMenuRow[]>(http.get('/system/menus'))
  return data.map(mapMenu)
}

export const createMenu = (data: MenuForm) => unwrap<{ id: number }>(http.post('/system/menus', data))

export const updateMenu = (id: number, data: MenuForm) => unwrap<null>(http.put(`/system/menus/${id}`, data))

export const deleteMenu = (id: number) => unwrap<null>(http.delete(`/system/menus/${id}`))

// ==================== 字典 ====================

export interface SysDictTypeRow {
  id: number
  name: string
  type: string
  status: string
  remark?: string | null
}

export interface SysDictDataRow {
  id: number
  sort: number
  label: string
  value: string
  dict_type: string
  status: string
  remark?: string | null
}

export interface DictTypeForm {
  name: string
  type: string
  status: string
  remark?: string
}

export interface DictDataForm {
  sort: number
  label: string
  value: string
  dictType: string
  status: string
  remark?: string
}

const mapDictType = (r: SysDictTypeRow): SysDictTypeRow => ({ ...r, id: num(r.id) })
const mapDictData = (r: SysDictDataRow): SysDictDataRow => ({ ...r, id: num(r.id), sort: num(r.sort) })

export const listDictTypes = async (params: { page: number; pageSize: number; keyword?: string }) => {
  const data = await unwrap<PageResult<SysDictTypeRow>>(http.get('/system/dict/types', { params }))
  return { ...data, list: data.list.map(mapDictType) }
}

export const createDictType = (data: DictTypeForm) => unwrap<{ id: number }>(http.post('/system/dict/types', data))

export const updateDictType = (id: number, data: DictTypeForm) => unwrap<null>(http.put(`/system/dict/types/${id}`, data))

export const deleteDictTypes = (ids: number[]) =>
  unwrap<null>(http.delete('/system/dict/types', { params: { ids: ids.join(',') } }))

export const listDictData = async (params: { page: number; pageSize: number; dictType?: string; keyword?: string }) => {
  const data = await unwrap<PageResult<SysDictDataRow>>(http.get('/system/dict/data', { params }))
  return { ...data, list: data.list.map(mapDictData) }
}

export const dictOptions = (dictType: string) =>
  unwrap<Array<{ label: string; value: string }>>(http.get('/system/dict/data/options', { params: { dictType } }))

export const createDictData = (data: DictDataForm) => unwrap<{ id: number }>(http.post('/system/dict/data', data))

export const updateDictData = (id: number, data: DictDataForm) => unwrap<null>(http.put(`/system/dict/data/${id}`, data))

export const deleteDictData = (ids: number[]) =>
  unwrap<null>(http.delete('/system/dict/data', { params: { ids: ids.join(',') } }))

// ==================== 系统设置（分组化配置：功能开关 / 邮件设置） ====================

export interface SettingItem {
  key: string
  label: string
  type: 'switch' | 'text' | 'number' | 'password'
  value: string
  placeholder?: string
  hint?: string
}

export interface SettingSection {
  title: string
  description: string
  items: SettingItem[]
}

export interface SettingTab {
  key: string
  title: string
  sections: SettingSection[]
}

export const fetchSettings = () => unwrap<SettingTab[]>(http.get('/system/settings'))

export const saveSettings = (values: Record<string, string>) =>
  unwrap<null>(http.put('/system/settings', { values }))

export const sendTestMail = (to: string) =>
  unwrap<{ message: string }>(http.post('/system/settings/test-mail', { to }))

// ==================== 数据备份 ====================

export interface BackupRecordRow {
  id: number
  recordKey: string
  status: string
  fileName: string
  sizeBytes: number
  parts: number
  expireAt?: string | null
  triggerType: string
  startedAt: string
  finishedAt?: string | null
  durationMs: number
  error?: string | null
  createdAt: string
}

export const listBackups = async (params: { page: number; pageSize: number }) => {
  const data = await unwrap<PageResult<BackupRecordRow>>(http.get('/system/backup/records', { params }))
  return { ...data, list: data.list.map((r) => ({ ...r, id: num(r.id), sizeBytes: num(r.sizeBytes), parts: num(r.parts), durationMs: num(r.durationMs) })) }
}

export const createBackup = (expireDays: number) =>
  unwrap<{ id: number; status: string }>(http.post('/system/backup/records', { expireDays }))

export const restoreBackup = (id: number) => unwrap<null>(http.post(`/system/backup/records/${id}/restore`))

export const deleteBackup = (id: number) => unwrap<null>(http.delete(`/system/backup/records/${id}`))

export const testS3Connection = () => unwrap<{ message: string }>(http.post('/system/backup/test-s3'))

/** 下载备份（blob，复用 axios 的 token 注入与 401 处理） */
export async function downloadBackup(id: number, fileName: string) {
  const res = await http.get(`/system/backup/records/${id}/download`, { responseType: 'blob' })
  const url = URL.createObjectURL(res.data as Blob)
  const a = document.createElement('a')
  a.href = url
  a.download = fileName
  document.body.appendChild(a)
  a.click()
  a.remove()
  URL.revokeObjectURL(url)
}

// ==================== 操作日志 ====================

export interface SysOperLogRow {
  id: number
  title: string
  business_type: string
  method: string
  path: string
  operator_id?: number | null
  operator_name: string
  ip: string
  status_code: number
  duration_ms: number
  params?: string | null
  created_at: string
}

const mapOperLog = (r: SysOperLogRow): SysOperLogRow => ({
  ...r,
  id: num(r.id),
  operator_id: r.operator_id == null ? null : num(r.operator_id),
  status_code: num(r.status_code),
  duration_ms: num(r.duration_ms),
})

export const listOperLogs = async (params: {
  page: number
  pageSize: number
  title?: string
  operator?: string
  status?: string
}) => {
  const data = await unwrap<PageResult<SysOperLogRow>>(http.get('/system/operlogs', { params }))
  return { ...data, list: data.list.map(mapOperLog) }
}
