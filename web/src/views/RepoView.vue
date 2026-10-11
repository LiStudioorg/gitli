<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import TopNav from '../components/layout/TopNav.vue'
import CloneDropdown from '../components/repo/CloneDropdown.vue'
import LanguageStats from '../components/repo/LanguageStats.vue'
import api from '../api/client'
import type { CloneInfo, LanguageStat, Repo } from '../stores'

interface RepoResponse {
  repo: Repo
  languages: LanguageStat[]
}

const route = useRoute()
const owner = route.params.owner as string
const name = route.params.repo as string

const repo = ref<Repo | null>(null)
const languages = ref<LanguageStat[]>([])
const cloneInfo = ref<CloneInfo | null>(null)
const branches = ref<string[]>([])
const selectedBranch = ref('')

const tabs = ['代码', '工单', '合并请求', '工作流', '项目', '发布', '百科', '活动', '更多']

onMounted(async () => {
  try {
    const { data } = await api.get<RepoResponse>(`/repos/${owner}/${name}`)
    repo.value = data.repo
    languages.value = data.languages ?? []
    selectedBranch.value = data.repo.default_branch || 'main'
  } catch {
    repo.value = null
  }
  try {
    const { data } = await api.get<CloneInfo>(`/repos/${owner}/${name}/clone-info`)
    cloneInfo.value = data
  } catch {
    cloneInfo.value = null
  }
  try {
    const { data } = await api.get<string[]>(`/repos/${owner}/${name}/branches`)
    branches.value = data ?? []
  } catch {
    branches.value = []
  }
})
</script>

<template>
  <div class="repo-page">
    <TopNav variant="app" />

    <header class="repo-header">
      <div class="title-row">
        <span class="repo-icon">&lt;/&gt;</span>
        <h2>{{ owner }} / {{ name }}</h2>
        <span v-if="repo?.is_private" class="badge private">私有</span>
        <div class="actions">
          <button class="chip">Watch 1</button>
          <button class="chip accent">Star 1</button>
          <button class="chip">Fork 0</button>
        </div>
      </div>
      <nav class="repo-tabs">
        <button v-for="(t, i) in tabs" :key="t" :class="{ active: i === 0 }">{{ t }}</button>
      </nav>
    </header>

    <div class="repo-meta">
      <span>225 提交</span>
      <span>{{ branches.length }} 分支</span>
      <span>7 Git 标签</span>
      <input class="code-search" type="search" placeholder="搜索代码..." />
    </div>

    <div class="repo-body">
      <main class="main-col">
        <div class="toolbar">
          <select v-model="selectedBranch" class="branch-select">
            <option v-for="b in branches.length ? branches : [selectedBranch]" :key="b" :value="b">{{ b }}</option>
          </select>
          <button class="chip">转到文件</button>
          <button class="chip primary">添加文件</button>
          <CloneDropdown :owner="owner" :repo="name" :clone-info="cloneInfo" />
        </div>

        <ul class="file-list">
          <li class="row">
            <span class="fi folder">📁</span>
            <span class="fname">internal</span>
            <span class="msg">refactor: structure</span>
            <span class="when">1 小时前</span>
          </li>
          <li class="row">
            <span class="fi folder">📁</span>
            <span class="fname">web</span>
            <span class="msg">feat: landing page</span>
            <span class="when">3 小时前</span>
          </li>
          <li class="row">
            <span class="fi file">📄</span>
            <span class="fname">.gitignore</span>
            <span class="msg">chore: init</span>
            <span class="when">上周</span>
          </li>
          <li class="row">
            <span class="fi file">📄</span>
            <span class="fname">main.go</span>
            <span class="msg">feat: entrypoint</span>
            <span class="when">上周</span>
          </li>
          <li class="row">
            <span class="fi file">📄</span>
            <span class="fname">README.md</span>
            <span class="msg">docs: readme</span>
            <span class="when">上周</span>
          </li>
        </ul>

        <section class="readme">
          <h3>README.md</h3>
          <p>本仓库为镜像仓库，内容随上游同步更新。</p>
        </section>
      </main>

      <aside class="side-col">
        <section class="card">
          <h4>描述</h4>
          <p>{{ repo?.description || '本仓库为镜像仓库...' }}</p>
        </section>
        <section class="card">
          <h4>管理主题</h4>
          <div class="tags">
            <span class="tag">自述文档</span>
            <span class="tag">AGPL-3.0</span>
          </div>
          <p class="size">2.1 MiB</p>
          <p class="social">1 点赞 · 1 关注</p>
        </section>
        <section class="card">
          <h4>发布</h4>
          <span class="tag accent">LiCore v0.9.11 最新</span>
        </section>
        <section class="card">
          <h4>语言</h4>
          <LanguageStats :languages="languages" />
        </section>
      </aside>
    </div>
  </div>
