<script setup lang="ts">
import { computed } from 'vue'
import type { Component } from 'vue'
import {
  AtSign,
  BookOpen,
  Calculator,
  Inbox,
  KeyRound,
  LayoutDashboard,
  ListTree,
  Mail,
  Menu,
  ScrollText,
  Settings,
  ShieldCheck,
  SlidersHorizontal,
  Terminal,
  Users,
  Wrench,
} from 'lucide-vue-next'

// 基础菜单图标映射；未登记的图标统一回退为 Menu。
const props = defineProps<{ name?: string | null; size?: number }>()

const icons: Record<string, Component> = {
  'layout-dashboard': LayoutDashboard,
  settings: Settings,
  users: Users,
  'shield-check': ShieldCheck,
  'list-tree': ListTree,
  'book-open': BookOpen,
  'sliders-horizontal': SlidersHorizontal,
  'settings-2': SlidersHorizontal,
  'scroll-text': ScrollText,
  mail: Mail,
  'at-sign': AtSign,
  inbox: Inbox,
  terminal: Terminal,
  'key-round': KeyRound,
  calculator: Calculator,
  wrench: Wrench,
}

const icon = computed(() => (props.name && icons[props.name] ? icons[props.name] : Menu))
</script>

<template>
  <component :is="icon" :size="size ?? 16" class="shrink-0" aria-hidden="true" />
</template>
