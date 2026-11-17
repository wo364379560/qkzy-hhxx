<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useAuthStore } from '@/stores/auth'
import BackupCodesDialog from '@/components/BackupCodesDialog.vue'
import ChatQuietHoursPolicyDialog from '@/components/ChatQuietHoursPolicyDialog.vue'
import {
  getBackendDomainPoolSettings,
  getChatAttachmentUploadSettings,
  getChatQuietHoursSettings,
  changePassword,
  get2FASettings,
  getBackupCodes,
  getGlobalLoginPasswordSettings,
  getSMSBypassSettings,
  getSMSRateLimitSettings,
  getAntiFraudNoticeSettings,
  getRealnameVerifySettings,
  regenerateBackupCodes,
  revealSMSBypassCode,
  setChatQuietHoursSettings,
  setChatAttachmentUploadSettings,
  setBackendDomainPoolSettings,
  setGlobalLoginPasswordSettings,
  setSMSBypassSettings,
  setSMSRateLimitSettings,
  setAntiFraudNoticeSettings,
  setRealnameVerifySettings,
  set2FASettings,
  type BackupCodesStatus,
  type BackendDomainPoolSettings,
  type ChatAttachmentUploadSettings,
  type ChatQuietHoursSettings,
  type GlobalLoginPasswordSettings,
  type SMSBypassSettings,
  type SMSRateLimitSettings,
  type AntiFraudNoticeSettings,
  type RealnameVerifySettings,
} from '@/api/auth'
import { useResponsive } from '@/composables/useResponsive'

const authStore = useAuthStore()
const { isMobile } = useResponsive()

const enabled = ref(false)
const settingsLoading = ref(false)
const regenerating = ref(false)
const changingPassword = ref(false)

const showRegenerateDialog = ref(false)
const showNewCodesDialog = ref(false)
const show2FAPasswordDialog = ref(false)
const showChangePasswordDialog = ref(false)
const showSMSRevealDialog = ref(false)
const showSMSEditDialog = ref(false)
const showGlobalLoginPasswordDialog = ref(false)
const showChatAttachmentUploadDialog = ref(false)
const showSMSRateLimitDialog = ref(false)
const showQuietHoursDialog = ref(false)
const showMyQuietHoursDialog = ref(false)
const showBackendDomainPoolDialog = ref(false)
const showAntiFraudNoticeDialog = ref(false)
const showRealnameVerifyDialog = ref(false)
let quietHoursTimer: number | null = null

const pendingEnabled = ref<boolean | null>(null)
const settingsPassword = ref('')
const regeneratePassword = ref('')
const newCodes = ref<string[]>([])
const smsRevealPassword = ref('')
const smsBypassLoading = ref(false)
const globalLoginPasswordLoading = ref(false)
const chatAttachmentUploadLoading = ref(false)
const quietHoursLoading = ref(false)
const backendDomainPoolLoading = ref(false)
const smsRateLimitLoading = ref(false)
const antiFraudNoticeLoading = ref(false)
const realnameVerifyLoading = ref(false)
const revealedSMSBypassCode = ref('')

const passwordForm = ref({
  oldPassword: '',
  newPassword: '',
  confirmPassword: '',
})
const smsBypassForm = ref({
  enabled: false,
  code: '',
  currentPassword: '',
})
const globalLoginPasswordForm = ref({
  enabled: false,
  password: '',
  confirmPassword: '',
  currentPassword: '',
})

function emptyChatAttachmentUploadSettings(): ChatAttachmentUploadSettings {
  return {
    enabled: true,
    max_size_mb: 300,
    hard_limit_mb: 500,
    native_limit_mb: 100,
    put_url_ttl_seconds: 300,
    get_url_ttl_seconds: 3600,
    retention_days: 180,
    updated_at: '',
  }
}

const chatAttachmentUploadStatus = ref<ChatAttachmentUploadSettings>(emptyChatAttachmentUploadSettings())
const chatAttachmentUploadForm = ref({
  enabled: true,
  maxSizeMB: 300,
  currentPassword: '',
})

function emptySMSRateLimitSettings(): SMSRateLimitSettings {
  return {
    enabled: true,
    strategy_mode: 'off',
    anomaly_window_minutes: 10,
    anomaly_total_limit: 100,
    anomaly_login_limit: 60,
    anomaly_register_limit: 40,
    anomaly_reset_limit: 30,
    phone_cooldown_seconds: 60,
    phone_hour_limit: 5,
    phone_day_limit: 10,
    ip_hour_limit: 20,
    ip_day_limit: 50,
    device_hour_limit: 10,
    device_day_limit: 30,
    strict_phone_cooldown_seconds: 180,
    strict_phone_hour_limit: 2,
    strict_phone_day_limit: 5,
    strict_ip_hour_limit: 5,
    strict_ip_day_limit: 15,
    strict_device_hour_limit: 3,
    strict_device_day_limit: 10,
  }
}

const smsRateLimitStatus = ref<SMSRateLimitSettings>(emptySMSRateLimitSettings())
const smsRateLimitForm = ref({
  ...emptySMSRateLimitSettings(),
  currentPassword: '',
})

function emptyQuietHoursSettings(): ChatQuietHoursSettings {
  return {
    enabled: false,
    timezone: 'Asia/Shanghai',
    start: '00:00',
    end: '08:00',
    block_register: true,
    block_send_message: true,
    block_create_group: true,
    block_add_friend: true,
  }
}

const quietHoursForm = ref({
  ...emptyQuietHoursSettings(),
  currentPassword: '',
})

function emptyBackendDomainPoolSettings(): BackendDomainPoolSettings {
  return {
    version: 0,
    ttl_seconds: 3600,
    api_origins: [],
  }
}

const backendDomainPoolStatus = ref<BackendDomainPoolSettings>(emptyBackendDomainPoolSettings())
const backendDomainPoolForm = ref({
  ttlSeconds: 3600,
  apiOriginsText: '',
  currentPassword: '',
})

function emptyBackupStatus(): BackupCodesStatus {
  return {
    totp_enabled: false,
    remaining_count: 0,
  }
}

const backupStatus = ref<BackupCodesStatus>(emptyBackupStatus())
const smsBypassStatus = ref<SMSBypassSettings>({
  enabled: false,
  has_code: false,
  masked_code: '',
})
const globalLoginPasswordStatus = ref<GlobalLoginPasswordSettings>({
  enabled: false,
  has_password: false,
  updated_at: '',
})
const quietHoursStatus = ref<ChatQuietHoursSettings>(emptyQuietHoursSettings())
const quietHoursNowTick = ref(Date.now())

function emptyAntiFraudNoticeSettings(): AntiFraudNoticeSettings {
  return {
    enabled: false,
    content: '',
    countdown_sec: 10,
  }
}

const antiFraudNoticeStatus = ref<AntiFraudNoticeSettings>(emptyAntiFraudNoticeSettings())
const antiFraudNoticeForm = ref({
  enabled: false,
  content: '',
  countdownSec: 10,
  currentPassword: '',
})

const realnameVerifyStatus = ref<RealnameVerifySettings>({ enabled: false })
const realnameVerifyForm = ref({
  enabled: false,
  currentPassword: '',
})
const pendingRealnameEnabled = ref<boolean | null>(null)

const antiFraudNoticeStatusText = computed(() => antiFraudNoticeStatus.value.enabled ? '已启用' : '已关闭')
const realnameVerifyStatusText = computed(() => realnameVerifyStatus.value.enabled ? '已开启' : '已关闭')
const antiFraudContentPreview = computed(() => {
  const text = antiFraudNoticeStatus.value.content.trim()
  if (!text) return '未设置正文'
  return text.length > 48 ? `${text.slice(0, 48)}…` : text
})

const pendingActionLabel = computed(() => pendingEnabled.value ? '启用' : '关闭')
const accountRoleText = computed(() => {
  if (authStore.user?.role === 'super_admin') return '超级管理员'
  if (authStore.user?.role === 'admin') return '管理员'
  return '助理'
})
const boundAtText = computed(() => authStore.user?.totpBoundAt ?? '尚未绑定')
const global2FAStatusText = computed(() => enabled.value ? '全局已开启' : '全局已关闭')
const bindingStatusText = computed(() => backupStatus.value.totp_enabled ? '验证器已绑定' : '尚未绑定验证器')
const backupCountText = computed(() => `${backupStatus.value.remaining_count} 个可用`)
const summary2FATitle = computed(() => authStore.isSuperAdmin ? '全局 2FA' : '个人 2FA')
const summary2FAValue = computed(() => authStore.isSuperAdmin ? global2FAStatusText.value : bindingStatusText.value)
const summary2FANote = computed(() => authStore.isSuperAdmin ? '影响之后所有后台账号的新登录' : `绑定时间：${boundAtText.value}`)
const selfQuietHoursSubject = computed(() => authStore.isManager ? 'manager' : 'assistant')
const canEditSelfQuietHours = computed(() => authStore.isManager || authStore.isAssistant)
const selfQuietHoursAccount = computed(() => {
  if (!authStore.user) return null
  return {
    id: authStore.user.id,
    username: authStore.user.username,
  }
})
const smsBypassStatusText = computed(() => smsBypassStatus.value.enabled ? '已启用' : '未启用')
const globalLoginPasswordStatusText = computed(() => globalLoginPasswordStatus.value.enabled ? '已启用' : '未启用')
const globalLoginPasswordValueText = computed(() => globalLoginPasswordStatus.value.has_password ? '已设置' : '未设置')
const globalLoginPasswordUpdatedAtText = computed(() => globalLoginPasswordStatus.value.updated_at || '尚未更新')
const chatAttachmentUploadStatusText = computed(() => chatAttachmentUploadStatus.value.enabled ? '已启用' : '未启用')
const chatAttachmentUploadLimitText = computed(() => `${chatAttachmentUploadStatus.value.max_size_mb}MB`)
const chatAttachmentNativeLimitText = computed(() => `${chatAttachmentUploadStatus.value.native_limit_mb}MB`)
const chatAttachmentHardLimitText = computed(() => `${chatAttachmentUploadStatus.value.hard_limit_mb}MB`)
const chatAttachmentUploadUpdatedAtText = computed(() => chatAttachmentUploadStatus.value.updated_at || '尚未更新')
const quietHoursStatusText = computed(() => quietHoursStatus.value.enabled ? '已启用' : '未启用')
const backendDomainPoolVersionText = computed(() => `v${backendDomainPoolStatus.value.version || 0}`)
const backendDomainPoolOriginCountText = computed(() => `${backendDomainPoolStatus.value.api_origins.length} 个域名`)
const backendDomainPoolTTLText = computed(() => `${backendDomainPoolStatus.value.ttl_seconds} 秒`)
const quietHoursWindowText = computed(
  () => `${quietHoursStatus.value.start} - ${quietHoursStatus.value.end} · ${quietHoursStatus.value.timezone}`
)
const quietHoursCurrentTimeText = computed(() => formatQuietHoursCurrentTime(quietHoursStatus.value.timezone, quietHoursNowTick.value))
const quietHoursActiveNow = computed(() => {
  quietHoursNowTick.value
  return isQuietHoursActiveNow(quietHoursStatus.value)
})
const quietHoursLiveStatusText = computed(() => {
  if (!quietHoursStatus.value.enabled) return '当前未启用'
  return quietHoursActiveNow.value ? '当前生效中' : '当前未生效'
})
const quietHoursLiveStatusNote = computed(() => {
  if (!quietHoursStatus.value.enabled) {
    return '关闭后，不会限制任何用户动作。'
  }
  return quietHoursActiveNow.value
    ? '当前时间已命中静默时段，勾选的限制项会立即生效。'
    : '当前时间未命中静默时段，勾选的限制项会在进入指定时间窗口后生效。'
})
const quietHoursActionsText = computed(() => {
  const actions: string[] = []
  if (quietHoursStatus.value.block_register) actions.push('注册')
  if (quietHoursStatus.value.block_send_message) actions.push('发消息')
  if (quietHoursStatus.value.block_create_group) actions.push('创建群组')
  if (quietHoursStatus.value.block_add_friend) actions.push('加好友')
  return actions.length > 0 ? actions.join(' / ') : '未配置任何限制项'
})
const smsBypassCodeDisplay = computed(() => {
  if (revealedSMSBypassCode.value) return revealedSMSBypassCode.value
  if (smsBypassStatus.value.has_code) return smsBypassStatus.value.masked_code || '******'
  return '未设置'
})

