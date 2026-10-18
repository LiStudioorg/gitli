<script setup lang="ts">
// 与原 web/src/App.vue 等价：路由出口 + 全局命令面板。
// 显式导入而不是依赖自动导入的名字：Nuxt 会按目录前缀命名
// （app/components/common/CommandPalette.vue → CommonCommandPalette），
// 显式导入可以让两个前端的行为完全一致。
import CommandPalette from '~/components/common/CommandPalette.vue'
</script>

<template>
  <NuxtPage />
  <CommandPalette />
</template>
