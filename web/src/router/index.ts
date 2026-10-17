import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import api from '../api/client'
import { useAuthStore } from '../stores'

// RouteMeta 默认是 Record<string|symbol, unknown>，不声明的话 to.meta.requiresAdmin
// 这类键名拼错也不会有任何提示（unknown 允许出现在条件表达式里）。
declare module 'vue-router' {
  interface RouteMeta {
    requiresAuth?: boolean
    requiresAdmin?: boolean
    guestOnly?: boolean
    /** 无需登录且跳过初始化检查（如 /setup 引导页）。 */
    public?: boolean
  }
}

const routes: RouteRecordRaw[] = [
  { path: '/', name: 'landing', component: () => import('../views/LandingView.vue') },
  { path: '/setup', name: 'setup', component: () => import('../views/SetupView.vue'), meta: { public: true } },
  { path: '/login', name: 'login', component: () => import('../views/LoginView.vue'), meta: { guestOnly: true } },
  { path: '/register', name: 'register', component: () => import('../views/RegisterView.vue'), meta: { guestOnly: true } },
  { path: '/dashboard', name: 'dashboard', component: () => import('../views/DashboardView.vue'), meta: { requiresAuth: true } },
  { path: '/new/repo', name: 'new-repo', component: () => import('../views/NewRepoView.vue'), meta: { requiresAuth: true } },
  { path: '/new/org', name: 'new-org', component: () => import('../views/NewOrgView.vue'), meta: { requiresAuth: true } },
  { path: '/new/import', name: 'new-import', component: () => import('../views/ImportRepoView.vue'), meta: { requiresAuth: true } },
  { path: '/explore', name: 'explore', component: () => import('../views/ExploreView.vue'), meta: { requiresAuth: true } },
  { path: '/search', name: 'search', component: () => import('../views/SearchView.vue'), meta: { requiresAuth: true } },
  { path: '/settings', name: 'settings', component: () => import('../views/UserSettingsView.vue'), meta: { requiresAuth: true } },
  { path: '/orgs/:org/settings', name: 'org-settings', component: () => import('../views/OrgSettingsView.vue'), meta: { requiresAuth: true } },
  { path: '/admin', name: 'admin', component: () => import('../views/AdminView.vue'), meta: { requiresAuth: true, requiresAdmin: true } },
  { path: '/oauth/authorize', name: 'oauth-authorize', component: () => import('../views/OAuthAuthorizeView.vue') },
  { path: '/users/:username', name: 'user', component: () => import('../views/UserProfileView.vue') },
  { path: '/:owner/:repo', name: 'repo', component: () => import('../views/RepoView.vue') },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach(async (to) => {
  if (to.meta.public) return true
  try {
    const { data } = await api.get('/version')
    if (data?.initialized === false) {
      return { path: '/setup' }
    }
  } catch {
    return true
  }
  const auth = useAuthStore()
  if (!auth.loaded) {
    await auth.fetchMe()
  }
  if (to.meta.requiresAuth && !auth.isLoggedIn) {
    return { path: '/login', query: { redirect: to.fullPath } }
  }
  if (to.meta.requiresAdmin && !auth.isAdmin) {
    return { path: '/dashboard' }
  }
  if (to.meta.guestOnly && auth.isLoggedIn) {
    return { path: '/dashboard' }
  }
  return true
})

export default router
