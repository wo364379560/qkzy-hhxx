/**
 * TRTC Web（trtc-sdk-v5）— 对齐 LingDoSpace：strRoomId = `mllx_${roomId}`
 * 注意：数字 roomId 与字符串 strRoomId 不是同一房间。
 */
// eslint-disable-next-line @typescript-eslint/no-explicit-any
type AnyTrtc = any

let trtc: AnyTrtc | null = null
let mediaBound = false
let wantVideo = false
const remoteUsers = new Set<string>()
const offMedia: Array<() => void> = []
let retryTimer = 0
let boostCtx: AudioContext | null = null
let boostIn: AudioNode | null = null
let boostGain: GainNode | null = null
const boostedEls = new WeakSet<HTMLMediaElement>()
let speakerBoostOn = true

async function loadTrtc(): Promise<AnyTrtc> {
  const mod = await import('trtc-sdk-v5')
  return (mod as { default?: AnyTrtc }).default || mod
}

export async function ensureTrtc() {
  if (!trtc) {
    const TRTC = await loadTrtc()
    trtc = TRTC.create()
  }
  return trtc
}

/** 对齐 LingDoSpace：strRoomId = mllx_${numericRoomId} */
export function toLingDoStrRoomId(roomId: string | number): string {
  const raw = String(roomId || '').trim()
  if (!raw) return 'mllx_1'
  if (raw.startsWith('mllx_')) return raw
  const n = raw.replace(/\D/g, '') || '1'
  return `mllx_${n}`
}

export async function ensureMediaPermission(video: boolean) {
  if (!navigator.mediaDevices?.getUserMedia) {
    throw new Error('当前环境不支持麦克风/摄像头')
  }
  try {
    const stream = await navigator.mediaDevices.getUserMedia({
      audio: {
        echoCancellation: true,
        noiseSuppression: true,
        autoGainControl: true,
      },
      video: video ? { width: 640, height: 480 } : false,
    })
    stream.getTracks().forEach((t) => t.stop())
  } catch (e) {
    const name = e instanceof DOMException ? e.name : ''
    if (name === 'NotAllowedError' || name === 'PermissionDeniedError') {
      throw new Error(video ? '请允许麦克风和摄像头后重试' : '请允许麦克风后重试')
    }
    if (name === 'NotFoundError') {
      throw new Error(video ? '未检测到摄像头或麦克风' : '未检测到麦克风')
    }
    throw new Error(e instanceof Error ? e.message : '无法打开音视频设备')
  }
}

function waitView(id: string, tries = 40): Promise<HTMLElement | null> {
  return new Promise((resolve) => {
    const tick = (left: number) => {
      const el = document.getElementById(id)
      if (el) {
        resolve(el)
        return
      }
      if (left <= 0) {
        resolve(null)
        return
      }
      window.setTimeout(() => tick(left - 1), 50)
    }
    tick(tries)
  })
}

function ensureBoostGraph() {
  if (boostCtx && boostGain && boostIn) return
  const AC = window.AudioContext || (window as unknown as { webkitAudioContext?: typeof AudioContext }).webkitAudioContext
  if (!AC) return
  boostCtx = new AC()
  const high = boostCtx.createBiquadFilter()
  high.type = 'highshelf'
  high.frequency.value = 1600
  high.gain.value = 9
  const mid = boostCtx.createBiquadFilter()
  mid.type = 'peaking'
  mid.frequency.value = 2800
  mid.Q.value = 0.9
  mid.gain.value = 4
  boostGain = boostCtx.createGain()
  boostGain.gain.value = speakerBoostOn ? 2.4 : 0.0001
  high.connect(mid)
  mid.connect(boostGain)
  boostGain.connect(boostCtx.destination)
  boostIn = high
}

function hookRemoteMediaEl(el: HTMLMediaElement) {
  if (boostedEls.has(el)) return
  if (el.tagName === 'VIDEO') return
  if (el.closest('#local-video-view')) return
  ensureBoostGraph()
  if (!boostCtx || !boostIn) return
  try {
    const src = boostCtx.createMediaElementSource(el)
    src.connect(boostIn)
    el.volume = 1
    el.muted = false
    boostedEls.add(el)
    void boostCtx.resume()
  } catch {
    /* 元素可能已被其它 AudioContext 占用 */
  }
}

function styleRemoteVideos() {
  const remote = document.getElementById('remote-video-view')
  if (!remote) return
  remote.querySelectorAll('video').forEach((v) => {
    v.setAttribute('playsinline', 'true')
    v.setAttribute('autoplay', 'true')
    v.muted = false
    v.playsInline = true
    v.style.width = '100%'
    v.style.height = '100%'
    v.style.objectFit = 'cover'
    v.style.background = '#0b1220'
    void v.play().catch(() => undefined)
  })
}

