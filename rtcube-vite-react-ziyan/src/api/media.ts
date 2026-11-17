import { chatGet } from './chatHttp'
import { routedFetch } from './transport'
import { putFileToCos } from './attachment'
import { getToken } from './session'

export type AvatarUploadTarget =
  | { type?: 'user'; userId: string }
  | { type: 'group'; groupId: string }

type AvatarUploadResult = {
  avatar_url?: string
  url?: string
  put_url?: string
}

function avatarQuery(target: AvatarUploadTarget) {
  if (target.type === 'group') {
    return `type=group&group_id=${encodeURIComponent(target.groupId)}`
  }
  return `type=user&user_id=${encodeURIComponent(target.userId)}`
}

async function uploadAvatarForm(file: File, target: AvatarUploadTarget): Promise<string> {
  const token = getToken()
  const form = new FormData()
  form.append('file', file)
  if (target.type === 'group') {
    form.append('type', 'group')
    form.append('group_id', target.groupId)
  } else {
    form.append('type', 'user')
    form.append('user_id', target.userId)
  }
  const res = await routedFetch('/upload/avatar', {
    method: 'POST', headers: token ? { Authorization: `Bearer ${token}` } : {}, body: form,
  })
  // A definitive missing endpoint may use the existing presign flow. An ambiguous
  // write failure, auth rejection or rate limit never triggers a second upload.
  if (res.status === 404 || res.status === 405) return uploadAvatarPresign(file, target)
  const json = await res.json().catch(() => ({})) as { code?: number; data?: AvatarUploadResult; message?: string; msg?: string }
  const url = json.data?.avatar_url || json.data?.url
  if (res.ok && (json.code === 0 || json.code === 200) && url) return String(url)
  throw new Error(json.message || json.msg || `头像上传失败（HTTP ${res.status}）`)
}

async function uploadAvatarPresign(file: File, target: AvatarUploadTarget): Promise<string> {
  const presign = await chatGet<AvatarUploadResult>(`/upload/avatar/presign?${avatarQuery(target)}`)
  const putUrl = String(presign.put_url || '')
  const avatarUrl = String(presign.avatar_url || presign.url || '')
  if (!putUrl || !avatarUrl) throw new Error('头像上传地址无效')
  await putFileToCos(putUrl, file, { contentType: file.type || 'image/jpeg' })
  return avatarUrl
}

/** 上传头像，返回可写入资料的 URL */
export async function uploadAvatarFile(file: File, target: AvatarUploadTarget): Promise<string> {
  return uploadAvatarForm(file, target)
}
