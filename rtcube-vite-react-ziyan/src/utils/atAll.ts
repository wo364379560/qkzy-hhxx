/** 产品展示文案；腾讯 SDK / 旧消息仍可能写成「所有人」 */
export const AT_ALL_LABEL = '全体成员'
export const AT_ALL_ALIASES = ['全体成员', '所有人'] as const

const AT_ALL_TEXT_RE = /@(?:全体成员|所有人)/

export function isAtAllLabel(value: string): boolean {
  return (AT_ALL_ALIASES as readonly string[]).includes(String(value || '').trim())
}

/** 正文里出现 @全体成员 或 @所有人，都视为 @all（中文不能用 \b） */
export function textHasAtAll(text: string): boolean {
  return AT_ALL_TEXT_RE.test(String(text || ''))
}

/** 气泡 / 列表统一展示为 @全体成员 */
export function displayAtAllText(text: string): string {
  return String(text || '').replace(/@所有人/g, `@${AT_ALL_LABEL}`)
}

export function isAtAllUserId(id: string): boolean {
  const raw = String(id || '').trim()
  if (isAtAllLabel(raw)) return true
  const s = raw.toUpperCase().replace(/[_-]/g, '')
  return s.includes('ATALL') || s.includes('MESSAGEATALL')
}
