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
    // H-07：只允许跳转到 http(s) 地址，杜绝 javascript: 伪协议 XSS。
    if (isSafeRedirect(data.redirect)) {
      window.location.href = data.redirect
    } else {
      error.value = '服务器返回了不安全的回调地址，已阻止跳转'
      busy.value = false
    }
  } catch (e) {
    if (axios.isAxiosError(e) && e.response?.status === 401) {
      redirectToLogin()
      return
    }
    error.value = errMessage(e)
    busy.value = false
  }
}

function isSafeRedirect(raw: string | undefined | null): raw is string {
  if (!raw) return false
  try {
    const u = new URL(raw, window.location.origin)
    return u.protocol === 'http:' || u.protocol === 'https:'
  } catch {
    return false
  }
}

function deny() {
  const target = new URL(info.value?.redirect_uri || redirectUri, window.location.origin)
  if (target.protocol !== 'http:' && target.protocol !== 'https:') {
    error.value = '回调地址协议不安全，已阻止跳转'
    return
  }
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
    <main class="flex min-h-[calc(100vh-3rem)] items-center justify-center bg-[#f6f8fa] px-4 py-10 dark:bg-[#0d1117]">
      <div class="w-full max-w-[440px] rounded-md border border-[#d1d9e0] bg-white p-6 dark:border-[#3d444d] dark:bg-[#0d1117]">
        <h2 class="mb-3 text-lg font-semibold text-[#1f2328] dark:text-[#e6edf3]">授权应用</h2>
        <p v-if="error" class="mb-3 rounded-md border border-[#ff818266] bg-[#ffebe9] px-3 py-2 text-sm text-[#cf222e]">{{ error }}</p>
        <p v-if="loading" class="text-sm text-[#59636e]">加载中…</p>
        <template v-else-if="info">
          <p class="text-sm text-[#1f2328] dark:text-[#e6edf3]">
            应用 <strong>{{ info.app?.name || info.app?.client_id || clientId }}</strong> 请求访问你的账户。
          </p>
          <p class="text-sm text-[#59636e]">客户端 ID：<code>{{ info.app?.client_id || clientId }}</code></p>
          <p class="text-sm text-[#59636e]">回调地址：<code class="break">{{ info.redirect_uri }}</code></p>
          <div class="my-3 rounded-md border border-[#d1d9e0] p-3 dark:border-[#3d444d]">
            <span class="text-sm text-[#59636e]">请求的权限：</span>
            <ul>
              <li v-for="s in info.scopes" :key="s"><code>{{ s }}</code></li>
            </ul>
          </div>
          <div class="mt-4 flex justify-end gap-2">
            <button class="h-8 rounded-md border border-[#d1d9e0] px-3 text-sm text-[#1f2328] hover:bg-[#f6f8fa] dark:border-[#3d444d] dark:text-[#e6edf3] dark:hover:bg-[#161b22]" type="button" :disabled="busy" @click="deny">取消</button>
            <button class="h-8 rounded-md bg-[#2da44e] px-3 text-sm font-semibold text-white hover:bg-[#2c974b] disabled:opacity-50" type="button" :disabled="busy" @click="approve">
              {{ busy ? '处理中…' : '授权' }}
            </button>
          </div>
        </template>
      </div>
    </main>
  </div>
</template>
