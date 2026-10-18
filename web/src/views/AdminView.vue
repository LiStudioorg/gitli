<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { errMessage } from '../utils/error'
import AppLayout from '../components/layout/AppLayout.vue'
import api from '../api/client'

interface AdminStats {
  users: number
  repos: number
  orgs: number
  issues: number
  pulls: number
  gists: number
  releases: number
  webhooks: number
  tokens: number
  uptime_seconds: number
}

interface AuditLog {
  id: number
  actor_id: number | null
  actor_type: string
  actor_ip: string
  event: string
  resource: string
  action: string
  target_type: string
  target_id: number | null
  target_name: string
  old_value: string | null
  new_value: string | null
  result: string
  error_message: string
  request_id: string
  metadata: string | null
  actor_user_agent: string
  created_at: string
}

interface AuditStats {
  by_day: { day: string; count: number }[]
  by_event: { key: string; count: number }[]
  by_actor: { key: string; count: number }[]
}

interface AlertRule {
  id: number
  name: string
  event: string
  threshold: number
  window_seconds: number
  channel: string
  target: string
  enabled: boolean
  last_fired_at: string | null
  created_at: string
}

type TabKey = 'users' | 'repos' | 'orgs' | 'tasks' | 'runner' | 'audit' | 'system'

interface Tab {
  key: TabKey
  label: string
}

const tabs: Tab[] = [
  { key: 'users', label: '用户' },
  { key: 'repos', label: '仓库' },
  { key: 'orgs', label: '组织' },
  { key: 'tasks', label: '任务' },
  { key: 'runner', label: 'Runner' },
  { key: 'audit', label: '审计日志' },
  { key: 'system', label: '系统状态' },
]

interface AdminUser {
  id: number
  username: string
  email: string
  is_admin: boolean
  is_active: boolean
  totp_enabled: boolean
  created_at: string
}

interface AdminRepo {
  id: number
  name: string
  owner_id: number
  is_private: boolean
  archived: boolean
  deleted: boolean
  size_kb: number
  default_branch: string
}

interface AdminOrg {
  ID: number
  Name: string
  DisplayName: string
  Description: string
  OwnerID: number
}

interface SystemInfo {
  go_version?: string
  goroutines?: number
  heap_alloc_mb?: number
  sys_mb?: number
  num_cpu?: number
  db_open?: number
  db_in_use?: number
  db_idle?: number
  db_wait_count?: number
  disk_total_mb?: number
  disk_free_mb?: number
  git_binary?: string
}

interface AdminViewState<T> {
  items: T
  loading: boolean
  loaded: boolean
  error: string
}

function emptyListState<T>(): AdminViewState<T[]> {
  return { items: [], loading: false, loaded: false, error: '' }
}

const usersState = ref<AdminViewState<AdminUser[]>>(emptyListState<AdminUser>())
const reposState = ref<AdminViewState<AdminRepo[]>>(emptyListState<AdminRepo>())
const orgsState = ref<AdminViewState<AdminOrg[]>>(emptyListState<AdminOrg>())

const usersQuery = ref('')
const reposQuery = ref('')

const userActionId = ref<number | null>(null)
const reposActionId = ref<number | null>(null)
const resetPassword = ref<{ username: string; password: string } | null>(null)

const systemInfo = ref<SystemInfo | null>(null)
const systemLoading = ref(false)
const systemError = ref('')

async function loadUsers() {
  usersState.value.loading = true
  usersState.value.error = ''
  try {
    const params: Record<string, string> = {}
    const q = usersQuery.value.trim()
    if (q) params.q = q
    const { data } = await api.get<AdminUser[]>('/admin/users', { params })
    usersState.value.items = data ?? []
    usersState.value.loaded = true
  } catch (e) {
    usersState.value.error = errMessage(e)
    usersState.value.items = []
  } finally {
    usersState.value.loading = false
  }
}

async function toggleUserActive(u: AdminUser) {
  const next = !u.is_active
  if (!next && !window.confirm(`确定要禁用用户「${u.username}」吗？`)) return
  usersState.value.error = ''
  userActionId.value = u.id
  try {
    const { data } = await api.patch<{ status: string; active: boolean }>(`/admin/users/${u.id}`, {
      active: next,
    })
    u.is_active = data?.active ?? next
  } catch (e) {
    usersState.value.error = errMessage(e)
  } finally {
    userActionId.value = null
  }
}

async function resetUserPassword(u: AdminUser) {
  if (!window.confirm(`确定要重置用户「${u.username}」的密码吗？旧密码将立即失效。`)) return
  usersState.value.error = ''
  userActionId.value = u.id
  try {
    const { data } = await api.post<{ password: string }>(`/admin/users/${u.id}/reset-password`)
    resetPassword.value = { username: u.username, password: data.password }
  } catch (e) {
    usersState.value.error = errMessage(e)
  } finally {
    userActionId.value = null
  }
}

async function resetUser2FA(u: AdminUser) {
  if (!window.confirm(`确定要重置用户「${u.username}」的两步验证吗？`)) return
  usersState.value.error = ''
  userActionId.value = u.id
  try {
    await api.post(`/admin/users/${u.id}/reset-2fa`)
    u.totp_enabled = false
  } catch (e) {
    usersState.value.error = errMessage(e)
  } finally {
    userActionId.value = null
  }
}

async function loadRepos() {
  reposState.value.loading = true
  reposState.value.error = ''
  try {
    const params: Record<string, string> = {}
    const q = reposQuery.value.trim()
    if (q) params.q = q
    const { data } = await api.get<AdminRepo[]>('/admin/repos', { params })
    reposState.value.items = data ?? []
    reposState.value.loaded = true
  } catch (e) {
    reposState.value.error = errMessage(e)
    reposState.value.items = []
  } finally {
    reposState.value.loading = false
  }
}

async function toggleRepoArchive(rp: AdminRepo) {
  const next = !rp.archived
  if (!window.confirm(`确定要${next ? '归档' : '解除归档'}仓库「${rp.name}」吗？`)) return
  reposState.value.error = ''
  reposActionId.value = rp.id
  try {
    const { data } = await api.post<{ status: string; archived: boolean }>(
      `/admin/repos/${rp.id}/archive`,
      { archived: next },
    )
    rp.archived = data?.archived ?? next
  } catch (e) {
    reposState.value.error = errMessage(e)
  } finally {
    reposActionId.value = null
  }
}

async function loadOrgs() {
  orgsState.value.loading = true
  orgsState.value.error = ''
  try {
    const { data } = await api.get<AdminOrg[]>('/admin/orgs')
    orgsState.value.items = data ?? []
    orgsState.value.loaded = true
  } catch (e) {
    orgsState.value.error = errMessage(e)
    orgsState.value.items = []
  } finally {
    orgsState.value.loading = false
  }
}

async function loadSystem() {
  systemLoading.value = true
  systemError.value = ''
  try {
    const { data } = await api.get<SystemInfo>('/admin/system')
    systemInfo.value = data
  } catch (e) {
    systemError.value = errMessage(e)
  } finally {
    systemLoading.value = false
  }
}

