import { onMounted, onUnmounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { TABS } from '../constants/tabs'

/**
 * 全局快捷键状态。帮助弹窗的开关放在模块级，便于任意组件读取/触发。
 */
export const shortcutHelpOpen = ref(false)

const SHORTCUTS = [
  { keys: ['/'], label: '聚焦全局搜索框' },
  { keys: ['?'], label: '显示本快捷键说明' },
  { keys: ['g', 'i'], label: '跳转到工单（仓库页）' },
  { keys: ['g', 'p'], label: '跳转到合并请求（仓库页）' },
  { keys: ['g', 'd'], label: '跳转到控制台' },
  { keys: ['g', 'e'], label: '跳转到探索' },
  { keys: ['c'], label: '新建工单（仓库页）' },
  { keys: ['Esc'], label: '关闭弹窗 / 取消搜索' },
] as const

export function shortcutList() {
  return SHORTCUTS
}

function isTyping(target: EventTarget | null): boolean {
  const el = target as HTMLElement | null
  if (!el) return false
  const tag = el.tagName?.toLowerCase()
  return tag === 'input' || tag === 'textarea' || tag === 'select' || el.isContentEditable === true
}

/**
 * 全局快捷键监听。挂载在 App 上即可作用于所有页面；仓库相关快捷键只在
 * 仓库页生效，其余导航快捷键在任何页面都可用。
 */
export function useGlobalShortcuts() {
  const router = useRouter()
  const route = useRoute()
  let pendingG = false
  let timer: number | undefined

  function currentRepo(): string | null {
    if (route.name !== 'repo') return null
    const owner = route.params.owner
    const repo = route.params.repo
    return typeof owner === 'string' && typeof repo === 'string' ? `${owner}/${repo}` : null
  }

  function focusSearch() {
    const candidates = Array.from(
      document.querySelectorAll<HTMLInputElement>('input[type="search"], input[data-shortcut="search"]'),
    ).filter((el) => !el.disabled && el.offsetParent !== null)
    const target = candidates[0]
    if (target) {
      target.focus()
      target.select?.()
    }
  }

  function onKeydown(e: KeyboardEvent) {
    if (e.metaKey || e.ctrlKey || e.altKey) return
    if (e.key === 'Escape') {
      shortcutHelpOpen.value = false
      return
    }
    if (isTyping(e.target)) return

    const repo = currentRepo()

    if (e.key === '?') {
      e.preventDefault()
      shortcutHelpOpen.value = !shortcutHelpOpen.value
      return
    }
    if (e.key === '/') {
      e.preventDefault()
      focusSearch()
      return
    }

    if (pendingG) {
      pendingG = false
      if (timer) window.clearTimeout(timer)
      if (e.key === 'i' && repo) {
        e.preventDefault()
        router.push(`/${repo}?tab=${encodeURIComponent(TABS.issues)}`)
        return
      }
      if (e.key === 'p' && repo) {
        e.preventDefault()
        router.push(`/${repo}?tab=${encodeURIComponent(TABS.pulls)}`)
        return
      }
      if (e.key === 'd') {
        e.preventDefault()
        router.push('/dashboard')
        return
      }
      if (e.key === 'e') {
        e.preventDefault()
        router.push('/explore')
        return
      }
      return
    }

    if (e.key === 'g') {
      pendingG = true
      timer = window.setTimeout(() => (pendingG = false), 900)
      return
    }
    if (e.key === 'c' && repo) {
      e.preventDefault()
      router.push(`/${repo}?tab=${encodeURIComponent(TABS.issues)}`)
    }
  }

  onMounted(() => window.addEventListener('keydown', onKeydown))
  onUnmounted(() => window.removeEventListener('keydown', onKeydown))
}

/**
 * 兼容旧调用点：仓库页传入仓库路径即可，实际监听已提升为全局。
 */
export function useShortcuts(_getRepoPath: () => string | null) {
  // 全局监听已在 App 挂载，这里保持空实现以免重复绑定。
}
