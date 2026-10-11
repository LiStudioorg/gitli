import { ref, watch } from 'vue'

export type Theme = 'light' | 'dark' | 'auto'

const STORAGE_KEY = 'gitli-theme'
const media =
  typeof window !== 'undefined' && typeof window.matchMedia === 'function'
    ? window.matchMedia('(prefers-color-scheme: dark)')
    : null

function readStored(): Theme {
  if (typeof localStorage === 'undefined') return 'auto'
  const value = localStorage.getItem(STORAGE_KEY)
  return value === 'light' || value === 'dark' || value === 'auto' ? value : 'auto'
}

const theme = ref<Theme>(readStored())

function resolved(t: Theme): 'light' | 'dark' {
  if (t === 'auto') return media?.matches ? 'dark' : 'light'
  return t
}

function apply(t: Theme) {
  if (typeof document === 'undefined') return
  document.documentElement.setAttribute('data-theme', resolved(t))
}

apply(theme.value)

watch(theme, (t) => {
  apply(t)
  if (typeof localStorage !== 'undefined') localStorage.setItem(STORAGE_KEY, t)
})

media?.addEventListener('change', () => {
  if (theme.value === 'auto') apply('auto')
})

const order: Theme[] = ['light', 'dark', 'auto']

export function useTheme() {
  function setTheme(t: Theme) {
    theme.value = t
  }

  function cycleTheme() {
    const idx = order.indexOf(theme.value)
    theme.value = order[(idx + 1) % order.length]
  }

  return { theme, resolvedTheme: () => resolved(theme.value), setTheme, cycleTheme }
}
