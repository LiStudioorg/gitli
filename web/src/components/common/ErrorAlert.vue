<script setup lang="ts">
/**
 * GitHub Primer 风格的错误提示。只渲染在主内容区顶部，
 * 绝不替换 Layout（TopNav / Sidebar 始终保留）。
 */
withDefaults(
  defineProps<{
    message: string
    title?: string
    dismissible?: boolean
  }>(),
  { title: '出错了', dismissible: true },
)

const emit = defineEmits<{ (e: 'dismiss'): void }>()
</script>

<template>
  <div
    v-if="message"
    role="alert"
    class="mb-4 flex items-start gap-2 rounded-md border border-[#ff818266] bg-[#ffebe9] px-4 py-3 text-sm text-[#1f2328]"
  >
    <svg class="mt-0.5 h-4 w-4 shrink-0 text-[#cf222e]" viewBox="0 0 16 16" fill="currentColor" aria-hidden="true">
      <path d="M8 1.5a6.5 6.5 0 1 0 0 13 6.5 6.5 0 0 0 0-13ZM0 8a8 8 0 1 1 16 0A8 8 0 0 1 0 8Zm9 3a1 1 0 1 1-2 0 1 1 0 0 1 2 0ZM7.25 4.5h1.5v5h-1.5v-5Z" />
    </svg>
    <div class="min-w-0 flex-1">
      <p class="font-semibold leading-5">{{ title }}</p>
      <p class="break-words leading-5 text-[#656d76]">{{ message }}</p>
    </div>
    <button
      v-if="dismissible"
      type="button"
      class="shrink-0 rounded p-1 text-[#656d76] hover:bg-[#ffd7d5]"
      aria-label="关闭提示"
      @click="emit('dismiss')"
    >
      <svg class="h-3.5 w-3.5" viewBox="0 0 16 16" fill="currentColor" aria-hidden="true">
        <path d="M3.72 3.72a.75.75 0 0 1 1.06 0L8 6.94l3.22-3.22a.75.75 0 1 1 1.06 1.06L9.06 8l3.22 3.22a.75.75 0 1 1-1.06 1.06L8 9.06l-3.22 3.22a.75.75 0 0 1-1.06-1.06L6.94 8 3.72 4.78a.75.75 0 0 1 0-1.06Z" />
      </svg>
    </button>
  </div>
</template>
