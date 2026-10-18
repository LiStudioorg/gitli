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
  <div class="empty-state">
    <div class="icon" aria-hidden="true">{{ icon }}</div>
    <p class="title">{{ title }}</p>
    <p v-if="description" class="desc">{{ description }}</p>
    <button v-if="showAction && actionLabel" class="ui-btn ui-btn-primary" type="button" @click="emit('action')">
      {{ actionLabel }}
    </button>
  </div>
</template>

<style scoped>
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
  gap: 0.4rem;
  padding: 3rem 1.5rem;
  border: 1px dashed var(--border);
  border-radius: var(--radius-md);
  background: color-mix(in srgb, var(--card) 70%, transparent);
}
.icon {
  width: 56px;
  height: 56px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 1.6rem;
  background: color-mix(in srgb, var(--primary) 12%, transparent);
  margin-bottom: 0.4rem;
}
.title {
  margin: 0;
  font-weight: 600;
  font-size: 1rem;
}
.desc {
  margin: 0 0 0.7rem;
  color: var(--muted);
  font-size: 0.86rem;
  max-width: 380px;
}
</style>