</template>

<style scoped>
.repo-page {
  max-width: 1200px;
  margin: 0 auto;
}
.repo-header {
  padding: 1rem 1.5rem 0;
}
.title-row {
  display: flex;
  align-items: center;
  gap: 0.6rem;
}
.repo-icon {
  color: var(--primary);
  font-weight: 700;
}
.title-row h2 {
  margin: 0;
  font-size: 1.3rem;
}
.badge.private {
  padding: 0.1rem 0.5rem;
  border: 1px solid var(--border);
  border-radius: 999px;
  font-size: 0.75rem;
  color: var(--muted);
}
.actions {
  margin-left: auto;
  display: flex;
  gap: 0.4rem;
}
.repo-tabs {
  display: flex;
  gap: 0.25rem;
  margin-top: 0.9rem;
  border-bottom: 1px solid var(--border);
  flex-wrap: wrap;
}
.repo-tabs button {
  padding: 0.5rem 0.8rem;
  border: none;
  background: transparent;
  color: var(--muted);
  border-bottom: 2px solid transparent;
}
.repo-tabs button.active {
  color: var(--primary);
  border-bottom-color: var(--primary);
  font-weight: 600;
}
.repo-meta {
  display: flex;
  align-items: center;
  gap: 1.25rem;
  padding: 0.75rem 1.5rem;
  border-bottom: 1px solid var(--border);
  color: var(--muted);
  font-size: 0.9rem;
}
.code-search {
  margin-left: auto;
  padding: 0.35rem 0.7rem;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: var(--color-bg);
  color: var(--color-fg);
}
.repo-body {
  display: grid;
  grid-template-columns: 1fr 320px;
  gap: 1.5rem;
  padding: 1.25rem 1.5rem;
  align-items: start;
}
.toolbar {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-bottom: 1rem;
  flex-wrap: wrap;
}
.branch-select {
  padding: 0.4rem 0.7rem;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--card);
  color: var(--color-fg);
  font-weight: 600;
}
.chip {
  padding: 0.4rem 0.8rem;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--card);
  color: var(--color-fg);
}
.chip:hover {
  background: var(--hover-bg);
  border-color: var(--primary);
  color: var(--primary);
}
.chip.primary {
  background: var(--primary);
  border-color: var(--primary);
  color: #fff;
}
.chip.primary:hover {
  background: var(--primary-hover);
  color: #fff;
}
.chip.accent {
  border-color: var(--accent);
  color: var(--accent);
}
.file-list {
  list-style: none;
  padding: 0;
  margin: 0 0 1.5rem;
  border: 1px solid var(--border);
  border-radius: 10px;
  overflow: hidden;
}
.row {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.6rem 0.9rem;
  border-bottom: 1px solid var(--border);
  font-size: 0.9rem;
  background: var(--card);
}
.row:last-child {
  border-bottom: none;
}
.row:hover {
  background: var(--hover-bg);
}
.fname {
  font-weight: 600;
  min-width: 160px;
}
.msg {
  flex: 1;
  color: var(--muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.when {
  color: var(--muted);
  font-size: 0.82rem;
}
.readme {
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: 1rem 1.25rem;
  background: var(--card);
}
.side-col {
  display: flex;
  flex-direction: column;
  gap: 0.9rem;
}
.card {
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: 0.9rem 1rem;
  background: var(--card);
}
.card h4 {
  margin: 0 0 0.55rem;
  font-size: 0.82rem;
  text-transform: uppercase;
  color: var(--muted);
}
.card p {
  margin: 0;
  font-size: 0.9rem;
}
.tags {
  display: flex;
  gap: 0.4rem;
  flex-wrap: wrap;
  margin-bottom: 0.5rem;
}
.tag {
  padding: 0.15rem 0.55rem;
  border-radius: 999px;
  border: 1px solid var(--border);
  font-size: 0.78rem;
  color: var(--muted);
}
.tag.accent {
  border-color: var(--accent);
  color: var(--accent);
  background: var(--accent-soft);
}
.size,
.social {
  color: var(--muted);
  font-size: 0.85rem;
  margin-top: 0.3rem;
}
@media (max-width: 900px) {
  .repo-body {
    grid-template-columns: 1fr;
  }
}
</style>
