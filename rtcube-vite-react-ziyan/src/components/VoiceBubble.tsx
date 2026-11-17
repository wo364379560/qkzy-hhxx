import { useEffect, useRef, useState } from 'react'
import { getMediaCache, subscribeMediaCache } from '@/im/mediaCache'
import { soundInfoFromMessage } from '@/im/mediaPayload'
import type { ChatMessage } from '@/im/types'
import styles from './MessageList.module.css'

async function playableUrl(url: string) {
  if (!url || url.startsWith('blob:') || url.startsWith('data:')) return url
  try {
    const res = await fetch(url)
    if (!res.ok) return url
    return URL.createObjectURL(await res.blob())
  } catch {
    return url
  }
}

export function VoiceBubble({ msg }: { msg: ChatMessage }) {
  const info = soundInfoFromMessage(msg)
  const [, bump] = useState(0)
  useEffect(() => subscribeMediaCache(() => bump((n) => n + 1)), [])
  const cached = getMediaCache(msg.ID)
  const rawUrl = cached?.blobUrl || info.url
  const audioRef = useRef<HTMLAudioElement>(null)
  const [src, setSrc] = useState(rawUrl)
  const [playing, setPlaying] = useState(false)
  const [sec, setSec] = useState(info.second)

  useEffect(() => {
    setSec(info.second)
    let revoke = ''
    const next = cached?.blobUrl || info.url
    if (!next) {
      setSrc('')
      return
    }
    if (next.startsWith('blob:') || next.startsWith('data:')) {
      setSrc(next)
      return
    }
    void playableUrl(next).then((u) => {
      if (u !== next) revoke = u
      setSrc(u)
    })
    return () => {
      if (revoke) URL.revokeObjectURL(revoke)
    }
  }, [cached?.blobUrl, info.url, info.second, msg.ID])

  if (!src) {
    return <span className={styles.voiceBtn}>[语音{sec > 0 ? ` ${sec}"` : ''}]</span>
  }

  function toggle() {
    const el = audioRef.current
    if (!el) return
    if (playing) {
      el.pause()
      el.currentTime = 0
      setPlaying(false)
      return
    }
    void el
      .play()
      .then(() => setPlaying(true))
      .catch(() => alert('语音无法播放'))
  }

  const width = Math.min(168, 72 + Math.max(0, sec) * 4)

  return (
    <button type="button" className={styles.voiceBtn} style={{ width }} onClick={toggle}>
      <i className={playing ? styles.voicePause : styles.voicePlay} />
      <span>{sec > 0 ? `${sec}"` : '语音'}</span>
      <audio
        ref={audioRef}
        src={src}
        preload="metadata"
        hidden
        onLoadedMetadata={(e) => {
          const d = Math.round(e.currentTarget.duration || 0)
          if (Number.isFinite(d) && d > 0) setSec(d)
        }}
        onEnded={() => setPlaying(false)}
      />
    </button>
  )
}
