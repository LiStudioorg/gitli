<script setup lang="ts">
import { formatDate } from '../../utils/datetime'
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
  return formatDate(raw)
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
    <header class="head">
      <h3>发布</h3>
      <div class="actions">
        <button class="ui-btn" type="button" @click="load">刷新</button>
        <button v-if="auth.isLoggedIn" class="ui-btn ui-btn-primary" type="button" @click="showForm = !showForm">
          {{ showForm ? '取消' : '新建发布' }}
        </button>
      </div>
    </header>

    <form v-if="showForm" class="ui-card ui-form create" @submit.prevent="create">
      <div class="ui-field">
        <label for="rel-tag">标签（必填）</label>
        <input id="rel-tag" v-model="form.tag" class="ui-input" placeholder="例如 v1.0.0" />
      </div>
      <div class="ui-field">
        <label for="rel-name">标题</label>
        <input id="rel-name" v-model="form.name" class="ui-input" placeholder="留空则使用标签名" />
      </div>
      <div class="ui-field">
        <label for="rel-body">说明</label>
        <textarea id="rel-body" v-model="form.body" class="ui-input" rows="4" placeholder="本次发布的主要变更" />
      </div>
      <button class="ui-btn ui-btn-primary" type="submit" :disabled="creating">
        <span v-if="creating" class="ui-spinner" aria-hidden="true" />
        {{ creating ? '创建中…' : '创建发布' }}
      </button>
    </form>

    <div v-if="loading" class="list">
      <SkeletonBlock v-for="i in 3" :key="i" :lines="2" />
    </div>
    <p v-else-if="error" class="ui-alert ui-alert-error">{{ error }}</p>
    <EmptyState
      v-else-if="!releases.length"
      icon="🏷"
      title="还没有发布版本"
      description="打一个 tag 并创建发布，用户即可下载源码归档。"
      action-label="新建发布"
      :show-action="auth.isLoggedIn"
      @action="showForm = true"
    />
    <ul v-else class="list">
      <li v-for="r in releases" :key="r.id" class="ui-card release">
        <div class="release-head">
          <span class="tag">{{ r.tag }}</span>
          <span v-if="r.is_prerelease" class="pill">预发布</span>
          <span v-if="r.is_draft" class="pill draft">草稿</span>
          <span class="name">{{ r.name }}</span>
          <time class="when">{{ dateLabel(r.created_at) }}</time>
        </div>
        <MarkdownView v-if="r.body" :source="r.body" class="body" />
        <div class="downloads">
          <a class="ui-btn" :href="archiveUrl(r.tag, 'zip')">源码 .zip</a>
          <a class="ui-btn" :href="archiveUrl(r.tag, 'tar.gz')">源码 .tar.gz</a>
        </div>
      </li>
    </ul>
  </section>
</template>

<style scoped>
.releases {
  display: flex;
  flex-direction: column;
  gap: 0.9rem;
}
.head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
}
.head h3 {
  margin: 0;
}
.actions {
  display: flex;
  gap: 0.5rem;
}
.create {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}
.list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}
.release {
  padding: 1rem 1.1rem;
}
.release-head {
  display: flex;
  align-items: center;
  gap: 0.55rem;
  flex-wrap: wrap;
}
.tag {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-weight: 700;
  color: var(--primary);
  background: var(--primary-soft);
  padding: 0.15rem 0.5rem;
  border-radius: 999px;
  font-size: 0.85rem;
}
.name {
  font-weight: 600;
}
.when {
  margin-left: auto;
  color: var(--muted);
  font-size: 0.8rem;
}
.pill {
  font-size: 0.72rem;
  padding: 0.1rem 0.45rem;
  border-radius: 999px;
  border: 1px solid var(--border);
  color: var(--muted);
}
.body {
  margin-top: 0.6rem;
}
.downloads {
  display: flex;
  gap: 0.5rem;
  margin-top: 0.75rem;
  flex-wrap: wrap;
}
.downloads .ui-btn {
  font-size: 0.8rem;
  padding: 0.3rem 0.7rem;
  text-decoration: none;
}
</style>
