<script setup lang="ts">
defineProps<{
  title: string
  description: string
  icon?: string
}>()
</script>

<template>
  <div class="feature-card">
    <div class="mb-1 flex h-12 w-12 items-center justify-center rounded-full bg-[#ea580c]/10 text-xl">{{ icon ?? '★' }}</div>
    <h3>{{ title }}</h3>
    <p>{{ description }}</p>
  </div>
</template>
