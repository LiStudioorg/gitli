<script setup lang="ts">
import { RouterLink, useRouter } from 'vue-router'
import { useAuthStore, useRepoStore, useUiStore } from '../../stores'

const auth = useAuthStore()
const repos = useRepoStore()
const ui = useUiStore()
const router = useRouter()

function go(path: string) {
  ui.sidebarOpen = false
  router.push(path)
}
</script>

<template>
  <aside class="sidebar" :class="{ 'mobile-open': ui.sidebarOpen }">
    <section class="group">
      <h4>导航</h4>
      <nav class="nav">
        <RouterLink to="/dashboard">首页</RouterLink>
        <RouterLink to="/explore">探索</RouterLink>
        <RouterLink to="/explore">仓库</RouterLink>
        <RouterLink to="/explore?tab=topics">话题</RouterLink>
        <RouterLink to="/explore?tab=trending">趋势</RouterLink>
        <RouterLink to="/search">搜索</RouterLink>
        <RouterLink to="/settings">设置</RouterLink>
        <RouterLink v-if="auth.isAdmin" to="/admin">管理面板</RouterLink>
      </nav>
    </section>

    <section class="group">
      <h4>我的仓库</h4>
      <ul class="repos">
        <li v-for="r in repos.myRepos.slice(0, 5)" :key="r.id">
          <RouterLink :to="`/${auth.user?.username}/${r.name}`">{{ r.name }}</RouterLink>
        </li>
        <li v-if="repos.myRepos.length === 0" class="empty">暂无仓库</li>
      </ul>
      <RouterLink class="all" to="/explore">查看全部</RouterLink>
    </section>

    <section class="group">
      <h4>快捷操作</h4>
      <div class="quick">
        <button class="quick-btn" type="button" @click="go('/new/repo')">＋ 新建仓库</button>
        <button class="quick-btn" type="button" @click="go('/new/org')">＋ 新建组织</button>
        <button class="quick-btn" type="button" @click="go('/new/import')">＋ 导入仓库</button>
      </div>
    </section>
  </aside>
</template>

<style scoped>
.sidebar {
  width: 240px;
  padding: 1rem;
  border-right: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
  background: var(--card);
}
h4 {
  font-size: 0.78rem;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--muted);
  margin: 0 0 0.5rem;
}
.nav,
.repos {
  display: flex;
  flex-direction: column;
  gap: 0.2rem;
}
.repos {
  list-style: none;
  padding: 0;
  margin: 0 0 0.4rem;
}
.nav a,
.repos a {
  padding: 0.35rem 0.5rem;
  border-radius: 6px;
  color: var(--color-fg);
  display: block;
}
.nav a:hover,
.repos a:hover {
  background: var(--hover-bg);
  color: var(--primary);
}
.nav a.router-link-active {
  background: var(--primary-soft);
  color: var(--primary);
  font-weight: 600;
}
.all {
  font-size: 0.85rem;
}
.empty {
  color: var(--muted);
  font-size: 0.9rem;
  padding: 0.3rem 0.5rem;
}
.quick {
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
}
.quick-btn {
  text-align: left;
  padding: 0.4rem 0.6rem;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: transparent;
  color: var(--color-fg);
}
.quick-btn:hover {
  background: var(--hover-bg);
  border-color: var(--primary);
  color: var(--primary);
}
@media (max-width: 768px) {
  .sidebar {
    width: 100%;
    border-right: none;
    border-bottom: 1px solid var(--border);
    display: none;
  }
  .sidebar.mobile-open {
    display: flex;
  }
}
</style>
