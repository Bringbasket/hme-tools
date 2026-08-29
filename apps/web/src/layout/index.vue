<script setup lang="ts">
// 平台外壳：侧栏、顶栏与主内容区。
import { computed } from 'vue'
import { RouterView } from 'vue-router'
import Sidebar from './components/Sidebar.vue'
import Header from './components/Header.vue'
import { useAppStore } from '@/stores/app'

const appStore = useAppStore()
const contentClass = computed(() => (appStore.sidebarCollapsed ? 'lg:pl-16' : 'lg:pl-[240px]'))
</script>

<template>
  <div class="min-h-screen bg-surface-page font-sans text-slate-900 dark:text-slate-100">
    <Sidebar />
    <div class="transition-[padding] duration-200" :class="contentClass">
      <Header />
      <main class="min-h-screen px-4 pb-9 pt-[104px] sm:px-[26px]">
        <RouterView />
      </main>
    </div>
  </div>
</template>
