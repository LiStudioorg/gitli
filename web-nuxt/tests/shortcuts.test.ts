import { describe, expect, it, vi } from 'vitest'
import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'

// useShortcuts 依赖 vue-router 的 useRouter；这里用最小 mock 截获 push。
const pushed: string[] = []
vi.mock('vue-router', () => ({
  useRouter: () => ({ push: (to: string) => pushed.push(to) }),
}))

const { useShortcuts } = await import('../app/composables/useShortcuts')

function mountShortcuts(repo: string | null) {
  const Comp = defineComponent({
    setup() {
      useShortcuts(() => repo)
      return () => h('div')
    },
  })
  return mount(Comp, { attachTo: document.body })
}

function press(key: string, target?: EventTarget) {
  const ev = new KeyboardEvent('keydown', { key, bubbles: true })
  ;(target ?? document.body).dispatchEvent(ev)
  return ev
}

describe('useShortcuts（与原前端行为一致）', () => {
  it('g 然后 i 跳到工单 Tab，g 然后 p 跳到合并请求 Tab', async () => {
    pushed.length = 0
    const w = mountShortcuts('alice/demo')
    press('g')
    press('i')
    expect(pushed[0]).toBe(`/alice/demo?tab=${encodeURIComponent('工单')}`)
    press('g')
    press('p')
    expect(pushed[1]).toBe(`/alice/demo?tab=${encodeURIComponent('合并请求')}`)
    w.unmount()
  })

  it('无仓库上下文时 g i 不跳转，c 也不跳转', () => {
    pushed.length = 0
    const w = mountShortcuts(null)
    press('g')
    press('i')
    press('c')
    expect(pushed).toHaveLength(0)
    w.unmount()
  })

  it('在输入框内打字不触发快捷键', () => {
    pushed.length = 0
    const w = mountShortcuts('alice/demo')
    const input = document.createElement('input')
    document.body.appendChild(input)
    press('g', input)
    press('i', input)
    expect(pushed).toHaveLength(0)
    input.remove()
    w.unmount()
  })

  it('/ 聚焦搜索框', () => {
    const w = mountShortcuts('alice/demo')
    const search = document.createElement('input')
    search.type = 'search'
    document.body.appendChild(search)
    press('/')
    expect(document.activeElement).toBe(search)
    search.remove()
    w.unmount()
  })
})
