export type MiddlewareKey = never
declare module '@nuxt/schema' {
  interface RouteRuleConfigExtensions {
    appMiddleware?: MiddlewareKey | MiddlewareKey[] | Record<MiddlewareKey, boolean>
  }
}
declare module 'nuxt/schema' {
  interface RouteRuleConfigExtensions {
    appMiddleware?: MiddlewareKey | MiddlewareKey[] | Record<MiddlewareKey, boolean>
  }
}