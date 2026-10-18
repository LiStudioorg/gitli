<script setup lang="ts">
import { computed } from 'vue'
import type { LanguageStat } from '../../stores'

const props = defineProps<{ languages: LanguageStat[] }>()

const colorMap: Record<string, string> = {
  Go: '#3b82f6',
  Shell: '#10b981',
  Makefile: '#f59e0b',
  Dockerfile: '#0ea5e9',
  Vue: '#42b883',
  TypeScript: '#3178c6',
  JavaScript: '#f1e05a',
  Python: '#3572a5',
  HTML: '#e34c26',
  CSS: '#563d7c',
  Markdown: '#083fa1',
  C: '#555555',
  'C++': '#f34b7d',
  Rust: '#dea584',
}

const palette = ['#8b5cf6', '#ec4899', '#14b8a6', '#ef4444', '#a855f7', '#22c55e', '#eab308', '#06b6d4']

function colorFor(lang: string, index: number): string {
  return colorMap[lang] ?? palette[index % palette.length] ?? 'var(--text-secondary)'
}

const segments = computed(() =>
  props.languages.map((l, i) => ({ ...l, color: colorFor(l.language, i) })),
)
</script>

<template>
  <div class="language-stats">
    <div class="bar">
      <span
        v-for="s in segments"
        :key="s.language"
        class="seg"
        :style="{ width: s.percentage + '%', background: s.color }"
        :title="`${s.language} ${s.percentage}%`"
      />
    </div>
    <ul class="list">
      <li v-for="s in segments" :key="s.language">
        <span class="dot" :style="{ background: s.color }" />
        <span class="name">{{ s.language }}</span>
        <span class="pct">{{ s.percentage }}%</span>
      </li>
    </ul>
    <p v-if="segments.length === 0" class="empty">暂无语言统计</p>
  </div>
</template>

<style scoped>
.language-stats {
  display: flex;
  flex-direction: column;
  gap: 0.6rem;
}
.bar {
  display: flex;
  height: 10px;
  border-radius: 999px;
  overflow: hidden;
  background: var(--border);
}
.seg {
  height: 100%;
  transition: width 0.5s ease;
}
.list {
  list-style: none;
  padding: 0;
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  font-size: 0.88rem;
}
.list li {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}
.dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  flex-shrink: 0;
}
.name {
  flex: 1;
}
.pct {
  color: var(--muted);
}
.empty {
  color: var(--muted);
  font-size: 0.9rem;
}
</style>
