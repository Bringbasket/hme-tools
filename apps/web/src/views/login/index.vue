<script setup lang="ts">
// 登录页：左右分栏、登录/注册切换和图形验证码。
import { onMounted, onUnmounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { RefreshCw } from 'lucide-vue-next'
import { fetchCaptcha, fetchRegisterConfig, sendRegisterEmailCode } from '@/api/auth'
import type { CaptchaData, RegisterConfigData } from '@/api/auth'
import { useUserStore } from '@/stores/user'

const router = useRouter()
const route = useRoute()
const userStore = useUserStore()

const activeTab = ref<'login' | 'register' | 'reset'>('login')
const captcha = reactive<CaptchaData>({ captchaEnabled: false, img: '', uuid: '', expr: '' })
const captchaLoading = ref(false)
const captchaReady = ref(false)

const form = reactive({ username: '', password: '', captchaCode: '' })
const submitting = ref(false)
const errorMsg = ref('')

const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

// ==================== 注册 ====================
const registerForm = reactive({ email: '', password: '', confirmPassword: '', nickname: '', code: '' })
const registerConfig = reactive<RegisterConfigData>({ registerEnabled: false, emailCodeEnabled: false })
const registerConfigLoaded = ref(false)
const registerSubmitting = ref(false)
const registerError = ref('')
const codeSending = ref(false)
const countdown = ref(0)
const devCodeHint = ref('')
let countdownTimer: ReturnType<typeof setInterval> | null = null

async function loadRegisterConfig() {
  try {
    Object.assign(registerConfig, await fetchRegisterConfig())
  } catch {
    registerConfig.registerEnabled = false
  } finally {
    registerConfigLoaded.value = true
  }
}

function onTabChange(tab: 'login' | 'register' | 'reset') {
  activeTab.value = tab
  if (tab === 'register' && !registerConfigLoaded.value) {
    void loadRegisterConfig()
  }
}

function startCountdown() {
  countdown.value = 60
  countdownTimer = setInterval(() => {
    countdown.value -= 1
    if (countdown.value <= 0 && countdownTimer) {
      clearInterval(countdownTimer)
      countdownTimer = null
    }
  }, 1000)
}

async function handleSendCode() {
  if (!emailRegex.test(registerForm.email)) {
    registerError.value = '请输入正确的邮箱地址'
    return
  }
  codeSending.value = true
  registerError.value = ''
  try {
    const res = await sendRegisterEmailCode(registerForm.email)
    devCodeHint.value = res.devCode ?? ''
    startCountdown()
  } catch (e) {
    registerError.value = e instanceof Error ? e.message : '验证码发送失败'
  } finally {
    codeSending.value = false
  }
}

async function handleRegister() {
  if (!emailRegex.test(registerForm.email)) {
    registerError.value = '请输入正确的邮箱地址'
    return
  }
  if (registerForm.password.length < 8) {
    registerError.value = '密码至少 8 位'
    return
  }
  if (registerForm.password !== registerForm.confirmPassword) {
    registerError.value = '两次输入的密码不一致'
    return
  }
  if (registerConfig.emailCodeEnabled && !registerForm.code) {
    registerError.value = '请输入邮箱验证码'
    return
  }
  registerSubmitting.value = true
  registerError.value = ''
  try {
    await userStore.register({
      email: registerForm.email,
      password: registerForm.password,
      nickname: registerForm.nickname || undefined,
      code: registerConfig.emailCodeEnabled ? registerForm.code : undefined,
    })
    const redirect = (route.query.redirect as string) || '/'
    void router.push(redirect)
  } catch (e) {
    registerError.value = e instanceof Error ? e.message : '注册失败'
  } finally {
    registerSubmitting.value = false
  }
}

async function refreshCaptcha() {
  captchaLoading.value = true
  try {
    Object.assign(captcha, await fetchCaptcha())
    form.captchaCode = ''
    captchaReady.value = true
  } catch {
    captchaReady.value = false
    errorMsg.value = '登录服务暂不可用，请稍后重试'
  } finally {
    captchaLoading.value = false
  }
}

async function handleLogin() {
  if (!captchaReady.value) {
    errorMsg.value = '登录服务暂不可用，请稍后重试'
    return
  }
  if (!form.username || !form.password) {
    errorMsg.value = '请输入用户名和密码'
    return
  }
  if (captcha.captchaEnabled && !form.captchaCode) {
    errorMsg.value = '请输入验证码'
    return
  }
  submitting.value = true
  errorMsg.value = ''
  try {
    await userStore.login({
      username: form.username,
      password: form.password,
      captchaUuid: captcha.captchaEnabled ? captcha.uuid : undefined,
      captchaCode: captcha.captchaEnabled ? form.captchaCode : undefined,
    })
    const redirect = (route.query.redirect as string) || '/'
    void router.push(redirect)
  } catch (e) {
    errorMsg.value = e instanceof Error ? e.message : '登录失败'
    if (captcha.captchaEnabled) void refreshCaptcha()
  } finally {
    submitting.value = false
  }
}

onMounted(refreshCaptcha)
onUnmounted(() => {
  if (countdownTimer) clearInterval(countdownTimer)
})
</script>

<template>
  <div class="relative min-h-screen bg-surface-page">
    <!-- 网格纹路 + 顶部渐变条 -->
    <div class="pointer-events-none absolute inset-0 bg-grid" />
    <div class="brand-gradient-bar pointer-events-none absolute inset-x-0 top-0 h-1" />
    <main class="relative z-10 mx-auto grid min-h-screen w-full max-w-6xl items-center gap-8 px-5 py-10 lg:grid-cols-[616px_440px] lg:px-8">
      <!-- 左栏品牌区（≥lg 显示） -->
      <section class="hidden lg:block">
        <div class="flex items-baseline gap-2">
          <p class="text-xs font-semibold uppercase text-brand-600">GoKeep</p>
          <p class="text-sm font-bold text-slate-900">后台管理系统</p>
        </div>
        <h1 class="mt-6 text-[48px] font-black leading-[48px] text-slate-950">欢迎使用管理控制台</h1>
        <p class="mt-5 max-w-xl text-base leading-8 text-slate-600">
          统一的身份认证、权限控制与系统配置入口。
        </p>
      </section>

      <!-- 右栏登录卡片 -->
      <section class="w-full animate-slide-up">
        <div class="card-glass p-7 sm:p-8">
          <div class="flex items-start justify-between">
            <div>
              <p class="text-xs font-semibold uppercase text-brand-600">账户中心</p>
              <h2 class="mt-3 text-2xl font-bold text-slate-900 dark:text-white">登录控制台</h2>
            </div>
            <span class="rounded-full border border-brand-100 bg-brand-50 px-3 py-1 text-xs text-brand-700">内部平台</span>
          </div>

          <!-- 分段标签栏 -->
          <div class="mb-6 mt-6 flex gap-1 rounded-lg bg-gray-100 p-1 dark:bg-slate-800">
            <button
              v-for="tab in (['login', 'register', 'reset'] as const)"
              :key="tab"
              class="flex-1 rounded-lg py-2 text-sm font-medium transition-colors"
              :class="activeTab === tab ? 'bg-white text-slate-900 shadow-soft-sm dark:bg-slate-700 dark:text-white' : 'text-slate-600 dark:text-slate-300'"
              @click="onTabChange(tab)"
            >
              {{ tab === 'login' ? '登录' : tab === 'register' ? '成员注册' : '找回密码' }}
            </button>
          </div>

          <!-- 登录表单 -->
          <form v-if="activeTab === 'login'" class="grid gap-4" @submit.prevent="handleLogin">
            <label class="grid gap-1.5 text-sm font-medium text-slate-700 dark:text-slate-200">
              用户名或邮箱
              <input v-model="form.username" class="input" type="text" placeholder="请输入用户名或邮箱" autocomplete="username" />
            </label>
            <label class="grid gap-1.5 text-sm font-medium text-slate-700 dark:text-slate-200">
              密码
              <input v-model="form.password" class="input" type="password" placeholder="至少 8 位" autocomplete="current-password" />
            </label>
            <div v-if="captcha.captchaEnabled" class="grid gap-1.5 text-sm font-medium text-slate-700 dark:text-slate-200">
              人机验证
              <div class="flex items-center gap-2">
                <img
                  v-if="captcha.img"
                  :src="captcha.img"
                  alt="验证码"
                  class="h-[37.6px] cursor-pointer rounded-md border border-[#dbe6ee]"
                  title="点击刷新"
                  @click="refreshCaptcha"
                />
                <input v-model="form.captchaCode" class="input" type="text" placeholder="计算结果" maxlength="4" />
                <button
                  type="button"
                  class="btn-secondary h-9 w-9 shrink-0"
                  :disabled="captchaLoading"
                  aria-label="刷新验证码"
                  @click="refreshCaptcha"
                >
                  <RefreshCw :size="14" :class="captchaLoading ? 'animate-spin' : ''" />
                </button>
              </div>
            </div>
            <p v-if="errorMsg" class="text-sm text-red-600" role="alert">{{ errorMsg }}</p>
            <button class="btn-primary mt-1 h-10 w-full" type="submit" :disabled="submitting || !captchaReady">
              {{ submitting ? '登录中…' : '登录' }}
            </button>
          </form>

          <!-- 注册表单（开关 sys.account.registerUser 控制） -->
          <form v-else-if="activeTab === 'register'" class="grid gap-4" @submit.prevent="handleRegister">
            <div
              v-if="registerConfigLoaded && !registerConfig.registerEnabled"
              class="rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-700 dark:border-amber-800/50 dark:bg-amber-900/20 dark:text-amber-400"
            >
              注册功能暂未开放，请联系管理员开通。
            </div>
            <template v-else>
              <label class="grid gap-1.5 text-sm font-medium text-slate-700 dark:text-slate-200">
                邮箱
                <input v-model="registerForm.email" class="input" type="email" placeholder="用于登录的邮箱地址" autocomplete="email" />
              </label>
              <label class="grid gap-1.5 text-sm font-medium text-slate-700 dark:text-slate-200">
                密码
                <input v-model="registerForm.password" class="input" type="password" placeholder="至少 8 位" autocomplete="new-password" />
              </label>
              <label class="grid gap-1.5 text-sm font-medium text-slate-700 dark:text-slate-200">
                确认密码
                <input v-model="registerForm.confirmPassword" class="input" type="password" placeholder="再次输入密码" autocomplete="new-password" />
              </label>
              <label class="grid gap-1.5 text-sm font-medium text-slate-700 dark:text-slate-200">
                昵称
                <input v-model="registerForm.nickname" class="input" type="text" placeholder="选填，默认取邮箱前缀" />
              </label>
              <div v-if="registerConfig.emailCodeEnabled" class="grid gap-1.5 text-sm font-medium text-slate-700 dark:text-slate-200">
                邮箱验证码
                <div class="flex items-center gap-2">
                  <input v-model="registerForm.code" class="input" type="text" placeholder="6 位验证码" maxlength="6" />
                  <button
                    type="button"
                    class="btn-secondary h-9 shrink-0 px-3"
                    :disabled="codeSending || countdown > 0"
                    @click="handleSendCode"
                  >
                    {{ countdown > 0 ? `${countdown}s 后重发` : codeSending ? '发送中…' : '发送验证码' }}
                  </button>
                </div>
                <p v-if="devCodeHint" class="text-xs text-slate-400">开发模式验证码：{{ devCodeHint }}（生产环境不会显示）</p>
              </div>
              <p v-if="registerError" class="text-sm text-red-600" role="alert">{{ registerError }}</p>
              <button class="btn-primary mt-1 h-10 w-full" type="submit" :disabled="registerSubmitting || (registerConfigLoaded && !registerConfig.registerEnabled)">
                {{ registerSubmitting ? '注册中…' : '注册并登录' }}
              </button>
            </template>
          </form>

          <!-- 占位面板 -->
          <div v-else class="grid gap-3 py-10 text-center text-sm text-slate-500">
            <p>「找回密码」功能开发中</p>
          </div>
        </div>
        <p class="mt-6 text-center text-xs text-slate-400">GoKeep · 管理后台</p>
      </section>
    </main>
  </div>
</template>
