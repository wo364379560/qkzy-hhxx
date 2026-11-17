import { useEffect, useState, type MouseEvent } from 'react'
import {
  downloadMediaAuto,
  getMediaCache,
  openCachedMedia,
  revealCachedMedia,
  subscribeMediaCache,
} from '@/im/mediaCache'
import { parseLargeAttachment, resolveMessageAttachmentUrl } from '@/im/largeAttachment'
import { fileInfoFromMessage, formatFileSize } from '@/im/mediaPayload'
import type { ChatMessage } from '@/im/types'
import styles from './MessageList.module.css'

function fileKind(name: string) {
  const ext = name.split('.').pop()?.toLowerCase() || ''
  if (['xls', 'xlsx', 'csv'].includes(ext)) return { label: 'X', tone: styles.fileIconXls }
  if (['doc', 'docx'].includes(ext)) return { label: 'W', tone: styles.fileIconDoc }
  if (ext === 'pdf') return { label: 'P', tone: styles.fileIconPdf }
  if (['zip', 'rar', '7z'].includes(ext)) return { label: 'Z', tone: styles.fileIconZip }
  if (['apk', 'ipa'].includes(ext)) return { label: 'A', tone: styles.fileIconApk }
  if (['ppt', 'pptx'].includes(ext)) return { label: 'P', tone: styles.fileIconPpt }
  return { label: '📄', tone: styles.fileIconDef }
}

export function MediaHoverActs({
  msgId,
  url,
  name,
  mine,
  msg,
}: {
  msgId: string
  url: string
  name: string
  mine: boolean
  msg?: ChatMessage
}) {
  const [, bump] = useState(0)
  const [busy, setBusy] = useState(false)
  useEffect(() => subscribeMediaCache(() => bump((n) => n + 1)), [])
  const cached = getMediaCache(msgId)
  const downloaded = Boolean(cached?.filePath || cached?.blobUrl || mine)

  async function resolveUrl() {
    if (msg && parseLargeAttachment(msg)?.fileID) {
      return resolveMessageAttachmentUrl(msg, true)
    }
    if (url) return url
    throw new Error('文件地址无效')
  }

  async function onDownload(e: MouseEvent) {
    e.stopPropagation()
    setBusy(true)
    try {
      await downloadMediaAuto(msgId, await resolveUrl(), name)
    } catch (err) {
      alert(err instanceof Error ? err.message : '下载失败')
    } finally {
      setBusy(false)
    }
  }

  async function onReveal(e: MouseEvent) {
    e.stopPropagation()
    if (!cached) return
    try {
      await revealCachedMedia(cached)
    } catch (err) {
      alert(err instanceof Error ? err.message : '无法定位文件')
    }
  }

  return (
    <div className={`${styles.fileActs} ${mine ? styles.fileActsMine : ''}`}>
      {cached?.filePath ? (
        <button type="button" className={styles.fileLocate} title="打开文件位置" onClick={(e) => void onReveal(e)}>
          <i />
        </button>
      ) : null}
      {!mine && !downloaded ? (
        <button type="button" className={styles.fileDown} title="下载到本机" disabled={busy} onClick={(e) => void onDownload(e)}>
          {busy ? '…' : <i />}
        </button>
      ) : null}
    </div>
  )
}

export function FileBubble({
  msg,
  mine,
  multi,
  onOpenVideo,
}: {
  msg: ChatMessage
  mine: boolean
  multi?: boolean
  onOpenVideo?: (url: string, name: string) => void
}) {
  const info = fileInfoFromMessage(msg)
  const att = parseLargeAttachment(msg)
  const [, bump] = useState(0)
  const [busy, setBusy] = useState(false)

  useEffect(() => subscribeMediaCache(() => bump((n) => n + 1)), [])

  const cached = getMediaCache(msg.ID)
  const downloaded = Boolean(cached?.filePath || cached?.blobUrl || (mine && (msg.localPreview || info.url)))
  const status = mine ? '已发送' : downloaded ? '已下载' : ''
  const kind = fileKind(info.name)

  async function resolveUrl(download = true) {
    if (att?.fileID) return resolveMessageAttachmentUrl(msg, download)
    if (info.url) return info.url
    throw new Error('文件地址无效')
  }

  async function onDownload(e: MouseEvent) {
    e.stopPropagation()
    setBusy(true)
    try {
      const url = await resolveUrl(true)
      await downloadMediaAuto(msg.ID, url, info.name)
    } catch (err) {
      alert(err instanceof Error ? err.message : '下载失败')
    } finally {
      setBusy(false)
    }
  }

  async function onOpen(e: MouseEvent) {
    e.stopPropagation()
    if (multi) return
    if (att?.isVideo && onOpenVideo) {
      setBusy(true)
      try {
        const url = await resolveUrl(false)
        onOpenVideo(url, info.name)
      } catch (err) {
        alert(err instanceof Error ? err.message : '无法打开视频')
      } finally {
        setBusy(false)
      }
      return
    }
    if (cached) {
      try {
        await openCachedMedia(cached, info.url)
      } catch (err) {
        alert(err instanceof Error ? err.message : '打开失败')
      }
      return
    }
    if (mine && (msg.localPreview || (info.url && !att?.fileID))) {
      const href = msg.localPreview || info.url
      if (href) window.open(href, '_blank', 'noopener,noreferrer')
      return
    }
    await onDownload(e)
  }

  async function onReveal(e: MouseEvent) {
    e.stopPropagation()
    if (!cached) return
    try {
      await revealCachedMedia(cached)
    } catch (err) {
      alert(err instanceof Error ? err.message : '无法定位文件')
    }
  }

  const locate = cached?.filePath ? (
    <button type="button" className={styles.fileLocate} title="打开文件位置" onClick={(e) => void onReveal(e)}>
      <i />
    </button>
  ) : null

  return (
    <div className={`${styles.fileRow} ${mine ? styles.fileRowMine : ''}`}>
      {mine ? locate : null}
      <button type="button" className={styles.fileBubble} onClick={(e) => void onOpen(e)}>
        <span className={styles.fileText}>
          <b title={info.name}>{info.name}</b>
          <em>
            {formatFileSize(info.size) || '文件'}
            {status ? ` ${status}` : ''}
          </em>
        </span>
        {!mine && !downloaded ? (
          <span
            className={styles.fileDown}
            title="下载到本机"
            role="button"
            onClick={(e) => void onDownload(e)}
          >
            {busy ? '…' : <i />}
          </span>
        ) : (
          <span className={`${styles.fileTypeIcon} ${kind.tone}`} aria-hidden>
            {kind.label}
          </span>
        )}
      </button>
      {!mine ? locate : null}
    </div>
  )
}
