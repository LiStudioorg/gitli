<script setup lang="ts">
import { computed, ref } from 'vue'
import api from '../api/client'
import { useAuthStore } from '../stores'
import { errMessage } from '../utils/error'
import { useToast } from '../composables/useToast'
import LogoIcon from '../components/icons/LogoIcon.vue'

const auth = useAuthStore()
const toast = useToast()

const siteName = ref('gitli')
const username = ref('')
const email = ref('')
const password = ref('')
const confirmPassword = ref('')
const error = ref('')
const loading = ref(false)

const emailError = computed(() => {
  const v = email.value.trim()
  if (!v) return ''
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(v) ? '' : '请输入有效的邮箱地址'
})

const passwordError = computed(() => {
  if (!password.value) return ''
  return password.value.length >= 8 ? '' : '密码至少 8 位'
})

const confirmError = computed(() => {
  if (!confirmPassword.value) return ''
  return confirmPassword.value === password.value ? '' : '两次输入的密码不一致'
})

const canSubmit = computed(
  () =>
    siteName.value.trim() !== '' &&
    username.value.trim() !== '' &&
    email.value.trim() !== '' &&
    password.value.length >= 8 &&
    confirmPassword.value === password.value &&
    !emailError.value,
)

async function submit() {
  error.value = ''
  if (!canSubmit.value) {
    error.value = emailError.value || passwordError.value || confirmError.value || '请完整填写表单'
    return
  }
  loading.value = true
  try {
    await api.post('/setup', {
      site_name: siteName.value.trim(),
      username: username.value.trim(),
      email: email.value.trim(),
      password: password.value,
    })
    // 创建成功后直接登录，用户无需再手动输入一次刚设置的密码。
    await auth.login(username.value.trim(), password.value)
    toast.success('初始化完成，欢迎使用 gitli')
    window.location.assign('/dashboard')
  } catch (e) {
    error.value = errMessage(e, '初始化失败，请重试')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <main class="flex min-h-screen items-center justify-center bg-[radial-gradient(1200px_400px_at_50%_-10%,rgba(234,88,12,0.12),transparent_70%)] px-4 py-12">
    <section class="w-full max-w-[460px] rounded-md border border-[#d1d9e0] bg-white p-6 shadow-lg dark:border-[#30363d] dark:bg-[#161b22] md:p-8">
      <div class="mb-4 flex items-center justify-center [&_svg]:h-11 [&_svg]:w-11"><LogoIcon /></div>
      <h1>初始化 gitli</h1>
      <p class="mb-5 mt-1 text-sm text-[#59636e]">创建第一个管理员账号并设置站点名称。完成后本页面会自动锁定。</p>

      <form class="flex flex-col gap-3" @submit.prevent="submit" novalidate>
        <div class="flex flex-col gap-1.5">
          <label for="setup-site">站点名称</label>
          <input id="setup-site" v-model="siteName" class="h-8 rounded-md border border-[#d1d9e0] bg-[#f6f8fa] px-2.5 text-sm text-[#1f2328] outline-none placeholder:text-[#59636e] focus:border-[#0969da] focus:bg-white focus:ring-2 focus:ring-[#0969da]/25 dark:border-[#3d444d] dark:bg-[#0d1117] dark:text-[#e6edf3]" type="text" placeholder="gitli" required />
        </div>

        <div class="flex flex-col gap-1.5">
          <label for="setup-username">管理员用户名</label>
          <input
            id="setup-username"
            v-model="username"
            class="h-8 rounded-md border border-[#d1d9e0] bg-[#f6f8fa] px-2.5 text-sm text-[#1f2328] outline-none placeholder:text-[#59636e] focus:border-[#0969da] focus:bg-white focus:ring-2 focus:ring-[#0969da]/25 dark:border-[#3d444d] dark:bg-[#0d1117] dark:text-[#e6edf3]"
            type="text"
            autocomplete="username"
            placeholder="admin"
            required
          />
        </div>

        <div class="flex flex-col gap-1.5">
          <label for="setup-email">邮箱</label>
          <input
            id="setup-email"
            v-model="email"
            class="h-8 rounded-md border border-[#d1d9e0] bg-[#f6f8fa] px-2.5 text-sm text-[#1f2328] outline-none placeholder:text-[#59636e] focus:border-[#0969da] focus:bg-white focus:ring-2 focus:ring-[#0969da]/25 dark:border-[#3d444d] dark:bg-[#0d1117] dark:text-[#e6edf3]"
            :class="{ '!border-[#cf222e]': emailError }"
            type="email"
            autocomplete="email"
            placeholder="admin@example.com"
            required
          />
          <span v-if="emailError" class="text-xs text-[#cf222e]">{{ emailError }}</span>
        </div>

        <div class="flex flex-col gap-1.5">
          <label for="setup-password">密码</label>
          <input
            id="setup-password"
            v-model="password"
            class="h-8 rounded-md border border-[#d1d9e0] bg-[#f6f8fa] px-2.5 text-sm text-[#1f2328] outline-none placeholder:text-[#59636e] focus:border-[#0969da] focus:bg-white focus:ring-2 focus:ring-[#0969da]/25 dark:border-[#3d444d] dark:bg-[#0d1117] dark:text-[#e6edf3]"
            :class="{ '!border-[#cf222e]': passwordError }"
            type="password"
            autocomplete="new-password"
            placeholder="至少 8 位"
            required
          />
          <span v-if="passwordError" class="text-xs text-[#cf222e]">{{ passwordError }}</span>
        </div>

        <div class="flex flex-col gap-1.5">
          <label for="setup-confirm">确认密码</label>
          <input
            id="setup-confirm"
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
          {{ loading ? '创建中…' : '创建管理员并进入' }}
        </button>
      </form>

      <p class="mt-4 text-center text-xs text-[#59636e]">该账号拥有最高管理权限，请妥善保管密码。</p>
    </section>
  </main>
</template>

