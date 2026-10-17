import { onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'

// Global navigation shortcuts. Ignored while typing in inputs.
export function useShortcuts(getRepoPath: () => string | null) {
  const router = useRouter()
  let pendingG = false
  let timer: number | undefined

  function isTyping(target: EventTarget | null): boolean {
    const el = target as HTMLElement | null
    if (!el) return false
    const tag = el.tagName?.toLowerCase()
    return tag === 'input' || tag === 'textarea' || tag === 'select' || el.isContentEditable
  }

  function onKeydown(e: KeyboardEvent) {
    if (isTyping(e.target) || e.metaKey || e.ctrlKey || e.altKey) return
    const repo = getRepoPath()

    if (pendingG) {
      pendingG = false
      if (timer) window.clearTimeout(timer)
      if (e.key === 'i' && repo) {
        e.preventDefault()
        router.push(`/${repo}?tab=${encodeURIComponent('工单')}`)
        return
      }
      if (e.key === 'p' && repo) {
        e.preventDefault()
        router.push(`/${repo}?tab=${encodeURIComponent('合并请求')}`)
        return
      }
    }

    if (e.key === 'g') {
      pendingG = true
      timer = window.setTimeout(() => (pendingG = false), 800)
      return
    }
    if (e.key === 'c' && repo) {
      e.preventDefault()
      router.push(`/${repo}?tab=${encodeURIComponent('工单')}`)
      return
    }
    if (e.key === '/') {
      const search = document.querySelector<HTMLInputElement>('input[type="search"]')
      if (search) {
        e.preventDefault()
        search.focus()
      }
    }
  }

  onMounted(() => window.addEventListener('keydown', onKeydown))
  onUnmounted(() => window.removeEventListener('keydown', onKeydown))
}
