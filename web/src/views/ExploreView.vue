<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import TopNav from '../components/layout/TopNav.vue'
import api from '../api/client'

interface ExploreRepo {
  id: number
  name: string
  owner: string
  owner_id: number
  description: string
  default_branch: string
  updated_at: string
  stars_count: number
}

const PAGE_SIZE = 20

const repos = ref<ExploreRepo[]>([])
const query = ref('')
const loading = ref(false)
const loadingMore = ref(false)
const error = ref('')
const hasMore = ref(true)

async function load(reset: boolean) {
  if (reset) {
    loading.value = true
    error.value = ''
    hasMore.value = true
  } else {
    if (!hasMore.value || loadingMore.value) return
    loadingMore.value = true
  }
  try {
    const offset = reset ? 0 : repos.value.length
    const { data } = await api.get<ExploreRepo[]>('/explore/repos', {
      params: { q: query.value || undefined, limit: PAGE_SIZE, offset },
    })
    const items = data ?? []
    repos.value = reset ? items : [...repos.value, ...items]
    hasMore.value = items.length === PAGE_SIZE
  } catch (e) {
    error.value = e instanceof Error ? e.message : '加载仓库失败'
  } finally {
    loading.value = false
    loadingMore.value = false
  }
}

async function search() {
  await load(true)
}

function loadMore() {
  void load(false)
}

function formatDate(v: string): string {
  const d = new Date(v)
  return Number.isNaN(d.getTime()) ? v : d.toLocaleString()
}

onMounted(() => load(true))
</script>

<template>
  <div class="explore">
    <TopNav variant="landing" />
    <main class="content">
      <header class="head">
        <h2>探索公开仓库</h2>
        <form class="search-box" @submit.prevent="search">
          <input v-model="query" type="search" placeholder="按仓库名称搜索..." />
          <button class="btn primary" type="submit">搜索</button>
        </form>
      </header>

      <p v-if="error" class="error">{{ error }}</p>

      <p v-if="loading" class="state">加载中...</p>

      <p v-else-if="!repos.length" class="state">没有找到公开仓库</p>

      <div v-else class="grid">
        <article v-for="r in repos" :key="r.id" class="card">
          <h3 class="card-title">
            <RouterLink :to="`/${r.owner}/${r.name}`">
              <span class="owner">{{ r.owner }}</span><span class="sep">/</span><span class="name">{{ r.name }}</span>
            </RouterLink>
          </h3>
          <p class="desc">{{ r.description || '暂无描述' }}</p>
          <div class="meta">
            <span class="branch" title="默认分支">⎇ {{ r.default_branch || 'main' }}</span>
            <span class="stars" title="Star 数">★ {{ r.stars_count }}</span>
            <span class="updated">更新于 {{ formatDate(r.updated_at) }}</span>
          </div>
        </article>
      </div>

      <div v-if="!loading && repos.length" class="more">
        <button v-if="hasMore" class="btn" :disabled="loadingMore" @click="loadMore">
          {{ loadingMore ? '加载中...' : '加载更多' }}
        </button>
        <p v-else class="state">已经到底啦</p>
      </div>
    </main>
  </div>
</template>

<style scoped>
.explore {
  min-height: 100vh;
}
.content {
  max-width: 1000px;
  margin: 0 auto;
  padding: 2rem 1.5rem;
}
.head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1.5rem;
  flex-wrap: wrap;
  margin-bottom: 1.5rem;
}
.head h2 {
  margin: 0;
}
.search-box {
  display: flex;
  gap: 0.5rem;
  flex: 1;
  max-width: 420px;
}
.search-box input {
  flex: 1;
  padding: 0.5rem 0.75rem;
  border-radius: 8px;
  border: 1px solid var(--border);
  background: var(--color-bg);
  color: var(--color-fg);
}
.btn {
  padding: 0.5rem 1.2rem;
  border: 1px solid var(--border);
  border-radius: 8px;
  font-size: 0.9rem;
  color: var(--color-fg);
  background: transparent;
  cursor: pointer;
}
.btn.primary {
  background: var(--primary);
  border-color: var(--primary);
  color: #fff;
}
.btn.primary:hover {
  background: var(--primary-hover);
}
.btn:disabled {
  opacity: 0.6;
  cursor: default;
}
.grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 1rem;
}
.card {
  padding: 1.2rem;
  border: 1px solid var(--border);
  border-radius: 10px;
  background: var(--card);
  transition: border-color 0.15s, box-shadow 0.15s;
}
.card:hover {
  border-color: var(--primary);
  box-shadow: 0 2px 10px rgba(59, 130, 246, 0.15);
}
.card-title {
  margin: 0 0 0.5rem;
  font-size: 1.05rem;
}
.card-title a {
  color: var(--primary);
}
.owner {
  color: var(--muted);
}
.sep {
  color: var(--muted);
  margin: 0 0.15rem;
}
.name {
  font-weight: 600;
}
.desc {
  margin: 0 0 0.75rem;
  color: var(--muted);
  font-size: 0.9rem;
  line-height: 1.5;
  min-height: 1.35rem;
}
.meta {
  display: flex;
  align-items: center;
  gap: 1rem;
  flex-wrap: wrap;
  font-size: 0.8rem;
  color: var(--muted);
}
.stars {
  color: var(--accent);
}
.state {
  color: var(--muted);
  padding: 1.5rem 0;
}
.error {
  color: var(--accent);
  font-size: 0.9rem;
}
.more {
  display: flex;
  justify-content: center;
  margin-top: 1.5rem;
}
.more .state {
  padding: 0;
}
@media (max-width: 640px) {
  .grid {
    grid-template-columns: 1fr;
  }
}
</style>
