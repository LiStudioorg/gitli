<script setup lang="ts">
import type { Activity } from '../../stores'
import { useAuthStore } from '../../stores'

defineProps<{ activity: Activity }>()
const auth = useAuthStore()
</script>

<template>
  <article class="activity-item">
    <span class="avatar">{{ auth.user?.username?.[0]?.toUpperCase() ?? '?' }}</span>
    <div class="body">
      <p class="line">
        <strong>{{ auth.user?.username }}</strong>
        {{ activity.verb }}
        <span class="object">{{ activity.object_type }} #{{ activity.object_id }}</span>
      </p>
      <time class="time">{{ activity.created_at }}</time>
    </div>
  </article>
</template>

<style scoped>
.activity-item {
  display: flex;
  gap: 0.75rem;
  padding: 0.9rem 1rem;
  border: 1px solid var(--border);
  border-radius: 10px;
  background: var(--card);
  transition: border-color 0.15s;
}
.activity-item:hover {
  border-color: var(--primary);
}
.avatar {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 34px;
  height: 34px;
  border-radius: 50%;
  background: var(--primary);
  color: #fff;
  font-weight: 600;
  flex-shrink: 0;
}
.body {
  flex: 1;
}
.line {
  margin: 0;
  font-size: 0.95rem;
}
.object {
  color: var(--primary);
}
.time {
  color: var(--muted);
  font-size: 0.82rem;
}
</style>
