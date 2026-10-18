// 与 web/src/router/index.ts 的 RouteMeta 声明合并保持一致；不声明的话
// definePageMeta 与 to.meta 上的自定义键都没有类型保护。
declare module '#app' {
  interface PageMeta {
    requiresAuth?: boolean
    requiresAdmin?: boolean
    guestOnly?: boolean
    /** 无需登录且跳过初始化检查（如 /setup 引导页）。 */
    public?: boolean
  }
}

declare module 'vue-router' {
  interface RouteMeta {
    requiresAuth?: boolean
    requiresAdmin?: boolean
    guestOnly?: boolean
    public?: boolean
  }
}

export {}
