<script setup lang="ts">
import { onMounted, ref } from 'vue'
import TopNav from '../components/layout/TopNav.vue'
import { useAuthStore } from '../stores'

const auth = useAuthStore()
const fullName = ref('')
const bio = ref('')
const saved = ref(false)

onMounted(async () => {
  if (!auth.loaded) await auth.fetchMe()
  fullName.value = auth.user?.full_name ?? ''
})
</script>

<template>
  <div>
    <TopNav />
    <main class="settings">
      <h2>用户设置</h2>
      <section class="card">
        <h3>个人资料</h3>
        <label>用户名<input :value="auth.user?.username" disabled /></label>
        <label>邮箱<input :value="auth.user?.email" disabled /></label>
        <label>姓名<input v-model="fullName" /></label>
        <label>简介<textarea v-model="bio" rows="3" /></label>
        <button @click="saved = true">保存</button>
        <span v-if="saved" class="ok">已保存（占位）</span>
      </section>
    </main>
  </div>
</template>

<style scoped>
.settings {
  max-width: 640px;
  margin: 0 auto;
  padding: 2rem 1.5rem;
}
.card {
  border: 1px solid var(--border, #d0d7de);
  border-radius: 10px;
  padding: 1.25rem;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}
label {
  display: flex;
  flex-direction: column;
  gap: 0.3rem;
  font-size: 0.9rem;
}
input,
textarea {
  padding: 0.5rem;
  border-radius: 6px;
  border: 1px solid var(--border, #d0d7de);
}
.ok {
  color: #1a7f37;
  font-size: 0.9rem;
}
</style>
