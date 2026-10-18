import { beforeEach, describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import { i18n, setLocale } from '../app/i18n'
import { useTheme } from '../app/composables/useTheme'

const TopNav = (await import('../app/components/layout/TopNav.vue')).default
const { useAuthStore, useUiStore } = await import('../app/stores')

const Stub = { template: '<div />' }

function makeRouter() {
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', component: Stub },
      { path: '/login', component: Stub },
      { path: '/register', component: Stub },
      { path: '/dashboard', component: Stub },
      { path: '/explore', component: Stub },
      { path: '/search', component: Stub },
      { path: '/settings', component: Stub },
      { path: '/admin', component: Stub },
      { path: '/new/repo', component: Stub },
      { path: '/new/org', component: Stub },
      { path: '/new/import', component: Stub },
      // 兜底：TopNav 里的 RouterLink 覆盖了更多目标，缺路由会产生
      // VUE_ROUTER_R0004 噪音，掩盖真正的断言失败信号。
      { path: '/:pathMatch(.*)*', component: Stub },
    ],
  })
}

async function mountNav(variant: 'landing' | 'app' = 'app') {
  const router = makeRouter()
  await router.push('/dashboard')
  await router.isReady()
  return mount(TopNav, {
    props: { variant },
    global: { plugins: [router, i18n] },
  })
}

describe('TopNav（导航 / 主题 / 管理入口 / 命令面板入口）', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    setLocale('zh-CN')
    localStorage.clear()
  })

  it('未登录时显示登录/注册，不显示管理与退出', async () => {
    const w = await mountNav()
    const text = w.text()
    expect(text).toContain('登录')
    expect(text).toContain('注册')
    expect(text).not.toContain('管理')
    expect(text).not.toContain('退出')
    w.unmount()
  })

  it('已登录管理员在头像菜单里能看到「管理面板」', async () => {
    const auth = useAuthStore()
    auth.user = {
      id: 1,
      username: 'root',
      email: 'root@example.com',
      full_name: '',
      bio: '',
      avatar_url: '',
      website: '',
      location: '',
      is_admin: true,
      totp_enabled: false,
    }
    auth.loaded = true
    const w = await mountNav()
    // 管理入口收进了头像下拉菜单，平铺文本里不再直接出现
    expect(w.text()).not.toContain('管理面板')
    await w.find('[aria-label^="用户菜单"]').trigger('click')
    expect(w.text()).toContain('管理面板')
    w.unmount()
  })

  it('普通用户的头像菜单里没有「管理面板」', async () => {
    const auth = useAuthStore()
    auth.user = {
      id: 2,
      username: 'bob',
      email: '',
      full_name: '',
      bio: '',
      avatar_url: '',
      website: '',
      location: '',
      is_admin: false,
      totp_enabled: false,
    }
    auth.loaded = true
    const w = await mountNav()
    await w.find('[aria-label^="用户菜单"]').trigger('click')
    expect(w.text()).toContain('个人主页')
    expect(w.text()).not.toContain('管理面板')
    w.unmount()
  })

  it('点击主题按钮循环切换（auto -> light -> dark）', async () => {
    // useTheme 是模块级单例，先把状态钉到确定值，避免依赖前序用例残留。
    const { theme } = useTheme()
    theme.value = 'auto'
    const w = await mountNav()
    const btn = w.find('[title^="主题"]')
    expect(btn.exists()).toBe(true)
    await btn.trigger('click')
    expect(theme.value).toBe('light')
    // auto 与 light 在 happy-dom 下解析色都是 light，只有切到 dark 才改变 data-theme
    await btn.trigger('click')
    await w.vm.$nextTick()
    expect(theme.value).toBe('dark')
    expect(document.documentElement.getAttribute('data-theme')).toBe('dark')
    w.unmount()
  })

  it('点击 ⌘K 打开命令面板（写 ui store）', async () => {
    const w = await mountNav()
    const ui = useUiStore()
    expect(ui.commandPaletteOpen).toBe(false)
    await w.find('[aria-label="打开命令面板"]').trigger('click')
    expect(ui.commandPaletteOpen).toBe(true)
    w.unmount()
  })

  it('英文语言下导航文案来自 en-US 语言包', async () => {
    setLocale('en-US')
    const w = await mountNav('landing')
    const text = w.text()
    expect(text).toContain('Sign in')
    expect(text).toContain('Sign up')
    expect(text).toContain('Features')
    w.unmount()
  })
})
