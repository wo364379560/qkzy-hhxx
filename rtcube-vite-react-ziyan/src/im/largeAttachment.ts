import { getChatAttachmentAccess } from '@/api/attachment'

export const LARGE_ATTACHMENT_BUSINESS_ID = 'chat_large_attachment'
export const CN_ATTACHMENT_V1_PREFIX = 'CN_ATTACHMENT_V1:'
export const NATIVE_FILE_LIMIT_BYTES = 100 * 1024 * 1024
export const FORCE_COS_VIDEO_DURATION_SEC = 180
export const DEFAULT_MAX_ATTACHMENT_MB = 300

export type LargeAttachmentInfo = {
  fileID: string
  fileName: string
  fileSize: number
  contentType: string
  isVideo: boolean
  downloadURL: string
  previewURL: string
  coverURL?: string
  durationSeconds?: number
}

const accessCache = new Map<string, { url: string; preview: string; exp: number }>()

function str(record: Record<string, unknown>, keys: string[], fallback = '') {
  for (const key of keys) {
    const value = record[key]
    if (value === undefined || value === null) continue
    const text = String(value).trim()
    if (text) return text
  }
  return fallback
}

function num(record: Record<string, unknown>, keys: string[]) {
  for (const key of keys) {
    const value = Number(record[key] || 0)
    if (Number.isFinite(value) && value > 0) return value
  }
  return 0
}

function bool(record: Record<string, unknown>, keys: string[]) {
  for (const key of keys) {
    const value = record[key]
    if (typeof value === 'boolean') return value
    if (typeof value === 'number') return value !== 0
    const text = String(value || '').trim().toLowerCase()
    if (text === 'true' || text === '1') return true
  }
  return false
}

function asRecord(input: unknown): Record<string, unknown> | null {
  if (input && typeof input === 'object' && !Array.isArray(input)) {
    return input as Record<string, unknown>
  }
  if (typeof input === 'string' && input.trim()) {
    try {
      const parsed = JSON.parse(input) as unknown
      if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
        return parsed as Record<string, unknown>
      }
    } catch {
      return null
    }
  }
  return null
}

function fromRecord(record: Record<string, unknown>): LargeAttachmentInfo | null {
  const fileID = str(record, ['fileID', 'file_id', 'fileId'])
  const fileName = str(record, ['fileName', 'file_name', 'name'])
  if (!fileID || !fileName) return null
  const contentType = str(record, ['contentType', 'content_type', 'mimeType', 'mime_type'], 'application/octet-stream')
  const downloadURL = str(record, ['downloadURL', 'download_url', 'url', 'fileUrl', 'file_url'])
  const previewURL = str(record, ['previewURL', 'preview_url']) || downloadURL
  const isVideo =
    bool(record, ['isVideo', 'is_video']) || contentType.toLowerCase().startsWith('video/')
  const duration = num(record, ['durationSeconds', 'duration_seconds', 'duration', 'videoSecond'])
  const coverURL = str(record, ['coverURL', 'cover_url', 'coverUrl']) || undefined
  return {
    fileID,
    fileName,
    fileSize: num(record, ['fileSize', 'file_size', 'size']),
    contentType,
    isVideo,
    downloadURL,
    previewURL,
    coverURL,
    durationSeconds: duration || undefined,
  }
}

function parseCNAttachmentV1(text: string): LargeAttachmentInfo | null {
  if (!text.includes(CN_ATTACHMENT_V1_PREFIX)) return null
  for (const line of text.split(/\r?\n/)) {
    const trimmed = line.trim()
    if (!trimmed.startsWith(CN_ATTACHMENT_V1_PREFIX)) continue
    const raw = trimmed.slice(CN_ATTACHMENT_V1_PREFIX.length).trim()
    const record = asRecord(raw)
    if (!record) return null
    return fromRecord(record)
  }
  return null
}