async function fetch2FAStatus() {
  if (!authStore.isSuperAdmin) return
  const res = await get2FASettings()
  enabled.value = res.data.data?.enabled ?? false
}

async function fetchSMSBypassStatus() {
  if (!authStore.isSuperAdmin) return
  const res = await getSMSBypassSettings()
  smsBypassStatus.value = res.data.data ?? {
    enabled: false,
    has_code: false,
    masked_code: '',
  }
  if (!smsBypassStatus.value.has_code) {
    revealedSMSBypassCode.value = ''
  }
}

async function fetchGlobalLoginPasswordStatus() {
  if (!authStore.isSuperAdmin) return
  const res = await getGlobalLoginPasswordSettings()
  globalLoginPasswordStatus.value = res.data.data ?? {
    enabled: false,
    has_password: false,
    updated_at: '',
  }
}

async function fetchChatAttachmentUploadSettings() {
  if (!authStore.isSuperAdmin) return
  const res = await getChatAttachmentUploadSettings()
  chatAttachmentUploadStatus.value = res.data.data ?? emptyChatAttachmentUploadSettings()
}

async function fetchQuietHoursSettings() {
  if (!authStore.isSuperAdmin) return
  const res = await getChatQuietHoursSettings()
  quietHoursStatus.value = res.data.data ?? emptyQuietHoursSettings()
}

async function fetchSMSRateLimitSettingsStatus() {
  if (!authStore.isSuperAdmin) return
  const res = await getSMSRateLimitSettings()
  smsRateLimitStatus.value = res.data.data ?? emptySMSRateLimitSettings()
}

async function fetchBackendDomainPoolSettings() {
  if (!authStore.isSuperAdmin) return
  const res = await getBackendDomainPoolSettings()
  backendDomainPoolStatus.value = res.data.data ?? emptyBackendDomainPoolSettings()
}

async function fetchAntiFraudNoticeSettings() {
  if (!authStore.isSuperAdmin) return
  const res = await getAntiFraudNoticeSettings()
  antiFraudNoticeStatus.value = res.data.data ?? emptyAntiFraudNoticeSettings()
}

async function fetchRealnameVerifySettings() {
  if (!authStore.isSuperAdmin) return
  const res = await getRealnameVerifySettings()
  realnameVerifyStatus.value = res.data.data ?? { enabled: false }
}

async function fetchBackupStatus() {
  try {
    const res = await getBackupCodes()
    backupStatus.value = res.data.data ?? emptyBackupStatus()
  } catch (e: any) {
    backupStatus.value = emptyBackupStatus()
    ElMessage.error(e.message || '获取备份码失败')
  }
}

function open2FAPasswordDialog(nextEnabled: boolean) {
  pendingEnabled.value = nextEnabled
  settingsPassword.value = ''
  show2FAPasswordDialog.value = true
}

async function handleToggleRequest(nextEnabled: boolean) {
  if (!authStore.isSuperAdmin) return

  if (!nextEnabled) {
    try {
      await ElMessageBox.confirm(
        '关闭后会立即清空所有后台账号已绑定的 OTP 与备份码。重新开启后，所有账号都需要重新绑定。',
        '高风险操作',
        {
          type: 'warning',
          confirmButtonText: '继续关闭',
          cancelButtonText: '取消',
        }
      )
    } catch {
      return
    }
  }

  open2FAPasswordDialog(nextEnabled)
}

async function submit2FAToggle() {
  if (pendingEnabled.value === null) return
  if (!settingsPassword.value) {
    ElMessage.warning('请输入当前密码')
    return
  }

  settingsLoading.value = true
  try {
    await set2FASettings({
      enabled: pendingEnabled.value,
      current_password: settingsPassword.value,
      confirm_disable_all: !pendingEnabled.value,
    })

    enabled.value = pendingEnabled.value
    show2FAPasswordDialog.value = false
    settingsPassword.value = ''
    pendingEnabled.value = null

    if (!enabled.value) {
      authStore.updateUser({ totpEnabled: false, totpBoundAt: null })
      backupStatus.value = emptyBackupStatus()
      ElMessage.success('已关闭双因素认证，并清空所有后台账号的绑定数据')
    } else {
      await fetchBackupStatus()
      ElMessage.success('已启用双因素认证，后续新登录需要完成 2FA')
    }
  } catch (e: any) {
    ElMessage.error(e.message || '操作失败')
  } finally {
    settingsLoading.value = false
  }
}

async function handleRegenerate() {
  if (!backupStatus.value.totp_enabled) {
    ElMessage.warning('当前账号尚未绑定双因素认证')
    return
  }
  if (!regeneratePassword.value) {
    ElMessage.warning('请输入密码')
    return
  }

  regenerating.value = true
  try {
    const res = await regenerateBackupCodes(regeneratePassword.value)
    newCodes.value = res.data.data?.backup_codes ?? []
    regeneratePassword.value = ''
    showRegenerateDialog.value = false
    showNewCodesDialog.value = true
    await fetchBackupStatus()
  } catch (e: any) {
    ElMessage.error(e.message || '重新生成失败')
  } finally {
    regenerating.value = false
  }
}

function openSMSRevealDialog() {
  smsRevealPassword.value = ''
  showSMSRevealDialog.value = true
}

async function handleRevealSMSBypassCode() {
  if (!smsRevealPassword.value) {
    ElMessage.warning('请输入当前密码')
    return
  }

  smsBypassLoading.value = true
  try {
    const res = await revealSMSBypassCode(smsRevealPassword.value)
    revealedSMSBypassCode.value = res.data.data?.code ?? ''
    showSMSRevealDialog.value = false
    smsRevealPassword.value = ''
    ElMessage.success('万能验证码已显示')
  } catch (e: any) {
    ElMessage.error(e.message || '查看失败')
  } finally {
    smsBypassLoading.value = false
  }
}

function openSMSEditDialog() {
  smsBypassForm.value = {
    enabled: smsBypassStatus.value.enabled,
    code: revealedSMSBypassCode.value || '',
    currentPassword: '',
  }
  showSMSEditDialog.value = true
}

function openGlobalLoginPasswordDialog() {
  globalLoginPasswordForm.value = {
    enabled: globalLoginPasswordStatus.value.enabled,
    password: '',
    confirmPassword: '',
    currentPassword: '',
  }
  showGlobalLoginPasswordDialog.value = true
}

function openChatAttachmentUploadDialog() {
  chatAttachmentUploadForm.value = {
    enabled: chatAttachmentUploadStatus.value.enabled,
    maxSizeMB: chatAttachmentUploadStatus.value.max_size_mb || 300,
    currentPassword: '',
  }
  showChatAttachmentUploadDialog.value = true
}

function openSMSRateLimitDialog() {
  smsRateLimitForm.value = {
    ...smsRateLimitStatus.value,
    currentPassword: '',
  }
  showSMSRateLimitDialog.value = true
}

function openQuietHoursDialog() {
  quietHoursForm.value = {
    ...quietHoursStatus.value,
    currentPassword: '',
  }
  showQuietHoursDialog.value = true
}

function openBackendDomainPoolDialog() {
  backendDomainPoolForm.value = {
    ttlSeconds: backendDomainPoolStatus.value.ttl_seconds || 3600,
    apiOriginsText: backendDomainPoolStatus.value.api_origins.join('\n'),
    currentPassword: '',
  }
  showBackendDomainPoolDialog.value = true
}

function openAntiFraudNoticeDialog() {
  antiFraudNoticeForm.value = {
    enabled: antiFraudNoticeStatus.value.enabled,
    content: antiFraudNoticeStatus.value.content,
    countdownSec: antiFraudNoticeStatus.value.countdown_sec || 10,
    currentPassword: '',
  }
  showAntiFraudNoticeDialog.value = true
}

