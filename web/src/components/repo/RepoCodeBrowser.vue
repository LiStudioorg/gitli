<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { RouterLink } from 'vue-router'
import api from '../../api/client'
import MarkdownView from '../common/MarkdownView.vue'
import SkeletonBlock from '../common/SkeletonBlock.vue'
import FileIcon from '../icons/FileIcon.vue'

const props = defineProps<{
  owner: string
  repo: string
  branch: string
  branches: string[]
  /** README filename detected by the server; empty means there is none. */
  readmeFilename?: string
}>()

const emit = defineEmits<{ (e: 'update:branch', value: string): void }>()

interface TreeEntry {
  mode: string
  type: 'blob' | 'tree' | 'commit'
  name: string
  size?: number
  hash: string
  last_commit?: Commit
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
const isEmptyRepo = ref(false)
const copiedQuickstart = ref(false)

const quickstart = computed(() => {
  const remote = `${window.location.origin}/${props.owner}/${props.repo}.git`
  return [
    'git init',
    'git add .',
    'git commit -m "first commit"',
    `git branch -M ${props.branch || 'main'}`,
    `git remote add origin ${remote}`,
    `git push -u origin ${props.branch || 'main'}`,
  ].join('\n')
})

async function copyQuickstart() {
  try {
    await navigator.clipboard.writeText(quickstart.value)
    copiedQuickstart.value = true
    setTimeout(() => (copiedQuickstart.value = false), 1500)
  } catch {
    copiedQuickstart.value = false
  }
}
const loading = ref(false)
const error = ref('')

const currentPath = computed(() => pathStack.value.join('/'))
const readmeName = computed(() => readme.value?.name ?? '')

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
    const { data } = await api.get<{ entries: TreeEntry[]; is_empty?: boolean }>(url)
    isEmptyRepo.value = data.is_empty === true
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
  // The server already told us whether a README exists and its exact filename,
  // so there is no need to probe multiple candidates (which produced 404s).
  const candidate = props.readmeFilename
  if (!candidate) {
    readme.value = null
    return
  }
  try {
    const { data } = await api.get(
      `/repos/${props.owner}/${props.repo}/blob/${encodeURIComponent(props.branch)}/${encodeURIComponent(candidate)}`,
      { responseType: 'text', transformResponse: [(v) => v] },
    )
    readme.value = typeof data === 'string' && data.length > 0 ? { name: candidate, content: data } : null
  } catch {
    readme.value = null
  }
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

function joinPath(name: string): string {
  return [currentPath.value, name].filter(Boolean).join('/')
}

function relTime(raw: string): string {
  const d = new Date(raw)
  if (Number.isNaN(d.getTime())) return ''
  const diff = Math.max(0, Date.now() - d.getTime())
  const min = Math.floor(diff / 60000)
  if (min < 1) return '刚刚'
  if (min < 60) return `${min} 分钟前`
  const h = Math.floor(min / 60)
  if (h < 24) return `${h} 小时前`
  const days = Math.floor(h / 24)
  if (days < 30) return `${days} 天前`
  return d.toLocaleDateString()
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
  <section class="flex flex-col gap-4">
    <!-- 工具栏：分支 + 面包屑 -->
    <div class="flex flex-wrap items-center gap-2">
      <select
        class="h-8 rounded-md border border-[#d1d9e0] bg-white px-2 text-sm text-[#1f2328] outline-none focus:border-[#ea580c] focus:ring-2 focus:ring-[#ea580c]/25 dark:border-[#30363d] dark:bg-[#161b22] dark:text-[#e6edf3]"
        :value="branch"
        aria-label="选择分支"
        title="选择分支"
        @change="changeBranch"
      >
        <option v-for="b in branches.length ? branches : [branch]" :key="b" :value="b">{{ b }}</option>
      </select>
      <nav class="flex min-w-0 flex-wrap items-center gap-0.5 text-sm" aria-label="路径">
        <button
          type="button"
          class="truncate rounded px-1 font-semibold text-[#ea580c] hover:underline"
          @click="jumpTo(0)"
        >
          {{ repo }}
        </button>
        <template v-for="(p, i) in pathStack" :key="p + i">
          <span class="text-[#59636e]">/</span>
          <button type="button" class="truncate rounded px-1 text-[#ea580c] hover:underline" @click="jumpTo(i + 1)">
            {{ p }}
          </button>
        </template>
      </nav>
      <button
        type="button"
        class="ml-auto h-8 rounded-md border border-[#d1d9e0] px-2.5 text-xs text-[#1f2328] hover:bg-[#f6f8fa] dark:border-[#30363d] dark:text-[#e6edf3] dark:hover:bg-[#21262d]"
        aria-label="复制当前路径"
        title="复制当前路径"
        @click="copyPath"
      >
        复制路径
      </button>
    </div>

    <p
      v-if="error && !isEmptyRepo"
      class="rounded-md border border-[#ff818266] bg-[#ffebe9] px-3 py-2 text-sm text-[#1f2328]"
    >
      {{ error }}
    </p>

    <!-- 空仓库引导 -->
    <section
      v-else-if="isEmptyRepo"
      class="flex flex-col items-center gap-2 rounded-md border border-dashed border-[#d1d9e0] bg-white px-4 py-10 text-center dark:border-[#30363d] dark:bg-[#161b22]"
    >
      <div class="flex h-12 w-12 items-center justify-center rounded-full bg-[#ea580c]/10 text-xl" aria-hidden="true">📦</div>
      <h3 class="text-base font-semibold text-[#1f2328] dark:text-[#e6edf3]">这是一个空仓库</h3>
      <p class="max-w-md text-sm text-[#59636e]">还没有任何提交。在本地推送第一个提交后，这里会显示文件列表与 README。</p>
      <pre class="mt-2 w-full max-w-[620px] overflow-x-auto rounded-md border border-[#d1d9e0] bg-[#f6f8fa] p-3 text-left font-mono text-xs leading-relaxed text-[#1f2328] dark:border-[#30363d] dark:bg-[#0d1117] dark:text-[#e6edf3]"><code>{{ quickstart }}</code></pre>
      <button
        type="button"
        class="mt-1 h-8 rounded-md border border-[#d1d9e0] px-3 text-sm text-[#1f2328] hover:bg-[#f6f8fa] dark:border-[#30363d] dark:text-[#e6edf3] dark:hover:bg-[#21262d]"
        @click="copyQuickstart"
      >
        {{ copiedQuickstart ? '已复制' : '复制命令' }}
      </button>
    </section>

    <!-- 加载骨架 -->
    <div v-else-if="loading" class="flex flex-col gap-2 rounded-md border border-[#d1d9e0] dark:border-[#30363d]">
      <div v-for="i in 6" :key="i" class="border-b border-[#d1d9e0] p-3 last:border-0 dark:border-[#30363d]">
        <SkeletonBlock :lines="1" height="1.1rem" />
      </div>
    </div>

    <template v-else>
      <!-- 最新提交条（表头） -->
      <div
        v-if="latestCommit"
        class="flex items-center gap-3 rounded-t-md border border-b-0 border-[#d1d9e0] bg-[#f6f8fa] p-1 dark:border-[#3d444d] dark:bg-[#161b22]"
      >
        <span class="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-[#ea580c] text-xs font-semibold text-white">
          {{ latestCommit.author?.[0]?.toUpperCase() ?? '?' }}
        </span>
        <div class="min-w-0 flex-1">
          <p class="truncate text-sm font-medium text-[#1f2328] dark:text-[#e6edf3]">{{ latestCommit.subject }}</p>
          <p class="truncate text-xs text-[#59636e]">
            {{ latestCommit.author }} · {{ dateLabel(latestCommit.date) }}
            <code class="rounded bg-[#d1d9e0]/40 px-1 font-mono dark:bg-[#30363d]">{{ latestCommit.short }}</code>
          </p>
        </div>
        <RouterLink
          class="inline-flex h-8 shrink-0 items-center gap-1.5 rounded-md bg-[#2da44e] px-3 text-xs font-semibold text-white hover:bg-[#2c974b]"
          :to="`/${owner}/${repo}?tab=活动`"
        >
          <svg class="h-3.5 w-3.5" viewBox="0 0 16 16" fill="currentColor" aria-hidden="true">
            <path d="M5 5.782V2.5h-.25a.75.75 0 0 1 0-1.5h6.5a.75.75 0 0 1 0 1.5H11v3.282l3.666 5.76C15.216 12.188 14.703 13 13.826 13H2.174c-.877 0-1.39-.811-.84-1.458Z" />
          </svg>
          代码
        </RouterLink>
      </div>

      <!-- 文件列表 -->
      <ul class="overflow-hidden rounded-b-md border border-t-0 border-[#d1d9e0] bg-white dark:border-[#3d444d] dark:bg-[#161b22]">
        <li
          v-if="currentPath"
          class="flex h-10 cursor-pointer items-center gap-2 border-t border-[#d1d9e0] pl-4 pr-3 text-sm hover:bg-[#f6f8fa] dark:border-[#3d444d] dark:hover:bg-[#161b22]"
          @click="jumpTo(pathStack.length - 1)"
        >
          <FileIcon name=".." type="parent" />
          <span class="truncate font-medium text-[#1f2328] dark:text-[#e6edf3]">..</span>
        </li>
        <li
          v-for="entry in entries"
          :key="entry.hash + entry.name"
          class="flex h-10 items-center gap-2 border-t border-[#d1d9e0] pl-4 pr-3 text-sm dark:border-[#3d444d]"
          :class="entry.type === 'tree' ? 'cursor-pointer hover:bg-[#f6f8fa] dark:hover:bg-[#161b22]' : ''"
          @click="enter(entry)"
        >
          <FileIcon :name="entry.name" :type="entry.type" />
          <RouterLink
            v-if="entry.type !== 'tree'"
            class="min-w-0 flex-1 truncate text-[#1f2328] hover:text-[#ea580c] hover:underline dark:text-[#e6edf3]"
            :to="`/${owner}/${repo}?tab=代码&path=${encodeURIComponent(joinPath(entry.name))}`"
          >
            {{ entry.name }}
          </RouterLink>
          <span v-else class="min-w-0 flex-1 truncate font-medium text-[#1f2328] dark:text-[#e6edf3]">
            {{ entry.name }}
          </span>
          <span class="hidden min-w-0 flex-1 truncate text-xs text-[#59636e] sm:block">
            {{ entry.last_commit?.subject ?? '' }}
          </span>
          <span class="hidden w-[125px] shrink-0 truncate text-right text-xs text-[#59636e] md:block">
            {{ entry.last_commit ? relTime(entry.last_commit.date) : '' }}
          </span>
        </li>
      </ul>
      <p v-if="!entries.length" class="py-8 text-center text-sm text-[#59636e]">仓库为空，或该分支还没有提交。</p>

      <!-- README 独立卡片 -->
      <section
        v-if="readme && !currentPath"
        class="mt-2 overflow-hidden rounded-md border border-[#d1d9e0] bg-white dark:border-[#30363d] dark:bg-[#161b22]"
      >
        <header class="flex items-center gap-2 border-b border-[#d1d9e0] px-4 py-2.5 dark:border-[#30363d]">
          <svg class="h-4 w-4 text-[#59636e]" viewBox="0 0 16 16" fill="currentColor" aria-hidden="true">
            <path d="M0 1.75A.75.75 0 0 1 .75 1h4.253c1.227 0 2.317.59 3 1.501A3.743 3.743 0 0 1 11.006 1h4.245a.75.75 0 0 1 .75.75v10.5a.75.75 0 0 1-.75.75h-4.507a2.25 2.25 0 0 0-1.591.659l-.622.621a.75.75 0 0 1-1.06 0l-.622-.621A2.25 2.25 0 0 0 5.258 13H.75a.75.75 0 0 1-.75-.75Z" />
          </svg>
          <span class="truncate text-sm font-semibold text-[#1f2328] dark:text-[#e6edf3]">{{ readmeName }}</span>
        </header>
        <div class="px-6 py-5">
          <MarkdownView :source="readme.content" />
        </div>
      </section>
    </template>
  </section>
</template>
