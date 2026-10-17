import { ref } from 'vue'

export type ToastKind = 'success' | 'error' | 'info'

export interface Toast {
  id: number
  message: string
  kind: ToastKind
}

const toasts = ref<Toast[]>([])
let seq = 0

function push(message: string, kind: ToastKind = 'info', ttl = 3500) {
  const id = ++seq
  toasts.value = [...toasts.value, { id, message, kind }]
  window.setTimeout(() => dismiss(id), ttl)
  return id
}

function dismiss(id: number) {
  toasts.value = toasts.value.filter((t) => t.id !== id)
}

export function useToast() {
  return {
    toasts,
    dismiss,
    success: (m: string) => push(m, 'success'),
    error: (m: string) => push(m, 'error', 5000),
    info: (m: string) => push(m, 'info'),
  }
}
