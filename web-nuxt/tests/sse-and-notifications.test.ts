import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest'
import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'

// --- 最小 EventSource stub：记录实例，便于主动派发事件 ---
class FakeEventSource {
  static instances: FakeEventSource[] = []
  url: string
  withCredentials: boolean
  onopen: (() => void) | null = null
  onerror: (() => void) | null = null
  closed = false
  private listeners = new Map<string, ((ev: MessageEvent<string>) => void)[]>()

  constructor(url: string, init?: { withCredentials?: boolean }) {
    this.url = url
    this.withCredentials = init?.withCredentials ?? false
    FakeEventSource.instances.push(this)
  }
  addEventListener(type: string, cb: (ev: MessageEvent<string>) => void) {
    const arr = this.listeners.get(type) ?? []
    arr.push(cb)
    this.listeners.set(type, arr)
  }
  emit(type: string, data: string) {
    for (const cb of this.listeners.get(type) ?? []) {
      cb({ data } as MessageEvent<string>)
    }
  }
  close() {
    this.closed = true
  }
}

const { useSSE } = await import('../app/composables/useSSE')
const { useNotificationStore } = await import('../app/stores')

function mountSSE(onNotification: (n: unknown) => void) {
  const Comp = defineComponent({
    setup() {
      return useSSE({ onNotification: onNotification as never })
    },
    render: () => h('div'),
  })
  return mount(Comp)
}

describe('useSSE（与原前端一致）', () => {
  beforeEach(() => {
    FakeEventSource.instances = []
    vi.stubGlobal('EventSource', FakeEventSource as unknown as typeof EventSource)
  })
  afterEach(() => {
    vi.unstubAllGlobals()
    vi.useRealTimers()
  })

  it('连到 /api/v1/events 并带凭据', () => {
    const w = mountSSE(() => {})
    const es = FakeEventSource.instances[0]!
    expect(es.url).toBe('/api/v1/events')
    expect(es.withCredentials).toBe(true)
    w.unmount()
  })

  it('notification 事件解析后回调，坏 JSON 被忽略', () => {
    const got: unknown[] = []
    const w = mountSSE((n) => got.push(n))
    const es = FakeEventSource.instances[0]!
    es.emit('notification', JSON.stringify({ ID: 1, Title: 'hello', IsRead: false }))
    es.emit('notification', '{not json')
    expect(got).toHaveLength(1)
    expect((got[0] as { Title: string }).Title).toBe('hello')
    w.unmount()
  })

  it('断线后按退避重连，unmount 后不再重连且关闭连接', () => {
    vi.useFakeTimers()
    const w = mountSSE(() => {})
    const first = FakeEventSource.instances[0]!
    first.onerror?.()
    expect(first.closed).toBe(true)
    vi.advanceTimersByTime(1000)
    expect(FakeEventSource.instances).toHaveLength(2)
    w.unmount()
    expect(FakeEventSource.instances[1]!.closed).toBe(true)
    vi.advanceTimersByTime(60_000)
    expect(FakeEventSource.instances).toHaveLength(2)
  })
})

describe('notification store（SSE 推送后的未读计数）', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('push 未读条目使 unread +1，markRead 递减并标记 IsRead', async () => {
    const store = useNotificationStore()
    store.push({ ID: 7, UserID: 1, SubjectType: 'issue', SubjectID: 1, Title: 't', URL: '/x', IsRead: false, CreatedAt: '' })
    expect(store.unread).toBe(1)
    expect(store.items[0]!.ID).toBe(7)
    // 已读条目不计入未读
    store.push({ ID: 8, UserID: 1, SubjectType: 'issue', SubjectID: 1, Title: 'u', URL: '/y', IsRead: true, CreatedAt: '' })
    expect(store.unread).toBe(1)
  })
})
