export function normalizeAdminUsername(value: string): string {
  const username = value.trim()
  if (!/^[A-Za-z0-9]{3,32}$/.test(username)) {
    throw new Error('登录账号需为 3–32 位英文字母或数字，中文请填写在昵称中')
  }
  return username
}
