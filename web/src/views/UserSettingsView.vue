<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import axios from 'axios'
import TopNav from '../components/layout/TopNav.vue'
import Sidebar from '../components/layout/Sidebar.vue'
import api from '../api/client'
import { useAuthStore } from '../stores'

interface SSHKey {
  ID?: number
  id?: number
  UserID?: number
  user_id?: number
  Title?: string
  title?: string
  Fingerprint?: string
  fingerprint?: string
  PublicKey?: string
  public_key?: string
  Verified?: boolean
  verified?: boolean
  LastUsedAt?: string | null
  last_used_at?: string | null
  CreatedAt?: string
  created_at?: string
}

interface AccessToken {
  ID?: number
  id?: number
  Name?: string
  name?: string
  Kind?: string
  kind?: string
  Scopes?: string[]
  scopes?: string[]
  LastUsedAt?: string | null
  last_used_at?: string | null
  ExpiresAt?: string | null
  expires_at?: string | null
  CreatedAt?: string
  created_at?: string
}

type TabKey = 'profile' | 'password' | 'keys' | 'tokens' | 'totp'

interface Tab {
  key: TabKey
  label: string
}

const tabs: Tab[] = [
  { key: 'profile', label: '个人资料' },
  { key: 'password', label: '密码' },
  { key: 'keys', label: 'SSH 密钥' },
  { key: 'tokens', label: '访问令牌' },
  { key: 'totp', label: '两步验证' },
]

const auth = useAuthStore()
const active = ref<TabKey>('profile')

function errMessage(e: unknown): string {
  if (axios.isAxiosError(e)) {
    const data = e.response?.data as { error?: string } | undefined
    if (data?.error) return data.error
    if (e.response?.status) return `请求失败（HTTP ${e.response.status}）`
    return e.message
  }
  return e instanceof Error ? e.message : '未知错误'
}

// --- profile ---
const profile = reactive({ full_name: '', bio: '', website: '', location: '' })
const profileError = ref('')
const profileOk = ref('')
const profileSaving = ref(false)

function fillProfile() {
  const u = auth.user
  profile.full_name = u?.full_name ?? ''
  profile.bio = u?.bio ?? ''
  profile.website = u?.website ?? ''
  profile.location = u?.location ?? ''
}

async function saveProfile() {
  profileSaving.value = true
  profileError.value = ''
  profileOk.value = ''
  try {
    await api.patch('/user/me', {
      full_name: profile.full_name,
      bio: profile.bio,
      website: profile.website,
      location: profile.location,
    })
    await auth.fetchMe()
    fillProfile()
    profileOk.value = '已保存'
  } catch (e) {
    profileError.value = errMessage(e)
  } finally {
    profileSaving.value = false
  }
}

// --- password ---
const pw = reactive({ current_password: '', new_password: '', totp_code: '' })
const pwError = ref('')
const pwOk = ref('')
const pwSaving = ref(false)

async function changePassword() {
  pwSaving.value = true
  pwError.value = ''
  pwOk.value = ''
  try {
    await api.post('/user/me/password', {
      current_password: pw.current_password,
      new_password: pw.new_password,
      totp_code: pw.totp_code,
    })
    pw.current_password = ''
    pw.new_password = ''
    pw.totp_code = ''
    pwOk.value = '密码已更新'
  } catch (e) {
    pwError.value = errMessage(e)
  } finally {
    pwSaving.value = false
  }
}

// --- ssh keys ---
const keys = ref<SSHKey[]>([])
const keysLoading = ref(false)
const keysError = ref('')
const newKey = reactive({ title: '', public_key: '' })
const keyAdding = ref(false)

function keyId(k: SSHKey): number | undefined {
  return k.ID ?? k.id
}
function keyTitle(k: SSHKey): string {
  return k.Title ?? k.title ?? ''
}
function keyFingerprint(k: SSHKey): string {
  return k.Fingerprint ?? k.fingerprint ?? ''
}

