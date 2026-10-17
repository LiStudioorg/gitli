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
    <main class="auth-page">
      <section class="auth-card">
        <h1>登录 gitli</h1>
        <p class="auth-subtitle">使用你的账号继续</p>

        <form class="ui-form" @submit.prevent="submit">
          <div class="ui-field">
            <label for="login-username">用户名</label>
            <input
              id="login-username"
              v-model="username"
              class="ui-input"
              type="text"
              autocomplete="username"
              placeholder="请输入用户名"
              required
            />
          </div>

          <div class="ui-field">
            <label for="login-password">密码</label>
            <input
              id="login-password"
              v-model="password"
              class="ui-input"
              type="password"
              autocomplete="current-password"
              placeholder="请输入密码"
              required
            />
          </div>

          <div v-if="needsTOTP" class="ui-field">
            <label for="login-totp">两步验证码</label>
            <input
              id="login-totp"
              v-model="totpCode"
              class="ui-input"
              inputmode="numeric"
              autocomplete="one-time-code"
              placeholder="6 位验证码"
            />
          </div>

          <p v-if="error" class="ui-alert ui-alert-error" role="alert">{{ error }}</p>

          <button class="ui-btn ui-btn-primary ui-btn-block ui-btn-lg" type="submit" :disabled="loading">
            <span v-if="loading" class="ui-spinner" aria-hidden="true" />
            {{ loading ? '登录中…' : '登录' }}
          </button>
        </form>

        <div class="ui-divider"><span>或</span></div>

        <div class="alt-actions">
          <button class="ui-btn ui-btn-block" type="button" @click="loginWithSAML">通过 SAML 登录</button>
          <button class="ui-btn ui-btn-block" type="button" @click="loginWithOIDC">通过 OIDC 登录</button>
        </div>

        <p class="switch">
          还没有账号？<RouterLink to="/register">注册</RouterLink>
        </p>
      </section>
    </main>
  </div>
</template>

<style scoped>
.alt-actions {
  display: flex;
  flex-direction: column;
  gap: 0.55rem;
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
