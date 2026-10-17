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
  if (password.value.length < 8) return '密码至少 8 位'
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
    password.value.length >= 8 &&
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
    <main class="auth-page">
      <section class="auth-card">
        <h1>创建账号</h1>
        <p class="auth-subtitle">加入 gitli，开始托管你的代码</p>

        <form class="ui-form" @submit.prevent="submit" novalidate>
          <div class="ui-field">
            <label for="reg-username">用户名</label>
            <input
              id="reg-username"
              v-model="username"
              class="ui-input"
              :class="{ invalid: usernameError }"
              type="text"
              autocomplete="username"
              placeholder="例如 alice"
              required
            />
            <span v-if="usernameError" class="ui-error">{{ usernameError }}</span>
          </div>

          <div class="ui-field">
            <label for="reg-email">邮箱</label>
            <input
              id="reg-email"
              v-model="email"
              class="ui-input"
              :class="{ invalid: emailError }"
              type="email"
              autocomplete="email"
              placeholder="you@example.com"
              required
            />
            <span v-if="emailError" class="ui-error">{{ emailError }}</span>
          </div>

          <div class="ui-field">
            <label for="reg-password">密码</label>
            <input
              id="reg-password"
              v-model="password"
              class="ui-input"
              :class="{ invalid: passwordError }"
              type="password"
              autocomplete="new-password"
              placeholder="至少 8 位"
              required
            />
            <span v-if="passwordError" class="ui-error">{{ passwordError }}</span>
            <span v-else class="ui-help">建议混合字母、数字与符号</span>
          </div>

          <div class="ui-field">
            <label for="reg-confirm">确认密码</label>
            <input
              id="reg-confirm"
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
            {{ loading ? '注册中…' : '创建账号' }}
          </button>
        </form>

        <p class="switch">已有账号？<RouterLink to="/login">登录</RouterLink></p>
      </section>
    </main>
  </div>
</template>

<style scoped>
.invalid {
  border-color: var(--danger) !important;
}
.switch {
  margin: 1.4rem 0 0;
  text-align: center;
  font-size: 0.85rem;
  color: var(--muted);
}
.switch a {
  color: var(--primary);
  font-weight: 600;
}
</style>
