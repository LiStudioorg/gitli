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
const description = ref('')
const isPrivate = ref(false)
const error = ref('')
const touched = ref(false)
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
  <AppLayout>
    <div class="mx-auto w-full max-w-[640px]">
      <h2 class="mb-4 text-lg font-semibold text-[#1f2328] dark:text-[#e6edf3]">新建仓库</h2>
      <ErrorAlert v-if="error" title="创建失败" :message="error" @dismiss="error = ''" />
      <form
        class="flex flex-col gap-4 rounded-md border border-[#d0d7de] bg-white p-4 dark:border-[#30363d] dark:bg-[#161b22] md:p-6"
        @submit.prevent="submit"
      >
        <label class="flex flex-col gap-1.5">
          <span class="text-sm font-semibold text-[#1f2328] dark:text-[#e6edf3]">仓库名称</span>
          <input
            v-model="name"
            placeholder="my-project"
            autocomplete="off"
            class="h-9 rounded-md border border-[#d0d7de] bg-[#f6f8fa] px-3 text-sm text-[#1f2328] outline-none placeholder:text-[#656d76] focus:border-[#0969da] focus:bg-white focus:ring-2 focus:ring-[#0969da]/25 dark:border-[#30363d] dark:bg-[#0d1117] dark:text-[#e6edf3]"
            @blur="touched = true"
          />
          <span v-if="touched && disabledReason" class="text-xs text-[#cf222e]">{{ disabledReason }}</span>
        </label>
        <label class="flex flex-col gap-1.5">
          <span class="text-sm font-semibold text-[#1f2328] dark:text-[#e6edf3]">描述（可选）</span>
          <input
            v-model="description"
            placeholder="一句话介绍这个仓库"
            class="h-9 rounded-md border border-[#d0d7de] bg-[#f6f8fa] px-3 text-sm text-[#1f2328] outline-none placeholder:text-[#656d76] focus:border-[#0969da] focus:bg-white focus:ring-2 focus:ring-[#0969da]/25 dark:border-[#30363d] dark:bg-[#0d1117] dark:text-[#e6edf3]"
          />
        </label>
        <label class="flex items-center gap-2 text-sm text-[#1f2328] dark:text-[#e6edf3]">
          <input v-model="isPrivate" type="checkbox" class="h-4 w-4 rounded border-[#d0d7de] accent-[#ea580c]" />
          私有仓库
        </label>
        <div class="flex gap-2">
          <button
            type="submit"
            :disabled="loading"
            class="inline-flex h-9 items-center rounded-md bg-[#2da44e] px-4 text-sm font-semibold text-white hover:bg-[#2c974b] disabled:opacity-50"
          >
            {{ loading ? '创建中…' : '创建仓库' }}
          </button>
          <button
            type="button"
            class="inline-flex h-9 items-center rounded-md border border-[#d0d7de] px-4 text-sm text-[#1f2328] hover:bg-[#f6f8fa] dark:border-[#30363d] dark:text-[#e6edf3] dark:hover:bg-[#21262d]"
            @click="router.back()"
          >
            取消
          </button>
        </div>
      </form>
    </div>
  </AppLayout>
</template>
