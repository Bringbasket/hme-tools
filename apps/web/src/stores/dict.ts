// 字典缓存（docs/05 §4：登录后按需加载，缓存复用）
import { defineStore } from 'pinia'
import { reactive } from 'vue'
import { dictOptions } from '@/api/system'

export interface DictOption {
  label: string
  value: string
}

export const useDictStore = defineStore('dict', () => {
  const cache = reactive<Record<string, DictOption[]>>({})

  async function load(dictType: string): Promise<DictOption[]> {
    if (!cache[dictType]) {
      cache[dictType] = await dictOptions(dictType)
    }
    return cache[dictType]
  }

  function reset() {
    Object.keys(cache).forEach((k) => delete cache[k])
  }

  return { cache, load, reset }
})