function boostRemoteViews() {
  styleRemoteVideos()
  const remote = document.getElementById('remote-video-view')
  const fromView = remote ? Array.from(remote.querySelectorAll('audio')) : []
  const fromBody = Array.from(document.querySelectorAll('audio')).filter((el) => Boolean(el.srcObject))
  const nodes = [...fromView, ...fromBody] as HTMLMediaElement[]
  nodes.forEach((el) => {
    if (el.closest('#local-video-view')) return
    el.volume = 1
    hookRemoteMediaEl(el)
  })
}

function setBoostOn(on: boolean) {
  speakerBoostOn = on
  if (boostGain && boostCtx) {
    boostGain.gain.setTargetAtTime(on ? 2.4 : 0.0001, boostCtx.currentTime, 0.02)
  }
}

function disposeBoost() {
  try {
    void boostCtx?.close()
  } catch {
    /* ignore */
  }
  boostCtx = null
  boostIn = null
  boostGain = null
}

function streamTypes(TRTC: AnyTrtc): unknown[] {
  const t = TRTC.TYPE || {}
  return [t.STREAM_TYPE_MAIN, t.STREAM_TYPE_SUB, 'main', 'sub', 0, 1].filter(
    (x, i, arr) => x !== undefined && arr.indexOf(x) === i,
  )
}

function userIdFrom(event: unknown): string {
  const e = event as { userId?: string; userID?: string; data?: { userId?: string; userID?: string } }
  return String(e?.userId || e?.userID || e?.data?.userId || e?.data?.userID || '')
}

function streamFrom(event: unknown, TRTC: AnyTrtc): unknown {
  const e = event as { streamType?: unknown; data?: { streamType?: unknown } }
  return e?.streamType ?? e?.data?.streamType ?? TRTC.TYPE?.STREAM_TYPE_MAIN ?? 'main'
}

async function playRemoteAudio(client: AnyTrtc, userId: string) {
  try {
    await client.startRemoteAudio?.({ userId })
  } catch {
    /* autoReceive 时可能已在播 */
  }
  try {
    await client.unmuteRemoteAudio?.(userId)
  } catch {
    /* ignore */
  }
  try {
    await client.setRemoteAudioVolume?.(userId, 100)
  } catch {
    /* ignore */
  }
  window.setTimeout(boostRemoteViews, 200)
  window.setTimeout(boostRemoteViews, 800)
}

function remoteVideoHasFrame() {
  const view = document.getElementById('remote-video-view')
  const videos = view ? Array.from(view.querySelectorAll('video')) : []
  return videos.some((v) => v.srcObject && v.videoWidth > 0 && v.readyState >= 2)
}

async function playRemoteVideo(client: AnyTrtc, userId: string, streamType?: unknown) {
  if (!userId) return
  remoteUsers.add(userId)
  const view = await waitView('remote-video-view')
  if (!view) return
  if (remoteVideoHasFrame()) {
    styleRemoteVideos()
    return
  }
  const TRTC = await loadTrtc()
  const preferred = streamType ?? TRTC.TYPE?.STREAM_TYPE_MAIN ?? 'main'
  const types = [preferred, ...streamTypes(TRTC)].filter((x, i, arr) => x !== undefined && arr.indexOf(x) === i)
  for (const st of types) {
    try {
      await client.startRemoteVideo?.({
        userId,
        streamType: st,
        view: 'remote-video-view',
        option: { fillMode: 'cover' },
      })
      styleRemoteVideos()
      return
    } catch {
      try {
        await client.startRemoteVideo?.({
          userId,
          streamType: st,
          view,
          option: { fillMode: 'cover' },
        })
        styleRemoteVideos()
        return
      } catch {
        /* 换一种 streamType */
      }
    }
  }
}

async function pullAllRemotes(client: AnyTrtc, video: boolean) {
  const TRTC = await loadTrtc()
  const lists = [
    client.getRemoteVideoList?.() || [],
    client.getRemoteUsers?.() || [],
    client.getRemoteUserList?.() || [],
  ]
  const seen = new Set<string>(remoteUsers)
  for (const list of lists) {
    for (const u of list as unknown[]) {
      const userId = typeof u === 'string' ? u : String((u as { userId?: string; userID?: string })?.userId || (u as { userID?: string })?.userID || '')
      if (userId) seen.add(userId)
    }
  }
  for (const userId of seen) {
    if (!userId) continue
    remoteUsers.add(userId)
    await playRemoteAudio(client, userId)
    if (video) await playRemoteVideo(client, userId, TRTC.TYPE?.STREAM_TYPE_MAIN)
  }
}

