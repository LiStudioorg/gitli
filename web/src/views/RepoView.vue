<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import TopNav from '../components/layout/TopNav.vue'
import CloneDropdown from '../components/repo/CloneDropdown.vue'
import RepoCodeBrowser from '../components/repo/RepoCodeBrowser.vue'
import RepoReleases from '../components/repo/RepoReleases.vue'
import RepoActivity from '../components/repo/RepoActivity.vue'
import RepoWiki from '../components/repo/RepoWiki.vue'
import DeployKeys from '../components/repo/DeployKeys.vue'
import RepoDangerZone from '../components/repo/RepoDangerZone.vue'
import LanguageStats from '../components/repo/LanguageStats.vue'
import SavedReplies from '../components/common/SavedReplies.vue'
import SkeletonBlock from '../components/common/SkeletonBlock.vue'
import EmptyState from '../components/common/EmptyState.vue'
import api from '../api/client'
import { useAuthStore } from '../stores'
import { REPO_TABS, TABS, isRepoTab, type RepoTab } from '../constants/tabs'
import type { CloneInfo, Issue, LanguageStat, Pull, Repo } from '../stores'

interface RepoResponse {
  repo: Repo
  languages: LanguageStat[]
  license: string
  is_empty?: boolean
  has_readme?: boolean
  readme_filename?: string
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
const auth = useAuthStore()
const owner = route.params.owner as string
const name = route.params.repo as string

const repo = ref<Repo | null>(null)
const readmeFilename = ref('')
const repoIsEmpty = ref(false)
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
const issuesLoaded = ref(false)
const pullsLoaded = ref(false)
const issuesError = ref('')
const pullsError = ref('')
const issueFilter = ref<'open' | 'closed' | 'all'>('open')
const issueFormOpen = ref(false)
const issueTitleTouched = ref(false)
const pullFilter = ref<'open' | 'closed' | 'all'>('open')
const diffOpen = ref<number | null>(null)
const diffText = ref('')
const diffLoading = ref(false)
const diffMessage = ref('')
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

const filteredIssues = computed(() =>
  issues.value.filter((i) => issueFilter.value === 'all' || i.State === issueFilter.value),
)
const filteredPulls = computed(() =>
  pulls.value.filter((p) => pullFilter.value === 'all' || p.State === pullFilter.value),
)

function formatSize(kb: number): string {
  if (!kb) return '0 B'
  if (kb < 1024) return `${kb} KB`
  if (kb < 1024 * 1024) return `${(kb / 1024).toFixed(1)} MB`
  return `${(kb / 1024 / 1024).toFixed(1)} GB`
}

function relTime(raw: string): string {
  const d = new Date(raw.endsWith('Z') || raw.includes('+') ? raw : raw.replace(' ', 'T') + 'Z')
  if (Number.isNaN(d.getTime())) return raw
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

// 行级配色：新增绿、删除红、hunk 头蓝、文件头加粗。
function diffLineClass(line: string): string {
  if (line.startsWith('+++') || line.startsWith('---') || line.startsWith('diff ')) return 'meta'
  if (line.startsWith('@@')) return 'hunk'
  if (line.startsWith('+')) return 'add'
  if (line.startsWith('-')) return 'del'
  return ''
}

async function toggleDiff(p: Pull) {
  if (diffOpen.value === p.ID) {
    diffOpen.value = null
    return
  }
  diffOpen.value = p.ID
  diffText.value = ''
  diffMessage.value = ''
  diffLoading.value = true
  try {
    const { data } = await api.get(`/repos/${owner}/${name}/diff`, {
      params: { base: p.BaseBranch, head: p.HeadBranch },
      responseType: 'text',
      transformResponse: [(v) => v],
    })
    if (typeof data === 'string' && data.trim().startsWith('{')) {
      const parsed = JSON.parse(data)
      if (parsed.too_large) {
        diffMessage.value = parsed.message || '变更内容过大，请在本地查看或下载'
      } else {
        diffText.value = data
      }
    } else {
      diffText.value = data || ''
    }
  } catch {
    diffMessage.value = '无法加载变更内容'
  } finally {
    diffLoading.value = false
  }
}

async function loadIssues() {
  issuesLoading.value = true
  issuesError.value = ''
  try {
    const { data } = await api.get<Issue[]>('/issues', { params: { owner, repo: name } })
    issues.value = data ?? []
    issuesLoaded.value = true
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
    pullsLoaded.value = true
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
    if (!issuesLoaded.value && !issuesLoading.value) loadIssues()
    loadIssueTemplates()
  }
  if (tab === '合并请求') {
    if (!pullsLoaded.value && !pullsLoading.value) loadPulls()
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
    readmeFilename.value = data.readme_filename ?? ''
    repoIsEmpty.value = data.is_empty === true
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
  <div class="flex min-h-screen flex-col bg-[#f6f8fa] dark:bg-[#0d1117]">
    <TopNav variant="app" />

    <header class="bg-white dark:bg-[#161b22]">
      <div class="mx-auto w-full max-w-[1280px] px-4 pt-4 md:px-6">
        <div class="flex flex-wrap items-center gap-2">
          <svg class="h-4 w-4 shrink-0 text-[#59636e]" viewBox="0 0 16 16" fill="currentColor" aria-hidden="true">
            <path d="M2 2.5A2.5 2.5 0 0 1 4.5 0h8.75a.75.75 0 0 1 .75.75v12.5a.75.75 0 0 1-.75.75h-2.5a.75.75 0 0 1 0-1.5h1.75v-2h-8a1 1 0 0 0-.714 1.7.75.75 0 1 1-1.072 1.05A2.495 2.495 0 0 1 2 11.5Zm10.5-1h-8a1 1 0 0 0-1 1v6.708A2.486 2.486 0 0 1 4.5 9h8ZM5 12.25a.25.25 0 0 1 .25-.25h3.5a.25.25 0 0 1 .25.25v3.25a.25.25 0 0 1-.4.2l-1.45-1.087a.249.249 0 0 0-.3 0L5.4 15.7a.25.25 0 0 1-.4-.2Z" />
          </svg>
          <h2 class="min-w-0 truncate text-lg font-semibold text-[#1f2328] dark:text-[#e6edf3] md:text-xl">
            <RouterLink :to="`/${owner}`" class="text-[#ea580c] hover:underline">{{ owner }}</RouterLink>
            <span class="mx-1 font-normal text-[#59636e]">/</span>
            <span class="break-all">{{ name }}</span>
          </h2>
          <span v-if="repo?.is_private" class="rounded-full border border-[#d1d9e0] px-2 py-0.5 text-xs text-[#59636e] dark:border-[#30363d]">私有</span>
          <span v-if="repo?.archived" class="rounded-full border border-[#d1d9e0] px-2 py-0.5 text-xs text-[#59636e] dark:border-[#30363d]">已归档</span>

          <!-- Watch / Fork / Star：GitHub 分体按钮 -->
          <div class="ml-auto flex flex-wrap items-center gap-2">
            <div class="inline-flex overflow-hidden rounded-md border border-[#d1d9e0] dark:border-[#30363d]">
              <button type="button" class="px-2.5 py-1 text-xs font-medium text-[#1f2328] hover:bg-[#f6f8fa] dark:text-[#e6edf3] dark:hover:bg-[#21262d]">Fork</button>
              <span class="flex items-center border-l border-[#d1d9e0] px-2 text-xs text-[#59636e] dark:border-[#30363d]">{{ repo?.ForksCount ?? 0 }}</span>
            </div>
            <div class="inline-flex overflow-hidden rounded-md border border-[#d1d9e0] dark:border-[#30363d]">
              <button type="button" class="px-2.5 py-1 text-xs font-medium text-[#1f2328] hover:bg-[#f6f8fa] dark:text-[#e6edf3] dark:hover:bg-[#21262d]">Star</button>
              <span class="flex items-center border-l border-[#d1d9e0] px-2 text-xs text-[#59636e] dark:border-[#30363d]">{{ repo?.StarsCount ?? 0 }}</span>
            </div>
          </div>
        </div>
        <p v-if="repo?.archived" class="mt-2 rounded-md border border-[#ff818266] bg-[#fff8c5] px-3 py-1.5 text-xs text-[#9a6700]">
          此仓库已归档，只读
        </p>
        <nav class="mt-4 flex gap-1 overflow-x-auto">
          <button
            v-for="t in tabs"
            :key="t"
            type="button"
            class="h-12 shrink-0 whitespace-nowrap border-b-2 px-3 text-sm transition-colors"
            :class="t === activeTab
              ? 'border-[#ea580c] font-semibold text-[#1f2328] dark:text-[#e6edf3]'
              : 'border-transparent text-[#59636e] hover:rounded-t-md hover:border-[#d1d9e0] hover:bg-[#f6f8fa] dark:hover:bg-[#21262d]'"
            @click="selectTab(t)"
          >
            {{ t }}
          </button>
        </nav>
      </div>
    </header>

    <div v-if="activeTab === '代码'" class="grid min-w-0 grid-cols-1 items-start gap-6 p-4 md:p-6 lg:grid-cols-[minmax(0,1fr)_220px]">
      <main class="min-w-0">
        <div class="mb-3 flex justify-end">
          <CloneDropdown :owner="owner" :repo="name" :clone-info="cloneInfo" />
        </div>

        <RepoCodeBrowser
          v-model:branch="selectedBranch"
          :owner="owner"
          :repo="name"
          :branches="branches"
          :readme-filename="readmeFilename"
        />
      </main>

      <aside class="flex min-w-0 flex-col gap-4">
        <section class="rounded-md border border-[#d1d9e0] bg-white p-4 dark:border-[#30363d] dark:bg-[#161b22]">
          <h3 class="mb-2 text-base font-semibold text-[#1f2328] dark:text-[#e6edf3]">About</h3>
          <p class="break-words text-sm text-[#1f2328] dark:text-[#e6edf3]">
            {{ repo?.description || '暂无描述' }}
          </p>
          <div v-if="license" class="mt-3 flex items-center gap-1.5 text-sm text-[#59636e]">
            <span aria-hidden="true">⚖</span>{{ license }}
          </div>
          <div v-if="repo && (repo.SizeKB ?? 0) > 0" class="mt-2 flex items-center gap-1.5 text-sm text-[#59636e]">
            <span aria-hidden="true">◔</span>{{ formatSize(repo.SizeKB ?? 0) }}
          </div>
          <div v-if="languages.length" class="mt-4 border-t border-[#d1d9e0] pt-3 dark:border-[#30363d]">
            <h4 class="mb-2 text-xs font-semibold uppercase tracking-wide text-[#59636e]">语言</h4>
            <LanguageStats :languages="languages" />
          </div>
        </section>
      </aside>
    </div>

    <div v-else-if="activeTab === '工单'" class="p-4 md:p-6">
      <div class="mb-4 flex flex-wrap items-center justify-between gap-3">
        <h3 class="text-base font-semibold text-[#1f2328] dark:text-[#e6edf3]">工单</h3>
        <div class="flex flex-wrap items-center gap-2">
          <div class="inline-flex overflow-hidden rounded-md border border-[#d1d9e0] dark:border-[#30363d]" role="group" aria-label="按状态筛选工单">
            <button
              v-for="f in [{ k: 'open', t: `开启 ${issues.filter((i) => i.State === 'open').length}` }, { k: 'closed', t: `已关闭 ${issues.filter((i) => i.State !== 'open').length}` }, { k: 'all', t: '全部' }]"
              :key="f.k"
              type="button"
              class="px-3 py-1.5 text-xs"
              :class="issueFilter === f.k ? 'bg-[#ea580c]/10 font-semibold text-[#ea580c]' : 'text-[#59636e] hover:bg-[#f6f8fa] dark:hover:bg-[#21262d]'"
              @click="issueFilter = f.k as 'open' | 'closed' | 'all'"
            >
              {{ f.t }}
            </button>
          </div>
          <button
            class="h-8 rounded-md border border-[#d1d9e0] px-3 text-xs text-[#1f2328] hover:bg-[#f6f8fa] dark:border-[#30363d] dark:text-[#e6edf3] dark:hover:bg-[#21262d]"
            type="button"
            @click="loadIssues"
          >
            刷新
          </button>
          <button
            v-if="auth.isLoggedIn"
            class="h-8 rounded-md bg-[#2da44e] px-3 text-xs font-semibold text-white hover:bg-[#2c974b]"
            type="button"
            @click="issueFormOpen = !issueFormOpen"
          >
            {{ issueFormOpen ? '取消' : '新建工单' }}
          </button>
        </div>
      </div>

      <form
        v-if="auth.isLoggedIn && issueFormOpen"
        class="mb-4 rounded-md border border-[#d1d9e0] bg-white dark:border-[#30363d] dark:bg-[#161b22]"
        @submit.prevent="createIssue"
      >
        <header class="border-b border-[#d1d9e0] px-4 py-2.5 text-sm font-semibold text-[#1f2328] dark:border-[#30363d] dark:text-[#e6edf3]">
          新建工单
        </header>
        <div class="flex flex-col gap-4 p-4">
          <div class="flex flex-col gap-1.5">
            <label for="issue-title" class="text-sm font-semibold text-[#1f2328] dark:text-[#e6edf3]">
              标题 <span class="text-[#cf222e]" aria-hidden="true">*</span>
            </label>
            <input
              id="issue-title"
              v-model="newTitle"
              class="h-9 rounded-md border border-[#d1d9e0] bg-[#f6f8fa] px-3 text-sm text-[#1f2328] outline-none placeholder:text-[#59636e] focus:border-[#0969da] focus:bg-white focus:ring-2 focus:ring-[#0969da]/25 dark:border-[#30363d] dark:bg-[#0d1117] dark:text-[#e6edf3]"
              :class="{ '!border-[#cf222e]': issueTitleTouched && !newTitle.trim() }"
              type="text"
              placeholder="简要描述问题"
              @blur="issueTitleTouched = true"
            />
            <span v-if="issueTitleTouched && !newTitle.trim()" class="text-xs text-[#cf222e]">标题不能为空</span>
          </div>

          <div v-if="issueTemplates.length" class="flex flex-col gap-1.5">
            <label for="issue-template" class="text-sm font-semibold text-[#1f2328] dark:text-[#e6edf3]">模板</label>
            <select
              id="issue-template"
              v-model="selectedTemplate"
              class="h-9 rounded-md border border-[#d1d9e0] bg-white px-2 text-sm text-[#1f2328] dark:border-[#30363d] dark:bg-[#161b22] dark:text-[#e6edf3]"
              @change="onTemplateChange"
            >
              <option value="">选择模板（可选）</option>
              <option v-for="tpl in issueTemplates" :key="tpl.path" :value="tpl.name">{{ tpl.name }}</option>
            </select>
          </div>

          <div class="flex flex-col gap-1.5">
            <div class="flex items-center justify-between">
              <label for="issue-body" class="text-sm font-semibold text-[#1f2328] dark:text-[#e6edf3]">描述</label>
              <SavedReplies @insert="onInsertReply" />
            </div>
            <textarea
              id="issue-body"
              v-model="newBody"
              class="rounded-md border border-[#d1d9e0] bg-white px-3 py-2 text-sm text-[#1f2328] outline-none placeholder:text-[#59636e] focus:border-[#ea580c] focus:ring-2 focus:ring-[#ea580c]/25 dark:border-[#30363d] dark:bg-[#0d1117] dark:text-[#e6edf3]"
              rows="6"
              placeholder="补充细节、复现步骤（支持 Markdown）"
            />
            <span class="text-xs text-[#59636e]">支持 Markdown、代码块与任务列表</span>
          </div>

          <p v-if="createError" class="rounded-md border border-[#ff818266] bg-[#ffebe9] px-3 py-2 text-sm text-[#cf222e]">
            {{ createError }}
          </p>

          <div class="flex gap-2">
            <button
              class="inline-flex h-8 items-center gap-2 rounded-md bg-[#2da44e] px-3 text-sm font-semibold text-white hover:bg-[#2c974b] disabled:opacity-50"
              type="submit"
              :disabled="creating"
            >
              <span v-if="creating" class="h-3.5 w-3.5 animate-spin rounded-full border-2 border-white/40 border-t-white" aria-hidden="true" />
              {{ creating ? '提交中…' : '创建工单' }}
            </button>
            <button
              class="h-8 rounded-md border border-[#d1d9e0] px-3 text-sm text-[#1f2328] hover:bg-[#f6f8fa] dark:border-[#30363d] dark:text-[#e6edf3] dark:hover:bg-[#21262d]"
              type="button"
              @click="issueFormOpen = false"
            >
              取消
            </button>
          </div>
        </div>
      </form>

      <div v-if="issuesLoading" class="flex flex-col gap-2">
        <div v-for="i in 4" :key="i" class="rounded-md border border-[#d1d9e0] bg-white p-4 dark:border-[#30363d] dark:bg-[#161b22]">
          <SkeletonBlock :lines="2" />
        </div>
      </div>
      <p v-else-if="issuesError" class="rounded-md border border-[#ff818266] bg-[#ffebe9] px-3 py-2 text-sm text-[#cf222e]">{{ issuesError }}</p>
      <EmptyState
        v-else-if="!filteredIssues.length"
        icon="◎"
        :title="issues.length ? '该状态下没有工单' : '还没有工单'"
        :description="issues.length ? '切换上方筛选条件查看其它状态的工单。' : '工单用于记录缺陷、功能建议与讨论。'"
        action-label="创建第一个工单"
        :show-action="auth.isLoggedIn && !issues.length"
        @action="issueFormOpen = true"
      />
      <ul v-else class="divide-y divide-[#d1d9e0] overflow-hidden rounded-md border border-[#d1d9e0] bg-white dark:divide-[#30363d] dark:border-[#30363d] dark:bg-[#161b22]">
        <li v-for="i in filteredIssues" :key="i.ID" class="flex items-center gap-3 px-4 py-3 text-sm">
          <span
            class="h-2.5 w-2.5 shrink-0 rounded-full"
            :class="i.State === 'open' ? 'bg-[#10b981]' : 'bg-[#ef4444]'"
            :title="i.State === 'open' ? '开启' : '已关闭'"
          />
          <div class="min-w-0 flex-1">
            <RouterLink
              class="block truncate font-semibold text-[#1f2328] hover:text-[#ea580c] dark:text-[#e6edf3]"
              :to="`/${owner}/${name}?tab=工单`"
            >
              #{{ i.Number }} {{ i.Title }}
            </RouterLink>
            <span class="block truncate text-xs text-[#59636e]">
              <template v-if="i.AuthorName">{{ i.AuthorName }} 创建于 </template>{{ relTime(i.CreatedAt) }}
            </span>
          </div>
          <span
            class="shrink-0 rounded-full px-2 py-0.5 text-xs font-medium"
            :class="i.State === 'open' ? 'bg-[#10b981]/15 text-[#10b981]' : 'bg-[#ef4444]/15 text-[#ef4444]'"
          >
            {{ i.State === 'open' ? '开启' : '已关闭' }}
          </span>
        </li>
      </ul>
    </div>

    <div v-else-if="activeTab === '合并请求'" class="p-4 md:p-6">
      <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
        <h3>合并请求</h3>
        <div class="flex flex-wrap items-center gap-2">
          <div class="inline-flex overflow-hidden rounded-md border border-[#d1d9e0] dark:border-[#30363d]" role="group" aria-label="按状态筛选合并请求">
            <button type="button" :class="{ active: pullFilter === 'open' }" @click="pullFilter = 'open'">
              开启 {{ pulls.filter((p) => p.State === 'open' && !p.Merged).length }}
            </button>
            <button type="button" :class="{ active: pullFilter === 'closed' }" @click="pullFilter = 'closed'">
              已合并 {{ pulls.filter((p) => p.Merged).length }} / 其他
            </button>
            <button type="button" :class="{ active: pullFilter === 'all' }" @click="pullFilter = 'all'">全部</button>
          </div>
          <button class="inline-flex h-8 items-center rounded-md border border-[#d1d9e0] px-3 text-sm text-[#1f2328] hover:bg-[#f6f8fa] dark:border-[#3d444d] dark:text-[#e6edf3] dark:hover:bg-[#161b22]" type="button" @click="loadPulls">刷新</button>
        </div>
      </div>

      <p v-if="pullsError" class="rounded-md border border-[#ff818266] bg-[#ffebe9] px-3 py-2 text-sm text-[#cf222e]">{{ pullsError }}</p>
      <div v-if="pullsLoading" class="list-skeleton">
        <SkeletonBlock v-for="i in 3" :key="i" :lines="2" />
      </div>
      <EmptyState
        v-else-if="!filteredPulls.length"
        icon="⇄"
        :title="pulls.length ? '该状态下没有合并请求' : '还没有合并请求'"
        :description="pulls.length ? '切换上方筛选条件查看其它状态。' : '从功能分支向主干发起合并请求，即可在此评审与合并。'"
        :action-label="pulls.length ? '' : '查看分支与克隆方式'"
        :show-action="!pulls.length"
        @action="activeTab = '代码'"
      />
      <ul v-else class="divide-y divide-[#d1d9e0] rounded-md border border-[#d1d9e0] bg-white dark:divide-[#30363d] dark:border-[#30363d] dark:bg-[#161b22]">
        <li v-for="p in filteredPulls" :key="p.ID" class="item-wrap">
          <div class="flex items-center gap-3 px-4 py-3 text-sm">
            <span class="state-dot" :class="p.Merged ? 'merged' : p.State === 'open' ? 'open' : 'closed'" />
            <div class="item-main">
              <button class="truncate text-left font-semibold text-[#1f2328] hover:text-[#ea580c] dark:text-[#e6edf3]" type="button" @click="toggleDiff(p)">
                #{{ p.Number }} {{ p.Title }}
              </button>
              <span class="text-xs text-[#59636e]">
                {{ p.HeadBranch }} → {{ p.BaseBranch }}
                <template v-if="p.AuthorName"> · {{ p.AuthorName }}</template>
                · {{ relTime(p.CreatedAt) }}
              </span>
            </div>
            <span class="shrink-0 rounded-full px-2 py-0.5 text-xs font-medium" :class="p.Merged ? 'bg-[#8b5cf6]/15 text-[#8b5cf6]' : p.State === 'open' ? 'bg-[#10b981]/15 text-[#10b981]' : 'bg-[#ef4444]/15 text-[#ef4444]'">
              {{ p.Merged ? '已合并' : p.State === 'open' ? '开启' : '已关闭' }}
            </span>
            <button class="inline-flex h-8 items-center rounded-md border border-[#d1d9e0] px-3 text-sm text-[#1f2328] hover:bg-[#f6f8fa] dark:border-[#3d444d] dark:text-[#e6edf3] dark:hover:bg-[#161b22]" type="button" @click="toggleDiff(p)">
              {{ diffOpen === p.ID ? '收起' : '查看变更' }}
            </button>
          </div>
          <div v-if="diffOpen === p.ID" class="diff">
            <p v-if="diffLoading" class="text-sm text-[#59636e]">正在加载变更…</p>
            <p v-else-if="diffMessage" class="rounded-md border border-[#d1d9e0] bg-[#f6f8fa] px-3 py-2 text-sm dark:border-[#3d444d] dark:bg-[#161b22]">{{ diffMessage }}</p>
            <pre v-else-if="diffText" class="mt-2 max-h-[420px] overflow-auto rounded-md bg-[#f6f8fa] p-3 font-mono text-xs leading-relaxed dark:bg-[#0d1117]"><code><span
              v-for="(line, idx) in diffText.split('\n')"
              :key="idx"
              class="diff-line"
              :class="diffLineClass(line)"
            >{{ line || ' ' }}
</span></code></pre>
            <p v-else class="text-sm text-[#59636e]">该合并请求没有文本变更。</p>
          </div>
        </li>
      </ul>
    </div>

    <div v-else-if="activeTab === '工作流'" class="p-4 md:p-6">
      <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
        <h3>工作流</h3>
        <button class="rounded-md border border-[#d1d9e0] px-3 py-1.5 text-sm text-[#1f2328] hover:bg-[#f6f8fa] dark:border-[#30363d] dark:text-[#e6edf3] dark:hover:bg-[#21262d]" @click="loadWorkflows">刷新</button>
      </div>

      <p v-if="workflowsError" class="error">{{ workflowsError }}</p>
      <div v-if="workflowsLoading" class="py-8 text-center text-sm text-[#59636e]">加载中…</div>
      <ul v-else-if="workflows.length" class="divide-y divide-[#d1d9e0] rounded-md border border-[#d1d9e0] bg-white dark:divide-[#30363d] dark:border-[#30363d] dark:bg-[#161b22]">
        <li
          v-for="j in workflows"
          :key="j.id"
          class="item-row wf-row"
          :class="{ selected: selectedJob?.id === j.id }"
          @click="openLog(j)"
        >
          <span class="shrink-0 rounded-full bg-[#f6f8fa] px-2 py-0.5 text-xs font-medium text-[#59636e] dark:bg-[#21262d]">{{ statusLabel(j.status) }}</span>
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
      <p v-else-if="!workflowsError" class="py-8 text-center text-sm text-[#59636e]">暂无工作流任务</p>

      <div v-if="selectedJob" class="log-panel">
        <div class="log-head">
          <strong>任务 #{{ selectedJob.id }} 日志</strong>
          <span class="log-status">{{ statusLabel(selectedJob.status) }}</span>
          <button class="rounded-md border border-[#d1d9e0] px-3 py-1.5 text-sm text-[#1f2328] hover:bg-[#f6f8fa] dark:border-[#30363d] dark:text-[#e6edf3] dark:hover:bg-[#21262d]" type="button" @click="closeLog">关闭</button>
        </div>
        <p v-if="logError" class="error">{{ logError }}</p>
        <pre class="log-view">{{ logLines.join('\n') }}</pre>
      </div>
    </div>

    <div v-else-if="activeTab === '依赖'" class="p-4 md:p-6">
      <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
        <h3>依赖</h3>
        <div class="dep-actions">
          <button class="rounded-md border border-[#d1d9e0] px-3 py-1.5 text-sm text-[#1f2328] hover:bg-[#f6f8fa] dark:border-[#30363d] dark:text-[#e6edf3] dark:hover:bg-[#21262d]" type="button" :disabled="dependenciesLoading" @click="loadDependencies">
            刷新列表
          </button>
          <button
            v-if="auth.isLoggedIn"
            class="rounded-md bg-[#ea580c] px-3 py-1.5 text-sm font-semibold text-white hover:bg-[#c2410c] disabled:opacity-50"
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

      <div v-if="dependenciesLoading" class="py-8 text-center text-sm text-[#59636e]">加载中…</div>
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
      <p v-else-if="!dependenciesError" class="py-8 text-center text-sm text-[#59636e]">
        未发现依赖清单（支持 go.mod / package.json / pom.xml / requirements.txt）
      </p>
    </div>

    <div v-else-if="activeTab === '项目'" class="p-4 md:p-6">
      <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
        <h3>项目</h3>
        <button class="rounded-md border border-[#d1d9e0] px-3 py-1.5 text-sm text-[#1f2328] hover:bg-[#f6f8fa] dark:border-[#30363d] dark:text-[#e6edf3] dark:hover:bg-[#21262d]" type="button" :disabled="projectsLoading" @click="loadProjects">
          刷新
        </button>
      </div>

      <p v-if="projectsError" class="error">{{ projectsError }}</p>

      <form v-if="auth.isLoggedIn" class="mb-4" @submit.prevent="createProject">
        <h4>新建项目</h4>
        <input v-model="projectForm.name" type="text" placeholder="项目名称" class="form-input" />
        <input v-model="projectForm.description" type="text" placeholder="描述（可选）" class="form-input" />
        <p v-if="projectFormError" class="error">{{ projectFormError }}</p>
        <button class="rounded-md bg-[#ea580c] px-3 py-1.5 text-sm font-semibold text-white hover:bg-[#c2410c] disabled:opacity-50" type="submit" :disabled="projectCreating">
          {{ projectCreating ? '创建中...' : '创建项目' }}
        </button>
      </form>

      <div v-if="projectsLoading" class="py-8 text-center text-sm text-[#59636e]">加载中…</div>
      <template v-else>
        <div v-if="projects.length" class="project-picker">
          <span class="muted-label">选择项目：</span>
          <button
            v-for="p in projects"
            :key="p.id"
            type="button"
            class="rounded-md border border-[#d1d9e0] px-3 py-1.5 text-sm text-[#1f2328] hover:bg-[#f6f8fa] dark:border-[#30363d] dark:text-[#e6edf3] dark:hover:bg-[#21262d]"
            :class="{ 'chip-active': p.id === selectedProjectId }"
            @click="selectProject(p.id)"
          >
            {{ p.name }}
          </button>
        </div>

        <p v-if="boardError" class="error">{{ boardError }}</p>
        <div v-if="boardLoading" class="py-8 text-center text-sm text-[#59636e]">加载看板中…</div>
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
              <button class="rounded-md bg-[#ea580c] px-3 py-1.5 text-sm font-semibold text-white hover:bg-[#c2410c] disabled:opacity-50" type="submit" :disabled="cardAdding === col.id">
                添加
              </button>
            </form>
          </section>
        </div>
        <EmptyState
          v-else-if="!projects.length && !projectsError"
          icon="🗂"
          title="还没有项目看板"
          description="用看板把工单和合并请求按阶段组织起来，便于跟踪进度。"
          action-label="创建项目看板"
          :show-action="auth.isLoggedIn"
          @action="activeTab = '项目'"
        />
      </template>
    </div>

    <div v-else-if="activeTab === '讨论'" class="p-4 md:p-6">
      <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
        <h3>讨论</h3>
        <button class="rounded-md border border-[#d1d9e0] px-3 py-1.5 text-sm text-[#1f2328] hover:bg-[#f6f8fa] dark:border-[#30363d] dark:text-[#e6edf3] dark:hover:bg-[#21262d]" type="button" :disabled="discussionsLoading" @click="loadDiscussions">
          刷新
        </button>
      </div>

      <div class="cat-chips">
        <button
          type="button"
          class="rounded-md border border-[#d1d9e0] px-3 py-1.5 text-sm text-[#1f2328] hover:bg-[#f6f8fa] dark:border-[#30363d] dark:text-[#e6edf3] dark:hover:bg-[#21262d]"
          :class="{ 'chip-active': selectedCategory === '' }"
          @click="filterCategory('')"
        >
          全部
        </button>
        <button
          v-for="c in discussionCategories"
          :key="c.id"
          type="button"
          class="rounded-md border border-[#d1d9e0] px-3 py-1.5 text-sm text-[#1f2328] hover:bg-[#f6f8fa] dark:border-[#30363d] dark:text-[#e6edf3] dark:hover:bg-[#21262d]"
          :class="{ 'chip-active': selectedCategory === c.id }"
          :title="c.description"
          @click="filterCategory(c.id)"
        >
          {{ c.name }}
        </button>
      </div>

      <form v-if="auth.isLoggedIn" class="mb-4" @submit.prevent="createDiscussion">
        <h4>发起讨论</h4>
        <input v-model="discussionForm.title" type="text" placeholder="标题" class="form-input" />
        <select v-model="discussionForm.category_id" class="form-input">
          <option v-if="!discussionCategories.length" :value="''" disabled>暂无分类</option>
          <option v-for="c in discussionCategories" :key="c.id" :value="c.id">{{ c.name }}</option>
        </select>
        <textarea v-model="discussionForm.body" rows="3" placeholder="内容（可选）" class="form-input" />
        <p v-if="discussionFormError" class="error">{{ discussionFormError }}</p>
        <button class="rounded-md bg-[#ea580c] px-3 py-1.5 text-sm font-semibold text-white hover:bg-[#c2410c] disabled:opacity-50" type="submit" :disabled="discussionCreating">
          {{ discussionCreating ? '提交中...' : '创建讨论' }}
        </button>
      </form>

      <p v-if="discussionsError" class="error">{{ discussionsError }}</p>
      <div v-if="discussionsLoading" class="py-8 text-center text-sm text-[#59636e]">加载中…</div>
      <ul v-else-if="discussions.length" class="divide-y divide-[#d1d9e0] rounded-md border border-[#d1d9e0] bg-white dark:divide-[#30363d] dark:border-[#30363d] dark:bg-[#161b22]">
        <li
          v-for="d in discussions"
          :key="d.id"
          class="item-row disc-row"
          :class="{ selected: selectedDiscussionId === d.id }"
          @click="openDiscussion(d)"
        >
          <span class="tag">{{ categoryLabel(d.category_id) }}</span>
          <span class="truncate font-semibold text-[#1f2328] hover:text-[#ea580c] dark:text-[#e6edf3]">{{ d.title }}</span>
          <span v-if="d.is_answered" class="state answered">已回答</span>
          <span class="when">{{ fmtDate(d.created_at) }}</span>
        </li>
      </ul>
      <EmptyState
        v-else-if="!discussionsError"
        icon="💬"
        title="还没有讨论"
        description="讨论适合开放式话题、答疑和方案评审。"
        action-label="发起第一个讨论"
        :show-action="auth.isLoggedIn"
        @action="activeTab = '讨论'"
      />

      <div v-if="selectedDiscussionId !== null" class="discussion-detail">
        <p v-if="discussionDetailError" class="error">{{ discussionDetailError }}</p>
        <div v-if="discussionDetailLoading" class="py-8 text-center text-sm text-[#59636e]">加载详情中…</div>
        <template v-else-if="discussionDetail">
          <div class="disc-head">
            <h4>{{ discussionDetail.discussion.title }}</h4>
            <span class="tag">{{ categoryLabel(discussionDetail.discussion.category_id) }}</span>
            <span v-if="discussionDetail.discussion.is_answered" class="state answered">已回答</span>
            <button class="rounded-md border border-[#d1d9e0] px-3 py-1.5 text-sm text-[#1f2328] hover:bg-[#f6f8fa] dark:border-[#30363d] dark:text-[#e6edf3] dark:hover:bg-[#21262d]" type="button" @click="closeDiscussion">关闭</button>
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
                  class="rounded-md border border-[#d1d9e0] px-3 py-1.5 text-sm text-[#1f2328] hover:bg-[#f6f8fa] dark:border-[#30363d] dark:text-[#e6edf3] dark:hover:bg-[#21262d]"
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
          <p v-else class="py-8 text-center text-sm text-[#59636e]">暂无评论</p>

          <form v-if="auth.isLoggedIn" class="mb-4" @submit.prevent="addComment">
            <textarea v-model="commentBody" rows="3" placeholder="写下你的评论..." class="form-input" />
            <button class="rounded-md bg-[#ea580c] px-3 py-1.5 text-sm font-semibold text-white hover:bg-[#c2410c] disabled:opacity-50" type="submit" :disabled="commentAdding || !commentBody.trim()">
              {{ commentAdding ? '提交中...' : '发表评论' }}
            </button>
          </form>
        </template>
      </div>
    </div>

    <div v-else-if="activeTab === '设置'" class="p-4 md:p-6">
      <DeployKeys :owner="owner" :repo="name" />
      <RepoDangerZone
        v-if="auth.isLoggedIn"
        :owner="owner"
        :repo="name"
        :archived="repo?.archived ?? false"
        @updated="onRepoUpdated"
      />
    </div>

    <div v-else-if="activeTab === '发布'" class="p-4 md:p-6">
      <RepoReleases :owner="owner" :repo="name" />
    </div>

    <div v-else-if="activeTab === '活动'" class="p-4 md:p-6">
      <RepoActivity :owner="owner" :repo="name" />
    </div>

    <div v-else-if="activeTab === '百科'" class="p-4 md:p-6">
      <RepoWiki :owner="owner" :repo="name" />
    </div>

    <div v-else class="p-4 md:p-6">
      <p class="placeholder">未知页面：{{ activeTab }}</p>
    </div>
  </div>
</template>
