
import type { CustomAppConfig } from 'nuxt/schema'

declare const inlineConfig = {
  "nuxt": {}
}
type IsAny<T> = 0 extends 1 & T ? true : false

type IsMergeable<T> = T extends readonly any[] | ((...args: any[]) => any) ? false : T extends Record<string, any> ? true : false

type MergedAppConfig<Resolved extends Record<string, unknown>, Custom extends Record<string, unknown>> = {
  [K in keyof (Resolved & Custom)]: K extends keyof Custom
    ? unknown extends Custom[K]
      ? Resolved[K]
      : IsAny<Custom[K]> extends true
        ? Resolved[K]
        : IsMergeable<Custom[K]> extends true
            ? IsMergeable<Resolved[K]> extends true
              ? MergedAppConfig<Resolved[K], Custom[K]>
              : Exclude<Custom[K], undefined>
            : Exclude<Custom[K], undefined>
    : Resolved[K]
}

declare module 'nuxt/schema' {
  interface SharedAppConfig extends MergedAppConfig<typeof inlineConfig, CustomAppConfig> { }
}
declare module '@nuxt/schema' {
  interface SharedAppConfig extends MergedAppConfig<typeof inlineConfig, CustomAppConfig> { }
}
