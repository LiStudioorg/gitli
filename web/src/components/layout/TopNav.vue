<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { useAuthStore, useNotificationStore, useUiStore } from '../../stores'
import { useTheme } from '../../composables/useTheme'
import { useI18n } from 'vue-i18n'
import LogoIcon from '../icons/LogoIcon.vue'

const { t } = useI18n()

withDefaults(defineProps<{ variant?: 'landing' | 'app' }>(), {
  variant: 'landing',
})

const auth = useAuthStore()
const notifications = useNotificationStore()
const ui = useUiStore()
const route = useRoute()
const router = useRouter()
const { theme, cycleTheme } = useTheme()

const searchQuery = ref(typeof route.query.q === 'string' ? route.query.q : '')

watch(
  () => route.query.q,
  (q) => {
    searchQuery.value = typeof q === 'string' ? q : ''
  },
)

function submitSearch() {
  const q = searchQuery.value.trim()
  if (!q) return
  router.push({ path: '/search', query: { q } })
}

const themeIcon = computed(() => (theme.value === 'dark' ? '☀️' : '🌙'))
const themeLabel = computed(() => {
  if (theme.value === 'light') return '主题：浅色（点击切换到深色）'
  if (theme.value === 'dark') return '主题：深色（点击切换到自动）'
  return '主题：跟随系统（点击切换到浅色）'
})

function openPalette() {
  ui.commandPaletteOpen = true
}

async function logout() {
  await auth.logout()
  router.push('/')
}

const plusOpen = ref(false)
const avatarOpen = ref(false)

const createItems = [
  { label: '新建仓库', to: '/new/repo' },
  { label: '新建组织', to: '/new/org' },
  { label: '新建工单', to: '/new/repo', disabled: true, hint: '请先选择仓库' },
  { label: '导入仓库', to: '/new/import' },
]

function goCreate(path: string) {
  plusOpen.value = false
  router.push(path)
}

function goAvatar(path: string) {
  avatarOpen.value = false
  router.push(path)
}

function closeMenus() {
  plusOpen.value = false
  avatarOpen.value = false
}

watch(
  () => route.fullPath,
  () => closeMenus(),
)

const unread = computed(() => notifications.unread)
</script>

