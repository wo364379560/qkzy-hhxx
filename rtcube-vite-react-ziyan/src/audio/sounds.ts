/**
 * 消息 / 音视频铃音（assets/voice）
 * - newmsg.mp3      普通消息 / 发送成功
 * - newmsgsys.mp3   系统 / 群提示
 * - avchat_ring.mp3 来电振铃
 * - avchat_play.mp3 拨出等待音
 */

import newMsgUrl from '@/assets/voice/newmsg.mp3'
import newMsgSysUrl from '@/assets/voice/newmsgsys.mp3'
import avRingUrl from '@/assets/voice/avchat_ring.mp3'
import avPlayUrl from '@/assets/voice/avchat_play.mp3'

type SoundKey = 'newmsg' | 'newmsgsys' | 'avRing' | 'avPlay'

const URLS: Record<SoundKey, string> = {
  newmsg: newMsgUrl,
  newmsgsys: newMsgSysUrl,
  avRing: avRingUrl,
  avPlay: avPlayUrl,
}

const cache = new Map<SoundKey, HTMLAudioElement>()
let loopAudio: HTMLAudioElement | null = null
let loopKey: SoundKey | null = null
let unlocked = false

function getAudio(key: SoundKey) {
  let a = cache.get(key)
  if (!a) {
    a = new Audio(URLS[key])
    a.preload = 'auto'
    cache.set(key, a)
  }
  return a
}

/** 首次用户手势后解锁自动播放（浏览器策略） */
export function unlockAudio() {
  if (unlocked) return
  unlocked = true
  // 只解锁消息提示音，不要播振铃/回铃，否则一点聊天就会响通话铃
  ;(['newmsg', 'newmsgsys'] as SoundKey[]).forEach((key) => {
    const a = getAudio(key)
    const prevMuted = a.muted
    a.muted = true
    void a
      .play()
      .then(() => {
        a.pause()
        a.currentTime = 0
        a.muted = prevMuted
      })
      .catch(() => {
        a.muted = prevMuted
        unlocked = false
      })
  })
}

export function installAudioUnlock() {
  const once = () => {
    unlockAudio()
    window.removeEventListener('pointerdown', once)
    window.removeEventListener('keydown', once)
  }
  window.addEventListener('pointerdown', once)
  window.addEventListener('keydown', once)
  return () => {
    window.removeEventListener('pointerdown', once)
    window.removeEventListener('keydown', once)
  }
}

export function playSound(key: SoundKey) {
  try {
    const a = getAudio(key)
    a.loop = false
    a.muted = false
    a.currentTime = 0
    void a.play().catch(() => {
      /* 浏览器未手势授权时忽略 */
    })
  } catch {
    /* ignore */
  }
}

function stopAvFile(key: Extract<SoundKey, 'avRing' | 'avPlay'>) {
  const a = cache.get(key)
  if (!a) return
  try {
    a.pause()
    a.currentTime = 0
    a.loop = false
  } catch {
    /* ignore */
  }
}

export function startLoopSound(key: Extract<SoundKey, 'avRing' | 'avPlay'>) {
  stopLoopSound()
  try {
    const a = getAudio(key)
    a.loop = true
    a.muted = false
    a.currentTime = 0
    loopAudio = a
    loopKey = key
    void a.play().catch(() => {
      /* ignore */
    })
  } catch {
    /* ignore */
  }
}

export function stopLoopSound() {
  stopAvFile('avRing')
  stopAvFile('avPlay')
  loopAudio = null
  loopKey = null
}

export function currentLoopSound() {
  return loopKey
}

export function playIncomingMessageSound(opts?: { system?: boolean }) {
  playSound(opts?.system ? 'newmsgsys' : 'newmsg')
}
