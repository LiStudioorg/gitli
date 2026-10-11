import axios from 'axios'

function readCookie(name: string): string {
  const match = document.cookie.match(new RegExp('(?:^|; )' + name + '=([^;]*)'))
  return match ? decodeURIComponent(match[1]) : ''
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

export default api
