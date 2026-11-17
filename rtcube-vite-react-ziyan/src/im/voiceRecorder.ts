/**
 * MediaRecorder 录音，停止后转 WAV（腾讯 IM 才能解析时长）
 */
import { blobToWavFile } from './wavEncode'

export type VoiceRecording = {
  blob: Blob
  durationSec: number
  mime: string
}

let mediaRecorder: MediaRecorder | null = null
let chunks: BlobPart[] = []
let startedAt = 0
let stream: MediaStream | null = null

function pickMime() {
  const candidates = ['audio/webm;codecs=opus', 'audio/webm', 'audio/ogg;codecs=opus', 'audio/mp4']
  for (const m of candidates) {
    if (typeof MediaRecorder !== 'undefined' && MediaRecorder.isTypeSupported?.(m)) return m
  }
  return ''
}

export function canRecordVoice() {
  return (
    typeof navigator !== 'undefined' &&
    !!navigator.mediaDevices?.getUserMedia &&
    typeof MediaRecorder !== 'undefined'
  )
}

export async function startVoiceRecord() {
  if (!canRecordVoice()) throw new Error('当前环境不支持录音')
  if (mediaRecorder && mediaRecorder.state === 'recording') {
    throw new Error('正在录音中')
  }
  stream = await navigator.mediaDevices.getUserMedia({ audio: true })
  const mime = pickMime()
  chunks = []
  mediaRecorder = mime ? new MediaRecorder(stream, { mimeType: mime }) : new MediaRecorder(stream)
  mediaRecorder.ondataavailable = (e) => {
    if (e.data && e.data.size > 0) chunks.push(e.data)
  }
  startedAt = Date.now()
  mediaRecorder.start(200)
}

export function isVoiceRecording() {
  return mediaRecorder?.state === 'recording'
}

export async function stopVoiceRecord(): Promise<VoiceRecording> {
  const rec = mediaRecorder
  if (!rec || rec.state === 'inactive') {
    cleanupStream()
    throw new Error('未在录音')
  }
  const mime = rec.mimeType || 'audio/webm'
  const blob = await new Promise<Blob>((resolve, reject) => {
    rec.onerror = () => reject(new Error('录音失败'))
    rec.onstop = () => resolve(new Blob(chunks, { type: mime }))
    try {
      rec.stop()
    } catch (e) {
      reject(e instanceof Error ? e : new Error('停止录音失败'))
    }
  })
  const fallbackSec = Math.max(1, Math.round((Date.now() - startedAt) / 1000))
  cleanupStream()
  mediaRecorder = null
  chunks = []
  if (blob.size < 256) throw new Error('录音太短')
  const wav = await blobToWavFile(blob, fallbackSec)
  return { blob: wav.file, durationSec: wav.durationSec, mime: wav.file.type }
}

export function cancelVoiceRecord() {
  try {
    if (mediaRecorder && mediaRecorder.state !== 'inactive') mediaRecorder.stop()
  } catch {
    /* ignore */
  }
  mediaRecorder = null
  chunks = []
  cleanupStream()
}

function cleanupStream() {
  stream?.getTracks().forEach((t) => t.stop())
  stream = null
}
