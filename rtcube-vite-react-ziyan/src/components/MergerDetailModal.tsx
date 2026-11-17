import { useEffect, useState } from 'react'
import { renderEmojiNodes } from '@/im/emojiCatalog'
import { downloadMergerMessages } from '@/im/merger'
import { parseLargeAttachment, resolveMessageAttachmentUrl } from '@/im/largeAttachment'
import { messageCopyText } from '@/im/messageActions'
import {
  fileInfoFromMessage,
  formatFileSize,
  imageUrlFromMessage,
  isFileMessage,
  isImageMessage,
  isMergerMessage,
  isSoundMessage,
  isVideoMessage,
  mergerInfoFromMessage,
  openOrDownloadUrl,
  soundInfoFromMessage,
  videoInfoFromMessage,
} from '@/im/mediaPayload'
import { parseGiftPayload, parseStickerPayload, resolveGiftDisplayUrl } from '@/im/stickerCatalog'
import type { ChatMessage } from '@/im/types'
import styles from './MergerDetailModal.module.css'

type Props = {
  open: boolean
  message: ChatMessage | null
  onClose: () => void
}

type Preview = { kind: 'image' | 'video'; url: string; poster?: string }

function nickOf(msg: ChatMessage) {
  return msg.nick || msg.from || '未知'
}

