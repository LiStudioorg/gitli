<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRoute } from 'vue-router'
import { apiStatus, errMessage as httpErrMessage } from '../utils/error'
import AppLayout from '../components/layout/AppLayout.vue'
import api from '../api/client'

interface IPRule {
  id: number
  org_id: number
  cidr: string
  description: string
  enabled: boolean
  created_at: string
}

interface IPAllowlistResponse {
  rules: IPRule[]
  enabled: boolean
}

const route = useRoute()
const org = String(route.params.org ?? '')

/** 组织设置页的 403/404 有专属文案，其余情况交回统一解析。 */
function errMessage(e: unknown): string {
  const status = apiStatus(e)
  if (status === 403) return '没有权限：需要该组织的管理员权限'
  if (status === 404) return '组织不存在'
  return httpErrMessage(e)
}

function fmtTime(v: string | null | undefined): string {
  if (!v) return '-'
  const d = new Date(v)
  return Number.isNaN(d.getTime()) ? v : d.toLocaleString()
}

const rules = ref<IPRule[]>([])
const enabled = ref(false)
const loading = ref(false)
const error = ref('')
const ok = ref('')
const toggling = ref(false)

async function load() {
  loading.value = true
  error.value = ''
  try {
    const { data } = await api.get<IPAllowlistResponse>(
      `/orgs/${encodeURIComponent(org)}/ip-allowlist`,
    )
    rules.value = data.rules ?? []
    enabled.value = data.enabled ?? false
  } catch (e) {
    error.value = errMessage(e)
    rules.value = []
  } finally {
    loading.value = false
  }
}

async function toggleEnabled(next: boolean) {
  const target = next
  if (target && !window.confirm('启用后，只有白名单内的 IP 可以访问该组织，确定启用吗？')) return
  toggling.value = true
  error.value = ''
  ok.value = ''
  try {
    const { data } = await api.put<{ status: string; enabled: boolean }>(
      `/orgs/${encodeURIComponent(org)}/ip-allowlist/toggle`,
      { enabled: target },
    )
    enabled.value = data.enabled ?? target
    ok.value = enabled.value ? 'IP 白名单已启用' : 'IP 白名单已关闭'
  } catch (e) {
    error.value = errMessage(e)
  } finally {
    toggling.value = false
  }
}

const form = reactive({ cidr: '', description: '', enabled: true })
const adding = ref(false)
const formError = ref('')

async function addRule() {
  formError.value = ''
  ok.value = ''
  if (!form.cidr.trim()) {
    formError.value = '请填写 CIDR'
    return
  }
  adding.value = true
  try {
    await api.post(`/orgs/${encodeURIComponent(org)}/ip-allowlist`, {
      cidr: form.cidr.trim(),
      description: form.description.trim(),
      enabled: form.enabled,
    })
    form.cidr = ''
    form.description = ''
    form.enabled = true
    ok.value = '已添加规则'
    await load()
  } catch (e) {
    formError.value = errMessage(e)
  } finally {
    adding.value = false
  }
}

const deletingId = ref<number | null>(null)
const listError = ref('')

async function deleteRule(r: IPRule) {
  if (!window.confirm(`确定要删除规则「${r.cidr}」吗？`)) return
  deletingId.value = r.id
  listError.value = ''
  ok.value = ''
  try {
    await api.delete(`/orgs/${encodeURIComponent(org)}/ip-allowlist/${r.id}`)
    ok.value = '已删除规则'
    await load()
  } catch (e) {
    listError.value = errMessage(e)
  } finally {
    deletingId.value = null
  }
}

onMounted(load)
</script>

<template>
  <AppLayout>
    <div class="mx-auto w-full max-w-[900px]">
        <h2 class="text-lg font-semibold text-[#1f2328] dark:text-[#e6edf3]">组织设置</h2>
        <p class="mb-4 truncate text-sm text-[#59636e]"><code>{{ org }}</code></p>

        <section class="flex flex-col gap-3 rounded-md border border-[#d1d9e0] bg-white p-4 dark:border-[#30363d] dark:bg-[#161b22]">
          <h3>安全 · IP 白名单</h3>

          <p v-if="error" class="text-sm text-[#cf222e]">{{ error }}</p>
          <p v-if="listError" class="text-sm text-[#cf222e]">{{ listError }}</p>
          <p v-if="ok" class="ok">{{ ok }}</p>

          <div class="toggle-row">
            <label class="check">
              <input
                type="checkbox"
                :checked="enabled"
                :disabled="loading || toggling"
                @change="toggleEnabled(($event.target as HTMLInputElement).checked)"
              />
              启用 IP 白名单
            </label>
            <span class="badge" :class="enabled ? 'badge-success' : 'badge-neutral'">
              {{ enabled ? '已启用' : '未启用' }}
            </span>
          </div>
          <p class="warning">
            启用后，只有白名单中列出的 CIDR 才能访问该组织。请先确认你的当前 IP 已加入白名单，否则可能被锁定在外。
          </p>

          <hr class="divider" />

          <h3>规则列表</h3>
          <p v-if="loading" class="muted">加载中…</p>
          <template v-else-if="rules.length">
            <table class="key-table">
              <thead>
                <tr>
                  <th>CIDR</th>
                  <th>说明</th>
                  <th class="center">状态</th>
                  <th>创建时间</th>
                  <th class="center">操作</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="r in rules" :key="r.id">
                  <td><code>{{ r.cidr }}</code></td>
                  <td>{{ r.description || '-' }}</td>
                  <td class="center">
                    <span class="badge" :class="r.enabled ? 'badge-success' : 'badge-neutral'">
                      {{ r.enabled ? '启用' : '停用' }}
                    </span>
                  </td>
                  <td class="muted nowrap">{{ fmtTime(r.created_at) }}</td>
                  <td class="center nowrap">
                    <button
                      class="inline-flex h-8 items-center rounded-md bg-[#cf222e] px-3 text-sm font-semibold text-white hover:bg-[#a40e26] disabled:opacity-50"
                      type="button"
                      :disabled="deletingId === r.id"
                      @click="deleteRule(r)"
                    >
                      {{ deletingId === r.id ? '删除中…' : '删除' }}
                    </button>
                  </td>
                </tr>
              </tbody>
            </table>
          </template>
          <p v-else class="muted">暂无规则</p>

          <hr class="divider" />

          <h3>添加规则</h3>
          <p v-if="formError" class="text-sm text-[#cf222e]">{{ formError }}</p>
          <label>
            CIDR
            <input v-model="form.cidr" placeholder="192.168.1.0/24" autocomplete="off" />
          </label>
          <label>
            说明
            <input v-model="form.description" placeholder="例如：办公室出口 IP" />
          </label>
          <label class="check">
            <input v-model="form.enabled" type="checkbox" />
            添加后立即启用
          </label>
          <button class="inline-flex h-8 items-center rounded-md bg-[#2da44e] px-3 text-sm font-semibold text-white hover:bg-[#2c974b] disabled:opacity-50" type="button" :disabled="adding" @click="addRule">
            {{ adding ? '添加中…' : '添加' }}
          </button>
        </section>
    </div>
  </AppLayout>
</template>
