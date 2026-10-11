<script setup lang="ts">
import { ref, onMounted } from 'vue'

const version = ref('')

onMounted(async () => {
  try {
    const res = await fetch('/api/v1/version')
    const data = await res.json()
    version.value = data.version ?? ''
  } catch {
    version.value = ''
  }
})
</script>

<template>
  <main class="home">
    <h1>gitli</h1>
    <p>轻量级自研 Git 服务平台</p>
    <p v-if="version" class="version">version {{ version }}</p>
  </main>
</template>

<style scoped>
.home {
  padding: 4rem 2rem;
  text-align: center;
}
.version {
  color: #888;
}
</style>
