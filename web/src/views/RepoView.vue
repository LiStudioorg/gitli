<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import TopNav from '../components/layout/TopNav.vue'
import CloneDropdown from '../components/repo/CloneDropdown.vue'
import RepoCodeBrowser from '../components/repo/RepoCodeBrowser.vue'
import DeployKeys from '../components/repo/DeployKeys.vue'
import RepoDangerZone from '../components/repo/RepoDangerZone.vue'
import LanguageStats from '../components/repo/LanguageStats.vue'
import SavedReplies from '../components/common/SavedReplies.vue'
import { useShortcuts } from '../composables/useShortcuts'
import api from '../api/client'
import { useAuthStore } from '../stores'
import { REPO_TABS, TABS, isRepoTab, type RepoTab } from '../constants/tabs'
import type { CloneInfo, Issue, LanguageStat, Pull, Repo } from '../stores'

interface RepoResponse {
  repo: Repo
  languages: LanguageStat[]
  license: string
}

interface IssueTemplate {
  name: string
  about: string
  title: string
  labels: string[]
  assignees: string[]
  body: string
  path: string
}

interface PrTemplate {
  body: string
}

const route = useRoute()
useShortcuts(() => `${route.params.owner}/${route.params.repo}`)
const auth = useAuthStore()
const owner = route.params.owner as string
const name = route.params.repo as string

const repo = ref<Repo | null>(null)
const license = ref('')
const languages = ref<LanguageStat[]>([])
const cloneInfo = ref<CloneInfo | null>(null)
const branches = ref<string[]>([])
const selectedBranch = ref('')

const tabs = REPO_TABS
const activeTab = ref<RepoTab>(TABS.code)
const initialTab = route.query.tab
if (isRepoTab(initialTab)) {
  activeTab.value = initialTab
}

const issues = ref<Issue[]>([])
const pulls = ref<Pull[]>([])
const issuesError = ref('')
const pullsError = ref('')
const issuesLoading = ref(false)
const pullsLoading = ref(false)

const newTitle = ref('')
const newBody = ref('')
const createError = ref('')
const creating = ref(false)

const issueTemplates = ref<IssueTemplate[]>([])
const selectedTemplate = ref('')
const templatesLoading = ref(false)
const templatesLoaded = ref(false)

const prBody = ref('')
const prTemplateLoading = ref(false)
const prTemplateLoaded = ref(false)

interface Dependency {
  ecosystem: string
  name: string
  version: string
}

interface DependenciesResponse {
  dependencies: Dependency[] | null
  groups: Record<string, Dependency[]> | null
  counts: Record<string, number> | null
}

const dependencies = ref<Dependency[]>([])
const dependencyGroups = ref<Record<string, Dependency[]>>({})
const dependencyCounts = ref<Record<string, number>>({})
const dependenciesError = ref('')
const dependenciesLoading = ref(false)
const dependenciesLoaded = ref(false)
const dependenciesRefreshing = ref(false)
const dependencyMessage = ref('')

const ecosystemLabels: Record<string, string> = {
  go: 'Go',
  npm: 'npm',
  maven: 'Maven',
  pip: 'Python (pip)',
}

function ecosystemLabel(eco: string): string {
  return ecosystemLabels[eco] ?? eco
}

function dependencyEntries(): [string, Dependency[]][] {
  const groups = dependencyGroups.value
  if (groups && Object.keys(groups).length) {
    return Object.entries(groups)
  }
  const fallback: Record<string, Dependency[]> = {}
  for (const dep of dependencies.value) {
    const key = dep.ecosystem || 'other'
    ;(fallback[key] ??= []).push(dep)
  }
  return Object.entries(fallback)
}

async function loadDependencies() {
  dependenciesLoading.value = true
  dependenciesError.value = ''
  try {
    const { data } = await api.get<DependenciesResponse>(`/repos/${owner}/${name}/dependencies`)
    dependencies.value = data?.dependencies ?? []
    dependencyGroups.value = data?.groups ?? {}
    dependencyCounts.value = data?.counts ?? {}
    dependenciesLoaded.value = true
  } catch {
    dependenciesError.value = '加载依赖失败'
    dependencies.value = []
    dependencyGroups.value = {}
    dependencyCounts.value = {}
  } finally {
    dependenciesLoading.value = false
  }
}

async function refreshDependencies() {
  dependenciesRefreshing.value = true
  dependenciesError.value = ''
  dependencyMessage.value = ''
  try {
    const { data } = await api.post<{ status: string; count: number }>(
      `/repos/${owner}/${name}/dependencies/refresh`,
    )
    dependencyMessage.value = `已刷新，共发现 ${data?.count ?? 0} 个依赖`
    await loadDependencies()
  } catch (e) {
    const status = (e as { response?: { status?: number } })?.response?.status
    if (status === 401 || status === 403) {
      dependenciesError.value = '需要仓库写权限才能刷新依赖'
    } else {
      dependenciesError.value = '刷新依赖失败'
    }
  } finally {
    dependenciesRefreshing.value = false
  }
}

interface WorkflowJob {
  id: number
  runner_id: string
  repo_id: number
  commit_sha: string
  ref: string
  workflow_path: string
  status: string
  started_at: string
  finished_at: string
  exit_code: number
  created_at: string
}

const workflows = ref<WorkflowJob[]>([])
const workflowsError = ref('')
const workflowsLoading = ref(false)
const workflowsLoaded = ref(false)

const selectedJob = ref<WorkflowJob | null>(null)
const logLines = ref<string[]>([])
const logError = ref('')
const logStreaming = ref(false)
let logSource: EventSource | null = null

