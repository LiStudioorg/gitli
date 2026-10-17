<script setup lang="ts">
import { computed, ref } from 'vue'
import type { Repo } from '../../stores'

const props = defineProps<{ repos: Repo[]; username?: string }>()
const emit = defineEmits<{ (e: 'remove', repo: Repo): void }>()

const tab = ref<'all' | 'owned' | 'fork' | 'collaborated'>('all')
const query = ref('')

const tabs = [
  { key: 'all', label: '所有' },
  { key: 'owned', label: '自建' },
  { key: 'fork', label: '派生' },
  { key: 'collaborated', label: '协作' },
] as const

const filtered = computed(() => {
  const q = query.value.trim().toLowerCase()
  return props.repos.filter((r) => {
    if (tab.value === 'owned' && r.relation !== 'owned') return false
    if (tab.value === 'fork' && !r.is_fork) return false
    if (tab.value === 'collaborated' && r.relation !== 'collaborated') return false
    if (!q) return true
    return r.name.toLowerCase().includes(q) || (r.description ?? '').toLowerCase().includes(q)
  })
})

function ownerOf(r: Repo): string {
  return r.owner_name || props.username || ''
}

function repoLink(r: Repo): string {
  return `/${ownerOf(r)}/${r.name}`
}
</script>

<template>
  <section class="card">
    <header class="head">
      <h3>你的仓库</h3>
      <RouterLink to="/explore" class="all">全部</RouterLink>
    </header>
    <nav class="tabs">
      <button
        v-for="t in tabs"
        :key="t.key"
        type="button"
        :class="{ active: tab === t.key }"
        @click="tab = t.key"
      >
        {{ t.label }}
      </button>
    </nav>
    <input v-model="query" class="search" type="search" placeholder="过滤仓库..." aria-label="过滤仓库" />
    <ul v-if="filtered.length" class="list">
      <li v-for="r in filtered" :key="r.id">
        <span class="repo-icon" aria-hidden="true">{{ r.is_fork ? '⑂' : r.is_private ? '🔒' : '📦' }}</span>
        <RouterLink :to="repoLink(r)" class="name">{{ r.name }}</RouterLink>
        <button
          v-if="r.relation === 'owned'"
          type="button"
          class="remove"
          title="删除仓库"
          aria-label="删除仓库"
          @click="emit('remove', r)"
        >
          ×
        </button>
      </li>
    </ul>
    <p v-else class="empty">没有匹配的仓库</p>
  </section>
</template>

<style scoped>
.card {
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 1rem;
  background: var(--card);
}
.head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  margin-bottom: 0.6rem;
}
h3 {
  margin: 0;
  font-size: 0.95rem;
}
.all {
  font-size: 0.78rem;
  color: var(--primary);
  text-decoration: none;
}
.tabs {
  display: flex;
  gap: 0.3rem;
  margin-bottom: 0.6rem;
  flex-wrap: wrap;
}
.tabs button {
  font-size: 0.76rem;
  padding: 0.2rem 0.5rem;
  border-radius: 999px;
  border: 1px solid var(--border);
  background: transparent;
  color: var(--muted);
  cursor: pointer;
}
.tabs button.active {
  border-color: var(--primary);
  color: var(--primary);
  font-weight: 600;
}
.search {
  width: 100%;
  box-sizing: border-box;
  padding: 0.4rem 0.6rem;
  border-radius: 6px;
  border: 1px solid var(--border);
  background: transparent;
  color: var(--fg);
  margin-bottom: 0.6rem;
  font-size: 0.82rem;
}
.list {
  list-style: none;
  padding: 0;
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 0.15rem;
}
.list li {
  display: flex;
  align-items: center;
  gap: 0.45rem;
  padding: 0.3rem 0.35rem;
  border-radius: 6px;
}
.list li:hover {
  background: color-mix(in srgb, var(--border) 30%, transparent);
}
.repo-icon {
  font-size: 0.85rem;
  flex-shrink: 0;
}
.name {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 0.86rem;
  color: var(--fg);
  text-decoration: none;
}
.name:hover {
  color: var(--primary);
}
.remove {
  border: none;
  background: transparent;
  color: var(--muted);
  cursor: pointer;
  font-size: 1rem;
  line-height: 1;
  padding: 0 0.25rem;
  border-radius: 4px;
}
.remove:hover {
  color: #d1242f;
  background: color-mix(in srgb, #d1242f 12%, transparent);
}
.empty {
  margin: 0.5rem 0 0;
  font-size: 0.8rem;
  color: var(--muted);
}
</style>
