<script setup lang="ts">
import { ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import TopNav from '../components/layout/TopNav.vue'
import { useAuthStore } from '../stores'

const auth = useAuthStore()
const router = useRouter()
const route = useRoute()

const username = ref('')
const password = ref('')
const totpCode = ref('')
const needsTOTP = ref(false)
const error = ref('')
const loading = ref(false)

async function submit() {
  error.value = ''
  loading.value = true
  try {
    await auth.login(username.value, password.value, totpCode.value)
    const redirect = (route.query.redirect as string) || '/dashboard'
    router.push(redirect)
  } catch (e: any) {
    if (e?.response?.status === 401 && e?.response?.data?.error === 'totp_required') {
      needsTOTP.value = true
      error.value = '请输入两步验证码'
    } else {
      error.value = '用户名或密码错误'
    }
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div>
    <TopNav />
    <main class="login">
      <h2>登录</h2>
      <form @submit.prevent="submit">
        <input v-model="username" placeholder="用户名" autocomplete="username" />
        <input v-model="password" type="password" placeholder="密码" autocomplete="current-password" />
        <input v-if="needsTOTP" v-model="totpCode" placeholder="两步验证码" inputmode="numeric" />
        <button type="submit" :disabled="loading">登录</button>
      </form>
      <p v-if="error" class="error">{{ error }}</p>
    </main>
  </div>
</template>

<style scoped>
.login {
  max-width: 320px;
  margin: 4rem auto;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}
form {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}
input,
button {
  padding: 0.55rem 0.75rem;
  border-radius: 6px;
  border: 1px solid var(--border, #d0d7de);
}
.error {
  color: #d1242f;
}
</style>
