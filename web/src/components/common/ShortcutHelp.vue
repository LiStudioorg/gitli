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
      <header class="head">
        <h2 id="shortcut-title">键盘快捷键</h2>
        <button type="button" class="close" aria-label="关闭快捷键说明" @click="close">×</button>
      </header>
      <ul class="list">
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

<style scoped>
.overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.45);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 300;
  padding: 1rem;
}
.panel {
  width: min(520px, 100%);
  background: var(--card);
  border: 1px solid var(--border);
  border-radius: 12px;
  box-shadow: 0 20px 50px rgba(0, 0, 0, 0.3);
  padding: 1.25rem 1.4rem;
}
.head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 0.9rem;
}
h2 {
  margin: 0;
  font-size: 1.05rem;
}
.close {
  border: none;
  background: transparent;
  font-size: 1.4rem;
  line-height: 1;
  cursor: pointer;
  color: var(--muted);
  padding: 0 0.25rem;
  border-radius: 4px;
}
.close:hover {
  background: var(--hover-bg, rgba(0, 0, 0, 0.06));
}
.list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 0.55rem;
}
.list li {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  font-size: 0.9rem;
}
.keys {
  display: inline-flex;
  align-items: center;
  gap: 0.3rem;
  flex-shrink: 0;
}
kbd {
  display: inline-block;
  min-width: 1.6rem;
  text-align: center;
  padding: 0.15rem 0.4rem;
  border: 1px solid var(--border);
  border-bottom-width: 2px;
  border-radius: 5px;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 0.78rem;
  background: color-mix(in srgb, var(--border) 30%, transparent);
}
.then {
  font-size: 0.72rem;
  color: var(--muted);
}
.label {
  color: var(--muted);
  text-align: right;
}
.hint {
  margin: 1rem 0 0;
  font-size: 0.78rem;
  color: var(--muted);
}
</style>
