import { describe, expect, it } from 'vitest'
import { formatDate, formatDateTime, formatRelative } from '../app/utils/datetime'

// 统一日期格式的目的：不跟随浏览器区域设置，国内用户始终看到 年-月-日。
describe('utils/datetime（固定为中文习惯格式）', () => {
  it('formatDate 输出 YYYY-MM-DD', () => {
    // 2026-10-18 06:00 UTC；不带时区后缀时按 UTC 解析
    expect(formatDate('2026-10-18T06:00:00Z')).toMatch(/^\d{4}-\d{2}-\d{2}$/)
  })

  it('formatDateTime 输出日期 + 时:分', () => {
    expect(formatDateTime('2026-10-18T06:00:00Z')).toMatch(/^\d{4}-\d{2}-\d{2} \d{2}:\d{2}$/)
  })

  it('后端常见的空格分隔、无时区串按 UTC 解析而非报 Invalid', () => {
    expect(formatDate('2026-10-18 06:00:00')).toMatch(/^\d{4}-\d{2}-\d{2}$/)
  })

  it('非法时间原样返回，不抛错', () => {
    expect(formatDate('not-a-date')).toBe('not-a-date')
    expect(formatDateTime('')).toBe('')
  })

  it('formatRelative 用中文相对时间，超过 30 天回退日期', () => {
    const now = Date.now()
    const iso = (msAgo: number) => new Date(now - msAgo).toISOString()
    expect(formatRelative(iso(10_000))).toBe('刚刚')
    expect(formatRelative(iso(5 * 60_000))).toBe('5 分钟前')
    expect(formatRelative(iso(3 * 3600_000))).toBe('3 小时前')
    expect(formatRelative(iso(2 * 86400_000))).toBe('2 天前')
    expect(formatRelative(iso(90 * 86400_000))).toMatch(/^\d{4}-\d{2}-\d{2}$/)
  })
})