function MergerItemBody({
  msg,
  onOpenMerger,
  onPreview,
}: {
  msg: ChatMessage
  onOpenMerger: (m: ChatMessage) => void
  onPreview: (p: Preview) => void
}) {
  if (isImageMessage(msg)) {
    const url = imageUrlFromMessage(msg)
    return url ? (
      <button
        type="button"
        className={styles.mediaBtn}
        onClick={() => onPreview({ kind: 'image', url })}
      >
        <img className={styles.media} src={url} alt="图片" />
      </button>
    ) : (
      <span>[图片]</span>
    )
  }
  if (isVideoMessage(msg)) {
    const v = videoInfoFromMessage(msg)
    const att = parseLargeAttachment(msg)
    if (v.url) {
      return (
        <video
          className={styles.mediaVideo}
          src={v.url}
          poster={v.snapshot || undefined}
          controls
          playsInline
        />
      )
    }
    if (v.snapshot || att) {
      return (
        <button
          type="button"
          className={styles.mediaBtn}
          onClick={() => {
            void (async () => {
              try {
                const url = att?.fileID ? await resolveMessageAttachmentUrl(msg, false) : v.url
                if (!url) {
                  alert('暂无播放地址')
                  return
                }
                onPreview({ kind: 'video', url, poster: v.snapshot })
              } catch (e) {
                alert(e instanceof Error ? e.message : '无法打开视频')
              }
            })()
          }}
        >
          {v.snapshot ? <img className={styles.media} src={v.snapshot} alt="视频封面" /> : <span>[视频] {att?.fileName || ''}</span>}
          <i className={styles.play} aria-hidden />
        </button>
      )
    }
    return <span>[视频]</span>
  }
  if (isFileMessage(msg)) {
    const f = fileInfoFromMessage(msg)
    const att = parseLargeAttachment(msg)
    return (
      <button
        type="button"
        className={styles.file}
        onClick={() => {
          void (async () => {
            try {
              const url = att?.fileID ? await resolveMessageAttachmentUrl(msg, true) : f.url
              if (!url) {
                alert('暂无下载地址')
                return
              }
              openOrDownloadUrl(url, f.name)
            } catch (e) {
              alert(e instanceof Error ? e.message : '无法打开文件')
            }
          })()
        }}
      >
        <b>{f.name}</b>
        <em>{formatFileSize(f.size) || '文件'}</em>
      </button>
    )
  }
  if (isSoundMessage(msg)) {
    const s = soundInfoFromMessage(msg)
    return s.url ? (
      <span className={styles.sound}>
        <audio controls src={s.url} preload="metadata" />
        {s.second > 0 ? <em>{s.second}"</em> : null}
      </span>
    ) : (
      <span>[语音]</span>
    )
  }
  if (isMergerMessage(msg)) {
    const info = mergerInfoFromMessage(msg)
    return (
      <button type="button" className={styles.nested} onClick={() => onOpenMerger(msg)}>
        <b>{info.title}</b>
        {info.abstracts.slice(0, 3).map((line, i) => (
          <span key={i}>{line}</span>
        ))}
        <em>聊天记录</em>
      </button>
    )
  }
  const gift = parseGiftPayload(msg.payload?.data)
  if (gift) {
    const url = resolveGiftDisplayUrl(gift)
    return (
      <span className={styles.gift}>
        {url ? <img src={url} alt={gift.giftName} /> : <em>[动图] {gift.giftName}</em>}
      </span>
    )
  }
  const sticker = parseStickerPayload(msg.payload?.data)
  if (sticker) return <span>[贴纸] {sticker.stickerName}</span>
  const text = messageCopyText(msg) || msg.payload?.text || '[消息]'
  return (
    <p className={styles.text}>
      {renderEmojiNodes(text).map((p, i) =>
        p.type === 'emoji' ? (
          p.assetUrl ? (
            <img key={i} className={styles.emoji} src={p.assetUrl} alt="" />
          ) : (
            <span key={i} className={styles.emoji}>
              {p.value}
            </span>
          )
        ) : (
          <span key={i}>{p.value}</span>
        ),
      )}
    </p>
  )
}

export function MergerDetailModal({ open, message, onClose }: Props) {
  const [stack, setStack] = useState<ChatMessage[]>([])
  const [list, setList] = useState<ChatMessage[]>([])
  const [err, setErr] = useState('')
  const [loading, setLoading] = useState(false)
  const [preview, setPreview] = useState<Preview | null>(null)

  const current = stack[stack.length - 1] || message

  useEffect(() => {
    if (!open || !message) {
      setStack([])
      setList([])
      setErr('')
      setPreview(null)
      return
    }
    setStack([message])
  }, [open, message])

  useEffect(() => {
    if (!open || !current) return
    let cancelled = false
    setLoading(true)
    setErr('')
    setList([])
    ;(async () => {
      try {
        const msgs = await downloadMergerMessages(current)
        if (!cancelled) setList(msgs)
      } catch (e) {
        if (!cancelled) setErr(e instanceof Error ? e.message : '加载失败')
      } finally {
        if (!cancelled) setLoading(false)
      }
    })()
    return () => {
      cancelled = true
    }
  }, [open, current])

  if (!open || !current) return null
  const title = mergerInfoFromMessage(current).title || '聊天记录'

  function goBack() {
    if (stack.length > 1) {
      setStack((s) => s.slice(0, -1))
      return
    }
    onClose()
  }

  return (
    <div className={styles.page}>
      <header className={styles.head}>
        <button type="button" className={styles.back} onClick={goBack} aria-label="返回">
          ←
        </button>
        <strong>{title}</strong>
        <span className={styles.headPad} />
      </header>
      <div className={styles.body}>
        {loading ? <p className={styles.status}>加载中…</p> : null}
        {err ? <p className={styles.err}>{err}</p> : null}
        {list.map((m, i) => (
          <article key={m.ID || `${m.from}-${m.time}-${i}`} className={styles.card}>
            <b>{nickOf(m)}</b>
            <MergerItemBody
              msg={m}
              onOpenMerger={(nested) => setStack((s) => [...s, nested])}
              onPreview={setPreview}
            />
          </article>
        ))}
        {!loading && !err && !list.length ? <p className={styles.status}>暂无内容</p> : null}
      </div>
      {preview ? (
        <div className={styles.lightbox} onClick={() => setPreview(null)}>
          {preview.kind === 'video' && preview.url ? (
            <video
              src={preview.url}
              poster={preview.poster}
              controls
              autoPlay
              playsInline
              onClick={(e) => e.stopPropagation()}
            />
          ) : (
            <img src={preview.url || preview.poster} alt="" onClick={(e) => e.stopPropagation()} />
          )}
        </div>
      ) : null}
    </div>
  )
}
