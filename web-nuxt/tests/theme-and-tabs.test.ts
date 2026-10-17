import { beforeEach, describe, expect, it } from 'vitest'
import { nextTick } from 'vue'
import { useTheme } from '../app/composables/useTheme'
import { REPO_TABS, TABS, isRepoTab } from '../app/constants/tabs'

describe('useTheme（与原前端行为一致）', () => {
  beforeEach(() => {
    localStorage.clear()
    document.documentElement.removeAttribute('data-theme')
  })

  it('cycleTheme 依次 light -> dark -> auto 并写入 data-theme 与 localStorage', async () => {
    const { theme, cycleTheme } = useTheme()
    theme.value = 'light'
    await nextTick()
    cycleTheme()
    await nextTick()
    expect(theme.value).toBe('dark')
    expect(document.documentElement.getAttribute('data-theme')).toBe('dark')
    expect(localStorage.getItem('gitli-theme')).toBe('dark')
    cycleTheme()
    await nextTick()
    expect(theme.value).toBe('auto')
    // auto 在 happy-dom 下 matchMedia 缺省为浅色
    expect(document.documentElement.getAttribute('data-theme')).toBe('light')
    cycleTheme()
    await nextTick()
    expect(theme.value).toBe('light')
  })
})

describe('仓库页 Tab 契约（URL 中文标签不可变）', () => {
  it('保持旧前端的标签集合与顺序', () => {
    expect(REPO_TABS).toEqual([
      '代码',
      '工单',
      '合并请求',
      '工作流',
      '依赖',
      '项目',
      '讨论',
      '发布',
      '百科',
      '活动',
      '设置',
    ])
  })

  it('isRepoTab 拒绝拼错与非字符串，保证不会静默落到占位分支', () => {
    expect(isRepoTab('工单')).toBe(true)
    expect(isRepoTab('合并请示')).toBe(false)
    expect(isRepoTab(undefined)).toBe(false)
    expect(isRepoTab(1)).toBe(false)
  })

  it('快捷键用到的常量就是 URL 里的中文值', () => {
    expect(TABS.issues).toBe('工单')
    expect(TABS.pulls).toBe('合并请求')
    expect(encodeURIComponent(TABS.issues)).toBe(encodeURIComponent('工单'))
  })
})
