<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import TopNav from '../components/layout/TopNav.vue'
import Sidebar from '../components/layout/Sidebar.vue'
import api from '../api/client'
import { useAuthStore } from '../stores'

const router = useRouter()
const auth = useAuthStore()
const name = ref('')
const description = ref('')
const isPrivate = ref(false)
const error = ref('')
const loading = ref(false)

const disabledReason = computed(() => {
  if (!name.value.trim()) return '请填写仓库名称'
  if (!/^[A-Za-z0-9._-]+$/.test(name.value.trim())) return '仓库名只能包含字母、数字、点、下划线和连字符'
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
    await api.post('/repos', {
      name: name.value.trim(),
      description: description.value,
      default_branch: 'main',
      is_private: isPrivate.value,
    })
    await router.push(`/${auth.user?.username}/${name.value.trim()}`)
  } catch (e: any) {
    error.value = e?.response?.data?.error ?? '创建仓库失败'
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
        <h2>新建仓库</h2>
        <form class="form" @submit.prevent="submit">
          <label>
            仓库名称
            <input v-model="name" placeholder="my-project" autocomplete="off" />
          </label>
          <label>
            描述（可选）
            <input v-model="description" placeholder="一句话介绍这个仓库" />
          </label>
          <label class="checkbox">
            <input v-model="isPrivate" type="checkbox" />
            私有仓库
          </label>
          <div class="actions">
            <button type="submit" :disabled="loading">{{ loading ? '创建中…' : '创建仓库' }}</button>
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
label.checkbox {
  flex-direction: row;
  align-items: center;
  gap: 0.5rem;
}
input {
  padding: 0.55rem 0.75rem;
  border-radius: 6px;
  border: 1px solid var(--border, #d0d7de);
}
input[type='checkbox'] {
  width: auto;
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
