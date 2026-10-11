<script setup lang="ts">
import TopNav from '../components/layout/TopNav.vue'
</script>

<template>
  <div>
    <TopNav />
    <main class="admin">
      <h2>管理面板</h2>
      <p>用户、仓库、任务、审计日志、系统状态（占位，仅管理员可见）。</p>
    </main>
  </div>
</template>

<style scoped>
.admin {
  max-width: 1000px;
  margin: 0 auto;
  padding: 2rem 1.5rem;
}
</style>