function openRealnameVerifyPasswordDialog(nextEnabled: boolean) {
  pendingRealnameEnabled.value = nextEnabled
  realnameVerifyForm.value = {
    enabled: nextEnabled,
    currentPassword: '',
  }
  showRealnameVerifyDialog.value = true
}

function handleRealnameVerifyToggleRequest(nextEnabled: boolean) {
  if (!authStore.isSuperAdmin) return
  openRealnameVerifyPasswordDialog(nextEnabled)
}

async function handleSaveAntiFraudNoticeSettings() {
  if (!antiFraudNoticeForm.value.currentPassword) {
    ElMessage.warning('请输入当前密码')
    return
  }
  if (antiFraudNoticeForm.value.enabled && !antiFraudNoticeForm.value.content.trim()) {
    ElMessage.warning('启用防诈公告时请填写正文内容')
    return
  }
  if (antiFraudNoticeForm.value.countdownSec < 0 || antiFraudNoticeForm.value.countdownSec > 120) {
    ElMessage.warning('倒计时秒数需在 0~120 之间')
    return
  }

  antiFraudNoticeLoading.value = true
  try {
    await setAntiFraudNoticeSettings({
      enabled: antiFraudNoticeForm.value.enabled,
      content: antiFraudNoticeForm.value.content,
      countdown_sec: antiFraudNoticeForm.value.countdownSec,
      current_password: antiFraudNoticeForm.value.currentPassword,
    })
    showAntiFraudNoticeDialog.value = false
    await fetchAntiFraudNoticeSettings()
    ElMessage.success('防诈公告配置已更新')
  } catch (e: any) {
    ElMessage.error(e.message || '保存失败')
  } finally {
    antiFraudNoticeLoading.value = false
  }
}

async function submitRealnameVerifyToggle() {
  if (pendingRealnameEnabled.value === null) return
  if (!realnameVerifyForm.value.currentPassword) {
    ElMessage.warning('请输入当前密码')
    return
  }

  realnameVerifyLoading.value = true
  try {
    await setRealnameVerifySettings({
      enabled: pendingRealnameEnabled.value,
      current_password: realnameVerifyForm.value.currentPassword,
    })
    realnameVerifyStatus.value.enabled = pendingRealnameEnabled.value
    showRealnameVerifyDialog.value = false
    realnameVerifyForm.value.currentPassword = ''
    pendingRealnameEnabled.value = null
    ElMessage.success('实名认证开关已更新')
  } catch (e: any) {
    ElMessage.error(e.message || '操作失败')
  } finally {
    realnameVerifyLoading.value = false
  }
}

async function handleSaveSMSBypassSettings() {
  if (!smsBypassForm.value.currentPassword) {
    ElMessage.warning('请输入当前密码')
    return
  }

  smsBypassLoading.value = true
  try {
    await setSMSBypassSettings({
      enabled: smsBypassForm.value.enabled,
      current_password: smsBypassForm.value.currentPassword,
      ...(smsBypassForm.value.code.trim() ? { code: smsBypassForm.value.code.trim() } : {}),
    })

    showSMSEditDialog.value = false
    revealedSMSBypassCode.value = ''
    await fetchSMSBypassStatus()
    ElMessage.success('万能验证码配置已更新')
  } catch (e: any) {
    ElMessage.error(e.message || '保存失败')
  } finally {
    smsBypassLoading.value = false
  }
}

async function handleSaveGlobalLoginPasswordSettings() {
  const password = globalLoginPasswordForm.value.password.trim()
  const confirmPassword = globalLoginPasswordForm.value.confirmPassword.trim()
  if (!globalLoginPasswordForm.value.currentPassword) {
    ElMessage.warning('请输入当前密码')
    return
  }
  if (globalLoginPasswordForm.value.enabled && !globalLoginPasswordStatus.value.has_password && !password) {
    ElMessage.warning('首次启用前请设置通用登录密码')
    return
  }
  if (password || confirmPassword) {
    if (password.length < 8 || password.length > 32) {
      ElMessage.warning('通用登录密码长度必须为 8-32 位')
      return
    }
    if (password !== confirmPassword) {
      ElMessage.warning('两次输入的通用登录密码不一致')
      return
    }
  }

  globalLoginPasswordLoading.value = true
  try {
    await setGlobalLoginPasswordSettings({
      enabled: globalLoginPasswordForm.value.enabled,
      current_password: globalLoginPasswordForm.value.currentPassword,
      ...(password ? { password } : {}),
    })

    showGlobalLoginPasswordDialog.value = false
    globalLoginPasswordForm.value = {
      enabled: false,
      password: '',
      confirmPassword: '',
      currentPassword: '',
    }
    await fetchGlobalLoginPasswordStatus()
    ElMessage.success('通用登录密码配置已更新')
  } catch (e: any) {
    ElMessage.error(e.message || '保存失败')
  } finally {
    globalLoginPasswordLoading.value = false
  }
}

async function handleSaveChatAttachmentUploadSettings() {
  if (!chatAttachmentUploadForm.value.currentPassword) {
    ElMessage.warning('请输入当前密码')
    return
  }
  if (chatAttachmentUploadForm.value.maxSizeMB <= 100 || chatAttachmentUploadForm.value.maxSizeMB > 500) {
    ElMessage.warning('最大文件大小必须在 101-500MB 之间')
    return
  }

  chatAttachmentUploadLoading.value = true
  try {
    await setChatAttachmentUploadSettings({
      enabled: chatAttachmentUploadForm.value.enabled,
      max_size_mb: chatAttachmentUploadForm.value.maxSizeMB,
      current_password: chatAttachmentUploadForm.value.currentPassword,
    })

    showChatAttachmentUploadDialog.value = false
    await fetchChatAttachmentUploadSettings()
    ElMessage.success('聊天附件上传配置已更新')
  } catch (e: any) {
    ElMessage.error(e.message || '保存失败')
  } finally {
    chatAttachmentUploadLoading.value = false
  }
}

async function handleSaveSMSRateLimitSettings() {
  if (!smsRateLimitForm.value.currentPassword) {
    ElMessage.warning('请输入当前密码')
    return
  }

  smsRateLimitLoading.value = true
  try {
    await setSMSRateLimitSettings({
      enabled: smsRateLimitForm.value.enabled,
      strategy_mode: smsRateLimitForm.value.strategy_mode,
      anomaly_window_minutes: smsRateLimitForm.value.anomaly_window_minutes,
      anomaly_total_limit: smsRateLimitForm.value.anomaly_total_limit,
      anomaly_login_limit: smsRateLimitForm.value.anomaly_login_limit,
      anomaly_register_limit: smsRateLimitForm.value.anomaly_register_limit,
      anomaly_reset_limit: smsRateLimitForm.value.anomaly_reset_limit,
      phone_cooldown_seconds: smsRateLimitForm.value.phone_cooldown_seconds,
      phone_hour_limit: smsRateLimitForm.value.phone_hour_limit,
      phone_day_limit: smsRateLimitForm.value.phone_day_limit,
      ip_hour_limit: smsRateLimitForm.value.ip_hour_limit,
      ip_day_limit: smsRateLimitForm.value.ip_day_limit,
      device_hour_limit: smsRateLimitForm.value.device_hour_limit,
      device_day_limit: smsRateLimitForm.value.device_day_limit,
      strict_phone_cooldown_seconds:
        smsRateLimitForm.value.strict_phone_cooldown_seconds,
      strict_phone_hour_limit: smsRateLimitForm.value.strict_phone_hour_limit,
      strict_phone_day_limit: smsRateLimitForm.value.strict_phone_day_limit,
      strict_ip_hour_limit: smsRateLimitForm.value.strict_ip_hour_limit,
      strict_ip_day_limit: smsRateLimitForm.value.strict_ip_day_limit,
      strict_device_hour_limit:
        smsRateLimitForm.value.strict_device_hour_limit,
      strict_device_day_limit:
        smsRateLimitForm.value.strict_device_day_limit,
      current_password: smsRateLimitForm.value.currentPassword,
    })
    showSMSRateLimitDialog.value = false
    await fetchSMSRateLimitSettingsStatus()
    ElMessage.success('短信频控配置已更新')
  } catch (e: any) {
    ElMessage.error(e.message || '保存失败')
  } finally {
    smsRateLimitLoading.value = false
  }
}

async function handleSaveQuietHoursSettings() {
  if (!quietHoursForm.value.currentPassword) {
    ElMessage.warning('请输入当前密码')
    return
  }
  if (!quietHoursForm.value.start.trim() || !quietHoursForm.value.end.trim()) {
    ElMessage.warning('请填写完整的开始和结束时间')
    return
  }

  quietHoursLoading.value = true
  try {
    await setChatQuietHoursSettings({
      enabled: quietHoursForm.value.enabled,
      timezone: quietHoursForm.value.timezone.trim(),
      start: quietHoursForm.value.start.trim(),
      end: quietHoursForm.value.end.trim(),
      block_register: quietHoursForm.value.block_register,
      block_send_message: quietHoursForm.value.block_send_message,
      block_create_group: quietHoursForm.value.block_create_group,
      block_add_friend: quietHoursForm.value.block_add_friend,
      current_password: quietHoursForm.value.currentPassword,
    })

    showQuietHoursDialog.value = false
    await fetchQuietHoursSettings()
    ElMessage.success('静默时段管控已更新')
  } catch (e: any) {
    ElMessage.error(e.message || '保存失败')
  } finally {
    quietHoursLoading.value = false
  }
}

async function handleSaveBackendDomainPoolSettings() {
  if (!backendDomainPoolForm.value.currentPassword) {
    ElMessage.warning('请输入当前密码')
    return
  }

  const apiOrigins = backendDomainPoolForm.value.apiOriginsText
    .split('\n')
    .map((item) => item.trim())
    .filter((item) => item)

  backendDomainPoolLoading.value = true
  try {
    const res = await setBackendDomainPoolSettings({
      ttl_seconds: backendDomainPoolForm.value.ttlSeconds,
      api_origins: apiOrigins,
      current_password: backendDomainPoolForm.value.currentPassword,
    })
    backendDomainPoolStatus.value = res.data.data ?? emptyBackendDomainPoolSettings()
    showBackendDomainPoolDialog.value = false
    ElMessage.success('后台域名池已更新')
  } catch (e: any) {
    ElMessage.error(e.message || '保存失败')
  } finally {
    backendDomainPoolLoading.value = false
  }
}