function formatSize(kb: number | undefined): string {
  if (kb == null) return '-'
  if (kb < 1024) return `${kb} KB`
  const mb = kb / 1024
  if (mb < 1024) return `${mb.toFixed(1)} MB`
  return `${(mb / 1024).toFixed(2)} GB`
}

const systemCards = computed(() => {
  const s = systemInfo.value
  if (!s) return []
  const disk =
    s.disk_total_mb != null && s.disk_free_mb != null
      ? `${s.disk_free_mb} / ${s.disk_total_mb} MB`
      : '-'
  return [
    { label: 'Go 版本', value: s.go_version ?? '-', mono: true },
    { label: 'Goroutines', value: s.goroutines ?? '-', mono: false },
    { label: '堆内存 (Heap)', value: s.heap_alloc_mb != null ? `${s.heap_alloc_mb} MB` : '-', mono: false },
    { label: '系统内存 (Sys)', value: s.sys_mb != null ? `${s.sys_mb} MB` : '-', mono: false },
    { label: 'CPU 核心', value: s.num_cpu ?? '-', mono: false },
    {
      label: 'DB 连接池 (open/in-use/idle)',
      value: `${s.db_open ?? '-'} / ${s.db_in_use ?? '-'} / ${s.db_idle ?? '-'}`,
      mono: false,
    },
    { label: 'DB 等待次数', value: s.db_wait_count ?? '-', mono: false },
    { label: '磁盘 (free / total)', value: disk, mono: false },
    { label: 'Git 二进制', value: s.git_binary ?? '-', mono: true },
  ]
})

interface Runner {
  id: number
  runner_id: string
  name: string
  labels: string | string[]
  version: string
  os: string
  arch: string
  status: string
  last_seen_at: string
}

interface CreatedRunner {
  id: number
  runner_id: string
  name: string
  labels?: string | string[]
  token: string
}

const runners = ref<Runner[]>([])
const runnersError = ref('')
const runnersLoading = ref(false)
const runnersLoaded = ref(false)

const runnerModal = ref(false)
const runnerSaving = ref(false)
const runnerFormError = ref('')
const runnerForm = ref({ name: '', labels: '' })

const createdRunner = ref<CreatedRunner | null>(null)

const editModal = ref(false)
const editSaving = ref(false)
const editError = ref('')
const editForm = ref({ id: 0, name: '', status: '', labels: '' })

function runnerLabels(r: Runner): string[] {
  if (Array.isArray(r.labels)) return r.labels
  if (!r.labels) return []
  const raw = r.labels.trim()
  if (!raw) return []
  if (raw.startsWith('[')) {
    try {
      const parsed = JSON.parse(raw)
      if (Array.isArray(parsed)) return parsed.map((v) => String(v))
    } catch {
      return []
    }
  }
  return raw.split(',').map((s) => s.trim()).filter(Boolean)
}

function parseLabelsInput(input: string): string[] {
  return input
    .split(',')
    .map((s) => s.trim())
    .filter(Boolean)
}

function runnerStatusLabel(status: string): string {
  if (status === 'active') return '在线'
  if (status === 'offline') return '离线'
  if (status === 'disabled') return '已禁用'
  return status || '-'
}

function formatTime(t: string | undefined): string {
  if (!t) return '-'
  const d = new Date(t)
  return Number.isNaN(d.getTime()) ? t : d.toLocaleString()
}

function hostForConfig(): string {
  return window.location.host
}

function runnerConfigSnippet(r: CreatedRunner): string {
  return `[runner]\nrunner_id = "${r.runner_id}"\ntoken = "${r.token}"\nurl = "https://${hostForConfig()}"`
}

const copiedKey = ref('')
async function copyText(text: string, key: string) {
  try {
    await navigator.clipboard.writeText(text)
    copiedKey.value = key
    window.setTimeout(() => {
      if (copiedKey.value === key) copiedKey.value = ''
    }, 1500)
  } catch {
    copiedKey.value = ''
  }
}

async function loadRunners() {
  runnersLoading.value = true
  runnersError.value = ''
  try {
    const { data } = await api.get<Runner[]>('/admin/runners')
    runners.value = data ?? []
    runnersLoaded.value = true
  } catch (e) {
    runnersError.value = errMessage(e)
    runners.value = []
  } finally {
    runnersLoading.value = false
  }
}

function openRunnerModal() {
  runnerForm.value = { name: '', labels: '' }
  runnerFormError.value = ''
  createdRunner.value = null
  runnerModal.value = true
}

function closeRunnerModal() {
  runnerModal.value = false
  createdRunner.value = null
}

async function createRunner() {
  runnerFormError.value = ''
  if (!runnerForm.value.name.trim()) {
    runnerFormError.value = '名称不能为空'
    return
  }
  runnerSaving.value = true
  try {
    const { data } = await api.post<CreatedRunner>('/admin/runners', {
      name: runnerForm.value.name.trim(),
      labels: parseLabelsInput(runnerForm.value.labels),
    })
    createdRunner.value = data
    await loadRunners()
  } catch (e) {
    runnerFormError.value = errMessage(e)
  } finally {
    runnerSaving.value = false
  }
}

async function resetToken(r: Runner) {
  if (!window.confirm(`确定要重置 Runner「${r.name || r.runner_id}」的 Token 吗？旧 Token 将立即失效。`)) return
  runnersError.value = ''
  try {
    const { data } = await api.post<{ token: string }>(`/admin/runners/${r.id}/reset-token`)
    createdRunner.value = { id: r.id, runner_id: r.runner_id, name: r.name, labels: r.labels, token: data.token }
    runnerModal.value = true
  } catch (e) {
    runnersError.value = errMessage(e)
  }
}

function openEdit(r: Runner) {
  editForm.value = {
    id: r.id,
    name: r.name,
    status: r.status,
    labels: runnerLabels(r).join(', '),
  }
  editError.value = ''
  editModal.value = true
}

async function saveEdit() {
  editError.value = ''
  if (!editForm.value.name.trim()) {
    editError.value = '名称不能为空'
    return
  }
  editSaving.value = true
  try {
    await api.patch(`/admin/runners/${editForm.value.id}`, {
      name: editForm.value.name.trim(),
      status: editForm.value.status,
      labels: parseLabelsInput(editForm.value.labels),
    })
    editModal.value = false
    await loadRunners()
  } catch (e) {
    editError.value = errMessage(e)
  } finally {
    editSaving.value = false
  }
}

async function deleteRunner(r: Runner) {
  if (!window.confirm(`确定要删除 Runner「${r.name || r.runner_id}」吗？此操作不可撤销。`)) return
  runnersError.value = ''
  try {
    await api.delete(`/admin/runners/${r.id}`)
    await loadRunners()
  } catch (e) {
    runnersError.value = errMessage(e)
  }
}

const active = ref<TabKey>('system')
const stats = ref<AdminStats | null>(null)
const statsError = ref('')
const loadingStats = ref(false)


