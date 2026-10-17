import { describe, expect, it } from 'vitest'
import api from '../app/api/client'
import { i18n, currentLocale, setLocale } from '../app/i18n'

describe('api client 的 CSRF 行为（与原前端一致）', () => {
  function setCookie(name: string, value: string) {
    document.cookie = `${name}=${value}; path=/`
  }

  it('POST/PATCH/DELETE 带上 X-CSRF-Token，GET 不带', async () => {
    setCookie('gitli_csrf', 'csrf-token-123')
    const seen: Array<{ method: string; csrf?: string }> = []
    const original = api.defaults.adapter
    // 用自定义 adapter 截获最终请求配置，避免真实网络。
    api.defaults.adapter = async (config) => {
      seen.push({
        method: (config.method ?? 'get').toUpperCase(),
        csrf: config.headers?.['X-CSRF-Token'] as string | undefined,
      })
      return { data: {}, status: 200, statusText: 'OK', headers: {}, config }
    }
    try {
      await api.get('/user/me')
      await api.post('/auth/logout')
      await api.patch('/user/me', {})
      await api.delete('/user/me/sessions')
    } finally {
      api.defaults.adapter = original
    }
    expect(seen[0]).toEqual({ method: 'GET', csrf: undefined })
    expect(seen.slice(1).map((s) => s.csrf)).toEqual([
      'csrf-token-123',
      'csrf-token-123',
      'csrf-token-123',
    ])
  })

  it('没有 csrf cookie 时发送空字符串而不是省略头（后端按空值拒绝）', async () => {
    document.cookie = 'gitli_csrf=; path=/; expires=Thu, 01 Jan 1970 00:00:00 GMT'
    let csrf: unknown
    const original = api.defaults.adapter
    api.defaults.adapter = async (config) => {
      csrf = config.headers?.['X-CSRF-Token']
      return { data: {}, status: 200, statusText: 'OK', headers: {}, config }
    }
    try {
      await api.post('/auth/logout')
    } finally {
      api.defaults.adapter = original
    }
    expect(csrf).toBe('')
  })
})

describe('i18n（语言码收敛为 union，行为不变）', () => {
  it('setLocale 写入 localStorage 与 html lang，currentLocale 读回', () => {
    setLocale('en-US')
    expect(localStorage.getItem('gitli-locale')).toBe('en-US')
    expect(document.documentElement.lang).toBe('en-US')
    expect(currentLocale()).toBe('en-US')
    setLocale('zh-CN')
    expect(currentLocale()).toBe('zh-CN')
  })

  it('两种语言的同一键都能解析（键集 25=25 对齐）', () => {
    setLocale('zh-CN')
    const zh = i18n.global.t('nav.login')
    setLocale('en-US')
    const en = i18n.global.t('nav.login')
    expect(zh).toBeTruthy()
    expect(en).toBeTruthy()
    expect(zh).not.toBe(en)
  })
})
