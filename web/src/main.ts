import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import './style.css'
import { i18n } from './i18n'

const app = createApp(App)

// 全局渲染错误兜底：单个组件崩溃不应让整页白屏。
// 生产环境把错误显示为友好提示，同时保留控制台详情便于排查。
app.config.errorHandler = (err, _instance, info) => {
  console.error('[gitli] 组件渲染错误:', err, info)
  const host = document.getElementById('app')
  if (!host) return
  if (host.querySelector('[data-fatal-error]')) return
  const box = document.createElement('div')
  box.setAttribute('data-fatal-error', '')
  box.style.cssText =
    'max-width:560px;margin:4rem auto;padding:1.5rem;border:1px solid #d0d7de;border-radius:12px;' +
    'font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif;line-height:1.6;' +
    'background:#fff;color:#1f2328;box-shadow:0 8px 24px rgba(0,0,0,.08)'
  box.innerHTML =
    '<h2 style="margin:0 0 .5rem;font-size:1.1rem">页面渲染出错</h2>' +
    '<p style="margin:0 0 1rem;color:#57606a;font-size:.9rem">请刷新重试。若问题持续，请把浏览器控制台的错误信息反馈给我们。</p>' +
    '<button style="padding:.5rem 1rem;border-radius:6px;border:none;background:#3b82f6;color:#fff;cursor:pointer;font-size:.9rem">刷新页面</button>'
  box.querySelector('button')?.addEventListener('click', () => location.reload())
  host.prepend(box)
}

app.use(createPinia()).use(router).use(i18n).mount('#app')
