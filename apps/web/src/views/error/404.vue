<script setup lang="ts">
import { useRouter } from 'vue-router'

const router = useRouter()
</script>

<template>
  <div class="flex min-h-[60vh] flex-col items-center justify-center gap-4">
    <p class="text-7xl font-black text-slate-200 dark:text-slate-700">404</p>
    <p class="text-sm text-slate-500">页面不存在或已被移除</p>
    <button class="btn-primary h-9 px-4" @click="router.push('/')">返回首页</button>
  </div>
</template>