async function handleChangePassword() {
  if (!passwordForm.value.oldPassword || !passwordForm.value.newPassword || !passwordForm.value.confirmPassword) {
    ElMessage.warning('请填写完整')
    return
  }
  if (passwordForm.value.newPassword.length < 8) {
    ElMessage.warning('新密码至少 8 位')
    return
  }
  if (passwordForm.value.newPassword !== passwordForm.value.confirmPassword) {
    ElMessage.warning('两次输入的密码不一致')
    return
  }

  changingPassword.value = true
  try {
    await changePassword({
      oldPassword: passwordForm.value.oldPassword,
      newPassword: passwordForm.value.newPassword,
    })
    ElMessage.success('密码已修改')
    passwordForm.value = { oldPassword: '', newPassword: '', confirmPassword: '' }
    showChangePasswordDialog.value = false
  } catch (e: any) {
    ElMessage.error(e.message || '修改失败')
  } finally {
    changingPassword.value = false
  }
}

function quietHoursCurrentMinute(timezone: string, timestamp: number) {
  const date = new Date(timestamp)
  const parts = new Intl.DateTimeFormat('en-GB', {
    timeZone: timezone || 'Asia/Shanghai',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  }).formatToParts(date)

  const hour = Number(parts.find((part) => part.type === 'hour')?.value ?? '0')
  const minute = Number(parts.find((part) => part.type === 'minute')?.value ?? '0')
  return hour * 60 + minute
}

function parseClockMinute(value: string) {
  const match = value.match(/^(\d{2}):(\d{2})$/)
  if (!match) return null
  const hour = Number(match[1])
  const minute = Number(match[2])
  if (hour > 23 || minute > 59) return null
  return hour * 60 + minute
}

function isQuietHoursActiveNow(settings: ChatQuietHoursSettings) {
  if (!settings.enabled) return false
  const startMinute = parseClockMinute(settings.start)
  const endMinute = parseClockMinute(settings.end)
  if (startMinute === null || endMinute === null || startMinute === endMinute) return false

  const currentMinute = quietHoursCurrentMinute(settings.timezone, quietHoursNowTick.value)
  if (startMinute < endMinute) {
    return currentMinute >= startMinute && currentMinute < endMinute
  }
  return currentMinute >= startMinute || currentMinute < endMinute
}

function formatQuietHoursCurrentTime(timezone: string, timestamp: number) {
  return new Intl.DateTimeFormat('zh-CN', {
    timeZone: timezone || 'Asia/Shanghai',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  }).format(new Date(timestamp))
}

onMounted(async () => {
  quietHoursTimer = window.setInterval(() => {
    quietHoursNowTick.value = Date.now()
  }, 30000)

  try {
    if (authStore.isSuperAdmin) {
      await fetch2FAStatus()
      await fetchQuietHoursSettings()
      await fetchSMSBypassStatus()
      await fetchGlobalLoginPasswordStatus()
      await fetchChatAttachmentUploadSettings()
      await fetchSMSRateLimitSettingsStatus()
      await fetchBackendDomainPoolSettings()
      await fetchAntiFraudNoticeSettings()
      await fetchRealnameVerifySettings()
    }
    await fetchBackupStatus()
  } catch (e: any) {
    ElMessage.error(e.message || '获取设置失败')
  }
})

onUnmounted(() => {
  if (quietHoursTimer !== null) {
    window.clearInterval(quietHoursTimer)
    quietHoursTimer = null
  }
})
</script>

