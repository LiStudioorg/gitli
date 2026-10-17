<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import TopNav from '../components/layout/TopNav.vue'
import api from '../api/client'
import { useAuthStore } from '../stores'

interface ProfileUser {
  id: number
  username: string
  email: string
  full_name: string
  bio: string
  avatar_url: string
  website: string
  location: string
  is_admin: boolean
}

interface ProfileRepo {
  ID: number
  Name: string
  Description: string
  DefaultBranch: string
  StarsCount: number
  updated_at?: string
}

interface ProfileResponse {
  user: ProfileUser
  repos: ProfileRepo[]
  followers: number
  following: number
  is_following: boolean
}

interface FollowUser {
  id: number
  username: string
  full_name: string
}

const route = useRoute()
const auth = useAuthStore()

const username = computed(() => route.params.username as string)

const profile = ref<ProfileResponse | null>(null)

// Only http(s) links are rendered as anchors; anything else (notably
// javascript:) is shown as plain text so a profile cannot execute script.
function safeWebsite(url: string | undefined): string {
  if (!url) return ''
  try {
    const parsed = new URL(url, window.location.origin)
    return parsed.protocol === 'http:' || parsed.protocol === 'https:' ? parsed.toString() : ''
  } catch {
    return ''
  }
}
const loading = ref(false)
const error = ref('')

const tab = ref<'repos' | 'followers' | 'following'>('repos')
const followLoading = ref(false)
const followError = ref('')

const followers = ref<FollowUser[]>([])
const following = ref<FollowUser[]>([])
const listLoading = ref(false)
const listError = ref('')
const listsLoaded = ref({ followers: false, following: false })

const isSelf = computed(
  () => auth.isLoggedIn && auth.user?.username === profile.value?.user.username,
)

function initial(name: string | undefined): string {
  return name?.[0]?.toUpperCase() ?? '?'
}

async function loadProfile() {
  loading.value = true
  error.value = ''
  try {
    const { data } = await api.get<ProfileResponse>(`/users/${username.value}`)
    profile.value = data
  } catch (e) {
    error.value = e instanceof Error ? e.message : '加载用户信息失败'
    profile.value = null
  } finally {
    loading.value = false
  }
}

async function toggleFollow() {
  if (!profile.value) return
  followLoading.value = true
  followError.value = ''
  try {
    if (profile.value.is_following) {
      await api.delete(`/users/${username.value}/follow`)
      profile.value.is_following = false
      profile.value.followers = Math.max(0, profile.value.followers - 1)
    } else {
      await api.post(`/users/${username.value}/follow`)
      profile.value.is_following = true
      profile.value.followers += 1
    }
  } catch (e) {
    followError.value = e instanceof Error ? e.message : '操作失败'
  } finally {
    followLoading.value = false
  }
}

async function loadList(kind: 'followers' | 'following') {
  if (listsLoaded.value[kind]) return
  listLoading.value = true
  listError.value = ''
  try {
    const { data } = await api.get<FollowUser[]>(`/users/${username.value}/${kind}`)
    if (kind === 'followers') followers.value = data ?? []
    else following.value = data ?? []
    listsLoaded.value[kind] = true
  } catch (e) {
    listError.value = e instanceof Error ? e.message : '加载列表失败'
  } finally {
    listLoading.value = false
  }
}

watch(username, () => {
  profile.value = null
  followers.value = []
  following.value = []
  listsLoaded.value = { followers: false, following: false }
  tab.value = 'repos'
  void loadProfile()
})

watch(tab, (t) => {
  if (t === 'followers' || t === 'following') void loadList(t)
})

void loadProfile()
</script>

