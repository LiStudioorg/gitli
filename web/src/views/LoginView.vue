<script setup lang="ts">
import { ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { RouterLink } from 'vue-router'
import TopNav from '../components/layout/TopNav.vue'
import { useAuthStore } from '../stores'
import { apiCode, apiStatus, errMessage } from '../utils/error'
import { useToast } from '../composables/useToast'

const auth = useAuthStore()
const router = useRouter()
const route = useRoute()
const toast = useToast()

const username = ref(typeof route.query.username === 'string' ? route.query.username : '')
const password = ref('')
const totpCode = ref('')
const needsTOTP = ref(false)
const error = ref('')
const loading = ref(false)

async function submit() {
  error.value = ''
  if (!username.value.trim() || !password.value) {
    error.value = '请输入用户名和密码'
    return
  }
  loading.value = true
  try {
    await auth.login(username.value.trim(), password.value, totpCode.value)
    toast.success('登录成功')
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/dashboard'
    await router.push(redirect)
  } catch (e) {
    if (apiStatus(e) === 401 && apiCode(e) === 'totp_required') {
      needsTOTP.value = true
      error.value = '请输入两步验证码'
    } else {
      error.value = errMessage(e, '用户名或密码错误')
    }
  } finally {
    loading.value = false
  }
}

function loginWithSAML() {
  window.location.href = '/api/v1/auth/saml/login'
}

function loginWithOIDC() {
  window.location.href = '/api/v1/auth/oidc/login'
}
</script>

<template>
  <div>
    <TopNav />
    <main class="flex min-h-[calc(100vh-3.5rem)] items-center justify-center bg-[radial-gradient(1200px_400px_at_50%_-10%,rgba(234,88,12,0.10),transparent_70%)] px-4 py-10">
      <section class="w-full max-w-[420px] rounded-md border border-[#d1d9e0] bg-white p-6 shadow-lg dark:border-[#30363d] dark:bg-[#161b22] md:p-8">
        <h1>登录 gitli</h1>
        <p class="mb-5 mt-1 text-sm text-[#59636e]">使用你的账号继续</p>

        <form class="flex flex-col gap-3" @submit.prevent="submit">
          <div class="flex flex-col gap-1.5">
            <label for="login-username">用户名</label>
            <input
              id="login-username"
              v-model="username"
              class="h-8 rounded-md border border-[#d1d9e0] bg-[#f6f8fa] px-2.5 text-sm text-[#1f2328] outline-none placeholder:text-[#59636e] focus:border-[#0969da] focus:bg-white focus:ring-2 focus:ring-[#0969da]/25 dark:border-[#3d444d] dark:bg-[#0d1117] dark:text-[#e6edf3]"
              type="text"
              autocomplete="username"
              placeholder="请输入用户名"
              required
            />
          </div>

          <div class="flex flex-col gap-1.5">
            <label for="login-password">密码</label>
            <input
              id="login-password"
              v-model="password"
              class="h-8 rounded-md border border-[#d1d9e0] bg-[#f6f8fa] px-2.5 text-sm text-[#1f2328] outline-none placeholder:text-[#59636e] focus:border-[#0969da] focus:bg-white focus:ring-2 focus:ring-[#0969da]/25 dark:border-[#3d444d] dark:bg-[#0d1117] dark:text-[#e6edf3]"
              type="password"
              autocomplete="current-password"
              placeholder="请输入密码"
              required
            />
          </div>

          <div v-if="needsTOTP" class="flex flex-col gap-1.5">
            <label for="login-totp">两步验证码</label>
            <input
              id="login-totp"
              v-model="totpCode"
              class="h-8 rounded-md border border-[#d1d9e0] bg-[#f6f8fa] px-2.5 text-sm text-[#1f2328] outline-none placeholder:text-[#59636e] focus:border-[#0969da] focus:bg-white focus:ring-2 focus:ring-[#0969da]/25 dark:border-[#3d444d] dark:bg-[#0d1117] dark:text-[#e6edf3]"
              inputmode="numeric"
              autocomplete="one-time-code"
              placeholder="6 位验证码"
            />
          </div>

          <p v-if="error" class="rounded-md border border-[#ff818266] bg-[#ffebe9] px-3 py-2 text-sm text-[#cf222e]" role="alert">{{ error }}</p>

          <button class="inline-flex h-10 w-full items-center justify-center gap-2 rounded-md bg-[#2da44e] px-4 text-sm font-semibold text-white hover:bg-[#2c974b] disabled:opacity-50" type="submit" :disabled="loading">
            <span v-if="loading" class="h-3.5 w-3.5 animate-spin rounded-full border-2 border-white/40 border-t-white" aria-hidden="true" />
            {{ loading ? '登录中…' : '登录' }}
          </button>
        </form>

        <div class="my-4 flex items-center gap-3 text-xs text-[#59636e]"><span>或</span></div>

        <div class="flex flex-col gap-2">
          <button class="inline-flex h-9 w-full items-center justify-center rounded-md border border-[#d1d9e0] px-3 text-sm text-[#1f2328] hover:bg-[#f6f8fa] dark:border-[#3d444d] dark:text-[#e6edf3] dark:hover:bg-[#161b22]" type="button" @click="loginWithSAML">通过 SAML 登录</button>
          <button class="inline-flex h-9 w-full items-center justify-center rounded-md border border-[#d1d9e0] px-3 text-sm text-[#1f2328] hover:bg-[#f6f8fa] dark:border-[#3d444d] dark:text-[#e6edf3] dark:hover:bg-[#161b22]" type="button" @click="loginWithOIDC">通过 OIDC 登录</button>
        </div>

        <p class="mt-6 text-center text-sm text-[#59636e]">
          还没有账号？<RouterLink to="/register" class="font-semibold text-[#ea580c] hover:underline">注册</RouterLink>
        </p>
      </section>
    </main>
  </div>
</template>

