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

      <div v-if="loading" class="py-2 text-xs text-[#59636e]">加载中…</div>
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
      <p v-else class="py-2 text-xs text-[#59636e]">暂无部署密钥</p>
    </template>
  </section>
</template>
