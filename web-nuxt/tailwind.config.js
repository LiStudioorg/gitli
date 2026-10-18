/** @type {import('tailwindcss').Config} */
// 与 web/tailwind.config.js 保持同一套设计令牌，只把 content 指向 Nuxt 的 app/ 目录。
// 两个前端在迁移期共存，令牌必须逐项对齐，否则同一个组件在两边显示不一致。
export default {
  content: ['./app/**/*.{vue,js,ts,jsx,tsx}', './app/**/*.md'],
  darkMode: ['class', '[data-theme="dark"]'],
  theme: {
    extend: {
      colors: {
        canvas: '#f6f8fa',
        surface: '#ffffff',
        line: '#d0d7de',
        ink: '#1f2328',
        subtle: '#656d76',
        danger: { bg: '#ffebe9', border: '#ff818266' },
      },
      fontFamily: {
        sans: ['-apple-system', 'BlinkMacSystemFont', 'Segoe UI', 'Noto Sans', 'Helvetica', 'Arial', 'sans-serif'],
        mono: ['ui-monospace', 'SFMono-Regular', 'Menlo', 'Consolas', 'monospace'],
      },
      borderRadius: { md: '6px' },
    },
  },
  plugins: [],
}