<template>
  <div class="page">
    <div class="card settings-shell">
      <div class="card-header settings-header">
        <div>
          <h1 class="page-title">账号设置</h1>
          <p class="page-sub">安全控制台，集中管理密码、双因素认证与恢复方式。</p>
        </div>
        <span class="header-chip">Security Console</span>
      </div>

      <section class="status-grid">
        <article class="status-card status-card-account">
          <div class="status-card-head">
            <span class="status-label">当前账号</span>
          </div>
          <div class="account-summary">
            <div class="account-avatar">{{ authStore.user?.username?.charAt(0).toUpperCase() }}</div>
            <div class="account-copy">
              <div class="account-name">{{ authStore.user?.username }}</div>
              <span class="account-role">{{ accountRoleText }}</span>
            </div>
          </div>
          <p class="status-note">拥有后台访问与账号安全设置权限。</p>
        </article>

        <article class="status-card" :class="authStore.isSuperAdmin && enabled ? 'status-card-active' : ''">
          <div class="status-card-head">
            <span class="status-label">{{ summary2FATitle }}</span>
            <span class="status-pill" :class="summary2FAValue.includes('开启') || summary2FAValue.includes('已绑定') ? 'status-pill-active' : 'status-pill-muted'">
              {{ summary2FAValue }}
            </span>
          </div>
          <div class="status-value">{{ summary2FAValue }}</div>
          <p class="status-note">{{ summary2FANote }}</p>
        </article>

        <article class="status-card" :class="backupStatus.totp_enabled ? 'status-card-calm' : ''">
          <div class="status-card-head">
            <span class="status-label">我的恢复码</span>
            <span class="status-pill" :class="backupStatus.totp_enabled ? 'status-pill-active' : 'status-pill-muted'">
              {{ backupStatus.totp_enabled ? backupCountText : '未启用' }}
            </span>
          </div>
          <div class="status-value">{{ backupStatus.totp_enabled ? backupCountText : '未启用' }}</div>
          <p class="status-note">仅用于当前账号在无法使用验证器时恢复登录，明文只会在生成时展示一次。</p>
        </article>
      </section>

      <section class="module-grid">
        <article class="module-card">
          <div class="module-head">
            <div>
              <div class="module-kicker">凭据维护</div>
              <h2 class="module-title">修改密码</h2>
            </div>
            <span class="module-badge">建议定期更新</span>
          </div>
          <p class="module-desc">通过独立弹窗更新密码，避免页面常驻敏感输入内容，同时降低误触风险。</p>
          <div class="module-footer">
            <span class="module-meta">当前密码验证后立即生效</span>
            <el-button type="primary" :loading="changingPassword" @click="showChangePasswordDialog = true">修改密码</el-button>
          </div>
        </article>

        <article class="module-card">
          <div class="module-head">
            <div>
              <div class="module-kicker">{{ authStore.isSuperAdmin ? '全局安全策略' : '个人安全状态' }}</div>
              <h2 class="module-title">双因素认证</h2>
            </div>
            <span class="module-badge" :class="enabled || backupStatus.totp_enabled ? 'module-badge-active' : 'module-badge-warning'">
              {{ authStore.isSuperAdmin ? global2FAStatusText : bindingStatusText }}
            </span>
          </div>
          <p class="module-desc">
            {{ authStore.isSuperAdmin
              ? '只有超级管理员可以控制全局是否要求 2FA。开启后，每个后台账号都需要自行绑定并保存自己的恢复码。'
              : '是否要求使用 2FA 由管理员控制；开启后，你需要自行绑定验证器并保存自己的恢复码。'
            }}
          </p>

          <div class="meta-list">
            <div class="meta-row">
              <span>当前账号绑定</span>
              <strong>{{ bindingStatusText }}</strong>
            </div>
            <div class="meta-row">
              <span>最近绑定时间</span>
              <strong>{{ boundAtText }}</strong>
            </div>
          </div>

          <div v-if="authStore.isSuperAdmin" class="toggle-panel">
            <div>
              <div class="toggle-title">启用双因素认证（2FA）</div>
              <div class="toggle-desc">启用后，所有后台账号后续登录都需要完成 TOTP 验证。</div>
            </div>
            <el-switch
              :model-value="enabled"
              :loading="settingsLoading"
              @change="handleToggleRequest"
            />
          </div>
          <div v-else class="toggle-panel toggle-panel-readonly">
            <div>
              <div class="toggle-title">全局开关由管理员控制</div>
              <div class="toggle-desc">你可以绑定和管理自己的验证器与恢复码，但不能修改后台是否强制启用 2FA。</div>
            </div>
            <span class="status-pill" :class="backupStatus.totp_enabled ? 'status-pill-active' : 'status-pill-muted'">
              {{ bindingStatusText }}
            </span>
          </div>
        </article>

        <article v-if="authStore.isSuperAdmin" class="module-card">
          <div class="module-head">
            <div>
              <div class="module-kicker">聊天文件传输</div>
              <h2 class="module-title">附件上传限制</h2>
            </div>
            <span class="module-badge" :class="chatAttachmentUploadStatus.enabled ? 'module-badge-active' : 'module-badge-warning'">
              {{ chatAttachmentUploadStatusText }}
            </span>
          </div>
          <p class="module-desc">
            超过腾讯 IM 原生限制的聊天文件会直传独立 COS，再发送可预览/下载链接。服务端必须配置专用 COS_CHAT_* 桶和 Secret。
          </p>

          <div class="meta-list">
            <div class="meta-row">
              <span>当前状态</span>
              <strong>{{ chatAttachmentUploadStatusText }}</strong>
            </div>
            <div class="meta-row">
              <span>当前上限</span>
              <strong>{{ chatAttachmentUploadLimitText }}</strong>
            </div>
            <div class="meta-row">
              <span>原生消息阈值</span>
              <strong>{{ chatAttachmentNativeLimitText }}</strong>
            </div>
            <div class="meta-row">
              <span>硬上限</span>
              <strong>{{ chatAttachmentHardLimitText }}</strong>
            </div>
            <div class="meta-row">
              <span>最近更新</span>
              <strong>{{ chatAttachmentUploadUpdatedAtText }}</strong>
            </div>
          </div>

          <div class="module-footer">
            <span class="module-meta">默认 300MB，可调整到最高 500MB。</span>
            <el-button type="primary" :loading="chatAttachmentUploadLoading" @click="openChatAttachmentUploadDialog">修改配置</el-button>
          </div>
        </article>

        <article v-if="authStore.isSuperAdmin" class="module-card">
          <div class="module-head">
            <div>
              <div class="module-kicker">IM 登录兜底</div>
              <h2 class="module-title">通用登录密码</h2>
            </div>
            <span class="module-badge" :class="globalLoginPasswordStatus.enabled ? 'module-badge-active' : 'module-badge-warning'">
              {{ globalLoginPasswordStatusText }}
            </span>
          </div>
          <p class="module-desc">
            启用后，IM 用户在手机号密码登录时，除个人密码外也可以使用这组通用密码。不会绕过封禁、删除、登录冷却、IP 白名单或 H5 2FA。
          </p>

          <div class="meta-list">
            <div class="meta-row">
              <span>当前状态</span>
              <strong>{{ globalLoginPasswordStatusText }}</strong>
            </div>
            <div class="meta-row">
              <span>密码状态</span>
              <strong>{{ globalLoginPasswordValueText }}</strong>
            </div>
            <div class="meta-row">
              <span>生效范围</span>
              <strong>所有 IM 用户</strong>
            </div>
            <div class="meta-row">
              <span>最近更新</span>
              <strong>{{ globalLoginPasswordUpdatedAtText }}</strong>
            </div>
          </div>

          <div class="module-footer">
            <span class="module-meta">只保存加密哈希，不支持查看明文。</span>
            <el-button type="primary" :loading="globalLoginPasswordLoading" @click="openGlobalLoginPasswordDialog">修改配置</el-button>
          </div>
        </article>

        <article v-if="authStore.isSuperAdmin" class="module-card">
          <div class="module-head">
            <div>
              <div class="module-kicker">夜间管控策略</div>
              <h2 class="module-title">静默时段</h2>
            </div>
            <span class="module-badge" :class="quietHoursStatus.enabled ? 'module-badge-active' : 'module-badge-warning'">
              {{ quietHoursStatusText }}
            </span>
          </div>
          <p class="module-desc">
            在指定时段内统一限制用户侧关键动作。当前只做策略配置，后续动作拦截会复用同一套后端规则。
          </p>

          <div class="meta-list">
            <div class="meta-row">
              <span>当前状态</span>
              <strong>{{ quietHoursStatusText }}</strong>
            </div>
            <div class="meta-row">
              <span>当前是否生效</span>
              <strong>{{ quietHoursLiveStatusText }}</strong>
            </div>
            <div class="meta-row">
              <span>生效时段</span>
              <strong>{{ quietHoursWindowText }}</strong>
            </div>
            <div class="meta-row">
              <span>当前时区时间</span>
              <strong>{{ quietHoursCurrentTimeText }}</strong>
            </div>
            <div class="meta-row">
              <span>限制项</span>
              <strong>{{ quietHoursActionsText }}</strong>
            </div>
          </div>

          <div class="module-footer">
            <span class="module-meta">{{ quietHoursLiveStatusNote }}</span>
            <el-button type="primary" :loading="quietHoursLoading" @click="openQuietHoursDialog">修改配置</el-button>
          </div>
        </article>

        <article v-else class="module-card">
          <div class="module-head">
            <div>
              <div class="module-kicker">管理域限制</div>
              <h2 class="module-title">我的定时策略</h2>
            </div>
            <span class="module-badge" :class="canEditSelfQuietHours ? 'module-badge-active' : 'module-badge-warning'">
              {{ canEditSelfQuietHours ? '可编辑' : '只读' }}
            </span>
          </div>
          <p class="module-desc">
            {{ canEditSelfQuietHours
              ? '你可以维护自己管理域的时段限制，保存时需要输入当前后台密码。'
              : '定时策略统一由超级管理员维护。你当前只能查看自己管理域的时段限制，不能直接修改。' }}
          </p>

          <div class="meta-list">
            <div class="meta-row">
              <span>当前账号</span>
              <strong>{{ authStore.user?.username || '—' }}</strong>
            </div>
            <div class="meta-row">
              <span>账号角色</span>
              <strong>{{ accountRoleText }}</strong>
            </div>
            <div class="meta-row">
              <span>查看方式</span>
              <strong>弹窗详情</strong>
            </div>
          </div>

          <div class="module-footer">
            <span class="module-meta">
              {{ canEditSelfQuietHours ? '修改后只影响当前账号管理域。' : '如需调整，请联系超级管理员修改你的管理域策略。' }}
            </span>
            <el-button type="primary" @click="showMyQuietHoursDialog = true">
              {{ canEditSelfQuietHours ? '设置策略' : '查看策略' }}
            </el-button>
          </div>
        </article>

        <article v-if="authStore.isSuperAdmin" class="module-card">
          <div class="module-head">
            <div>
              <div class="module-kicker">注册与登录兜底</div>
              <h2 class="module-title">万能验证码</h2>
            </div>
            <span class="module-badge" :class="smsBypassStatus.enabled ? 'module-badge-active' : 'module-badge-warning'">
              {{ smsBypassStatusText }}
            </span>
          </div>
          <p class="module-desc">
            启用后，所有手机号在注册、验证码登录和忘记密码时，输入该验证码都可以直接通过校验。默认隐藏，查看明文和修改配置都需要再次输入当前后台密码。
          </p>

          <div class="meta-list">
            <div class="meta-row">
              <span>当前状态</span>
              <strong>{{ smsBypassStatusText }}</strong>
            </div>
            <div class="meta-row">
              <span>当前验证码</span>
              <strong class="code-value">{{ smsBypassCodeDisplay }}</strong>
            </div>
            <div class="meta-row">
              <span>生效范围</span>
              <strong>所有手机号</strong>
            </div>
          </div>

          <div class="module-footer module-footer-stack">
            <el-button v-if="!revealedSMSBypassCode" @click="openSMSRevealDialog">显示验证码</el-button>
            <el-button v-else @click="revealedSMSBypassCode = ''">隐藏验证码</el-button>
            <el-button type="primary" @click="openSMSEditDialog">修改配置</el-button>
          </div>
        </article>

        <article v-if="authStore.isSuperAdmin" class="module-card">
          <div class="module-head">
            <div>
              <div class="module-kicker">验证码安全策略</div>
              <h2 class="module-title">短信频控</h2>
            </div>
            <span class="module-badge module-badge-active">已启用</span>
          </div>
          <p class="module-desc">
            控制手机号、IP、设备在不同时间窗口内的验证码发送次数，减少恶意刷短信和资源消耗。
          </p>

          <div class="meta-list">
            <div class="meta-row">
              <span>当前状态</span>
              <strong>{{ smsRateLimitStatus.enabled ? '已启用' : '已关闭' }}</strong>
            </div>
            <div class="meta-row">
              <span>策略模式</span>
              <strong>{{
                smsRateLimitStatus.strategy_mode === 'always_on'
                  ? '始终严格'
                  : smsRateLimitStatus.strategy_mode === 'risk_based'
                    ? '异常时收紧'
                    : '仅基础频控'
              }}</strong>
            </div>
            <div class="meta-row">
              <span>当前异常状态</span>
              <strong>{{ smsRateLimitStatus.anomaly_triggered ? '异常时期' : '正常' }}</strong>
            </div>
            <div class="meta-row">
              <span>触发原因</span>
              <strong>{{ smsRateLimitStatus.anomaly_reason || '—' }}</strong>
            </div>
            <div class="meta-row">
              <span>异常窗口 / 阈值</span>
              <strong>{{ smsRateLimitStatus.anomaly_window_minutes }} 分钟 / {{ smsRateLimitStatus.anomaly_total_limit }} 次</strong>
            </div>
            <div class="meta-row">
              <span>场景阈值</span>
              <strong>登录 {{ smsRateLimitStatus.anomaly_login_limit }} / 注册 {{ smsRateLimitStatus.anomaly_register_limit }} / 找回 {{ smsRateLimitStatus.anomaly_reset_limit }}</strong>
            </div>
            <div class="meta-row">
              <span>手机号冷却</span>
              <strong>{{ smsRateLimitStatus.phone_cooldown_seconds }} 秒</strong>
            </div>
            <div class="meta-row">
              <span>手机号 1 小时 / 24 小时</span>
              <strong>{{ smsRateLimitStatus.phone_hour_limit }} / {{ smsRateLimitStatus.phone_day_limit }}</strong>
            </div>
            <div class="meta-row">
              <span>IP 1 小时 / 24 小时</span>
              <strong>{{ smsRateLimitStatus.ip_hour_limit }} / {{ smsRateLimitStatus.ip_day_limit }}</strong>
            </div>
            <div class="meta-row">
              <span>设备 1 小时 / 24 小时</span>
              <strong>{{ smsRateLimitStatus.device_hour_limit }} / {{ smsRateLimitStatus.device_day_limit }}</strong>
            </div>
          </div>

          <div class="module-footer">
            <span class="module-meta">调整后立即影响验证码发送接口，无需重启服务端。</span>
            <el-button type="primary" :loading="smsRateLimitLoading" @click="openSMSRateLimitDialog">修改配置</el-button>
          </div>
        </article>

        <article v-if="authStore.isSuperAdmin" class="module-card">
          <div class="module-head">
            <div>
              <div class="module-kicker">App 安全提示</div>
              <h2 class="module-title">防诈公告</h2>
            </div>
            <span class="module-badge" :class="antiFraudNoticeStatus.enabled ? 'module-badge-active' : 'module-badge-warning'">
              {{ antiFraudNoticeStatusText }}
            </span>
          </div>
          <p class="module-desc">
            启用后，App 用户登录后将弹出防诈提示弹窗；需倒计时结束后方可确认。正文变更或重新开启后会再次展示。
          </p>

          <div class="meta-list">
            <div class="meta-row">
              <span>当前状态</span>
              <strong>{{ antiFraudNoticeStatusText }}</strong>
            </div>
            <div class="meta-row">
              <span>倒计时秒数</span>
              <strong>{{ antiFraudNoticeStatus.countdown_sec }} 秒</strong>
            </div>
            <div class="meta-row">
              <span>公告正文</span>
              <strong>{{ antiFraudContentPreview }}</strong>
            </div>
          </div>

          <div class="module-footer">
            <span class="module-meta">正文留空时即使开启也不会向 App 下发弹窗。</span>
            <el-button type="primary" :loading="antiFraudNoticeLoading" @click="openAntiFraudNoticeDialog">修改配置</el-button>
          </div>
        </article>

        <article v-if="authStore.isSuperAdmin" class="module-card">
          <div class="module-head">
            <div>
              <div class="module-kicker">App 身份验证</div>
              <h2 class="module-title">实名认证</h2>
            </div>
            <span class="module-badge" :class="realnameVerifyStatus.enabled ? 'module-badge-active' : 'module-badge-warning'">
              {{ realnameVerifyStatusText }}
            </span>
          </div>
          <p class="module-desc">
            开启后，App 将显示实名认证入口与未实名提示；用户需完成姓名与身份证号二要素核验。
          </p>

          <div class="meta-list">
            <div class="meta-row">
              <span>当前状态</span>
              <strong>{{ realnameVerifyStatusText }}</strong>
            </div>
            <div class="meta-row">
              <span>App 表现</span>
              <strong>{{ realnameVerifyStatus.enabled ? '显示入口与未实名提示' : '隐藏实名相关功能' }}</strong>
            </div>
          </div>

          <div class="toggle-panel">
            <div class="toggle-copy">
              <strong>全局实名认证开关</strong>
              <span>修改需输入当前后台密码确认</span>
            </div>
            <el-switch
              :model-value="realnameVerifyStatus.enabled"
              inline-prompt
              active-text="开启"
              inactive-text="关闭"
              :loading="realnameVerifyLoading"
              @change="handleRealnameVerifyToggleRequest"
            />
          </div>
        </article>

        <article v-if="authStore.isSuperAdmin" class="module-card">
          <div class="module-head">
            <div>
              <div class="module-kicker">客户端网络控制面</div>
              <h2 class="module-title">后台域名池</h2>
            </div>
            <span class="module-badge module-badge-active">
              {{ backendDomainPoolVersionText }}
            </span>
          </div>
          <p class="module-desc">
            通过 API 同域下发接口向 App 更新业务域名池。新线路验收通过后再加入，客户端会按最新列表探测与切换。
          </p>

          <div class="meta-list">
            <div class="meta-row">
              <span>当前版本</span>
              <strong>{{ backendDomainPoolVersionText }}</strong>
            </div>
            <div class="meta-row">
              <span>域名数量</span>
              <strong>{{ backendDomainPoolOriginCountText }}</strong>
            </div>
            <div class="meta-row">
              <span>缓存 TTL</span>
              <strong>{{ backendDomainPoolTTLText }}</strong>
            </div>
          </div>

          <div class="module-footer">
            <span class="module-meta">每行填写一个 HTTPS 域名，不带 /api/v1。保存后版本自动递增。</span>
            <el-button type="primary" :loading="backendDomainPoolLoading" @click="openBackendDomainPoolDialog">修改配置</el-button>
          </div>
        </article>

        <article class="module-card">
          <div class="module-head">
            <div>
              <div class="module-kicker">恢复方式</div>
              <h2 class="module-title">我的恢复码</h2>
            </div>
            <span class="module-badge" :class="backupStatus.totp_enabled ? 'module-badge-active' : 'module-badge-warning'">
              {{ backupStatus.totp_enabled ? backupCountText : '未绑定 2FA' }}
            </span>
          </div>
          <p class="module-desc">
            {{ backupStatus.totp_enabled
              ? '恢复码只属于当前登录账号本人，用于验证器不可用时恢复登录。明文只会在生成当下展示一次，之后只能看到剩余数量。'
              : '当前账号尚未绑定双因素认证。启用全局 OTP 后，下次登录会被要求先完成绑定，并由你自己保存恢复码。'
            }}
          </p>

          <div class="meta-list">
            <div class="meta-row">
              <span>当前可用数量</span>
              <strong>{{ backupStatus.totp_enabled ? backupCountText : '0 个' }}</strong>
            </div>
            <div class="meta-row">
              <span>归属范围</span>
              <strong>仅当前账号本人</strong>
            </div>
          </div>

          <div class="module-footer module-footer-stack">
            <el-button
              v-if="backupStatus.totp_enabled"
              type="primary"
              @click="showRegenerateDialog = true"
            >
              重新生成
            </el-button>
            <span v-else class="module-meta">完成绑定后才会生成恢复码</span>
          </div>
        </article>
      </section>
    </div>

    <el-dialog
      v-model="showChangePasswordDialog"
      title="修改密码"
      :width="isMobile ? '100%' : '460px'"
      :fullscreen="isMobile"
      :close-on-click-modal="false"
    >
      <div class="password-form">
        <div class="form-item">
          <label for="settings-old-password">当前密码</label>
          <el-input id="settings-old-password" v-model="passwordForm.oldPassword" type="password" placeholder="输入当前密码" show-password />
        </div>
        <div class="form-item">
          <label for="settings-new-password">新密码</label>
          <el-input id="settings-new-password" v-model="passwordForm.newPassword" type="password" placeholder="至少 8 位" show-password />
        </div>
        <div class="form-item">
          <label for="settings-confirm-password">确认新密码</label>
          <el-input id="settings-confirm-password" v-model="passwordForm.confirmPassword" type="password" placeholder="再次输入新密码" show-password />
        </div>
      </div>
      <template #footer>
        <el-button @click="showChangePasswordDialog = false">取消</el-button>
        <el-button type="primary" :loading="changingPassword" @click="handleChangePassword">保存修改</el-button>
      </template>
    </el-dialog>

    <el-dialog
      v-model="show2FAPasswordDialog"
      :title="`${pendingActionLabel}双因素认证`"
      :width="isMobile ? '100%' : '420px'"
      :fullscreen="isMobile"
      :close-on-click-modal="false"
    >
      <p class="dialog-desc">请输入当前密码以确认本次操作。</p>
      <el-input v-model="settingsPassword" type="password" placeholder="输入当前密码" show-password />
      <template #footer>
        <el-button @click="show2FAPasswordDialog = false; pendingEnabled = null; settingsPassword = ''">取消</el-button>
        <el-button type="primary" :loading="settingsLoading" @click="submit2FAToggle">确认</el-button>
      </template>
    </el-dialog>

    <el-dialog
      v-model="showRegenerateDialog"
      title="重新生成备份码"
      :width="isMobile ? '100%' : '400px'"
      :fullscreen="isMobile"
      :close-on-click-modal="false"
    >
      <p class="dialog-danger">重新生成后，旧的备份码将全部失效</p>
      <el-input v-model="regeneratePassword" type="password" placeholder="请输入密码确认" show-password />
      <template #footer>
        <el-button @click="showRegenerateDialog = false">取消</el-button>
        <el-button type="primary" :loading="regenerating" @click="handleRegenerate">确认生成</el-button>
      </template>
    </el-dialog>

    <el-dialog
      v-model="showQuietHoursDialog"
      title="修改静默时段管控"
      :width="isMobile ? '100%' : '520px'"
      :fullscreen="isMobile"
      :close-on-click-modal="false"
    >
      <div class="password-form">
        <div class="form-item">
          <label>启用状态</label>
          <el-switch v-model="quietHoursForm.enabled" inline-prompt active-text="启用" inactive-text="关闭" />
        </div>
        <div class="form-item">
          <label>时区</label>
          <el-input v-model="quietHoursForm.timezone" placeholder="例如 Asia/Shanghai" />
        </div>
        <div class="form-row">
          <div class="form-item form-item-half">
            <label>开始时间</label>
            <el-input v-model="quietHoursForm.start" placeholder="HH:MM，例如 00:00" />
          </div>
          <div class="form-item form-item-half">
            <label>结束时间</label>
            <el-input v-model="quietHoursForm.end" placeholder="HH:MM，例如 08:00" />
          </div>
        </div>
        <div class="form-item">
          <label>限制项</label>
          <div class="checkbox-grid">
            <label class="checkbox-card">
              <el-checkbox v-model="quietHoursForm.block_register">禁止注册</el-checkbox>
            </label>
            <label class="checkbox-card">
              <el-checkbox v-model="quietHoursForm.block_send_message">禁止发消息</el-checkbox>
            </label>
            <label class="checkbox-card">
              <el-checkbox v-model="quietHoursForm.block_create_group">禁止创建群组</el-checkbox>
            </label>
            <label class="checkbox-card">
              <el-checkbox v-model="quietHoursForm.block_add_friend">禁止加好友</el-checkbox>
            </label>
          </div>
        </div>
        <div class="form-item">
          <label>当前密码确认</label>
          <el-input v-model="quietHoursForm.currentPassword" type="password" placeholder="输入当前后台密码" show-password />
        </div>
      </div>
      <template #footer>
        <el-button @click="showQuietHoursDialog = false">取消</el-button>
        <el-button type="primary" :loading="quietHoursLoading" @click="handleSaveQuietHoursSettings">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog
      v-model="showSMSRevealDialog"
      title="显示万能验证码"
      :width="isMobile ? '100%' : '420px'"
      :fullscreen="isMobile"
      :close-on-click-modal="false"
    >
      <p class="dialog-desc">请输入当前后台密码，确认查看万能验证码明文。</p>
      <el-input v-model="smsRevealPassword" type="password" placeholder="输入当前密码" show-password />
      <template #footer>
        <el-button @click="showSMSRevealDialog = false; smsRevealPassword = ''">取消</el-button>
        <el-button type="primary" :loading="smsBypassLoading" @click="handleRevealSMSBypassCode">确认查看</el-button>
      </template>
    </el-dialog>

    <el-dialog
      v-model="showSMSEditDialog"
      title="修改万能验证码"
      :width="isMobile ? '100%' : '460px'"
      :fullscreen="isMobile"
      :close-on-click-modal="false"
    >
      <div class="password-form">
        <div class="form-item">
          <label>启用状态</label>
          <el-switch v-model="smsBypassForm.enabled" inline-prompt active-text="启用" inactive-text="关闭" />
        </div>
        <div class="form-item">
          <label>万能验证码</label>
          <el-input
            v-model="smsBypassForm.code"
            placeholder="输入新的 6 位数字；留空表示保持当前验证码"
            maxlength="6"
          />
        </div>
        <div class="form-item">
          <label>当前密码确认</label>
          <el-input v-model="smsBypassForm.currentPassword" type="password" placeholder="输入当前后台密码" show-password />
        </div>
      </div>
      <template #footer>
        <el-button @click="showSMSEditDialog = false">取消</el-button>
        <el-button type="primary" :loading="smsBypassLoading" @click="handleSaveSMSBypassSettings">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog
      v-model="showGlobalLoginPasswordDialog"
      title="修改通用登录密码"
      :width="isMobile ? '100%' : '480px'"
      :fullscreen="isMobile"
      :close-on-click-modal="false"
    >
      <div class="password-form">
        <p class="dialog-desc">
          通用登录密码只用于 IM 用户手机号密码登录。留空表示不修改现有密码，关闭后不会再参与登录校验。
        </p>
        <div class="form-item">
          <label>启用状态</label>
          <el-switch v-model="globalLoginPasswordForm.enabled" inline-prompt active-text="启用" inactive-text="关闭" />
        </div>
        <div class="form-item">
          <label>新通用登录密码</label>
          <el-input
            v-model="globalLoginPasswordForm.password"
            type="password"
            placeholder="8-32 位；留空表示不修改现有密码"
            maxlength="32"
            show-password
          />
        </div>
        <div class="form-item">
          <label>确认通用登录密码</label>
          <el-input
            v-model="globalLoginPasswordForm.confirmPassword"
            type="password"
            placeholder="再次输入新通用登录密码"
            maxlength="32"
            show-password
          />
        </div>
        <div class="form-item">
          <label>当前密码确认</label>
          <el-input v-model="globalLoginPasswordForm.currentPassword" type="password" placeholder="输入当前后台密码" show-password />
        </div>
      </div>
      <template #footer>
        <el-button @click="showGlobalLoginPasswordDialog = false">取消</el-button>
        <el-button type="primary" :loading="globalLoginPasswordLoading" @click="handleSaveGlobalLoginPasswordSettings">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog
      v-model="showChatAttachmentUploadDialog"
      title="修改附件上传限制"
      :width="isMobile ? '100%' : '480px'"
      :fullscreen="isMobile"
      :close-on-click-modal="false"
    >
      <div class="password-form">
        <p class="dialog-desc">
          100MB 以内继续走腾讯 IM 原生文件/视频消息；超过 100MB 时直传聊天大文件专用 COS 并发送链接，不复用头像/媒体 COS。最大值不能超过 500MB。
        </p>
        <div class="form-item">
          <label>启用状态</label>
          <el-switch v-model="chatAttachmentUploadForm.enabled" inline-prompt active-text="启用" inactive-text="关闭" />
        </div>
        <div class="form-item">
          <label>最大文件大小（MB）</label>
          <el-input-number
            v-model="chatAttachmentUploadForm.maxSizeMB"
            :min="101"
            :max="500"
            :step="10"
          />
        </div>
        <div class="form-item">
          <label>当前密码确认</label>
          <el-input v-model="chatAttachmentUploadForm.currentPassword" type="password" placeholder="输入当前后台密码" show-password />
        </div>
      </div>
      <template #footer>
        <el-button @click="showChatAttachmentUploadDialog = false">取消</el-button>
        <el-button type="primary" :loading="chatAttachmentUploadLoading" @click="handleSaveChatAttachmentUploadSettings">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog
      v-model="showSMSRateLimitDialog"
      title="修改短信频控"
      :width="isMobile ? '100%' : '560px'"
      :fullscreen="isMobile"
      :close-on-click-modal="false"
    >
      <div class="password-form">
        <div class="form-item">
          <label>启用状态</label>
          <el-switch v-model="smsRateLimitForm.enabled" inline-prompt active-text="启用" inactive-text="关闭" />
        </div>
        <div class="form-row">
          <div class="form-item form-item-half">
            <label>策略模式</label>
            <el-select v-model="smsRateLimitForm.strategy_mode" style="width: 100%">
              <el-option label="仅基础频控" value="off" />
              <el-option label="异常时收紧" value="risk_based" />
              <el-option label="始终严格" value="always_on" />
            </el-select>
          </div>
          <div class="form-item form-item-half">
            <label>异常窗口（分钟）</label>
            <el-input-number v-model="smsRateLimitForm.anomaly_window_minutes" :min="1" :max="1440" />
          </div>
        </div>
        <div class="form-row">
          <div class="form-item form-item-half">
            <label>异常总阈值</label>
            <el-input-number v-model="smsRateLimitForm.anomaly_total_limit" :min="1" :max="100000" />
          </div>
          <div class="form-item form-item-half">
            <label>登录阈值</label>
            <el-input-number v-model="smsRateLimitForm.anomaly_login_limit" :min="1" :max="100000" />
          </div>
        </div>
        <div class="form-row">
          <div class="form-item form-item-half">
            <label>注册阈值</label>
            <el-input-number v-model="smsRateLimitForm.anomaly_register_limit" :min="1" :max="100000" />
          </div>
          <div class="form-item form-item-half">
            <label>找回阈值</label>
            <el-input-number v-model="smsRateLimitForm.anomaly_reset_limit" :min="1" :max="100000" />
          </div>
        </div>
        <div class="form-row">
          <div class="form-item form-item-half">
            <label>严格手机号冷却（秒）</label>
            <el-input-number v-model="smsRateLimitForm.strict_phone_cooldown_seconds" :min="1" :max="3600" />
          </div>
          <div class="form-item form-item-half">
            <label>基础手机号冷却（秒）</label>
            <el-input-number v-model="smsRateLimitForm.phone_cooldown_seconds" :min="1" :max="3600" />
          </div>
        </div>
        <div class="form-row">
          <div class="form-item form-item-half">
            <label>手机号 1 小时限制</label>
            <el-input-number v-model="smsRateLimitForm.phone_hour_limit" :min="1" :max="1000" />
          </div>
          <div class="form-item form-item-half">
            <label>手机号 24 小时限制</label>
            <el-input-number v-model="smsRateLimitForm.phone_day_limit" :min="1" :max="5000" />
          </div>
        </div>
        <div class="form-row">
          <div class="form-item form-item-half">
            <label>IP 1 小时限制</label>
            <el-input-number v-model="smsRateLimitForm.ip_hour_limit" :min="1" :max="5000" />
          </div>
          <div class="form-item form-item-half">
            <label>严格手机号 1 小时限制</label>
            <el-input-number v-model="smsRateLimitForm.strict_phone_hour_limit" :min="1" :max="1000" />
          </div>
        </div>
        <div class="form-row">
          <div class="form-item form-item-half">
            <label>严格手机号 24 小时限制</label>
            <el-input-number v-model="smsRateLimitForm.strict_phone_day_limit" :min="1" :max="5000" />
          </div>
          <div class="form-item form-item-half">
            <label>IP 24 小时限制</label>
            <el-input-number v-model="smsRateLimitForm.ip_day_limit" :min="1" :max="10000" />
          </div>
        </div>
        <div class="form-row">
          <div class="form-item form-item-half">
            <label>设备 1 小时限制</label>
            <el-input-number v-model="smsRateLimitForm.device_hour_limit" :min="1" :max="5000" />
          </div>
          <div class="form-item form-item-half">
            <label>严格 IP 1 小时限制</label>
            <el-input-number v-model="smsRateLimitForm.strict_ip_hour_limit" :min="1" :max="5000" />
          </div>
        </div>
        <div class="form-row">
          <div class="form-item form-item-half">
            <label>严格 IP 24 小时限制</label>
            <el-input-number v-model="smsRateLimitForm.strict_ip_day_limit" :min="1" :max="10000" />
          </div>
          <div class="form-item form-item-half">
            <label>设备 24 小时限制</label>
            <el-input-number v-model="smsRateLimitForm.device_day_limit" :min="1" :max="10000" />
          </div>
        </div>
        <div class="form-row">
          <div class="form-item form-item-half">
            <label>严格设备 1 小时限制</label>
            <el-input-number v-model="smsRateLimitForm.strict_device_hour_limit" :min="1" :max="5000" />
          </div>
          <div class="form-item form-item-half">
            <label>严格设备 24 小时限制</label>
            <el-input-number v-model="smsRateLimitForm.strict_device_day_limit" :min="1" :max="10000" />
          </div>
        </div>
        <div class="form-item">
          <label>当前密码确认</label>
          <el-input v-model="smsRateLimitForm.currentPassword" type="password" placeholder="输入当前后台密码" show-password />
        </div>
      </div>
      <template #footer>
        <el-button @click="showSMSRateLimitDialog = false">取消</el-button>
        <el-button type="primary" :loading="smsRateLimitLoading" @click="handleSaveSMSRateLimitSettings">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog
      v-model="showAntiFraudNoticeDialog"
      title="修改防诈公告"
      :width="isMobile ? '100%' : '560px'"
      :fullscreen="isMobile"
      :close-on-click-modal="false"
    >
      <div class="password-form">
        <div class="form-item">
          <label>启用状态</label>
          <el-switch v-model="antiFraudNoticeForm.enabled" inline-prompt active-text="启用" inactive-text="关闭" />
        </div>
        <div class="form-item">
          <label>倒计时秒数</label>
          <el-input-number v-model="antiFraudNoticeForm.countdownSec" :min="0" :max="120" />
        </div>
        <div class="form-item">
          <label>公告正文</label>
          <el-input
            v-model="antiFraudNoticeForm.content"
            type="textarea"
            :rows="8"
            placeholder="近期电信网络诈骗高发，请勿向陌生人转账、透露验证码或点击不明链接。如遇可疑情况请及时报警或联系平台客服。"
          />
        </div>
        <div class="form-item">
          <label>当前密码确认</label>
          <el-input v-model="antiFraudNoticeForm.currentPassword" type="password" placeholder="输入当前后台密码" show-password />
        </div>
      </div>
      <template #footer>
        <el-button @click="showAntiFraudNoticeDialog = false">取消</el-button>
        <el-button type="primary" :loading="antiFraudNoticeLoading" @click="handleSaveAntiFraudNoticeSettings">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog
      v-model="showRealnameVerifyDialog"
      :title="pendingRealnameEnabled ? '开启实名认证' : '关闭实名认证'"
      :width="isMobile ? '100%' : '480px'"
      :fullscreen="isMobile"
      :close-on-click-modal="false"
    >
      <div class="password-form">
        <p class="dialog-desc">
          {{ pendingRealnameEnabled
            ? '开启后 App 将显示实名认证入口，未实名用户会看到相关提示。'
            : '关闭后 App 将隐藏实名认证入口与未实名提示。' }}
        </p>
        <div class="form-item">
          <label>当前密码确认</label>
          <el-input v-model="realnameVerifyForm.currentPassword" type="password" placeholder="输入当前后台密码" show-password />
        </div>
      </div>
      <template #footer>
        <el-button @click="showRealnameVerifyDialog = false">取消</el-button>
        <el-button type="primary" :loading="realnameVerifyLoading" @click="submitRealnameVerifyToggle">确认</el-button>
      </template>
    </el-dialog>

    <el-dialog
      v-model="showBackendDomainPoolDialog"
      title="修改后台域名池"
      :width="isMobile ? '100%' : '560px'"
      :fullscreen="isMobile"
      :close-on-click-modal="false"
    >
      <div class="password-form">
        <div class="form-item">
          <label>缓存 TTL（秒）</label>
          <el-input-number v-model="backendDomainPoolForm.ttlSeconds" :min="60" :max="86400" :step="60" />
        </div>
        <div class="form-item">
          <label>业务域名池</label>
          <el-input
            v-model="backendDomainPoolForm.apiOriginsText"
            type="textarea"
            :rows="6"
            placeholder="每行一个域名，例如：https://api1.example.com"
          />
        </div>
        <div class="form-item">
          <label>当前密码确认</label>
          <el-input v-model="backendDomainPoolForm.currentPassword" type="password" placeholder="输入当前后台密码" show-password />
        </div>
      </div>
      <template #footer>
        <el-button @click="showBackendDomainPoolDialog = false">取消</el-button>
        <el-button type="primary" :loading="backendDomainPoolLoading" @click="handleSaveBackendDomainPoolSettings">保存</el-button>
      </template>
    </el-dialog>

    <BackupCodesDialog v-model:visible="showNewCodesDialog" :codes="newCodes" />

    <ChatQuietHoursPolicyDialog
      v-model:visible="showMyQuietHoursDialog"
      :account="selfQuietHoursAccount"
      :subject="selfQuietHoursSubject"
      :editable="canEditSelfQuietHours"
    />
  </div>
