<script setup lang="ts">
import { ref } from 'vue'
import type { CloneInfo } from '../../stores'

const props = defineProps<{
  owner: string
  repo: string
  cloneInfo: CloneInfo | null
}>()

const open = ref(false)
const tab = ref<'https' | 'ssh' | 'cli'>('https')
const copied = ref('')

function toggle() {
  open.value = !open.value
}

async function copy(text: string, key: string) {
  try {
    await navigator.clipboard.writeText(text)
    copied.value = key
    setTimeout(() => (copied.value = ''), 1500)
  } catch {
    copied.value = ''
  }
}

function download(format: 'zip' | 'tar.gz' | 'bundle') {
  const ref_ = props.cloneInfo?.clone_default_branch || 'main'
  window.location.href = `/api/v1/repos/${props.owner}/${props.repo}/archive/${format}?ref=${encodeURIComponent(ref_)}`
}
</script>

<template>
  <div class="relative inline-block">
    <button
      class="inline-flex h-8 items-center gap-1.5 rounded-md bg-[#2da44e] px-3 text-sm font-semibold text-white hover:bg-[#2c974b]"
      type="button"
      aria-label="获取克隆地址"
      :aria-expanded="open"
      @click="toggle"
    >
      <svg class="h-4 w-4" viewBox="0 0 16 16" fill="currentColor" aria-hidden="true">
        <path d="M4.72.22a.75.75 0 0 1 1.06 0l1 .999a.75.75 0 0 1 0 1.061l-1.001 1A.75.75 0 0 1 4.72 2.22L5.19 1.75 4.72 1.28a.75.75 0 0 1 0-1.06ZM7.75 0h1v2.5h-1V0Zm-3.5 5.75a.75.75 0 0 1 .75.75v1.5a.75.75 0 0 1-1.5 0v-1.5a.75.75 0 0 1 .75-.75Z" />
      </svg>
      代码
      <span class="text-[10px] opacity-80" aria-hidden="true">▾</span>
    </button>

    <div
      v-if="open"
      class="absolute right-0 top-full z-50 mt-1.5 w-[360px] max-w-[calc(100vw-2rem)] rounded-md border border-[#d0d7de] bg-white p-4 shadow-lg dark:border-[#30363d] dark:bg-[#161b22]"
      @mouseleave="open = false"
    >
      <p class="mb-2 text-xs font-semibold text-[#656d76]">克隆</p>
      <div class="mb-3 flex overflow-hidden rounded-md border border-[#d0d7de] dark:border-[#30363d]">
        <button
          v-for="t in [{ k: 'https', l: 'HTTPS' }, { k: 'ssh', l: 'SSH' }, { k: 'cli', l: 'CLI' }]"
          :key="t.k"
          type="button"
          class="flex-1 px-2 py-1.5 text-xs"
          :class="tab === t.k ? 'bg-[#ea580c]/10 font-semibold text-[#ea580c]' : 'text-[#656d76] hover:bg-[#f6f8fa] dark:hover:bg-[#21262d]'"
          @click="tab = t.k as 'https' | 'ssh' | 'cli'"
        >
          {{ t.l }}
        </button>
      </div>

      <div v-if="tab !== 'cli'" class="flex items-center gap-1.5">
        <code class="min-w-0 flex-1 truncate rounded-md border border-[#d0d7de] bg-[#f6f8fa] px-2 py-1.5 font-mono text-xs text-[#1f2328] dark:border-[#30363d] dark:bg-[#0d1117] dark:text-[#e6edf3]">
          {{ tab === 'https' ? (cloneInfo?.https ?? '—') : (cloneInfo?.ssh ?? '—') }}
        </code>
        <button
          type="button"
          class="inline-flex h-8 shrink-0 items-center rounded-md border border-[#d0d7de] px-2.5 text-xs text-[#1f2328] hover:bg-[#f6f8fa] dark:border-[#30363d] dark:text-[#e6edf3] dark:hover:bg-[#21262d]"
          @click="copy(tab === 'https' ? (cloneInfo?.https ?? '') : (cloneInfo?.ssh ?? ''), tab)"
        >
          {{ copied === tab ? '已复制' : '复制' }}
        </button>
      </div>
      <p v-else class="text-xs leading-relaxed text-[#656d76]">
        目前请使用上方 git clone/push 命令。gli 命令行工具尚未提供。
      </p>

      <p class="mb-2 mt-4 text-xs font-semibold text-[#656d76]">下载源码</p>
      <div class="flex flex-wrap gap-1.5">
        <button
          v-for="f in [{ k: 'zip', l: 'ZIP' }, { k: 'tar.gz', l: 'TAR.GZ' }, { k: 'bundle', l: 'BUNDLE' }]"
          :key="f.k"
          type="button"
          class="h-7 rounded-md border border-[#d0d7de] px-2.5 text-xs text-[#1f2328] hover:bg-[#f6f8fa] dark:border-[#30363d] dark:text-[#e6edf3] dark:hover:bg-[#21262d]"
          @click="download(f.k as 'zip' | 'tar.gz' | 'bundle')"
        >
          {{ f.l }}
        </button>
      </div>
    </div>
  </div>
</template>
