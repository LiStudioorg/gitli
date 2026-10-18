<script setup lang="ts">
import { ref } from 'vue'
import api from '../../api/client'

const props = defineProps<{
  owner: string
  repo: string
  archived: boolean
}>()

const emit = defineEmits<{
  (e: 'updated', payload: { name?: string; archived?: boolean }): void
}>()

const base = () => `/repos/${props.owner}/${props.repo}`

const newName = ref('')
const savingName = ref(false)
const nameError = ref('')
const nameOk = ref('')

const archiving = ref(false)
const archiveError = ref('')

async function rename() {
  nameError.value = ''
  nameOk.value = ''
  const name = newName.value.trim()
  if (!name) {
    nameError.value = '仓库名称不能为空'
    return
  }
  if (name === props.repo) {
    nameError.value = '新名称与当前名称相同'
    return
  }
  savingName.value = true
  try {
    const { data } = await api.post<{ status: string; name: string }>(`${base()}/rename`, { name })
    newName.value = ''
    nameOk.value = '已重命名，请刷新'
    emit('updated', { name: data?.name ?? name })
  } catch (e) {
    const status = (e as { response?: { status?: number } })?.response?.status
    if (status === 401 || status === 403) {
      nameError.value = '需要仓库管理员权限'
    } else {
      nameError.value = '重命名失败'
    }
  } finally {
    savingName.value = false
  }
}

async function toggleArchive() {
  archiveError.value = ''
  const target = !props.archived
  if (target && !window.confirm('确定要归档该仓库吗？归档后仓库将变为只读。')) return
  archiving.value = true
  try {
    const { data } = await api.post<{ status: string; archived: boolean }>(`${base()}/archive`, {
      archived: target,
    })
    emit('updated', { archived: data?.archived ?? target })
  } catch (e) {
    const status = (e as { response?: { status?: number } })?.response?.status
    if (status === 401 || status === 403) {
      archiveError.value = '需要仓库管理员权限'
    } else {
      archiveError.value = target ? '归档失败' : '解除归档失败'
    }
  } finally {
    archiving.value = false
  }
}
</script>

<template>
  <section class="danger-zone">
    <div class="panel-head">
      <h3>危险操作</h3>
    </div>

    <form class="action-card" @submit.prevent="rename">
      <h4>仓库重命名</h4>
      <p class="hint">修改后旧地址将失效，克隆地址需同步更新。</p>
      <div class="row">
        <input v-model="newName" type="text" placeholder="新的仓库名称" class="form-input" />
        <button class="chip primary" type="submit" :disabled="savingName">
          {{ savingName ? '保存中...' : '保存' }}
        </button>
      </div>
      <p v-if="nameError" class="error">{{ nameError }}</p>
      <p v-if="nameOk" class="ok">{{ nameOk }}</p>
    </form>

    <div class="action-card">
      <h4>{{ archived ? '解除归档' : '归档仓库' }}</h4>
      <p class="hint">
        {{ archived ? '解除归档后仓库恢复读写。' : '归档后仓库将变为只读，无法推送。' }}
      </p>
      <p v-if="archiveError" class="error">{{ archiveError }}</p>
      <button
        class="chip"
        :class="archived ? 'primary' : 'danger'"
        type="button"
        :disabled="archiving"
        @click="toggleArchive"
      >
        {{ archiving ? '处理中...' : archived ? '解除归档' : '归档仓库' }}
      </button>
    </div>
  </section>
</template>

<style scoped>
.panel-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 1rem;
}
.panel-head h3 {
  margin: 0;
}
.action-card {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 1rem;
  margin-bottom: 1.25rem;
  background: var(--card);
}
.action-card h4 {
  margin: 0;
  font-size: 0.9rem;
  color: var(--muted);
}
.hint {
  margin: 0;
  font-size: 0.85rem;
  color: var(--muted);
}
.row {
  display: flex;
  gap: 0.5rem;
  align-items: center;
}
.form-input {
  flex: 1;
  padding: 0.5rem 0.7rem;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--color-bg);
  color: var(--color-fg);
  font: inherit;
}
.chip {
  padding: 0.4rem 0.8rem;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--card);
  color: var(--color-fg);
}
.chip:hover {
  background: var(--hover-bg);
  border-color: var(--primary);
  color: var(--primary);
}
.chip.primary {
  background: var(--primary);
  border-color: var(--primary);
  color: #fff;
}
.chip.primary:hover {
  background: var(--primary-hover);
  color: #fff;
}
.chip.danger {
  border-color: var(--accent);
  color: var(--accent);
  align-self: flex-start;
}
.chip.danger:hover {
  background: var(--accent-soft);
  color: var(--accent);
}
.action-card .chip.primary {
  align-self: flex-start;
}
.error {
  margin: 0;
  color: var(--accent);
  font-size: 0.85rem;
}
.ok {
  margin: 0;
  color: var(--success);
  font-size: 0.85rem;
}
</style>
