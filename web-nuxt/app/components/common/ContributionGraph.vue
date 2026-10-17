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
  <section class="card">
    <header class="head">
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

<style scoped>
.card {
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: 1rem;
  background: var(--card);
}
.head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  margin-bottom: 0.75rem;
}
h3 {
  margin: 0;
  font-size: 0.95rem;
}
.range {
  font-size: 0.75rem;
  color: var(--muted);
}
.grid {
  display: grid;
  grid-auto-flow: column;
  grid-template-rows: repeat(7, 12px);
  grid-auto-columns: 12px;
  gap: 3px;
  overflow-x: auto;
  padding-bottom: 0.25rem;
}
.cell {
  width: 12px;
  height: 12px;
  border-radius: 3px;
  background: color-mix(in srgb, var(--border) 70%, transparent);
  display: inline-block;
}
.level-1 {
  background: color-mix(in srgb, var(--primary) 30%, transparent);
}
.level-2 {
  background: color-mix(in srgb, var(--primary) 55%, transparent);
}
.level-3 {
  background: color-mix(in srgb, var(--primary) 78%, transparent);
}
.level-4 {
  background: var(--primary);
}
.foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 0.6rem;
  font-size: 0.75rem;
  color: var(--muted);
  gap: 0.5rem;
  flex-wrap: wrap;
}
.legend {
  display: inline-flex;
  align-items: center;
  gap: 3px;
}
</style>
