<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { RouterLink } from 'vue-router'
import api from '../../api/client'
import MarkdownView from '../common/MarkdownView.vue'
import SkeletonBlock from '../common/SkeletonBlock.vue'

const props = defineProps<{
  owner: string
  repo: string
  branch: string
  branches: string[]
}>()

const emit = defineEmits<{ (e: 'update:branch', value: string): void }>()

interface TreeEntry {
  mode: string
  type: 'blob' | 'tree' | 'commit'
  name: string
  size?: number
  hash: string
}

interface Commit {
  hash: string
  short: string
  author: string
  email: string
  date: string
  subject: string
}

const pathStack = ref<string[]>([])
const entries = ref<TreeEntry[]>([])
const latestCommit = ref<Commit | null>(null)
const readme = ref<{ name: string; content: string } | null>(null)
const loading = ref(false)
const error = ref('')

const currentPath = computed(() => pathStack.value.join('/'))
const readmeName = computed(() => readme.value?.name ?? '')

const README_CANDIDATES = ['README.md', 'readme.md', 'README.markdown', 'README.rst', 'README']

function sizeLabel(n?: number): string {
  if (!n) return '—'
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`
  return `${(n / 1024 / 1024).toFixed(1)} MB`
}

function dateLabel(raw: string): string {
  const d = new Date(raw)
  if (Number.isNaN(d.getTime())) return raw
  return d.toLocaleDateString()
}

async function loadTree() {
  loading.value = true
  error.value = ''
  try {
    const base = `/repos/${props.owner}/${props.repo}/tree/${encodeURIComponent(props.branch)}`
    const url = currentPath.value ? `${base}/${currentPath.value}` : base
    const { data } = await api.get<{ entries: TreeEntry[] }>(url)
    const list = data.entries ?? []
    list.sort((a, b) => {
      if (a.type === b.type) return a.name.localeCompare(b.name)
      return a.type === 'tree' ? -1 : 1
    })
    entries.value = list
    await loadLatestCommit()
    if (!currentPath.value) await loadReadme()
  } catch (e: any) {
    error.value = e?.response?.data?.error ?? '无法加载仓库文件'
    entries.value = []
  } finally {
    loading.value = false
  }
}

async function loadLatestCommit() {
  try {
    const { data } = await api.get<Commit[]>(`/repos/${props.owner}/${props.repo}/commits/${encodeURIComponent(props.branch)}`, {
      params: { limit: 1 },
    })
    latestCommit.value = data?.[0] ?? null
  } catch {
    latestCommit.value = null
  }
}

async function loadReadme() {
  for (const candidate of README_CANDIDATES) {
    try {
      const { data } = await api.get(
        `/repos/${props.owner}/${props.repo}/blob/${encodeURIComponent(props.branch)}/${candidate}`,
        { responseType: 'text', transformResponse: [(v) => v] },
      )
      if (typeof data === 'string' && data.length > 0) {
        readme.value = { name: candidate, content: data }
        return
      }
    } catch {
      /* try next candidate */
    }
  }
  readme.value = null
}

function enter(entry: TreeEntry) {
  if (entry.type !== 'tree') return
  pathStack.value = [...pathStack.value, entry.name]
}

function jumpTo(index: number) {
  pathStack.value = pathStack.value.slice(0, index)
}

function changeBranch(e: Event) {
  const value = (e.target as HTMLSelectElement).value
  emit('update:branch', value)
}

function copyPath() {
  const full = [props.branch, currentPath.value].filter(Boolean).join('/')
  void navigator.clipboard?.writeText(full)
}

watch(
  () => [props.branch, props.owner, props.repo],
  () => {
    pathStack.value = []
    void loadTree()
  },
)

watch(currentPath, () => void loadTree())

onMounted(() => void loadTree())
</script>

<template>
  <section class="browser">
    <div class="toolbar">
      <select class="branch-select" :value="branch" aria-label="选择分支" title="选择分支" @change="changeBranch">
        <option v-for="b in branches.length ? branches : [branch]" :key="b" :value="b">{{ b }}</option>
      </select>
      <nav class="breadcrumb">
        <button type="button" class="crumb" @click="jumpTo(0)">{{ repo }}</button>
        <template v-for="(p, i) in pathStack" :key="p + i">
          <span class="sep">/</span>
          <button type="button" class="crumb" @click="jumpTo(i + 1)">{{ p }}</button>
        </template>
      </nav>
      <button type="button" class="chip" aria-label="复制当前路径" title="复制当前路径" @click="copyPath">复制路径</button>
    </div>

    <p v-if="error" class="error">{{ error }}</p>
    <div v-else-if="loading" class="skeleton-list">
      <SkeletonBlock v-for="i in 5" :key="i" :lines="1" height="1.4rem" />
    </div>

    <template v-else>
      <div v-if="latestCommit" class="commit-bar">
        <span class="avatar">{{ latestCommit.author?.[0]?.toUpperCase() ?? '?' }}</span>
        <div class="commit-main">
          <p class="commit-subject">{{ latestCommit.subject }}</p>
          <p class="muted small">
            {{ latestCommit.author }} · {{ dateLabel(latestCommit.date) }}
            <code class="hash">{{ latestCommit.short }}</code>
          </p>
        </div>
        <RouterLink class="chip" :to="`/${owner}/${repo}?tab=活动`">历史</RouterLink>
      </div>

      <ul class="file-list">
        <li v-if="currentPath" class="file-row" @click="jumpTo(pathStack.length - 1)">
          <span class="fi">↰</span>
          <span class="fname">..</span>
        </li>
        <li
          v-for="entry in entries"
          :key="entry.hash + entry.name"
          class="file-row"
          :class="{ dir: entry.type === 'tree' }"
          @click="enter(entry)"
        >
          <span class="fi">{{ entry.type === 'tree' ? '📁' : entry.type === 'commit' ? '🔗' : '📄' }}</span>
          <span class="fname" :class="{ clickable: entry.type === 'tree' }">{{ entry.name }}</span>
          <span class="size">{{ entry.type === 'tree' ? '—' : sizeLabel(entry.size) }}</span>
          <span class="msg">{{ entry.type === 'tree' ? '' : entry.hash.slice(0, 7) }}</span>
        </li>
      </ul>
      <p v-if="!entries.length" class="muted empty">仓库为空，或该分支还没有提交。</p>

      <section v-if="readme && !currentPath" class="readme">
        <header class="readme-head">
          <span>📖 {{ readmeName }}</span>
        </header>
        <MarkdownView :source="readme.content" />
      </section>
    </template>
  </section>
</template>

<style scoped>
.browser {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}
.toolbar {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  flex-wrap: wrap;
}
.branch-select {
  padding: 0.35rem 0.6rem;
  border-radius: 6px;
  border: 1px solid var(--border);
  background: var(--card);
  color: var(--fg);
}
.breadcrumb {
  display: flex;
  align-items: center;
  gap: 0.15rem;
  flex-wrap: wrap;
}
.crumb {
  border: none;
  background: transparent;
  color: var(--primary);
  cursor: pointer;
  font-size: 0.88rem;
  padding: 0.1rem 0.2rem;
}
.sep {
  color: var(--muted);
}
.chip {
  padding: 0.3rem 0.7rem;
  border-radius: 6px;
  border: 1px solid var(--border);
  background: transparent;
  color: var(--fg);
  font-size: 0.82rem;
  cursor: pointer;
  text-decoration: none;
}
.commit-bar {
  display: flex;
  align-items: center;
  gap: 0.7rem;
  padding: 0.6rem 0.8rem;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--card);
}
.avatar {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  background: var(--primary);
  color: #fff;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-weight: 600;
  font-size: 0.8rem;
}
.commit-main {
  flex: 1;
  min-width: 0;
}
.commit-subject {
  margin: 0;
  font-size: 0.9rem;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.muted {
  color: var(--muted);
}
.small {
  font-size: 0.78rem;
  margin: 0;
}
.hash {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  background: color-mix(in srgb, var(--border) 40%, transparent);
  padding: 0 0.3rem;
  border-radius: 4px;
}
.file-list {
  list-style: none;
  margin: 0;
  padding: 0;
  border: 1px solid var(--border);
  border-radius: 8px;
  overflow: hidden;
}
.file-row {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  padding: 0.5rem 0.8rem;
  border-bottom: 1px solid var(--border);
  font-size: 0.88rem;
}
.file-row:last-child {
  border-bottom: none;
}
.file-row.dir {
  cursor: pointer;
}
.file-row:hover {
  background: color-mix(in srgb, var(--border) 25%, transparent);
}
.fi {
  width: 1.1rem;
  text-align: center;
}
.fname {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.fname.clickable {
  color: var(--primary);
}
.size,
.msg {
  color: var(--muted);
  font-size: 0.78rem;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
}
.empty {
  margin: 0.25rem 0;
  font-size: 0.85rem;
}
.readme {
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--card);
  overflow: hidden;
}
.readme-head {
  padding: 0.55rem 0.9rem;
  border-bottom: 1px solid var(--border);
  font-size: 0.86rem;
  font-weight: 600;
  color: var(--muted);
}
.readme :deep(.markdown-body) {
  padding: 1rem 1.1rem;
}
.skeleton-list {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 0.75rem;
}
.error {
  color: #d1242f;
  font-size: 0.85rem;
}
</style>