<template>
  <div class="profile">
    <TopNav variant="landing" />
    <main class="content">
      <p v-if="loading" class="state">加载中...</p>
      <p v-else-if="error" class="state error">{{ error }}</p>

      <template v-else-if="profile">
        <header class="head">
          <span class="avatar">{{ initial(profile.user.username) }}</span>
          <div class="identity">
            <h1 class="username">{{ profile.user.username }}</h1>
            <p v-if="profile.user.full_name" class="fullname">{{ profile.user.full_name }}</p>
            <p v-if="profile.user.bio" class="bio">{{ profile.user.bio }}</p>
            <ul class="facts">
              <li v-if="profile.user.location">📍 {{ profile.user.location }}</li>
              <li v-if="profile.user.website">
                <a v-if="safeWebsite(profile.user.website)" :href="safeWebsite(profile.user.website)" target="_blank" rel="noopener noreferrer">🔗 {{ profile.user.website }}</a>
                <span v-else>🔗 {{ profile.user.website }}</span>
              </li>
            </ul>
          </div>
          <div class="actions">
            <button
              v-if="!isSelf && auth.isLoggedIn"
              class="btn"
              :class="{ primary: !profile.is_following }"
              :disabled="followLoading"
              @click="toggleFollow"
            >
              {{ profile.is_following ? '取消关注' : '关注' }}
            </button>
          </div>
        </header>

        <p v-if="followError" class="error">{{ followError }}</p>

        <nav class="stats">
          <span class="stat"><strong>{{ profile.repos.length }}</strong> 仓库</span>
          <span class="stat"><strong>{{ profile.following }}</strong> 关注</span>
          <span class="stat"><strong>{{ profile.followers }}</strong> 粉丝</span>
        </nav>

        <div class="tabs">
          <button class="tab" :class="{ active: tab === 'repos' }" @click="tab = 'repos'">仓库</button>
          <button class="tab" :class="{ active: tab === 'followers' }" @click="tab = 'followers'">粉丝</button>
          <button class="tab" :class="{ active: tab === 'following' }" @click="tab = 'following'">关注</button>
        </div>

        <section v-if="tab === 'repos'" class="panel">
          <p v-if="!profile.repos.length" class="state">暂无仓库</p>
          <div v-else class="grid">
            <article v-for="r in profile.repos" :key="r.ID" class="card">
              <h3 class="card-title">
                <RouterLink :to="`/${profile.user.username}/${r.Name}`">
                  <span class="owner">{{ profile.user.username }}</span><span class="sep">/</span><span class="name">{{ r.Name }}</span>
                </RouterLink>
              </h3>
              <p class="desc">{{ r.Description || '暂无描述' }}</p>
              <div class="meta">
                <span class="branch" title="默认分支">⎇ {{ r.DefaultBranch || 'main' }}</span>
                <span class="stars" title="Star 数">★ {{ r.StarsCount }}</span>
              </div>
            </article>
          </div>
        </section>

        <section v-else class="panel">
          <p v-if="listLoading" class="state">加载中...</p>
          <p v-else-if="listError" class="state error">{{ listError }}</p>
          <p v-else-if="!(tab === 'followers' ? followers : following).length" class="state">
            暂无{{ tab === 'followers' ? '粉丝' : '关注' }}
          </p>
          <ul v-else class="user-list">
            <li v-for="u in tab === 'followers' ? followers : following" :key="u.id">
              <RouterLink class="user-link" :to="`/users/${u.username}`">
                <span class="avatar sm">{{ initial(u.username) }}</span>
                <span class="user-name">{{ u.username }}</span>
                <span v-if="u.full_name" class="user-fullname">{{ u.full_name }}</span>
              </RouterLink>
            </li>
          </ul>
        </section>
      </template>
    </main>
  </div>
</template>

