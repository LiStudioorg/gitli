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
  return colorMap[lang] ?? palette[index % palette.length]
}

const segments = computed(() =>
  props.languages.map((l, i) => ({ ...l, color: colorFor(l.language, i) })),
)
</script>

<template>
  <div v-if="segments.length" class="flex flex-col gap-2">
    <div class="flex h-2 w-full overflow-hidden rounded-full bg-[#d1d9e0] dark:bg-[#3d444d]">
      <span
        v-for="seg in segments"
        :key="seg.language"
        class="h-full"
        :style="{ width: seg.percentage + '%', backgroundColor: seg.color }"
        :title="`${seg.language} ${seg.percentage.toFixed(1)}%`"
      />
    </div>
    <ul class="flex flex-wrap gap-x-4 gap-y-1 text-xs">
      <li v-for="seg in segments" :key="seg.language" class="flex items-center gap-1.5">
        <span class="h-2.5 w-2.5 rounded-full" :style="{ backgroundColor: seg.color }" aria-hidden="true" />
        <span class="font-semibold text-[#1f2328] dark:text-[#e6edf3]">{{ seg.language }}</span>
        <span class="text-[#59636e]">{{ seg.percentage.toFixed(1) }}%</span>
      </li>
    </ul>
  </div>
</template>
