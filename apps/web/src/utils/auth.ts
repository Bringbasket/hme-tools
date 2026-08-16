// Token 本地存取（Web 端 localStorage；生产建议评估 httpOnly Cookie 方案，见 docs/04 §4）
const TOKEN_KEY = 'gokeep_token'

export function getToken(): string {
  return localStorage.getItem(TOKEN_KEY) || ''
}

export function setToken(token: string) {
  localStorage.setItem(TOKEN_KEY, token)
}

export function clearAuth() {
  localStorage.removeItem(TOKEN_KEY)
}
