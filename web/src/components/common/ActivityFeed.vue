<script setup lang="ts">
import { onMounted, ref } from 'vue'
import api from '../../api/client'

interface PublicActivity {
  id: number
  actor_id: number
  verb: string
  object_type: string
  object_id: number
  repo_id: number | null
  created_at: string
}

const props = withDefaults(defineProps<{ limit?: number }>(), { limit: 20 })

const items = ref<PublicActivity[]>([])
const loading = ref(true)
const error = ref('')

onMounted(async () => {
  try {
    const { data } = await api.get<PublicActivity[]>('/explore/activity', {
      params: { limit: props.limit },
    })
    items.value = data ?? []
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'failed to load activity'
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <section class="activity-feed">
    <h3>活动</h3>
    <p v-if="loading" class="state">加载中…</p>
    <p v-else-if="error" class="state error">{{ error }}</p>
    <p v-else-if="items.length === 0" class="state">暂无活动</p>
    <ul v-else>
      <li v-for="a in items" :key="a.id">
        <span class="verb">{{ a.verb }}</span>
        <span class="object">{{ a.object_type }} #{{ a.object_id }}</span>
        <time class="time">{{ a.created_at }}</time>
      </li>
    </ul>
  </section>
</template>
