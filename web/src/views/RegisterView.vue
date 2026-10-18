<script setup lang="ts">
import { computed, ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import TopNav from '../components/layout/TopNav.vue'
import { useAuthStore } from '../stores'
import { errMessage } from '../utils/error'
import api from '../api/client'
import { useToast } from '../composables/useToast'

const auth = useAuthStore()
const router = useRouter()
const toast = useToast()

const username = ref('')
const email = ref('')
const password = ref('')
const confirmPassword = ref('')
const error = ref('')
const loading = ref(false)

const usernameError = computed(() => {
  const v = username.value.trim()
  if (!v) return ''
  if (!/^[A-Za-z0-9._-]{2,39}$/.test(v)) return '用户名为 2–39 位字母、数字、点、下划线或连字符'
  return ''
})

const emailError = computed(() => {
  const v = email.value.trim()
  if (!v) return ''
  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(v)) return '请输入有效的邮箱地址'
  return ''
})

const passwordError = computed(() => {
  if (!password.value) return ''
  if (password.value.length < 12) return '密码至少 12 位'
  return ''
})

const confirmError = computed(() => {
  if (!confirmPassword.value) return ''
  if (confirmPassword.value !== password.value) return '两次输入的密码不一致'
  return ''
})

const canSubmit = computed(
  () =>
    username.value.trim() !== '' &&
    email.value.trim() !== '' &&
    password.value.length >= 12 &&
    confirmPassword.value === password.value &&
    !usernameError.value &&
    !emailError.value,
)

async function submit() {
  error.value = ''
  if (!canSubmit.value) {
    error.value = usernameError.value || emailError.value || passwordError.value || confirmError.value || '请完整填写表单'
    return
  }
  loading.value = true
  try {
    await api.post('/auth/register', {
      username: username.value.trim(),
      email: email.value.trim(),
      password: password.value,
    })
    await auth.login(username.value.trim(), password.value)
    toast.success('注册成功，已自动登录')
    await router.push('/dashboard')
  } catch (e) {
    error.value = errMessage(e, '注册失败，请稍后重试')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div>
    <TopNav />
    <main class="flex min-h-[calc(100vh-3.5rem)] items-center justify-center bg-[radial-gradient(1200px_400px_at_50%_-10%,rgba(234,88,12,0.10),transparent_70%)] px-4 py-10">
      <section class="w-full max-w-[440px] rounded-md border border-[#d1d9e0] bg-white p-6 shadow-lg dark:border-[#30363d] dark:bg-[#161b22] md:p-8">
        <h1>创建账号</h1>
        <p class="mb-5 mt-1 text-sm text-[#59636e]">加入 gitli，开始托管你的代码</p>

        <form class="flex flex-col gap-3" @submit.prevent="submit" novalidate>
          <div class="flex flex-col gap-1.5">
            <label for="reg-username">用户名</label>
            <input
              id="reg-username"
              v-model="username"
              class="h-8 rounded-md border border-[#d1d9e0] bg-[#f6f8fa] px-2.5 text-sm text-[#1f2328] outline-none placeholder:text-[#59636e] focus:border-[#0969da] focus:bg-white focus:ring-2 focus:ring-[#0969da]/25 dark:border-[#3d444d] dark:bg-[#0d1117] dark:text-[#e6edf3]"
              :class="{ '!border-[#cf222e]': usernameError }"
              type="text"
              autocomplete="username"
              placeholder="例如 alice"
              required
            />
            <span v-if="usernameError" class="text-xs text-[#cf222e]">{{ usernameError }}</span>
          </div>

          <div class="flex flex-col gap-1.5">
            <label for="reg-email">邮箱</label>
            <input
              id="reg-email"
              v-model="email"
              class="h-8 rounded-md border border-[#d1d9e0] bg-[#f6f8fa] px-2.5 text-sm text-[#1f2328] outline-none placeholder:text-[#59636e] focus:border-[#0969da] focus:bg-white focus:ring-2 focus:ring-[#0969da]/25 dark:border-[#3d444d] dark:bg-[#0d1117] dark:text-[#e6edf3]"
              :class="{ '!border-[#cf222e]': emailError }"
              type="email"
              autocomplete="email"
              placeholder="you@example.com"
              required
            />
            <span v-if="emailError" class="text-xs text-[#cf222e]">{{ emailError }}</span>
          </div>

          <div class="flex flex-col gap-1.5">
            <label for="reg-password">密码</label>
            <input
              id="reg-password"
              v-model="password"
              class="h-8 rounded-md border border-[#d1d9e0] bg-[#f6f8fa] px-2.5 text-sm text-[#1f2328] outline-none placeholder:text-[#59636e] focus:border-[#0969da] focus:bg-white focus:ring-2 focus:ring-[#0969da]/25 dark:border-[#3d444d] dark:bg-[#0d1117] dark:text-[#e6edf3]"
              :class="{ '!border-[#cf222e]': passwordError }"
              type="password"
              autocomplete="new-password"
              placeholder="至少 12 位"
              required
            />
            <span v-if="passwordError" class="text-xs text-[#cf222e]">{{ passwordError }}</span>
            <span v-else class="text-xs text-[#59636e]">建议混合字母、数字与符号</span>
          </div>

          <div class="flex flex-col gap-1.5">
            <label for="reg-confirm">确认密码</label>
            <input
              id="reg-confirm"
              v-model="confirmPassword"
              class="h-8 rounded-md border border-[#d1d9e0] bg-[#f6f8fa] px-2.5 text-sm text-[#1f2328] outline-none placeholder:text-[#59636e] focus:border-[#0969da] focus:bg-white focus:ring-2 focus:ring-[#0969da]/25 dark:border-[#3d444d] dark:bg-[#0d1117] dark:text-[#e6edf3]"
              :class="{ '!border-[#cf222e]': confirmError }"
              type="password"
              autocomplete="new-password"
              placeholder="再次输入密码"
              required
            />
            <span v-if="confirmError" class="text-xs text-[#cf222e]">{{ confirmError }}</span>
          </div>

          <p v-if="error" class="rounded-md border border-[#ff818266] bg-[#ffebe9] px-3 py-2 text-sm text-[#cf222e]" role="alert">{{ error }}</p>

          <button class="inline-flex h-10 w-full items-center justify-center gap-2 rounded-md bg-[#2da44e] px-4 text-sm font-semibold text-white hover:bg-[#2c974b] disabled:opacity-50" type="submit" :disabled="loading">
            <span v-if="loading" class="h-3.5 w-3.5 animate-spin rounded-full border-2 border-white/40 border-t-white" aria-hidden="true" />
            {{ loading ? '注册中…' : '创建账号' }}
          </button>
        </form>

        <p class="mt-6 text-center text-sm text-[#59636e]">已有账号？<RouterLink to="/login" class="font-semibold text-[#ea580c] hover:underline">登录</RouterLink></p>
      </section>
    </main>
  </div>
</template>

