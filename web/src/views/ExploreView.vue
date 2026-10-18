<script setup lang="ts">
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

function formatDate(v: string): string {
  const d = new Date(v)
  return Number.isNaN(d.getTime()) ? v : d.toLocaleString()
}

onMounted(() => load(true))
</script>

<template>
  <div class="min-h-screen bg-[#f6f8fa] dark:bg-[#0d1117]">
    <TopNav variant="landing" />
    <main class="mx-auto w-full max-w-[1280px] px-4 py-6 md:px-6">
      <header class="mb-4 flex flex-wrap items-center justify-between gap-3">
        <h2 class="text-lg font-semibold text-[#1f2328] dark:text-[#e6edf3]">探索</h2>
        <form v-if="activeTab === 'repos'" class="flex gap-2" @submit.prevent="search">
          <input
            v-model="query"
            type="search"
            placeholder="按仓库名称搜索..."
            aria-label="按仓库名称搜索"
            class="h-8 w-64 rounded-md border border-[#d1d9e0] bg-[#f6f8fa] px-3 text-sm text-[#1f2328] outline-none placeholder:text-[#59636e] focus:border-[#0969da] focus:bg-white focus:ring-2 focus:ring-[#0969da]/25 dark:border-[#30363d] dark:bg-[#0d1117] dark:text-[#e6edf3]"
          />
          <button class="h-8 rounded-md bg-[#2da44e] px-3 text-sm font-semibold text-white hover:bg-[#2c974b]" type="submit">搜索</button>
        </form>
      </header>

      <nav class="mb-4 flex gap-1 border-b border-[#d1d9e0] dark:border-[#30363d]">
        <button
          v-for="t in tabs"
          :key="t.key"
          class="h-10 border-b-2 px-3 text-sm"
          :class="activeTab === t.key
            ? 'border-[#ea580c] font-semibold text-[#1f2328] dark:text-[#e6edf3]'
            : 'border-transparent text-[#59636e] hover:rounded-t-md hover:border-[#d1d9e0] hover:bg-[#f6f8fa] dark:hover:bg-[#21262d]'"
          type="button"
          @click="selectTab(t.key)"
        >
          {{ t.label }}
        </button>
      </nav>

      <section v-if="activeTab === 'repos'">
        <p v-if="error" class="mb-3 rounded-md border border-[#ff818266] bg-[#ffebe9] px-3 py-2 text-sm text-[#cf222e]">{{ error }}</p>
        <div v-if="loading" class="grid grid-cols-1 gap-4 md:grid-cols-2"><div v-for="i in 4" :key="i" class="h-28 animate-pulse rounded-md border border-[#d1d9e0] bg-white dark:border-[#30363d] dark:bg-[#161b22]" /></div>
        <p v-else-if="!repos.length" class="py-10 text-center text-sm text-[#59636e]">没有找到公开仓库</p>
        <div v-else class="grid grid-cols-1 gap-4 md:grid-cols-2">
          <article v-for="r in repos" :key="r.id" class="flex flex-col rounded-md border border-[#d1d9e0] bg-white p-4 dark:border-[#30363d] dark:bg-[#161b22]">
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
        <div v-else class="grid grid-cols-1 gap-4 md:grid-cols-2">
          <article v-for="r in trending" :key="r.id" class="flex flex-col rounded-md border border-[#d1d9e0] bg-white p-4 dark:border-[#30363d] dark:bg-[#161b22]">
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
          <div v-else class="grid grid-cols-1 gap-4 md:grid-cols-2">
            <article v-for="r in topicRepos" :key="r.id" class="flex flex-col rounded-md border border-[#d1d9e0] bg-white p-4 dark:border-[#30363d] dark:bg-[#161b22]">
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
