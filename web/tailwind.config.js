/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{vue,js,ts,jsx,tsx}'],
  darkMode: ['class', '[data-theme="dark"]'],
  theme: {
    extend: {
      colors: {
        // GitHub Primer 真实取值（从 github/ 抓取源码中的 CSS 变量提取）
        canvas: '#f6f8fa',
        surface: '#ffffff',
        line: '#d1d9e0',
        'line-muted': '#d1d9e0',
        ink: '#1f2328',
        subtle: '#59636e',
        brand: '#ea580c',
        'brand-hover': '#c2410c',
        'btn-green': '#2da44e',
        'btn-green-hover': '#2c974b',
        danger: { bg: '#ffebe9', border: '#ff818266' },
      },
      fontFamily: {
        sans: ['-apple-system', 'BlinkMacSystemFont', 'Segoe UI', 'Noto Sans', 'Helvetica', 'Arial', 'sans-serif'],
        mono: ['ui-monospace', 'SFMono-Regular', 'Menlo', 'Consolas', 'monospace'],
      },
      borderRadius: {
        // --borderRadius-medium: .375rem
        md: '0.375rem',
      },
      spacing: {
        // --base-size-* 系列
        'base-2': '0.125rem',
        'base-4': '0.25rem',
        'base-8': '0.5rem',
        'base-12': '0.75rem',
        'base-16': '1rem',
        'base-20': '1.25rem',
        'base-24': '1.5rem',
      },
      height: {
        // --control-medium-size / --control-small-size
        control: '2rem',
        'control-sm': '1.75rem',
      },
    },
  },
  plugins: [],
}
