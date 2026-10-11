<script setup lang="ts">
import { onMounted } from 'vue'
import TopNav from '../components/layout/TopNav.vue'
import Sidebar from '../components/layout/Sidebar.vue'
import ActivityItem from '../components/common/ActivityItem.vue'
import { useAuthStore, useRepoStore } from '../stores'

const auth = useAuthStore()
const repos = useRepoStore()

onMounted(async () => {
  if (!auth.loaded) await auth.fetchMe()
  await Promise.all([repos.fetchMyRepos(), repos.fetchActivities()])
})
</script>

<template>
  <div class="dashboard">
    <TopNav variant="app" />
    <div class="body">
      <Sidebar />
      <main class="content">
        <h2>动态流</h2>
        <div v-if="repos.activities.length" class="feed">
          <ActivityItem v-for="a in repos.activities" :key="a.id" :activity="a" />
        </div>
        <p v-else class="empty">暂无动态</p>
      </main>
    </div>
  </div>
</template>

<style scoped>
.body {
  display: flex;
  min-height: calc(100vh - 57px);
}
.content {
  flex: 1;
  padding: 1.5rem;
}
.content h2 {
  margin-top: 0;
}
.feed {
  display: flex;
  flex-direction: column;
  gap: 0.6rem;
}
.empty {
  color: var(--muted);
}
@media (max-width: 768px) {
  .body {
    flex-direction: column;
  }
}
</style>
