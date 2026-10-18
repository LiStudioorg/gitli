<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{ name: string; type: string }>()

/** 按扩展名给出与主流代码托管一致的配色。 */
const palette: Record<string, string> = {
  go: 'text-[#00add8]',
  js: 'text-[#f1e05a]',
  ts: 'text-[#3178c6]',
  vue: 'text-[#42b883]',
  py: 'text-[#3572a5]',
  rs: 'text-[#dea584]',
  java: 'text-[#b07219]',
  rb: 'text-[#701516]',
  php: 'text-[#4f5d95]',
  css: 'text-[#563d7c]',
  scss: 'text-[#c6538c]',
  html: 'text-[#e34c26]',
  md: 'text-[#083fa1]',
  json: 'text-[#cbcb41]',
  yml: 'text-[#cb171e]',
  yaml: 'text-[#cb171e]',
  toml: 'text-[#9c4221]',
  sh: 'text-[#89e051]',
  sql: 'text-[#e38c00]',
  dockerfile: 'text-[#384d54]',
}

const ext = computed(() => {
  const lower = props.name.toLowerCase()
  if (lower === 'dockerfile') return 'dockerfile'
  const parts = lower.split('.')
  return parts.length > 1 ? parts[parts.length - 1] : ''
})

const color = computed(() => palette[ext.value] ?? 'text-[#59636e]')
const isDir = computed(() => props.type === 'tree')
const isParent = computed(() => props.type === 'parent')
</script>

<template>
  <span class="inline-flex h-4 w-4 shrink-0 items-center justify-center" aria-hidden="true">
    <!-- 目录 -->
    <svg v-if="isDir" class="h-4 w-4 text-[#54aeff]" viewBox="0 0 16 16" fill="currentColor">
      <path d="M1.75 1A1.75 1.75 0 0 0 0 2.75v10.5C0 14.216.784 15 1.75 15h12.5A1.75 1.75 0 0 0 16 13.25v-8.5A1.75 1.75 0 0 0 14.25 3H7.5a.25.25 0 0 1-.2-.1l-.9-1.2C6.07 1.26 5.55 1 5 1Z" />
    </svg>
    <!-- 返回上级 -->
    <svg v-else-if="isParent" class="h-4 w-4 text-[#59636e]" viewBox="0 0 16 16" fill="currentColor">
      <path d="M9.78 12.78a.75.75 0 0 1-1.06 0L4.47 8.53a.75.75 0 0 1 0-1.06l4.25-4.25a.751.751 0 0 1 1.042.018.751.751 0 0 1 .018 1.042L6.06 8l3.72 3.72a.75.75 0 0 1 0 1.06Z" />
    </svg>
    <!-- 文件 -->
    <svg v-else class="h-4 w-4" :class="color" viewBox="0 0 16 16" fill="currentColor">
      <path d="M2 1.75C2 .784 2.784 0 3.75 0h6.586c.464 0 .909.184 1.237.513l2.914 2.914c.329.328.513.773.513 1.237v9.586A1.75 1.75 0 0 1 13.25 16h-9.5A1.75 1.75 0 0 1 2 14.25Zm1.75-.25a.25.25 0 0 0-.25.25v12.5c0 .138.112.25.25.25h9.5a.25.25 0 0 0 .25-.25V6h-2.75A1.75 1.75 0 0 1 9 4.25V1.5Z" />
    </svg>
  </span>
</template>
