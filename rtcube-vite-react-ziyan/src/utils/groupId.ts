/** 普通群 Public：9 位数字群 ID（100000000–999999999，不以 0 开头） */
export function generatePublicGroupId(): string {
  const min = 100_000_000
  const max = 999_999_999
  return String(Math.floor(Math.random() * (max - min + 1)) + min)
}

/** @deprecated 兼容旧调用 → 普通群 */
export function generateGroupId(): string {
  return generatePublicGroupId()
}

/**
 * 社群 Community 自定义群号示例（可选）。
 * 腾讯要求必须以 `@TGS#` 开头；也可留空由 SDK 自动分配。
 */
export function generateCommunityGroupId(): string {
  const chars = 'ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnpqrstuvwxyz23456789'
  let suffix = ''
  for (let i = 0; i < 10; i++) {
    suffix += chars[Math.floor(Math.random() * chars.length)]
  }
  return `@TGS#${suffix}`
}

export function isCommunityGroupId(id: string): boolean {
  return /^@TGS#/i.test(id.trim())
}

/**
 * @returns 规范化后的群号；社群留空时返回 undefined（交给 SDK 自动生成 @TGS#…）
 */
export function normalizeGroupId(
  raw: string | undefined,
  kind: 'community' | 'public' = 'public',
): string | undefined {
  const trimmed = raw?.trim() ?? ''

  if (kind === 'community') {
    if (!trimmed) return undefined
    if (!isCommunityGroupId(trimmed)) {
      throw new Error('社群群号必须以 @TGS# 开头，或留空由系统自动分配')
    }
    // 统一大写前缀
    return trimmed.replace(/^@tgs#/i, '@TGS#')
  }

  const id = trimmed || generatePublicGroupId()
  if (!/^\d{9}$/.test(id) || id.startsWith('0')) {
    throw new Error('普通群群号须为 9 位数字且不能以 0 开头')
  }
  return id
}
