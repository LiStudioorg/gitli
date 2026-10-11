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

type TabKey = 'users' | 'repos' | 'tasks' | 'audit' | 'system'

interface Tab {
  key: TabKey
  label: string
}

const tabs: Tab[] = [
  { key: 'users', label: '用户' },
  { key: 'repos', label: '仓库' },
  { key: 'tasks', label: '任务' },
  { key: 'audit', label: '审计日志' },
  { key: 'system', label: '系统状态' },
]

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
@media (max-width: 768px) {
  .body {
    flex-direction: column;
  }
}
</style>