const purgeRepoId = ref('')
const purging = ref(false)
const purgeError = ref('')
const purgeOk = ref('')

async function purgeRepo() {
  purgeError.value = ''
  purgeOk.value = ''
  const id = Number(purgeRepoId.value)
  if (!Number.isInteger(id) || id <= 0) {
    purgeError.value = '请输入有效的仓库 ID'
    return
  }
  if (
    !window.confirm(
      `确定要永久删除仓库 #${id} 吗？此操作不可恢复，仓库及其数据将被彻底清除。`,
    )
  ) {
    return
  }
  purging.value = true
  try {
    const { data } = await api.delete<{ status: string; purged: boolean }>(
      `/admin/repos/${id}/purge`,
    )
    purgeOk.value = data?.purged
      ? `仓库 #${id} 已永久删除`
      : `请求已完成（仓库 #${id} 未删除）`
    purgeRepoId.value = ''
  } catch (e) {
    purgeError.value = errMessage(e)
  } finally {
    purging.value = false
  }
}

const PAGE_SIZE = 50

interface AuditFilters {
  event: string
  actor_id: string
  result: string
  from: string
  to: string
}

const auditFilters = ref<AuditFilters>({ event: '', actor_id: '', result: '', from: '', to: '' })
const auditLogs = ref<AuditLog[]>([])
const auditError = ref('')
const loadingAudit = ref(false)
const auditLoaded = ref(false)
const auditOffset = ref(0)
const expandedId = ref<number | null>(null)

const auditStats = ref<AuditStats | null>(null)
const auditStatsError = ref('')
const loadingAuditStats = ref(false)

const archiveRetentionDays = ref(365)
const archiving = ref(false)
const archiveMessage = ref('')
const archiveError = ref('')

const exportFormat = ref<'csv' | 'json'>('csv')

const alertRules = ref<AlertRule[]>([])
const alertRulesError = ref('')
const loadingAlertRules = ref(false)
const alertRulesLoaded = ref(false)
const alertSaving = ref(false)
const alertFormError = ref('')
const alertForm = ref({
  name: '',
  event: '',
  threshold: 1,
  window_seconds: 3600,
  channel: 'email' as 'email' | 'webhook',
  target: '',
  enabled: true,
})

function auditQueryParams(): Record<string, string | number> {
  const f = auditFilters.value
  const params: Record<string, string | number> = {}
  if (f.event.trim()) params.event = f.event.trim()
  if (f.actor_id.trim() && !Number.isNaN(Number(f.actor_id))) params.actor_id = Number(f.actor_id)
  if (f.result) params.result = f.result
  if (f.from) params.from = new Date(f.from).toISOString()
  if (f.to) params.to = new Date(f.to).toISOString()
  return params
}

function resultLabel(result: string): string {
  if (result === 'success') return '成功'
  if (result === 'failure') return '失败'
  if (result === 'denied') return '拒绝'
  return result || '-'
}

function auditTime(log: AuditLog): string {
  if (!log.created_at) return '-'
  const d = new Date(log.created_at)
  return Number.isNaN(d.getTime()) ? log.created_at : d.toLocaleString()
}

function auditTargetDisplay(log: AuditLog): string {
  if (log.target_name) return log.target_name
  if (log.target_type && log.target_id != null) return `${log.target_type}#${log.target_id}`
  return log.target_type || '-'
}

function prettyValue(v: string | null): string {
  if (v == null || v === '') return ''
  try {
    return JSON.stringify(JSON.parse(v), null, 2)
  } catch {
    return v
  }
}

function toggleExpand(id: number) {
  expandedId.value = expandedId.value === id ? null : id
}

async function loadAudit() {
  loadingAudit.value = true
  auditError.value = ''
  expandedId.value = null
  try {
    const { data } = await api.get<{ logs: AuditLog[] }>('/admin/audit', {
      params: { ...auditQueryParams(), limit: PAGE_SIZE, offset: auditOffset.value },
    })
    auditLogs.value = data.logs ?? []
    auditLoaded.value = true
  } catch (e) {
    auditError.value = errMessage(e)
    auditLogs.value = []
  } finally {
    loadingAudit.value = false
  }
}

function applyFilters() {
  auditOffset.value = 0
  loadAudit()
}

function nextPage() {
  if (auditLogs.value.length < PAGE_SIZE) return
  auditOffset.value += PAGE_SIZE
  loadAudit()
}

function prevPage() {
  if (auditOffset.value === 0) return
  auditOffset.value = Math.max(0, auditOffset.value - PAGE_SIZE)
  loadAudit()
}

async function loadAuditStats() {
  loadingAuditStats.value = true
  auditStatsError.value = ''
  try {
    const { data } = await api.get<AuditStats>('/admin/audit/stats', { params: { days: 30 } })
    auditStats.value = data
  } catch (e) {
    auditStatsError.value = errMessage(e)
    auditStats.value = null
  } finally {
    loadingAuditStats.value = false
  }
}

function exportAudit() {
  const params = new URLSearchParams()
  params.set('format', exportFormat.value)
  const q = auditQueryParams()
  if (q.event != null) params.set('event', String(q.event))
  if (q.result != null) params.set('result', String(q.result))
  if (q.from != null) params.set('from', String(q.from))
  if (q.to != null) params.set('to', String(q.to))
  const a = document.createElement('a')
  a.href = `/api/v1/admin/audit/export?${params.toString()}`
  a.download = ''
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
}

async function archiveAudit() {
  archiving.value = true
  archiveMessage.value = ''
  archiveError.value = ''
  try {
    const { data } = await api.post<{ archived: number; retention_days: number }>(
      '/admin/audit/archive',
      { retention_days: archiveRetentionDays.value },
    )
    archiveMessage.value = `已归档 ${data.archived} 条日志（保留 ${data.retention_days} 天）`
    auditOffset.value = 0
    await loadAudit()
  } catch (e) {
    archiveError.value = errMessage(e)
  } finally {
    archiving.value = false
  }
}

async function loadAlertRules() {
  loadingAlertRules.value = true
  alertRulesError.value = ''
  try {
    const { data } = await api.get<AlertRule[]>('/admin/audit/alert-rules')
    alertRules.value = data ?? []
    alertRulesLoaded.value = true
  } catch (e) {
    alertRulesError.value = errMessage(e)
    alertRules.value = []
  } finally {
    loadingAlertRules.value = false
  }
}

async function createAlertRule() {
  alertFormError.value = ''
  const f = alertForm.value
  if (!f.name.trim()) {
    alertFormError.value = '名称不能为空'
    return
  }
  if (!f.event.trim()) {
    alertFormError.value = '事件不能为空'
    return
  }
  alertSaving.value = true
  try {
    await api.post('/admin/audit/alert-rules', {
      name: f.name.trim(),
      event: f.event.trim(),
      threshold: Number(f.threshold) || 1,
      window_seconds: Number(f.window_seconds) || 3600,
      channel: f.channel,
      target: f.target.trim(),
      enabled: f.enabled,
    })
    alertForm.value = {
      name: '',
      event: '',
      threshold: 1,
      window_seconds: 3600,
      channel: 'email',
      target: '',
      enabled: true,
    }
    await loadAlertRules()
  } catch (e) {
    alertFormError.value = errMessage(e)
  } finally {
    alertSaving.value = false
  }
}

