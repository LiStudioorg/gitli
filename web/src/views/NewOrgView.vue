<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import AppLayout from '../components/layout/AppLayout.vue'
import ErrorAlert from '../components/common/ErrorAlert.vue'
import api from '../api/client'

const router = useRouter()
const name = ref('')
const displayName = ref('')
const error = ref('')
const touched = ref(false)
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
  <AppLayout>
    <div class="mx-auto w-full max-w-[640px]">
      <h2 class="mb-4 text-lg font-semibold text-[#1f2328] dark:text-[#e6edf3]">新建组织</h2>
      <ErrorAlert v-if="error" title="操作失败" :message="error" @dismiss="error = ''" />
      <form
        class="flex flex-col gap-4 rounded-md border border-[#d0d7de] bg-white p-4 dark:border-[#30363d] dark:bg-[#161b22] md:p-6"
        @submit.prevent="submit"
      >
        <label class="flex flex-col gap-1.5">
          <span class="text-sm font-semibold text-[#1f2328] dark:text-[#e6edf3]">组织标识</span>
          <input
            v-model="name"
            placeholder="my-team"
            autocomplete="off"
            class="h-9 rounded-md border border-[#d0d7de] bg-[#f6f8fa] px-3 text-sm text-[#1f2328] outline-none placeholder:text-[#656d76] focus:border-[#0969da] focus:bg-white focus:ring-2 focus:ring-[#0969da]/25 dark:border-[#30363d] dark:bg-[#0d1117] dark:text-[#e6edf3]"
            @blur="touched = true"
          />
        </label>
        <label class="flex flex-col gap-1.5">
          <span class="text-sm font-semibold text-[#1f2328] dark:text-[#e6edf3]">显示名称（可选）</span>
          <input
            v-model="displayName"
            placeholder="我的团队"
            autocomplete="off"
            class="h-9 rounded-md border border-[#d0d7de] bg-[#f6f8fa] px-3 text-sm text-[#1f2328] outline-none placeholder:text-[#656d76] focus:border-[#0969da] focus:bg-white focus:ring-2 focus:ring-[#0969da]/25 dark:border-[#30363d] dark:bg-[#0d1117] dark:text-[#e6edf3]"
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
            {{ loading ? '处理中…' : '创建组织' }}
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
