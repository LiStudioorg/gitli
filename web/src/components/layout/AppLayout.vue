<script setup lang="ts">
/**
 * 应用骨架：顶部导航与侧边栏固定，只有主内容区滚动。
 * PC（>=768px）显示 w-64 侧边栏；移动端隐藏侧边栏，由 TopNav 的汉堡按钮唤出抽屉。
 */
import TopNav from './TopNav.vue'
import Sidebar from './Sidebar.vue'
</script>

<template>
  <div class="flex h-screen flex-col overflow-hidden bg-[#f6f8fa]">
    <TopNav variant="app" />
    <div class="flex min-h-0 flex-1">
      <!-- 侧边栏：仅 PC 显示，自身可滚动 -->
      <Sidebar />
      <!-- 主内容：唯一滚动容器 -->
      <main class="min-w-0 flex-1 overflow-y-auto">
        <div class="mx-auto w-full max-w-[1200px] p-4 md:p-6">
          <slot />
        </div>
      </main>
    </div>
  </div>
</template>
