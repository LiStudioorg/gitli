<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import api from '../api/client'

const router = useRouter()
const siteName = ref('gitli')
const username = ref('')
const email = ref('')
const password = ref('')
const error = ref('')
const loading = ref(false)

const disabledReason = computed(() => {
  if (!username.value) return '请填写管理员用户名'
  if (!email.value) return '请填写邮箱'
  if (password.value.length < 8) return '密码至少 8 位'
  return ''
})

async function submit() {
  error.value = ''
  if (!username.value || !email.value) {
    error.value = '请填写用户名和邮箱'
    return
  }
  if (password.value.length < 8) {
    error.value = '密码至少 8 位'
    return
  }
  loading.value = true
  try {
    await api.post('/setup', {
      site_name: siteName.value,
      username: username.value,
      email: email.value,
      password: password.value,
    })
    router.push({ path: '/login', query: { username: username.value } })
  } catch (e: any) {
    error.value = e?.response?.data?.error ?? '初始化失败，请重试'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <main class="setup">
    <h1>初始化 gitli</h1>
    <p class="subtitle">创建第一个管理员账号。完成后本页面将自动锁定。</p>
    <form @submit.prevent="submit">
      <label>
        站点名称
        <input v-model="siteName" placeholder="gitli" />
      </label>
      <label>
        管理员用户名
        <input v-model="username" placeholder="admin" autocomplete="username" />
      </label>
      <label>
        邮箱
        <input v-model="email" type="email" placeholder="admin@example.com" autocomplete="email" />
      </label>
      <label>
        密码
        <input v-model="password" type="password" placeholder="密码（至少 8 位）" autocomplete="new-password" />
      </label>
      <button type="submit" :disabled="loading">{{ loading ? '创建中…' : '创建管理员' }}</button>
    </form>
    <p v-if="error" class="error">{{ error }}</p>
    <p v-else-if="disabledReason" class="hint">{{ disabledReason }}</p>
  </main>
</template>

<style scoped>
.setup {
  max-width: 400px;
  margin: 4rem auto;
  display: flex;
  flex-direction: column;
  gap: 0.85rem;
  padding: 0 1rem;
}
.subtitle {
  color: var(--muted, #57606a);
  font-size: 0.9rem;
}
form {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}
label {
  display: flex;
  flex-direction: column;
  gap: 0.3rem;
  font-size: 0.9rem;
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
.hint {
  color: var(--muted, #57606a);
  font-size: 0.85rem;
}
</style>
