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
    if (err?.response?.status === 412 && err.response.data?.code === 'setup_required') {
      if (location.pathname !== '/setup') {
        location.assign('/setup')
      }
    }
    return Promise.reject(err)
  },
)

export default api
