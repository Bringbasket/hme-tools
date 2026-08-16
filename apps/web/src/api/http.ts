import axios from 'axios'
import { clearAuth, getToken } from '@/utils/auth'

/**
 * 统一 Axios 实例（docs/05-前端开发规范 §6）
 * - baseURL: /api/v1（Vite 代理到 Go 网关）
 * - 请求拦截：附加 Bearer Token
 * - 响应拦截：HTTP 401 → 清空会话并跳登录（业务失败走统一响应包 code 字段）
 */
export const http = axios.create({
  baseURL: import.meta.env.VITE_APP_BASE_API || '/api/v1',
  timeout: 15000,
})

http.interceptors.request.use((config) => {
  const token = getToken()
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

http.interceptors.response.use(
  (res) => res,
  (error) => {
    if (error.response?.status === 401) {
      clearAuth()
      if (!window.location.pathname.startsWith('/login')) {
        const redirect = encodeURIComponent(window.location.pathname + window.location.search)
        window.location.href = `/login?redirect=${redirect}`
      }
    }
    return Promise.reject(error)
  },
)

/** 后端统一响应包（docs/04-API接口规范 §2） */
export interface ApiResponse<T = unknown> {
  code: number
  msg: string
  data: T
}

/** 分页响应（docs/04 §3） */
export interface PageResult<T> {
  list: T[]
  total: number
  page: number
  pageSize: number
}

/** 解包响应：code=200 返回 data，否则抛带 msg 的错误 */
export async function unwrap<T>(promise: Promise<{ data: ApiResponse<T> }>): Promise<T> {
  const { data: body } = await promise
  if (body.code !== 200) {
    throw new Error(body.msg || `请求失败（${body.code}）`)
  }
  return body.data
}
