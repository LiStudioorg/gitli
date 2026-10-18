import { i18n } from '~/i18n'

// 保持与旧前端 main.ts 完全相同的 vue-i18n 用法（legacy:false + 全局实例），
// 因此这里只做插件注册，不引入 @nuxtjs/i18n 改写调用点。
// 文件名带 .client 后缀：app/i18n.ts 在模块顶层读 localStorage/navigator，
// 只在浏览器端加载才安全（generate 预渲染外壳时不会执行）。
export default defineNuxtPlugin((nuxtApp) => {
  nuxtApp.vueApp.use(i18n)
})
