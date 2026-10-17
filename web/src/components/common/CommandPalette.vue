<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore, useUiStore } from '../../stores'
import { useTheme } from '../../composables/useTheme'

interface Action {
  id: string
  label: string
  hint: string
  run: () => void | Promise<void>
}

const router = useRouter()
const auth = useAuthStore()
const ui = useUiStore()
const { cycleTheme } = useTheme()

const query = ref('')
const active = ref(0)
const inputEl = ref<HTMLInputElement | null>(null)

function close() {
  ui.commandPaletteOpen = false
  query.value = ''
  active.value = 0
}

const actions = computed<Action[]>(() => {
  const list: Action[] = [
    { id: 'dashboard', label: '前往控制台', hint: 'Dashboard', run: () => router.push('/dashboard') },
    { id: 'explore', label: '前往探索', hint: 'Explore', run: () => router.push('/explore') },
    { id: 'settings', label: '前往设置', hint: 'Settings', run: () => router.push('/settings') },
    { id: 'new-repo', label: '新建仓库', hint: 'New Repo', run: () => router.push('/new/repo') },
    { id: 'new-org', label: '新建组织', hint: 'New Org', run: () => router.push('/new/org') },
    { id: 'new-import', label: '导入仓库', hint: 'Import Repo', run: () => router.push('/new/import') },
    { id: 'theme', label: '切换主题', hint: 'Toggle theme', run: () => cycleTheme() },
  ]
  if (auth.isLoggedIn) {
    list.push({
      id: 'logout',
      label: '退出登录',
      hint: 'Logout',
      run: async () => {
        await auth.logout()
        router.push('/')
      },
    })
  }
  return list
})

const filtered = computed(() => {
  const q = query.value.trim().toLowerCase()
  if (!q) return actions.value
  return actions.value.filter(
    (a) => a.label.toLowerCase().includes(q) || a.hint.toLowerCase().includes(q),
  )
})

watch(filtered, () => {
  active.value = 0
})

watch(
  () => ui.commandPaletteOpen,
  async (open) => {
    if (open) {
      query.value = ''
      active.value = 0
      await nextTick()
      inputEl.value?.focus()
    }
  },
)

function move(delta: number) {
  if (filtered.value.length === 0) return
  active.value = (active.value + delta + filtered.value.length) % filtered.value.length
}

async function execute(index = active.value) {
  const action = filtered.value[index]
  if (!action) return
  close()
  await action.run()
}

function onKeydown(e: KeyboardEvent) {
  if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'k') {
    e.preventDefault()
    ui.commandPaletteOpen = !ui.commandPaletteOpen
  }
}

function onDialogKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    e.preventDefault()
    close()
  } else if (e.key === 'ArrowDown') {
    e.preventDefault()
    move(1)
  } else if (e.key === 'ArrowUp') {
    e.preventDefault()
    move(-1)
  } else if (e.key === 'Enter') {
    e.preventDefault()
    void execute()
  }
}

onMounted(() => window.addEventListener('keydown', onKeydown))
onUnmounted(() => window.removeEventListener('keydown', onKeydown))
</script>

<template>
  <div
    v-if="ui.commandPaletteOpen"
    class="overlay"
    role="dialog"
    aria-modal="true"
    aria-label="命令面板"
    @click.self="close"
    @keydown="onDialogKeydown"
  >
    <div class="panel">
      <input
        ref="inputEl"
        v-model="query"
        class="palette-input"
        type="text"
        placeholder="输入命令..."
        aria-label="命令搜索"
        @keydown="onDialogKeydown"
      />
      <ul class="results" role="listbox" aria-label="命令列表">
        <li
          v-for="(a, i) in filtered"
          :key="a.id"
          role="option"
          :aria-selected="i === active"
          class="result"
          :class="{ active: i === active }"
          @mouseenter="active = i"
          @click="execute(i)"
        >
          <span class="label">{{ a.label }}</span>
          <span class="hint">{{ a.hint }}</span>
        </li>
        <li v-if="filtered.length === 0" class="result empty">无匹配命令</li>
      </ul>
    </div>
  </div>
</template>

<style scoped>
.overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.45);
  display: flex;
  align-items: flex-start;
  justify-content: center;
  padding-top: 12vh;
  z-index: 1000;
}
.panel {
  width: min(560px, 92vw);
  background: var(--card);
  border: 1px solid var(--border);
  border-radius: 8px;
  overflow: hidden;
  box-shadow: 0 12px 40px rgba(0, 0, 0, 0.3);
}
.palette-input {
  width: 100%;
  padding: 0.85rem 1rem;
  border: none;
  border-bottom: 1px solid var(--border);
  background: transparent;
  color: var(--color-fg);
  font-size: 1rem;
  outline: none;
}
.results {
  list-style: none;
  margin: 0;
  padding: 0.35rem;
  max-height: 320px;
  overflow-y: auto;
}
.result {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.55rem 0.75rem;
  border-radius: 6px;
  cursor: pointer;
  color: var(--color-fg);
}
.result.active {
  background: var(--primary-soft);
  color: var(--primary);
}
.result .hint {
  color: var(--muted);
  font-size: 0.8rem;
}
.result.empty {
  color: var(--muted);
  cursor: default;
  justify-content: center;
}
</style>
