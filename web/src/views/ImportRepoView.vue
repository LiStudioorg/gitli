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
const sourceURL = ref('')
const error = ref('')
const touched = ref(false)
const loading = ref(false)

const disabledReason = computed(() => {
  if (!name.value.trim()) return '请填写仓库名称'
  if (!/^[A-Za-z0-9._-]+$/.test(name.value.trim())) return '仓库名只能包含字母、数字、点、下划线和连字符'
  if (!sourceURL.value.trim()) return '请填写来源仓库地址'
  if (!/^(https?|ssh):\/\/|^git@/.test(sourceURL.value.trim())) return '来源地址需以 http(s)://、ssh:// 或 git@ 开头'
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
    const repoName = name.value.trim()
    await api.post('/repos', {
      name: repoName,
      description: `imported from ${sourceURL.value.trim()}`,
      default_branch: 'main',
    })
    await api.post(`/repos/${auth.user?.username}/${repoName}/imports`, {
      source: 'git',
      source_url: sourceURL.value.trim(),
    })
    await router.push(`/${auth.user?.username}/${repoName}`)
  } catch (e: any) {
    error.value = e?.response?.data?.error ?? '导入失败'
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
        <h2>导入仓库</h2>
        <form class="form" @submit.prevent="submit">
          <label>
            新仓库名称
            <input v-model="name" placeholder="my-imported-repo" autocomplete="off" @blur="touched = true" />
          </label>
          <label>
            来源仓库地址
            <input v-model="sourceURL" placeholder="https://github.com/user/repo.git" autocomplete="off" @blur="touched = true" />
          </label>
          <div class="actions">
            <button type="submit" :disabled="loading">{{ loading ? '导入中…' : '开始导入' }}</button>
            <button type="button" class="secondary" @click="router.back()">取消</button>
          </div>
        </form>
        <p v-if="error" class="error">{{ error }}</p>
        <p v-else-if="touched && disabledReason" class="hint">{{ disabledReason }}</p>
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