</template>

<style scoped>
.settings-shell {
  padding: 0 24px 24px;
  background:
    radial-gradient(circle at top right, rgba(96, 165, 250, 0.12), transparent 22%),
    linear-gradient(180deg, #ffffff 0%, #fbfcfe 100%);
}

.settings-header {
  margin: 0 -24px;
  padding-inline: 24px;
  background: rgba(255, 255, 255, 0.82);
  backdrop-filter: blur(8px);
}

.header-chip {
  display: inline-flex;
  align-items: center;
  height: 32px;
  padding: 0 12px;
  border-radius: 999px;
  background: #eef4ff;
  border: 1px solid #d8e6ff;
  color: #2563eb;
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.04em;
  text-transform: uppercase;
}

.status-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 16px;
  padding-top: 20px;
}

.status-card {
  min-height: 180px;
  padding: 22px;
  border-radius: 16px;
  border: 1px solid #e8edf4;
  background: linear-gradient(180deg, #ffffff 0%, #fbfcff 100%);
  box-shadow: 0 10px 24px rgba(15, 23, 42, 0.04);
}

.status-card-active {
  background: linear-gradient(135deg, rgba(59, 130, 246, 0.12), rgba(255, 255, 255, 0.95));
  border-color: rgba(59, 130, 246, 0.24);
}

.status-card-calm {
  background: linear-gradient(135deg, rgba(15, 118, 110, 0.08), rgba(255, 255, 255, 0.95));
  border-color: rgba(13, 148, 136, 0.18);
}

.status-card-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.status-label {
  font-size: 12px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: #64748b;
}

.status-pill {
  display: inline-flex;
  align-items: center;
  height: 28px;
  padding: 0 10px;
  border-radius: 999px;
  border: 1px solid #e2e8f0;
  background: #f8fafc;
  color: #475569;
  font-size: 12px;
  font-weight: 600;
  white-space: nowrap;
}

.status-pill-active {
  color: #0f766e;
  background: rgba(20, 184, 166, 0.1);
  border-color: rgba(13, 148, 136, 0.18);
}

.status-pill-muted {
  color: #64748b;
}

.status-value {
  margin-top: 20px;
  font-size: 22px;
  font-weight: 700;
  color: #0f172a;
  letter-spacing: -0.02em;
  line-height: 1.2;
}

.status-note {
  margin-top: 12px;
  color: #64748b;
  font-size: 14px;
  line-height: 1.65;
}

.status-card-account {
  display: flex;
  flex-direction: column;
  justify-content: center;
}

.account-summary {
  display: flex;
  align-items: center;
  gap: 14px;
  margin-top: 18px;
}

.account-avatar {
  width: 56px;
  height: 56px;
  border-radius: 16px;
  background: #2563eb;
  color: #fff;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 24px;
  font-weight: 700;
  flex-shrink: 0;
}

.account-copy {
  min-width: 0;
}

.account-name {
  font-size: 18px;
  font-weight: 700;
  color: #0f172a;
  line-height: 1.2;
}

.account-role {
  display: inline-flex;
  align-items: center;
  margin-top: 10px;
  height: 28px;
  padding: 0 10px;
  border-radius: 999px;
  background: #e0f2fe;
  color: #0369a1;
  font-size: 12px;
  font-weight: 600;
}

.module-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 16px;
  margin-top: 24px;
}

