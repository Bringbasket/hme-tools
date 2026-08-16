// 登录态与用户信息（docs/05 §4：只放跨页面共享状态）
import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import * as authApi from '@/api/auth'
import type { LoginParams, UserInfo } from '@/api/auth'
import { clearAuth, getToken, setToken } from '@/utils/auth'

export const useUserStore = defineStore('user', () => {
  const token = ref(getToken())
  const userInfo = ref<UserInfo | null>(null)
  const roles = ref<string[]>([])
  const permissions = ref<string[]>([])
  const isAdmin = ref(false)

  const nickname = computed(() => userInfo.value?.nickname || userInfo.value?.username || '')

  async function login(params: LoginParams) {
    const { token: t } = await authApi.doLogin(params)
    token.value = t
    setToken(t)
    await fetchInfo()
  }

  async function register(params: authApi.RegisterParams) {
    const { token: t } = await authApi.doRegister(params)
    token.value = t
    setToken(t)
    await fetchInfo()
  }

  async function fetchInfo() {
    const data = await authApi.fetchUserInfo()
    userInfo.value = data.user
    roles.value = data.roles
    permissions.value = data.permissions
    isAdmin.value = data.isAdmin
  }

  async function logout() {
    try {
      await authApi.doLogout()
    } catch {
      // 会话已失效时后端可能返回 401，忽略
    }
    reset()
  }

  function reset() {
    token.value = ''
    userInfo.value = null
    roles.value = []
    permissions.value = []
    isAdmin.value = false
    clearAuth()
  }

  function hasPerm(perm: string): boolean {
    return isAdmin.value || permissions.value.includes(perm)
  }

  return { token, userInfo, roles, permissions, isAdmin, nickname, login, register, fetchInfo, logout, reset, hasPerm }
})