<style scoped>
.profile {
  min-height: 100vh;
}
.content {
  max-width: 1000px;
  margin: 0 auto;
  padding: 2rem 1.5rem;
}
.head {
  display: flex;
  align-items: flex-start;
  gap: 1.25rem;
  flex-wrap: wrap;
}
.avatar {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 84px;
  height: 84px;
  border-radius: 50%;
  background: var(--primary);
  color: #fff;
  font-size: 2rem;
  font-weight: 600;
  flex-shrink: 0;
}
.avatar.sm {
  width: 34px;
  height: 34px;
  font-size: 0.95rem;
}
.identity {
  flex: 1;
  min-width: 200px;
}
.username {
  margin: 0;
  font-size: 1.6rem;
}
.fullname {
  margin: 0.15rem 0 0;
  color: var(--muted);
  font-size: 1.05rem;
}
.bio {
  margin: 0.75rem 0 0;
  color: var(--color-fg);
  line-height: 1.5;
}
.facts {
  list-style: none;
  padding: 0;
  margin: 0.6rem 0 0;
  display: flex;
  flex-wrap: wrap;
  gap: 1rem;
  color: var(--muted);
  font-size: 0.88rem;
}
.facts a {
  color: var(--primary);
}
.actions {
  align-self: center;
}
.btn {
  padding: 0.5rem 1.2rem;
  border: 1px solid var(--border);
  border-radius: 8px;
  font-size: 0.9rem;
  color: var(--color-fg);
  background: transparent;
  cursor: pointer;
}
.btn.primary {
  background: var(--primary);
  border-color: var(--primary);
  color: #fff;
}
.btn.primary:hover {
  background: var(--primary-hover);
}
.btn:disabled {
  opacity: 0.6;
  cursor: default;
}
.stats {
  display: flex;
  gap: 1.5rem;
  margin: 1.5rem 0;
  padding-bottom: 1rem;
  border-bottom: 1px solid var(--border);
  color: var(--muted);
  font-size: 0.9rem;
}
.stats strong {
  color: var(--color-fg);
  font-size: 1rem;
}
.tabs {
  display: flex;
  gap: 0.5rem;
  margin-bottom: 1.25rem;
}
.tab {
  padding: 0.4rem 1rem;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: transparent;
  color: var(--color-fg);
  font-size: 0.9rem;
  cursor: pointer;
}
.tab.active {
  background: var(--primary);
  border-color: var(--primary);
  color: #fff;
}
.grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 1rem;
}
.card {
  padding: 1.2rem;
  border: 1px solid var(--border);
  border-radius: 10px;
  background: var(--card);
  transition: border-color 0.15s, box-shadow 0.15s;
}
.card:hover {
  border-color: var(--primary);
  box-shadow: 0 2px 10px rgba(59, 130, 246, 0.15);
}
.card-title {
  margin: 0 0 0.5rem;
  font-size: 1.05rem;
}
.card-title a {
  color: var(--primary);
}
.owner {
  color: var(--muted);
}
.sep {
  color: var(--muted);
  margin: 0 0.15rem;
}
.name {
  font-weight: 600;
}
.desc {
  margin: 0 0 0.75rem;
  color: var(--muted);
  font-size: 0.9rem;
  line-height: 1.5;
  min-height: 1.35rem;
}
.meta {
  display: flex;
  align-items: center;
  gap: 1rem;
  flex-wrap: wrap;
  font-size: 0.8rem;
  color: var(--muted);
}
.stars {
  color: var(--accent);
}
.user-list {
  list-style: none;
  margin: 0;
  padding: 0;
}
.user-list li {
  border-bottom: 1px solid var(--border);
}
.user-list li:last-child {
  border-bottom: none;
}
.user-link {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.6rem 0;
}
.user-name {
  font-weight: 600;
  color: var(--color-fg);
}
.user-fullname {
  color: var(--muted);
  font-size: 0.88rem;
}
.state {
  color: var(--muted);
  padding: 1.5rem 0;
}
.error {
  color: var(--accent);
  font-size: 0.9rem;
}
@media (max-width: 640px) {
  .grid {
    grid-template-columns: 1fr;
  }
  .actions {
    width: 100%;
  }
}
</style>
