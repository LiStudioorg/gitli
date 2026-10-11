<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { useAuthStore, useUiStore } from '../../stores'
import { useTheme } from '../../composables/useTheme'

const props = withDefaults(defineProps<{ variant?: 'landing' | 'app' }>(), {
  variant: 'landing',
})

const auth = useAuthStore()
const ui = useUiStore()
const router = useRouter()
const { theme, cycleTheme } = useTheme()

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
          <a href="#features">功能</a>
          <a href="https://github.com/LiStudioorg/gitli" target="_blank" rel="noopener">文档</a>
          <a href="#about">关于</a>
        </nav>
      </template>
      <template v-else>
        <input class="search" type="search" placeholder="搜索..." />
      </template>
    </div>

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
        <RouterLink v-if="variant === 'app'" class="icon-btn" to="/dashboard" title="通知">通知</RouterLink>
        <RouterLink v-if="variant === 'landing'" class="btn primary" to="/dashboard">进入控制台</RouterLink>
        <template v-else>
          <span class="avatar">{{ auth.user?.username?.[0]?.toUpperCase() }}</span>
          <RouterLink class="icon-btn" to="/settings" title="用户设置">设置</RouterLink>
          <RouterLink v-if="auth.isAdmin" class="icon-btn" to="/admin" title="管理面板">管理</RouterLink>
          <button class="btn dropdown-toggle" @click="logout">退出</button>
        </template>
      </template>
      <template v-else>
        <RouterLink class="btn" to="/login">登录</RouterLink>
        <RouterLink class="btn primary" to="/register">注册</RouterLink>
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
