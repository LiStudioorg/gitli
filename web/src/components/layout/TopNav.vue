<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { useAuthStore, useNotificationStore, useUiStore } from '../../stores'
import { useTheme } from '../../composables/useTheme'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const props = withDefaults(defineProps<{ variant?: 'landing' | 'app' }>(), {
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

const variantClass = computed(() => `topnav-${props.variant}`)

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
  <header class="topnav" :class="variantClass">
    <div class="left">
      <button
        v-if="variant === 'app'"
        class="icon-btn hamburger"
        type="button"
        aria-label="切换侧边栏"
        @click="ui.toggleSidebar()"
      >
        ☰
      </button>
      <RouterLink class="brand" to="/">gitli</RouterLink>
      <template v-if="variant === 'landing'">
        <nav class="links">
          <a href="#features">{{ t('nav.features') }}</a>
          <a href="https://github.com/LiStudioorg/gitli" target="_blank" rel="noopener">{{ t('nav.docs') }}</a>
          <a href="#about">{{ t('nav.about') }}</a>
        </nav>
      </template>
      <template v-else>
        <form class="search-form" @submit.prevent="submitSearch">
          <input
            v-model="searchQuery"
            class="search"
            type="search"
            placeholder="搜索仓库、Issue、用户..."
            aria-label="全局搜索"
          />
        </form>
      </template>
    </div>

    <nav v-if="variant === 'app'" class="app-links">
      <RouterLink to="/dashboard">控制台</RouterLink>
      <RouterLink to="/explore">仓库</RouterLink>
      <RouterLink to="/explore?tab=trending">趋势</RouterLink>
      <RouterLink to="/explore?tab=topics">话题</RouterLink>
      <RouterLink to="/search">搜索</RouterLink>
    </nav>

    <div class="right">
      <button
        class="icon-btn palette-trigger"
        type="button"
        aria-label="打开命令面板"
        title="命令面板 (Ctrl/Cmd+K)"
        @click="openPalette"
      >
        ⌘K
      </button>
      <button
        class="icon-btn theme-toggle"
        type="button"
        :aria-label="themeLabel"
        :title="themeLabel"
        @click="cycleTheme"
      >
        {{ themeIcon }}
      </button>
      <template v-if="auth.isLoggedIn">
        <RouterLink v-if="variant === 'landing'" class="btn primary" to="/dashboard">{{ t('nav.dashboard') }}</RouterLink>
        <template v-else>
          <RouterLink class="icon-btn bell" to="/dashboard" title="通知" aria-label="通知">
            🔔<span v-if="unread > 0" class="bell-badge">{{ unread > 99 ? '99+' : unread }}</span>
          </RouterLink>
          <div class="menu-wrap">
            <button class="icon-btn plus" type="button" aria-label="快速创建" title="快速创建" @click="plusOpen = !plusOpen">
              +
            </button>
            <div v-if="plusOpen" class="menu" role="menu">
              <button
                v-for="item in createItems"
                :key="item.label"
                type="button"
                role="menuitem"
                class="menu-item"
                :disabled="item.disabled"
                @click="!item.disabled && goCreate(item.to)"
              >
                {{ item.label }}
                <small v-if="item.hint">{{ item.hint }}</small>
              </button>
            </div>
          </div>
          <div class="menu-wrap">
            <button class="avatar" type="button" :aria-label="`用户菜单 ${auth.user?.username}`" @click="avatarOpen = !avatarOpen">
              {{ auth.user?.username?.[0]?.toUpperCase() }}
            </button>
            <div v-if="avatarOpen" class="menu" role="menu">
              <div class="menu-head">{{ auth.user?.username }}</div>
              <button type="button" class="menu-item" role="menuitem" @click="goAvatar(`/${auth.user?.username}`)">个人主页</button>
              <button type="button" class="menu-item" role="menuitem" @click="goAvatar('/settings')">设置</button>
              <button v-if="auth.isAdmin" type="button" class="menu-item" role="menuitem" @click="goAvatar('/admin')">管理面板</button>
              <button type="button" class="menu-item danger" role="menuitem" @click="logout">退出登录</button>
            </div>
          </div>
        </template>
      </template>
      <template v-else>
        <RouterLink class="btn" to="/login">{{ t('nav.login') }}</RouterLink>
        <RouterLink class="btn primary" to="/register">{{ t('nav.register') }}</RouterLink>
      </template>
    </div>
  </header>
</template>

<style scoped>
.topnav {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1.5rem;
  padding: 0.65rem 1.5rem;
  border-bottom: 1px solid var(--border);
  background: var(--card);
}
.left,
.right {
  display: flex;
  align-items: center;
  gap: 1rem;
}
.brand {
  font-weight: 700;
  font-size: 1.3rem;
  color: var(--color-fg);
}
.links {
  display: flex;
  gap: 1.1rem;
  color: var(--muted);
}
.search-form {
  display: flex;
}
.search {
  padding: 0.4rem 0.75rem;
  border-radius: 6px;
  border: 1px solid var(--border);
  background: var(--color-bg);
  color: var(--color-fg);
  min-width: 220px;
}
.btn {
  padding: 0.35rem 0.9rem;
  border: 1px solid var(--border);
  border-radius: 6px;
  font-size: 0.9rem;
  color: var(--color-fg);
  background: transparent;
}
.btn.primary {
  background: var(--primary);
  border-color: var(--primary);
  color: #fff;
}
.btn.primary:hover {
  background: var(--primary-hover);
}
.icon-btn {
  color: var(--color-fg);
  background: transparent;
  border: none;
  font-size: 1rem;
  padding: 0.25rem 0.4rem;
  border-radius: 6px;
  line-height: 1;
}
.icon-btn:hover {
  background: var(--hover-bg);
}
.palette-trigger {
  border: 1px solid var(--border);
  font-size: 0.8rem;
  color: var(--muted);
}
.hamburger {
  display: none;
}
.avatar {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background: var(--primary);
  color: #fff;
  font-weight: 600;
}
.dropdown-toggle {
  color: var(--color-fg);
}
.app-links {
  display: flex;
  gap: 1.1rem;
  color: var(--muted);
  font-size: 0.9rem;
}
.app-links a {
  color: var(--muted);
  text-decoration: none;
}
.app-links a:hover,
.app-links a.router-link-active {
  color: var(--color-fg);
}
.bell {
  position: relative;
}
.bell-badge {
  position: absolute;
  top: -4px;
  right: -6px;
  min-width: 16px;
  height: 16px;
  padding: 0 4px;
  border-radius: 999px;
  background: #d1242f;
  color: #fff;
  font-size: 0.68rem;
  font-weight: 600;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}
.menu-wrap {
  position: relative;
}
.plus {
  border: 1px solid var(--border);
  font-size: 1.1rem;
  font-weight: 600;
  width: 30px;
  height: 30px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}
.avatar {
  border: none;
  cursor: pointer;
}
.menu {
  position: absolute;
  right: 0;
  top: calc(100% + 6px);
  min-width: 180px;
  background: var(--card);
  border: 1px solid var(--border);
  border-radius: 8px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.12);
  padding: 0.3rem;
  z-index: 50;
  display: flex;
  flex-direction: column;
}
.menu-head {
  padding: 0.4rem 0.6rem;
  font-size: 0.8rem;
  color: var(--muted);
  border-bottom: 1px solid var(--border);
  margin-bottom: 0.25rem;
}
.menu-item {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 0.1rem;
  text-align: left;
  border: none;
  background: transparent;
  color: var(--color-fg);
  padding: 0.45rem 0.6rem;
  border-radius: 6px;
  font-size: 0.88rem;
  cursor: pointer;
  width: 100%;
}
.menu-item:hover:not(:disabled) {
  background: var(--hover-bg);
}
.menu-item:disabled {
  opacity: 0.45;
  cursor: default;
}
.menu-item small {
  color: var(--muted);
  font-size: 0.72rem;
}
.menu-item.danger {
  color: #d1242f;
}
@media (max-width: 900px) {
  .app-links {
    display: none;
  }
}
@media (max-width: 720px) {
  .search {
    min-width: 120px;
  }
  .links {
    display: none;
  }
}
@media (max-width: 768px) {
  .topnav-app .hamburger {
    display: inline-flex;
  }
}
</style>