function shortSha(sha: string): string {
  return sha ? sha.slice(0, 7) : '-'
}

function statusLabel(status: string): string {
  switch (status) {
    case 'pending':
      return '等待中'
    case 'running':
      return '运行中'
    case 'success':
      return '成功'
    case 'failed':
      return '失败'
    case 'cancelled':
      return '已取消'
    default:
      return status || '-'
  }
}

function formatDuration(job: WorkflowJob): string {
  if (!job.started_at || !job.finished_at) return '-'
  const start = new Date(job.started_at).getTime()
  const end = new Date(job.finished_at).getTime()
  if (Number.isNaN(start) || Number.isNaN(end) || end < start) return '-'
  const total = Math.round((end - start) / 1000)
  const m = Math.floor(total / 60)
  const s = total % 60
  return m ? `${m}分${s}秒` : `${s}秒`
}

function canCancel(job: WorkflowJob): boolean {
  return job.status === 'pending' || job.status === 'running'
}

async function loadWorkflows() {
  workflowsLoading.value = true
  workflowsError.value = ''
  try {
    const { data } = await api.get<WorkflowJob[]>(`/repos/${owner}/${name}/workflows`)
    workflows.value = data ?? []
    workflowsLoaded.value = true
  } catch {
    workflowsError.value = '加载工作流失败'
    workflows.value = []
  } finally {
    workflowsLoading.value = false
  }
}

function closeLog() {
  if (logSource) {
    logSource.close()
    logSource = null
  }
  logStreaming.value = false
}

function openLog(job: WorkflowJob) {
  closeLog()
  selectedJob.value = job
  logLines.value = []
  logError.value = ''
  logStreaming.value = true
  const url = `/api/v1/repos/${owner}/${name}/workflows/${job.id}/logs`
  const source = new EventSource(url)
  logSource = source
  source.addEventListener('log', (ev: MessageEvent) => {
    logLines.value.push(String(ev.data))
  })
  source.onerror = () => {
    logError.value = '日志流已断开'
    closeLog()
  }
  source.onopen = () => {
    logError.value = ''
  }
}

async function cancelJob(job: WorkflowJob) {
  if (!window.confirm(`确定要取消该工作流任务 #${job.id} 吗？`)) return
  workflowsError.value = ''
  try {
    await api.post(`/repos/${owner}/${name}/workflows/${job.id}/cancel`)
    await loadWorkflows()
    if (selectedJob.value?.id === job.id) {
      const updated = workflows.value.find((w) => w.id === job.id)
      if (updated) selectedJob.value = updated
    }
  } catch {
    workflowsError.value = '取消工作流失败'
  }
}

async function loadIssues() {
  issuesLoading.value = true
  issuesError.value = ''
  try {
    const { data } = await api.get<Issue[]>('/issues', { params: { owner, repo: name } })
    issues.value = data ?? []
  } catch {
    issuesError.value = '加载工单失败'
    issues.value = []
  } finally {
    issuesLoading.value = false
  }
}

async function loadPulls() {
  pullsLoading.value = true
  pullsError.value = ''
  try {
    const { data } = await api.get<Pull[]>('/pulls', { params: { owner, repo: name } })
    pulls.value = data ?? []
  } catch {
    pullsError.value = '加载合并请求失败'
    pulls.value = []
  } finally {
    pullsLoading.value = false
  }
}

function onInsertReply(body: string) {
  newBody.value = newBody.value ? `${newBody.value}\n${body}` : body
}

async function createIssue() {
  createError.value = ''
  if (!newTitle.value.trim()) {
    createError.value = '标题不能为空'
    return
  }
  creating.value = true
  try {
    await api.post('/issues', {
      owner,
      repo: name,
      title: newTitle.value,
      body: newBody.value,
    })
    newTitle.value = ''
    newBody.value = ''
    await loadIssues()
  } catch {
    createError.value = '创建工单失败'
  } finally {
    creating.value = false
  }
}

async function loadIssueTemplates() {
  if (templatesLoaded.value || templatesLoading.value) return
  templatesLoading.value = true
  try {
    const { data } = await api.get<IssueTemplate[]>(`/repos/${owner}/${name}/issue-templates`)
    issueTemplates.value = data ?? []
    templatesLoaded.value = true
    if (issueTemplates.value.length === 1) {
      applyTemplate(issueTemplates.value[0])
    }
  } catch {
    issueTemplates.value = []
    templatesLoaded.value = true
  } finally {
    templatesLoading.value = false
  }
}

function applyTemplate(tpl: IssueTemplate) {
  if (tpl.title) newTitle.value = tpl.title
  if (tpl.body) newBody.value = tpl.body
}

function onTemplateChange() {
  const tpl = issueTemplates.value.find((t) => t.name === selectedTemplate.value)
  if (tpl) applyTemplate(tpl)
}

async function loadPrTemplate() {
  if (prTemplateLoaded.value || prTemplateLoading.value) return
  prTemplateLoading.value = true
  try {
    const { data } = await api.get<PrTemplate>(`/repos/${owner}/${name}/pr-template`)
    prBody.value = data?.body ?? ''
  } catch {
    prBody.value = ''
  } finally {
    prTemplateLoading.value = false
    prTemplateLoaded.value = true
  }
}

function onRepoUpdated(payload: { name?: string; archived?: boolean }) {
  if (repo.value) {
    if (typeof payload.archived === 'boolean') repo.value.archived = payload.archived
  }
}

interface Project {
  id: number
  name: string
  description: string
}

interface ProjectCard {
  id: number
  column_id: number
  issue_id: number | null
  pull_id: number | null
  title: string
  position: number
}

