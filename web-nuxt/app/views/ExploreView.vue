<script setup lang="ts">
import { formatDateTime } from '../utils/datetime'
import { onMounted, ref, watch } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
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

interface TrendingRepo {
  id: number
  owner: string
  name: string
  description: string
  stars_count: number
  forks_count: number
}

interface Topic {
  topic: string
  count: number
}

type Tab = 'repos' | 'trending' | 'topics'

const LANGUAGES = ['全部', 'Go', 'JavaScript', 'TypeScript', 'Python', 'Shell']
const PAGE_SIZE = 20

const tabs: { key: Tab; label: string }[] = [
  { key: 'repos', label: '仓库' },
  { key: 'trending', label: '趋势' },
  { key: 'topics', label: '话题' },
]
const route = useRoute()
const router = useRouter()
const validTabs: Tab[] = ['repos', 'trending', 'topics']
const initialTab = validTabs.includes(route.query.tab as Tab) ? (route.query.tab as Tab) : 'repos'
const activeTab = ref<Tab>(initialTab)

watch(
  () => route.query.tab,
  (tab) => {
    if (validTabs.includes(tab as Tab) && activeTab.value !== tab) {
      activeTab.value = tab as Tab
      if (activeTab.value === 'trending' && !trending.value.length && !trendingLoading.value) loadTrending()
      if (activeTab.value === 'topics' && !topics.value.length && !topicsLoading.value) loadTopics()
    }
  },
)

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

const trending = ref<TrendingRepo[]>([])
const trendingLanguage = ref('全部')
const trendingLoading = ref(false)
const trendingError = ref('')

async function loadTrending() {
  trendingLoading.value = true
  trendingError.value = ''
  try {
    const { data } = await api.get<TrendingRepo[]>('/explore/trending', {
      params: {
        language: trendingLanguage.value === '全部' ? undefined : trendingLanguage.value,
        limit: PAGE_SIZE,
      },
    })
    trending.value = data ?? []
  } catch (e) {
    trendingError.value = e instanceof Error ? e.message : '加载趋势失败'
    trending.value = []
  } finally {
    trendingLoading.value = false
  }
}

const topics = ref<Topic[]>([])
const topicsLoading = ref(false)
const topicsError = ref('')
const selectedTopic = ref('')
const topicRepos = ref<TrendingRepo[]>([])
const topicReposLoading = ref(false)
const topicReposError = ref('')

async function loadTopics() {
  topicsLoading.value = true
  topicsError.value = ''
  try {
    const { data } = await api.get<Topic[]>('/explore/topics')
    topics.value = data ?? []
  } catch (e) {
    topicsError.value = e instanceof Error ? e.message : '加载话题失败'
    topics.value = []
  } finally {
    topicsLoading.value = false
  }
}

async function selectTopic(topic: string) {
  selectedTopic.value = topic
  topicReposLoading.value = true
  topicReposError.value = ''
  try {
    const { data } = await api.get<TrendingRepo[]>(`/explore/topics/${encodeURIComponent(topic)}`, {
      params: { language: trendingLanguage.value === '全部' ? undefined : trendingLanguage.value },
    })
    topicRepos.value = data ?? []
  } catch (e) {
    topicReposError.value = e instanceof Error ? e.message : '加载话题仓库失败'
    topicRepos.value = []
  } finally {
    topicReposLoading.value = false
  }
}

function selectTab(tab: Tab) {
  activeTab.value = tab
  router.replace({ query: { ...route.query, tab } })
  if (tab === 'trending' && !trending.value.length && !trendingLoading.value) loadTrending()
  if (tab === 'topics' && !topics.value.length && !topicsLoading.value) loadTopics()
}

function fmtTime(v: string): string {
  return formatDateTime(v)
}

onMounted(() => load(true))
</script>

