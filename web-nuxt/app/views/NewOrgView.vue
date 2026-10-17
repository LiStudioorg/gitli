<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import TopNav from '../components/layout/TopNav.vue'
import Sidebar from '../components/layout/Sidebar.vue'
import api from '../api/client'

const router = useRouter()
const name = ref('')
const displayName = ref('')
const error = ref('')
const loading = ref(false)

const disabledReason = computed(() => {
  if (!name.value.trim()) return '请填写组织标识'
  if (!/^[A-Za-z0-9._-]+$/.test(name.value.trim())) return '组织标识只能包含字母、数字、点、下划线和连字符'
  return ''
})

async function submit() {
  error.value = ''
  if (disabledReason.value) {
    error.value = disabledReason.value
    return
  }
  loading.value = true
  try {
    await api.post('/orgs', {
      name: name.value.trim(),
      display_name: displayName.value || name.value.trim(),
    })
    await router.push('/dashboard')
  } catch (e: any) {
    error.value = e?.response?.data?.error ?? '创建组织失败'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div>
    <TopNav variant="app" />
    <div class="body">
      <Sidebar />
      <main class="content">
        <h2>新建组织</h2>
        <form class="form" @submit.prevent="submit">
          <label>
            组织标识
            <input v-model="name" placeholder="my-team" autocomplete="off" />
          </label>
          <label>
            显示名称（可选）
            <input v-model="displayName" placeholder="我的团队" />
          </label>
          <div class="actions">
            <button type="submit" :disabled="loading">{{ loading ? '创建中…' : '创建组织' }}</button>
            <button type="button" class="secondary" @click="router.back()">取消</button>
          </div>
        </form>
        <p v-if="error" class="error">{{ error }}</p>
        <p v-else-if="disabledReason" class="hint">{{ disabledReason }}</p>
      </main>
    </div>
  </div>
</template>

<style scoped>
.body {
  display: flex;
}
.content {
  flex: 1;
  padding: 1.5rem;
  max-width: 640px;
}
.form {
  display: flex;
  flex-direction: column;
  gap: 0.9rem;
}
label {
  display: flex;
  flex-direction: column;
  gap: 0.3rem;
  font-size: 0.9rem;
}
input {
  padding: 0.55rem 0.75rem;
  border-radius: 6px;
  border: 1px solid var(--border, #d0d7de);
}
.actions {
  display: flex;
  gap: 0.6rem;
}
button {
  padding: 0.55rem 0.9rem;
  border-radius: 6px;
  border: 1px solid var(--border, #d0d7de);
  cursor: pointer;
}
button.secondary {
  background: transparent;
}
.error {
  color: #d1242f;
}
.hint {
  color: var(--muted, #57606a);
  font-size: 0.85rem;
}
</style>
