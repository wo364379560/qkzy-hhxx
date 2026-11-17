import { useEffect, useRef, useState, type MouseEvent as ReactMouseEvent } from 'react'
import { startRemoteVideos } from '@/im/trtcClient'
import styles from './CallOverlay.module.css'

type Props = {
  open: boolean
  mediaType: 1 | 2
  peerName: string
  status: 'calling' | 'incoming' | 'connected' | 'ended'
  durationSec?: number
  muted?: boolean
  speakerOn?: boolean
  onAccept?: () => void
  onReject?: () => void
  onCancel?: () => void
  onHangup?: () => void
  onToggleMute?: () => void
  onToggleSpeaker?: () => void
}

function formatDur(sec: number) {
  const m = String(Math.floor(sec / 60)).padStart(2, '0')
  const s = String(sec % 60).padStart(2, '0')
  return `${m}:${s}`
}

type WinSize = 'min' | 'normal' | 'max'

/** 浮窗通话：默认同图三大小，可拖、可放大缩小，不挡聊天 */
export function CallOverlay({
  open,
  mediaType,
  peerName,
  status,
  durationSec = 0,
  muted = false,
  speakerOn = true,
  onAccept,
  onReject,
  onCancel,
  onHangup,
  onToggleMute,
  onToggleSpeaker,
}: Props) {
  const [tick, setTick] = useState(durationSec)
  const [size, setSize] = useState<WinSize>('normal')
  const [pos, setPos] = useState({ x: 0, y: 72 })
  const [pip, setPip] = useState<{ x: number; y: number } | null>(null)
  const dragRef = useRef<{ dx: number; dy: number } | null>(null)
  const pipDrag = useRef<{ ox: number; oy: number; sx: number; sy: number } | null>(null)
  const boxRef = useRef<HTMLDivElement>(null)
  const stageRef = useRef<HTMLDivElement>(null)
  const localRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    setTick(durationSec)
  }, [durationSec])
  useEffect(() => {
    if (status !== 'connected') return
    const id = window.setInterval(() => setTick((n) => n + 1), 1000)
    return () => window.clearInterval(id)
  }, [status])
  useEffect(() => {
    if (!open || mediaType !== 2 || status !== 'connected') return
    void startRemoteVideos()
    const id = window.setInterval(() => void startRemoteVideos(), 2000)
    return () => window.clearInterval(id)
  }, [open, mediaType, status])

  useEffect(() => {
    if (!open) {
      setSize('normal')
      return
    }
    const w = mediaType === 2 ? 480 : 340
    setPos({
      x: Math.max(24, Math.round((window.innerWidth - w) / 2)),
      y: 72,
    })
    setPip(null)
  }, [open, mediaType])

  if (!open) return null
  const media = mediaType === 2 ? '视频' : '语音'
  const video = mediaType === 2
  const statusText =
    status === 'calling'
      ? '正在呼叫…'
      : status === 'incoming'
        ? '邀请你通话'
        : status === 'connected'
          ? `通话中 ${formatDur(tick)}`
          : '已结束'

  function onDragStart(e: ReactMouseEvent) {
    if ((e.target as HTMLElement).closest('button')) return
    e.preventDefault()
    const box = boxRef.current
    if (!box) return
    const rect = box.getBoundingClientRect()
    dragRef.current = { dx: e.clientX - rect.left, dy: e.clientY - rect.top }
    const onMove = (ev: MouseEvent) => {
      if (!dragRef.current) return
      const maxX = window.innerWidth - 80
      const maxY = window.innerHeight - 48
      setPos({
        x: Math.min(maxX, Math.max(8, ev.clientX - dragRef.current.dx)),
        y: Math.min(maxY, Math.max(8, ev.clientY - dragRef.current.dy)),
      })
    }
    const onUp = () => {
      dragRef.current = null
      window.removeEventListener('mousemove', onMove)
      window.removeEventListener('mouseup', onUp)
    }
    window.addEventListener('mousemove', onMove)
    window.addEventListener('mouseup', onUp)
  }

  function onPipStart(e: ReactMouseEvent) {
    e.preventDefault()
    e.stopPropagation()
    const stage = stageRef.current
    const local = localRef.current
    if (!stage || !local) return
    const sr = stage.getBoundingClientRect()
    const lr = local.getBoundingClientRect()
    const cur = pip ?? { x: lr.left - sr.left, y: lr.top - sr.top }
    pipDrag.current = { ox: e.clientX, oy: e.clientY, sx: cur.x, sy: cur.y }
    const onMove = (ev: MouseEvent) => {
      if (!pipDrag.current) return
      const maxX = Math.max(0, stage.clientWidth - local.offsetWidth)
      const maxY = Math.max(0, stage.clientHeight - local.offsetHeight)
      setPip({
        x: Math.min(maxX, Math.max(0, pipDrag.current.sx + ev.clientX - pipDrag.current.ox)),
        y: Math.min(maxY, Math.max(0, pipDrag.current.sy + ev.clientY - pipDrag.current.oy)),
      })
    }
    const onUp = () => {
      pipDrag.current = null
      window.removeEventListener('mousemove', onMove)
      window.removeEventListener('mouseup', onUp)
    }
    window.addEventListener('mousemove', onMove)
    window.addEventListener('mouseup', onUp)
  }

  return (
    <>
      {size === 'min' ? (
        <div
          className={styles.mini}
          style={{ left: pos.x, top: pos.y }}
          onMouseDown={onDragStart}
        >
          <button type="button" className={styles.miniMain} onClick={() => setSize('normal')}>
            <strong>
              {media}通话 · {peerName}
            </strong>
            <span>{statusText}</span>
          </button>
          {status === 'connected' ? (
            <button type="button" className={styles.miniHang} onClick={onHangup}>
              挂断
            </button>
          ) : null}
        </div>
      ) : null}
      <div
        ref={boxRef}
        className={`${styles.card} ${video ? styles.videoCard : ''} ${size === 'max' ? styles.maxCard : ''}`}
        style={{
          left: pos.x,
          top: pos.y,
          visibility: size === 'min' ? 'hidden' : 'visible',
          pointerEvents: size === 'min' ? 'none' : 'auto',
        }}
      >
      <div className={styles.bar} onMouseDown={onDragStart}>
        <em>拖动</em>
        <div className={styles.winBtns}>
          <button type="button" title="缩小" onClick={() => setSize('min')}>
            –
          </button>
          <button
            type="button"
            title={size === 'max' ? '还原' : '放大'}
            onClick={() => setSize((s) => (s === 'max' ? 'normal' : 'max'))}
          >
            {size === 'max' ? '▢' : '□'}
          </button>
        </div>
      </div>
      <div ref={stageRef} className={styles.videoStage} style={video ? undefined : { display: 'none' }}>
        <div id="remote-video-view" className={styles.remoteView} />
        <div
          id="local-video-view"
          ref={localRef}
          className={styles.localView}
          title="拖动预览"
          style={pip ? { left: pip.x, top: pip.y, right: 'auto', bottom: 'auto' } : undefined}
          onMouseDown={onPipStart}
        />
      </div>
      <div className={styles.title}>
        {media}通话 · {peerName}
      </div>
      <p className={styles.status}>{statusText}</p>
      <div className={styles.actions}>
        {status === 'incoming' ? (
          <>
            <button type="button" className={styles.danger} onClick={onReject}>
              拒绝
            </button>
            <button type="button" className={styles.ok} onClick={onAccept}>
              接听
            </button>
          </>
        ) : null}
        {status === 'calling' ? (
          <button type="button" className={styles.danger} onClick={onCancel}>
            取消
          </button>
        ) : null}
        {status === 'connected' ? (
          <>
            <button type="button" className={muted ? styles.on : styles.secondary} onClick={onToggleMute}>
              {muted ? '已静音' : '静音'}
            </button>
            <button
              type="button"
              className={speakerOn ? styles.secondary : styles.on}
              onClick={onToggleSpeaker}
            >
              {speakerOn ? '扬声器' : '听筒'}
            </button>
            <button type="button" className={styles.danger} onClick={onHangup}>
              挂断
            </button>
          </>
        ) : null}
      </div>
    </div>
    </>
  )
}
