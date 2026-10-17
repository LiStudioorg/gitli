import { createI18n } from 'vue-i18n'
import zhCN from './locales/zh-CN.json'
import enUS from './locales/en-US.json'

const STORAGE_KEY = 'gitli-locale'

function detectLocale(): string {
  const saved = localStorage.getItem(STORAGE_KEY)
  if (saved === 'zh-CN' || saved === 'en-US') return saved
  const nav = navigator.language || 'zh-CN'
  return nav.toLowerCase().startsWith('en') ? 'en-US' : 'zh-CN'
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

export function setLocale(locale: string): void {
  if (locale !== 'zh-CN' && locale !== 'en-US') return
  ;(i18n.global.locale as unknown as { value: string }).value = locale
  localStorage.setItem(STORAGE_KEY, locale)
  document.documentElement.lang = locale
}

export function currentLocale(): string {
  return (i18n.global.locale as unknown as { value: string }).value
}

export const localeOptions = [
  { value: 'zh-CN', label: '简体中文' },
  { value: 'en-US', label: 'English' },
]
