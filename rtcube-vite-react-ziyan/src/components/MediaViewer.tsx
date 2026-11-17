import { useEffect, useState } from 'react'
import { openSavedMedia, saveMedia, type SavedMedia } from '@/im/mediaDownload'
import styles from './MediaViewer.module.css'

export type MediaPreviewItem = {
  kind: 'image' | 'video' | 'file'
  url: string
  name: string
}

type Props = {
  item: MediaPreviewItem | null
  onClose: () => void
}

export function MediaViewer({ item, onClose }: Props) {
  const [busy, setBusy] = useState('')
  const [saved, setSaved] = useState<SavedMedia | null>(null)

  useEffect(() => {
    setBusy('')
    setSaved(null)
  }, [item?.url])

  useEffect(() => {
    if (!item) return
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [item, onClose])

  if (!item) return null

  async function onDownload() {
    if (!item) return
    setBusy('下载中…')
    try {
      const next = await saveMedia(item.url, item.name)
      setSaved(next)
    } catch (e) {
      alert(e instanceof Error ? e.message : '下载失败')
    } finally {
      setBusy('')
    }
  }

  async function onOpen() {
    if (!saved) return
    try {
      await openSavedMedia(saved)
    } catch (e) {
      alert(e instanceof Error ? e.message : '打开失败')
    }
  }

  return (
    <div className={styles.mask} onClick={onClose} role="presentation">
      <div className={styles.box} onClick={(e) => e.stopPropagation()}>
        <button type="button" className={styles.close} onClick={onClose}>
          ×
        </button>
        {item.kind === 'image' ? (
          <img className={styles.img} src={item.url} alt={item.name} />
        ) : item.kind === 'video' ? (
          <video className={styles.video} src={item.url} controls autoPlay />
        ) : (
          <div className={styles.fileCard}>
            <b>{item.name}</b>
            <span>{item.url ? '点击下载后可打开' : '文件地址无效'}</span>
          </div>
        )}
        <div className={styles.bar}>
          <span className={styles.name}>{item.name}</span>
          {saved ? (
            <button type="button" className={styles.primary} onClick={() => void onOpen()}>
              打开
            </button>
          ) : (
            <button type="button" className={styles.primary} disabled={Boolean(busy)} onClick={() => void onDownload()}>
              {busy || '下载'}
            </button>
          )}
        </div>
      </div>
    </div>
  )
}