async function loadKeys() {
  keysLoading.value = true
  keysError.value = ''
  try {
    const { data } = await api.get<SSHKey[]>('/user/me/keys')
    keys.value = data ?? []
  } catch (e) {
    keysError.value = errMessage(e)
  } finally {
    keysLoading.value = false
  }
}

async function addKey() {
  if (!newKey.title || !newKey.public_key) return
  keyAdding.value = true
  keysError.value = ''
  try {
    await api.post('/user/me/keys', {
      title: newKey.title,
      public_key: newKey.public_key,
    })
    newKey.title = ''
    newKey.public_key = ''
    await loadKeys()
  } catch (e) {
    keysError.value = errMessage(e)
  } finally {
    keyAdding.value = false
  }
}

async function deleteKey(id: number | undefined) {
  if (id == null) return
  keysError.value = ''
  try {
    await api.delete(`/user/me/keys/${id}`)
    await loadKeys()
  } catch (e) {
    keysError.value = errMessage(e)
  }
}

// --- tokens ---
const tokens = ref<AccessToken[]>([])
const tokensLoading = ref(false)
const tokensError = ref('')
const newToken = reactive({ name: '', kind: 'classic' })
const tokenCreating = ref(false)
const createdToken = ref('')
const copied = ref(false)

function tokenId(t: AccessToken): number | undefined {
  return t.ID ?? t.id
}
function tokenName(t: AccessToken): string {
  return t.Name ?? t.name ?? ''
}
function tokenKind(t: AccessToken): string {
  return t.Kind ?? t.kind ?? ''
}
function tokenCreatedAt(t: AccessToken): string {
  const v = t.CreatedAt ?? t.created_at
  if (!v) return '-'
  const d = new Date(v)
  return Number.isNaN(d.getTime()) ? v : d.toLocaleString()
}

async function loadTokens() {
  tokensLoading.value = true
  tokensError.value = ''
  try {
    const { data } = await api.get<AccessToken[]>('/user/me/tokens')
    tokens.value = data ?? []
  } catch (e) {
    tokensError.value = errMessage(e)
  } finally {
    tokensLoading.value = false
  }
}

async function createToken() {
  if (!newToken.name) return
  tokenCreating.value = true
  tokensError.value = ''
  copied.value = false
  try {
    const { data } = await api.post<{ id: number; token: string; name: string }>(
      '/user/me/tokens',
      { name: newToken.name, kind: newToken.kind },
    )
    createdToken.value = data.token
    newToken.name = ''
    await loadTokens()
  } catch (e) {
    tokensError.value = errMessage(e)
  } finally {
    tokenCreating.value = false
  }
}

async function copyToken() {
  try {
    await navigator.clipboard.writeText(createdToken.value)
    copied.value = true
  } catch {
    copied.value = false
  }
}

async function revokeToken(id: number | undefined) {
  if (id == null) return
  tokensError.value = ''
  try {
    await api.delete(`/user/me/tokens/${id}`)
    await loadTokens()
  } catch (e) {
    tokensError.value = errMessage(e)
  }
}

// --- totp ---
const totpSecret = ref('')
const totpURL = ref('')
const totpCode = ref('')
const totpPassword = ref('')
const totpRecoveryCodes = ref<string[]>([])
const totpError = ref('')
const totpOk = ref('')
const totpBusy = ref(false)

async function totpBegin() {
  totpBusy.value = true
  totpError.value = ''
  totpOk.value = ''
  totpRecoveryCodes.value = []
  try {
    const { data } = await api.post<{ secret: string; otpauth_url: string }>(
      '/user/me/totp/begin',
    )
    totpSecret.value = data.secret
    totpURL.value = data.otpauth_url
  } catch (e) {
    totpError.value = errMessage(e)
  } finally {
    totpBusy.value = false
  }
}

