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
  <section class="rounded-md border border-[#d1d9e0] bg-white p-4 dark:border-[#3d444d] dark:bg-[#0d1117]">
    <header class="mb-3 flex items-center justify-between">
      <h3 class="text-base font-semibold text-[#1f2328] dark:text-[#e6edf3]">你的仓库</h3>
      <RouterLink to="/explore" class="text-xs text-[#0969da] hover:underline">全部</RouterLink>
    </header>
    <nav class="mb-2 flex flex-wrap gap-1">
      <button
        v-for="t in tabs"
        :key="t.key"
        type="button"
        class="rounded-full border px-2 py-0.5 text-xs"
        :class="tab === t.key
          ? 'border-[#ea580c] font-semibold text-[#ea580c]'
          : 'border-[#d1d9e0] text-[#59636e] hover:bg-[#f6f8fa] dark:border-[#3d444d] dark:hover:bg-[#161b22]'"
        @click="tab = t.key"
      >
        {{ t.label }}
      </button>
    </nav>
    <input
      v-model="query"
      class="mb-2 h-8 w-full rounded-md border border-[#d1d9e0] bg-[#f6f8fa] px-2.5 text-sm text-[#1f2328] outline-none placeholder:text-[#59636e] focus:border-[#0969da] focus:bg-white focus:ring-2 focus:ring-[#0969da]/25 dark:border-[#3d444d] dark:bg-[#0d1117] dark:text-[#e6edf3]"
      type="search"
      placeholder="过滤仓库..."
      aria-label="过滤仓库"
    />
    <ul v-if="filtered.length" class="flex flex-col">
      <li
        v-for="r in filtered"
        :key="r.id"
        class="flex items-center gap-1.5 rounded-md px-1.5 py-1 hover:bg-[#f6f8fa] dark:hover:bg-[#161b22]"
      >
        <span class="shrink-0 text-xs" aria-hidden="true">{{ r.is_fork ? '⑂' : r.is_private ? '🔒' : '📦' }}</span>
        <RouterLink :to="repoLink(r)" class="min-w-0 flex-1 truncate text-sm text-[#1f2328] hover:text-[#0969da] dark:text-[#e6edf3]">
          {{ r.name }}
        </RouterLink>
        <button
          v-if="r.relation === 'owned'"
          type="button"
          class="shrink-0 rounded px-1 text-[#59636e] hover:bg-[#ffebe9] hover:text-[#cf222e]"
          title="删除仓库"
          aria-label="删除仓库"
          @click="emit('remove', r)"
        >
          ×
        </button>
      </li>
    </ul>
    <p v-else class="py-2 text-xs text-[#59636e]">没有匹配的仓库</p>
  </section>
</template>
