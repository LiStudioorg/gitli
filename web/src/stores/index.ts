import { defineStore } from 'pinia'
import api from '../api/client'

export interface User {
  id: number
  username: string
  email: string
  full_name: string
  is_admin: boolean
  totp_enabled: boolean
}

export interface Repo {
  id: number
  name: string
  description: string
  default_branch: string
  is_private: boolean
  updated_at: string
}

export interface Activity {
  id: number
  verb: string
  object_type: string
  object_id: number
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
      const { data } = await api.post<User>('/auth/login', {
        username,
        password,
        totp_code: totpCode,
      })
      this.user = data
      this.loaded = true
      return data
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
  },
})
