import { fileURLToPath, URL } from 'node:url'
import { writeFileSync } from 'node:fs'
import { defineConfig } from 'vite'
import type { Plugin } from 'vite'
import vue from '@vitejs/plugin-vue'

// 构建后补回 .gitkeep 占位文件：emptyOutDir 会清空 dist，
// 但 server/webui 的 go:embed 要求 dist 目录在全新检出时也至少有一个文件
function keepDistPlaceholder(): Plugin {
  return {
    name: 'keep-dist-placeholder',
    closeBundle() {
      writeFileSync(
        fileURLToPath(new URL('../../server/webui/dist/.gitkeep', import.meta.url)),
        '# placeholder: keeps dist dir tracked so go:embed compiles on fresh checkout; vite build fills real assets here\n',
      )
    },
  }
}

// 开发代理：/api 与 /healthz 转发到本地 Go 网关（server 默认 :8080）
// 生产构建产物输出到 server/webui/dist，由 Go embed 打包（见 02-总体架构与目录规范）
export default defineConfig({
  plugins: [vue(), keepDistPlaceholder()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  server: {
    host: true, // 监听所有网卡：手机等局域网设备可通过 http://<本机IP>:5173 访问（移动端适配测试）
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true,
      },
      '/healthz': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true,
      },
      '/readyz': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: '../../server/webui/dist',
    emptyOutDir: true,
  },
})
