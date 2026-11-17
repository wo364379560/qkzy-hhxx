/** 把录音转成 WAV，腾讯 IM 才能读出时长，对端才不是 0 秒 */
export async function blobToWavFile(blob: Blob, fallbackSec: number) {
  const AC = window.AudioContext || (window as unknown as { webkitAudioContext?: typeof AudioContext }).webkitAudioContext
  if (!AC) {
    return {
      file: new File([blob], `voice_${Date.now()}.webm`, { type: blob.type || 'audio/webm' }),
      durationSec: Math.max(1, fallbackSec),
    }
  }
  const ctx = new AC()
  try {
    const raw = await blob.arrayBuffer()
    const buf = await ctx.decodeAudioData(raw.slice(0))
    const wav = encodeWav(buf)
    const durationSec = Math.max(1, Math.round(buf.duration || fallbackSec))
    return {
      file: new File([wav], `voice_${Date.now()}.wav`, { type: 'audio/wav' }),
      durationSec,
    }
  } catch {
    return {
      file: new File([blob], `voice_${Date.now()}.webm`, { type: blob.type || 'audio/webm' }),
      durationSec: Math.max(1, fallbackSec),
    }
  } finally {
    void ctx.close()
  }
}

function encodeWav(buffer: AudioBuffer): Blob {
  const channels = buffer.numberOfChannels
  const sampleRate = buffer.sampleRate
  const samples = buffer.length
  const bytesPerSample = 2
  const blockAlign = channels * bytesPerSample
  const dataSize = samples * blockAlign
  const out = new ArrayBuffer(44 + dataSize)
  const view = new DataView(out)
  writeStr(view, 0, 'RIFF')
  view.setUint32(4, 36 + dataSize, true)
  writeStr(view, 8, 'WAVE')
  writeStr(view, 12, 'fmt ')
  view.setUint32(16, 16, true)
  view.setUint16(20, 1, true)
  view.setUint16(22, channels, true)
  view.setUint32(24, sampleRate, true)
  view.setUint32(28, sampleRate * blockAlign, true)
  view.setUint16(32, blockAlign, true)
  view.setUint16(34, 16, true)
  writeStr(view, 36, 'data')
  view.setUint32(40, dataSize, true)
  let offset = 44
  for (let i = 0; i < samples; i++) {
    for (let ch = 0; ch < channels; ch++) {
      const s = Math.max(-1, Math.min(1, buffer.getChannelData(ch)[i] || 0))
      view.setInt16(offset, s < 0 ? s * 0x8000 : s * 0x7fff, true)
      offset += 2
    }
  }
  return new Blob([out], { type: 'audio/wav' })
}

function writeStr(view: DataView, offset: number, text: string) {
  for (let i = 0; i < text.length; i++) view.setUint8(offset + i, text.charCodeAt(i))
}
