/**
 * 仓库页 Tab 是「URL 契约」的一部分：?tab=工单 这类链接已经存在于用户书签、
 * 通知与快捷键里，所以标识符只能用中文标签本身，不能改成英文 key。
 * 这里把它收敛成 union 类型，拼错时由 vue-tsc 直接报错，而不是静默落到
 * 「即将推出」占位分支。
 */
export const TABS = {
  code: '代码',
  issues: '工单',
  pulls: '合并请求',
  workflows: '工作流',
  deps: '依赖',
  projects: '项目',
  discussions: '讨论',
  releases: '发布',
  wiki: '百科',
  activity: '活动',
  settings: '设置',
} as const

export type RepoTab = (typeof TABS)[keyof typeof TABS]

/** 渲染顺序即 tab 顺序。 */
export const REPO_TABS: readonly RepoTab[] = Object.values(TABS)

export function isRepoTab(value: unknown): value is RepoTab {
  return typeof value === 'string' && (REPO_TABS as readonly string[]).includes(value)
}
