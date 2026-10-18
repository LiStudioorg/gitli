export default defineNuxtConfig({
  // 可用性优先的第一条：产物必须是纯静态文件，交给 Go 的 //go:embed 提供。
  // Nuxt 默认的 Node server 输出会直接违反 AGENTS.md 的「单二进制 / 无外部强依赖 /
  // 空闲内存 < 150MB」，所以这里禁用 SSR，并用 static preset 关掉 Nitro 运行时。
  ssr: false,
  compatibilityDate: '2025-09-01',

  modules: ['@pinia/nuxt', '@nuxtjs/tailwindcss'],

  tailwindcss: {
    // 直接复用仓库根 web-nuxt/tailwind.config.js，避免模块内置默认值覆盖设计令牌。
    configPath: 'tailwind.config.js',
    viewer: false,
  },

  app: {
    head: {
      htmlAttrs: { lang: 'zh-CN' },
      title: 'gitli',
      meta: [
        { name: 'viewport', content: 'width=device-width, initial-scale=1.0' },
        { charset: 'utf-8' },
      ],
    },
  },

  css: ['~/style.css'],

  nitro: {
    preset: 'static',
    prerender: {
      // generate 会同时产出 index.html 与 200.html（内容一致），因此现有
      // web/embed.go 的「未知路径回退 index.html」零改动即可支撑
      // /owner/repo、/users/x、?tab=… 这些无法预渲染的动态路由。
      routes: ['/'],
    },
  },

  typescript: {
    strict: true,
    typeCheck: false,
  },

  devServer: {
    port: 5174,
  },

  vite: {
    server: {
      proxy: {
        '/api': 'http://127.0.0.1:8080',
      },
    },
  },
})
