export function filesFromClipboard(data: DataTransfer | null): File[] {
  if (!data) return []
  const out: File[] = []
  const items = Array.from(data.items || [])
  for (const item of items) {
    if (item.kind !== 'file' || !item.type.startsWith('image/')) continue
    const file = item.getAsFile()
    if (file) out.push(file)
  }
  if (out.length) return out
  return Array.from(data.files || []).filter((f) => f.type.startsWith('image/'))
}

function waitForVideoFrame(video: HTMLVideoElement) {
  return new Promise<void>((resolve, reject) => {
    const timer = window.setTimeout(() => reject(new Error('截图超时')), 8000)
    const done = () => {
      window.clearTimeout(timer)
      resolve()
    }
    if (typeof video.requestVideoFrameCallback === 'function') {
      video.requestVideoFrameCallback(() => done())
      return
    }
    if (video.readyState >= 2 && video.videoWidth) {
      done()
      return
    }
    video.addEventListener('loadeddata', () => done(), { once: true })
  })
}

function sleep(ms: number) {
  return new Promise((resolve) => window.setTimeout(resolve, ms))
}

function toBlobPart(buf: ArrayBuffer | Uint8Array): BlobPart {
  if (buf instanceof ArrayBuffer) return buf
  return Uint8Array.from(buf)
}

async function pngBufferToCanvas(buf: ArrayBuffer | Uint8Array): Promise<HTMLCanvasElement> {
  const blob = new Blob([toBlobPart(buf)], { type: 'image/png' })
  const url = URL.createObjectURL(blob)
  try {
    const img = new Image()
    await new Promise<void>((resolve, reject) => {
      img.onload = () => resolve()
      img.onerror = () => reject(new Error('截图失败'))
      img.src = url
    })
    const canvas = document.createElement('canvas')
    canvas.width = img.naturalWidth
    canvas.height = img.naturalHeight
    const ctx = canvas.getContext('2d')
    if (!ctx) throw new Error('截图失败')
    ctx.drawImage(img, 0, 0)
    return canvas
  } finally {
    URL.revokeObjectURL(url)
  }
}

async function captureViaElectron(): Promise<HTMLCanvasElement | null> {
  if (!window.desktop?.captureScreen) return null
  const buf = await window.desktop.captureScreen()
  if (!buf) return null
  return pngBufferToCanvas(buf)
}

async function captureViaDisplayMedia(): Promise<HTMLCanvasElement> {
  if (!navigator.mediaDevices?.getDisplayMedia) {
    throw new Error('当前浏览器不支持截图，请用 QQ / 系统截图后 Ctrl+V 粘贴')
  }
  const stream = await navigator.mediaDevices.getDisplayMedia({
    audio: false,
    video: {
      frameRate: { ideal: 1, max: 5 },
      width: { ideal: 3840, max: 7680 },
      height: { ideal: 2160, max: 4320 },
      displaySurface: 'monitor',
    },
    preferCurrentTab: false,
    selfBrowserSurface: 'exclude',
    monitorTypeSurfaces: 'include',
    surfaceSwitching: 'exclude',
    systemAudio: 'exclude',
  } as DisplayMediaStreamOptions)
  const video = document.createElement('video')
  video.muted = true
  video.playsInline = true
  video.srcObject = stream
  try {
    await video.play()
    await waitForVideoFrame(video)
    await sleep(180)
    await waitForVideoFrame(video)
    const canvas = document.createElement('canvas')
    canvas.width = video.videoWidth
    canvas.height = video.videoHeight
    if (!canvas.width || !canvas.height) throw new Error('未能获取画面')
    const surface = String(stream.getVideoTracks()[0]?.getSettings?.().displaySurface || '')
    if (
      (surface === 'window' || surface === 'browser' || canvas.width < 800 || canvas.height < 500) &&
      canvas.width * canvas.height < 800 * 500
    ) {
      throw new Error('截到的画面太小。请选择「整个屏幕」，不要选单个窗口')
    }
    const ctx = canvas.getContext('2d')
    if (!ctx) throw new Error('截图失败')
    ctx.drawImage(video, 0, 0)
    return canvas
  } finally {
    stream.getTracks().forEach((t) => t.stop())
    video.srcObject = null
  }
}

export async function captureDisplayCanvas(): Promise<HTMLCanvasElement> {
  const native = await captureViaElectron().catch(() => null)
  if (native && native.width > 0) return native
  return captureViaDisplayMedia()
}

export function cropCanvasToFile(
  source: HTMLCanvasElement,
  rect: { x: number; y: number; w: number; h: number },
  name = `screenshot-${Date.now()}.png`,
): Promise<File> {
  const x = Math.max(0, Math.round(rect.x))
  const y = Math.max(0, Math.round(rect.y))
  const w = Math.max(1, Math.round(rect.w))
  const h = Math.max(1, Math.round(rect.h))
  const canvas = document.createElement('canvas')
  canvas.width = Math.min(w, source.width - x)
  canvas.height = Math.min(h, source.height - y)
  const ctx = canvas.getContext('2d')
  if (!ctx) return Promise.reject(new Error('截图失败'))
  ctx.drawImage(source, x, y, canvas.width, canvas.height, 0, 0, canvas.width, canvas.height)
  return canvasToPngFile(canvas, name)
}

export function canvasToPngFile(canvas: HTMLCanvasElement, name: string): Promise<File> {
  return new Promise((resolve, reject) => {
    canvas.toBlob((blob) => {
      if (!blob) {
        reject(new Error('截图失败'))
        return
      }
      resolve(new File([blob], name, { type: 'image/png' }))
    }, 'image/png')
  })
}

export async function copyImageFile(file: File) {
  const ClipboardItemCtor = window.ClipboardItem
  if (!navigator.clipboard?.write || !ClipboardItemCtor) return false
  try {
    const blob =
      file.type === 'image/png'
        ? file
        : await file.arrayBuffer().then((b) => new Blob([b], { type: 'image/png' }))
    await navigator.clipboard.write([new ClipboardItemCtor({ 'image/png': blob })])
    return true
  } catch {
    return false
  }
}
