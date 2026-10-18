
import type { InternalApi } from 'nitropack/types'

declare module '@nuxt/schema' {
  // eslint-disable-next-line @typescript-eslint/no-empty-object-type
  interface ServerRoutes extends InternalApi {}
}
declare module 'nuxt/schema' {
  // eslint-disable-next-line @typescript-eslint/no-empty-object-type
  interface ServerRoutes extends InternalApi {}
}
