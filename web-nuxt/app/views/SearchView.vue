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
  <div class="search-page">
    <TopNav variant="app" />
    <main class="content">
      <header class="head">
        <h2>搜索</h2>
        <form class="search-box" @submit.prevent="submit">
          <input v-model="input" type="search" placeholder="搜索仓库、Issue、用户..." aria-label="搜索" />
          <button class="btn primary" type="submit">搜索</button>
        </form>
      </header>

      <div class="syntax-hint">
        <span class="hint-label">语法提示</span>
        <code>language:go</code>
        <code>is:open</code>
        <code>author:alice</code>
        <code>org:acme</code>
        <code>stars:&gt;100</code>
      </div>

      <p v-if="error" class="error">{{ error }}</p>
      <p v-else-if="loading" class="state">搜索中...</p>
      <p v-else-if="!query" class="state">输入关键词开始搜索</p>
      <p v-else-if="!hasResults" class="state">没有找到与「{{ query }}」相关的结果</p>

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

<style scoped>
.search-page {
  min-height: 100vh;
}
.content {
  max-width: 860px;
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
  max-width: 460px;
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
.syntax-hint {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
  margin-bottom: 1.5rem;
  font-size: 0.8rem;
  color: var(--muted);
}
.syntax-hint .hint-label {
  font-weight: 600;
}
.syntax-hint code {
  padding: 0.15rem 0.5rem;
  border: 1px solid var(--border);
  border-radius: 999px;
  background: var(--card);
  color: var(--color-fg);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
}
.group {
  margin-bottom: 2rem;
}
.group h3 {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin: 0 0 0.75rem;
  font-size: 1rem;
  border-bottom: 1px solid var(--border);
  padding-bottom: 0.5rem;
}
.count {
  padding: 0.05rem 0.5rem;
  border-radius: 999px;
  background: var(--border);
  color: var(--muted);
  font-size: 0.75rem;
}
.result-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}
.result {
  padding: 0.75rem 0.9rem;
  border: 1px solid var(--border);
  border-radius: 10px;
  background: var(--card);
  display: flex;
  align-items: center;
  gap: 0.6rem;
  flex-wrap: wrap;
}
.result .title {
  color: var(--primary);
  font-weight: 600;
}
.owner,
.sep {
  color: var(--muted);
}
.desc {
  width: 100%;
  margin: 0;
  color: var(--muted);
  font-size: 0.88rem;
  line-height: 1.5;
}
.meta {
  display: flex;
  gap: 0.9rem;
  font-size: 0.78rem;
  color: var(--muted);
}
.stars {
  color: var(--accent);
}
.kind {
  padding: 0.1rem 0.5rem;
  border-radius: 6px;
  border: 1px solid var(--border);
  font-size: 0.72rem;
  color: var(--muted);
}
.title-text {
  flex: 1;
}
.user-link {
  color: var(--primary);
  font-weight: 600;
}
.full-name {
  color: var(--muted);
  font-size: 0.82rem;
}
.state-badge {
  padding: 0.1rem 0.55rem;
  border-radius: 999px;
  border: 1px solid var(--border);
  font-size: 0.72rem;
  color: var(--muted);
}
.state {
  color: var(--muted);
  padding: 1.5rem 0;
}
.error {
  color: var(--accent);
  font-size: 0.9rem;
}
</style>