async function totpEnable() {
  if (!totpCode.value) return
  totpBusy.value = true
  totpError.value = ''
  totpOk.value = ''
  try {
    const { data } = await api.post<{ recovery_codes: string[] }>(
      '/user/me/totp/enable',
      { code: totpCode.value },
    )
    totpRecoveryCodes.value = data.recovery_codes ?? []
    totpCode.value = ''
    totpSecret.value = ''
    totpURL.value = ''
    await auth.fetchMe()
    totpOk.value = '两步验证已启用'
  } catch (e) {
    totpError.value = errMessage(e)
  } finally {
    totpBusy.value = false
  }
}

async function totpDisable() {
  totpBusy.value = true
  totpError.value = ''
  totpOk.value = ''
  try {
    await api.post('/user/me/totp/disable', {
      code: totpCode.value,
      password: totpPassword.value,
    })
    totpCode.value = ''
    totpPassword.value = ''
    await auth.fetchMe()
    totpOk.value = '两步验证已关闭'
  } catch (e) {
    totpError.value = errMessage(e)
  } finally {
    totpBusy.value = false
  }
}

function selectTab(key: TabKey) {
  active.value = key
  if (key === 'keys' && keys.value.length === 0 && !keysLoading.value) loadKeys()
  if (key === 'tokens' && tokens.value.length === 0 && !tokensLoading.value) loadTokens()
}

onMounted(async () => {
  if (!auth.loaded) await auth.fetchMe()
  fillProfile()
})
</script>

