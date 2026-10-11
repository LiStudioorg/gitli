<script setup lang="ts">
import { useRoute } from 'vue-router'

const route = useRoute()
</script>

<template>
  <main class="repo">
    <h2>{{ route.params.owner }} / {{ route.params.repo }}</h2>
    <p>仓库首页（占位）</p>
  </main>
</template>

<style scoped>
.repo {
  padding: 2rem;
}
</style>
