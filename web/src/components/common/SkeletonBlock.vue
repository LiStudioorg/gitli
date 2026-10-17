<script setup lang="ts">
withDefaults(defineProps<{ lines?: number; height?: string }>(), { lines: 3, height: '1rem' })
</script>

<template>
  <div class="skeleton" aria-hidden="true">
    <div v-for="i in lines" :key="i" class="line" :style="{ height }" :class="{ short: i === lines && lines > 1 }" />
  </div>
</template>

<style scoped>
.skeleton {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  width: 100%;
}
.line {
  border-radius: 6px;
  background: linear-gradient(
    90deg,
    color-mix(in srgb, var(--border) 55%, transparent) 25%,
    color-mix(in srgb, var(--border) 25%, transparent) 37%,
    color-mix(in srgb, var(--border) 55%, transparent) 63%
  );
  background-size: 400% 100%;
  animation: shimmer 1.3s ease infinite;
}
.line.short {
  width: 62%;
}
@keyframes shimmer {
  0% {
    background-position: 100% 50%;
  }
  100% {
    background-position: 0 50%;
  }
}
@media (prefers-reduced-motion: reduce) {
  .line {
    animation: none;
  }
}
</style>
