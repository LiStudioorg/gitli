<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import TopNav from '../components/layout/TopNav.vue'
import Sidebar from '../components/layout/Sidebar.vue'
import ActivityItem from '../components/common/ActivityItem.vue'
import ContributionGraph from '../components/common/ContributionGraph.vue'
import RepoSidebarCard from '../components/common/RepoSidebarCard.vue'
import { useAuthStore, useNotificationStore, useRepoStore } from '../stores'
import type { Repo } from '../stores'
import { useSSE } from '../composables/useSSE'

const auth = useAuthStore()
const repos = useRepoStore()
const notifications = useNotificationStore()
const router = useRouter()
const removing = ref<number | null>(null)
const removeError = ref('')

useSSE({
  onNotification: (item) => notifications.push(item),
})

onMounted(async () => {
  if (!auth.loaded) await auth.fetchMe()
  if (!auth.isLoggedIn) {
    await router.replace({ path: '/login', query: { redirect: '/dashboard' } })
    return
  }
  await Promise.all([
    repos.fetchMyRepos().catch(() => undefined),
    repos.fetchActivities().catch(() => undefined),
    notifications.fetchAll().catch(() => undefined),
  ])
})

const hasActivities = computed(() => repos.activities.length > 0)

async function markRead(id: number) {
  try {
    await notifications.markRead(id)
  } catch {
    notifications.error = '标记已读失败'
  }
}

async function removeRepo(repo: Repo) {
  if (!confirm(`确定删除仓库 ${repo.name}？此操作可在管理后台恢复。`)) return
  removeError.value = ''
  removing.value = repo.id
  try {
    await repos.removeRepo(repo)
  } catch {
    removeError.value = '删除仓库失败，请稍后重试'
  } finally {
    removing.value = null
  }
}
</script>

<template>
  <div class="dashboard">
    <TopNav variant="app" />
    <div class="shell">
      <Sidebar />
      <main class="content">
        <div class="columns">
          <section class="main-col">
            <header class="col-head">
              <h2>动态流</h2>
              <span class="muted">你最近的仓库活动</span>
            </header>

            <div v-if="hasActivities" class="feed">
              <ActivityItem v-for="a in repos.activities" :key="a.id" :activity="a" />
            </div>
            <div v-else class="empty-state">
              <p class="empty-title">还没有任何动态</p>
              <p class="muted">创建仓库、推送代码或提交工单后，这里会显示你的活动记录。</p>
              <RouterLink class="cta" to="/new/repo">创建第一个仓库</RouterLink>
            </div>

            <section v-if="notifications.items.length" class="panel">
              <header class="panel-head">
                <h2>通知</h2>
                <span v-if="notifications.unread" class="badge">{{ notifications.unread }}</span>
              </header>
              <p v-if="notifications.error" class="error">{{ notifications.error }}</p>
              <ul class="notif-list">
                <li v-for="n in notifications.items" :key="n.ID" :class="{ read: n.IsRead }">
                  <span class="dot" :class="{ unread: !n.IsRead }" />
                  <RouterLink v-if="n.URL" class="notif-title" :to="n.URL">{{ n.Title }}</RouterLink>
                  <span v-else class="notif-title">{{ n.Title }}</span>
                  <button v-if="!n.IsRead" class="link-btn" type="button" @click="markRead(n.ID)">
                    标记已读
                  </button>
                </li>
              </ul>
            </section>
          </section>

          <aside class="side-col">
            <RepoSidebarCard :repos="repos.myRepos" :username="auth.user?.username" @remove="removeRepo" />
            <p v-if="removeError" class="error">{{ removeError }}</p>
            <ContributionGraph :activities="repos.activities" />
          </aside>
        </div>
      </main>
    </div>
  </div>
</template>

<style scoped>
.shell {
  display: flex;
  min-height: calc(100vh - 57px);
}
.content {
  flex: 1;
  min-width: 0;
  padding: 1.5rem;
}
.columns {
  display: grid;
  grid-template-columns: minmax(0, 7fr) minmax(260px, 3fr);
  gap: 1.5rem;
  align-items: start;
  max-width: 1200px;
}
.col-head {
  display: flex;
  align-items: baseline;
  gap: 0.6rem;
  margin-bottom: 0.9rem;
}
.col-head h2 {
  margin: 0;
  font-size: 1.1rem;
}
.feed {
  display: flex;
  flex-direction: column;
  gap: 0.6rem;
}
.side-col {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  position: sticky;
  top: 1.5rem;
}
.empty-state {
  border: 1px dashed var(--border);
  border-radius: 10px;
  padding: 2rem 1.25rem;
  text-align: center;
  background: var(--card);
}
.empty-title {
  margin: 0 0 0.35rem;
  font-weight: 600;
}
.muted {
  color: var(--muted);
  font-size: 0.85rem;
}
.cta {
  display: inline-block;
  margin-top: 0.9rem;
  padding: 0.45rem 0.9rem;
  border-radius: 6px;
  background: var(--primary);
  color: #fff;
  text-decoration: none;
  font-size: 0.86rem;
}
.panel {
  border: 1px solid var(--border);
  border-radius: 10px;
  background: var(--card);
  padding: 0.9rem 1rem;
  margin-top: 1.5rem;
}
.panel-head {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}
.panel-head h2 {
  margin: 0;
  font-size: 1rem;
}
.badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 20px;
  height: 20px;
  padding: 0 0.4rem;
  border-radius: 999px;
  background: var(--accent, #f59e0b);
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
.notif-list li.read {
  opacity: 0.6;
}
.dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: transparent;
  flex-shrink: 0;
}
.dot.unread {
  background: var(--primary);
}
.notif-title {
  flex: 1;
  min-width: 0;
  color: var(--fg);
  text-decoration: none;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
a.notif-title:hover {
  color: var(--primary);
}
.link-btn {
  border: none;
  background: transparent;
  color: var(--primary);
  font-size: 0.78rem;
  cursor: pointer;
  padding: 0;
}
.error {
  color: #d1242f;
  font-size: 0.82rem;
}
@media (max-width: 960px) {
  .columns {
    grid-template-columns: minmax(0, 1fr);
  }
  .side-col {
    position: static;
  }
}
</style>
