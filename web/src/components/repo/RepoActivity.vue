<script setup lang="ts">
import { onMounted, ref } from 'vue'
import api from '../../api/client'
import { errMessage } from '../../utils/error'
import ActivityItem from '../common/ActivityItem.vue'
import EmptyState from '../common/EmptyState.vue'
import SkeletonBlock from '../common/SkeletonBlock.vue'
import type { Activity } from '../../stores'

const props = defineProps<{ owner: string; repo: string }>()

const items = ref<Activity[]>([])
const loading = ref(true)
const error = ref('')

async function load() {
  loading.value = true
  error.value = ''
  try {
    const { data } = await api.get<Activity[] | null>(`/repos/${props.owner}/${props.repo}/activities`)
    items.value = data ?? []
  } catch (e) {
    error.value = errMessage(e, '加载活动失败')
    items.value = []
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<template>
  <section class="activity">
    <header class="mb-3 flex items-center justify-between gap-2">
      <h3>活动</h3>
      <button class="inline-flex h-8 items-center rounded-md border border-[#d1d9e0] px-3 text-sm text-[#1f2328] hover:bg-[#f6f8fa] dark:border-[#3d444d] dark:text-[#e6edf3] dark:hover:bg-[#161b22]" type="button" @click="load">刷新</button>
    </header>
    <div v-if="loading" class="flex flex-col gap-2">
      <SkeletonBlock v-for="i in 4" :key="i" :lines="2" />
    </div>
    <p v-else-if="error" class="rounded-md border border-[#ff818266] bg-[#ffebe9] px-3 py-2 text-sm text-[#cf222e]">{{ error }}</p>
    <EmptyState
      v-else-if="!items.length"
      icon="📈"
      title="还没有活动记录"
      description="推送代码、创建工单或发布版本后，这里会显示时间线。"
      :show-action="false"
    />
    <div v-else class="flex flex-col gap-2">
      <ActivityItem v-for="a in items" :key="a.id" :activity="a" />
    </div>
  </section>
</template>
