<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import TopNav from '../components/layout/TopNav.vue'
import { useAuthStore } from '../stores'

const auth = useAuthStore()
const router = useRouter()
const username = ref('')
const email = ref('')
const password = ref('')
const error = ref('')
const loading = ref(false)

const disabledReason = computed(() => {
  if (!username.value) return '请填写用户名'
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
    const { default: api } = await import('../api/client')
    await api.post('/auth/register', {
      username: username.value,
      email: email.value,
      password: password.value,
    })
    await auth.login(username.value, password.value)
    router.push('/dashboard')
  } catch (e: any) {
    error.value = e?.response?.data?.error ?? '注册失败'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div>
    <TopNav />
    <main class="register">
      <h2>注册</h2>
      <form @submit.prevent="submit">
        <input v-model="username" placeholder="用户名" autocomplete="username" />
        <input v-model="email" type="email" placeholder="邮箱" autocomplete="email" />
        <input v-model="password" type="password" placeholder="密码（至少 8 位）" autocomplete="new-password" />
        <button type="submit" :disabled="loading">{{ loading ? '注册中…' : '注册' }}</button>
      </form>
      <p v-if="disabledReason && !error" class="hint">{{ disabledReason }}</p>
      <p v-if="error" class="error">{{ error }}</p>
    </main>
  </div>
</template>

<style scoped>
.register {
  max-width: 340px;
  margin: 3rem auto;
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
.hint {
  color: var(--muted, #57606a);
  font-size: 0.85rem;
}
</style>