<template>
  <div class="settings-page">
    <TopNav variant="app" />
    <div class="body">
      <Sidebar />
      <main class="settings">
        <h2>用户设置</h2>
        <nav class="tabs" role="tablist">
          <button
            v-for="t in tabs"
            :key="t.key"
            class="tab"
            :class="{ active: active === t.key }"
            type="button"
            role="tab"
            :aria-selected="active === t.key"
            @click="selectTab(t.key)"
          >
            {{ t.label }}
          </button>
        </nav>

        <!-- 个人资料 -->
        <section v-if="active === 'profile'" class="card">
          <p v-if="profileError" class="error">{{ profileError }}</p>
          <p v-if="profileOk" class="ok">{{ profileOk }}</p>
          <label>用户名<input :value="auth.user?.username" disabled /></label>
          <label>邮箱<input :value="auth.user?.email" disabled /></label>
          <label>姓名<input v-model="profile.full_name" /></label>
          <label>简介<textarea v-model="profile.bio" rows="3" /></label>
          <label>网站<input v-model="profile.website" type="url" placeholder="https://" /></label>
          <label>位置<input v-model="profile.location" /></label>
          <button class="primary" :disabled="profileSaving" @click="saveProfile">
            {{ profileSaving ? '保存中…' : '保存' }}
          </button>
        </section>

        <!-- 密码 -->
        <section v-else-if="active === 'password'" class="card">
          <p v-if="pwError" class="error">{{ pwError }}</p>
          <p v-if="pwOk" class="ok">{{ pwOk }}</p>
          <label>当前密码<input v-model="pw.current_password" type="password" autocomplete="current-password" /></label>
          <label>新密码<input v-model="pw.new_password" type="password" autocomplete="new-password" /></label>
          <label>两步验证码（如已启用）<input v-model="pw.totp_code" inputmode="numeric" /></label>
          <button class="primary" :disabled="pwSaving" @click="changePassword">
            {{ pwSaving ? '提交中…' : '修改密码' }}
          </button>
        </section>

        <!-- SSH 密钥 -->
        <section v-else-if="active === 'keys'" class="card">
          <p v-if="keysError" class="error">{{ keysError }}</p>
          <h3>已登记的密钥</h3>
          <p v-if="keysLoading" class="muted">加载中…</p>
          <template v-else-if="keys.length">
            <ul class="list">
              <li v-for="k in keys" :key="keyId(k)">
                <div class="li-main">
                  <strong>{{ keyTitle(k) }}</strong>
                  <code class="fingerprint">{{ keyFingerprint(k) }}</code>
                </div>
                <button class="danger" @click="deleteKey(keyId(k))">删除</button>
              </li>
            </ul>
          </template>
          <p v-else class="muted">暂无数据</p>
          <h3>添加密钥</h3>
          <label>标题<input v-model="newKey.title" placeholder="例如：笔记本" /></label>
          <label>公钥<textarea v-model="newKey.public_key" rows="4" placeholder="ssh-ed25519 AAAA..." /></label>
          <button class="primary" :disabled="keyAdding" @click="addKey">
            {{ keyAdding ? '添加中…' : '添加密钥' }}
          </button>
        </section>

        <!-- 访问令牌 -->
        <section v-else-if="active === 'tokens'" class="card">
          <p v-if="tokensError" class="error">{{ tokensError }}</p>
          <div v-if="createdToken" class="token-callout">
            <p class="token-warn">请立即复制并妥善保存，令牌仅显示这一次。</p>
            <div class="token-row">
              <code class="token-value">{{ createdToken }}</code>
              <button type="button" @click="copyToken">{{ copied ? '已复制' : '复制' }}</button>
            </div>
            <button class="link" type="button" @click="createdToken = ''">我已保存，关闭</button>
          </div>
          <h3>已创建的令牌</h3>
          <p v-if="tokensLoading" class="muted">加载中…</p>
          <template v-else-if="tokens.length">
            <ul class="list">
              <li v-for="t in tokens" :key="tokenId(t)">
                <div class="li-main">
                  <strong>{{ tokenName(t) }}</strong>
                  <span class="muted">{{ tokenKind(t) }} · {{ tokenCreatedAt(t) }}</span>
                </div>
                <button class="danger" @click="revokeToken(tokenId(t))">吊销</button>
              </li>
            </ul>
          </template>
          <p v-else class="muted">暂无数据</p>
          <h3>新建令牌</h3>
          <label>名称<input v-model="newToken.name" placeholder="例如：CI 部署" /></label>
          <label>类型
            <select v-model="newToken.kind">
              <option value="classic">经典令牌</option>
              <option value="fine_grained">细粒度令牌</option>
            </select>
          </label>
          <button class="primary" :disabled="tokenCreating" @click="createToken">
            {{ tokenCreating ? '生成中…' : '生成令牌' }}
          </button>
        </section>

        <!-- 两步验证 -->
        <section v-else class="card">
          <p v-if="totpError" class="error">{{ totpError }}</p>
          <p v-if="totpOk" class="ok">{{ totpOk }}</p>
          <p>
            当前状态：
            <strong>{{ auth.user?.totp_enabled ? '已启用' : '未启用' }}</strong>
          </p>

          <template v-if="!auth.user?.totp_enabled">
            <button class="primary" :disabled="totpBusy" @click="totpBegin">
              {{ totpBusy ? '请稍候…' : '开始设置' }}
            </button>
            <template v-if="totpSecret">
              <label>密钥<code class="secret">{{ totpSecret }}</code></label>
              <label>otpauth URL<input :value="totpURL" disabled /></label>
              <label>输入认证器中的 6 位验证码<input v-model="totpCode" inputmode="numeric" /></label>
              <button class="primary" :disabled="totpBusy || !totpCode" @click="totpEnable">
                {{ totpBusy ? '验证中…' : '启用两步验证' }}
              </button>
            </template>
          </template>

          <template v-else>
            <label>验证码<input v-model="totpCode" inputmode="numeric" /></label>
            <label>当前密码<input v-model="totpPassword" type="password" /></label>
            <button class="danger" :disabled="totpBusy" @click="totpDisable">
              {{ totpBusy ? '关闭中…' : '关闭两步验证' }}
            </button>
          </template>

          <div v-if="totpRecoveryCodes.length" class="token-callout">
            <p class="token-warn">恢复码仅显示这一次，请立即保存：</p>
            <ul class="recovery">
              <li v-for="c in totpRecoveryCodes" :key="c"><code>{{ c }}</code></li>
            </ul>
          </div>
        </section>
      </main>
    </div>
  </div>
