<script setup lang="ts">
import { RouterLink, useRouter } from 'vue-router'
import { useAuthStore, useRepoStore, useUiStore } from '../../stores'

const auth = useAuthStore()
const repos = useRepoStore()
const ui = useUiStore()
const router = useRouter()

const navItems = [
  { to: '/dashboard', label: '首页' },
  { to: '/explore', label: '探索' },
  { to: '/explore?tab=topics', label: '话题' },
  { to: '/explore?tab=trending', label: '趋势' },
  { to: '/search', label: '搜索' },
  { to: '/settings', label: '设置' },
]
const quickActions = [
  { path: '/new/repo', label: '新建仓库' },
  { path: '/new/org', label: '新建组织' },
  { path: '/new/import', label: '导入仓库' },
]

function go(path: string) {
  ui.sidebarOpen = false
  router.push(path)
}
</script>

<template>
  <!-- 移动端遮罩 -->
  <div
    v-if="ui.sidebarOpen"
    class="fixed inset-0 z-30 bg-black/40 md:hidden"
    aria-hidden="true"
    @click="ui.sidebarOpen = false"
  />
  <aside
    class="fixed inset-y-0 left-0 z-40 flex w-64 shrink-0 flex-col gap-6 overflow-y-auto border-r border-[#d0d7de] bg-white p-4 transition-transform dark:border-[#30363d] dark:bg-[#161b22] md:static md:z-auto md:translate-x-0"
    :class="ui.sidebarOpen ? 'translate-x-0 shadow-xl' : '-translate-x-full md:shadow-none'"
    aria-label="侧边导航"
  >
    <section>
      <h4 class="mb-2 text-xs font-semibold uppercase tracking-wide text-[#656d76]">导航</h4>
      <nav class="flex flex-col gap-0.5">
        <RouterLink
          v-for="item in navItems"
          :key="item.to + item.label"
          :to="item.to"
          class="truncate rounded-md px-2.5 py-1.5 text-sm text-[#1f2328] hover:bg-[#f6f8fa] dark:text-[#e6edf3] dark:hover:bg-[#21262d]"
          active-class="bg-[#ea580c]/10 font-semibold text-[#ea580c] hover:bg-[#ea580c]/10 dark:bg-[#ea580c]/15 dark:text-[#f97316]"
          @click="ui.sidebarOpen = false"
        >
          {{ item.label }}
        </RouterLink>
        <RouterLink
          v-if="auth.isAdmin"
          to="/admin"
          class="truncate rounded-md px-2.5 py-1.5 text-sm text-[#1f2328] hover:bg-[#f6f8fa] dark:text-[#e6edf3] dark:hover:bg-[#21262d]"
          active-class="bg-[#ea580c]/10 font-semibold text-[#ea580c] hover:bg-[#ea580c]/10 dark:bg-[#ea580c]/15 dark:text-[#f97316]"
          @click="ui.sidebarOpen = false"
        >
          管理面板
        </RouterLink>
      </nav>
    </section>

    <section>
      <h4 class="mb-2 text-xs font-semibold uppercase tracking-wide text-[#656d76]">我的仓库</h4>
      <ul class="flex flex-col gap-0.5">
        <li v-for="r in repos.myRepos.slice(0, 5)" :key="r.id" class="min-w-0">
          <RouterLink
            :to="`/${auth.user?.username}/${r.name}`"
            class="block truncate rounded-md px-2.5 py-1.5 text-sm text-[#1f2328] hover:bg-[#f6f8fa] dark:text-[#e6edf3] dark:hover:bg-[#21262d]"
            @click="ui.sidebarOpen = false"
          >
            {{ r.name }}
          </RouterLink>
        </li>
        <li v-if="repos.myRepos.length === 0" class="px-2.5 py-1.5 text-sm text-[#656d76]">暂无仓库</li>
      </ul>
      <RouterLink to="/explore" class="mt-1 inline-block px-2.5 text-xs text-[#ea580c] hover:underline" @click="ui.sidebarOpen = false">
        查看全部
      </RouterLink>
    </section>

    <section>
      <h4 class="mb-2 text-xs font-semibold uppercase tracking-wide text-[#656d76]">快捷操作</h4>
      <div class="flex flex-col gap-1.5">
        <button
          v-for="q in quickActions"
          :key="q.path"
          type="button"
          class="w-full truncate rounded-md border border-[#d0d7de] px-2.5 py-1.5 text-left text-sm text-[#1f2328] hover:border-[#ea580c]/50 hover:bg-[#ea580c]/5 dark:border-[#30363d] dark:text-[#e6edf3]"
          @click="go(q.path)"
        >
          ＋ {{ q.label }}
        </button>
      </div>
    </section>
  </aside>
</template>

