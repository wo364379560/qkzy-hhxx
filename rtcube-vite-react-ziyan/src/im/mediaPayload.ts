/** 从 SDK Message payload 提取媒体 URL / 元信息 */

import { parseLargeAttachment } from './largeAttachment'

// eslint-disable-next-line @typescript-eslint/no-explicit-any
type AnyPayload = any
// eslint-disable-next-line @typescript-eslint/no-explicit-any
type AnyMsg = any

export function messageTypeKey(msg: AnyMsg): string {
  return String(msg?.type || msg?.elemType || '').toUpperCase()
}

export function isImageMessage(msg: AnyMsg): boolean {
  const t = messageTypeKey(msg)
  if (t.includes('IMAGE') && !t.includes('VIDEO')) return true
  const p = msg?.payload as AnyPayload
  return Array.isArray(p?.imageInfoArray) && p.imageInfoArray.length > 0
}

export function isVideoMessage(msg: AnyMsg): boolean {
  const att = parseLargeAttachment(msg)
  if (att?.isVideo) return true
  const t = messageTypeKey(msg)
  if (t.includes('VIDEO')) return true
  const p = msg?.payload as AnyPayload
  return Boolean(p?.remoteVideoUrl || p?.videoUrl || (p?.snapshotUrl && p?.videoSecond != null))
}

export function isFileMessage(msg: AnyMsg): boolean {
  if (isVideoMessage(msg)) return false
  const att = parseLargeAttachment(msg)
  if (att && !att.isVideo) return true
  const t = messageTypeKey(msg)
  if (t.includes('FILE')) return true
  const p = msg?.payload as AnyPayload
  return Boolean(p?.fileName || p?.fileUrl)
}

export function isSoundMessage(msg: AnyMsg): boolean {
  const t = messageTypeKey(msg)
  if (t.includes('SOUND') || t.includes('AUDIO')) return true
  const p = msg?.payload as AnyPayload
  return Boolean(p?.second != null && (p?.url || p?.remoteAudioUrl || p?.audioUrl))
}

export function isTextMessage(msg: AnyMsg): boolean {
  if (parseLargeAttachment(msg)) return false
  const t = messageTypeKey(msg)
  return t.includes('TEXT') || Boolean(msg?.payload?.text && !isImageMessage(msg) && !isMergerMessage(msg))
}

export function imageUrlFromMessage(msg: AnyMsg): string {
  const p = msg.payload as AnyPayload
  const arr = p?.imageInfoArray as
    | Array<{ url?: string; imageUrl?: string; sizeType?: number }>
    | undefined
  let remote = ''
  if (Array.isArray(arr) && arr.length) {
    const withUrl = arr.filter((x) => x?.url || x?.imageUrl)
    const pick =
      withUrl.find((x) => Number(x.sizeType) === 1) ||
      withUrl.find((x) => Number(x.sizeType) === 0) ||
      withUrl[0] ||
      arr[0]
    remote = String(pick?.url || pick?.imageUrl || '')
  }
  if (!remote) remote = String(p?.imageUrl || p?.url || '')
  if (remote && !remote.startsWith('blob:')) return remote
  return remote || String(msg.localPreview || '')
}

export function videoInfoFromMessage(msg: AnyMsg): {
  url: string
  snapshot: string
} {
  const att = parseLargeAttachment(msg)
  if (att?.isVideo) {
    return {
      url: String(att.previewURL || att.downloadURL || msg.localPreview || ''),
      snapshot: String(att.coverURL || ''),
    }
  }
  const p = msg.payload as AnyPayload
  return {
    url: String(p?.remoteVideoUrl || p?.videoUrl || p?.url || p?.videoFileUrl || msg.localPreview || ''),
    snapshot: String(p?.snapshotUrl || p?.thumbUrl || p?.coverUrl || p?.snapshotPath || ''),
  }
}

export function fileInfoFromMessage(msg: AnyMsg): {
  name: string
  size: number
  url: string
} {
  const att = parseLargeAttachment(msg)
  if (att) {
    return {
      name: att.fileName || '文件',
      size: att.fileSize,
      url: String(att.downloadURL || att.previewURL || msg.localPreview || ''),
    }
  }
  const p = msg.payload as AnyPayload
  return {
    name: String(p?.fileName || p?.name || '文件'),
    size: Number(p?.fileSize || p?.size || 0),
    url: String(
      p?.fileUrl ||
        p?.url ||
        p?.downloadUrl ||
        p?.remoteUrl ||
        p?.remoteFileUrl ||
        p?.fileDownloadUrl ||
        msg.localPreview ||
        '',
    ),
  }
}

export function soundInfoFromMessage(msg: AnyMsg): {
  url: string
  second: number
} {
  const p = msg.payload as AnyPayload
  return {
    url: String(
      p?.url ||
        p?.remoteAudioUrl ||
        p?.audioUrl ||
        p?.remoteUrl ||
        p?.fileUrl ||
        p?.downloadUrl ||
        msg.localPreview ||
        '',
    ),
    second: Number(p?.second || p?.duration || 0),
  }
}

export function openOrDownloadUrl(url: string, fileName?: string) {
  const href = String(url || '').trim()
  if (!href) throw new Error('文件地址无效，无法打开')
  const a = document.createElement('a')
  a.href = href
  a.target = '_blank'
  a.rel = 'noopener noreferrer'
  if (fileName) a.download = fileName
  document.body.appendChild(a)
  a.click()
  a.remove()
}

export function formatFileSize(n: number) {
  if (!n || n < 0) return ''
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`
  if (n < 1024 * 1024 * 1024) return `${(n / 1024 / 1024).toFixed(1)} MB`
  return `${(n / 1024 / 1024 / 1024).toFixed(1)} GB`
}

export function isMergerMessage(msg: AnyMsg): boolean {
  const t = messageTypeKey(msg)
  if (t.includes('RELAY') || t.includes('MERGER')) return true
  const p = msg?.payload as AnyPayload
  return Boolean(p?.title && (p?.messageList || p?.abstractList))
}

export function mergerInfoFromMessage(msg: AnyMsg): {
  title: string
  abstracts: string[]
} {
  const p = msg?.payload as AnyPayload
  const abstracts = Array.isArray(p?.abstractList)
    ? (p.abstractList as unknown[]).map(String)
    : []
  return {
    title: String(p?.title || '聊天记录'),
    abstracts,
  }
}