<template>
  <header
    class="sticky top-0 z-40 flex h-12 shrink-0 items-center border-b border-[#d1d9e0] bg-[#f6f8fa] px-4 dark:border-[#3d444d] dark:bg-[#010409]"
  >
    <div class="flex min-w-0 items-center gap-2 pr-2">
      <button
        v-if="variant === 'app'"
        class="inline-flex h-8 w-8 items-center justify-center rounded-md text-[#59636e] hover:bg-[#f6f8fa] md:hidden dark:hover:bg-[#21262d]"
        type="button"
        aria-label="打开侧边导航"
        @click="ui.toggleSidebar()"
      >
        <svg class="h-5 w-5" viewBox="0 0 16 16" fill="currentColor" aria-hidden="true">
          <path d="M1 2.75A.75.75 0 0 1 1.75 2h12.5a.75.75 0 0 1 0 1.5H1.75A.75.75 0 0 1 1 2.75Zm0 5A.75.75 0 0 1 1.75 7h12.5a.75.75 0 0 1 0 1.5H1.75A.75.75 0 0 1 1 7.75ZM1.75 12h12.5a.75.75 0 0 1 0 1.5H1.75a.75.75 0 0 1 0-1.5Z" />
        </svg>
      </button>

      <RouterLink class="flex shrink-0 items-center gap-2 text-base font-semibold text-[#1f2328] dark:text-[#e6edf3]" to="/" aria-label="gitli 首页">
        <LogoIcon />
        <span>gitli</span>
      </RouterLink>

      <template v-if="variant === 'landing'">
        <nav class="hidden items-center gap-4 text-sm text-[#59636e] md:flex">
          <a href="#features" class="hover:text-[#1f2328] dark:hover:text-[#e6edf3]">{{ t('nav.features') }}</a>
          <a href="https://github.com/LiStudioorg/gitli" target="_blank" rel="noopener" class="hover:text-[#1f2328] dark:hover:text-[#e6edf3]">{{ t('nav.docs') }}</a>
          <a href="#about" class="hover:text-[#1f2328] dark:hover:text-[#e6edf3]">{{ t('nav.about') }}</a>
        </nav>
      </template>
      <form v-else class="hidden min-w-0 flex-1 md:block" @submit.prevent="submitSearch">
        <input
          v-model="searchQuery"
          class="h-8 w-64 rounded-md border border-[#d1d9e0] bg-[#f6f8fa] px-3 text-sm text-[#1f2328] outline-none placeholder:text-[#59636e] focus:border-[#ea580c] focus:bg-white focus:ring-2 focus:ring-[#ea580c]/25 dark:border-[#30363d] dark:bg-[#0d1117] dark:text-[#e6edf3] dark:focus:bg-[#0d1117] lg:w-80"
          type="search"
          placeholder="搜索仓库、Issue、用户..."
          aria-label="全局搜索"
        />
      </form>
    </div>

    <nav v-if="variant === 'app'" class="hidden items-center gap-4 text-sm text-[#59636e] lg:flex">
      <RouterLink to="/dashboard" class="hover:text-[#1f2328] dark:hover:text-[#e6edf3]">控制台</RouterLink>
      <RouterLink to="/explore" class="hover:text-[#1f2328] dark:hover:text-[#e6edf3]">仓库</RouterLink>
      <RouterLink to="/explore?tab=trending" class="hover:text-[#1f2328] dark:hover:text-[#e6edf3]">趋势</RouterLink>
      <RouterLink to="/explore?tab=topics" class="hover:text-[#1f2328] dark:hover:text-[#e6edf3]">话题</RouterLink>
    </nav>

    <div class="ml-auto flex shrink-0 items-center gap-2 pl-2">
      <RouterLink
        v-if="variant === 'app'"
        to="/search"
        class="inline-flex h-8 w-8 items-center justify-center rounded-md text-[#59636e] hover:bg-[#f6f8fa] md:hidden dark:hover:bg-[#21262d]"
        aria-label="搜索"
      >
        <svg class="h-4 w-4" viewBox="0 0 16 16" fill="currentColor" aria-hidden="true">
          <path d="M10.68 11.74a6 6 0 0 1-7.922-8.982 6 6 0 0 1 8.982 7.922l3.04 3.04a.749.749 0 0 1-.326 1.275.749.749 0 0 1-.734-.215ZM11.5 7a4.499 4.499 0 1 0-8.997 0A4.499 4.499 0 0 0 11.5 7Z" />
        </svg>
      </RouterLink>

      <button
        class="hidden h-8 items-center rounded-md border border-[#d1d9e0] px-2 text-xs text-[#59636e] hover:bg-[#f6f8fa] md:inline-flex dark:border-[#30363d] dark:hover:bg-[#21262d]"
        type="button"
        aria-label="打开命令面板"
        title="命令面板 (Ctrl/Cmd+K)"
        @click="openPalette"
      >
        ⌘K
      </button>
      <button
        class="inline-flex h-8 w-8 items-center justify-center rounded-md text-[#59636e] hover:bg-[#f6f8fa] dark:hover:bg-[#21262d]"
        type="button"
        :aria-label="themeLabel"
        :title="themeLabel"
        @click="cycleTheme"
      >
        {{ themeIcon }}
      </button>

      <template v-if="auth.isLoggedIn">
        <RouterLink
          v-if="variant === 'landing'"
          class="inline-flex h-8 items-center rounded-md bg-[#ea580c] px-3 text-sm font-semibold text-white hover:bg-[#c2410c]"
          to="/dashboard"
        >
          {{ t('nav.dashboard') }}
        </RouterLink>
        <template v-else>
          <RouterLink
            class="relative inline-flex h-8 w-8 items-center justify-center rounded-md text-[#59636e] hover:bg-[#f6f8fa] dark:hover:bg-[#21262d]"
            to="/dashboard"
            title="通知"
            aria-label="通知"
          >
            🔔
            <span
              v-if="unread > 0"
              class="absolute -right-0.5 -top-0.5 inline-flex h-4 min-w-4 items-center justify-center rounded-full bg-[#cf222e] px-1 text-[10px] font-semibold text-white"
            >
              {{ unread > 99 ? '99+' : unread }}
            </span>
          </RouterLink>

          <div class="relative">
            <button
              class="inline-flex h-8 w-8 items-center justify-center rounded-md border border-[#d1d9e0] text-base font-semibold text-[#59636e] hover:bg-[#f6f8fa] dark:border-[#30363d] dark:hover:bg-[#21262d]"
              type="button"
              aria-label="快速创建"
              title="快速创建"
              @click="plusOpen = !plusOpen"
            >
              +
            </button>
            <div
              v-if="plusOpen"
              class="absolute right-0 top-full z-50 mt-1.5 flex w-48 flex-col rounded-md border border-[#d1d9e0] bg-white p-1 shadow-lg dark:border-[#30363d] dark:bg-[#161b22]"
              role="menu"
            >
              <button
                v-for="item in createItems"
                :key="item.label"
                type="button"
                role="menuitem"
                class="flex flex-col items-start gap-0.5 rounded-md px-2.5 py-1.5 text-left text-sm text-[#1f2328] hover:bg-[#f6f8fa] disabled:cursor-not-allowed disabled:opacity-45 dark:text-[#e6edf3] dark:hover:bg-[#21262d]"
                :disabled="item.disabled"
                @click="!item.disabled && goCreate(item.to)"
              >
                {{ item.label }}
                <small v-if="item.hint" class="text-xs text-[#59636e]">{{ item.hint }}</small>
              </button>
            </div>
          </div>

          <div class="relative">
            <button
              class="inline-flex h-8 w-8 items-center justify-center rounded-full bg-[#ea580c] text-sm font-semibold text-white"
              type="button"
              :aria-label="`用户菜单 ${auth.user?.username}`"
              @click="avatarOpen = !avatarOpen"
            >
              {{ auth.user?.username?.[0]?.toUpperCase() }}
            </button>
            <div
              v-if="avatarOpen"
              class="absolute right-0 top-full z-50 mt-1.5 flex w-44 flex-col rounded-md border border-[#d1d9e0] bg-white p-1 shadow-lg dark:border-[#30363d] dark:bg-[#161b22]"
              role="menu"
            >
              <div class="truncate px-2.5 py-1.5 text-xs text-[#59636e]">{{ auth.user?.username }}</div>
              <button type="button" role="menuitem" class="rounded-md px-2.5 py-1.5 text-left text-sm text-[#1f2328] hover:bg-[#f6f8fa] dark:text-[#e6edf3] dark:hover:bg-[#21262d]" @click="goAvatar(`/${auth.user?.username}`)">个人主页</button>
              <button type="button" role="menuitem" class="rounded-md px-2.5 py-1.5 text-left text-sm text-[#1f2328] hover:bg-[#f6f8fa] dark:text-[#e6edf3] dark:hover:bg-[#21262d]" @click="goAvatar('/settings')">设置</button>
              <button v-if="auth.isAdmin" type="button" role="menuitem" class="rounded-md px-2.5 py-1.5 text-left text-sm text-[#1f2328] hover:bg-[#f6f8fa] dark:text-[#e6edf3] dark:hover:bg-[#21262d]" @click="goAvatar('/admin')">管理面板</button>
              <button type="button" role="menuitem" class="rounded-md px-2.5 py-1.5 text-left text-sm text-[#cf222e] hover:bg-[#ffebe9]" @click="logout">退出登录</button>
            </div>
          </div>
        </template>
      </template>
      <template v-else>
        <RouterLink class="inline-flex h-8 items-center rounded-md px-2.5 text-sm text-[#1f2328] hover:bg-[#f6f8fa] dark:text-[#e6edf3] dark:hover:bg-[#21262d]" to="/login">{{ t('nav.login') }}</RouterLink>
        <RouterLink class="inline-flex h-8 items-center rounded-md bg-[#ea580c] px-2.5 text-sm font-semibold text-white hover:bg-[#c2410c]" to="/register">{{ t('nav.register') }}</RouterLink>
      </template>
    </div>
  </header>
</template>