function parseCustomPayload(raw: unknown): LargeAttachmentInfo | null {
  const root = asRecord(raw)
  if (!root) return null
  const business = str(root, ['businessID', 'business_id', 'businessId'])
  const candidates: Record<string, unknown>[] = [root]
  for (const key of ['attachment', 'file', 'payload', 'data', 'customData']) {
    const nested = asRecord(root[key])
    if (nested) candidates.push(nested)
  }
  for (const candidate of candidates) {
    const bid = str(candidate, ['businessID', 'business_id', 'businessId']) || business
    const hit = fromRecord(candidate)
    if (!hit) continue
    if (bid && bid !== LARGE_ATTACHMENT_BUSINESS_ID) continue
    if (bid === LARGE_ATTACHMENT_BUSINESS_ID || hit.fileID) return hit
  }
  return null
}

export function parseLargeAttachment(msg: {
  type?: string
  payload?: {
    text?: string
    data?: unknown
    description?: string
    extension?: string
    fileName?: string
    fileSize?: number
    fileID?: string
  }
}): LargeAttachmentInfo | null {
  const payload = msg?.payload
  if (!payload) return null
  const ext = String(payload.extension || '')
  const fromCustom = parseCustomPayload(payload.data)
  if (fromCustom) return fromCustom
  if (ext === LARGE_ATTACHMENT_BUSINESS_ID || String(payload.description || '').includes('[文件]') || String(payload.description || '').includes('[视频]')) {
    const fromDesc = parseCustomPayload(payload.data)
    if (fromDesc) return fromDesc
  }
  const text = String(payload.text || '')
  const fromText = parseCNAttachmentV1(text)
  if (fromText) return fromText
  if (payload.fileID && payload.fileName) {
    return fromRecord(payload as Record<string, unknown>)
  }
  return null
}

export function isLargeAttachmentMessage(msg: unknown): boolean {
  return Boolean(parseLargeAttachment(msg as Parameters<typeof parseLargeAttachment>[0]))
}

export function shouldRouteToCOS(fileSize: number, videoDurationSeconds = 0, isVideo = false) {
  if (fileSize >= NATIVE_FILE_LIMIT_BYTES) return true
  if (isVideo && videoDurationSeconds >= FORCE_COS_VIDEO_DURATION_SEC) return true
  return false
}

export function isNativeSizeLimitError(err: unknown) {
  const extra = err as { code?: number; message?: string; data?: { code?: number; message?: string } }
  const raw = extra?.message || extra?.data?.message || (err instanceof Error ? err.message : String(err || ''))
  const lower = raw.toLowerCase()
  const code =
    Number(extra?.code) ||
    Number(extra?.data?.code) ||
    Number(/(\d{4,5})/.exec(raw)?.[1]) ||
    0
  return (
    code === 7005 ||
    raw.includes('7005') ||
    raw.includes('104857600') ||
    lower.includes('file size exceeds limit') ||
    lower.includes('filesize exceed') ||
    raw.includes('文件大小超出') ||
    raw.includes('最大限制是100')
  )
}

export function buildLargeAttachmentPayload(info: LargeAttachmentInfo) {
  return {
    businessID: LARGE_ATTACHMENT_BUSINESS_ID,
    version: 1,
    fileID: info.fileID,
    fileName: info.fileName,
    fileSize: info.fileSize,
    contentType: info.contentType,
    isVideo: info.isVideo,
    downloadURL: info.downloadURL,
    previewURL: info.previewURL || info.downloadURL,
    ...(info.coverURL ? { coverURL: info.coverURL } : {}),
    ...(info.durationSeconds ? { durationSeconds: info.durationSeconds } : {}),
  }
}

export function largeAttachmentDescription(info: Pick<LargeAttachmentInfo, 'isVideo' | 'fileName'>) {
  return `${info.isVideo ? '[视频]' : '[文件]'} ${info.fileName}`
}

