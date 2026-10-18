import axios from 'axios'

function readCookie(name: string): string {
  const match = document.cookie.match(new RegExp('(?:^|; )' + name + '=([^;]*)'))
  const raw = match?.[1]
  return raw ? decodeURIComponent(raw) : ''
}

const api = axios.create({
  baseURL: '/api/v1',
  withCredentials: true,
})

api.interceptors.request.use((config) => {
  const method = (config.method ?? 'get').toLowerCase()
  if (method !== 'get' && method !== 'head') {
    config.headers = config.headers ?? {}
    config.headers['X-CSRF-Token'] = readCookie('gitli_csrf')
  }
  return config
})

api.interceptors.response.use(
  (res) => res,
  (err) => {
    const status = err?.response?.status
    if (status === 412 && err.response.data?.code === 'setup_required') {
      if (location.pathname !== '/setup') location.assign('/setup')
      return Promise.reject(err)
    }
    if (status === 401) {
      // Only session loss matters here. A 401 from /auth/login is the user's
      // own wrong password and is reported inline by the login form.
      const url = String(err.config?.url ?? '')
      const isLoginAttempt = url.includes('/auth/login')
      if (!isLoginAttempt && !location.pathname.startsWith('/login') && !location.pathname.startsWith('/setup')) {
        void notifyExpired()
      }
    }
    return Promise.reject(err)
  },
)

async function notifyExpired() {
  const { useToast } = await import('../composables/useToast')
  useToast().error('登录已过期，请重新登录')
  const redirect = encodeURIComponent(location.pathname + location.search)
  window.setTimeout(() => {
    location.assign(`/login?redirect=${redirect}`)
  }, 400)
}

export default api
