<script setup lang="ts">
import NewRepoView from '~/views/NewRepoView.vue'


definePageMeta({ requiresAuth: true })
</script>

<template>
  <NewRepoView />
</template>