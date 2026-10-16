<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import axios from 'axios'
import TopNav from '../components/layout/TopNav.vue'
import Sidebar from '../components/layout/Sidebar.vue'
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

type TabKey = 'users' | 'repos' | 'tasks' | 'runner' | 'audit' | 'system'

interface Tab {
  key: TabKey
  label: string
}

const tabs: Tab[] = [
  { key: 'users', label: '用户' },
  { key: 'repos', label: '仓库' },
  { key: 'tasks', label: '任务' },
  { key: 'runner', label: 'Runner' },
  { key: 'audit', label: '审计日志' },
  { key: 'system', label: '系统状态' },
]

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

function errMessage(e: unknown): string {
  if (axios.isAxiosError(e)) {
    const data = e.response?.data as { error?: string } | undefined
    if (data?.error) return data.error
    if (e.response?.status) return `请求失败（HTTP ${e.response.status}）`
    return e.message
  }
  return e instanceof Error ? e.message : '未知错误'
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
  if (key === 'system' && !stats.value && !loadingStats.value) loadStats()
  if (key === 'audit') {
    if (!auditLoaded.value && !loadingAudit.value) loadAudit()
    if (!auditStats.value && !loadingAuditStats.value) loadAuditStats()
    if (!alertRulesLoaded.value && !loadingAlertRules.value) loadAlertRules()
  }
  if (key === 'runner' && !runnersLoaded.value && !runnersLoading.value) loadRunners()
}

onMounted(() => {
  loadStats()
})
</script>