export function readVideoDuration(file: File): Promise<number> {
  if (!file) return Promise.resolve(0)
  return new Promise((resolve) => {
    const url = URL.createObjectURL(file)
    const video = document.createElement('video')
    const done = (sec: number) => {
      URL.revokeObjectURL(url)
      resolve(Number.isFinite(sec) && sec > 0 ? Math.round(sec) : 0)
    }
    const timer = window.setTimeout(() => done(0), 4000)
    video.preload = 'metadata'
    video.onloadedmetadata = () => {
      window.clearTimeout(timer)
      done(Number(video.duration) || 0)
    }
    video.onerror = () => {
      window.clearTimeout(timer)
      done(0)
    }
    video.src = url
  })
}

function cacheKey(fileID: string, download: boolean) {
  return `${download ? 'dl' : 'pv'}:${fileID}`
}

export async function resolveAttachmentUrl(fileID: string, download = false): Promise<string> {
  const id = fileID.trim()
  if (!id) throw new Error('文件信息不完整')
  const key = cacheKey(id, download)
  const hit = accessCache.get(key)
  if (hit && hit.exp > Date.now() + 10_000) {
    return download ? hit.url : hit.preview || hit.url
  }
  const access = await getChatAttachmentAccess(id, download)
  const url = String(access.download_url || access.url || '').trim()
  const preview = String(access.preview_url || access.url || url).trim()
  if (!url && !preview) throw new Error('无法获取文件地址')
  const exp = Date.now() + 4 * 60 * 1000
  accessCache.set(cacheKey(id, true), { url: url || preview, preview: preview || url, exp })
  accessCache.set(cacheKey(id, false), { url: url || preview, preview: preview || url, exp })
  return download ? url || preview : preview || url
}

export async function resolveLargeAttachmentUrl(
  info: LargeAttachmentInfo | null | undefined,
  download = false,
) {
  if (!info) throw new Error('文件信息不完整')
  const existing = download ? info.downloadURL : info.previewURL || info.downloadURL
  if (existing.startsWith('http')) return existing
  return resolveAttachmentUrl(info.fileID, download)
}

export function isVideoFileName(name: string, mime = '') {
  if (String(mime || '').toLowerCase().startsWith('video/')) return true
  return /\.(mp4|mov|m4v|avi|mkv|webm|3gp|3gpp|flv|wmv)$/i.test(name)
}

export function explainChatAttachmentError(err: unknown) {
  const raw = err instanceof Error ? err.message : String(err || '')
  if (
    raw.includes('COS 配置未完成') ||
    raw.includes('cos config is incomplete') ||
    /COS_CHAT/i.test(raw)
  ) {
    return '大文件 COS 未配置：服务器缺少 COS_CHAT_SECRET_ID / COS_CHAT_SECRET_KEY / COS_CHAT_BUCKET / COS_CHAT_REGION。必须用独立私有桶，不能套头像或媒体那套 COS。'
  }
  if (raw.includes('聊天大文件上传已关闭') || raw.includes('文件上传功能已关闭')) {
    return '大文件上传已关闭，请在后台「附件上传限制」里开启。'
  }
  if (raw.includes('云端上传失败') || raw.includes('Failed to fetch') || /HTTP 0/.test(raw)) {
    return raw.includes('云端上传失败')
      ? `${raw}。若后台已配 COS_CHAT_*，请确认该桶 CORS 允许当前页面 PUT。`
      : raw
  }
  return raw.trim() || '发送失败'
}

export async function resolveMessageAttachmentUrl(
  msg: Parameters<typeof parseLargeAttachment>[0],
  download = false,
) {
  const att = parseLargeAttachment(msg)
  if (!att?.fileID) return ''
  try {
    return await resolveAttachmentUrl(att.fileID, download)
  } catch {
    const fallback = download ? att.downloadURL : att.previewURL || att.downloadURL
    if (fallback.startsWith('http')) return fallback
    throw new Error('无法获取文件地址')
  }
}