export async function bindTrtcMedia(video: boolean) {
  wantVideo = video
  const client = await ensureTrtc()
  const TRTC = await loadTrtc()
  const EVENT = TRTC.EVENT || {}

  const onAudio = (event: unknown) => {
    const userId = userIdFrom(event)
    if (userId) void playRemoteAudio(client, userId)
  }
  const onVideo = (event: unknown) => {
    const userId = userIdFrom(event)
    if (userId) void playRemoteVideo(client, userId, streamFrom(event, TRTC))
  }
  const onEnter = (event: unknown) => {
    const userId = userIdFrom(event)
    if (!userId) return
    remoteUsers.add(userId)
    void playRemoteAudio(client, userId)
    if (wantVideo) void playRemoteVideo(client, userId)
  }

  if (!mediaBound) {
    mediaBound = true
    const evts: Array<[string, (e: unknown) => void]> = [
      [EVENT.REMOTE_AUDIO_AVAILABLE, onAudio],
      [EVENT.REMOTE_VIDEO_AVAILABLE, onVideo],
      [EVENT.REMOTE_USER_ENTER, onEnter],
    ]
    evts.forEach(([name, fn]) => {
      if (!name) return
      client.on(name, fn)
      offMedia.push(() => client.off(name, fn))
    })
  }

  await pullAllRemotes(client, video)

  if (video) {
    const local = await waitView('local-video-view')
    if (local) {
      try {
        await client.startLocalVideo?.({ view: local, option: { fillMode: 'cover' } })
      } catch (e) {
        console.warn('[trtc] startLocalVideo retry', e)
      }
    }
  }

  window.clearInterval(retryTimer)
  let n = 0
  retryTimer = window.setInterval(() => {
    n += 1
    void pullAllRemotes(client, video)
    boostRemoteViews()
    if (n >= 30) window.clearInterval(retryTimer)
  }, 1000)
}

export async function enterTrtcRoom(opts: {
  sdkAppId: number
  userId: string
  userSig: string
  roomId: string | number
  audio?: boolean
  video?: boolean
}) {
  await ensureMediaPermission(Boolean(opts.video))
  const client = await ensureTrtc()
  const TRTC = await loadTrtc()
  try {
    await client.exitRoom?.()
  } catch {
    /* ignore */
  }

  const strRoomId = toLingDoStrRoomId(opts.roomId)
  wantVideo = Boolean(opts.video)
  await bindTrtcMedia(Boolean(opts.video))
  await client.enterRoom({
    sdkAppId: Number(opts.sdkAppId),
    userId: String(opts.userId),
    userSig: String(opts.userSig),
    strRoomId,
    scene: TRTC.TYPE?.SCENE_RTC || 'rtc',
    autoReceiveAudio: true,
    autoReceiveVideo: true,
  })

  if (opts.audio !== false) {
    try {
      await client.startLocalAudio?.({
        option: { profile: TRTC.TYPE?.AUDIO_PROFILE_HIGH || 'high' },
      })
    } catch {
      try {
        await client.startLocalAudio?.()
      } catch (e) {
        try {
          await client.exitRoom?.()
        } catch {
          /* ignore */
        }
        const msg = e instanceof Error ? e.message : String(e)
        if (/NotAllowed|Permission|denied/i.test(msg)) {
          throw new Error('麦克风权限被拒绝：请允许后重试')
        }
        throw e instanceof Error ? e : new Error(msg)
      }
    }
  }
  if (opts.video) {
    const local = await waitView('local-video-view', 16)
    if (local) {
      try {
        await client.startLocalVideo?.({ view: local, option: { fillMode: 'cover' } })
      } catch (e) {
        console.warn('[trtc] startLocalVideo failed', e)
      }
    }
  }
  await bindTrtcMedia(Boolean(opts.video))
  return client
}

export async function exitTrtcRoom() {
  window.clearInterval(retryTimer)
  retryTimer = 0
  remoteUsers.clear()
  wantVideo = false
  offMedia.splice(0).forEach((fn) => {
    try {
      fn()
    } catch {
      /* ignore */
    }
  })
  mediaBound = false
  disposeBoost()
  if (!trtc) return
  try {
    try {
      await trtc.stopLocalAudio?.()
    } catch {
      /* ignore */
    }
    try {
      await trtc.stopLocalVideo?.()
    } catch {
      /* ignore */
    }
    await trtc.exitRoom()
  } finally {
    try {
      trtc.destroy?.()
    } catch {
      /* ignore */
    }
    trtc = null
  }
}

export async function setTrtcMicMuted(muted: boolean) {
  const client = await ensureTrtc()
  try {
    await client.updateLocalAudio?.({ mute: muted })
    return
  } catch {
    /* fallback */
  }
  if (muted) await client.muteLocalAudio?.()
  else await client.unmuteLocalAudio?.()
}

export async function setTrtcSpeakerOn(on: boolean) {
  setBoostOn(on)
  const client = await ensureTrtc()
  const vol = on ? 100 : 0
  try {
    const users = client.getRemoteUsers?.() || []
    if (users.length) {
      for (const u of users) {
        const userId = typeof u === 'string' ? u : u?.userId
        if (userId) await client.setRemoteAudioVolume?.(userId, vol)
      }
      return
    }
  } catch {
    /* ignore */
  }
  try {
    await client.setRemoteAudioVolume?.('*', vol)
  } catch {
    /* ignore */
  }
}

export async function startRemoteVideos(viewId = 'remote-video-view') {
  const client = await ensureTrtc()
  await waitView(viewId)
  await pullAllRemotes(client, true)
}