<template>
  <div class="admin-page">
    <TopNav variant="app" />
    <div class="body">
      <Sidebar />
      <main class="admin">
        <h2>管理面板</h2>
        <nav class="tabs" role="tablist">
          <button
            v-for="t in tabs"
            :key="t.key"
            class="tab"
            :class="{ active: active === t.key }"
            type="button"
            role="tab"
            :aria-selected="active === t.key"
            @click="selectTab(t.key)"
          >
            {{ t.label }}
          </button>
        </nav>

        <section v-if="active === 'system'" class="panel">
          <p v-if="statsError" class="error">{{ statsError }}</p>
          <p v-if="loadingStats" class="muted">加载中…</p>
          <template v-else-if="stats">
            <div class="cards">
              <div v-for="c in statCards" :key="c.label" class="stat-card">
                <span class="stat-value">{{ c.value }}</span>
                <span class="stat-label">{{ c.label }}</span>
              </div>
            </div>
            <p class="uptime">运行时间：{{ formatUptime(stats.uptime_seconds) }}</p>
          </template>
          <p v-else-if="!statsError" class="muted">暂无数据</p>
        </section>

        <section v-else-if="active === 'runner'" class="panel">
          <div class="panel-toolbar">
            <button class="btn primary" type="button" @click="openRunnerModal">新建 Runner</button>
            <button class="btn" type="button" @click="loadRunners">刷新</button>
          </div>

          <p v-if="runnersError" class="error">{{ runnersError }}</p>
          <p v-if="runnersLoading" class="muted">加载中…</p>
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
                    <span v-else class="muted">-</span>
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
          <p v-else-if="!runnersError" class="muted">暂无 Runner</p>
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
              <button class="btn" type="button" @click="applyFilters">查询</button>
              <button class="btn" type="button" @click="loadAudit">刷新</button>
            </div>
          </div>

          <div class="panel-toolbar">
            <div class="export-group">
              <select v-model="exportFormat" class="format-select">
                <option value="csv">CSV</option>
                <option value="json">JSON</option>
              </select>
              <button class="btn" type="button" @click="exportAudit">导出</button>
            </div>
            <div class="archive-group">
              <label class="inline-field">
                <span>保留天数</span>
                <input v-model.number="archiveRetentionDays" type="number" min="1" />
              </label>
              <button class="btn" type="button" :disabled="archiving" @click="archiveAudit">
                {{ archiving ? '归档中…' : '手动归档' }}
              </button>
            </div>
          </div>
          <p v-if="archiveMessage" class="notice">{{ archiveMessage }}</p>
          <p v-if="archiveError" class="error">{{ archiveError }}</p>

          <p v-if="auditError" class="error">{{ auditError }}</p>
          <p v-if="loadingAudit" class="muted">加载中…</p>
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
                              <span v-else class="muted">-</span>
                            </div>
                            <div class="value-col">
                              <span class="value-label">新值 new_value</span>
                              <pre v-if="log.new_value" class="value-pre">{{ prettyValue(log.new_value) }}</pre>
                              <span v-else class="muted">-</span>
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
              <button class="btn" type="button" :disabled="auditOffset === 0" @click="prevPage">上一页</button>
              <span class="muted">第 {{ Math.floor(auditOffset / PAGE_SIZE) + 1 }} 页</span>
              <button class="btn" type="button" :disabled="auditLogs.length < PAGE_SIZE" @click="nextPage">下一页</button>
            </div>
          </template>
          <p v-else-if="!auditError" class="muted">暂无数据</p>

          <div class="subsection">
            <h3>统计（近 30 天）</h3>
            <p v-if="auditStatsError" class="error">{{ auditStatsError }}</p>
            <p v-if="loadingAuditStats" class="muted">加载中…</p>
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
                  <p v-else class="muted">暂无数据</p>
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
                  <p v-else class="muted">暂无数据</p>
                </div>
                <div class="chart-card">
                  <h4>活跃用户</h4>
                  <ul v-if="topActors.length" class="actor-list">
                    <li v-for="a in topActors" :key="a.key">
                      <span class="mono">{{ a.key }}</span>
                      <span class="bar-count">{{ a.count }}</span>
                    </li>
                  </ul>
                  <p v-else class="muted">暂无数据</p>
                </div>
              </div>
            </template>
          </div>

          <div class="subsection">
            <h3>告警规则</h3>
            <p v-if="alertRulesError" class="error">{{ alertRulesError }}</p>
            <p v-if="loadingAlertRules" class="muted">加载中…</p>
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
            <p v-else class="muted">暂无告警规则</p>

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
                <button class="btn primary" type="submit" :disabled="alertSaving">
                  {{ alertSaving ? '创建中…' : '新建规则' }}
                </button>
              </div>
            </form>
            <p v-if="alertFormError" class="error">{{ alertFormError }}</p>
          </div>
        </section>

        <section v-else class="panel">
          <p class="muted">暂无数据</p>
        </section>
      </main>
    </div>

    <div v-if="runnerModal" class="modal-overlay" @click.self="closeRunnerModal">
      <div class="modal">
        <h3 v-if="!createdRunner">新建 Runner</h3>
        <h3 v-else>Runner 已创建</h3>

        <template v-if="!createdRunner">
          <label class="field">
            <span>名称</span>
            <input v-model="runnerForm.name" type="text" placeholder="my-runner" />
          </label>
          <label class="field">
            <span>标签（逗号分隔）</span>
            <input v-model="runnerForm.labels" type="text" placeholder="linux, amd64, docker" />
          </label>
          <p v-if="runnerFormError" class="error">{{ runnerFormError }}</p>
          <div class="modal-actions">
            <button class="btn" type="button" @click="closeRunnerModal">取消</button>
            <button class="btn primary" type="button" :disabled="runnerSaving" @click="createRunner">
              {{ runnerSaving ? '创建中…' : '创建' }}
            </button>
          </div>
        </template>

        <template v-else>
          <p class="muted">请立即保存以下凭据，Token 仅显示一次。</p>
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
          <div class="modal-actions">
            <button
              class="btn"
              type="button"
              @click="copyText(runnerConfigSnippet(createdRunner), 'snippet')"
            >
              {{ copiedKey === 'snippet' ? '已复制配置' : '复制配置' }}
            </button>
            <button class="btn primary" type="button" @click="closeRunnerModal">完成</button>
          </div>
        </template>
      </div>
    </div>

    <div v-if="editModal" class="modal-overlay" @click.self="editModal = false">
      <div class="modal">
        <h3>编辑 Runner</h3>
        <label class="field">
          <span>名称</span>
          <input v-model="editForm.name" type="text" />
        </label>
        <label class="field">
          <span>状态</span>
          <select v-model="editForm.status">
            <option value="active">在线</option>
            <option value="offline">离线</option>
            <option value="disabled">已禁用</option>
          </select>
        </label>
        <label class="field">
          <span>标签（逗号分隔）</span>
          <input v-model="editForm.labels" type="text" />
        </label>
        <p v-if="editError" class="error">{{ editError }}</p>
        <div class="modal-actions">
          <button class="btn" type="button" @click="editModal = false">取消</button>
          <button class="btn primary" type="button" :disabled="editSaving" @click="saveEdit">
            {{ editSaving ? '保存中…' : '保存' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.body {
  display: flex;
  min-height: calc(100vh - 57px);
}
.admin {
  flex: 1;
  padding: 1.5rem;
  min-width: 0;
}
.admin h2 {
  margin-top: 0;
}
.tabs {
  display: flex;
  gap: 0.25rem;
  border-bottom: 1px solid var(--border);
  margin-bottom: 1.25rem;
  flex-wrap: wrap;
}
.tab {
  padding: 0.5rem 0.9rem;
  border: none;
  background: transparent;
  color: var(--muted);
  font-size: 0.95rem;
  border-bottom: 2px solid transparent;
  margin-bottom: -1px;
}
.tab:hover {
  color: var(--color-fg);
}
.tab.active {
  color: var(--primary);
  border-bottom-color: var(--primary);
  font-weight: 600;
}
.cards {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
  gap: 0.75rem;
}
.stat-card {
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: 1rem;
  background: var(--card);
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}
.stat-value {
  font-size: 1.6rem;
  font-weight: 700;
  color: var(--primary);
}
.stat-label {
  font-size: 0.85rem;
  color: var(--muted);
}
.uptime {
  margin-top: 1rem;
  color: var(--muted);
  font-size: 0.9rem;
}
.table-wrap {
  overflow-x: auto;
  border: 1px solid var(--border);
  border-radius: 10px;
}
table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.9rem;
}
th,
td {
  text-align: left;
  padding: 0.55rem 0.75rem;
  border-bottom: 1px solid var(--border);
  white-space: nowrap;
}
th {
  color: var(--muted);
  font-weight: 600;
  background: var(--color-bg);
}
tbody tr:last-child td {
  border-bottom: none;
}
.error {
  color: #d1242f;
  background: rgba(209, 36, 47, 0.08);
  border: 1px solid rgba(209, 36, 47, 0.4);
  border-radius: 8px;
  padding: 0.6rem 0.8rem;
  font-size: 0.9rem;
}
.muted {
  color: var(--muted);
}
.panel-toolbar {
  display: flex;
  gap: 0.5rem;
  margin-bottom: 1rem;
}
.btn {
  padding: 0.4rem 0.85rem;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--card);
  color: var(--color-fg);
  font: inherit;
}
.btn:hover {
  background: var(--hover-bg);
  border-color: var(--primary);
  color: var(--primary);
}
.btn.primary {
  background: var(--primary);
  border-color: var(--primary);
  color: #fff;
}
.btn.primary:hover {
  background: var(--primary-hover);
  color: #fff;
}
.btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
.mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 0.82rem;
}
.labels {
  display: inline-flex;
  gap: 0.3rem;
  flex-wrap: wrap;
}
.label {
  padding: 0.05rem 0.5rem;
  border: 1px solid var(--border);
  border-radius: 999px;
  font-size: 0.75rem;
  color: var(--muted);
}
.status {
  padding: 0.1rem 0.55rem;
  border-radius: 999px;
  font-size: 0.78rem;
  border: 1px solid var(--border);
  color: var(--muted);
}
.status-active {
  color: var(--success);
  border-color: var(--success);
  background: var(--success-soft);
}
.status-offline {
  color: var(--muted);
  border-color: var(--border);
  background: var(--color-bg);
}
.status-disabled {
  color: #d1242f;
  border-color: rgba(209, 36, 47, 0.5);
  background: rgba(209, 36, 47, 0.08);
}
.row-actions {
  display: flex;
  gap: 0.6rem;
}
.link-btn {
  border: none;
  background: transparent;
  color: var(--primary);
  font: inherit;
  padding: 0;
}
.link-btn:hover {
  text-decoration: underline;
}
.link-btn.danger {
  color: #d1242f;
}
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.45);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 50;
  padding: 1rem;
}
.modal {
  background: var(--card);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 1.25rem;
  width: min(520px, 100%);
  max-height: 90vh;
  overflow-y: auto;
}
.modal h3 {
  margin: 0 0 1rem;
}
.field {
  display: flex;
  flex-direction: column;
  gap: 0.3rem;
  margin-bottom: 0.75rem;
  font-size: 0.85rem;
  color: var(--muted);
}
.field input,
.field select {
  padding: 0.45rem 0.7rem;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--color-bg);
  color: var(--color-fg);
  font: inherit;
}
.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
  margin-top: 1rem;
}
.token-box {
  border: 1px solid var(--accent);
  background: var(--accent-soft);
  border-radius: 10px;
  padding: 0.75rem;
  display: flex;
  flex-direction: column;
  gap: 0.6rem;
}
.token-row {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  flex-wrap: wrap;
}
.token-label {
  font-size: 0.75rem;
  color: var(--muted);
  min-width: 62px;
}
.token-row code {
  flex: 1;
  word-break: break-all;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 0.82rem;
}
.snippet-label {
  margin: 0.9rem 0 0.35rem;
  font-size: 0.8rem;
  color: var(--muted);
}
.snippet {
  margin: 0;
  padding: 0.75rem;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--color-bg);
  overflow-x: auto;
  font-size: 0.82rem;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
}
.filters {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
  align-items: flex-end;
  margin-bottom: 1rem;
}
.filter {
  display: flex;
  flex-direction: column;
  gap: 0.3rem;
  font-size: 0.8rem;
  color: var(--muted);
}
.filter input,
.filter select {
  padding: 0.4rem 0.6rem;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--color-bg);
  color: var(--color-fg);
  font: inherit;
}
.filter-actions {
  flex-direction: row;
  align-items: center;
  gap: 0.5rem;
}
.checkbox-field {
  flex-direction: row;
  align-items: center;
  gap: 0.4rem;
}
.export-group,
.archive-group {
  display: flex;
  gap: 0.5rem;
  align-items: center;
}
.format-select {
  padding: 0.4rem 0.6rem;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--color-bg);
  color: var(--color-fg);
  font: inherit;
}
.inline-field {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  font-size: 0.8rem;
  color: var(--muted);
}
.inline-field input {
  width: 80px;
  padding: 0.4rem 0.6rem;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--color-bg);
  color: var(--color-fg);
  font: inherit;
}
.notice {
  color: var(--success);
  background: var(--success-soft);
  border: 1px solid var(--success);
  border-radius: 8px;
  padding: 0.6rem 0.8rem;
  font-size: 0.9rem;
}
.panel-toolbar {
  flex-wrap: wrap;
  justify-content: space-between;
}
.clickable {
  cursor: pointer;
}
.clickable:hover {
  background: var(--hover-bg);
}
.result-success {
  color: var(--success);
  border-color: var(--success);
  background: var(--success-soft);
}
.result-failure {
  color: #d1242f;
  border-color: rgba(209, 36, 47, 0.5);
  background: rgba(209, 36, 47, 0.08);
}
.result-denied {
  color: var(--accent);
  border-color: var(--accent);
  background: var(--accent-soft);
}
.detail-row td {
  background: var(--color-bg);
  white-space: normal;
}
.detail {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  padding: 0.5rem 0;
}
.detail-values {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.75rem;
}
.value-col {
  display: flex;
  flex-direction: column;
  gap: 0.3rem;
  min-width: 0;
}
.value-label {
  font-size: 0.78rem;
  font-weight: 600;
  color: var(--muted);
}
.value-pre {
  margin: 0;
  padding: 0.6rem;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--card);
  overflow-x: auto;
  max-height: 260px;
  font-size: 0.8rem;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
}
.detail-meta {
  display: flex;
  flex-direction: column;
  gap: 0.3rem;
  font-size: 0.82rem;
}
.detail-meta > div {
  display: flex;
  gap: 0.6rem;
  word-break: break-all;
}
.meta-label {
  color: var(--muted);
  min-width: 92px;
}
.err-text {
  color: #d1242f;
}
.pagination {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  margin-top: 0.9rem;
}
.subsection {
  margin-top: 2rem;
  border-top: 1px solid var(--border);
  padding-top: 1.25rem;
}
.subsection h3 {
  margin: 0 0 1rem;
}
.subsection h4 {
  margin: 0 0 0.6rem;
  font-size: 0.9rem;
  color: var(--muted);
}
.charts {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 1rem;
}
.chart-card {
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: 1rem;
  background: var(--card);
  min-width: 0;
}
.bars {
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
}
.bar-row {
  display: grid;
  grid-template-columns: 120px 1fr 40px;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.8rem;
}
.bar-label {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.bar-track {
  height: 12px;
  background: var(--color-bg);
  border-radius: 999px;
  overflow: hidden;
}
.bar-fill {
  display: block;
  height: 100%;
  background: var(--primary);
  border-radius: 999px;
}
.bar-count {
  text-align: right;
  color: var(--muted);
}
.chart-svg {
  width: 100%;
}
.chart-svg svg {
  width: 100%;
  height: 120px;
  display: block;
}
.day-bar {
  fill: var(--primary);
}
.axis {
  display: flex;
  justify-content: space-between;
  font-size: 0.72rem;
  color: var(--muted);
  margin-top: 0.25rem;
}
.actor-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
}
.actor-list li {
  display: flex;
  justify-content: space-between;
  font-size: 0.82rem;
  padding-bottom: 0.35rem;
  border-bottom: 1px solid var(--border);
}
.alert-form {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
  align-items: flex-end;
  margin-top: 1rem;
  padding: 1rem;
  border: 1px solid var(--border);
  border-radius: 10px;
  background: var(--card);
}
@media (max-width: 768px) {
  .body {
    flex-direction: column;
  }
  .detail-values {
    grid-template-columns: 1fr;
  }
}
</style>
