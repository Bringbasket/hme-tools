// 按钮级权限指令（docs/05 §5：后端接口为准，指令仅为体验优化）
import type { Directive } from 'vue'
import { useUserStore } from '@/stores/user'

export const permission: Directive<HTMLElement, string> = {
  mounted(el, binding) {
    const store = useUserStore()
    if (!store.hasPerm(binding.value)) {
      el.parentNode?.removeChild(el)
    }
  },
}
