<script setup lang="ts">
import { computed, ref } from 'vue'
import api from '../api/client'
import { useAuthStore } from '../stores'
import { errMessage } from '../utils/error'
import { useToast } from '../composables/useToast'

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
  <main class="auth-page setup-page">
    <section class="auth-card setup-card">
      <div class="brand-badge" aria-hidden="true">&lt;/&gt;</div>
      <h1>初始化 gitli</h1>
      <p class="auth-subtitle">创建第一个管理员账号并设置站点名称。完成后本页面会自动锁定。</p>

      <form class="ui-form" @submit.prevent="submit" novalidate>
        <div class="ui-field">
          <label for="setup-site">站点名称</label>
          <input id="setup-site" v-model="siteName" class="ui-input" type="text" placeholder="gitli" required />
        </div>

        <div class="ui-field">
          <label for="setup-username">管理员用户名</label>
          <input
            id="setup-username"
            v-model="username"
            class="ui-input"
            type="text"
            autocomplete="username"
            placeholder="admin"
            required
          />
        </div>

        <div class="ui-field">
          <label for="setup-email">邮箱</label>
          <input
            id="setup-email"
            v-model="email"
            class="ui-input"
            :class="{ invalid: emailError }"
            type="email"
            autocomplete="email"
            placeholder="admin@example.com"
            required
          />
          <span v-if="emailError" class="ui-error">{{ emailError }}</span>
        </div>

        <div class="ui-field">
          <label for="setup-password">密码</label>
          <input
            id="setup-password"
            v-model="password"
            class="ui-input"
            :class="{ invalid: passwordError }"
            type="password"
            autocomplete="new-password"
            placeholder="至少 8 位"
            required
          />
          <span v-if="passwordError" class="ui-error">{{ passwordError }}</span>
        </div>

        <div class="ui-field">
          <label for="setup-confirm">确认密码</label>
          <input
            id="setup-confirm"
            v-model="confirmPassword"
            class="ui-input"
            :class="{ invalid: confirmError }"
            type="password"
            autocomplete="new-password"
            placeholder="再次输入密码"
            required
          />
          <span v-if="confirmError" class="ui-error">{{ confirmError }}</span>
        </div>

        <p v-if="error" class="ui-alert ui-alert-error" role="alert">{{ error }}</p>

        <button class="ui-btn ui-btn-primary ui-btn-block ui-btn-lg" type="submit" :disabled="loading">
          <span v-if="loading" class="ui-spinner" aria-hidden="true" />
          {{ loading ? '创建中…' : '创建管理员并进入' }}
        </button>
      </form>

      <p class="hint">该账号拥有最高管理权限，请妥善保管密码。</p>
    </section>
  </main>
</template>

<style scoped>
.setup-page {
  padding-top: 3rem;
}
.setup-card {
  width: min(460px, 100%);
}
.brand-badge {
  width: 46px;
  height: 46px;
  border-radius: 12px;
  background: var(--primary);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-weight: 700;
  font-size: 1.15rem;
  margin-bottom: 1rem;
  box-shadow: var(--shadow-md);
}
.invalid {
  border-color: var(--danger) !important;
}
.hint {
  margin: 1.1rem 0 0;
  font-size: 0.78rem;
  color: var(--muted);
  text-align: center;
}
</style>
