<script setup lang="ts">
import UserSettingsView from '~/views/UserSettingsView.vue'


definePageMeta({ requiresAuth: true })
</script>

<template>
  <UserSettingsView />
</template>