async function deleteAlertRule(rule: AlertRule) {
  if (!window.confirm(`确定要删除告警规则「${rule.name}」吗？`)) return
  alertRulesError.value = ''
  try {
    await api.delete(`/admin/audit/alert-rules/${rule.id}`)
    await loadAlertRules()
  } catch (e) {
    alertRulesError.value = errMessage(e)
  }
}

const topEvents = computed(() => (auditStats.value?.by_event ?? []).slice(0, 10))
const byDay30 = computed(() => auditStats.value?.by_day ?? [])
const topActors = computed(() => auditStats.value?.by_actor ?? [])

function barWidth(count: number, max: number): number {
  if (max <= 0) return 0
  return Math.max(2, Math.round((count / max) * 100))
}
const maxEventCount = computed(() => Math.max(1, ...topEvents.value.map((e) => e.count)))
const maxDayCount = computed(() => Math.max(1, ...byDay30.value.map((e) => e.count)))

function formatUptime(seconds: number | undefined): string {
  if (seconds == null || seconds < 0) return '-'
  const s = Math.floor(seconds)
  const d = Math.floor(s / 86400)
  const h = Math.floor((s % 86400) / 3600)
  const m = Math.floor((s % 3600) / 60)
  const parts: string[] = []
  if (d) parts.push(`${d} 天`)
  if (h) parts.push(`${h} 小时`)
  parts.push(`${m} 分钟`)
  return parts.join(' ')
}

const statCards = computed(() => {
  const s = stats.value
  if (!s) return []
  return [
    { label: '用户', value: s.users },
    { label: '仓库', value: s.repos },
    { label: '组织', value: s.orgs },
    { label: '工单', value: s.issues },
    { label: '合并请求', value: s.pulls },
    { label: 'Gist', value: s.gists },
    { label: '发行版', value: s.releases },
    { label: 'Webhook', value: s.webhooks },
    { label: '访问令牌', value: s.tokens },
  ]
})

async function loadStats() {
  loadingStats.value = true
  statsError.value = ''
  try {
    const { data } = await api.get<AdminStats>('/admin/stats')
    stats.value = data
  } catch (e) {
    statsError.value = errMessage(e)
  } finally {
    loadingStats.value = false
  }
}

function selectTab(key: TabKey) {
  active.value = key
  if (key === 'system') {
    if (!stats.value && !loadingStats.value) loadStats()
    if (!systemInfo.value && !systemLoading.value) loadSystem()
  }
  if (key === 'users' && !usersState.value.loaded && !usersState.value.loading) loadUsers()
  if (key === 'repos' && !reposState.value.loaded && !reposState.value.loading) loadRepos()
  if (key === 'orgs' && !orgsState.value.loaded && !orgsState.value.loading) loadOrgs()
  if (key === 'audit') {
    if (!auditLoaded.value && !loadingAudit.value) loadAudit()
    if (!auditStats.value && !loadingAuditStats.value) loadAuditStats()
    if (!alertRulesLoaded.value && !loadingAlertRules.value) loadAlertRules()
  }
  if (key === 'runner' && !runnersLoaded.value && !runnersLoading.value) loadRunners()
}

onMounted(() => {
  loadStats()
  if (active.value === 'users') loadUsers()
  if (active.value === 'system' && !systemInfo.value && !systemLoading.value) loadSystem()
})
</script>

