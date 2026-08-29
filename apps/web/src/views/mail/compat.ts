import { isAxiosError } from 'axios'
import { reactive } from 'vue'

export type ToastTone = 'success' | 'error' | 'warning' | 'info'

export interface ToastItem {
  id: number
  message: string
  tone: ToastTone
}

export const toastState = reactive({ items: [] as ToastItem[] })
let nextToastID = 1

export function dismissToast(id: number) {
  toastState.items = toastState.items.filter((item) => item.id !== id)
}

export function showToast(message: string, tone: ToastTone = 'success', duration = 3200) {
  const item = { id: nextToastID++, message, tone }
  toastState.items.push(item)
  if (duration > 0) window.setTimeout(() => dismissToast(item.id), duration)
  return item.id
}

export function errorMessage(error: unknown): string {
  if (isAxiosError(error)) {
    const payload = error.response?.data as { msg?: string; error?: { message?: string } } | undefined
    return payload?.error?.message || payload?.msg || error.message || '请求失败，请稍后重试'
  }
  if (error instanceof Error) return error.message
  if (typeof error === 'string') return error
  return '请求失败，请稍后重试'
}
