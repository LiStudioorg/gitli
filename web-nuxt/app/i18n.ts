import { createI18n } from 'vue-i18n'
import type { WritableComputedRef } from 'vue'
import zhCN from './locales/zh-CN.json'
import enUS from './locales/en-US.json'

const STORAGE_KEY = 'gitli-locale'

export type AppLocale = 'zh-CN' | 'en-US'

export const localeOptions: ReadonlyArray<{ value: AppLocale; label: string }> = [
  { value: 'zh-CN', label: '简体中文' },
  { value: 'en-US', label: 'English' },
]

function isAppLocale(value: unknown): value is AppLocale {
  return value === 'zh-CN' || value === 'en-US'
}

// 中文优先：这是面向国内团队自托管的平台，默认界面必须是中文。
// 只有浏览器语言明确是英文时才切到 en-US，其余（含未设置、其他语言）一律中文。
// 用户手动切换过则以 localStorage 为准。
function detectLocale(): AppLocale {
  const saved = localStorage.getItem(STORAGE_KEY)
  if (isAppLocale(saved)) return saved
  const nav = (navigator.language || '').toLowerCase()
  return nav.startsWith('en') ? 'en-US' : 'zh-CN'
}

export const i18n = createI18n({
  legacy: false,
  locale: detectLocale(),
  fallbackLocale: 'zh-CN',
  messages: {
    'zh-CN': zhCN,
    'en-US': enUS,
  },
})

// vue-i18n 在 legacy:false 下把 global 推断成 Composer|VueI18n 联合类型，取不到
// WritableComputedRef。断言集中在这一个访问器里，而不是散落到每个调用点。
const globalLocale = i18n.global.locale as unknown as WritableComputedRef<AppLocale>

export function setLocale(locale: AppLocale): void {
  globalLocale.value = locale
  localStorage.setItem(STORAGE_KEY, locale)
  document.documentElement.lang = locale
}

export function currentLocale(): AppLocale {
  return globalLocale.value
}