<template>
  <AppLayout>
    <div class="mx-auto w-full max-w-[1100px]">
        <h2 class="mb-4 text-lg font-semibold text-[#1f2328] dark:text-[#e6edf3]">管理面板</h2>
        <nav class="mb-4 flex flex-wrap gap-1 border-b border-[#d1d9e0] dark:border-[#30363d]" role="tablist">
          <button
            v-for="t in tabs"
            :key="t.key"
            class="h-10 border-b-2 px-3 text-sm"
            :class="active === t.key
              ? 'border-[#ea580c] font-semibold text-[#1f2328] dark:text-[#e6edf3]'
              : 'border-transparent text-[#59636e] hover:rounded-t-md hover:border-[#d1d9e0] hover:bg-[#f6f8fa] dark:hover:bg-[#21262d]'"
            type="button"
            role="tab"
            :aria-selected="active === t.key"
            @click="selectTab(t.key)"
          >
            {{ t.label }}
          </button>
        </nav>

        <section v-if="active === 'users'" class="rounded-md border border-[#d1d9e0] bg-white p-4 dark:border-[#3d444d] dark:bg-[#0d1117]">
          <form class="filters" @submit.prevent="loadUsers">
            <label class="filter search-field">
              <span>搜索用户</span>
              <input v-model="usersQuery" type="search" placeholder="用户名或邮箱" aria-label="搜索用户" />
            </label>
            <div class="filter filter-actions">
              <button class="inline-flex h-8 items-center rounded-md bg-[#2da44e] px-3 text-sm font-semibold text-white hover:bg-[#2c974b] disabled:opacity-50" type="submit">搜索</button>
              <button class="inline-flex h-8 items-center rounded-md border border-[#d1d9e0] px-3 text-sm text-[#1f2328] hover:bg-[#f6f8fa] dark:border-[#30363d] dark:text-[#e6edf3] dark:hover:bg-[#21262d]" type="button" @click="loadUsers">刷新</button>
            </div>
          </form>

          <p v-if="resetPassword" class="notice password-box">
            <span>
              用户「{{ resetPassword.username }}」的新密码（仅显示一次，请立即保存）：
              <code class="mono">{{ resetPassword.password }}</code>
            </span>
            <button class="link-btn" type="button" @click="copyText(resetPassword.password, 'password')">
              {{ copiedKey === 'password' ? '已复制' : '复制' }}
            </button>
            <button class="link-btn" type="button" @click="resetPassword = null">关闭</button>
          </p>

          <p v-if="usersState.error" class="text-sm text-[#cf222e]">{{ usersState.error }}</p>
          <p v-if="usersState.loading" class="text-sm text-[#59636e]">加载中…</p>
          <div v-else-if="usersState.items.length" class="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>ID</th>
                  <th>用户名</th>
                  <th>邮箱</th>
                  <th>管理员</th>
                  <th>状态</th>
                  <th>2FA</th>
                  <th>创建时间</th>
                  <th>操作</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="u in usersState.items" :key="u.id">
                  <td>{{ u.id }}</td>
                  <td class="mono">{{ u.username }}</td>
                  <td>{{ u.email || '-' }}</td>
                  <td>
                    <span class="status" :class="u.is_admin ? 'status-active' : 'status-offline'">
                      {{ u.is_admin ? '是' : '否' }}
                    </span>
                  </td>
                  <td>
                    <span class="status" :class="u.is_active ? 'status-active' : 'status-disabled'">
                      {{ u.is_active ? '启用' : '已禁用' }}
                    </span>
                  </td>
                  <td>
                    <span class="status" :class="u.totp_enabled ? 'status-active' : 'status-offline'">
                      {{ u.totp_enabled ? '已启用' : '未启用' }}
                    </span>
                  </td>
                  <td>{{ formatTime(u.created_at) }}</td>
                  <td class="row-actions">
                    <button
                      class="link-btn"
                      type="button"
                      :disabled="userActionId === u.id"
                      @click="toggleUserActive(u)"
                    >
                      {{ u.is_active ? '禁用' : '启用' }}
                    </button>
                    <button
                      class="link-btn"
                      type="button"
                      :disabled="userActionId === u.id"
                      @click="resetUserPassword(u)"
                    >
                      重置密码
                    </button>
                    <button
                      class="link-btn"
                      type="button"
                      :disabled="userActionId === u.id || !u.totp_enabled"
                      @click="resetUser2FA(u)"
                    >
                      重置 2FA
                    </button>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
          <p v-else-if="!usersState.error" class="text-sm text-[#59636e]">暂无用户</p>
          <p class="cli-hint">
            彻底删除用户请使用 CLI：<code>gitli admin user delete &lt;username&gt; --purge</code>
          </p>
        </section>

        <section v-else-if="active === 'repos'" class="rounded-md border border-[#d1d9e0] bg-white p-4 dark:border-[#3d444d] dark:bg-[#0d1117]">
          <form class="filters" @submit.prevent="loadRepos">
            <label class="filter search-field">
              <span>搜索仓库</span>
              <input v-model="reposQuery" type="search" placeholder="仓库名称" aria-label="搜索仓库" />
            </label>
            <div class="filter filter-actions">
              <button class="inline-flex h-8 items-center rounded-md bg-[#2da44e] px-3 text-sm font-semibold text-white hover:bg-[#2c974b] disabled:opacity-50" type="submit">搜索</button>
              <button class="inline-flex h-8 items-center rounded-md border border-[#d1d9e0] px-3 text-sm text-[#1f2328] hover:bg-[#f6f8fa] dark:border-[#30363d] dark:text-[#e6edf3] dark:hover:bg-[#21262d]" type="button" @click="loadRepos">刷新</button>
            </div>
          </form>

          <p v-if="reposState.error" class="text-sm text-[#cf222e]">{{ reposState.error }}</p>
          <p v-if="reposState.loading" class="text-sm text-[#59636e]">加载中…</p>
          <div v-else-if="reposState.items.length" class="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>ID</th>
                  <th>名称</th>
                  <th>所有者 ID</th>
                  <th>私有</th>
                  <th>已归档</th>
                  <th>已删除</th>
                  <th>大小</th>
                  <th>默认分支</th>
                  <th>操作</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="rp in reposState.items" :key="rp.id">
                  <td>{{ rp.id }}</td>
                  <td class="mono">{{ rp.name }}</td>
                  <td>{{ rp.owner_id }}</td>
                  <td>{{ rp.is_private ? '是' : '否' }}</td>
                  <td>
                    <span class="status" :class="rp.archived ? 'status-disabled' : 'status-offline'">
                      {{ rp.archived ? '已归档' : '否' }}
                    </span>
                  </td>
                  <td>
                    <span class="status" :class="rp.deleted ? 'status-disabled' : 'status-offline'">
                      {{ rp.deleted ? '是' : '否' }}
                    </span>
                  </td>
                  <td>{{ formatSize(rp.size_kb) }}</td>
                  <td class="mono">{{ rp.default_branch || '-' }}</td>
                  <td class="row-actions">
                    <button
                      class="link-btn"
                      type="button"
                      :disabled="reposActionId === rp.id"
                      @click="toggleRepoArchive(rp)"
                    >
                      {{ rp.archived ? '解除归档' : '归档' }}
                    </button>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
          <p v-else-if="!reposState.error" class="text-sm text-[#59636e]">暂无仓库</p>
          <div class="subsection">
            <h3 class="panel-title">永久删除仓库</h3>
            <p class="text-sm text-[#59636e]">
              按仓库 ID 永久删除仓库及其数据，<strong class="danger-text">此操作不可恢复</strong>（CLI 专属操作）。
            </p>
            <p v-if="purgeError" class="text-sm text-[#cf222e]">{{ purgeError }}</p>
            <p v-if="purgeOk" class="notice">{{ purgeOk }}</p>
            <form class="purge-form" @submit.prevent="purgeRepo">
              <label class="filter">
                <span>仓库 ID</span>
                <input v-model="purgeRepoId" type="number" min="1" step="1" placeholder="仓库 ID" />
              </label>
              <div class="filter filter-actions">
                <button class="inline-flex h-8 items-center rounded-md bg-[#2da44e] px-3 text-sm font-semibold text-white hover:bg-[#2c974b] disabled:opacity-50" type="submit" :disabled="purging">
                  {{ purging ? '删除中…' : '永久删除' }}
                </button>
              </div>
            </form>
          </div>
        </section>

        <section v-else-if="active === 'orgs'" class="rounded-md border border-[#d1d9e0] bg-white p-4 dark:border-[#3d444d] dark:bg-[#0d1117]">
          <div class="panel-toolbar">
            <button class="inline-flex h-8 items-center rounded-md border border-[#d1d9e0] px-3 text-sm text-[#1f2328] hover:bg-[#f6f8fa] dark:border-[#30363d] dark:text-[#e6edf3] dark:hover:bg-[#21262d]" type="button" @click="loadOrgs">刷新</button>
          </div>
          <p v-if="orgsState.error" class="text-sm text-[#cf222e]">{{ orgsState.error }}</p>
          <p v-if="orgsState.loading" class="text-sm text-[#59636e]">加载中…</p>
          <div v-else-if="orgsState.items.length" class="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>ID</th>
                  <th>名称</th>
                  <th>显示名称</th>
                  <th>所有者 ID</th>
                  <th>描述</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="o in orgsState.items" :key="o.ID">
                  <td>{{ o.ID }}</td>
                  <td class="mono">{{ o.Name }}</td>
                  <td>{{ o.DisplayName || '-' }}</td>
                  <td>{{ o.OwnerID }}</td>
                  <td class="desc-cell">{{ o.Description || '-' }}</td>
                </tr>
              </tbody>
            </table>
          </div>
          <p v-else-if="!orgsState.error" class="text-sm text-[#59636e]">暂无组织</p>
        </section>

        <section v-else-if="active === 'system'" class="rounded-md border border-[#d1d9e0] bg-white p-4 dark:border-[#3d444d] dark:bg-[#0d1117]">
          <h3 class="panel-title">概览</h3>
          <p v-if="statsError" class="text-sm text-[#cf222e]">{{ statsError }}</p>
          <p v-if="loadingStats" class="text-sm text-[#59636e]">加载中…</p>
          <template v-else-if="stats">
            <div class="cards">
              <div v-for="c in statCards" :key="c.label" class="stat-card">
                <span class="stat-value">{{ c.value }}</span>
                <span class="stat-label">{{ c.label }}</span>
              </div>
            </div>
            <p class="uptime">运行时间：{{ formatUptime(stats.uptime_seconds) }}</p>
          </template>
          <p v-else-if="!statsError" class="text-sm text-[#59636e]">暂无数据</p>

          <div class="subsection">
            <div class="panel-toolbar system-toolbar">
              <h3>系统运行状态</h3>
              <button class="inline-flex h-8 items-center rounded-md border border-[#d1d9e0] px-3 text-sm text-[#1f2328] hover:bg-[#f6f8fa] dark:border-[#30363d] dark:text-[#e6edf3] dark:hover:bg-[#21262d]" type="button" :disabled="systemLoading" @click="loadSystem">
                {{ systemLoading ? '刷新中…' : '刷新' }}
              </button>
            </div>
            <p v-if="systemError" class="text-sm text-[#cf222e]">{{ systemError }}</p>
            <p v-if="systemLoading && !systemInfo" class="text-sm text-[#59636e]">加载中…</p>
            <div v-else-if="systemInfo" class="cards">
              <div v-for="c in systemCards" :key="c.label" class="stat-card">
                <span class="stat-value" :class="{ 'mono': c.mono, 'stat-value-sm': c.mono }">{{ c.value }}</span>
                <span class="stat-label">{{ c.label }}</span>
              </div>
            </div>
            <p v-else-if="!systemError" class="text-sm text-[#59636e]">暂无数据</p>
          </div>
        </section>

        <section v-else-if="active === 'runner'" class="rounded-md border border-[#d1d9e0] bg-white p-4 dark:border-[#3d444d] dark:bg-[#0d1117]">
          <div class="panel-toolbar">
            <button class="inline-flex h-8 items-center rounded-md bg-[#2da44e] px-3 text-sm font-semibold text-white hover:bg-[#2c974b] disabled:opacity-50" type="button" @click="openRunnerModal">新建 Runner</button>
            <button class="inline-flex h-8 items-center rounded-md border border-[#d1d9e0] px-3 text-sm text-[#1f2328] hover:bg-[#f6f8fa] dark:border-[#30363d] dark:text-[#e6edf3] dark:hover:bg-[#21262d]" type="button" @click="loadRunners">刷新</button>
          </div>

          <p v-if="runnersError" class="text-sm text-[#cf222e]">{{ runnersError }}</p>
          <p v-if="runnersLoading" class="text-sm text-[#59636e]">加载中…</p>
          <div v-else-if="runners.length" class="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>runner_id</th>
                  <th>名称</th>
                  <th>标签</th>
                  <th>状态</th>
                  <th>最后在线</th>
                  <th>版本</th>
                  <th>系统 / 架构</th>
                  <th>操作</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="r in runners" :key="r.id">
                  <td class="mono">{{ r.runner_id }}</td>
                  <td>{{ r.name || '-' }}</td>
                  <td>
                    <span v-if="runnerLabels(r).length" class="labels">
                      <span v-for="l in runnerLabels(r)" :key="l" class="label">{{ l }}</span>
                    </span>
                    <span v-else class="text-sm text-[#59636e]">-</span>
                  </td>
                  <td>
                    <span class="status" :class="'status-' + r.status">{{ runnerStatusLabel(r.status) }}</span>
                  </td>
                  <td>{{ formatTime(r.last_seen_at) }}</td>
                  <td class="mono">{{ r.version || '-' }}</td>
                  <td class="mono">{{ r.os || '-' }} / {{ r.arch || '-' }}</td>
                  <td class="row-actions">
                    <button class="link-btn" type="button" @click="openEdit(r)">编辑</button>
                    <button class="link-btn" type="button" @click="resetToken(r)">重置 Token</button>
                    <button class="link-btn danger" type="button" @click="deleteRunner(r)">删除</button>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
          <p v-else-if="!runnersError" class="text-sm text-[#59636e]">暂无 Runner</p>
        </section>

        <section v-else-if="active === 'audit'" class="panel audit-panel">
          <div class="filters">
            <label class="filter">
              <span>事件类型</span>
              <input v-model="auditFilters.event" type="text" placeholder="前缀匹配，如 repo." @change="applyFilters" />
            </label>
            <label class="filter">
              <span>用户 ID</span>
              <input v-model="auditFilters.actor_id" type="number" min="1" placeholder="用户 ID" @change="applyFilters" />
            </label>
            <label class="filter">
              <span>结果</span>
              <select v-model="auditFilters.result" @change="applyFilters">
                <option value="">全部</option>
                <option value="success">success</option>
                <option value="failure">failure</option>
                <option value="denied">denied</option>
              </select>
            </label>
            <label class="filter">
              <span>开始时间</span>
              <input v-model="auditFilters.from" type="datetime-local" @change="applyFilters" />
            </label>
            <label class="filter">
              <span>结束时间</span>
              <input v-model="auditFilters.to" type="datetime-local" @change="applyFilters" />
            </label>
            <div class="filter filter-actions">
              <button class="inline-flex h-8 items-center rounded-md border border-[#d1d9e0] px-3 text-sm text-[#1f2328] hover:bg-[#f6f8fa] dark:border-[#30363d] dark:text-[#e6edf3] dark:hover:bg-[#21262d]" type="button" @click="applyFilters">查询</button>
              <button class="inline-flex h-8 items-center rounded-md border border-[#d1d9e0] px-3 text-sm text-[#1f2328] hover:bg-[#f6f8fa] dark:border-[#30363d] dark:text-[#e6edf3] dark:hover:bg-[#21262d]" type="button" @click="loadAudit">刷新</button>
            </div>
          </div>

          <div class="panel-toolbar">
            <div class="export-group">
              <select v-model="exportFormat" class="format-select">
                <option value="csv">CSV</option>
                <option value="json">JSON</option>
              </select>
              <button class="inline-flex h-8 items-center rounded-md border border-[#d1d9e0] px-3 text-sm text-[#1f2328] hover:bg-[#f6f8fa] dark:border-[#30363d] dark:text-[#e6edf3] dark:hover:bg-[#21262d]" type="button" @click="exportAudit">导出</button>
            </div>
            <div class="archive-group">
              <label class="inline-field">
                <span>保留天数</span>
                <input v-model.number="archiveRetentionDays" type="number" min="1" />
              </label>
              <button class="inline-flex h-8 items-center rounded-md border border-[#d1d9e0] px-3 text-sm text-[#1f2328] hover:bg-[#f6f8fa] dark:border-[#30363d] dark:text-[#e6edf3] dark:hover:bg-[#21262d]" type="button" :disabled="archiving" @click="archiveAudit">
                {{ archiving ? '归档中…' : '手动归档' }}
              </button>
            </div>
          </div>
          <p v-if="archiveMessage" class="notice">{{ archiveMessage }}</p>
          <p v-if="archiveError" class="text-sm text-[#cf222e]">{{ archiveError }}</p>

          <p v-if="auditError" class="text-sm text-[#cf222e]">{{ auditError }}</p>
          <p v-if="loadingAudit" class="text-sm text-[#59636e]">加载中…</p>
          <template v-else-if="auditLogs.length">
            <div class="table-wrap">
              <table>
                <thead>
                  <tr>
                    <th>时间</th>
                    <th>事件</th>
                    <th>用户ID</th>
                    <th>目标</th>
                    <th>结果</th>
                    <th>IP</th>
                  </tr>
                </thead>
                <tbody>
                  <template v-for="log in auditLogs" :key="log.id">
                    <tr class="clickable" @click="toggleExpand(log.id)">
                      <td>{{ auditTime(log) }}</td>
                      <td class="mono">{{ log.event || '-' }}</td>
                      <td>{{ log.actor_id ?? '-' }}</td>
                      <td>{{ auditTargetDisplay(log) }}</td>
                      <td>
                        <span class="status" :class="'result-' + log.result">{{ resultLabel(log.result) }}</span>
                      </td>
                      <td class="mono">{{ log.actor_ip || '-' }}</td>
                    </tr>
                    <tr v-if="expandedId === log.id" class="detail-row">
                      <td colspan="6">
                        <div class="detail">
                          <div class="detail-values">
                            <div class="value-col">
                              <span class="value-label">旧值 old_value</span>
                              <pre v-if="log.old_value" class="value-pre">{{ prettyValue(log.old_value) }}</pre>
                              <span v-else class="text-sm text-[#59636e]">-</span>
                            </div>
                            <div class="value-col">
                              <span class="value-label">新值 new_value</span>
                              <pre v-if="log.new_value" class="value-pre">{{ prettyValue(log.new_value) }}</pre>
                              <span v-else class="text-sm text-[#59636e]">-</span>
                            </div>
                          </div>
                          <div class="detail-meta">
                            <div><span class="meta-label">request_id</span><span class="mono">{{ log.request_id || '-' }}</span></div>
                            <div><span class="meta-label">user_agent</span><span class="mono">{{ log.actor_user_agent || '-' }}</span></div>
                            <div v-if="log.error_message"><span class="meta-label">error_message</span><span class="mono err-text">{{ log.error_message }}</span></div>
                          </div>
                        </div>
                      </td>
                    </tr>
                  </template>
                </tbody>
              </table>
            </div>
            <div class="pagination">
              <button class="inline-flex h-8 items-center rounded-md border border-[#d1d9e0] px-3 text-sm text-[#1f2328] hover:bg-[#f6f8fa] dark:border-[#30363d] dark:text-[#e6edf3] dark:hover:bg-[#21262d]" type="button" :disabled="auditOffset === 0" @click="prevPage">上一页</button>
              <span class="text-sm text-[#59636e]">第 {{ Math.floor(auditOffset / PAGE_SIZE) + 1 }} 页</span>
              <button class="inline-flex h-8 items-center rounded-md border border-[#d1d9e0] px-3 text-sm text-[#1f2328] hover:bg-[#f6f8fa] dark:border-[#30363d] dark:text-[#e6edf3] dark:hover:bg-[#21262d]" type="button" :disabled="auditLogs.length < PAGE_SIZE" @click="nextPage">下一页</button>
            </div>
          </template>
          <p v-else-if="!auditError" class="text-sm text-[#59636e]">暂无数据</p>

          <div class="subsection">
            <h3>统计（近 30 天）</h3>
            <p v-if="auditStatsError" class="text-sm text-[#cf222e]">{{ auditStatsError }}</p>
            <p v-if="loadingAuditStats" class="text-sm text-[#59636e]">加载中…</p>
            <template v-else-if="auditStats">
              <div class="charts">
                <div class="chart-card">
                  <h4>事件 Top 10</h4>
                  <div v-if="topEvents.length" class="bars">
                    <div v-for="e in topEvents" :key="e.key" class="bar-row">
                      <span class="bar-label mono" :title="e.key">{{ e.key }}</span>
                      <span class="bar-track"><span class="bar-fill" :style="{ width: barWidth(e.count, maxEventCount) + '%' }"></span></span>
                      <span class="bar-count">{{ e.count }}</span>
                    </div>
                  </div>
                  <p v-else class="text-sm text-[#59636e]">暂无数据</p>
                </div>
                <div class="chart-card">
                  <h4>每日趋势</h4>
                  <div v-if="byDay30.length" class="chart-svg">
                    <svg :viewBox="`0 0 ${byDay30.length * 20} 120`" preserveAspectRatio="none" role="img">
                      <rect
                        v-for="(d, i) in byDay30"
                        :key="d.day"
                        :x="i * 20 + 2"
                        :y="120 - (d.count / maxDayCount) * 110"
                        width="16"
                        :height="(d.count / maxDayCount) * 110"
                        class="day-bar"
                      >
                        <title>{{ d.day }}: {{ d.count }}</title>
                      </rect>
                    </svg>
                    <div class="axis">
                      <span>{{ byDay30[0]?.day }}</span>
                      <span>{{ byDay30[byDay30.length - 1]?.day }}</span>
                    </div>
                  </div>
                  <p v-else class="text-sm text-[#59636e]">暂无数据</p>
                </div>
                <div class="chart-card">
                  <h4>活跃用户</h4>
                  <ul v-if="topActors.length" class="actor-list">
                    <li v-for="a in topActors" :key="a.key">
                      <span class="mono">{{ a.key }}</span>
                      <span class="bar-count">{{ a.count }}</span>
                    </li>
                  </ul>
                  <p v-else class="text-sm text-[#59636e]">暂无数据</p>
                </div>
              </div>
            </template>
          </div>

          <div class="subsection">
            <h3>告警规则</h3>
            <p v-if="alertRulesError" class="text-sm text-[#cf222e]">{{ alertRulesError }}</p>
            <p v-if="loadingAlertRules" class="text-sm text-[#59636e]">加载中…</p>
            <div v-else-if="alertRules.length" class="table-wrap">
              <table>
                <thead>
                  <tr>
                    <th>名称</th>
                    <th>事件</th>
                    <th>阈值</th>
                    <th>窗口(秒)</th>
                    <th>通道</th>
                    <th>目标</th>
                    <th>状态</th>
                    <th>操作</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="rule in alertRules" :key="rule.id">
                    <td>{{ rule.name }}</td>
                    <td class="mono">{{ rule.event }}</td>
                    <td>{{ rule.threshold }}</td>
                    <td>{{ rule.window_seconds }}</td>
                    <td>{{ rule.channel }}</td>
                    <td class="mono">{{ rule.target || '-' }}</td>
                    <td>
                      <span class="status" :class="rule.enabled ? 'status-active' : 'status-offline'">
                        {{ rule.enabled ? '启用' : '停用' }}
                      </span>
                    </td>
                    <td>
                      <button class="link-btn danger" type="button" @click="deleteAlertRule(rule)">删除</button>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
            <p v-else class="text-sm text-[#59636e]">暂无告警规则</p>

            <form class="alert-form" @submit.prevent="createAlertRule">
              <label class="filter">
                <span>名称</span>
                <input v-model="alertForm.name" type="text" required />
              </label>
              <label class="filter">
                <span>事件</span>
                <input v-model="alertForm.event" type="text" placeholder="如 login.failure" required />
              </label>
              <label class="filter">
                <span>阈值</span>
                <input v-model.number="alertForm.threshold" type="number" min="1" />
              </label>
              <label class="filter">
                <span>窗口(秒)</span>
                <input v-model.number="alertForm.window_seconds" type="number" min="1" />
              </label>
              <label class="filter">
                <span>通道</span>
                <select v-model="alertForm.channel">
                  <option value="email">email</option>
                  <option value="webhook">webhook</option>
                </select>
              </label>
              <label class="filter">
                <span>目标</span>
                <input v-model="alertForm.target" type="text" placeholder="邮箱或 URL" />
              </label>
              <label class="filter checkbox-field">
                <input v-model="alertForm.enabled" type="checkbox" />
                <span>启用</span>
              </label>
              <div class="filter filter-actions">
                <button class="inline-flex h-8 items-center rounded-md bg-[#2da44e] px-3 text-sm font-semibold text-white hover:bg-[#2c974b] disabled:opacity-50" type="submit" :disabled="alertSaving">
                  {{ alertSaving ? '创建中…' : '新建规则' }}
                </button>
              </div>
            </form>
            <p v-if="alertFormError" class="text-sm text-[#cf222e]">{{ alertFormError }}</p>
          </div>
        </section>

        <section v-else class="rounded-md border border-[#d1d9e0] bg-white p-4 dark:border-[#3d444d] dark:bg-[#0d1117]">
          <p class="text-sm text-[#59636e]">暂无数据</p>
        </section>
    </div>

    <div v-if="runnerModal" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4" @click.self="closeRunnerModal">
      <div class="w-full max-w-lg rounded-md border border-[#d1d9e0] bg-white p-5 shadow-lg dark:border-[#3d444d] dark:bg-[#0d1117]">
        <h3 v-if="!createdRunner">新建 Runner</h3>
        <h3 v-else>Runner 已创建</h3>

        <template v-if="!createdRunner">
          <label class="flex flex-col gap-1.5 text-sm text-[#1f2328] dark:text-[#e6edf3]">
            <span>名称</span>
            <input v-model="runnerForm.name" type="text" placeholder="my-runner" />
          </label>
          <label class="flex flex-col gap-1.5 text-sm text-[#1f2328] dark:text-[#e6edf3]">
            <span>标签（逗号分隔）</span>
            <input v-model="runnerForm.labels" type="text" placeholder="linux, amd64, docker" />
          </label>
          <p v-if="runnerFormError" class="text-sm text-[#cf222e]">{{ runnerFormError }}</p>
          <div class="mt-4 flex justify-end gap-2">
            <button class="inline-flex h-8 items-center rounded-md border border-[#d1d9e0] px-3 text-sm text-[#1f2328] hover:bg-[#f6f8fa] dark:border-[#30363d] dark:text-[#e6edf3] dark:hover:bg-[#21262d]" type="button" @click="closeRunnerModal">取消</button>
            <button class="inline-flex h-8 items-center rounded-md bg-[#2da44e] px-3 text-sm font-semibold text-white hover:bg-[#2c974b] disabled:opacity-50" type="button" :disabled="runnerSaving" @click="createRunner">
              {{ runnerSaving ? '创建中…' : '创建' }}
            </button>
          </div>
        </template>

        <template v-else>
          <p class="text-sm text-[#59636e]">请立即保存以下凭据，Token 仅显示一次。</p>
          <div class="token-box">
            <div class="token-row">
              <span class="token-label">runner_id</span>
              <code>{{ createdRunner.runner_id }}</code>
              <button class="link-btn" type="button" @click="copyText(createdRunner.runner_id, 'id')">
                {{ copiedKey === 'id' ? '已复制' : '复制' }}
              </button>
            </div>
            <div class="token-row">
              <span class="token-label">token</span>
              <code class="token">{{ createdRunner.token }}</code>
              <button class="link-btn" type="button" @click="copyText(createdRunner.token, 'token')">
                {{ copiedKey === 'token' ? '已复制' : '复制' }}
              </button>
            </div>
          </div>
          <p class="snippet-label">配置片段</p>
          <pre class="snippet">{{ runnerConfigSnippet(createdRunner) }}</pre>
          <div class="mt-4 flex justify-end gap-2">
            <button
              class="inline-flex h-8 items-center rounded-md border border-[#d1d9e0] px-3 text-sm text-[#1f2328] hover:bg-[#f6f8fa] dark:border-[#30363d] dark:text-[#e6edf3] dark:hover:bg-[#21262d]"
              type="button"
              @click="copyText(runnerConfigSnippet(createdRunner), 'snippet')"
            >
              {{ copiedKey === 'snippet' ? '已复制配置' : '复制配置' }}
            </button>
            <button class="inline-flex h-8 items-center rounded-md bg-[#2da44e] px-3 text-sm font-semibold text-white hover:bg-[#2c974b] disabled:opacity-50" type="button" @click="closeRunnerModal">完成</button>
          </div>
        </template>
      </div>
    </div>

    <div v-if="editModal" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4" @click.self="editModal = false">
      <div class="w-full max-w-lg rounded-md border border-[#d1d9e0] bg-white p-5 shadow-lg dark:border-[#3d444d] dark:bg-[#0d1117]">
        <h3>编辑 Runner</h3>
        <label class="flex flex-col gap-1.5 text-sm text-[#1f2328] dark:text-[#e6edf3]">
          <span>名称</span>
          <input v-model="editForm.name" type="text" />
        </label>
        <label class="flex flex-col gap-1.5 text-sm text-[#1f2328] dark:text-[#e6edf3]">
          <span>状态</span>
          <select v-model="editForm.status">
            <option value="active">在线</option>
            <option value="offline">离线</option>
            <option value="disabled">已禁用</option>
          </select>
        </label>
        <label class="flex flex-col gap-1.5 text-sm text-[#1f2328] dark:text-[#e6edf3]">
          <span>标签（逗号分隔）</span>
          <input v-model="editForm.labels" type="text" />
        </label>
        <p v-if="editError" class="text-sm text-[#cf222e]">{{ editError }}</p>
        <div class="mt-4 flex justify-end gap-2">
          <button class="inline-flex h-8 items-center rounded-md border border-[#d1d9e0] px-3 text-sm text-[#1f2328] hover:bg-[#f6f8fa] dark:border-[#30363d] dark:text-[#e6edf3] dark:hover:bg-[#21262d]" type="button" @click="editModal = false">取消</button>
          <button class="inline-flex h-8 items-center rounded-md bg-[#2da44e] px-3 text-sm font-semibold text-white hover:bg-[#2c974b] disabled:opacity-50" type="button" :disabled="editSaving" @click="saveEdit">
            {{ editSaving ? '保存中…' : '保存' }}
          </button>
        </div>
      </div>
    </div>
  </AppLayout>
  
</template>
