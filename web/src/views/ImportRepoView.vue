<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import AppLayout from '../components/layout/AppLayout.vue'
import ErrorAlert from '../components/common/ErrorAlert.vue'
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
  <AppLayout>
    <div class="mx-auto w-full max-w-2xl">
      <h2 class="mb-4 text-lg font-semibold text-[#1f2328] dark:text-[#e6edf3]">导入仓库</h2>
      <ErrorAlert v-if="error" title="操作失败" :message="error" @dismiss="error = ''" />
      <form
        class="flex flex-col gap-4 rounded-md border border-[#d1d9e0] bg-white p-4 dark:border-[#30363d] dark:bg-[#161b22] md:p-6"
        @submit.prevent="submit"
      >
        <label class="flex flex-col gap-1.5">
          <span class="text-sm font-semibold text-[#1f2328] dark:text-[#e6edf3]">新仓库名称 <span class="text-[#cf222e]">*</span></span>
          <input
            v-model="name"
            placeholder="my-imported-repo"
            autocomplete="off"
            class="h-9 rounded-md border border-[#d1d9e0] bg-[#f6f8fa] px-3 text-sm text-[#1f2328] outline-none placeholder:text-[#59636e] focus:border-[#0969da] focus:bg-white focus:ring-2 focus:ring-[#0969da]/25 dark:border-[#30363d] dark:bg-[#0d1117] dark:text-[#e6edf3]"
            @blur="touched = true"
          />
        </label>
        <label class="flex flex-col gap-1.5">
          <span class="text-sm font-semibold text-[#1f2328] dark:text-[#e6edf3]">来源仓库地址 <span class="text-[#cf222e]">*</span></span>
          <input
            v-model="sourceURL"
            placeholder="https://example.com/user/repo.git"
            autocomplete="off"
            class="h-9 rounded-md border border-[#d1d9e0] bg-[#f6f8fa] px-3 text-sm text-[#1f2328] outline-none placeholder:text-[#59636e] focus:border-[#0969da] focus:bg-white focus:ring-2 focus:ring-[#0969da]/25 dark:border-[#30363d] dark:bg-[#0d1117] dark:text-[#e6edf3]"
            @blur="touched = true"
          />
        </label>
        <p v-if="touched && disabledReason" class="text-xs text-[#cf222e]">{{ disabledReason }}</p>
        <div class="flex gap-2">
          <button
            type="submit"
            :disabled="loading"
            class="inline-flex h-9 items-center rounded-md bg-[#2da44e] px-4 text-sm font-semibold text-white hover:bg-[#2c974b] disabled:opacity-50"
          >
            {{ loading ? '处理中…' : '开始导入' }}
          </button>
          <button
            type="button"
            class="inline-flex h-9 items-center rounded-md border border-[#d1d9e0] px-4 text-sm text-[#1f2328] hover:bg-[#f6f8fa] dark:border-[#30363d] dark:text-[#e6edf3] dark:hover:bg-[#21262d]"
            @click="router.back()"
          >
            取消
          </button>
        </div>
      </form>
    </div>
  </AppLayout>
</template>
