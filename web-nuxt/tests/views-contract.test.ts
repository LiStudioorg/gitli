import { describe, expect, it } from 'vitest'

// 视图「能否在 Nuxt 目录结构下被正确导入」是迁移里最容易被静默破坏的一环：
// 相对路径深度、组件名、组合式函数引用都会在构建期才暴露。这里做静态契约校验，
// 不做 DOM 挂载 —— 挂载期的数据形状由「Nuxt dev + 真实 gitli」端到端冒烟负责，
// 避免用万能 mock 制造绿色假象。
const views: Array<{ name: string; path: string; load: () => Promise<{ default: unknown }> }> = [
  { name: 'LandingView', path: '/', load: () => import('../app/views/LandingView.vue') },
  { name: 'LoginView', path: '/login', load: () => import('../app/views/LoginView.vue') },
  { name: 'RegisterView', path: '/register', load: () => import('../app/views/RegisterView.vue') },
  { name: 'DashboardView', path: '/dashboard', load: () => import('../app/views/DashboardView.vue') },
  { name: 'ExploreView', path: '/explore', load: () => import('../app/views/ExploreView.vue') },
  { name: 'SearchView', path: '/search', load: () => import('../app/views/SearchView.vue') },
  { name: 'UserSettingsView', path: '/settings', load: () => import('../app/views/UserSettingsView.vue') },
  { name: 'OrgSettingsView', path: '/orgs/:org/settings', load: () => import('../app/views/OrgSettingsView.vue') },
  { name: 'AdminView', path: '/admin', load: () => import('../app/views/AdminView.vue') },
  { name: 'OAuthAuthorizeView', path: '/oauth/authorize', load: () => import('../app/views/OAuthAuthorizeView.vue') },
  { name: 'UserProfileView', path: '/users/:username', load: () => import('../app/views/UserProfileView.vue') },
  { name: 'RepoView', path: '/:owner/:repo', load: () => import('../app/views/RepoView.vue') },
]

const pages: Array<{ path: string; load: () => Promise<{ default: unknown }> }> = [
  { path: '/', load: () => import('../app/pages/index.vue') },
  { path: '/login', load: () => import('../app/pages/login.vue') },
  { path: '/register', load: () => import('../app/pages/register.vue') },
  { path: '/dashboard', load: () => import('../app/pages/dashboard.vue') },
  { path: '/explore', load: () => import('../app/pages/explore.vue') },
  { path: '/search', load: () => import('../app/pages/search.vue') },
  { path: '/settings', load: () => import('../app/pages/settings.vue') },
  { path: '/admin', load: () => import('../app/pages/admin.vue') },
  { path: '/orgs/:org/settings', load: () => import('../app/pages/orgs/[org]/settings.vue') },
  { path: '/oauth/authorize', load: () => import('../app/pages/oauth/authorize.vue') },
  { path: '/users/:username', load: () => import('../app/pages/users/[username].vue') },
  { path: '/:owner/:repo', load: () => import('../app/pages/[owner]/[repo].vue') },
]

describe('视图导入契约', () => {
  for (const v of views) {
    it(`${v.name} 可被导入且组件对象有效`, async () => {
      const mod = await v.load()
      const comp = mod.default as { setup?: unknown; render?: unknown }
      expect(comp).toBeTruthy()
      // <script setup> 编译后一定有 setup 或 render 之一
      expect(typeof comp.setup === 'function' || typeof comp.render === 'function').toBe(true)
    })
  }
})

describe('页面壳与路由契约', () => {
  it('12 条路由的页面壳与旧 router 表一一对应', () => {
    expect(pages.map((p) => p.path).sort()).toEqual(views.map((v) => v.path).sort())
  })

  it('每个页面壳都能被导入', async () => {
    for (const p of pages) {
      const mod = await p.load()
      expect(mod.default, `page for ${p.path}`).toBeTruthy()
    }
  })
})
