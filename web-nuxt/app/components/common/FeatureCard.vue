<script setup lang="ts">
defineProps<{
  title: string
  description: string
  icon?: string
}>()
</script>

<template>
  <div class="feature-card">
    <div class="icon">{{ icon ?? '★' }}</div>
    <h3>{{ title }}</h3>
    <p>{{ description }}</p>
  </div>
</template>

<style scoped>
.feature-card {
  padding: 1.4rem;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--card);
  transition: border-color 0.15s, box-shadow 0.15s;
}
.feature-card:hover {
  border-color: var(--primary);
  box-shadow: 0 2px 10px rgba(59, 130, 246, 0.15);
}
.icon {
  font-size: 1.4rem;
  margin-bottom: 0.5rem;
}
h3 {
  margin: 0 0 0.4rem;
  font-size: 1.05rem;
}
p {
  margin: 0;
  color: var(--muted);
  font-size: 0.9rem;
  line-height: 1.5;
}
</style>
