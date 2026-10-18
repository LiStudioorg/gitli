<script setup lang="ts">
import { computed } from 'vue'
import type { Activity } from '../../stores'
import { useAuthStore } from '../../stores'

const props = defineProps<{ activity: Activity }>()
const auth = useAuthStore()

const icons: Record<string, string> = {
  pushed: '⎇',
  published_tag: '🏷',
  published_release: '🚀',
  opened_issue: '◎',
  closed_issue: '✔',
  opened_pull: '⇄',
  merged_pull: '⤵',
  closed_pull: '✕',
  started_discussion: '💬',
  created_repo: '📦',
}

const labels: Record<string, string> = {
  pushed: '推送代码到',
  published_tag: '发布了标签',
  published_release: '发布了版本',
  opened_issue: '创建了工单',
  closed_issue: '关闭了工单',
  opened_pull: '创建了合并请求',
  merged_pull: '合并了合并请求',
  closed_pull: '关闭了合并请求',
  started_discussion: '发起了讨论',
  created_repo: '创建了仓库',
}

const icon = computed(() => icons[props.activity.verb] ?? '•')
const label = computed(() => labels[props.activity.verb] ?? props.activity.verb)
const actor = computed(() => props.activity.actor_name || auth.user?.username || 'someone')
const repoName = computed(() => props.activity.repo_name || '')
const repoLink = computed(() => (repoName.value ? `/${repoName.value}` : ''))
const time = computed(() => relativeTime(props.activity.created_at))

function relativeTime(raw: string): string {
  const then = new Date(raw.endsWith('Z') || raw.includes('+') ? raw : raw.replace(' ', 'T') + 'Z').getTime()
  if (Number.isNaN(then)) return raw
  const diff = Math.max(0, Date.now() - then)
  const min = Math.floor(diff / 60000)
  if (min < 1) return '刚刚'
  if (min < 60) return `${min} 分钟前`
  const hours = Math.floor(min / 60)
  if (hours < 24) return `${hours} 小时前`
  const days = Math.floor(hours / 24)
  if (days < 30) return `${days} 天前`
  const months = Math.floor(days / 30)
  if (months < 12) return `${months} 个月前`
  return `${Math.floor(months / 12)} 年前`
}
</script>

<template>
  <article class="activity-item">
    <span class="icon" :class="`verb-${activity.verb}`" aria-hidden="true">{{ icon }}</span>
    <div class="body">
      <p class="line">
        <strong>{{ actor }}</strong>
        {{ label }}
        <RouterLink v-if="repoLink" class="repo" :to="repoLink">{{ repoName }}</RouterLink>
        <span v-else-if="repoName">{{ repoName }}</span>
      </p>
      <p v-if="activity.summary" class="summary">{{ activity.summary }}</p>
      <time class="time">{{ time }}</time>
    </div>
  </article>
</template>

<style scoped>
.activity-item {
  display: flex;
  gap: 0.85rem;
  padding: 0.9rem 1rem;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--card);
  transition: border-color 0.15s;
}
.activity-item:hover {
  border-color: var(--primary);
}
.icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 34px;
  height: 34px;
  border-radius: 50%;
  background: color-mix(in srgb, var(--primary) 14%, transparent);
  color: var(--primary);
  font-size: 1rem;
  flex-shrink: 0;
}
.body {
  flex: 1;
  min-width: 0;
}
.line {
  margin: 0;
  font-size: 0.93rem;
  line-height: 1.45;
}
.repo {
  color: var(--primary);
  font-weight: 600;
  text-decoration: none;
}
.repo:hover {
  text-decoration: underline;
}
.summary {
  margin: 0.3rem 0 0;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 0.82rem;
  color: var(--fg);
  background: color-mix(in srgb, var(--border) 35%, transparent);
  border-radius: 6px;
  padding: 0.25rem 0.5rem;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.time {
  color: var(--muted);
  font-size: 0.78rem;
}
</style>
