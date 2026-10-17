import { describe, expect, it, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'

const pushed: string[] = []
vi.mock('vue-router', () => ({
  useRouter: () => ({ push: (to: string) => pushed.push(to) }),
  RouterLink: { template: '<a><slot /></a>' },
}))

const { default: CommandPalette } = await import('../app/components/common/CommandPalette.vue')
const { useUiStore } = await import('../app/stores')

function ctrlK() {
  const ev = new KeyboardEvent('keydown', { key: 'k', ctrlKey: true, bubbles: true, cancelable: true })
  window.dispatchEvent(ev)
  return ev
}

describe('CommandPalette（Ctrl+K 命令面板）', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('Ctrl+K 切换 ui.commandPaletteOpen，并阻止浏览器默认行为', async () => {
    const w = mount(CommandPalette, { attachTo: document.body })
    const ui = useUiStore()
    expect(ui.commandPaletteOpen).toBe(false)
    const ev = ctrlK()
    expect(ui.commandPaletteOpen).toBe(true)
    expect(ev.defaultPrevented).toBe(true)
    ctrlK()
    expect(ui.commandPaletteOpen).toBe(false)
    w.unmount()
  })

  it('面板打开时渲染动作列表，未登录不出现退出登录项', async () => {
    const w = mount(CommandPalette, { attachTo: document.body })
    const ui = useUiStore()
    ui.commandPaletteOpen = true
    await w.vm.$nextTick()
    const text = w.text()
    expect(text).toContain('前往控制台')
    expect(text).toContain('切换主题')
    expect(text).not.toContain('退出登录')
    w.unmount()
  })

  it('Escape 关闭面板', async () => {
    const w = mount(CommandPalette, { attachTo: document.body })
    const ui = useUiStore()
    ui.commandPaletteOpen = true
    await w.vm.$nextTick()
    const dialog = w.find('[role="dialog"]')
    expect(dialog.exists()).toBe(true)
    dialog.trigger('keydown', { key: 'Escape' })
    await w.vm.$nextTick()
    expect(ui.commandPaletteOpen).toBe(false)
    w.unmount()
  })
})
