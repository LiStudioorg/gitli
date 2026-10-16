import { onBeforeUnmount, ref } from 'vue'
import type { Notification } from '../stores'

export interface UseSSEOptions {
  path?: string
  onNotification?: (item: Notification) => void
}

const EVENTS_PATH = '/api/v1/events'
const MAX_BACKOFF = 30_000

export function useSSE(options: UseSSEOptions = {}) {
  const path = options.path ?? EVENTS_PATH
  const connected = ref(false)
  const supported = typeof window !== 'undefined' && typeof window.EventSource === 'function'

  let source: EventSource | null = null
  let retryTimer: ReturnType<typeof setTimeout> | null = null
  let backoff = 1_000
  let closed = false

  function parse(data: string): Notification | null {
    try {
      return JSON.parse(data) as Notification
    } catch {
      return null
    }
  }

  function scheduleReconnect() {
    if (closed || retryTimer) return
    retryTimer = setTimeout(() => {
      retryTimer = null
      connect()
    }, backoff)
    backoff = Math.min(backoff * 2, MAX_BACKOFF)
  }

  function connect() {
    if (!supported || closed || source) return
    try {
      source = new EventSource(path, { withCredentials: true })
    } catch {
      scheduleReconnect()
      return
    }

    source.onopen = () => {
      connected.value = true
      backoff = 1_000
    }
    source.onerror = () => {
      connected.value = false
      if (source) {
        source.close()
        source = null
      }
      scheduleReconnect()
    }
    source.addEventListener('notification', (ev) => {
      const item = parse((ev as MessageEvent<string>).data)
      if (item) options.onNotification?.(item)
    })
  }

  function disconnect() {
    closed = true
    if (retryTimer) {
      clearTimeout(retryTimer)
      retryTimer = null
    }
    if (source) {
      source.close()
      source = null
    }
    connected.value = false
  }

  connect()
  onBeforeUnmount(disconnect)

  return { connected, supported, disconnect }
}

export default useSSE
