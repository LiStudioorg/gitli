<script setup lang="ts">
import { RouterView } from 'vue-router'
import CommandPalette from './components/common/CommandPalette.vue'
import ToastHost from './components/common/ToastHost.vue'
import ShortcutHelp from './components/common/ShortcutHelp.vue'
import { useGlobalShortcuts } from './composables/useShortcuts'
useGlobalShortcuts()
</script>

<template>
  <RouterView />
  <CommandPalette />
  <ToastHost />
  <ShortcutHelp />
</template>
