import request from './request'
import type { ApiResponse, AdminUser } from '@/types'

export interface LoginParams {
  username: string
  password: string
  turnstile_token?: string
}

export interface ChangePasswordParams {
  oldPassword: string
  newPassword: string
}

export interface LoginResult {
  token?: string
  user?: AdminUser
  require_totp_setup?: boolean
  require_totp_verify?: boolean
  temp_token?: string
  totp_secret?: string
  totp_uri?: string
}

export interface BindTOTPResult {
  token: string
  user: AdminUser
  backup_codes: string[]
}

export interface VerifyTOTPResult {
  token: string
  user: AdminUser
}

export interface BackupCodesStatus {
  totp_enabled: boolean
  remaining_count: number
}

export interface RegenerateBackupCodesResult {
  backup_codes: string[]
}

export interface Set2FAParams {
  enabled: boolean
  current_password: string
  confirm_disable_all?: boolean
}

export interface SMSBypassSettings {
  enabled: boolean
  has_code: boolean
  masked_code: string
}

export interface GlobalLoginPasswordSettings {
  enabled: boolean
  has_password: boolean
  updated_at?: string
}

export interface ChatAttachmentUploadSettings {
  enabled: boolean
  max_size_mb: number
  hard_limit_mb: number
  native_limit_mb: number
  put_url_ttl_seconds: number
  get_url_ttl_seconds: number
  retention_days: number
  updated_at?: string
}

export interface SMSRateLimitSettings {
  enabled: boolean
  strategy_mode: 'off' | 'risk_based' | 'always_on'
  anomaly_window_minutes: number
  anomaly_total_limit: number
  anomaly_login_limit: number
  anomaly_register_limit: number
  anomaly_reset_limit: number
  anomaly_triggered?: boolean
  anomaly_reason?: string
  phone_cooldown_seconds: number
  phone_hour_limit: number
  phone_day_limit: number
  ip_hour_limit: number
  ip_day_limit: number
  device_hour_limit: number
  device_day_limit: number
  strict_phone_cooldown_seconds: number
  strict_phone_hour_limit: number
  strict_phone_day_limit: number
  strict_ip_hour_limit: number
  strict_ip_day_limit: number
  strict_device_hour_limit: number
  strict_device_day_limit: number
}

export interface ChatQuietHoursSettings {
  enabled: boolean
  timezone: string
  start: string
  end: string
  block_register: boolean
  block_send_message: boolean
  block_create_group: boolean
  block_add_friend: boolean
}

export interface SetSMSBypassParams {
  enabled: boolean
  current_password: string
  code?: string
}

export interface SetGlobalLoginPasswordParams {
  enabled: boolean
  current_password: string
  password?: string
}

export interface SetChatAttachmentUploadParams {
  enabled: boolean
  max_size_mb: number
  current_password: string
}

export interface SetChatQuietHoursParams extends ChatQuietHoursSettings {
  current_password: string
}

export interface SetSMSRateLimitParams extends SMSRateLimitSettings {
  current_password: string
}

export interface BackendDomainPoolSettings {
  version: number
  ttl_seconds: number
  api_origins: string[]
}

export interface SetBackendDomainPoolParams {
  ttl_seconds: number
  api_origins: string[]
  current_password: string
}

export function login(params: LoginParams) {
  return request.post<ApiResponse<LoginResult>>('/login', params)
}

export function logout() {
  return request.post<ApiResponse<null>>('/logout')
}

export function changePassword(params: ChangePasswordParams) {
  return request.post<ApiResponse<null>>('/change-password', params)
}

export function getBackupCodes() {
  return request.get<ApiResponse<BackupCodesStatus>>('/totp/backup-codes')
}

export function regenerateBackupCodes(password: string) {
  return request.post<ApiResponse<RegenerateBackupCodesResult>>('/totp/regenerate-backup-codes', { current_password: password })
}