</template>

<style scoped>
.body {
  display: flex;
  min-height: calc(100vh - 57px);
}
.settings {
  flex: 1;
  max-width: 720px;
  padding: 1.5rem;
  min-width: 0;
}
.settings h2 {
  margin-top: 0;
}
.tabs {
  display: flex;
  gap: 0.25rem;
  border-bottom: 1px solid var(--border);
  margin-bottom: 1.25rem;
  flex-wrap: wrap;
}
.tab {
  padding: 0.5rem 0.9rem;
  border: none;
  background: transparent;
  color: var(--muted);
  font-size: 0.95rem;
  border-bottom: 2px solid transparent;
  margin-bottom: -1px;
}
.tab:hover {
  color: var(--color-fg);
}
.tab.active {
  color: var(--primary);
  border-bottom-color: var(--primary);
  font-weight: 600;
}
.card {
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: 1.25rem;
  background: var(--card);
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}
.card h3 {
  margin: 0.5rem 0 0;
  font-size: 1rem;
}
.card h3:first-child {
  margin-top: 0;
}
label {
  display: flex;
  flex-direction: column;
  gap: 0.3rem;
  font-size: 0.9rem;
}
input,
textarea,
select {
  padding: 0.5rem;
  border-radius: 6px;
  border: 1px solid var(--border);
  background: var(--color-bg);
  color: var(--color-fg);
  font-family: inherit;
}
button.primary {
  align-self: flex-start;
  padding: 0.45rem 1rem;
  border-radius: 6px;
  border: 1px solid var(--primary);
  background: var(--primary);
  color: #fff;
  font-size: 0.9rem;
}
button.primary:hover:not(:disabled) {
  background: var(--primary-hover);
}
button.danger {
  padding: 0.3rem 0.7rem;
  border-radius: 6px;
  border: 1px solid #d1242f;
  background: transparent;
  color: #d1242f;
  font-size: 0.85rem;
}
button:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
.list {
  list-style: none;
  padding: 0;
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}
.list li {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 0.6rem 0.8rem;
}
.li-main {
  display: flex;
  flex-direction: column;
  gap: 0.15rem;
  min-width: 0;
}
.fingerprint {
  font-size: 0.78rem;
  color: var(--muted);
  word-break: break-all;
}
.token-callout {
  border: 1px solid var(--accent);
  background: var(--accent-soft);
  border-radius: 8px;
  padding: 0.85rem;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}
.token-warn {
  margin: 0;
  color: #7a4b00;
  font-size: 0.85rem;
  font-weight: 600;
}
.token-row {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
}
.token-value {
  flex: 1;
  min-width: 200px;
  padding: 0.5rem;
  border-radius: 6px;
  border: 1px solid var(--border);
  background: var(--card);
  color: var(--color-fg);
  word-break: break-all;
}
.link {
  align-self: flex-start;
  background: none;
  border: none;
  color: var(--primary);
  padding: 0;
  font-size: 0.85rem;
}
.recovery {
  list-style: none;
  padding: 0;
  margin: 0;
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(120px, 1fr));
  gap: 0.35rem;
}
.recovery code {
  display: block;
  padding: 0.3rem;
  background: var(--card);
  border-radius: 4px;
  text-align: center;
}
.secret {
  padding: 0.4rem 0.6rem;
  border: 1px dashed var(--border);
  border-radius: 6px;
  word-break: break-all;
}
.error {
  margin: 0;
  color: #d1242f;
  background: rgba(209, 36, 47, 0.08);
  border: 1px solid rgba(209, 36, 47, 0.4);
  border-radius: 8px;
  padding: 0.6rem 0.8rem;
  font-size: 0.9rem;
}
.ok {
  margin: 0;
  color: #1a7f37;
  font-size: 0.9rem;
}
.muted {
  color: var(--muted);
  font-size: 0.9rem;
}
@media (max-width: 768px) {
  .body {
    flex-direction: column;
  }
}
</style>
