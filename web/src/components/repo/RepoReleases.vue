<script setup lang="ts">
import { onMounted, ref } from 'vue'
import api from '../../api/client'
import { errMessage } from '../../utils/error'
import SkeletonBlock from '../common/SkeletonBlock.vue'
import EmptyState from '../common/EmptyState.vue'
import MarkdownView from '../common/MarkdownView.vue'
import { useAuthStore } from '../../stores'
import { useToast } from '../../composables/useToast'

interface Release {
  id: number
  tag: string
  name: string
  body: string
  is_draft: boolean
  is_prerelease: boolean
  created_at: string
}

const props = defineProps<{ owner: string; repo: string }>()
const auth = useAuthStore()
const toast = useToast()

const releases = ref<Release[]>([])
const loading = ref(true)
const error = ref('')
const creating = ref(false)
const showForm = ref(false)
const form = ref({ tag: '', name: '', body: '' })

function dateLabel(raw: string): string {
  const d = new Date(raw.endsWith('Z') || raw.includes('+') ? raw : raw.replace(' ', 'T') + 'Z')
  return Number.isNaN(d.getTime()) ? raw : d.toLocaleDateString()
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    const { data } = await api.get<Release[] | null>(`/repos/${props.owner}/${props.repo}/releases`)
    releases.value = data ?? []
  } catch (e) {
    error.value = errMessage(e, '加载发布列表失败')
    releases.value = []
  } finally {
    loading.value = false
  }
}

async function create() {
  if (!form.value.tag.trim()) {
    toast.error('请填写标签名')
    return
  }
  creating.value = true
  try {
    await api.post(`/repos/${props.owner}/${props.repo}/releases`, {
      tag: form.value.tag.trim(),
      name: form.value.name.trim() || form.value.tag.trim(),
      body: form.value.body,
    })
    toast.success('发布已创建')
    showForm.value = false
    form.value = { tag: '', name: '', body: '' }
    await load()
  } catch (e) {
    toast.error(errMessage(e, '创建发布失败'))
  } finally {
    creating.value = false
  }
}

function archiveUrl(tag: string, format: 'zip' | 'tar.gz') {
  return `/api/v1/repos/${props.owner}/${props.repo}/archive/${format}?ref=${encodeURIComponent(tag)}`
}

onMounted(load)
</script>

<template>
  <section class="releases">
    <header class="mb-3 flex items-center justify-between gap-2">
      <h3>发布</h3>
      <div class="flex items-center gap-2">
        <button class="inline-flex h-8 items-center rounded-md border border-[#d1d9e0] px-3 text-sm text-[#1f2328] hover:bg-[#f6f8fa] dark:border-[#3d444d] dark:text-[#e6edf3] dark:hover:bg-[#161b22]" type="button" @click="load">刷新</button>
        <button v-if="auth.isLoggedIn" class="inline-flex h-8 items-center rounded-md bg-[#2da44e] px-3 text-sm font-semibold text-white hover:bg-[#2c974b] disabled:opacity-50" type="button" @click="showForm = !showForm">
          {{ showForm ? '取消' : '新建发布' }}
        </button>
      </div>
    </header>

    <form v-if="showForm" class="flex flex-col gap-3 rounded-md border border-[#d1d9e0] bg-white p-4 dark:border-[#3d444d] dark:bg-[#0d1117]" @submit.prevent="create">
      <div class="flex flex-col gap-1.5">
        <label for="rel-tag">标签（必填）</label>
        <input id="rel-tag" v-model="form.tag" class="h-8 rounded-md border border-[#d1d9e0] bg-[#f6f8fa] px-2.5 text-sm text-[#1f2328] outline-none placeholder:text-[#59636e] focus:border-[#0969da] focus:bg-white focus:ring-2 focus:ring-[#0969da]/25 dark:border-[#3d444d] dark:bg-[#0d1117] dark:text-[#e6edf3]" placeholder="例如 v1.0.0" />
      </div>
      <div class="flex flex-col gap-1.5">
        <label for="rel-name">标题</label>
        <input id="rel-name" v-model="form.name" class="h-8 rounded-md border border-[#d1d9e0] bg-[#f6f8fa] px-2.5 text-sm text-[#1f2328] outline-none placeholder:text-[#59636e] focus:border-[#0969da] focus:bg-white focus:ring-2 focus:ring-[#0969da]/25 dark:border-[#3d444d] dark:bg-[#0d1117] dark:text-[#e6edf3]" placeholder="留空则使用标签名" />
      </div>
      <div class="flex flex-col gap-1.5">
        <label for="rel-body">说明</label>
        <textarea id="rel-body" v-model="form.body" class="h-8 rounded-md border border-[#d1d9e0] bg-[#f6f8fa] px-2.5 text-sm text-[#1f2328] outline-none placeholder:text-[#59636e] focus:border-[#0969da] focus:bg-white focus:ring-2 focus:ring-[#0969da]/25 dark:border-[#3d444d] dark:bg-[#0d1117] dark:text-[#e6edf3]" rows="4" placeholder="本次发布的主要变更" />
      </div>
      <button class="inline-flex h-8 items-center rounded-md bg-[#2da44e] px-3 text-sm font-semibold text-white hover:bg-[#2c974b] disabled:opacity-50" type="submit" :disabled="creating">
        <span v-if="creating" class="h-3.5 w-3.5 animate-spin rounded-full border-2 border-white/40 border-t-white" aria-hidden="true" />
        {{ creating ? '创建中…' : '创建发布' }}
      </button>
    </form>

    <div v-if="loading" class="flex flex-col gap-2">
      <SkeletonBlock v-for="i in 3" :key="i" :lines="2" />
    </div>
    <p v-else-if="error" class="rounded-md border border-[#ff818266] bg-[#ffebe9] px-3 py-2 text-sm text-[#cf222e]">{{ error }}</p>
    <EmptyState
      v-else-if="!releases.length"
      icon="🏷"
      title="还没有发布版本"
      description="打一个 tag 并创建发布，用户即可下载源码归档。"
      action-label="新建发布"
      :show-action="auth.isLoggedIn"
      @action="showForm = true"
    />
    <ul v-else class="flex flex-col gap-2">
      <li v-for="r in releases" :key="r.id" class="rounded-md border border-[#d1d9e0] bg-white p-4 dark:border-[#3d444d] dark:bg-[#0d1117]">
        <div class="release-head">
          <span class="tag">{{ r.tag }}</span>
          <span v-if="r.is_prerelease" class="pill">预发布</span>
          <span v-if="r.is_draft" class="pill draft">草稿</span>
          <span class="name">{{ r.name }}</span>
          <time class="when">{{ dateLabel(r.created_at) }}</time>
        </div>
        <MarkdownView v-if="r.body" :source="r.body" class="body" />
        <div class="downloads">
          <a class="inline-flex h-8 items-center rounded-md border border-[#d1d9e0] px-3 text-sm text-[#1f2328] hover:bg-[#f6f8fa] dark:border-[#3d444d] dark:text-[#e6edf3] dark:hover:bg-[#161b22]" :href="archiveUrl(r.tag, 'zip')">源码 .zip</a>
          <a class="inline-flex h-8 items-center rounded-md border border-[#d1d9e0] px-3 text-sm text-[#1f2328] hover:bg-[#f6f8fa] dark:border-[#3d444d] dark:text-[#e6edf3] dark:hover:bg-[#161b22]" :href="archiveUrl(r.tag, 'tar.gz')">源码 .tar.gz</a>
        </div>
      </li>
    </ul>
  </section>
</template>