export function bindTOTP(tempToken: string, code: string) {
  return request.post<ApiResponse<BindTOTPResult>>(
    '/totp/bind',
    { code },
    { headers: { Authorization: `Bearer ${tempToken}` } }
  )
}

export function verifyTOTP(tempToken: string, payload: { code?: string; backup_code?: string }) {
  return request.post<ApiResponse<VerifyTOTPResult>>(
    '/totp/verify',
    payload,
    { headers: { Authorization: `Bearer ${tempToken}` } }
  )
}

export function get2FASettings() {
  return request.get<ApiResponse<{ enabled: boolean }>>('/settings/2fa')
}

export function set2FASettings(params: Set2FAParams) {
  return request.post<ApiResponse<null>>('/settings/2fa', params)
}

export function getChatQuietHoursSettings() {
  return request.get<ApiResponse<ChatQuietHoursSettings>>('/settings/chat-quiet-hours')
}

export function setChatQuietHoursSettings(params: SetChatQuietHoursParams) {
  return request.post<ApiResponse<null>>('/settings/chat-quiet-hours', params)
}

export function getSMSBypassSettings() {
  return request.get<ApiResponse<SMSBypassSettings>>('/settings/sms-bypass')
}

export function revealSMSBypassCode(currentPassword: string) {
  return request.post<ApiResponse<{ code: string }>>('/settings/sms-bypass/reveal', {
    current_password: currentPassword,
  })
}

export function setSMSBypassSettings(params: SetSMSBypassParams) {
  return request.post<ApiResponse<null>>('/settings/sms-bypass', params)
}

export function getGlobalLoginPasswordSettings() {
  return request.get<ApiResponse<GlobalLoginPasswordSettings>>('/settings/global-login-password')
}

export function setGlobalLoginPasswordSettings(params: SetGlobalLoginPasswordParams) {
  return request.post<ApiResponse<null>>('/settings/global-login-password', params)
}

export function getChatAttachmentUploadSettings() {
  return request.get<ApiResponse<ChatAttachmentUploadSettings>>('/settings/chat-attachment-upload')
}

export function setChatAttachmentUploadSettings(params: SetChatAttachmentUploadParams) {
  return request.post<ApiResponse<null>>('/settings/chat-attachment-upload', params)
}

export function getBackendDomainPoolSettings() {
  return request.get<ApiResponse<BackendDomainPoolSettings>>('/settings/backend-domain-pool')
}

export function setBackendDomainPoolSettings(params: SetBackendDomainPoolParams) {
  return request.post<ApiResponse<BackendDomainPoolSettings>>('/settings/backend-domain-pool', params)
}

export function getSMSRateLimitSettings() {
  return request.get<ApiResponse<SMSRateLimitSettings>>('/settings/sms-rate-limit')
}

export function setSMSRateLimitSettings(params: SetSMSRateLimitParams) {
  return request.post<ApiResponse<null>>('/settings/sms-rate-limit', params)
}

export interface AntiFraudNoticeSettings {
  enabled: boolean
  content: string
  countdown_sec: number
}

export interface SetAntiFraudNoticeParams extends AntiFraudNoticeSettings {
  current_password: string
}

export interface RealnameVerifySettings {
  enabled: boolean
}

export interface SetRealnameVerifyParams {
  enabled: boolean
  current_password: string
}

export function getAntiFraudNoticeSettings() {
  return request.get<ApiResponse<AntiFraudNoticeSettings>>('/settings/anti-fraud-notice')
}

export function setAntiFraudNoticeSettings(params: SetAntiFraudNoticeParams) {
  return request.post<ApiResponse<null>>('/settings/anti-fraud-notice', params)
}

export function getRealnameVerifySettings() {
  return request.get<ApiResponse<RealnameVerifySettings>>('/settings/realname-verify')
}

export function setRealnameVerifySettings(params: SetRealnameVerifyParams) {
  return request.post<ApiResponse<null>>('/settings/realname-verify', params)
}
