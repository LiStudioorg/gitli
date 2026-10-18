<script setup lang="ts">
import { shortcutHelpOpen, shortcutList } from '../../composables/useShortcuts'

const shortcuts = shortcutList()

function close() {
  shortcutHelpOpen.value = false
}
</script>

<template>
  <div
    v-if="shortcutHelpOpen"
    class="overlay"
    role="dialog"
    aria-modal="true"
    aria-labelledby="shortcut-title"
    @click.self="close"
  >
    <section class="panel">
      <header class="mb-3 flex items-center justify-between gap-2">
        <h2 id="shortcut-title">键盘快捷键</h2>
        <button type="button" class="close" aria-label="关闭快捷键说明" @click="close">×</button>
      </header>
      <ul class="flex flex-col gap-2">
        <li v-for="s in shortcuts" :key="s.label">
          <span class="keys">
            <template v-for="(k, i) in s.keys" :key="k">
              <kbd>{{ k }}</kbd>
              <span v-if="i < s.keys.length - 1" class="then">然后</span>
            </template>
          </span>
          <span class="label">{{ s.label }}</span>
        </li>
      </ul>
      <p class="hint">在输入框中输入时快捷键不生效。按 <kbd>Esc</kbd> 关闭本窗口。</p>
    </section>
  </div>
</template>