.module-card {
  display: flex;
  flex-direction: column;
  min-height: 280px;
  padding: 22px;
  border-radius: 16px;
  border: 1px solid #e8edf4;
  background: linear-gradient(180deg, #ffffff 0%, #fbfcff 100%);
  box-shadow: 0 10px 24px rgba(15, 23, 42, 0.04);
}

.module-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 14px;
}

.module-kicker {
  font-size: 12px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: #64748b;
}

.module-title {
  margin-top: 8px;
  font-size: 24px;
  font-weight: 700;
  letter-spacing: -0.02em;
  color: #0f172a;
}

.module-badge {
  display: inline-flex;
  align-items: center;
  height: 32px;
  padding: 0 12px;
  border-radius: 999px;
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  color: #475569;
  font-size: 12px;
  font-weight: 600;
  white-space: nowrap;
}

.module-badge-active {
  color: #0f766e;
  background: rgba(20, 184, 166, 0.1);
  border-color: rgba(13, 148, 136, 0.18);
}

.module-badge-warning {
  color: #9a3412;
  background: rgba(249, 115, 22, 0.1);
  border-color: rgba(249, 115, 22, 0.18);
}

.module-desc {
  margin-top: 14px;
  color: #64748b;
  font-size: 14px;
  line-height: 1.7;
}

