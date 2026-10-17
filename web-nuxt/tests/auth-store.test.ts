import { describe, expect, it, beforeEach, afterEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import api from '../app/api/client'
import { useAuthStore } from '../app/stores'

type Call = { method: string; url: string }

function stubAdapter(routes: Record<string, unknown>) {
  const calls: Call[] = []
  const original = api.defaults.adapter
  api.defaults.adapter = async (config) => {
    const method = (config.method ?? 'get').toUpperCase()
    const url = config.url ?? ''
    calls.push({ method, url })
    const key = `${method} ${url}`
    const res = { headers: {}, config, statusText: 'OK' }
    if (key in routes) {
      return { ...res, data: routes[key], status: 200 }
    }
    // 模拟 axios 的真实行为：非 2xx 必须以 reject 结束，否则 store 的 catch 分支不会被触发。
    const err = new Error(`Request failed with status code 404`) as Error & {
      isAxiosError: boolean
      response: { status: number; data: unknown; config: unknown; headers: object; statusText: string }
    }
    err.isAxiosError = true
    err.response = { ...res, data: { error: 'not found' }, status: 404 }
    throw err
  }
  return { calls, restore: () => { api.defaults.adapter = original } }
}

describe('auth store 登录（登录后 isAdmin 必须为真）', () => {
  beforeEach(() => setActivePinia(createPinia()))
  afterEach(() => {
    api.defaults.adapter = undefined
  })

  it('login 之后回读 /user/me，is_admin 生效、isAdmin getter 为 true', async () => {
    const { calls, restore } = stubAdapter({
      'POST /auth/login': { id: 1, username: 'root' }, // 后端即便只回最小字段也不影响
      'GET /user/me': {
        id: 1,
        username: 'root',
        email: 'root@example.com',
        full_name: '',
        bio: '',
        avatar_url: '',
        website: '',
        location: '',
        is_admin: true,
        totp_enabled: false,
      },
    })
    try {
      const auth = useAuthStore()
      const user = await auth.login('root', 'Sup3rSecretPassword1')
      expect(calls.map((c) => `${c.method} ${c.url}`)).toEqual(['POST /auth/login', 'GET /user/me'])
      expect(user?.is_admin).toBe(true)
      expect(auth.isAdmin).toBe(true)
      expect(auth.isLoggedIn).toBe(true)
      expect(auth.loaded).toBe(true)
    } finally {
      restore()
    }
  })

  it('登录失败时抛出且不写入 user', async () => {
    const { restore } = stubAdapter({})
    try {
      const auth = useAuthStore()
      await expect(auth.login('root', 'wrong')).rejects.toBeTruthy()
      expect(auth.user).toBeNull()
    } finally {
      restore()
    }
  })

  it('/user/me 401 时 fetchMe 归零为未登录且 loaded 置真', async () => {
    const { restore } = stubAdapter({})
    try {
      const auth = useAuthStore()
      await auth.fetchMe()
      expect(auth.user).toBeNull()
      expect(auth.loaded).toBe(true)
      expect(auth.isAdmin).toBe(false)
    } finally {
      restore()
    }
  })
})