<template>
  <div class="explore">
    <TopNav variant="landing" />
    <main class="content">
      <header class="head">
        <h2>探索</h2>
        <form v-if="activeTab === 'repos'" class="search-box" @submit.prevent="search">
          <input v-model="query" type="search" placeholder="按仓库名称搜索..." aria-label="按仓库名称搜索" />
          <button class="btn primary" type="submit">搜索</button>
        </form>
      </header>

      <nav class="tabs">
        <button
          v-for="t in tabs"
          :key="t.key"
          :class="{ active: activeTab === t.key }"
          type="button"
          @click="selectTab(t.key)"
        >
          {{ t.label }}
        </button>
      </nav>

      <section v-if="activeTab === 'repos'">
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
              <span class="updated">更新于 {{ fmtTime(r.updated_at) }}</span>
            </div>
          </article>
        </div>
        <div v-if="!loading && repos.length" class="more">
          <button v-if="hasMore" class="btn" :disabled="loadingMore" @click="loadMore">
            {{ loadingMore ? '加载中...' : '加载更多' }}
          </button>
          <p v-else class="state">已经到底啦</p>
        </div>
      </section>

      <section v-else-if="activeTab === 'trending'">
        <div class="filter-row">
          <label class="lang-label" for="trending-lang">语言</label>
          <select
            id="trending-lang"
            v-model="trendingLanguage"
            class="lang-select"
            @change="loadTrending"
          >
            <option v-for="l in LANGUAGES" :key="l" :value="l">{{ l }}</option>
          </select>
        </div>
        <p v-if="trendingError" class="error">{{ trendingError }}</p>
        <p v-if="trendingLoading" class="state">加载中...</p>
        <p v-else-if="!trending.length" class="state">没有找到趋势仓库</p>
        <div v-else class="grid">
          <article v-for="r in trending" :key="r.id" class="card">
            <h3 class="card-title">
              <RouterLink :to="`/${r.owner}/${r.name}`">
                <span class="owner">{{ r.owner }}</span><span class="sep">/</span><span class="name">{{ r.name }}</span>
              </RouterLink>
            </h3>
            <p class="desc">{{ r.description || '暂无描述' }}</p>
            <div class="meta">
              <span class="stars" title="Star 数">★ {{ r.stars_count }}</span>
              <span class="forks" title="Fork 数">⑂ {{ r.forks_count }}</span>
            </div>
          </article>
        </div>
      </section>

      <section v-else class="topics-section">
        <div class="filter-row">
          <label class="lang-label" for="topic-lang">语言</label>
          <select
            id="topic-lang"
            v-model="trendingLanguage"
            class="lang-select"
            :disabled="!selectedTopic"
            @change="selectedTopic && selectTopic(selectedTopic)"
          >
            <option v-for="l in LANGUAGES" :key="l" :value="l">{{ l }}</option>
          </select>
        </div>
        <p v-if="topicsError" class="error">{{ topicsError }}</p>
        <p v-if="topicsLoading" class="state">加载中...</p>
        <p v-else-if="!topics.length" class="state">暂无话题</p>
        <div v-else class="chips">
          <button
            v-for="t in topics"
            :key="t.topic"
            class="topic-chip"
            :class="{ active: t.topic === selectedTopic }"
            type="button"
            @click="selectTopic(t.topic)"
          >
            {{ t.topic }} <span class="chip-count">{{ t.count }}</span>
          </button>
        </div>

        <template v-if="selectedTopic">
          <h3 class="topic-head">#{{ selectedTopic }}</h3>
          <p v-if="topicReposError" class="error">{{ topicReposError }}</p>
          <p v-if="topicReposLoading" class="state">加载中...</p>
          <p v-else-if="!topicRepos.length" class="state">该话题下暂无仓库</p>
          <div v-else class="grid">
            <article v-for="r in topicRepos" :key="r.id" class="card">
              <h3 class="card-title">
                <RouterLink :to="`/${r.owner}/${r.name}`">
                  <span class="owner">{{ r.owner }}</span><span class="sep">/</span><span class="name">{{ r.name }}</span>
                </RouterLink>
              </h3>
              <p class="desc">{{ r.description || '暂无描述' }}</p>
              <div class="meta">
                <span class="stars" title="Star 数">★ {{ r.stars_count }}</span>
                <span class="forks" title="Fork 数">⑂ {{ r.forks_count }}</span>
              </div>
            </article>
          </div>
        </template>
      </section>
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
  margin-bottom: 1rem;
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
.tabs {
  display: flex;
  gap: 0.25rem;
  margin-bottom: 1.5rem;
  border-bottom: 1px solid var(--border);
}
.tabs button {
  padding: 0.5rem 0.9rem;
  border: none;
  background: transparent;
  color: var(--muted);
  border-bottom: 2px solid transparent;
  cursor: pointer;
  font: inherit;
}
.tabs button.active {
  color: var(--primary);
  border-bottom-color: var(--primary);
  font-weight: 600;
}
.filter-row {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  margin-bottom: 1.25rem;
}
.lang-label {
  color: var(--muted);
  font-size: 0.85rem;
}
.lang-select {
  padding: 0.4rem 0.7rem;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--card);
  color: var(--color-fg);
}
.lang-select:disabled {
  opacity: 0.6;
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
  border-radius: 8px;
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
.forks {
  color: var(--muted);
}
.chips {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}
.topic-chip {
  padding: 0.35rem 0.8rem;
  border: 1px solid var(--border);
  border-radius: 999px;
  background: var(--card);
  color: var(--color-fg);
  cursor: pointer;
  font: inherit;
}
.topic-chip:hover {
  border-color: var(--primary);
  color: var(--primary);
}
.topic-chip.active {
  background: var(--primary);
  border-color: var(--primary);
  color: #fff;
}
.chip-count {
  color: var(--muted);
  font-size: 0.78rem;
}
.topic-chip.active .chip-count {
  color: rgba(255, 255, 255, 0.8);
}
.topic-head {
  margin: 1.75rem 0 0.75rem;
  font-size: 1.1rem;
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
