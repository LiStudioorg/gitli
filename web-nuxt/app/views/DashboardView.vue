<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import AppLayout from '../components/layout/AppLayout.vue'
import ErrorAlert from '../components/common/ErrorAlert.vue'
import ActivityItem from '../components/common/ActivityItem.vue'
import ContributionGraph from '../components/common/ContributionGraph.vue'
import RepoSidebarCard from '../components/common/RepoSidebarCard.vue'
import SkeletonBlock from '../components/common/SkeletonBlock.vue'
import { useAuthStore, useNotificationStore, useRepoStore } from '../stores'
import type { Repo } from '../stores'
import { useSSE } from '../composables/useSSE'

const auth = useAuthStore()
const repos = useRepoStore()
const notifications = useNotificationStore()
const router = useRouter()
const removing = ref<number | null>(null)
const loading = ref(true)
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
  try {
    await Promise.all([
      repos.fetchMyRepos().catch(() => undefined),
      repos.fetchActivities().catch(() => undefined),
      notifications.fetchAll().catch(() => undefined),
    ])
  } finally {
    loading.value = false
  }
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
  <AppLayout>
    <ErrorAlert v-if="removeError" title="操作失败" :message="removeError" @dismiss="removeError = ''" />
    <ErrorAlert
      v-if="notifications.error && !notifications.items.length"
      title="通知加载失败"
      :message="notifications.error"
      @dismiss="notifications.error = ''"
    />

    <!-- 概览卡片：PC 三列，移动端单列 -->
    <section class="mb-6 grid grid-cols-1 gap-4 md:grid-cols-3">
      <div class="rounded-md border border-[#d0d7de] bg-white p-4 dark:border-[#30363d] dark:bg-[#161b22]">
        <p class="text-xs font-medium text-[#656d76]">我的仓库</p>
        <p class="mt-1 truncate text-xl font-semibold text-[#1f2328] dark:text-[#e6edf3]">
          {{ repos.myRepos.length }}
        </p>
      </div>
      <div class="rounded-md border border-[#d0d7de] bg-white p-4 dark:border-[#30363d] dark:bg-[#161b22]">
        <p class="text-xs font-medium text-[#656d76]">未读通知</p>
        <p class="mt-1 truncate text-xl font-semibold text-[#1f2328] dark:text-[#e6edf3]">
          {{ notifications.unread }}
        </p>
      </div>
      <div class="rounded-md border border-[#d0d7de] bg-white p-4 dark:border-[#30363d] dark:bg-[#161b22]">
        <p class="text-xs font-medium text-[#656d76]">最近活动</p>
        <p class="mt-1 truncate text-xl font-semibold text-[#1f2328] dark:text-[#e6edf3]">
          {{ repos.activities.length }}
        </p>
      </div>
    </section>

    <div class="grid grid-cols-1 items-start gap-4 lg:grid-cols-3">
      <section class="min-w-0 lg:col-span-2">
        <header class="mb-3 flex items-baseline gap-2">
          <h2 class="text-base font-semibold text-[#1f2328] dark:text-[#e6edf3]">动态流</h2>
          <span class="text-xs text-[#656d76]">你最近的仓库活动</span>
        </header>

        <div v-if="loading" class="flex flex-col gap-3">
          <div
            v-for="i in 4"
            :key="i"
            class="rounded-md border border-[#d0d7de] bg-white p-4 dark:border-[#30363d] dark:bg-[#161b22]"
          >
            <SkeletonBlock :lines="3" />
          </div>
        </div>
        <div v-else-if="hasActivities" class="flex flex-col gap-3">
          <ActivityItem v-for="a in repos.activities" :key="a.id" :activity="a" />
        </div>
        <div
          v-else
          class="rounded-md border border-dashed border-[#d0d7de] bg-white px-4 py-10 text-center dark:border-[#30363d] dark:bg-[#161b22]"
        >
          <p class="text-sm font-semibold text-[#1f2328] dark:text-[#e6edf3]">还没有任何动态</p>
          <p class="mx-auto mt-1 max-w-md text-sm text-[#656d76]">
            创建仓库、推送代码或提交工单后，这里会显示你的活动记录。
          </p>
          <RouterLink
            to="/new/repo"
            class="mt-4 inline-flex rounded-md bg-[#ea580c] px-3 py-1.5 text-sm font-semibold text-white hover:bg-[#c2410c]"
          >
            创建第一个仓库
          </RouterLink>
        </div>

        <section
          v-if="notifications.items.length"
          class="mt-6 rounded-md border border-[#d0d7de] bg-white p-4 dark:border-[#30363d] dark:bg-[#161b22]"
        >
          <header class="mb-2 flex items-center gap-2">
            <h2 class="text-base font-semibold text-[#1f2328] dark:text-[#e6edf3]">通知</h2>
            <span
              v-if="notifications.unread"
              class="rounded-full bg-[#ea580c] px-2 py-0.5 text-xs font-semibold text-white"
            >
              {{ notifications.unread }}
            </span>
          </header>
          <ul class="divide-y divide-[#d0d7de] dark:divide-[#30363d]">
            <li
              v-for="n in notifications.items"
              :key="n.ID"
              class="flex items-center gap-2 py-2 text-sm"
              :class="{ 'opacity-60': n.IsRead }"
            >
              <span
                class="h-2 w-2 shrink-0 rounded-full"
                :class="n.IsRead ? 'bg-transparent' : 'bg-[#ea580c]'"
              />
              <RouterLink v-if="n.URL" :to="n.URL" class="min-w-0 flex-1 truncate text-[#1f2328] hover:text-[#ea580c] dark:text-[#e6edf3]">
                {{ n.Title }}
              </RouterLink>
              <span v-else class="min-w-0 flex-1 truncate text-[#1f2328] dark:text-[#e6edf3]">{{ n.Title }}</span>
              <button
                v-if="!n.IsRead"
                type="button"
                class="shrink-0 text-xs font-medium text-[#ea580c] hover:underline"
                @click="markRead(n.ID)"
              >
                标记已读
              </button>
            </li>
          </ul>
        </section>
      </section>

      <aside class="flex min-w-0 flex-col gap-4">
        <RepoSidebarCard :repos="repos.myRepos" :username="auth.user?.username" @remove="removeRepo" />
        <ContributionGraph :activities="repos.activities" />
      </aside>
    </div>
  </AppLayout>
</template>