interface ProjectColumn {
  id: number
  project_id: number
  name: string
  position: number
  cards: ProjectCard[]
}

interface ProjectBoard {
  project: Project
  columns: ProjectColumn[]
}

interface DiscussionCategory {
  id: number
  name: string
  description: string
  is_answerable: boolean
}

interface Discussion {
  id: number
  repo_id: number
  category_id: number
  title: string
  body: string
  author_id: number
  is_answered: boolean
  created_at: string
}

interface DiscussionComment {
  id: number
  discussion_id: number
  author_id: number
  body: string
  is_answer: boolean
  created_at: string
}

interface DiscussionDetail {
  discussion: Discussion
  comments: DiscussionComment[] | null
}

const projects = ref<Project[]>([])
const projectsLoading = ref(false)
const projectsLoaded = ref(false)
const projectsError = ref('')
const projectForm = ref({ name: '', description: '' })
const projectCreating = ref(false)
const projectFormError = ref('')

const board = ref<ProjectBoard | null>(null)
const boardLoading = ref(false)
const boardError = ref('')
const selectedProjectId = ref<number | null>(null)
const cardDrafts = ref<Record<number, string>>({})
const cardAdding = ref<number | null>(null)
const dragCardId = ref<number | null>(null)

async function loadProjects() {
  projectsLoading.value = true
  projectsError.value = ''
  try {
    const { data } = await api.get<Project[]>(`/repos/${owner}/${name}/projects`)
    projects.value = data ?? []
    projectsLoaded.value = true
    if (!selectedProjectId.value && projects.value.length) {
      await selectProject(projects.value[0].id)
    }
  } catch {
    projectsError.value = '加载项目失败'
    projects.value = []
  } finally {
    projectsLoading.value = false
  }
}

async function createProject() {
  projectFormError.value = ''
  if (!projectForm.value.name.trim()) {
    projectFormError.value = '名称不能为空'
    return
  }
  projectCreating.value = true
  try {
    await api.post(`/repos/${owner}/${name}/projects`, {
      name: projectForm.value.name,
      description: projectForm.value.description,
    })
    projectForm.value = { name: '', description: '' }
    projectsLoaded.value = false
    await loadProjects()
  } catch {
    projectFormError.value = '创建项目失败'
  } finally {
    projectCreating.value = false
  }
}

async function selectProject(id: number) {
  selectedProjectId.value = id
  boardLoading.value = true
  boardError.value = ''
  try {
    const { data } = await api.get<ProjectBoard>(`/repos/${owner}/${name}/projects/${id}`)
    board.value = data
    cardDrafts.value = {}
  } catch {
    boardError.value = '加载看板失败'
    board.value = null
  } finally {
    boardLoading.value = false
  }
}

function cardsOf(col: ProjectColumn): ProjectCard[] {
  return [...(col.cards ?? [])].sort((a, b) => a.position - b.position)
}

async function addCard(col: ProjectColumn) {
  if (!board.value) return
  const title = (cardDrafts.value[col.id] ?? '').trim()
  if (!title) return
  cardAdding.value = col.id
  boardError.value = ''
  try {
    await api.post(`/repos/${owner}/${name}/projects/${board.value.project.id}/cards`, {
      column_id: col.id,
      title,
    })
    cardDrafts.value[col.id] = ''
    await selectProject(board.value.project.id)
  } catch {
    boardError.value = '添加卡片失败'
  } finally {
    cardAdding.value = null
  }
}

async function deleteCard(card: ProjectCard) {
  if (!board.value) return
  if (!window.confirm(`确定删除卡片「${card.title}」吗？`)) return
  boardError.value = ''
  try {
    await api.delete(`/repos/${owner}/${name}/projects/${board.value.project.id}/cards/${card.id}`)
    await selectProject(board.value.project.id)
  } catch {
    boardError.value = '删除卡片失败'
  }
}

function onDragStart(card: ProjectCard) {
  dragCardId.value = card.id
}

function onDrop(col: ProjectColumn) {
  const id = dragCardId.value
  dragCardId.value = null
  if (!board.value || id == null) return
  const target = cardsOf(col)
  const index = target.findIndex((c) => c.id === id)
  const position = index >= 0 ? index : target.length
  moveCard(id, col.id, position)
}

async function moveCard(cardId: number, columnId: number, position: number) {
  if (!board.value) return
  boardError.value = ''
  try {
    await api.patch(`/repos/${owner}/${name}/projects/${board.value.project.id}/cards/${cardId}`, {
      column_id: columnId,
      position,
    })
    await selectProject(board.value.project.id)
  } catch {
    boardError.value = '移动卡片失败'
  }
}

const discussions = ref<Discussion[]>([])
const discussionsLoading = ref(false)
const discussionsLoaded = ref(false)
const discussionsError = ref('')
const discussionCategories = ref<DiscussionCategory[]>([])
const categoriesLoaded = ref(false)
const selectedCategory = ref<number | ''>('')
const discussionForm = ref({ title: '', body: '', category_id: '' as number | '' })
const discussionCreating = ref(false)
const discussionFormError = ref('')

const discussionDetail = ref<DiscussionDetail | null>(null)
const discussionDetailLoading = ref(false)
const discussionDetailError = ref('')
const selectedDiscussionId = ref<number | null>(null)
const commentBody = ref('')
const commentAdding = ref(false)
const answerBusy = ref<number | null>(null)

function fmtDate(v: string): string {
  if (!v) return '-'
  const d = new Date(v)
  return Number.isNaN(d.getTime()) ? v : d.toLocaleString()
}

