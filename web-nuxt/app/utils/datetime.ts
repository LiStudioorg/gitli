// 日期/时间与相对时间的统一格式化。
//
// 为什么要统一：Vue 组件里原本散落 11 处无参 `toLocaleDateString()`，
// 结果跟随浏览器区域设置 —— 中国用户常看到 10/18/2026 这种月/日/年格式，
// 与国内习惯（年-月-日）相反；同一页面里两个组件还可能显示不同格式。
// 这里固定输出 2026-10-18 与 2026-10-18 14:30，不随环境漂移。

const pad = (n: number): string => String(n).padStart(2, '0')

// 后端返回的时间可能是秒/毫秒/带 Z/带空格/不带时区，统一成 Date。
function toDate(raw: string): Date | null {
  if (!raw) return null
  const normalized = raw.endsWith('Z') || raw.includes('+') ? raw : raw.replace(' ', 'T') + 'Z'
  const d = new Date(normalized)
  return Number.isNaN(d.getTime()) ? null : d
}

/** 只到日期：2026-10-18。 */
export function formatDate(raw: string): string {
  const d = toDate(raw)
  if (!d) return raw
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

/** 到分钟：2026-10-18 14:30。 */
export function formatDateTime(raw: string): string {
  const d = toDate(raw)
  if (!d) return raw
  return `${formatDate(raw)} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

/** 相对时间：刚刚 / 5 分钟前 / 3 小时前 / 2 天前，超过 30 天回退到日期。 */
export function formatRelative(raw: string): string {
  const d = toDate(raw)
  if (!d) return raw
  const diff = Date.now() - d.getTime()
  if (diff < 0) return formatDate(raw)
  const minutes = Math.floor(diff / 60000)
  if (minutes < 1) return '刚刚'
  if (minutes < 60) return `${minutes} 分钟前`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours} 小时前`
  const days = Math.floor(hours / 24)
  if (days < 30) return `${days} 天前`
  return formatDate(raw)
}
