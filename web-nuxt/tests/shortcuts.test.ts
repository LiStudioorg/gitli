import { beforeEach, describe, expect, it } from 'vitest'
import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import { createRouter, createMemoryHistory } from 'vue-router'

// main 版把快捷键收敛到 useGlobalShortcuts()：无参数，靠当前路由 name === 'repo'
// 判断是否在仓库页，并用真实 router.push 跳转。因此这里用真实 memory router，
// 通过 currentRoute 断言，而不是 mock push。
const { useGlobalShortcuts, shortcutHelpOpen } = await import('../app/composables/useShortcuts')

const Stub = { template: '<div />' }

function makeRouter() {
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/dashboard', name: 'dashboard', component: Stub },
      { path: '/explore', name: 'explore', component: Stub },
      { path: '/:owner/:repo', name: 'repo', component: Stub },
    ],
  })
}

async function mountOn(path: string) {
  const router = makeRouter()
  await router.push(path)
  await router.isReady()
  const Comp = defineComponent({ setup: () => (useGlobalShortcuts(), () => h('div')) })
  const w = mount(Comp, { global: { plugins: [router] }, attachTo: document.body })
  return { w, router }
}

function press(key: string, init?: Partial<KeyboardEventInit>) {
  const ev = new KeyboardEvent('keydown', { key, bubbles: true, cancelable: true, ...init })
  window.dispatchEvent(ev)
  return ev
}

// vue-router 的 push 是异步的，dispatch 后要等一个宏任务让导航落地。
const settle = () => new Promise((r) => setTimeout(r, 0))

describe('useGlobalShortcuts（与 main 版行为一致）', () => {
  beforeEach(() => {
    shortcutHelpOpen.value = false
  })

  it('仓库页 g i 跳工单、g p 跳合并请求', async () => {
    const { w, router } = await mountOn('/alice/demo')
    press('g')
    press('i')
    await settle()
    expect(router.currentRoute.value.fullPath).toBe(`/alice/demo?tab=${encodeURIComponent('工单')}`)
    press('g')
    press('p')
    await settle()
    expect(router.currentRoute.value.fullPath).toBe(`/alice/demo?tab=${encodeURIComponent('合并请求')}`)
    w.unmount()
  })

  it('非仓库页 g i / c 不跳转，但 g d 仍回控制台', async () => {
    const { w, router } = await mountOn('/explore')
    press('g')
    press('i')
    await settle()
    expect(router.currentRoute.value.name).toBe('explore')
    press('c')
    await settle()
    expect(router.currentRoute.value.name).toBe('explore')
    press('g')
    press('d')
    await settle()
    expect(router.currentRoute.value.fullPath).toBe('/dashboard')
    w.unmount()
  })

  it('输入框内打字不触发快捷键', async () => {
    const { w, router } = await mountOn('/alice/demo')
    const input = document.createElement('input')
    document.body.appendChild(input)
    input.focus()
    press('g', undefined)
    const ev = new KeyboardEvent('keydown', { key: 'i', bubbles: true, cancelable: true })
    input.dispatchEvent(ev)
    expect(router.currentRoute.value.fullPath).toBe('/alice/demo')
    input.remove()
    w.unmount()
  })

  it('? 切换快捷键帮助面板', async () => {
    const { w } = await mountOn('/dashboard')
    expect(shortcutHelpOpen.value).toBe(false)
    press('?')
    expect(shortcutHelpOpen.value).toBe(true)
    press('?')
    expect(shortcutHelpOpen.value).toBe(false)
    w.unmount()
  })

  it('带 ctrl/alt/meta 的组合键不触发裸键快捷键', async () => {
    const { w, router } = await mountOn('/alice/demo')
    press('g', { ctrlKey: true })
    press('i')
    expect(router.currentRoute.value.fullPath).toBe('/alice/demo')
    w.unmount()
  })
})
