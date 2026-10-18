<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import TopNav from '../components/layout/TopNav.vue'
import api from '../api/client'

interface RepoResult {
  id: number
  owner: string
  name: string
  description: string
  stars_count: number
  language: string
}

interface IssueResult {
  kind: string
  id: number
  repo_id: number
  number: number
  title: string
  state: string
}

interface UserResult {
  kind: string
  id: number
  name: string
  full_name: string
}

interface SearchResponse {
  repos: RepoResult[]
  issues: IssueResult[]
  users: UserResult[]
}

const route = useRoute()
const router = useRouter()

const query = ref('')
const input = ref('')
const loading = ref(false)
const error = ref('')
const repos = ref<RepoResult[]>([])
const issues = ref<IssueResult[]>([])
const users = ref<UserResult[]>([])

const hasResults = computed(() => repos.value.length + issues.value.length + users.value.length > 0)

async function runSearch(q: string) {
  if (!q) {
    repos.value = []
    issues.value = []
    users.value = []
    return
  }
  loading.value = true
  error.value = ''
  try {
    const { data } = await api.get<SearchResponse>('/search', { params: { q, limit: 30 } })
    repos.value = data.repos ?? []
    issues.value = data.issues ?? []
    users.value = data.users ?? []
  } catch (e) {
    error.value = e instanceof Error ? e.message : '搜索失败'
    repos.value = []
    issues.value = []
    users.value = []
  } finally {
    loading.value = false
  }
}

function submit() {
  const q = input.value.trim()
  if (!q) return
  router.push({ path: '/search', query: { q } })
}

watch(
  () => route.query.q,
  (q) => {
    const term = typeof q === 'string' ? q : ''
    query.value = term
    input.value = term
    void runSearch(term)
  },
  { immediate: true },
)
</script>

<template>
  <div class="min-h-screen bg-[#f6f8fa] dark:bg-[#0d1117]">
    <TopNav variant="app" />
    <main class="mx-auto w-full max-w-[1000px] px-4 py-6 md:px-6">
      <header class="mb-4 flex flex-wrap items-center justify-between gap-3">
        <h2 class="text-lg font-semibold text-[#1f2328] dark:text-[#e6edf3]">搜索</h2>
        <form class="flex gap-2" @submit.prevent="submit">
          <input
            v-model="input"
            type="search"
            placeholder="搜索仓库、Issue、用户..."
            aria-label="搜索"
            class="h-8 w-72 rounded-md border border-[#d1d9e0] bg-[#f6f8fa] px-3 text-sm text-[#1f2328] outline-none placeholder:text-[#59636e] focus:border-[#0969da] focus:bg-white focus:ring-2 focus:ring-[#0969da]/25 dark:border-[#30363d] dark:bg-[#0d1117] dark:text-[#e6edf3]"
          />
          <button class="h-8 rounded-md bg-[#2da44e] px-3 text-sm font-semibold text-white hover:bg-[#2c974b]" type="submit">搜索</button>
        </form>
      </header>

      <div class="mb-4 flex flex-wrap items-center gap-2 rounded-md border border-[#d1d9e0] bg-white px-3 py-2 text-xs text-[#59636e] dark:border-[#30363d] dark:bg-[#161b22]">
        <span class="font-semibold">语法提示</span>
        <code>language:go</code>
        <code>is:open</code>
        <code>author:alice</code>
        <code>org:acme</code>
        <code>stars:&gt;100</code>
      </div>

      <p v-if="error" class="mb-3 rounded-md border border-[#ff818266] bg-[#ffebe9] px-3 py-2 text-sm text-[#cf222e]">{{ error }}</p>
      <p v-else-if="loading" class="py-10 text-center text-sm text-[#59636e]">搜索中…</p>
      <p v-else-if="!query" class="py-10 text-center text-sm text-[#59636e]">输入关键词开始搜索</p>
      <p v-else-if="!hasResults" class="py-10 text-center text-sm text-[#59636e]">没有找到与「{{ query }}」相关的结果</p>

      <template v-else>
        <section v-if="repos.length" class="group">
          <h3>仓库 <span class="count">{{ repos.length }}</span></h3>
          <ul class="result-list">
            <li v-for="r in repos" :key="`repo-${r.id}`" class="result">
              <RouterLink class="title" :to="`/${r.owner}/${r.name}`">
                <span class="owner">{{ r.owner }}</span><span class="sep">/</span><span class="name">{{ r.name }}</span>
              </RouterLink>
              <p class="desc">{{ r.description || '暂无描述' }}</p>
              <div class="meta">
                <span v-if="r.language" class="language">{{ r.language }}</span>
                <span class="stars" title="Star 数">★ {{ r.stars_count }}</span>
              </div>
            </li>
          </ul>
        </section>

        <section v-if="issues.length" class="group">
          <h3>Issue 与 PR <span class="count">{{ issues.length }}</span></h3>
          <ul class="result-list">
            <li v-for="i in issues" :key="`issue-${i.kind}-${i.id}`" class="result">
              <span class="kind">{{ i.kind === 'pull' ? 'PR' : 'Issue' }}</span>
              <span class="title-text">#{{ i.number }} {{ i.title }}</span>
              <span class="state-badge">{{ i.state }}</span>
            </li>
          </ul>
        </section>

        <section v-if="users.length" class="group">
          <h3>用户与组织 <span class="count">{{ users.length }}</span></h3>
          <ul class="result-list">
            <li v-for="u in users" :key="`user-${u.id}`" class="result">
              <span class="kind">{{ u.kind === 'org' ? '组织' : '用户' }}</span>
              <RouterLink v-if="u.kind !== 'org'" class="title-text user-link" :to="`/users/${u.name}`">
                {{ u.name }}
              </RouterLink>
              <span v-else class="title-text">{{ u.name }}</span>
              <span class="full-name">{{ u.full_name }}</span>
            </li>
          </ul>
        </section>
      </template>
    </main>
  </div>
</template>
