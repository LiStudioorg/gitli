<script setup lang="ts">
withDefaults(
  defineProps<{
    icon?: string
    title: string
    description?: string
    actionLabel?: string
    showAction?: boolean
  }>(),
  { icon: '📭', description: '', actionLabel: '', showAction: true },
)

const emit = defineEmits<{ (e: 'action'): void }>()
</script>

<template>
  <div class="flex flex-col items-center justify-center gap-1 rounded-md border border-dashed border-[#d1d9e0] bg-white px-4 py-10 text-center dark:border-[#3d444d] dark:bg-[#0d1117]">
    <div class="mb-1 flex h-12 w-12 items-center justify-center rounded-full bg-[#ea580c]/10 text-xl" aria-hidden="true">{{ icon }}</div>
    <p class="text-sm font-semibold text-[#1f2328] dark:text-[#e6edf3]">{{ title }}</p>
    <p v-if="description" class="max-w-md text-sm text-[#59636e]">{{ description }}</p>
    <button v-if="showAction && actionLabel" class="inline-flex h-8 items-center rounded-md bg-[#2da44e] px-3 text-sm font-semibold text-white hover:bg-[#2c974b] disabled:opacity-50" type="button" @click="emit('action')">
      {{ actionLabel }}
    </button>
  </div>
</template>
