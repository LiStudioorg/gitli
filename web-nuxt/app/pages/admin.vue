<script setup lang="ts">
import AdminView from '~/views/AdminView.vue'


definePageMeta({ requiresAuth: true, requiresAdmin: true })
</script>

<template>
  <AdminView />
</template>