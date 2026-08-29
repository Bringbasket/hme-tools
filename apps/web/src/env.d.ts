/// <reference types="vite/client" />

interface ImportMetaEnv {
  /** API 前缀，默认 /api/v1 */
  readonly VITE_APP_BASE_API: string
  /** 构建时写入的发布版本与提交标识。 */
  readonly VITE_APP_VERSION?: string
  readonly VITE_APP_REVISION?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}

declare module '*.vue' {
  import type { DefineComponent } from 'vue'
  const component: DefineComponent<Record<string, unknown>, Record<string, unknown>, unknown>
  export default component
}
