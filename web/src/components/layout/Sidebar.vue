<script setup lang="ts">
import { RouterLink, useRouter } from 'vue-router'
import { useAuthStore, useRepoStore, useUiStore } from '../../stores'
import NavIcon from '../icons/NavIcon.vue'

const auth = useAuthStore()
const repos = useRepoStore()
const ui = useUiStore()
const router = useRouter()

const navItems = [
  { to: '/dashboard', label: '首页', icon: 'home' },
  { to: '/explore', label: '探索', icon: 'compass' },
  { to: '/explore?tab=topics', label: '话题', icon: 'tag' },
  { to: '/explore?tab=trending', label: '趋势', icon: 'trend' },
  { to: '/search', label: '搜索', icon: 'search' },
  { to: '/settings', label: '设置', icon: 'gear' },
]
const quickActions = [
  { path: '/new/repo', label: '新建仓库', icon: 'repo' },
  { path: '/new/org', label: '新建组织', icon: 'plus' },
  { path: '/new/import', label: '导入仓库', icon: 'trend' },
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
    class="fixed inset-y-0 left-0 z-40 flex w-64 shrink-0 flex-col gap-6 overflow-y-auto border-r border-[#d1d9e0] bg-white p-4 pt-20 transition-transform dark:border-[#30363d] dark:bg-[#161b22] md:static md:z-auto md:w-[220px] md:translate-x-0 md:pt-4"
    :class="ui.sidebarOpen ? 'translate-x-0 shadow-xl' : '-translate-x-full md:shadow-none'"
    aria-label="侧边导航"
  >
    <section>
      <h4 class="mb-2 text-xs font-semibold uppercase tracking-wide text-[#59636e]">导航</h4>
      <nav class="flex flex-col gap-0.5">
        <RouterLink
          v-for="item in navItems"
          :key="item.to + item.label"
          :to="item.to"
          class="flex items-center gap-2 rounded-md px-2.5 py-1.5 text-sm text-[#1f2328] hover:bg-[#f6f8fa] dark:text-[#e6edf3] dark:hover:bg-[#161b22]"
          active-class="!bg-[#ea580c]/10 font-semibold text-[#ea580c] hover:!bg-[#ea580c]/10 dark:!bg-[#ea580c]/15 dark:text-[#f97316]"
          @click="ui.sidebarOpen = false"
        >
          <NavIcon :name="item.icon" />
          <span class="truncate">{{ item.label }}</span>
        </RouterLink>
        <RouterLink
          v-if="auth.isAdmin"
          to="/admin"
          class="flex items-center gap-2 rounded-md px-2.5 py-1.5 text-sm text-[#1f2328] hover:bg-[#f6f8fa] dark:text-[#e6edf3] dark:hover:bg-[#161b22]"
          active-class="bg-[#ea580c]/10 font-semibold text-[#ea580c] hover:bg-[#ea580c]/10 dark:bg-[#ea580c]/15 dark:text-[#f97316]"
          @click="ui.sidebarOpen = false"
        >
          <NavIcon name="shield" />
          <span class="truncate">管理面板</span>
        </RouterLink>
      </nav>
    </section>

    <section>
      <h4 class="mb-2 text-xs font-semibold uppercase tracking-wide text-[#59636e]">我的仓库</h4>
      <ul class="flex flex-col gap-0.5">
        <li v-for="r in repos.myRepos.slice(0, 5)" :key="r.id" class="min-w-0">
          <RouterLink
            :to="`/${auth.user?.username}/${r.name}`"
            class="flex items-center gap-2 rounded-md px-2.5 py-1.5 text-sm text-[#1f2328] hover:bg-[#f6f8fa] dark:text-[#e6edf3] dark:hover:bg-[#161b22]"
            @click="ui.sidebarOpen = false"
          >
            <NavIcon name="repo" />
            <span class="truncate">{{ r.name }}</span>
          </RouterLink>
        </li>
        <li v-if="repos.myRepos.length === 0" class="px-2.5 py-1.5 text-sm text-[#59636e]">暂无仓库</li>
      </ul>
      <RouterLink to="/explore" class="mt-1 inline-block px-2.5 text-xs text-[#ea580c] hover:underline" @click="ui.sidebarOpen = false">
        查看全部
      </RouterLink>
    </section>

    <section>
      <h4 class="mb-2 text-xs font-semibold uppercase tracking-wide text-[#59636e]">快捷操作</h4>
      <div class="flex flex-col gap-1.5">
        <button
          v-for="q in quickActions"
          :key="q.path"
          type="button"
          class="flex w-full items-center gap-2 rounded-md px-2.5 py-1.5 text-left text-sm text-[#1f2328] hover:bg-[#f6f8fa] dark:text-[#e6edf3] dark:hover:bg-[#161b22]"
          @click="go(q.path)"
        >
          <NavIcon :name="q.icon" />
          {{ q.label }}
        </button>
      </div>
    </section>
  </aside>
</template>

