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
  <div class="clone-dropdown">
    <button class="code-btn" @click="toggle">
      <span class="angle">&lt;&gt;</span> 代码
      <span class="caret">▾</span>
    </button>

    <div v-if="open" class="menu" @mouseleave="open = false">
      <div class="tabs">
        <button :class="{ active: tab === 'https' }" @click="tab = 'https'">HTTPS</button>
        <button :class="{ active: tab === 'ssh' }" @click="tab = 'ssh'">SSH</button>
        <button :class="{ active: tab === 'cli' }" @click="tab = 'cli'">CLI 工具</button>
      </div>

      <div class="section">
        <template v-if="tab === 'https'">
          <div class="url-row">
            <code>{{ cloneInfo?.https ?? '—' }}</code>
            <button class="copy" @click="copy(cloneInfo?.https ?? '', 'https')">
              {{ copied === 'https' ? '已复制' : '复制' }}
            </button>
          </div>
        </template>
        <template v-else-if="tab === 'ssh'">
          <div class="url-row">
            <code>{{ cloneInfo?.ssh ?? '—' }}</code>
            <button class="copy" @click="copy(cloneInfo?.ssh ?? '', 'ssh')">
              {{ copied === 'ssh' ? '已复制' : '复制' }}
            </button>
          </div>
        </template>
        <template v-else>
          <p class="placeholder-title">gli 命令行工具即将推出，敬请期待。</p>
          <p class="placeholder-sub">届时可用：</p>
          <pre class="placeholder-code">gli clone &lt;owner&gt;/&lt;repo&gt;
gli issue list
gli pr create</pre>
        </template>
      </div>

      <div class="section">
        <p class="label">下载</p>
        <div class="row">
          <button class="chip accent" @click="download('zip')">下载 ZIP</button>
          <button class="chip accent" @click="download('tar.gz')">下载 TAR.GZ</button>
          <button class="chip accent" @click="download('bundle')">下载 BUNDLE</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.clone-dropdown {
  position: relative;
  display: inline-block;
}
.code-btn {
  padding: 0.4rem 0.9rem;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--card);
  color: var(--color-fg);
  font-weight: 600;
}
.code-btn:hover {
  background: var(--hover-bg);
  border-color: var(--primary);
  color: var(--primary);
}
.angle {
  color: var(--primary);
}
.caret {
  margin-left: 0.3rem;
  color: var(--muted);
}
.menu {
  position: absolute;
  top: calc(100% + 6px);
  left: 0;
  z-index: 30;
  width: 420px;
  max-width: 90vw;
  padding: 0.9rem;
  background: var(--card);
  border: 1px solid var(--border);
  border-radius: 10px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.18);
}
.tabs {
  display: flex;
  gap: 0.25rem;
  margin-bottom: 0.75rem;
  border-bottom: 1px solid var(--border);
}
.tabs button {
  padding: 0.4rem 0.75rem;
  border: none;
  background: transparent;
  color: var(--muted);
  border-bottom: 2px solid transparent;
}
.tabs button.active {
  color: var(--primary);
  border-bottom-color: var(--primary);
  font-weight: 600;
}
.section {
  margin-bottom: 0.9rem;
}
.section:last-child {
  margin-bottom: 0;
}
.label {
  font-size: 0.78rem;
  text-transform: uppercase;
  color: var(--muted);
  margin: 0 0 0.4rem;
}
.url-row {
  display: flex;
  gap: 0.5rem;
  align-items: center;
}
.url-row code {
  flex: 1;
  padding: 0.45rem 0.6rem;
  background: var(--color-bg);
  border: 1px solid var(--border);
  border-radius: 6px;
  font-size: 0.82rem;
  overflow-x: auto;
  white-space: nowrap;
}
.copy {
  padding: 0.4rem 0.7rem;
  border: 1px solid var(--primary);
  border-radius: 6px;
  background: transparent;
  color: var(--primary);
  white-space: nowrap;
}
.copy:hover {
  background: var(--primary-soft);
}
.placeholder-title {
  color: var(--color-fg);
  font-size: 0.9rem;
  margin: 0 0 0.5rem;
}
.placeholder-sub {
  color: var(--muted);
  font-size: 0.85rem;
  margin: 0 0 0.4rem;
}
.placeholder-code {
  margin: 0;
  padding: 0.6rem 0.75rem;
  background: var(--color-bg);
  border: 1px solid var(--border);
  border-radius: 6px;
  font-size: 0.82rem;
  color: var(--primary);
  overflow-x: auto;
}
.row {
  display: flex;
  flex-wrap: wrap;
  gap: 0.4rem;
}
.chip {
  padding: 0.35rem 0.7rem;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: transparent;
  color: var(--color-fg);
  font-size: 0.85rem;
}
.chip:hover {
  background: var(--hover-bg);
  border-color: var(--primary);
  color: var(--primary);
}
.chip.accent {
  border-color: var(--accent);
  color: var(--accent);
}
.chip.accent:hover {
  background: var(--accent-soft);
}
</style>
