<script setup lang="ts">
import { computed } from 'vue'
import type { Activity } from '../../stores'

const props = defineProps<{ activities: Activity[] }>()

const DAYS = 91

interface Cell {
  date: string
  level: number
}

const cells = computed<Cell[]>(() => {
  const counts = new Map<string, number>()
  for (const a of props.activities) {
    const key = dayKey(a.created_at)
    if (key) counts.set(key, (counts.get(key) ?? 0) + 1)
  }
  const out: Cell[] = []
  const today = new Date()
  today.setHours(0, 0, 0, 0)
  for (let i = DAYS - 1; i >= 0; i--) {
    const d = new Date(today.getTime() - i * 86400000)
    const key = localKey(d)
    const n = counts.get(key) ?? 0
    out.push({ date: key, level: n === 0 ? 0 : n <= 1 ? 1 : n <= 3 ? 2 : n <= 6 ? 3 : 4 })
  }
  return out
})

const total = computed(() => cells.value.reduce((sum, c) => sum + (c.level > 0 ? 1 : 0), 0))

function localKey(d: Date): string {
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
}

function dayKey(raw: string): string {
  const d = new Date(raw.endsWith('Z') || raw.includes('+') ? raw : raw.replace(' ', 'T') + 'Z')
  if (Number.isNaN(d.getTime())) return ''
  return localKey(d)
}
</script>

<template>
  <section class="rounded-md border border-[#d1d9e0] bg-white p-4 dark:border-[#3d444d] dark:bg-[#0d1117]">
    <header class="mb-3 flex items-center justify-between gap-2">
      <h3>贡献热力图</h3>
      <span class="range">最近 3 个月</span>
    </header>
    <div class="grid" role="img" aria-label="贡献热力图">
      <span
        v-for="(c, i) in cells"
        :key="i"
        class="cell"
        :class="`level-${c.level}`"
        :title="`${c.date}：${c.level} 次活动`"
      />
    </div>
    <footer class="foot">
      <span>最近 3 个月 {{ total }} 天有活动</span>
      <span class="legend">
        少
        <i class="cell level-0" />
        <i class="cell level-1" />
        <i class="cell level-2" />
        <i class="cell level-3" />
        <i class="cell level-4" />
        多
      </span>
    </footer>
  </section>
</template>