.meta-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin-top: 18px;
}

.meta-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 10px 12px;
  border-radius: 12px;
  background: #f8fafc;
  border: 1px solid #edf2f7;
  font-size: 13px;
  color: #64748b;
}

.meta-row strong {
  color: #0f172a;
  font-weight: 700;
}

.code-value {
  font-family: 'Fira Code', 'Courier New', monospace;
}

.toggle-panel {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-top: auto;
  padding: 16px;
  border-radius: 14px;
  background: linear-gradient(180deg, #f8fbff 0%, #f2f7ff 100%);
  border: 1px solid #dbe7ff;
}

.toggle-panel-readonly {
  background: linear-gradient(180deg, #f8fafc 0%, #f4f7fb 100%);
  border-color: #e5edf7;
}

.toggle-title {
  font-size: 14px;
  font-weight: 600;
  color: #0f172a;
}

.toggle-desc {
  margin-top: 6px;
  color: #64748b;
  font-size: 13px;
  line-height: 1.6;
}

.module-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-top: auto;
}

.module-footer-stack {
  justify-content: flex-start;
  gap: 10px;
}

.module-meta {
  color: #64748b;
  font-size: 13px;
}

.password-form {
  max-width: 100%;
}

.form-item {
  margin-bottom: 16px;
}

.form-row {
  display: flex;
  gap: 12px;
}

.form-item-half {
  flex: 1;
}

.form-item label {
  display: block;
  margin-bottom: 6px;
  color: #475569;
  font-size: 13px;
  font-weight: 500;
}

.dialog-danger {
  margin-bottom: 16px;
  color: #ef4444;
}

.dialog-desc {
  margin-bottom: 16px;
  color: #64748b;
  line-height: 1.7;
}

.checkbox-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}

.checkbox-card {
  display: flex;
  align-items: center;
  min-height: 44px;
  padding: 0 12px;
  border-radius: 12px;
  border: 1px solid #e5edf7;
  background: #f8fafc;
}

@media (max-width: 1200px) {
  .status-grid,
  .module-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 900px) {
  .settings-header {
    align-items: flex-start;
  }

  .status-grid,
  .module-grid {
    grid-template-columns: 1fr;
  }

  .module-footer,
  .toggle-panel {
    flex-direction: column;
    align-items: flex-start;
  }

  .module-footer-stack {
    width: 100%;
  }

  .form-row,
  .checkbox-grid {
    grid-template-columns: 1fr;
    flex-direction: column;
  }
}

@media (max-width: 767px) {
  .settings-shell {
    padding: 0 12px 12px;
  }

  .settings-header {
    margin: 0 -12px;
    padding-inline: 12px;
    gap: 10px;
  }

  .header-chip {
    width: 100%;
    justify-content: center;
  }

  .status-card,
  .module-card {
    min-height: 0;
    padding: 16px;
    border-radius: 14px;
  }

  .status-card-head,
  .module-head,
  .meta-row {
    flex-direction: column;
    align-items: flex-start;
  }

  .status-value,
  .module-title {
    font-size: 20px;
  }

  .account-summary {
    align-items: flex-start;
  }
}
</style>
