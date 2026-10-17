<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import axios from 'axios'
import TopNav from '../components/layout/TopNav.vue'
import api from '../api/client'
import { errMessage } from '../utils/error'

interface AuthorizeInfo {
  app?: { name: string; client_id: string }
  redirect_uri: string
  scopes: string[]
  state?: string
}

const route = useRoute()

const info = ref<AuthorizeInfo | null>(null)
const loading = ref(true)
const busy = ref(false)
const error = ref('')

function queryString(key: string): string {
  const v = route.query[key]
  if (Array.isArray(v)) return v[0] ?? ''
  return typeof v === 'string' ? v : ''
}

const clientId = queryString('client_id')
const redirectUri = queryString('redirect_uri')
const scope = queryString('scope')
const state = queryString('state')

function redirectToLogin() {
  const current = route.fullPath
  window.location.href = `/login?redirect=${encodeURIComponent(current)}`
}

async function load() {
  loading.value = true
  error.value = ''
  if (!clientId || !redirectUri) {
    error.value = '缺少 client_id 或 redirect_uri 参数'
    loading.value = false
    return
  }
  try {
    const { data } = await api.get<AuthorizeInfo>('/oauth/authorize-info', {
      params: { client_id: clientId, redirect_uri: redirectUri, scope, state },
    })
    info.value = data
  } catch (e) {
    if (axios.isAxiosError(e) && e.response?.status === 401) {
      redirectToLogin()
      return
    }
    error.value = errMessage(e)
  } finally {
    loading.value = false
  }
}

async function approve() {
  if (!info.value) return
  busy.value = true
  error.value = ''
  try {
    const { data } = await api.post<{ redirect: string }>('/oauth/approve', {
      client_id: clientId,
      redirect_uri: info.value.redirect_uri || redirectUri,
      scopes: info.value.scopes ?? [],
      state: info.value.state ?? state,
    })
    window.location.href = data.redirect
  } catch (e) {
    if (axios.isAxiosError(e) && e.response?.status === 401) {
      redirectToLogin()
      return
    }
    error.value = errMessage(e)
    busy.value = false
  }
}

function deny() {
  const target = new URL(info.value?.redirect_uri || redirectUri, window.location.origin)
  target.searchParams.set('error', 'access_denied')
  const st = info.value?.state ?? state
  if (st) target.searchParams.set('state', st)
  window.location.href = target.toString()
}

onMounted(load)
</script>

<template>
  <div>
    <TopNav />
    <main class="authorize">
      <div class="card">
        <h2>授权应用</h2>
        <p v-if="error" class="error">{{ error }}</p>
        <p v-if="loading" class="muted">加载中…</p>
        <template v-else-if="info">
          <p class="lead">
            应用 <strong>{{ info.app?.name || info.app?.client_id || clientId }}</strong> 请求访问你的账户。
          </p>
          <p class="muted">客户端 ID：<code>{{ info.app?.client_id || clientId }}</code></p>
          <p class="muted">回调地址：<code class="break">{{ info.redirect_uri }}</code></p>
          <div class="scopes">
            <span class="muted">请求的权限：</span>
            <ul>
              <li v-for="s in info.scopes" :key="s"><code>{{ s }}</code></li>
            </ul>
          </div>
          <div class="actions">
            <button class="ghost" type="button" :disabled="busy" @click="deny">取消</button>
            <button class="primary" type="button" :disabled="busy" @click="approve">
              {{ busy ? '处理中…' : '授权' }}
            </button>
          </div>
        </template>
      </div>
    </main>
  </div>
</template>

<style scoped>
.authorize {
  max-width: 460px;
  margin: 3rem auto;
  padding: 0 1rem;
}
.card {
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 1.5rem;
  background: var(--card);
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}
.card h2 {
  margin: 0;
}
.lead {
  margin: 0;
  font-size: 0.95rem;
}
.scopes {
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 0.6rem 0.8rem;
  font-size: 0.9rem;
}
.scopes ul {
  margin: 0.35rem 0 0;
  padding-left: 1.1rem;
}
.actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
  margin-top: 0.25rem;
}
button.primary {
  padding: 0.45rem 1rem;
  border-radius: 6px;
  border: 1px solid var(--primary);
  background: var(--primary);
  color: #fff;
  font-size: 0.9rem;
}
button.primary:hover:not(:disabled) {
  background: var(--primary-hover);
}
.ghost {
  padding: 0.45rem 1rem;
  border-radius: 6px;
  border: 1px solid var(--border);
  background: transparent;
  color: var(--color-fg);
  font-size: 0.9rem;
}
button:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
.error {
  margin: 0;
  color: #d1242f;
  background: rgba(209, 36, 47, 0.08);
  border: 1px solid rgba(209, 36, 47, 0.4);
  border-radius: 8px;
  padding: 0.6rem 0.8rem;
  font-size: 0.9rem;
}
.muted {
  margin: 0;
  color: var(--muted);
  font-size: 0.9rem;
}
.break {
  word-break: break-all;
}
</style>
