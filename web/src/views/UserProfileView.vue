<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import TopNav from '../components/layout/TopNav.vue'
import ErrorAlert from '../components/common/ErrorAlert.vue'
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
    // Defensive normalisation: a null repos array would crash template
    // evaluation and leave the whole page blank.
    profile.value = {
      ...data,
      repos: data.repos ?? [],
      followers: data.followers ?? 0,
      following: data.following ?? 0,
    }
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
  <div class="min-h-screen bg-[#f6f8fa] dark:bg-[#0d1117]">
    <TopNav variant="landing" />
    <main class="mx-auto w-full max-w-[1280px] px-4 py-6 md:px-6">
      <div v-if="loading" class="flex flex-col gap-4">
        <div class="flex gap-4 rounded-md border border-[#d1d9e0] bg-white p-6 dark:border-[#30363d] dark:bg-[#161b22]">
          <div class="h-20 w-20 shrink-0 animate-pulse rounded-full bg-[#d1d9e0] dark:bg-[#30363d]" />
          <div class="flex flex-1 flex-col gap-2">
            <div class="h-5 w-40 animate-pulse rounded bg-[#d1d9e0] dark:bg-[#30363d]" />
            <div class="h-4 w-64 animate-pulse rounded bg-[#d1d9e0] dark:bg-[#30363d]" />
            <div class="h-4 w-32 animate-pulse rounded bg-[#d1d9e0] dark:bg-[#30363d]" />
          </div>
        </div>
        <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
          <div v-for="i in 4" :key="i" class="h-28 animate-pulse rounded-md border border-[#d1d9e0] bg-white dark:border-[#30363d] dark:bg-[#161b22]" />
        </div>
      </div>

      <ErrorAlert v-else-if="error" title="加载用户资料失败" :message="error" :dismissible="false" />

      <template v-else-if="profile">
        <!-- 资料头 -->
        <section class="mb-6 flex flex-wrap items-start gap-5 rounded-md border border-[#d1d9e0] bg-white p-6 dark:border-[#30363d] dark:bg-[#161b22]">
          <span class="flex h-20 w-20 shrink-0 items-center justify-center rounded-full bg-[#ea580c] text-2xl font-semibold text-white">
            {{ initial(profile.user.username) }}
          </span>
          <div class="min-w-0 flex-1">
            <div class="flex flex-wrap items-center gap-3">
              <h1 class="truncate text-2xl font-semibold text-[#1f2328] dark:text-[#e6edf3]">{{ profile.user.username }}</h1>
              <button
                v-if="!isSelf && auth.isLoggedIn"
                class="h-8 rounded-md border border-[#d1d9e0] px-3 text-sm text-[#1f2328] hover:bg-[#f6f8fa] dark:border-[#30363d] dark:text-[#e6edf3] dark:hover:bg-[#21262d]"
                :disabled="followLoading"
                @click="toggleFollow"
              >
                {{ profile.is_following ? '取消关注' : '关注' }}
              </button>
            </div>
            <p v-if="profile.user.full_name" class="truncate text-sm text-[#59636e]">{{ profile.user.full_name }}</p>
            <p v-if="profile.user.bio" class="mt-2 break-words text-sm text-[#1f2328] dark:text-[#e6edf3]">{{ profile.user.bio }}</p>
            <ul class="mt-2 flex flex-wrap gap-4 text-sm text-[#59636e]">
              <li v-if="profile.user.location" class="truncate">📍 {{ profile.user.location }}</li>
              <li v-if="profile.user.website" class="min-w-0 truncate">
                <a v-if="safeWebsite(profile.user.website)" :href="safeWebsite(profile.user.website)" target="_blank" rel="noopener noreferrer" class="text-[#0969da] hover:underline">🔗 {{ profile.user.website }}</a>
                <span v-else>🔗 {{ profile.user.website }}</span>
              </li>
            </ul>
            <p v-if="followError" class="mt-2 text-sm text-[#cf222e]">{{ followError }}</p>
          </div>
        </section>

        <!-- 统计 -->
        <div class="mb-4 flex flex-wrap gap-6 text-sm text-[#59636e]">
          <span><strong class="text-[#1f2328] dark:text-[#e6edf3]">{{ profile.repos.length }}</strong> 仓库</span>
          <span><strong class="text-[#1f2328] dark:text-[#e6edf3]">{{ profile.following }}</strong> 关注</span>
          <span><strong class="text-[#1f2328] dark:text-[#e6edf3]">{{ profile.followers }}</strong> 粉丝</span>
        </div>

        <!-- Tab -->
        <nav class="mb-4 flex gap-1 border-b border-[#d1d9e0] dark:border-[#30363d]">
          <button
            v-for="t in [{ k: 'repos', label: '仓库' }, { k: 'followers', label: '粉丝' }, { k: 'following', label: '关注' }]"
            :key="t.k"
            type="button"
            class="h-11 border-b-2 px-3 text-sm"
            :class="tab === t.k
              ? 'border-[#ea580c] font-semibold text-[#1f2328] dark:text-[#e6edf3]'
              : 'border-transparent text-[#59636e] hover:rounded-t-md hover:border-[#d1d9e0] hover:bg-[#f6f8fa] dark:hover:bg-[#21262d]'"
            @click="tab = t.k as 'repos' | 'followers' | 'following'"
          >
            {{ t.label }}
          </button>
        </nav>

        <section v-if="tab === 'repos'">
          <div v-if="!profile.repos.length" class="rounded-md border border-dashed border-[#d1d9e0] bg-white px-4 py-10 text-center dark:border-[#30363d] dark:bg-[#161b22]">
            <p class="text-sm font-semibold text-[#1f2328] dark:text-[#e6edf3]">还没有仓库</p>
            <p class="mt-1 text-sm text-[#59636e]">该用户尚未创建任何仓库。</p>
          </div>
          <div v-else class="grid grid-cols-1 gap-4 md:grid-cols-2">
            <article
              v-for="r in profile.repos"
              :key="r.ID"
              class="flex flex-col rounded-md border border-[#d1d9e0] bg-white p-4 dark:border-[#30363d] dark:bg-[#161b22]"
            >
              <h3 class="truncate text-base font-semibold">
                <RouterLink :to="`/${profile.user.username}/${r.Name}`" class="text-[#0969da] hover:underline">
                  {{ profile.user.username }}<span class="text-[#59636e]">/</span>{{ r.Name }}
                </RouterLink>
              </h3>
              <p class="mt-1 line-clamp-2 min-h-[2.5rem] text-sm text-[#59636e]">{{ r.Description || '暂无描述' }}</p>
              <div class="mt-3 flex items-center gap-4 text-xs text-[#59636e]">
                <span class="truncate">⎇ {{ r.DefaultBranch || 'main' }}</span>
                <span class="shrink-0">★ {{ r.StarsCount ?? 0 }}</span>
              </div>
            </article>
          </div>
        </section>

        <section v-else>
          <p v-if="listLoading" class="py-8 text-center text-sm text-[#59636e]">加载中…</p>
          <ErrorAlert v-else-if="listError" title="加载失败" :message="listError" :dismissible="false" />
          <p v-else-if="!(tab === 'followers' ? followers : following).length" class="py-8 text-center text-sm text-[#59636e]">
            暂无{{ tab === 'followers' ? '粉丝' : '关注' }}
          </p>
          <ul v-else class="divide-y divide-[#d1d9e0] overflow-hidden rounded-md border border-[#d1d9e0] bg-white dark:divide-[#30363d] dark:border-[#30363d] dark:bg-[#161b22]">
            <li v-for="u in tab === 'followers' ? followers : following" :key="u.id">
              <RouterLink class="flex items-center gap-3 px-4 py-3 hover:bg-[#f6f8fa] dark:hover:bg-[#21262d]" :to="`/users/${u.username}`">
                <span class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-[#ea580c] text-xs font-semibold text-white">{{ initial(u.username) }}</span>
                <span class="truncate text-sm font-semibold text-[#1f2328] dark:text-[#e6edf3]">{{ u.username }}</span>
                <span v-if="u.full_name" class="truncate text-sm text-[#59636e]">{{ u.full_name }}</span>
              </RouterLink>
            </li>
          </ul>
        </section>
      </template>
    </main>
  </div>
</template>
