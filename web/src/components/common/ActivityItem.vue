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
  <article class="flex gap-3 rounded-md border border-[#d1d9e0] bg-white p-4 transition-colors hover:border-[#ea580c]/40 dark:border-[#3d444d] dark:bg-[#0d1117]">
    <span
      class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-[#ea580c]/10 text-sm text-[#ea580c]"
      aria-hidden="true"
    >
      {{ icon }}
    </span>
    <div class="min-w-0 flex-1">
      <p class="text-sm leading-relaxed text-[#1f2328] dark:text-[#e6edf3]">
        <strong class="font-semibold">{{ actor }}</strong>
        {{ label }}
        <RouterLink v-if="repoLink" class="font-semibold text-[#0969da] hover:underline" :to="repoLink">{{ repoName }}</RouterLink>
        <span v-else-if="repoName">{{ repoName }}</span>
      </p>
      <p
        v-if="activity.summary"
        class="mt-1 truncate rounded bg-[#f6f8fa] px-2 py-1 font-mono text-xs text-[#1f2328] dark:bg-[#161b22] dark:text-[#e6edf3]"
      >
        {{ activity.summary }}
      </p>
      <time class="mt-1 block text-xs text-[#59636e]">{{ time }}</time>
    </div>
  </article>
</template>
