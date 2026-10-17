<script setup lang="ts">
import { RouterView } from 'vue-router'
import CommandPalette from './components/common/CommandPalette.vue'
import ToastHost from './components/common/ToastHost.vue'
</script>

<template>
  <RouterView />
  <CommandPalette />
  <ToastHost />
</template>
