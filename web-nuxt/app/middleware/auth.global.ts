import api from '~/api/client'
import { useAuthStore } from '~/stores'

// 与 beta 分支 web/src/router/index.ts 的 beforeEach 逐条对应：
// 1) public 页面直接放行（如 /setup）；
// 2) 未初始化实例一律重定向到 /setup（查询 /version 失败则放行，避免后端抖动把用户锁死）；
// 3) 之后才做登录态与权限判定。
export default defineNuxtRouteMiddleware(async (to) => {
  if (to.meta.public) return
  try {
    const { data } = await api.get<{ initialized?: boolean }>('/version')
    if (data?.initialized === false) {
      return navigateTo('/setup')
    }
  } catch {
    return
  }
  const auth = useAuthStore()
  if (!auth.loaded) {
    await auth.fetchMe()
  }
  if (to.meta.requiresAuth && !auth.isLoggedIn) {
    return navigateTo({ path: '/login', query: { redirect: to.fullPath } })
  }
  if (to.meta.requiresAdmin && !auth.isAdmin) {
    return navigateTo('/dashboard')
  }
  if (to.meta.guestOnly && auth.isLoggedIn) {
    return navigateTo('/dashboard')
  }
})
