<script setup lang="ts">
import { onMounted, ref } from 'vue'
import api from '../../api/client'

interface DeployKey {
  ID: number
  RepoID: number
  Title: string
  Fingerprint: string
  PublicKey: string
  ReadOnly: boolean
  CreatedAt: string
  LastUsedAt: string | null
}

const props = defineProps<{
  owner: string
  repo: string
}>()

const base = () => `/repos/${props.owner}/${props.repo}/keys`

const keys = ref<DeployKey[]>([])
const loading = ref(false)
const error = ref('')
const forbidden = ref(false)

const title = ref('')
const publicKey = ref('')
const readOnly = ref(true)
const creating = ref(false)
const createError = ref('')
const createOk = ref('')

async function load() {
  loading.value = true
  error.value = ''
  forbidden.value = false
  try {
    const { data } = await api.get<DeployKey[]>(base())
    keys.value = data ?? []
  } catch (e) {
    const status = (e as { response?: { status?: number } })?.response?.status
    if (status === 401 || status === 403) {
      forbidden.value = true
    } else {
      error.value = '加载部署密钥失败'
    }
    keys.value = []
  } finally {
    loading.value = false
  }
}

async function create() {
  createError.value = ''
  createOk.value = ''
  if (!title.value.trim()) {
    createError.value = '标题不能为空'
    return
  }
  if (!publicKey.value.trim()) {
    createError.value = '公钥不能为空'
    return
  }
  creating.value = true
  try {
    await api.post(base(), {
      title: title.value.trim(),
      public_key: publicKey.value.trim(),
      read_only: readOnly.value,
    })
    title.value = ''
    publicKey.value = ''
    readOnly.value = true
    createOk.value = '部署密钥已添加'
    await load()
  } catch (e) {
    const status = (e as { response?: { status?: number } })?.response?.status
    if (status === 401 || status === 403) {
      forbidden.value = true
    } else {
      createError.value = '添加部署密钥失败'
    }
  } finally {
    creating.value = false
  }
}

async function toggleReadOnly(key: DeployKey) {
  error.value = ''
  try {
    await api.patch(`${base()}/${key.ID}`, { read_only: !key.ReadOnly })
    key.ReadOnly = !key.ReadOnly
  } catch {
    error.value = '更新权限失败'
  }
}

async function remove(key: DeployKey) {
  if (!window.confirm(`确定要删除部署密钥「${key.Title}」吗？`)) return
  error.value = ''
  try {
    await api.delete(`${base()}/${key.ID}`)
    keys.value = keys.value.filter((k) => k.ID !== key.ID)
  } catch {
    error.value = '删除部署密钥失败'
  }
}

function formatDate(value: string | null): string {
  if (!value) return '从未使用'
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return '-'
  return d.toLocaleString()
}

onMounted(load)
</script>

<template>
  <section class="deploy-keys">
    <div class="panel-head">
      <h3>部署密钥</h3>
      <button v-if="!forbidden" class="chip" type="button" @click="load">刷新</button>
    </div>

    <p v-if="forbidden" class="notice">需要仓库管理员权限</p>

    <template v-else>
      <p v-if="error" class="error">{{ error }}</p>

      <form class="create-form" @submit.prevent="create">
        <h4>添加部署密钥</h4>
        <input v-model="title" type="text" placeholder="标题" class="form-input" />
        <textarea
          v-model="publicKey"
          rows="4"
          placeholder="ssh-ed25519 AAAAC3Nza... user@host"
          class="form-input mono"
        />
        <div class="radio-row">
          <label><input v-model="readOnly" type="radio" :value="true" /> 只读</label>
          <label><input v-model="readOnly" type="radio" :value="false" /> 读写</label>
        </div>
        <p v-if="createError" class="error">{{ createError }}</p>
        <p v-if="createOk" class="ok">{{ createOk }}</p>
        <button class="chip primary" type="submit" :disabled="creating">
          {{ creating ? '提交中...' : '添加密钥' }}
        </button>
      </form>

      <div v-if="loading" class="empty">加载中…</div>
      <ul v-else-if="keys.length" class="item-list">
        <li v-for="k in keys" :key="k.ID" class="item-row">
          <span class="item-title">{{ k.Title }}</span>
          <span class="mono fingerprint">{{ k.Fingerprint }}</span>
          <button
            class="badge"
            type="button"
            :class="k.ReadOnly ? 'ro' : 'rw'"
            :title="k.ReadOnly ? '点击切换为读写' : '点击切换为只读'"
            @click="toggleReadOnly(k)"
          >
            {{ k.ReadOnly ? '只读' : '读写' }}
          </button>
          <span class="last-used">{{ formatDate(k.LastUsedAt) }}</span>
          <button class="chip danger" type="button" @click="remove(k)">删除</button>
        </li>
      </ul>
      <p v-else class="empty">暂无部署密钥</p>
    </template>
  </section>
</template>

<style scoped>
.panel-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 1rem;
}
.panel-head h3 {
  margin: 0;
}
.notice {
  color: var(--muted);
  padding: 1.5rem 0;
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
.mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 0.85rem;
}
.radio-row {
  display: flex;
  gap: 1rem;
  font-size: 0.9rem;
}
.radio-row label {
  display: flex;
  align-items: center;
  gap: 0.3rem;
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
.item-title {
  flex: 1;
  font-weight: 600;
}
.fingerprint {
  color: var(--muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 260px;
}
.badge {
  padding: 0.1rem 0.55rem;
  border-radius: 999px;
  border: 1px solid var(--border);
  font-size: 0.75rem;
  background: transparent;
  cursor: pointer;
}
.badge.ro {
  border-color: var(--success);
  color: var(--success);
  background: var(--success-soft);
}
.badge.rw {
  border-color: var(--accent);
  color: var(--accent);
  background: var(--accent-soft);
}
.last-used {
  color: var(--muted);
  font-size: 0.82rem;
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
.chip.danger {
  border-color: var(--accent);
  color: var(--accent);
}
.chip.danger:hover {
  background: var(--accent-soft);
  color: var(--accent);
}
.error {
  color: var(--accent);
  font-size: 0.85rem;
}
.ok {
  color: var(--success);
  font-size: 0.85rem;
}
.empty {
  color: var(--muted);
}
</style>
