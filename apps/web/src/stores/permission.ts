// 动态路由与侧栏菜单（docs/05 §5：路由表由后端菜单接口下发）
import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { RouteRecordRaw } from 'vue-router'
import router from '@/router'
import { fetchRouters } from '@/api/auth'
import type { MenuNode } from '@/api/auth'

// 页面组件映射：component 字符串（如 system/user/index）→ 懒加载组件
const viewModules = import.meta.glob('/src/views/**/*.vue')

function resolveComponent(component: string) {
  const key = `/src/views/${component}.vue`
  const loader = viewModules[key]
  if (!loader) {
    console.warn(`[permission] 未找到页面组件: ${key}`)
    return undefined
  }
  return loader
}

function isExternal(path: string) {
  return /^(https?:|mailto:|tel:)/.test(path)
}

export const usePermissionStore = defineStore('permission', () => {
  const loaded = ref(false)
  const menus = ref<MenuNode[]>([])
  const flatPerms = ref<string[]>([])

  function collectPerms(nodes: MenuNode[], out: string[]) {
    for (const n of nodes) {
      if (n.perms) out.push(n.perms)
      if (n.children?.length) collectPerms(n.children, out)
    }
  }

  /** 依据菜单树挂载动态路由（挂在根 Layout 下，避免嵌套外壳） */
  function buildRoutes(tree: MenuNode[]): RouteRecordRaw[] {
    const routes: RouteRecordRaw[] = []
    const parentView = () => import('@/views/parent/index.vue')

    const walk = (nodes: MenuNode[]) => {
      for (const node of nodes) {
        if (node.menuType === 'F') continue
        if (isExternal(node.path)) continue
        const childRoutes = node.children?.filter((c) => !isExternal(c.path)) ?? []
        if (node.menuType === 'M') {
          // 目录：包一层 RouterView 透传组件
          routes.push({
            path: node.path,
            name: `menu-${node.id}`,
            component: parentView,
            meta: { title: node.title, icon: node.icon, hidden: node.hidden, perms: node.perms },
            children: childRoutes.map((c) => buildLeaf(c)),
          })
        } else {
          routes.push(buildLeaf(node))
        }
      }
    }

    const buildLeaf = (node: MenuNode): RouteRecordRaw => {
      const comp = resolveComponent(node.component)
      return {
        path: node.path,
        name: `menu-${node.id}`,
        component: comp ?? (() => import('@/views/error/404.vue')),
        meta: { title: node.title, icon: node.icon, hidden: node.hidden, perms: node.perms },
      }
    }

    walk(tree)
    return routes
  }

  /** 登录后调用一次：拉菜单 → 挂路由 → 记侧栏数据 */
  async function loadRoutes() {
    if (loaded.value) return
    const tree = await fetchRouters()
    menus.value = tree
    const perms: string[] = []
    collectPerms(tree, perms)
    flatPerms.value = perms

    const rootRoute = router.getRoutes().find((r) => r.path === '/')
    if (rootRoute) {
      // 清掉旧动态路由，重建
      router.getRoutes().forEach((r) => {
        if (r.name && String(r.name).startsWith('menu-')) router.removeRoute(r.name)
      })
    }
    const dynamic = buildRoutes(tree)
    // addRoute 第一参数是父路由 NAME（根路由 name: 'root'），不是路径
    dynamic.forEach((r) => router.addRoute('root', r))
    loaded.value = true
  }

  function reset() {
    loaded.value = false
    menus.value = []
    flatPerms.value = []
    router.getRoutes().forEach((r) => {
      if (r.name && String(r.name).startsWith('menu-')) router.removeRoute(r.name)
    })
  }

  return { loaded, menus, flatPerms, loadRoutes, reset }
})
