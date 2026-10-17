import { defineStore } from 'pinia'
import api from '../api/client'

export interface User {
  id: number
  username: string
  email: string
  full_name: string
  bio: string
  avatar_url: string
  website: string
  location: string
  is_admin: boolean
  totp_enabled: boolean
}

export interface Repo {
  id: number
  name: string
  description: string
  default_branch: string
  is_private: boolean
  is_fork?: boolean
  archived: boolean
  archived_at: string | null
  updated_at: string
  owner_name?: string
  relation?: 'owned' | 'collaborated'
}

export interface Activity {
  id: number
  actor_id: number
  verb: string
  object_type: string
  object_id: number
  repo_id?: number | null
  repo_name?: string
  summary?: string
  actor_name?: string
  created_at: string
}

export interface LanguageStat {
  language: string
  bytes: number
  percentage: number
}

export interface RepoDetail {
  repo: Repo
  languages: LanguageStat[]
}

export interface CloneInfo {
  https: string
  ssh: string
  clone_default_branch: string
}

export interface Notification {
  ID: number
  UserID: number
  SubjectType: string
  SubjectID: number
  Title: string
  URL: string
  IsRead: boolean
  CreatedAt: string
}

export interface NotificationList {
  items: Notification[]
  unread: number
}

export interface Issue {
  ID: number
  RepoID: number
  Number: number
  Title: string
  Body: string
  AuthorID: number
  State: string
  CreatedAt: string
  UpdatedAt: string
}

export interface Pull {
  ID: number
  RepoID: number
  Number: number
  Title: string
  Body: string
  AuthorID: number
  State: string
  Merged: boolean
  HeadBranch: string
  BaseBranch: string
  CreatedAt: string
  UpdatedAt: string
}

export const useUiStore = defineStore('ui', {
  state: () => ({
    commandPaletteOpen: false,
    sidebarOpen: false,
  }),
  actions: {
    toggleSidebar() {
      this.sidebarOpen = !this.sidebarOpen
    },
    closeSidebar() {
      this.sidebarOpen = false
    },
  },
})

export const useAuthStore = defineStore('auth', {
  state: () => ({
    user: null as User | null,
    loaded: false,
  }),
  getters: {
    isLoggedIn: (s) => s.user !== null,
    isAdmin: (s) => s.user?.is_admin ?? false,
  },
  actions: {
    async fetchMe() {
      try {
        const { data } = await api.get<User>('/user/me')
        this.user = data
      } catch {
        this.user = null
      } finally {
        this.loaded = true
      }
      return this.user
    },
    async login(username: string, password: string, totpCode = '') {
      // 登录响应现在返回完整公开档案；/user/me 回读用于确认会话 Cookie 已生效。
      // 若回读偶发失败（网络抖动/代理丢 Cookie），回退到登录响应，避免“登录成功却进不去”。
      const { data } = await api.post<User>('/auth/login', {
        username,
        password,
        totp_code: totpCode,
      })
      const me = await this.fetchMe()
      if (!me && data) {
        this.user = data
        this.loaded = true
        return this.user
      }
      return me
    },
    async logout() {
      try {
        await api.post('/auth/logout')
      } finally {
        this.user = null
      }
    },
  },
})

export const useNotificationStore = defineStore('notifications', {
  state: () => ({
    items: [] as Notification[],
    unread: 0,
    loading: false,
    error: '' as string,
  }),
  actions: {
    async fetchAll() {
      this.loading = true
      this.error = ''
      try {
        const { data } = await api.get<NotificationList>('/notifications')
        this.items = data.items ?? []
        this.unread = data.unread ?? 0
      } catch (e) {
        this.error = e instanceof Error ? e.message : '加载通知失败'
      } finally {
        this.loading = false
      }
    },
    push(item: Notification) {
      this.items.unshift(item)
      if (!item.IsRead) this.unread += 1
    },
    async markRead(id: number) {
      await api.post(`/notifications/${id}/read`)
      const target = this.items.find((n) => n.ID === id)
      if (target && !target.IsRead) {
        target.IsRead = true
        this.unread = Math.max(0, this.unread - 1)
      }
    },
  },
})

export const useRepoStore = defineStore('repos', {
  state: () => ({
    myRepos: [] as Repo[],
    activities: [] as Activity[],
  }),
  actions: {
    async fetchMyRepos() {
      const { data } = await api.get<Repo[]>('/user/repos')
      this.myRepos = data ?? []
    },
    async fetchActivities() {
      const { data } = await api.get<Activity[]>('/activities', {
        params: { page: 1, per_page: 20 },
      })
      this.activities = data ?? []
    },
    async removeRepo(repo: Repo) {
      const owner = repo.owner_name || ''
      await api.delete(`/repos/${owner}/${repo.name}`)
      this.myRepos = this.myRepos.filter((r) => r.id !== repo.id)
    },
  },
})
