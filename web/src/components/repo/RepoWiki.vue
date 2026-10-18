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
    <header class="head">
      <h3>百科</h3>
      <div class="actions">
        <button class="ui-btn" type="button" @click="load">刷新</button>
        <button v-if="auth.isLoggedIn" class="ui-btn ui-btn-primary" type="button" @click="showForm = !showForm">
          {{ showForm ? '取消' : '新建页面' }}
        </button>
      </div>
    </header>

    <form v-if="showForm" class="ui-card ui-form" @submit.prevent="create">
      <div class="ui-field">
        <label for="wiki-title">标题</label>
        <input id="wiki-title" v-model="form.title" class="ui-input" placeholder="例如 安装指南" />
      </div>
      <div class="ui-field">
        <label for="wiki-body">内容（Markdown）</label>
        <textarea id="wiki-body" v-model="form.body" class="ui-input" rows="8" placeholder="# 标题" />
      </div>
      <button class="ui-btn ui-btn-primary" type="submit" :disabled="saving">
        <span v-if="saving" class="ui-spinner" aria-hidden="true" />
        {{ saving ? '保存中…' : '保存页面' }}
      </button>
    </form>

    <div v-if="loading" class="layout">
      <SkeletonBlock :lines="6" />
    </div>
    <p v-else-if="error" class="ui-alert ui-alert-error">{{ error }}</p>
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
      <article v-if="active" class="ui-card page">
        <h2>{{ active.title }}</h2>
        <MarkdownView :source="active.body" />
      </article>
    </div>
  </section>
</template>

<style scoped>
.wiki {
  display: flex;
  flex-direction: column;
  gap: 0.9rem;
}
.head {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.head h3 {
  margin: 0;
}
.actions {
  display: flex;
  gap: 0.5rem;
}
.layout {
  display: grid;
  grid-template-columns: 200px minmax(0, 1fr);
  gap: 1rem;
  align-items: start;
}
.nav {
  display: flex;
  flex-direction: column;
  gap: 0.2rem;
}
.nav-item {
  text-align: left;
  border: none;
  background: transparent;
  color: var(--muted);
  padding: 0.45rem 0.6rem;
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-size: 0.88rem;
}
.nav-item:hover {
  background: var(--hover-bg);
  color: var(--color-fg);
}
.nav-item.active {
  background: var(--primary-soft);
  color: var(--primary);
  font-weight: 600;
}
.page {
  padding: 1.25rem 1.4rem;
}
@media (max-width: 800px) {
  .layout {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
