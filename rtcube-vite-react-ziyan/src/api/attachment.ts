import { chatGet, chatPost } from './chatHttp'

export type ChatAttachmentUploadSettings = {
  enabled: boolean
  max_size_mb: number
  hard_limit_mb: number
  native_limit_mb: number
  put_url_ttl_seconds: number
  get_url_ttl_seconds: number
  retention_days: number
}

export type ChatAttachmentPresignResult = {
  file_id: string
  file_name?: string
  put_url: string
  download_url: string
  preview_url: string
  object_key: string
  content_type?: string
  max_size_mb: number
  native_limit_mb: number
  is_video: boolean
  expires_at?: string
}

export type ChatAttachmentAccessResult = {
  url: string
  download_url?: string
  preview_url?: string
  file_id: string
  file_name: string
  file_size: number
  content_type: string
  is_video: boolean
  expires_at?: string
}

export function normalizeChatAttachmentConversationID(conversationID: string) {
  const raw = String(conversationID || '').trim()
  if (raw.startsWith('c2c_') || raw.startsWith('group_')) return raw
  if (raw.startsWith('C2C')) return `c2c_${raw.slice(3)}`
  if (raw.startsWith('GROUP')) return `group_${raw.slice(5)}`
  return raw
}

export async function getChatAttachmentUploadSettings() {
  return chatGet<ChatAttachmentUploadSettings>('/upload/attachment/settings')
}

export async function getChatAttachmentPresignUrl(
  file: File,
  conversationID: string,
  options?: { isVideo?: boolean; contentType?: string },
) {
  const isVideo = options?.isVideo === true
  const contentType =
    options?.contentType ||
    (isVideo && !String(file.type || '').toLowerCase().startsWith('video/')
      ? 'video/mp4'
      : file.type || 'application/octet-stream')
  return chatPost<ChatAttachmentPresignResult>('/upload/attachment/presign', {
    conversation_id: normalizeChatAttachmentConversationID(conversationID),
    file_name: file.name,
    file_size: file.size,
    content_type: contentType,
    is_video: isVideo,
  })
}

export async function completeChatAttachmentUpload(fileID: string, maxAttempts = 5) {
  let last: unknown
  for (let attempt = 1; attempt <= maxAttempts; attempt++) {
    try {
      await chatPost('/upload/attachment/complete', { file_id: fileID }, true)
      return
    } catch (e) {
      last = e
      const msg = e instanceof Error ? e.message : String(e || '')
      if (attempt >= maxAttempts || !msg.includes('文件尚未上传完成')) throw e
      await new Promise((r) => window.setTimeout(r, 250 * attempt))
    }
  }
  throw last instanceof Error ? last : new Error('确认上传失败')
}

export async function getChatAttachmentAccess(fileID: string, download = false) {
  const id = encodeURIComponent(fileID.trim())
  const qs = download ? '?download=1' : ''
  return chatGet<ChatAttachmentAccessResult>(`/files/${id}/access${qs}`)
}

/** 直传 COS：body 用 File，不要先读成 ArrayBuffer。 */
export async function putFileToCos(
  putUrl: string,
  file: File,
  options?: { contentType?: string; onProgress?: (percent: number) => void },
) {
  const contentType = options?.contentType || file.type || 'application/octet-stream'
  const localPath = (file as File & { path?: string }).path
  if (localPath && window.desktop?.putToCos) {
    options?.onProgress?.(0.05)
    await window.desktop.putToCos({ url: putUrl, filePath: localPath, contentType })
    options?.onProgress?.(1)
    return
  }

  const devProxy = Boolean(import.meta.env.DEV)
  await new Promise<void>((resolve, reject) => {
    const xhr = new XMLHttpRequest()
    xhr.open('PUT', devProxy ? '/__cos_put' : putUrl)
    if (devProxy) xhr.setRequestHeader('X-Cos-Put-Url', putUrl)
    xhr.setRequestHeader('Content-Type', contentType)
    xhr.upload.onprogress = (e) => {
      if (!e.lengthComputable) return
      const total = e.total || file.size
      if (!total) return
      options?.onProgress?.(Math.max(0, Math.min(1, e.loaded / total)))
    }
    xhr.onload = () => {
      if (xhr.status >= 200 && xhr.status < 300) {
        options?.onProgress?.(1)
        resolve()
        return
      }
      const hint = String(xhr.responseText || '').replace(/\s+/g, ' ').slice(0, 120)
      reject(new Error(hint || `云端上传失败（${xhr.status}）`))
    }
    xhr.onerror = () => reject(new Error('云端上传失败，请检查网络'))
    xhr.onabort = () => reject(new Error('已取消上传'))
    xhr.send(file)
  })
}

export async function uploadChatAttachment(
  file: File,
  conversationID: string,
  options?: {
    isVideo?: boolean
    contentType?: string
    onProgress?: (percent: number) => void
  },
) {
  const presign = await getChatAttachmentPresignUrl(file, conversationID, {
    isVideo: options?.isVideo,
    contentType: options?.contentType,
  })
  await putFileToCos(presign.put_url, file, {
    contentType: presign.content_type || options?.contentType || file.type || 'application/octet-stream',
    onProgress: (p) => options?.onProgress?.(p * 0.96),
  })
  await completeChatAttachmentUpload(presign.file_id, 5)
  options?.onProgress?.(1)
  return presign
}
