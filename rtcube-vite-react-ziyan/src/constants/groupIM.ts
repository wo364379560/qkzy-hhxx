/** 禁看成员资料：优先规格 inner_attr；兼容旧 canglian key */
export const BLOCK_MEMBER_PROFILE_KEYS = [
  'inner_attr_member_profile_access',
  'canglian_block_member_profile',
] as const

export type JoinInviteOption =
  | 'FreeAccess'
  | 'NeedPermission'
  | 'DisableApply'
  | 'DisableInvite'

export const JOIN_OPTION_LABELS: Record<string, string> = {
  DisableApply: '禁止加群',
  NeedPermission: '需要验证',
  FreeAccess: '允许任何人加群',
}

export const INVITE_OPTION_LABELS: Record<string, string> = {
  DisableInvite: '禁止成员邀请',
  NeedPermission: '需要验证',
  FreeAccess: '允许邀请',
}

/** 腾讯角色可能是 Admin / Owner，也可能带 GRP_ 前缀。 */
export function isGroupManagerRole(role: string): boolean {
  const r = String(role || '').toLowerCase()
  return r.includes('owner') || r.includes('admin')
}

/**
 * 邀请进群：禁止邀请时谁都不能拉；需要验证时仅群主/管理员可拉；
 * 允许邀请时普通成员也可以拉。
 */
export function canInviteGroupMembers(inviteOption: string, canManage: boolean): boolean {
  if (inviteOption === 'DisableInvite') return false
  if (inviteOption === 'NeedPermission') return canManage
  return true
}

/** 建群默认社群；可选普通群 Public */
export const GROUP_KIND_LABELS = {
  community: '社群（Community）',
  public: '普通群（Public）',
} as const

/** 群自定义属性：发言限频（秒/条，0=不限制） */
export const SPEAK_FREQ_ATTR_KEY = 'canglian_speak_freq_sec'

export const SPEAK_FREQ_OPTIONS = [
  { sec: 0, label: '不限制发言' },
  { sec: 5, label: '5秒1条' },
  { sec: 10, label: '10秒1条' },
  { sec: 30, label: '30秒1条' },
  { sec: 60, label: '1分钟1条' },
] as const

export function speakFreqLabel(sec: number): string {
  return SPEAK_FREQ_OPTIONS.find((x) => x.sec === sec)?.label || '不限制发言'
}

export const DEFAULT_GROUP_KIND = 'community' as const
