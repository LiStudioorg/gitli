<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import api from '../../api/client'
import { errMessage } from '../../utils/error'
import EmptyState from '../common/EmptyState.vue'
import SkeletonBlock from '../common/SkeletonBlock.vue'
import MarkdownView from '../common/MarkdownView.vue'
import { useAuthStore } from '../../stores'
import { useToast } from '../../composables/useToast'

interface WikiPage {
  id: number
  slug: string
  title: string
  body: string
}

const props = defineProps<{ owner: string; repo: string }>()
const auth = useAuthStore()
const toast = useToast()

const pages = ref<WikiPage[]>([])
const active = ref<WikiPage | null>(null)
const loading = ref(true)
const error = ref('')
const showForm = ref(false)
const saving = ref(false)
const form = ref({ title: '', body: '' })

const list = computed(() => pages.value)

async function load() {
  loading.value = true
  error.value = ''
  try {
    const { data } = await api.get<WikiPage[] | null>(`/repos/${props.owner}/${props.repo}/wiki`)
    pages.value = data ?? []
    if (!active.value && pages.value.length) active.value = pages.value[0]
  } catch (e) {
    error.value = errMessage(e, '加载百科失败')
    pages.value = []
  } finally {
    loading.value = false
  }
}

async function create() {
  if (!form.value.title.trim()) {
    toast.error('请填写标题')
    return
  }
  saving.value = true
  try {
    const slug = form.value.title
      .trim()
      .toLowerCase()
      .replace(/[^a-z0-9\u4e00-\u9fa5]+/g, '-')
      .replace(/^-|-$/g, '')
    await api.post(`/repos/${props.owner}/${props.repo}/wiki`, {
      slug,
      title: form.value.title.trim(),
      body: form.value.body,
    })
    toast.success('百科页面已创建')
    showForm.value = false
    form.value = { title: '', body: '' }
    active.value = null
    await load()
  } catch (e) {
    toast.error(errMessage(e, '创建百科页面失败'))
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>

<template>
  <section class="wiki">
    <header class="mb-3 flex items-center justify-between gap-2">
      <h3>百科</h3>
      <div class="flex items-center gap-2">
        <button class="inline-flex h-8 items-center rounded-md border border-[#d1d9e0] px-3 text-sm text-[#1f2328] hover:bg-[#f6f8fa] dark:border-[#3d444d] dark:text-[#e6edf3] dark:hover:bg-[#161b22]" type="button" @click="load">刷新</button>
        <button v-if="auth.isLoggedIn" class="inline-flex h-8 items-center rounded-md bg-[#2da44e] px-3 text-sm font-semibold text-white hover:bg-[#2c974b] disabled:opacity-50" type="button" @click="showForm = !showForm">
          {{ showForm ? '取消' : '新建页面' }}
        </button>
      </div>
    </header>

    <form v-if="showForm" class="flex flex-col gap-3 rounded-md border border-[#d1d9e0] bg-white p-4 dark:border-[#3d444d] dark:bg-[#0d1117]" @submit.prevent="create">
      <div class="flex flex-col gap-1.5">
        <label for="wiki-title">标题</label>
        <input id="wiki-title" v-model="form.title" class="h-8 rounded-md border border-[#d1d9e0] bg-[#f6f8fa] px-2.5 text-sm text-[#1f2328] outline-none placeholder:text-[#59636e] focus:border-[#0969da] focus:bg-white focus:ring-2 focus:ring-[#0969da]/25 dark:border-[#3d444d] dark:bg-[#0d1117] dark:text-[#e6edf3]" placeholder="例如 安装指南" />
      </div>
      <div class="flex flex-col gap-1.5">
        <label for="wiki-body">内容（Markdown）</label>
        <textarea id="wiki-body" v-model="form.body" class="h-8 rounded-md border border-[#d1d9e0] bg-[#f6f8fa] px-2.5 text-sm text-[#1f2328] outline-none placeholder:text-[#59636e] focus:border-[#0969da] focus:bg-white focus:ring-2 focus:ring-[#0969da]/25 dark:border-[#3d444d] dark:bg-[#0d1117] dark:text-[#e6edf3]" rows="8" placeholder="# 标题" />
      </div>
      <button class="inline-flex h-8 items-center rounded-md bg-[#2da44e] px-3 text-sm font-semibold text-white hover:bg-[#2c974b] disabled:opacity-50" type="submit" :disabled="saving">
        <span v-if="saving" class="h-3.5 w-3.5 animate-spin rounded-full border-2 border-white/40 border-t-white" aria-hidden="true" />
        {{ saving ? '保存中…' : '保存页面' }}
      </button>
    </form>

    <div v-if="loading" class="layout">
      <SkeletonBlock :lines="6" />
    </div>
    <p v-else-if="error" class="rounded-md border border-[#ff818266] bg-[#ffebe9] px-3 py-2 text-sm text-[#cf222e]">{{ error }}</p>
    <EmptyState
      v-else-if="!list.length"
      icon="📖"
      title="百科还是空的"
      description="用 Markdown 记录项目文档、安装步骤与设计说明。"
      action-label="创建第一个页面"
      :show-action="auth.isLoggedIn"
      @action="showForm = true"
    />
    <div v-else class="layout">
      <nav class="nav" aria-label="百科页面">
        <button
          v-for="p in list"
          :key="p.id"
          type="button"
          class="nav-item"
          :class="{ active: active?.id === p.id }"
          @click="active = p"
        >
          {{ p.title }}
        </button>
      </nav>
      <article v-if="active" class="rounded-md border border-[#d1d9e0] bg-white p-4 dark:border-[#3d444d] dark:bg-[#0d1117]">
        <h2>{{ active.title }}</h2>
        <MarkdownView :source="active.body" />
      </article>
    </div>
  </section>
</template>
