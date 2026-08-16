// 路由：静态路由 + 全局守卫 + 动态路由（docs/05 §5）
import { createRouter, createWebHistory } from 'vue-router'
import { usePermissionStore } from '@/stores/permission'
import { useUserStore } from '@/stores/user'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('@/views/login/index.vue'),
      meta: { title: '登录' },
    },
    {
      path: '/',
      name: 'root',
      component: () => import('@/layout/index.vue'),
      redirect: '/dashboard',
      children: [],
    },
    {
      path: '/:pathMatch(.*)*',
      name: 'not-found',
      component: () => import('@/views/error/404.vue'),
      meta: { title: '404' },
    },
  ],
})

const whiteList = ['/login']

router.beforeEach(async (to) => {
  const userStore = useUserStore()
  const permissionStore = usePermissionStore()

  if (whiteList.includes(to.path)) {
    if (userStore.token) {
      return { path: '/' }
    }
    return true
  }

  if (!userStore.token) {
    return { path: '/login', query: { redirect: to.fullPath } }
  }

  if (!permissionStore.loaded) {
    try {
      await userStore.fetchInfo()
      await permissionStore.loadRoutes()
      // 动态路由挂载后重试解析。注意：不能直接 {...to}——to 可能已匹配 404 携带 name，
      // 必须用 path/query/hash 重建导航位置（vue-router 4 动态路由约定）
      return { path: to.path, query: to.query, hash: to.hash, replace: true }
    } catch {
      userStore.reset()
      permissionStore.reset()
      return { path: '/login', query: { redirect: to.fullPath } }
    }
  }
  return true
})

router.afterEach((to) => {
  const title = to.meta.title as string | undefined
  document.title = title ? `${title} · GoKeep 管理后台` : 'GoKeep 管理后台'
})

export default router
