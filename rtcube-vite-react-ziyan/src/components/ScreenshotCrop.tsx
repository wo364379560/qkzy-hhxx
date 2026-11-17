import { useEffect, useMemo, useRef, useState, type PointerEvent } from 'react'
import { cropCanvasToFile } from '@/im/screenshot'
import styles from './ScreenshotCrop.module.css'

type Rect = { x: number; y: number; w: number; h: number }
type Drag = { x: number; y: number; sx: number; sy: number; armed: boolean }

type Props = {
  canvas: HTMLCanvasElement
  onCancel: () => void
  onConfirm: (file: File) => void
}

const MIN_DRAG_PX = 10
const MIN_BOX_PX = 24

function imageBox(el: HTMLElement, naturalW: number, naturalH: number) {
  const r = el.getBoundingClientRect()
  const scale = Math.min(r.width / Math.max(1, naturalW), r.height / Math.max(1, naturalH))
  const drawW = naturalW * scale
  const drawH = naturalH * scale
  return {
    left: r.left + (r.width - drawW) / 2,
    top: r.top + (r.height - drawH) / 2,
    scale: scale || 1,
  }
}

function clampRect(a: Rect, maxW: number, maxH: number): Rect {
  const x = Math.max(0, Math.min(maxW, a.x))
  const y = Math.max(0, Math.min(maxH, a.y))
  return {
    x,
    y,
    w: Math.max(1, Math.min(maxW - x, a.w)),
    h: Math.max(1, Math.min(maxH - y, a.h)),
  }
}

export function ScreenshotCrop({ canvas, onCancel, onConfirm }: Props) {
  const stageRef = useRef<HTMLDivElement>(null)
  const dragRef = useRef<Drag | null>(null)
  const rectRef = useRef<Rect | null>(null)
  const [rect, setRect] = useState<Rect | null>(null)
  const [dragging, setDragging] = useState(false)
  const [busy, setBusy] = useState(false)
  const [, bump] = useState(0)
  const src = useMemo(() => canvas.toDataURL('image/png'), [canvas])
  const nw = canvas.width
  const nh = canvas.height

  useEffect(() => {
    bump(1)
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onCancel()
    }
    const onResize = () => bump((n) => n + 1)
    window.addEventListener('keydown', onKey)
    window.addEventListener('resize', onResize)
    return () => {
      window.removeEventListener('keydown', onKey)
      window.removeEventListener('resize', onResize)
    }
  }, [onCancel])

  function toImagePoint(clientX: number, clientY: number) {
    const el = stageRef.current
    if (!el) return { x: 0, y: 0 }
    const box = imageBox(el, nw, nh)
    return {
      x: (clientX - box.left) / box.scale,
      y: (clientY - box.top) / box.scale,
    }
  }

  function applyDrag(clientX: number, clientY: number) {
    const start = dragRef.current
    if (!start) return
    const dist = Math.hypot(clientX - start.sx, clientY - start.sy)
    if (!start.armed) {
      if (dist < MIN_DRAG_PX) return
      start.armed = true
      setDragging(true)
    }
    const p = toImagePoint(clientX, clientY)
    const x = Math.min(start.x, p.x)
    const y = Math.min(start.y, p.y)
    const next = clampRect({ x, y, w: Math.abs(p.x - start.x), h: Math.abs(p.y - start.y) }, nw, nh)
    rectRef.current = next
    setRect(next)
  }

  function onPointerDown(e: PointerEvent<HTMLDivElement>) {
    if (e.button !== 0) return
    e.preventDefault()
    e.currentTarget.setPointerCapture(e.pointerId)
    const p = toImagePoint(e.clientX, e.clientY)
    dragRef.current = { x: p.x, y: p.y, sx: e.clientX, sy: e.clientY, armed: false }
    setDragging(true)
  }

  function onPointerMove(e: PointerEvent<HTMLDivElement>) {
    if (!dragRef.current) return
    applyDrag(e.clientX, e.clientY)
  }

  function onPointerUp(e: PointerEvent<HTMLDivElement>) {
    const start = dragRef.current
    dragRef.current = null
    setDragging(false)
    try {
      e.currentTarget.releasePointerCapture(e.pointerId)
    } catch {
      /* ignore */
    }
    if (!start?.armed) {
      rectRef.current = null
      setRect(null)
      return
    }
    const el = stageRef.current
    const box = el ? imageBox(el, nw, nh) : null
    const cur = rectRef.current
    if (!box || !cur) return
    const screenW = cur.w * box.scale
    const screenH = cur.h * box.scale
    if (screenW < MIN_BOX_PX || screenH < MIN_BOX_PX) {
      rectRef.current = null
      setRect(null)
    }
  }

  async function confirm() {
    if (busy) return
    setBusy(true)
    try {
      const el = stageRef.current
      const box = el ? imageBox(el, nw, nh) : null
      const cur = rectRef.current
      const usable =
        cur &&
        box &&
        cur.w * box.scale >= MIN_BOX_PX &&
        cur.h * box.scale >= MIN_BOX_PX
      const area = usable ? cur : { x: 0, y: 0, w: nw, h: nh }
      const file = await cropCanvasToFile(canvas, area)
      onConfirm(file)
    } catch (err) {
      alert(err instanceof Error ? err.message : '截图失败')
      setBusy(false)
    }
  }

  const el = stageRef.current
  const box = el ? imageBox(el, nw, nh) : null
  const view = rect && box
    ? {
        left: box.left + rect.x * box.scale,
        top: box.top + rect.y * box.scale,
        width: Math.max(1, rect.w * box.scale),
        height: Math.max(1, rect.h * box.scale),
      }
    : null
  const showBar = !dragging && !busy

  return (
    <div className={styles.mask}>
      <div
        ref={stageRef}
        className={styles.stage}
        onPointerDown={onPointerDown}
        onPointerMove={onPointerMove}
        onPointerUp={onPointerUp}
        onPointerCancel={onPointerUp}
      >
        <img className={styles.shot} src={src} alt="" draggable={false} />
        {view ? (
          <>
            <i className={styles.shade} style={{ left: 0, top: 0, right: 0, height: view.top }} />
            <i className={styles.shade} style={{ left: 0, top: view.top, width: view.left, height: view.height }} />
            <i
              className={styles.shade}
              style={{
                left: view.left + view.width,
                top: view.top,
                right: 0,
                height: view.height,
              }}
            />
            <i className={styles.shade} style={{ left: 0, top: view.top + view.height, right: 0, bottom: 0 }} />
            <i
              className={styles.frame}
              style={{ left: view.left, top: view.top, width: view.width, height: view.height }}
            />
          </>
        ) : (
          <i className={styles.shade} style={{ inset: 0 }} />
        )}
      </div>
      <div className={styles.hint}>按住拖动框选，松开后再点完成。Esc 取消，不框选则截整屏</div>
      {showBar ? (
        <div
          className={styles.bar}
          style={{
            left: view ? Math.min(window.innerWidth - 188, Math.max(12, view.left)) : 12,
            top: view
              ? Math.min(window.innerHeight - 52, view.top + view.height + 8)
              : window.innerHeight - 56,
          }}
        >
          <button type="button" className={styles.cancel} onClick={onCancel}>
            取消
          </button>
          <button type="button" className={styles.ok} onClick={() => void confirm()}>
            完成
          </button>
        </div>
      ) : null}
    </div>
  )
}
