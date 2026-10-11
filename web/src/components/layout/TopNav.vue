<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { useAuthStore } from '../../stores'

const props = withDefaults(defineProps<{ variant?: 'landing' | 'app' }>(), {
  variant: 'landing',
})

const auth = useAuthStore()
const router = useRouter()

const variantClass = computed(() => `topnav-${props.variant}`)

async function logout() {
  await auth.logout()
  router.push('/')
}
</script>

<template>
  <header class="topnav" :class="variantClass">
    <div class="left">
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
      <template v-if="auth.isLoggedIn">
        <RouterLink v-if="variant === 'app'" class="icon-btn" to="/dashboard" title="通知">通知</RouterLink>
        <RouterLink v-if="variant === 'landing'" class="btn primary" to="/dashboard">进入控制台</RouterLink>
        <template v-else>
          <span class="avatar">{{ auth.user?.username?.[0]?.toUpperCase() }}</span>
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
</style>
