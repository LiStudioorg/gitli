<script setup lang="ts">
import { onMounted } from 'vue'
import TopNav from '../components/layout/TopNav.vue'
import Sidebar from '../components/layout/Sidebar.vue'
import ActivityItem from '../components/common/ActivityItem.vue'
import { useAuthStore, useNotificationStore, useRepoStore } from '../stores'
import { useSSE } from '../composables/useSSE'

const auth = useAuthStore()
const repos = useRepoStore()
const notifications = useNotificationStore()

useSSE({
  onNotification: (item) => notifications.push(item),
})

onMounted(async () => {
  if (!auth.loaded) await auth.fetchMe()
  await Promise.all([
    repos.fetchMyRepos(),
    repos.fetchActivities(),
    notifications.fetchAll(),
  ])
})

async function markRead(id: number) {
  try {
    await notifications.markRead(id)
  } catch {
    notifications.error = '标记已读失败'
  }
}
</script>

<template>
  <div class="dashboard">
    <TopNav variant="app" />
    <div class="body">
      <Sidebar />
      <main class="content">
        <section class="panel">
          <header class="panel-head">
            <h2>通知</h2>
            <span v-if="notifications.unread" class="badge">{{ notifications.unread }}</span>
          </header>
          <p v-if="notifications.error" class="error">{{ notifications.error }}</p>
          <ul v-if="notifications.items.length" class="notif-list">
            <li v-for="n in notifications.items" :key="n.ID" :class="{ read: n.IsRead }">
              <span class="dot" :class="{ unread: !n.IsRead }" />
              <span class="notif-title">{{ n.Title }}</span>
              <button v-if="!n.IsRead" class="link-btn" @click="markRead(n.ID)">标记已读</button>
            </li>
          </ul>
          <p v-else-if="!notifications.loading" class="empty">暂无通知</p>
        </section>

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
.panel {
  border: 1px solid var(--border);
  border-radius: 10px;
  background: var(--card);
  padding: 0.9rem 1rem;
  margin-bottom: 1.5rem;
}
.panel-head {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}
.panel-head h2 {
  margin: 0;
}
.badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 20px;
  height: 20px;
  padding: 0 0.4rem;
  border-radius: 999px;
  background: var(--accent);
  color: #fff;
  font-size: 0.75rem;
  font-weight: 600;
}
.notif-list {
  list-style: none;
  padding: 0;
  margin: 0.75rem 0 0;
}
.notif-list li {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  padding: 0.5rem 0;
  border-bottom: 1px solid var(--border);
  font-size: 0.9rem;
}
.notif-list li:last-child {
  border-bottom: none;
}
.notif-list li.read .notif-title {
  color: var(--muted);
}
.dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--border);
  flex-shrink: 0;
}
.dot.unread {
  background: var(--primary);
}
.notif-title {
  flex: 1;
}
.link-btn {
  border: none;
  background: transparent;
  color: var(--primary);
  padding: 0;
  font-size: 0.82rem;
}
.link-btn:hover {
  text-decoration: underline;
}
.error {
  color: var(--accent);
  font-size: 0.85rem;
}
@media (max-width: 768px) {
  .body {
    flex-direction: column;
  }
}
</style>