async function loadCategories() {
  if (categoriesLoaded.value) return
  try {
    const { data } = await api.get<DiscussionCategory[]>(
      `/repos/${owner}/${name}/discussions/categories`,
    )
    discussionCategories.value = data ?? []
    categoriesLoaded.value = true
    if (!discussionForm.value.category_id && discussionCategories.value.length) {
      discussionForm.value.category_id = discussionCategories.value[0].id
    }
  } catch {
    discussionCategories.value = []
    categoriesLoaded.value = true
  }
}

async function loadDiscussions() {
  discussionsLoading.value = true
  discussionsError.value = ''
  try {
    const { data } = await api.get<Discussion[]>(`/repos/${owner}/${name}/discussions`, {
      params: selectedCategory.value === '' ? {} : { category_id: selectedCategory.value },
    })
    discussions.value = data ?? []
    discussionsLoaded.value = true
  } catch {
    discussionsError.value = '加载讨论失败'
    discussions.value = []
  } finally {
    discussionsLoading.value = false
  }
}

function filterCategory(id: number | '') {
  selectedCategory.value = id
  loadDiscussions()
}

function categoryLabel(id: number): string {
  return discussionCategories.value.find((c) => c.id === id)?.name ?? `#${id}`
}

function isAnswerable(d: Discussion): boolean {
  return discussionCategories.value.find((c) => c.id === d.category_id)?.is_answerable ?? false
}

async function createDiscussion() {
  discussionFormError.value = ''
  if (!discussionForm.value.title.trim()) {
    discussionFormError.value = '标题不能为空'
    return
  }
  if (discussionForm.value.category_id === '') {
    discussionFormError.value = '请选择分类'
    return
  }
  discussionCreating.value = true
  try {
    await api.post(`/repos/${owner}/${name}/discussions`, {
      title: discussionForm.value.title,
      body: discussionForm.value.body,
      category_id: discussionForm.value.category_id,
    })
    discussionForm.value.title = ''
    discussionForm.value.body = ''
    await loadDiscussions()
  } catch {
    discussionFormError.value = '创建讨论失败'
  } finally {
    discussionCreating.value = false
  }
}

async function openDiscussion(d: Discussion) {
  selectedDiscussionId.value = d.id
  discussionDetailLoading.value = true
  discussionDetailError.value = ''
  commentBody.value = ''
  try {
    const { data } = await api.get<DiscussionDetail>(
      `/repos/${owner}/${name}/discussions/${d.id}`,
    )
    discussionDetail.value = data
  } catch {
    discussionDetailError.value = '加载讨论详情失败'
    discussionDetail.value = null
  } finally {
    discussionDetailLoading.value = false
  }
}

function closeDiscussion() {
  selectedDiscussionId.value = null
  discussionDetail.value = null
  discussionDetailError.value = ''
}

async function addComment() {
  if (!discussionDetail.value) return
  const body = commentBody.value.trim()
  if (!body) return
  commentAdding.value = true
  discussionDetailError.value = ''
  try {
    await api.post(
      `/repos/${owner}/${name}/discussions/${discussionDetail.value.discussion.id}/comments`,
      { body },
    )
    commentBody.value = ''
    await openDiscussion(discussionDetail.value.discussion)
  } catch {
    discussionDetailError.value = '发表评论失败'
  } finally {
    commentAdding.value = false
  }
}

async function markAnswer(comment: DiscussionComment) {
  if (!discussionDetail.value) return
  answerBusy.value = comment.id
  discussionDetailError.value = ''
  try {
    await api.post(
      `/repos/${owner}/${name}/discussions/${discussionDetail.value.discussion.id}/answer`,
      { comment_id: comment.id },
    )
    await openDiscussion(discussionDetail.value.discussion)
  } catch {
    discussionDetailError.value = '标记答案失败'
  } finally {
    answerBusy.value = null
  }
}

function selectTab(tab: RepoTab) {
  activeTab.value = tab
  if (tab === '工单') {
    if (!issues.value.length && !issuesLoading.value) loadIssues()
    loadIssueTemplates()
  }
  if (tab === '合并请求') {
    if (!pulls.value.length && !pullsLoading.value) loadPulls()
    loadPrTemplate()
  }
  if (tab === '工作流' && !workflowsLoaded.value && !workflowsLoading.value) loadWorkflows()
  if (tab === '依赖' && !dependenciesLoaded.value && !dependenciesLoading.value) loadDependencies()
  if (tab === '项目' && !projectsLoaded.value && !projectsLoading.value) loadProjects()
  if (tab === '讨论') {
    loadCategories()
    if (!discussionsLoaded.value && !discussionsLoading.value) loadDiscussions()
  }
  if (tab !== '工作流') closeLog()
}

