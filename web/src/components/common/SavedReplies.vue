<script setup lang="ts">
import { onMounted, ref } from 'vue'
import api from '../../api/client'

interface SavedReply {
  id: number
  title: string
  body: string
}

const emit = defineEmits<{
  (e: 'insert', body: string): void
}>()

const open = ref(false)
const manageOpen = ref(false)
const replies = ref<SavedReply[]>([])
const loading = ref(false)
const error = ref('')

const editingId = ref<number | null>(null)
const formTitle = ref('')
const formBody = ref('')
const saving = ref(false)
const formError = ref('')

async function load() {
  loading.value = true
  error.value = ''
  try {
    const { data } = await api.get<SavedReply[]>('/user/saved-replies')
    replies.value = data ?? []
  } catch (e) {
    error.value = e instanceof Error ? e.message : '加载快速回复失败'
    replies.value = []
  } finally {
    loading.value = false
  }
}

function toggleDropdown() {
  open.value = !open.value
  if (open.value) void load()
}

function insert(reply: SavedReply) {
  emit('insert', reply.body)
  open.value = false
}

function resetForm() {
  editingId.value = null
  formTitle.value = ''
  formBody.value = ''
  formError.value = ''
}

function startEdit(reply: SavedReply) {
  editingId.value = reply.id
  formTitle.value = reply.title
  formBody.value = reply.body
  formError.value = ''
}

async function save() {
  if (!formTitle.value.trim() || !formBody.value.trim()) {
    formError.value = '标题与内容不能为空'
    return
  }
  saving.value = true
  formError.value = ''
  try {
    if (editingId.value === null) {
      await api.post('/user/saved-replies', { title: formTitle.value, body: formBody.value })
    } else {
      await api.patch(`/user/saved-replies/${editingId.value}`, {
        title: formTitle.value,
        body: formBody.value,
      })
    }
    resetForm()
    await load()
  } catch (e) {
    formError.value = e instanceof Error ? e.message : '保存失败'
  } finally {
    saving.value = false
  }
}

async function remove(reply: SavedReply) {
  if (!window.confirm(`确定删除快速回复「${reply.title}」吗？`)) return
  error.value = ''
  try {
    await api.delete(`/user/saved-replies/${reply.id}`)
    if (editingId.value === reply.id) resetForm()
    await load()
  } catch (e) {
    error.value = e instanceof Error ? e.message : '删除失败'
  }
}

function toggleManage() {
  manageOpen.value = !manageOpen.value
  if (manageOpen.value) {
    open.value = false
    void load()
  } else {
    resetForm()
  }
}

onMounted(() => {
  void load()
})
</script>

<template>
  <div class="saved-replies">
    <div class="bar">
      <div class="dropdown">
        <button class="chip" type="button" @click="toggleDropdown">
          快速回复 ▾
        </button>
        <div v-if="open" class="menu">
          <p v-if="loading" class="menu-state">加载中...</p>
          <p v-else-if="error" class="menu-error">{{ error }}</p>
          <p v-else-if="!replies.length" class="menu-state">暂无快速回复</p>
          <ul v-else class="menu-list">
            <li v-for="r in replies" :key="r.id">
              <button class="menu-item" type="button" @click="insert(r)">
                <span class="menu-title">{{ r.title }}</span>
                <span class="menu-preview">{{ r.body }}</span>
              </button>
            </li>
          </ul>
        </div>
      </div>
      <button class="chip" type="button" @click="toggleManage">
        {{ manageOpen ? '收起管理' : '管理快速回复' }}
      </button>
    </div>

    <div v-if="manageOpen" class="manage">
      <p v-if="error" class="error">{{ error }}</p>
      <ul v-if="replies.length" class="manage-list">
        <li v-for="r in replies" :key="r.id" class="manage-row">
          <span class="manage-title">{{ r.title }}</span>
          <div class="manage-actions">
            <button class="chip" type="button" @click="startEdit(r)">编辑</button>
            <button class="chip danger" type="button" @click="remove(r)">删除</button>
          </div>
        </li>
      </ul>
      <p v-else-if="!loading" class="py-2 text-xs text-[#59636e]">暂无快速回复</p>

      <form class="form" @submit.prevent="save">
        <h4>{{ editingId === null ? '新建快速回复' : '编辑快速回复' }}</h4>
        <input v-model="formTitle" class="form-input" type="text" placeholder="标题" />
        <textarea v-model="formBody" class="form-input" rows="3" placeholder="内容" />
        <p v-if="formError" class="error">{{ formError }}</p>
        <div class="form-actions">
          <button class="chip primary" type="submit" :disabled="saving">
            {{ saving ? '保存中...' : '保存' }}
          </button>
          <button v-if="editingId !== null" class="chip" type="button" @click="resetForm">取消</button>
        </div>
      </form>
    </div>
  </div>
</template>
