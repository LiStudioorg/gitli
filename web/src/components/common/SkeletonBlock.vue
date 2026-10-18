<script setup lang="ts">
withDefaults(defineProps<{ lines?: number; height?: string }>(), { lines: 3, height: '1rem' })
</script>

<template>
  <div class="flex w-full flex-col gap-2" aria-hidden="true">
    <div v-for="i in lines" :key="i" class="w-full rounded-md bg-[linear-gradient(90deg,#eaeef2_25%,#f6f8fa_37%,#eaeef2_63%)] bg-[length:400%_100%] animate-pulse" :style="{ height }" :class="{ short: i === lines && lines > 1 }" />
  </div>
</template>