onMounted(async () => {
  try {
    const { data } = await api.get<RepoResponse>(`/repos/${owner}/${name}`)
    repo.value = data.repo
    languages.value = data.languages ?? []
    license.value = data.license ?? ''
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

onUnmounted(() => {
  closeLog()
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
        <span v-if="repo?.archived" class="badge archived">Archived</span>
        <div class="actions">
          <button class="chip">Watch 1</button>
          <button class="chip accent">Star 1</button>
          <button class="chip">Fork 0</button>
        </div>
      </div>
      <p v-if="repo?.archived" class="archived-hint">此仓库已归档，只读</p>
      <nav class="repo-tabs">
        <button
          v-for="t in tabs"
          :key="t"
          :class="{ active: t === activeTab }"
          @click="selectTab(t)"
        >
          {{ t }}
        </button>
      </nav>
    </header>

    <div class="repo-meta">
      <span>{{ branches.length }} 分支</span>
    </div>

    <div v-if="activeTab === '代码'" class="repo-body">
      <main class="main-col">
        <div class="toolbar-actions">
          <CloneDropdown :owner="owner" :repo="name" :clone-info="cloneInfo" />
        </div>

        <RepoCodeBrowser
          v-model:branch="selectedBranch"
          :owner="owner"
          :repo="name"
          :branches="branches"
        />
      </main>

      <aside class="side-col">
        <section class="card">
          <h4>描述</h4>
          <p>{{ repo?.description || '本仓库为镜像仓库...' }}</p>
          <span v-if="license" class="license-badge">{{ license }}</span>
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

    <div v-else-if="activeTab === '工单'" class="tab-panel">
      <div class="panel-head">
        <h3>工单</h3>
        <button class="chip" @click="loadIssues">刷新</button>
      </div>

      <form v-if="auth.isLoggedIn" class="create-form" @submit.prevent="createIssue">
        <h4>新建工单</h4>
        <select
          v-if="issueTemplates.length"
          v-model="selectedTemplate"
          class="form-input"
          @change="onTemplateChange"
        >
          <option value="">选择模板（可选）</option>
          <option v-for="tpl in issueTemplates" :key="tpl.path" :value="tpl.name">{{ tpl.name }}</option>
        </select>
        <input v-model="newTitle" type="text" placeholder="标题" class="form-input" />
        <SavedReplies @insert="onInsertReply" />
        <textarea v-model="newBody" rows="3" placeholder="描述（可选）" class="form-input" />
        <p v-if="createError" class="error">{{ createError }}</p>
        <button class="chip primary" type="submit" :disabled="creating">
          {{ creating ? '提交中...' : '创建' }}
        </button>
      </form>

      <p v-if="issuesError" class="error">{{ issuesError }}</p>
      <ul v-if="issues.length" class="item-list">
        <li v-for="i in issues" :key="i.ID" class="item-row">
          <span class="num">#{{ i.Number }}</span>
          <span class="item-title">{{ i.Title }}</span>
          <span class="state" :class="i.State">{{ i.State }}</span>
        </li>
      </ul>
      <p v-else-if="!issuesLoading && !issuesError" class="empty">暂无工单</p>
    </div>

    <div v-else-if="activeTab === '合并请求'" class="tab-panel">
      <div class="panel-head">
        <h3>合并请求</h3>
        <button class="chip" @click="loadPulls">刷新</button>
      </div>
      <form v-if="auth.isLoggedIn" class="create-form" @submit.prevent>
        <h4>新建合并请求</h4>
        <textarea
          v-model="prBody"
          rows="5"
          placeholder="描述（可选）"
          class="form-input"
          :disabled="prTemplateLoading"
        />
      </form>
      <p v-if="pullsError" class="error">{{ pullsError }}</p>
      <ul v-if="pulls.length" class="item-list">
        <li v-for="p in pulls" :key="p.ID" class="item-row">
          <span class="num">#{{ p.Number }}</span>
          <span class="item-title">{{ p.Title }}</span>
          <span class="state" :class="p.State">{{ p.State }}</span>
        </li>
      </ul>
      <p v-else-if="!pullsLoading && !pullsError" class="empty">暂无合并请求</p>
    </div>

    <div v-else-if="activeTab === '工作流'" class="tab-panel">
      <div class="panel-head">
        <h3>工作流</h3>
        <button class="chip" @click="loadWorkflows">刷新</button>
      </div>

      <p v-if="workflowsError" class="error">{{ workflowsError }}</p>
      <div v-if="workflowsLoading" class="empty">加载中…</div>
      <ul v-else-if="workflows.length" class="item-list">
        <li
          v-for="j in workflows"
          :key="j.id"
          class="item-row wf-row"
          :class="{ selected: selectedJob?.id === j.id }"
          @click="openLog(j)"
        >
          <span class="state" :class="j.status">{{ statusLabel(j.status) }}</span>
          <span class="mono">{{ shortSha(j.commit_sha) }}</span>
          <span class="wf-ref">{{ j.ref || '-' }}</span>
          <span class="wf-path">{{ j.workflow_path || '-' }}</span>
          <span class="wf-duration">{{ formatDuration(j) }}</span>
          <span class="wf-runner">{{ j.runner_id || '-' }}</span>
          <button
            v-if="canCancel(j)"
            class="chip danger"
            type="button"
            @click.stop="cancelJob(j)"
          >
            取消
          </button>
        </li>
      </ul>
      <p v-else-if="!workflowsError" class="empty">暂无工作流任务</p>

      <div v-if="selectedJob" class="log-panel">
        <div class="log-head">
          <strong>任务 #{{ selectedJob.id }} 日志</strong>
          <span class="log-status">{{ statusLabel(selectedJob.status) }}</span>
          <button class="chip" type="button" @click="closeLog">关闭</button>
        </div>
        <p v-if="logError" class="error">{{ logError }}</p>
        <pre class="log-view">{{ logLines.join('\n') }}</pre>
      </div>
    </div>

    <div v-else-if="activeTab === '依赖'" class="tab-panel">
      <div class="panel-head">
        <h3>依赖</h3>
        <div class="dep-actions">
          <button class="chip" type="button" :disabled="dependenciesLoading" @click="loadDependencies">
            刷新列表
          </button>
          <button
            v-if="auth.isLoggedIn"
            class="chip primary"
            type="button"
            :disabled="dependenciesRefreshing"
            @click="refreshDependencies"
          >
            {{ dependenciesRefreshing ? '扫描中...' : '刷新' }}
          </button>
        </div>
      </div>

      <p v-if="dependenciesError" class="error">{{ dependenciesError }}</p>
      <p v-if="dependencyMessage" class="dep-ok">{{ dependencyMessage }}</p>

      <div v-if="dependenciesLoading" class="empty">加载中…</div>
      <div v-else-if="dependencyEntries().length" class="dep-groups">
        <section v-for="[eco, items] in dependencyEntries()" :key="eco" class="dep-group">
          <header class="dep-group-head">
            <strong>{{ ecosystemLabel(eco) }}</strong>
            <span class="dep-count">{{ dependencyCounts[eco] ?? items.length }} 个</span>
          </header>
          <ul class="dep-list">
            <li v-for="dep in items" :key="`${eco}-${dep.name}-${dep.version}`" class="dep-row">
              <span class="dep-name">{{ dep.name }}</span>
              <span class="dep-version">@ {{ dep.version || '-' }}</span>
            </li>
          </ul>
        </section>
      </div>
      <p v-else-if="!dependenciesError" class="empty">
        未发现依赖清单（支持 go.mod / package.json / pom.xml / requirements.txt）
      </p>
    </div>

    <div v-else-if="activeTab === '项目'" class="tab-panel">
      <div class="panel-head">
        <h3>项目</h3>
        <button class="chip" type="button" :disabled="projectsLoading" @click="loadProjects">
          刷新
        </button>
      </div>

      <p v-if="projectsError" class="error">{{ projectsError }}</p>

      <form v-if="auth.isLoggedIn" class="create-form" @submit.prevent="createProject">
        <h4>新建项目</h4>
        <input v-model="projectForm.name" type="text" placeholder="项目名称" class="form-input" />
        <input v-model="projectForm.description" type="text" placeholder="描述（可选）" class="form-input" />
        <p v-if="projectFormError" class="error">{{ projectFormError }}</p>
        <button class="chip primary" type="submit" :disabled="projectCreating">
          {{ projectCreating ? '创建中...' : '创建项目' }}
        </button>
      </form>

      <div v-if="projectsLoading" class="empty">加载中…</div>
      <template v-else>
        <div v-if="projects.length" class="project-picker">
          <span class="muted-label">选择项目：</span>
          <button
            v-for="p in projects"
            :key="p.id"
            type="button"
            class="chip"
            :class="{ 'chip-active': p.id === selectedProjectId }"
            @click="selectProject(p.id)"
          >
            {{ p.name }}
          </button>
        </div>

        <p v-if="boardError" class="error">{{ boardError }}</p>
        <div v-if="boardLoading" class="empty">加载看板中…</div>
        <div v-else-if="board" class="kanban">
          <section v-for="col in board.columns" :key="col.id" class="kanban-col">
            <header class="kanban-col-head">
              <strong>{{ col.name }}</strong>
              <span class="dep-count">{{ cardsOf(col).length }}</span>
            </header>
            <ul
              class="kanban-cards"
              @dragover.prevent
              @drop="onDrop(col)"
            >
              <li
                v-for="card in cardsOf(col)"
                :key="card.id"
                class="kanban-card"
                draggable="true"
                @dragstart="onDragStart(card)"
              >
                <span class="kanban-card-title">{{ card.title }}</span>
                <button
                  class="card-del"
                  type="button"
                  title="删除卡片"
                  @click="deleteCard(card)"
                >
                  ×
                </button>
              </li>
              <li v-if="!cardsOf(col).length" class="kanban-empty">拖拽卡片到这里</li>
            </ul>
            <form class="kanban-add" @submit.prevent="addCard(col)">
              <input
                v-model="cardDrafts[col.id]"
                type="text"
                placeholder="新增卡片标题"
                class="form-input"
              />
              <button class="chip primary" type="submit" :disabled="cardAdding === col.id">
                添加
              </button>
            </form>
          </section>
        </div>
        <p v-else-if="!projects.length && !projectsError" class="empty">
          暂无项目，请先创建
        </p>
      </template>
    </div>

    <div v-else-if="activeTab === '讨论'" class="tab-panel">
      <div class="panel-head">
        <h3>讨论</h3>
        <button class="chip" type="button" :disabled="discussionsLoading" @click="loadDiscussions">
          刷新
        </button>
      </div>

      <div class="cat-chips">
        <button
          type="button"
          class="chip"
          :class="{ 'chip-active': selectedCategory === '' }"
          @click="filterCategory('')"
        >
          全部
        </button>
        <button
          v-for="c in discussionCategories"
          :key="c.id"
          type="button"
          class="chip"
          :class="{ 'chip-active': selectedCategory === c.id }"
          :title="c.description"
          @click="filterCategory(c.id)"
        >
          {{ c.name }}
        </button>
      </div>

      <form v-if="auth.isLoggedIn" class="create-form" @submit.prevent="createDiscussion">
        <h4>发起讨论</h4>
        <input v-model="discussionForm.title" type="text" placeholder="标题" class="form-input" />
        <select v-model="discussionForm.category_id" class="form-input">
          <option v-if="!discussionCategories.length" :value="''" disabled>暂无分类</option>
          <option v-for="c in discussionCategories" :key="c.id" :value="c.id">{{ c.name }}</option>
        </select>
        <textarea v-model="discussionForm.body" rows="3" placeholder="内容（可选）" class="form-input" />
        <p v-if="discussionFormError" class="error">{{ discussionFormError }}</p>
        <button class="chip primary" type="submit" :disabled="discussionCreating">
          {{ discussionCreating ? '提交中...' : '创建讨论' }}
        </button>
      </form>

      <p v-if="discussionsError" class="error">{{ discussionsError }}</p>
      <div v-if="discussionsLoading" class="empty">加载中…</div>
      <ul v-else-if="discussions.length" class="item-list">
        <li
          v-for="d in discussions"
          :key="d.id"
          class="item-row disc-row"
          :class="{ selected: selectedDiscussionId === d.id }"
          @click="openDiscussion(d)"
        >
          <span class="tag">{{ categoryLabel(d.category_id) }}</span>
          <span class="item-title">{{ d.title }}</span>
          <span v-if="d.is_answered" class="state answered">已回答</span>
          <span class="when">{{ fmtDate(d.created_at) }}</span>
        </li>
      </ul>
      <p v-else-if="!discussionsError" class="empty">暂无讨论</p>

      <div v-if="selectedDiscussionId !== null" class="discussion-detail">
        <p v-if="discussionDetailError" class="error">{{ discussionDetailError }}</p>
        <div v-if="discussionDetailLoading" class="empty">加载详情中…</div>
        <template v-else-if="discussionDetail">
          <div class="disc-head">
            <h4>{{ discussionDetail.discussion.title }}</h4>
            <span class="tag">{{ categoryLabel(discussionDetail.discussion.category_id) }}</span>
            <span v-if="discussionDetail.discussion.is_answered" class="state answered">已回答</span>
            <button class="chip" type="button" @click="closeDiscussion">关闭</button>
          </div>
          <div class="disc-body">{{ discussionDetail.discussion.body || '（无内容）' }}</div>

          <h5 class="disc-comments-title">评论</h5>
          <ul
            v-if="(discussionDetail.comments ?? []).length"
            class="comment-list"
          >
            <li
              v-for="c in discussionDetail.comments ?? []"
              :key="c.id"
              class="comment"
              :class="{ answer: c.is_answer }"
            >
              <div class="comment-head">
                <span class="mono">用户 #{{ c.author_id }}</span>
                <span class="when">{{ fmtDate(c.created_at) }}</span>
                <span v-if="c.is_answer" class="state answered">答案</span>
                <button
                  v-if="isAnswerable(discussionDetail.discussion) && !c.is_answer && auth.isLoggedIn"
                  class="chip"
                  type="button"
                  :disabled="answerBusy === c.id"
                  @click="markAnswer(c)"
                >
                  标记为答案
                </button>
              </div>
              <p class="comment-body">{{ c.body }}</p>
            </li>
          </ul>
          <p v-else class="empty">暂无评论</p>

          <form v-if="auth.isLoggedIn" class="create-form" @submit.prevent="addComment">
            <textarea v-model="commentBody" rows="3" placeholder="写下你的评论..." class="form-input" />
            <button class="chip primary" type="submit" :disabled="commentAdding || !commentBody.trim()">
              {{ commentAdding ? '提交中...' : '发表评论' }}
            </button>
          </form>
        </template>
      </div>
    </div>

    <div v-else-if="activeTab === '设置'" class="tab-panel">
      <DeployKeys :owner="owner" :repo="name" />
      <RepoDangerZone
        v-if="auth.isLoggedIn"
        :owner="owner"
        :repo="name"
        :archived="repo?.archived ?? false"
        @updated="onRepoUpdated"
      />
    </div>

    <div v-else class="tab-panel">
      <p class="placeholder">「{{ activeTab }}」即将推出</p>
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
.toolbar-actions {
  display: flex;
  justify-content: flex-end;
  margin-bottom: 0.5rem;
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
.tab-panel {
  padding: 1.25rem 1.5rem;
}
.panel-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 1rem;
}
.panel-head h3 {
  margin: 0;
}
.create-form {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: 1rem;
  margin-bottom: 1.25rem;
  background: var(--card);
}
.create-form h4 {
  margin: 0 0 0.25rem;
  font-size: 0.9rem;
  color: var(--muted);
}
.form-input {
  padding: 0.5rem 0.7rem;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--color-bg);
  color: var(--color-fg);
  font: inherit;
  resize: vertical;
}
.create-form .chip.primary {
  align-self: flex-start;
}
.item-list {
  list-style: none;
  padding: 0;
  margin: 0;
  border: 1px solid var(--border);
  border-radius: 10px;
  overflow: hidden;
}
.item-row {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.65rem 0.9rem;
  border-bottom: 1px solid var(--border);
  background: var(--card);
  font-size: 0.9rem;
}
.item-row:last-child {
  border-bottom: none;
}
.item-row:hover {
  background: var(--hover-bg);
}
.item-row .num {
  color: var(--muted);
  font-weight: 600;
}
.item-title {
  flex: 1;
}
.state {
  padding: 0.1rem 0.55rem;
  border-radius: 999px;
  border: 1px solid var(--border);
  font-size: 0.75rem;
  color: var(--muted);
}
.state.open {
  border-color: var(--success);
  color: var(--success);
  background: var(--success-soft);
}
.placeholder {
  color: var(--muted);
  text-align: center;
  padding: 3rem 0;
}
.error {
  color: var(--accent);
  font-size: 0.85rem;
}
.dep-actions {
  display: flex;
  gap: 0.5rem;
}
.dep-ok {
  color: var(--success);
  font-size: 0.85rem;
}
.dep-groups {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}
.dep-group {
  border: 1px solid var(--border);
  border-radius: 10px;
  overflow: hidden;
  background: var(--card);
}
.dep-group-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.6rem 0.9rem;
  border-bottom: 1px solid var(--border);
  background: var(--hover-bg);
}
.dep-count {
  color: var(--muted);
  font-size: 0.8rem;
}
.dep-list {
  list-style: none;
  margin: 0;
  padding: 0;
}
.dep-row {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.45rem 0.9rem;
  border-bottom: 1px solid var(--border);
  font-size: 0.88rem;
}
.dep-row:last-child {
  border-bottom: none;
}
.dep-row:hover {
  background: var(--hover-bg);
}
.dep-name {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-weight: 600;
}
.dep-version {
  color: var(--muted);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 0.82rem;
}
.empty {
  color: var(--muted);
}
.wf-row {
  cursor: pointer;
}
.wf-row.selected {
  background: var(--hover-bg);
}
.wf-row .state {
  min-width: 56px;
  text-align: center;
}
.wf-row .state.success {
  border-color: var(--success);
  color: var(--success);
  background: var(--success-soft);
}
.wf-row .state.failed,
.wf-row .state.cancelled {
  border-color: var(--accent);
  color: var(--accent);
  background: var(--accent-soft);
}
.wf-row .state.running,
.wf-row .state.pending {
  border-color: var(--primary);
  color: var(--primary);
  background: var(--primary-soft);
}
.mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 0.82rem;
}
.wf-ref,
.wf-path {
  flex: 1;
  color: var(--muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.wf-duration {
  color: var(--muted);
  font-size: 0.82rem;
}
.wf-runner {
  color: var(--muted);
  font-size: 0.82rem;
}
.chip.danger {
  border-color: var(--accent);
  color: var(--accent);
}
.chip.danger:hover {
  background: var(--accent-soft);
  color: var(--accent);
}
.log-panel {
  margin-top: 1.25rem;
  border: 1px solid var(--border);
  border-radius: 10px;
  overflow: hidden;
  background: var(--card);
}
.log-head {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.6rem 0.9rem;
  border-bottom: 1px solid var(--border);
}
.log-head .chip {
  margin-left: auto;
}
.log-status {
  color: var(--muted);
  font-size: 0.82rem;
}
.chip-active {
  border-color: var(--primary);
  color: var(--primary);
  background: var(--primary-soft);
}
.project-picker {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
  margin-bottom: 1rem;
}
.muted-label {
  color: var(--muted);
  font-size: 0.85rem;
}
.kanban {
  display: flex;
  gap: 0.9rem;
  overflow-x: auto;
  align-items: flex-start;
  padding-bottom: 0.5rem;
}
.kanban-col {
  flex: 0 0 260px;
  border: 1px solid var(--border);
  border-radius: 10px;
  background: var(--card);
  display: flex;
  flex-direction: column;
}
.kanban-col-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.6rem 0.9rem;
  border-bottom: 1px solid var(--border);
  background: var(--hover-bg);
}
.kanban-cards {
  list-style: none;
  margin: 0;
  padding: 0.6rem;
  min-height: 60px;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}
.kanban-card {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 0.7rem;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--color-bg);
  cursor: grab;
  font-size: 0.9rem;
}
.kanban-card:active {
  cursor: grabbing;
}
.kanban-card-title {
  flex: 1;
  word-break: break-word;
}
.card-del {
  border: none;
  background: transparent;
  color: var(--muted);
  font-size: 1rem;
  line-height: 1;
  padding: 0 0.2rem;
}
.card-del:hover {
  color: var(--accent);
}
.kanban-empty {
  color: var(--muted);
  font-size: 0.82rem;
  text-align: center;
  padding: 0.5rem 0;
  border: 1px dashed var(--border);
  border-radius: 8px;
}
.kanban-add {
  display: flex;
  gap: 0.4rem;
  padding: 0.6rem;
  border-top: 1px solid var(--border);
}
.kanban-add .form-input {
  flex: 1;
  min-width: 0;
}
.cat-chips {
  display: flex;
  gap: 0.4rem;
  flex-wrap: wrap;
  margin-bottom: 1rem;
}
.disc-row {
  cursor: pointer;
}
.disc-row.selected {
  background: var(--hover-bg);
}
.state.answered {
  border-color: var(--success);
  color: var(--success);
  background: var(--success-soft);
}
.discussion-detail {
  margin-top: 1.25rem;
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: 1rem;
  background: var(--card);
}
.disc-head {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  flex-wrap: wrap;
}
.disc-head h4 {
  margin: 0;
  flex: 1;
}
.disc-body {
  white-space: pre-wrap;
  margin: 0.9rem 0;
  font-size: 0.92rem;
  line-height: 1.6;
}
.disc-comments-title {
  margin: 1rem 0 0.5rem;
  font-size: 0.85rem;
  text-transform: uppercase;
  color: var(--muted);
}
.comment-list {
  list-style: none;
  margin: 0 0 1rem;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 0.6rem;
}
.comment {
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 0.6rem 0.8rem;
  background: var(--color-bg);
}
.comment.answer {
  border-color: var(--success);
}
.comment-head {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  flex-wrap: wrap;
  color: var(--muted);
  font-size: 0.8rem;
}
.comment-head .chip {
  margin-left: auto;
  padding: 0.25rem 0.6rem;
  font-size: 0.8rem;
}
.comment-body {
  margin: 0.4rem 0 0;
  white-space: pre-wrap;
  font-size: 0.9rem;
}
.log-view {
  margin: 0;
  padding: 0.75rem 0.9rem;
  max-height: 380px;
  overflow: auto;
  background: var(--color-bg);
  color: var(--color-fg);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 0.82rem;
  line-height: 1.5;
  white-space: pre-wrap;
  word-break: break-all;
}
</style>
