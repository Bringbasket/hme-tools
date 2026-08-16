// 认证接口（docs/04 §4）
import { http, unwrap } from './http'

export interface CaptchaData {
  captchaEnabled: boolean
  img: string
  uuid: string
  expr: string
}

export interface LoginParams {
  username: string
  password: string
  captchaUuid?: string
  captchaCode?: string
}

export interface UserInfo {
  userId: number
  username: string
  nickname: string
  phone?: string
  email?: string
  avatar?: string
}

export interface GetInfoData {
  user: UserInfo
  roles: string[]
  permissions: string[]
  isAdmin: boolean
}

/** 菜单路由节点（docs/05 §5 动态路由来源） */
export interface MenuNode {
  id: number
  parentId: number
  title: string
  path: string
  component: string
  menuType: string
  icon?: string
  hidden: boolean
  perms?: string
  children?: MenuNode[]
}

export const fetchCaptcha = () => unwrap<CaptchaData>(http.get('/auth/captcha'))

export const doLogin = (params: LoginParams) => unwrap<{ token: string }>(http.post('/auth/login', params))

export const doLogout = () => unwrap<null>(http.post('/auth/logout'))

export const fetchUserInfo = () => unwrap<GetInfoData>(http.get('/auth/getInfo'))

export const fetchRouters = () => unwrap<MenuNode[]>(http.get('/auth/routers'))

// ==================== 注册（邮箱 + 可选验证码） ====================

export interface RegisterConfigData {
  registerEnabled: boolean
  emailCodeEnabled: boolean
}

export interface RegisterParams {
  email: string
  password: string
  nickname?: string
  code?: string
}

export const fetchRegisterConfig = () => unwrap<RegisterConfigData>(http.get('/auth/register/config'))

export const sendRegisterEmailCode = (email: string) =>
  unwrap<{ message: string; devCode?: string }>(http.post('/auth/register/email-code', { email }))

export const doRegister = (params: RegisterParams) =>
  unwrap<{ token: string; userId: number; username: string; nickname: string }>(http.post('/auth/register', params))
