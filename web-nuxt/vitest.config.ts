import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'

// 行为回归测试跑在 app/ 之外：这些用例验证的是「迁移后行为与旧前端一致」，
// 不需要经 Nuxt 构建链路，因此直接用 happy-dom 挂载组件 / 调用组合式函数。
export default defineConfig({
  plugins: [vue()],
  test: {
    environment: 'happy-dom',
    include: ['tests/**/*.test.ts'],
    globals: true,
  },
  resolve: {
    alias: {
      '~': new URL('./app', import.meta.url).pathname,
    },
  },
})
