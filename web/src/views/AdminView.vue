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
  ID?: number
  id?: number
  ActorID?: number | null
  actor_id?: number | null
  Action?: string
  action?: string
  TargetType?: string
  target_type?: string
  TargetID?: number | null
  target_id?: number | null
  IP?: string
  ip?: string
  Detail?: string
  detail?: string
  CreatedAt?: string
  created_at?: string
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
const auditLogs = ref<AuditLog[]>([])
const statsError = ref('')
const auditError = ref('')
const loadingStats = ref(false)
const loadingAudit = ref(false)

const auditLoaded = ref(false)

function errMessage(e: unknown): string {
  if (axios.isAxiosError(e)) {
    const data = e.response?.data as { error?: string } | undefined
    if (data?.error) return data.error
    if (e.response?.status) return `请求失败（HTTP ${e.response.status}）`
    return e.message
  }
  return e instanceof Error ? e.message : '未知错误'
}

function actorId(log: AuditLog): number | string {
  const v = log.ActorID ?? log.actor_id
  return v == null ? '-' : v
}

function auditAction(log: AuditLog): string {
  return log.Action ?? log.action ?? '-'
}

function auditTarget(log: AuditLog): string {
  return log.TargetType ?? log.target_type ?? '-'
}

function auditIP(log: AuditLog): string {
  return log.IP ?? log.ip ?? '-'
}

function auditTime(log: AuditLog): string {
  const t = log.CreatedAt ?? log.created_at
  if (!t) return '-'
  const d = new Date(t)
  return Number.isNaN(d.getTime()) ? t : d.toLocaleString()
}

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

async function loadAudit() {
  loadingAudit.value = true
  auditError.value = ''
  try {
    const { data } = await api.get<{ logs: AuditLog[] }>('/admin/audit', {
      params: { limit: 50 },
    })
    auditLogs.value = data.logs ?? []
    auditLoaded.value = true
  } catch (e) {
    auditError.value = errMessage(e)
  } finally {
    loadingAudit.value = false
  }
}

function selectTab(key: TabKey) {
  active.value = key
  if (key === 'system' && !stats.value && !loadingStats.value) loadStats()
  if (key === 'audit' && !auditLoaded.value && !loadingAudit.value) loadAudit()
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

        <section v-else-if="active === 'audit'" class="panel">
          <p v-if="auditError" class="error">{{ auditError }}</p>
          <p v-if="loadingAudit" class="muted">加载中…</p>
          <template v-else-if="auditLogs.length">
            <div class="table-wrap">
              <table>
                <thead>
                  <tr>
                    <th>操作者</th>
                    <th>动作</th>
                    <th>目标类型</th>
                    <th>IP</th>
                    <th>时间</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="(log, i) in auditLogs" :key="log.ID ?? log.id ?? i">
                    <td>{{ actorId(log) }}</td>
                    <td>{{ auditAction(log) }}</td>
                    <td>{{ auditTarget(log) }}</td>
                    <td>{{ auditIP(log) }}</td>
                    <td>{{ auditTime(log) }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </template>
          <p v-else-if="!auditError" class="muted">暂无数据</p>
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
@media (max-width: 768px) {
  .body {
    flex-direction: column;
  }
}
</style>
