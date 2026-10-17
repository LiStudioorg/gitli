import axios from 'axios'

/**
 * 后端 writeError 的统一信封是 {error: "..."}（见 internal/transport/http/util.go
 * 的 apiError），因此这里只认 `error` 字段，不去猜其它键名。
 */
export function errMessage(e: unknown, fallback = '未知错误'): string {
  if (axios.isAxiosError(e)) {
    const data = e.response?.data as { error?: string } | undefined
    if (data?.error) return data.error
    if (e.response?.status) return `请求失败（HTTP ${e.response.status}）`
    return e.message
  }
  return e instanceof Error ? e.message : fallback
}

export function apiStatus(e: unknown): number | undefined {
  return axios.isAxiosError(e) ? e.response?.status : undefined
}
