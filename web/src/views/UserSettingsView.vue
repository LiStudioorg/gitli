<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { errMessage } from '../utils/error'
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
  CanAuthenticate?: boolean
  can_authenticate?: boolean
  CanSign?: boolean
  can_sign?: boolean
  Verified?: boolean
  verified?: boolean
  LastUsedAt?: string | null
  last_used_at?: string | null
  CreatedAt?: string
  created_at?: string
}

interface GPGKey {
  ID?: number
  id?: number
  KeyType?: string
  key_type?: string
  Title?: string
  title?: string
  Fingerprint?: string
  fingerprint?: string
  PublicKey?: string
  public_key?: string
  Verified?: boolean
  verified?: boolean
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

interface DeletedRepo {
  id: number
  name: string
  deleted_at: string
}

interface ActivityLog {
  id: number
  event: string
  resource?: string
  result: string
  target_type?: string
  target_name?: string
  actor_ip?: string
  created_at?: string
}

interface Session {
  id: number
  ip?: string
  user_agent?: string
  created_at?: string
  last_seen_at?: string
  expires_at?: string
  current?: boolean
}

interface OAuthApp {
  id: number
  owner_id?: number
  name: string
  client_id: string
  redirect_uris: string[]
  scopes: string[]
  created_at?: string
}

interface OAuthAuthorization {
  id: number
  user_id?: number
  app_id: number
  app_name: string
  client_id: string
  scopes: string[]
  created_at?: string
}

type TabKey =
  | 'profile'
  | 'password'
  | 'keys'
  | 'tokens'
  | 'oauth'
  | 'totp'
  | 'deleted'
  | 'security'
  | 'general'

interface Tab {
  key: TabKey
  label: string
}

const tabs: Tab[] = [
  { key: 'profile', label: '个人资料' },
  { key: 'password', label: '密码' },
  { key: 'keys', label: 'SSH / GPG 密钥' },
  { key: 'tokens', label: '访问令牌' },
  { key: 'oauth', label: 'OAuth 应用' },
  { key: 'totp', label: '两步验证' },
  { key: 'deleted', label: '已删除的仓库' },
  { key: 'security', label: '安全活动' },
  { key: 'general', label: '通用' },
]

import { localeOptions, setLocale, currentLocale, type AppLocale } from '../i18n'

const auth = useAuthStore()
const selectedLocale = ref(currentLocale())

function changeLocale(v: AppLocale) {
  setLocale(v)
  selectedLocale.value = v
}

type NotificationFrequency = 'realtime' | 'daily' | 'weekly' | 'off'

const notificationFrequencyOptions: { value: NotificationFrequency; label: string }[] = [
  { value: 'realtime', label: '实时' },
  { value: 'daily', label: '每日' },
  { value: 'weekly', label: '每周' },
  { value: 'off', label: '关闭' },
]
const notificationFrequency = ref<NotificationFrequency>('realtime')
const notifLoading = ref(false)
const notifSaving = ref(false)
const notifError = ref('')
const notifOk = ref('')
const notifLoaded = ref(false)

async function loadNotificationFrequency() {
  if (notifLoading.value) return
  notifLoading.value = true
  notifError.value = ''
  try {
    const { data } = await api.get<{ frequency: NotificationFrequency }>(
      '/user/notification-frequency',
    )
    if (data?.frequency) notificationFrequency.value = data.frequency
    notifLoaded.value = true
  } catch (e) {
    notifError.value = errMessage(e)
  } finally {
    notifLoading.value = false
  }
}

async function saveNotificationFrequency() {
  notifSaving.value = true
  notifError.value = ''
  notifOk.value = ''
  try {
    await api.put('/user/notification-frequency', { frequency: notificationFrequency.value })
    notifOk.value = '通知频率已保存'
  } catch (e) {
    notifError.value = errMessage(e)
  } finally {
    notifSaving.value = false
  }
}
const active = ref<TabKey>('profile')


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
const newKey = reactive({
  title: '',
  public_key: '',
  can_authenticate: true,
  can_sign: false,
})
const keyAdding = ref(false)

const editOpen = ref(false)
const editBusy = ref(false)
const editError = ref('')
const editTarget = ref<SSHKey | null>(null)
const editForm = reactive({ can_authenticate: true, can_sign: false })

function keyId(k: SSHKey): number | undefined {
  return k.ID ?? k.id
}
function keyTitle(k: SSHKey): string {
  return k.Title ?? k.title ?? ''
}
function keyFingerprint(k: SSHKey): string {
  return k.Fingerprint ?? k.fingerprint ?? ''
}
function keyCanAuth(k: SSHKey): boolean {
  return k.CanAuthenticate ?? k.can_authenticate ?? false
}
function keyCanSign(k: SSHKey): boolean {
  return k.CanSign ?? k.can_sign ?? false
}
function fmtTime(v: string | null | undefined): string {
  if (!v) return '-'
  const d = new Date(v)
  return Number.isNaN(d.getTime()) ? v : d.toLocaleString()
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
  keysError.value = ''
  if (!newKey.title || !newKey.public_key) {
    keysError.value = '请填写标题与公钥'
    return
  }
  if (!newKey.can_authenticate && !newKey.can_sign) {
    keysError.value = '至少选择一种用途（认证或签名）'
    return
  }
  keyAdding.value = true
  try {
    await api.post('/user/me/keys', {
      title: newKey.title,
      public_key: newKey.public_key,
      can_authenticate: newKey.can_authenticate,
      can_sign: newKey.can_sign,
    })
    newKey.title = ''
    newKey.public_key = ''
    newKey.can_authenticate = true
    newKey.can_sign = false
    await loadKeys()
  } catch (e) {
    keysError.value = errMessage(e)
  } finally {
    keyAdding.value = false
  }
}

function openEdit(k: SSHKey) {
  editTarget.value = k
  editForm.can_authenticate = keyCanAuth(k)
  editForm.can_sign = keyCanSign(k)
  editError.value = ''
  editOpen.value = true
}

function closeEdit() {
  editOpen.value = false
  editTarget.value = null
  editError.value = ''
}

async function saveEdit() {
  if (!editTarget.value) return
  if (!editForm.can_authenticate && !editForm.can_sign) {
    editError.value = '至少选择一种用途（认证或签名）'
    return
  }
  const id = keyId(editTarget.value)
  if (id == null) return
  editBusy.value = true
  editError.value = ''
  try {
    await api.patch(`/user/me/keys/${id}`, {
      can_authenticate: editForm.can_authenticate,
      can_sign: editForm.can_sign,
    })
    closeEdit()
    await loadKeys()
  } catch (e) {
    editError.value = errMessage(e)
  } finally {
    editBusy.value = false
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

// --- gpg keys ---
const gpgKeys = ref<GPGKey[]>([])
const gpgLoading = ref(false)
const gpgError = ref('')
const newGpg = reactive({ title: '', public_key: '', key_type: 'gpg' })
const gpgAdding = ref(false)

function gpgId(k: GPGKey): number | undefined {
  return k.ID ?? k.id
}
function gpgTitle(k: GPGKey): string {
  return k.Title ?? k.title ?? ''
}
function gpgFingerprint(k: GPGKey): string {
  return k.Fingerprint ?? k.fingerprint ?? ''
}

async function loadGpg() {
  gpgLoading.value = true
  gpgError.value = ''
  try {
    const { data } = await api.get<GPGKey[]>('/user/me/gpg-keys')
    gpgKeys.value = data ?? []
  } catch (e) {
    gpgError.value = errMessage(e)
  } finally {
    gpgLoading.value = false
  }
}

async function addGpg() {
  gpgError.value = ''
  if (!newGpg.title || !newGpg.public_key) {
    gpgError.value = '请填写标题与公钥'
    return
  }
  gpgAdding.value = true
  try {
    await api.post('/user/me/gpg-keys', {
      title: newGpg.title,
      public_key: newGpg.public_key,
      key_type: newGpg.key_type,
    })
    newGpg.title = ''
    newGpg.public_key = ''
    newGpg.key_type = 'gpg'
    await loadGpg()
  } catch (e) {
    gpgError.value = errMessage(e)
  } finally {
    gpgAdding.value = false
  }
}

async function deleteGpg(id: number | undefined) {
  if (id == null) return
  gpgError.value = ''
  try {
    await api.delete(`/user/me/gpg-keys/${id}`)
    await loadGpg()
  } catch (e) {
    gpgError.value = errMessage(e)
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
  if (
    !window.confirm(
      `将为「${newToken.name}」生成一个访问令牌。令牌明文只显示一次，请务必立即保存。确定继续？`,
    )
  ) {
    return
  }
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
  if (!window.confirm('吊销后该令牌立即失效，使用它的脚本/CI 将无法再访问。确定吊销？')) return
  tokensError.value = ''
  try {
    await api.delete(`/user/me/tokens/${id}`)
    await loadTokens()
  } catch (e) {
    tokensError.value = errMessage(e)
  }
}

// --- oauth apps ---
const oauthScopes: { value: string; label: string }[] = [
  { value: 'read:user', label: 'read:user（读取用户资料）' },
  { value: 'read:repo', label: 'read:repo（读取仓库）' },
  { value: 'write:repo', label: 'write:repo（写入仓库）' },
]

const oauthApps = ref<OAuthApp[]>([])
const oauthAppsLoading = ref(false)
const oauthAppsError = ref('')
const newApp = reactive({ name: '', redirect_uris: '', scopes: ['read:user'] as string[] })
const appCreating = ref(false)
const createdSecret = ref('')
const createdAppName = ref('')
const secretCopied = ref(false)

const oauthAuths = ref<OAuthAuthorization[]>([])
const oauthAuthsLoading = ref(false)
const oauthAuthsError = ref('')
const revokingAuth = ref<number | null>(null)

async function loadOAuthApps() {
  oauthAppsLoading.value = true
  oauthAppsError.value = ''
  try {
    const { data } = await api.get<OAuthApp[]>('/user/oauth-apps')
    oauthApps.value = data ?? []
  } catch (e) {
    oauthAppsError.value = errMessage(e)
  } finally {
    oauthAppsLoading.value = false
  }
}

async function loadOAuthAuths() {
  oauthAuthsLoading.value = true
  oauthAuthsError.value = ''
  try {
    const { data } = await api.get<OAuthAuthorization[]>('/user/oauth-authorizations')
    oauthAuths.value = data ?? []
  } catch (e) {
    oauthAuthsError.value = errMessage(e)
  } finally {
    oauthAuthsLoading.value = false
  }
}

async function createApp() {
  oauthAppsError.value = ''
  createdSecret.value = ''
  secretCopied.value = false
  if (!newApp.name.trim()) {
    oauthAppsError.value = '请填写应用名称'
    return
  }
  const uris = newApp.redirect_uris
    .split('\n')
    .map((s) => s.trim())
    .filter((s) => s.length > 0)
  if (uris.length === 0) {
    oauthAppsError.value = '请至少填写一个回调地址'
    return
  }
  appCreating.value = true
  try {
    const { data } = await api.post<{ app: OAuthApp; client_secret: string }>(
      '/user/oauth-apps',
      { name: newApp.name.trim(), redirect_uris: uris, scopes: newApp.scopes },
    )
    createdSecret.value = data.client_secret
    createdAppName.value = data.app?.name ?? newApp.name.trim()
    newApp.name = ''
    newApp.redirect_uris = ''
    newApp.scopes = ['read:user']
    await loadOAuthApps()
  } catch (e) {
    oauthAppsError.value = errMessage(e)
  } finally {
    appCreating.value = false
  }
}

async function copySecret() {
  try {
    await navigator.clipboard.writeText(createdSecret.value)
    secretCopied.value = true
  } catch {
    secretCopied.value = false
  }
}

async function deleteApp(id: number) {
  if (!window.confirm('确定要删除该 OAuth 应用吗？已授权的用户将立即失效。')) return
  oauthAppsError.value = ''
  try {
    await api.delete(`/user/oauth-apps/${id}`)
    await loadOAuthApps()
  } catch (e) {
    oauthAppsError.value = errMessage(e)
  }
}

async function revokeAuthorization(appId: number) {
  if (!window.confirm('确定要撤销该应用的授权吗？')) return
  revokingAuth.value = appId
  oauthAuthsError.value = ''
  try {
    await api.delete(`/user/oauth-authorizations/${appId}`)
    await loadOAuthAuths()
  } catch (e) {
    oauthAuthsError.value = errMessage(e)
  } finally {
    revokingAuth.value = null
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
  if (!window.confirm('关闭两步验证会降低账号安全性，确定继续？')) return
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

// --- deleted repos ---
const deletedRepos = ref<DeletedRepo[]>([])
const deletedLoading = ref(false)
const deletedError = ref('')
const deletedOk = ref('')
const restoringId = ref<number | null>(null)

async function loadDeletedRepos() {
  deletedLoading.value = true
  deletedError.value = ''
  try {
    const { data } = await api.get<DeletedRepo[]>('/user/deleted-repos')
    deletedRepos.value = data ?? []
  } catch (e) {
    deletedError.value = errMessage(e)
    deletedRepos.value = []
  } finally {
    deletedLoading.value = false
  }
}

async function restoreRepo(r: DeletedRepo) {
  if (!window.confirm(`确定要恢复仓库「${r.name}」吗？`)) return
  restoringId.value = r.id
  deletedError.value = ''
  deletedOk.value = ''
  try {
    await api.post(`/user/deleted-repos/${r.id}/restore`)
    deletedOk.value = `已恢复「${r.name}」`
    await loadDeletedRepos()
  } catch (e) {
    deletedError.value = errMessage(e)
  } finally {
    restoringId.value = null
  }
}

// --- security activity ---
const ACTIVITY_LIMIT = 25
const activityResultOptions: { value: string; label: string }[] = [
  { value: '', label: '全部' },
  { value: 'success', label: 'success' },
  { value: 'failure', label: 'failure' },
  { value: 'denied', label: 'denied' },
]

const logs = ref<ActivityLog[]>([])
const logsLoading = ref(false)
const logsError = ref('')
const logsPage = ref(0)
const logsHasNext = ref(false)
const logsResult = ref('')

interface ActivityResponse {
  logs: ActivityLog[]
  total?: number
}

async function loadLogs() {
  logsLoading.value = true
  logsError.value = ''
  try {
    const params: Record<string, string | number> = {
      limit: ACTIVITY_LIMIT,
      offset: logsPage.value * ACTIVITY_LIMIT,
    }
    if (logsResult.value) params.result = logsResult.value
    const { data } = await api.get<ActivityResponse>('/user/activity', { params })
    logs.value = data.logs ?? []
    logsHasNext.value = (data.logs?.length ?? 0) === ACTIVITY_LIMIT
  } catch (e) {
    logsError.value = errMessage(e)
  } finally {
    logsLoading.value = false
  }
}

function activityPrev() {
  if (logsPage.value === 0) return
  logsPage.value -= 1
  loadLogs()
}

function activityNext() {
  if (!logsHasNext.value) return
  logsPage.value += 1
  loadLogs()
}

function applyResultFilter() {
  logsPage.value = 0
  loadLogs()
}

function resultClass(result: string): string {
  if (result === 'success') return 'badge-success'
  if (result === 'failure') return 'badge-failure'
  if (result === 'denied') return 'badge-denied'
  return 'badge-neutral'
}

// --- sessions ---
const sessions = ref<Session[]>([])
const sessionsLoading = ref(false)
const sessionsError = ref('')
const sessionsOk = ref('')
const revoking = ref<number | null>(null)
const revokingAll = ref(false)

async function loadSessions() {
  sessionsLoading.value = true
  sessionsError.value = ''
  try {
    const { data } = await api.get<Session[]>('/user/me/sessions')
    sessions.value = data ?? []
  } catch (e) {
    sessionsError.value = errMessage(e)
  } finally {
    sessionsLoading.value = false
  }
}

async function revokeSession(s: Session) {
  if (s.current) return
  if (!window.confirm('确定要注销该设备吗？')) return
  revoking.value = s.id
  sessionsError.value = ''
  sessionsOk.value = ''
  try {
    await api.delete(`/user/me/sessions/${s.id}`)
    sessionsOk.value = '已注销该设备'
    await loadSessions()
  } catch (e) {
    sessionsError.value = errMessage(e)
  } finally {
    revoking.value = null
  }
}

async function revokeOthers() {
  if (!window.confirm('确定要注销除当前设备外的所有设备吗？')) return
  revokingAll.value = true
  sessionsError.value = ''
  sessionsOk.value = ''
  try {
    await api.delete('/user/me/sessions')
    sessionsOk.value = '已注销其他所有设备'
    await loadSessions()
  } catch (e) {
    sessionsError.value = errMessage(e)
  } finally {
    revokingAll.value = false
  }
}

function truncateUA(ua: string | undefined): string {
  if (!ua) return '-'
  return ua.length > 60 ? `${ua.slice(0, 57)}…` : ua
}

function selectTab(key: TabKey) {
  active.value = key
  if (key === 'keys') {
    if (keys.value.length === 0 && !keysLoading.value) loadKeys()
    if (gpgKeys.value.length === 0 && !gpgLoading.value) loadGpg()
  }
  if (key === 'tokens' && tokens.value.length === 0 && !tokensLoading.value) loadTokens()
  if (key === 'oauth') {
    if (oauthApps.value.length === 0 && !oauthAppsLoading.value) loadOAuthApps()
    if (oauthAuths.value.length === 0 && !oauthAuthsLoading.value) loadOAuthAuths()
  }
  if (key === 'deleted' && deletedRepos.value.length === 0 && !deletedLoading.value) {
    loadDeletedRepos()
  }
  if (key === 'security') {
    if (logs.value.length === 0 && !logsLoading.value) loadLogs()
    if (sessions.value.length === 0 && !sessionsLoading.value) loadSessions()
  }
  if (key === 'general' && !notifLoaded.value && !notifLoading.value) {
    notifOk.value = ''
    loadNotificationFrequency()
  }
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

        <!-- SSH / GPG 密钥 -->
        <section v-else-if="active === 'keys'" class="card">
          <h3>SSH 密钥</h3>
          <p v-if="keysError" class="error">{{ keysError }}</p>
          <p v-if="keysLoading" class="muted">加载中…</p>
          <template v-else-if="keys.length">
            <table class="key-table">
              <thead>
                <tr>
                  <th>标题</th>
                  <th>指纹</th>
                  <th class="center">认证</th>
                  <th class="center">签名</th>
                  <th class="center">操作</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="k in keys" :key="keyId(k)">
                  <td>{{ keyTitle(k) }}</td>
                  <td><code class="fingerprint">{{ keyFingerprint(k) }}</code></td>
                  <td class="center">
                    <span class="badge" :class="keyCanAuth(k) ? 'badge-on' : 'badge-off'">
                      {{ keyCanAuth(k) ? '✅' : '❌' }}
                    </span>
                  </td>
                  <td class="center">
                    <span class="badge" :class="keyCanSign(k) ? 'badge-on' : 'badge-off'">
                      {{ keyCanSign(k) ? '✅' : '❌' }}
                    </span>
                  </td>
                  <td class="center nowrap">
                    <button class="ghost" type="button" @click="openEdit(k)">编辑</button>
                    <button class="danger" type="button" @click="deleteKey(keyId(k))">删除</button>
                  </td>
                </tr>
              </tbody>
            </table>
          </template>
          <p v-else class="muted">暂无 SSH 密钥</p>

          <h3>添加 SSH 密钥</h3>
          <label>标题<input v-model="newKey.title" placeholder="例如：笔记本" /></label>
          <label>公钥<textarea v-model="newKey.public_key" rows="4" placeholder="ssh-ed25519 AAAA..." /></label>
          <label class="check">
            <input v-model="newKey.can_authenticate" type="checkbox" />
            允许用于认证（Git clone/push/pull）
          </label>
          <label class="check">
            <input v-model="newKey.can_sign" type="checkbox" />
            允许用于提交签名
          </label>
          <button class="primary" :disabled="keyAdding" @click="addKey">
            {{ keyAdding ? '添加中…' : '添加密钥' }}
          </button>

          <hr class="divider" />

          <h3>GPG 密钥</h3>
          <p v-if="gpgError" class="error">{{ gpgError }}</p>
          <p v-if="gpgLoading" class="muted">加载中…</p>
          <template v-else-if="gpgKeys.length">
            <table class="key-table">
              <thead>
                <tr>
                  <th>标题</th>
                  <th>指纹</th>
                  <th>上传时间</th>
                  <th class="center">操作</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="g in gpgKeys" :key="gpgId(g)">
                  <td>{{ gpgTitle(g) }}</td>
                  <td><code class="fingerprint">{{ gpgFingerprint(g) }}</code></td>
                  <td class="muted nowrap">{{ fmtTime(g.CreatedAt ?? g.created_at) }}</td>
                  <td class="center">
                    <button class="danger" type="button" @click="deleteGpg(gpgId(g))">删除</button>
                  </td>
                </tr>
              </tbody>
            </table>
          </template>
          <p v-else class="muted">暂无 GPG 密钥</p>

          <h3>添加 GPG 密钥</h3>
          <label>标题<input v-model="newGpg.title" placeholder="例如：工作密钥" /></label>
          <label>公钥
            <textarea
              v-model="newGpg.public_key"
              rows="5"
              placeholder="-----BEGIN PGP PUBLIC KEY BLOCK-----"
            />
          </label>
          <button class="primary" :disabled="gpgAdding" @click="addGpg">
            {{ gpgAdding ? '添加中…' : '添加 GPG 密钥' }}
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

        <!-- OAuth 应用 -->
        <section v-else-if="active === 'oauth'" class="card">
          <div v-if="createdSecret" class="token-callout">
            <p class="token-warn">
              应用「{{ createdAppName }}」已创建。Client Secret 仅显示这一次，请立即复制保存。
            </p>
            <div class="token-row">
              <code class="token-value">{{ createdSecret }}</code>
              <button type="button" @click="copySecret">{{ secretCopied ? '已复制' : '复制' }}</button>
            </div>
            <button class="link" type="button" @click="createdSecret = ''">我已保存，关闭</button>
          </div>

          <h3>我的应用</h3>
          <p v-if="oauthAppsError" class="error">{{ oauthAppsError }}</p>
          <p v-if="oauthAppsLoading" class="muted">加载中…</p>
          <template v-else-if="oauthApps.length">
            <table class="key-table">
              <thead>
                <tr>
                  <th>应用</th>
                  <th>回调地址 / 权限</th>
                  <th class="center">操作</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="a in oauthApps" :key="a.id">
                  <td>
                    <strong>{{ a.name }}</strong>
                    <div class="muted"><code class="fingerprint">{{ a.client_id }}</code></div>
                    <div class="muted nowrap">创建于 {{ fmtTime(a.created_at) }}</div>
                  </td>
                  <td>
                    <div v-for="u in a.redirect_uris" :key="u" class="muted">
                      <code class="fingerprint">{{ u }}</code>
                    </div>
                    <div class="scope-chips">
                      <span v-for="s in a.scopes" :key="s" class="scope-chip">{{ s }}</span>
                    </div>
                  </td>
                  <td class="center nowrap">
                    <button class="danger" type="button" @click="deleteApp(a.id)">删除</button>
                  </td>
                </tr>
              </tbody>
            </table>
          </template>
          <p v-else class="muted">暂无 OAuth 应用</p>

          <h3>创建 OAuth 应用</h3>
          <label>应用名称<input v-model="newApp.name" placeholder="例如：我的集成" /></label>
          <label>
            回调地址（每行一个）
            <textarea
              v-model="newApp.redirect_uris"
              rows="3"
              placeholder="https://example.com/oauth/callback"
            />
          </label>
          <div class="scope-group">
            <span class="muted">权限范围</span>
            <label v-for="s in oauthScopes" :key="s.value" class="check">
              <input v-model="newApp.scopes" type="checkbox" :value="s.value" />
              {{ s.label }}
            </label>
          </div>
          <button class="primary" :disabled="appCreating" @click="createApp">
            {{ appCreating ? '创建中…' : '创建应用' }}
          </button>

          <hr class="divider" />

          <h3>已授权应用</h3>
          <p v-if="oauthAuthsError" class="error">{{ oauthAuthsError }}</p>
          <p v-if="oauthAuthsLoading" class="muted">加载中…</p>
          <template v-else-if="oauthAuths.length">
            <table class="key-table">
              <thead>
                <tr>
                  <th>应用</th>
                  <th>权限</th>
                  <th>授权时间</th>
                  <th class="center">操作</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="g in oauthAuths" :key="g.id">
                  <td>
                    <strong>{{ g.app_name }}</strong>
                    <div class="muted"><code class="fingerprint">{{ g.client_id }}</code></div>
                  </td>
                  <td>
                    <span v-for="s in g.scopes" :key="s" class="scope-chip">{{ s }}</span>
                  </td>
                  <td class="muted nowrap">{{ fmtTime(g.created_at) }}</td>
                  <td class="center nowrap">
                    <button
                      class="danger"
                      type="button"
                      :disabled="revokingAuth === g.app_id"
                      @click="revokeAuthorization(g.app_id)"
                    >
                      {{ revokingAuth === g.app_id ? '撤销中…' : '撤销' }}
                    </button>
                  </td>
                </tr>
              </tbody>
            </table>
          </template>
          <p v-else class="muted">暂无已授权应用</p>
        </section>

        <!-- 两步验证 -->
        <section v-else-if="active === 'totp'" class="card">
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

        <!-- 已删除的仓库 -->
        <section v-else-if="active === 'deleted'" class="card">
          <h3>已删除的仓库</h3>
          <span class="deleted-hint">已删除的仓库处于「已删除」状态，可在此恢复。</span>
          <p v-if="deletedError" class="error">{{ deletedError }}</p>
          <p v-if="deletedOk" class="ok">{{ deletedOk }}</p>
          <p v-if="deletedLoading" class="muted">加载中…</p>
          <template v-else-if="deletedRepos.length">
            <table class="key-table">
              <thead>
                <tr>
                  <th>仓库名称</th>
                  <th>删除时间</th>
                  <th class="center">操作</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="r in deletedRepos" :key="r.id">
                  <td>{{ r.name }}</td>
                  <td class="muted nowrap">{{ fmtTime(r.deleted_at) }}</td>
                  <td class="center nowrap">
                    <span class="badge badge-deleted">已删除</span>
                    <button
                      class="ghost"
                      type="button"
                      :disabled="restoringId === r.id"
                      @click="restoreRepo(r)"
                    >
                      {{ restoringId === r.id ? '恢复中…' : '恢复' }}
                    </button>
                  </td>
                </tr>
              </tbody>
            </table>
          </template>
          <p v-else class="muted">暂无已删除的仓库</p>
        </section>

        <!-- 安全活动 -->
        <section v-else-if="active === 'general'" class="card">
          <h3>通用</h3>
          <label class="field">
            语言 / Language
            <select v-model="selectedLocale" @change="changeLocale(selectedLocale)">
              <option v-for="o in localeOptions" :key="o.value" :value="o.value">{{ o.label }}</option>
            </select>
          </label>
          <p class="hint">切换后立即生效并保存。</p>
          <label class="field">
            通知频率
            <select v-model="notificationFrequency" :disabled="notifLoading">
              <option
                v-for="o in notificationFrequencyOptions"
                :key="o.value"
                :value="o.value"
              >
                {{ o.label }}
              </option>
            </select>
          </label>
          <p v-if="notifLoading" class="muted">加载中…</p>
          <p v-if="notifError" class="error">{{ notifError }}</p>
          <p v-if="notifOk" class="ok">{{ notifOk }}</p>
          <button class="primary" :disabled="notifSaving || notifLoading" @click="saveNotificationFrequency">
            {{ notifSaving ? '保存中…' : '保存通知频率' }}
          </button>
        </section>
        <section v-else class="card">
          <h3>我的活动</h3>
          <p v-if="logsError" class="error">{{ logsError }}</p>
          <div class="activity-toolbar">
            <label class="inline">
              结果
              <select v-model="logsResult" @change="applyResultFilter">
                <option v-for="o in activityResultOptions" :key="o.value" :value="o.value">
                  {{ o.label }}
                </option>
              </select>
            </label>
          </div>
          <p v-if="logsLoading" class="muted">加载中…</p>
          <template v-else-if="logs.length">
            <table class="key-table">
              <thead>
                <tr>
                  <th>时间</th>
                  <th>事件</th>
                  <th class="center">结果</th>
                  <th>目标</th>
                  <th>IP</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="l in logs" :key="l.id">
                  <td class="muted nowrap">{{ fmtTime(l.created_at) }}</td>
                  <td>{{ l.event }}</td>
                  <td class="center">
                    <span class="badge" :class="resultClass(l.result)">{{ l.result }}</span>
                  </td>
                  <td>
                    <span v-if="l.target_name || l.target_type">
                      {{ l.target_type ? l.target_type + ' · ' : '' }}{{ l.target_name || '-' }}
                    </span>
                    <span v-else class="muted">-</span>
                  </td>
                  <td class="muted nowrap"><code>{{ l.actor_ip || '-' }}</code></td>
                </tr>
              </tbody>
            </table>
          </template>
          <p v-else class="muted">暂无活动记录</p>
          <div class="pager">
            <button class="ghost" type="button" :disabled="logsPage === 0 || logsLoading" @click="activityPrev">
              上一页
            </button>
            <span class="muted">第 {{ logsPage + 1 }} 页</span>
            <button class="ghost" type="button" :disabled="!logsHasNext || logsLoading" @click="activityNext">
              下一页
            </button>
          </div>

          <hr class="divider" />

          <h3>登录设备</h3>
          <p v-if="sessionsError" class="error">{{ sessionsError }}</p>
          <p v-if="sessionsOk" class="ok">{{ sessionsOk }}</p>
          <p v-if="sessionsLoading" class="muted">加载中…</p>
          <template v-else-if="sessions.length">
            <table class="key-table">
              <thead>
                <tr>
                  <th>IP</th>
                  <th>User-Agent</th>
                  <th>创建时间</th>
                  <th>最后活跃</th>
                  <th>过期时间</th>
                  <th class="center">操作</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="s in sessions" :key="s.id">
                  <td class="nowrap"><code>{{ s.ip || '-' }}</code></td>
                  <td :title="s.user_agent || ''">{{ truncateUA(s.user_agent) }}</td>
                  <td class="muted nowrap">{{ fmtTime(s.created_at) }}</td>
                  <td class="muted nowrap">{{ fmtTime(s.last_seen_at) }}</td>
                  <td class="muted nowrap">{{ fmtTime(s.expires_at) }}</td>
                  <td class="center nowrap">
                    <span v-if="s.current" class="badge badge-success">当前设备</span>
                    <button
                      v-else
                      class="danger"
                      type="button"
                      :disabled="revoking === s.id"
                      @click="revokeSession(s)"
                    >
                      {{ revoking === s.id ? '注销中…' : '注销' }}
                    </button>
                  </td>
                </tr>
              </tbody>
            </table>
            <button
              class="danger"
              type="button"
              :disabled="revokingAll"
              @click="revokeOthers"
            >
              {{ revokingAll ? '注销中…' : '注销其他所有设备' }}
            </button>
          </template>
          <p v-else class="muted">暂无会话</p>
        </section>
      </main>
    </div>

    <div v-if="editOpen" class="modal-backdrop" @click.self="closeEdit">
      <div class="modal" role="dialog" aria-modal="true" aria-labelledby="key-edit-title">
        <h3 id="key-edit-title">编辑 SSH 密钥用途</h3>
        <p class="muted">{{ editTarget ? keyTitle(editTarget) : '' }}</p>
        <p v-if="editError" class="error">{{ editError }}</p>
        <label class="check">
          <input v-model="editForm.can_authenticate" type="checkbox" />
          允许用于认证（Git clone/push/pull）
        </label>
        <label class="check">
          <input v-model="editForm.can_sign" type="checkbox" />
          允许用于提交签名
        </label>
        <div class="modal-actions">
          <button class="ghost" type="button" :disabled="editBusy" @click="closeEdit">取消</button>
          <button class="primary" type="button" :disabled="editBusy" @click="saveEdit">
            {{ editBusy ? '保存中…' : '保存' }}
          </button>
        </div>
      </div>
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
.key-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.9rem;
}
.key-table th,
.key-table td {
  border-bottom: 1px solid var(--border);
  padding: 0.5rem 0.6rem;
  text-align: left;
  vertical-align: middle;
}
.key-table th {
  color: var(--muted);
  font-weight: 600;
  font-size: 0.82rem;
}
.key-table .center {
  text-align: center;
}
.nowrap {
  white-space: nowrap;
}
.badge {
  display: inline-block;
  min-width: 1.4rem;
  padding: 0.1rem 0.35rem;
  border-radius: 999px;
  font-size: 0.8rem;
  line-height: 1.4;
}
.badge-on {
  background: var(--success-soft);
  border: 1px solid var(--success);
}
.badge-off {
  background: transparent;
  border: 1px solid var(--border);
  opacity: 0.7;
}
label.check {
  flex-direction: row;
  align-items: center;
  gap: 0.5rem;
}
label.check input {
  width: auto;
  margin: 0;
}
.ghost {
  padding: 0.3rem 0.7rem;
  border-radius: 6px;
  border: 1px solid var(--border);
  background: transparent;
  color: var(--color-fg);
  font-size: 0.85rem;
  margin-right: 0.4rem;
}
.ghost:hover:not(:disabled) {
  background: var(--hover-bg);
}
.divider {
  border: none;
  border-top: 1px solid var(--border);
  margin: 0.75rem 0 0.25rem;
  width: 100%;
}
.modal-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.45);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1rem;
  z-index: 50;
}
.modal {
  background: var(--card);
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: 1.25rem;
  width: 100%;
  max-width: 420px;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}
.modal h3 {
  margin: 0;
}
.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
}
.modal-actions .ghost {
  margin-right: 0;
}
.activity-toolbar {
  display: flex;
  justify-content: flex-end;
}
label.inline {
  flex-direction: row;
  align-items: center;
  gap: 0.5rem;
}
.badge-success {
  background: var(--success-soft);
  border: 1px solid var(--success);
  color: #1a7f37;
}
.badge-failure {
  background: rgba(209, 36, 47, 0.12);
  border: 1px solid #d1242f;
  color: #d1242f;
}
.badge-denied {
  background: var(--accent-soft);
  border: 1px solid var(--accent);
  color: #7a4b00;
}
.badge-neutral {
  background: transparent;
  border: 1px solid var(--border);
  color: var(--muted);
}
.badge-deleted {
  background: rgba(209, 36, 47, 0.12);
  border: 1px solid rgba(209, 36, 47, 0.5);
  color: #d1242f;
  margin-right: 0.4rem;
}
.deleted-hint {
  color: var(--muted);
  font-size: 0.85rem;
}
.scope-group {
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
}
.scope-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 0.3rem;
  margin-top: 0.35rem;
}
.scope-chip {
  display: inline-block;
  margin: 0 0.25rem 0.25rem 0;
  padding: 0.1rem 0.5rem;
  border-radius: 999px;
  border: 1px solid var(--border);
  background: var(--hover-bg, transparent);
  font-size: 0.78rem;
  color: var(--muted);
}
.pager {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  justify-content: center;
}
@media (max-width: 768px) {
  .body {
    flex-direction: column;
  }
}
</style>
