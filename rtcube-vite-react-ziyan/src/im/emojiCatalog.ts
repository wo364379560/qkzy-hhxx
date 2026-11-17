/**
 * 与 web-h5 / Android 同源 [TUIEmoji_*] tag。
 * 输入和旧消息保留 tag 解析；发送时转换 Unicode，与原生端互通。
 */
export type EmojiItem = {
  tag: string
  glyph: string
  assetUrl?: string
}

import { TAGS_AND_GLYPHS } from './emojiWire'

const pngModules = {
  ...import.meta.glob('../assets/emoj/*.{png,webp,gif}', {
    eager: true,
    query: '?url',
    import: 'default',
  }),
  ...import.meta.glob('../assets/emoji/*.{png,webp,gif}', {
    eager: true,
    query: '?url',
    import: 'default',
  }),
} as Record<string, string>

function pngForIndex(index: number) {
  // This is the same zero-based pack used by the Android client.
  const nativeAsset = pngModules[`../assets/emoji/emoji_${index}.png`]
  if (nativeAsset) return nativeAsset
  const oneBased = index + 1
  const candidates = [
    `emoji_${String(oneBased).padStart(3, '0')}.`,
    `emoji_${String(oneBased).padStart(2, '0')}.`,
    `emoji_${oneBased}.`,
    `emoji_${index}.`,
  ]
  for (const key of candidates) {
    for (const [path, url] of Object.entries(pngModules)) {
      if (path.includes(key)) return url
    }
  }
  return undefined
}

export const EMOJI_ITEMS: EmojiItem[] = TAGS_AND_GLYPHS.map(([tag, glyph], index) => ({
  tag,
  glyph,
  assetUrl: pngForIndex(index),
}))

/** 兼容旧引用 */
export const EMOJI_LIST = EMOJI_ITEMS

const tagToGlyph = new Map(EMOJI_ITEMS.map((e) => [e.tag, e.glyph]))
const tagToAsset = new Map(
  EMOJI_ITEMS.filter((e) => e.assetUrl).map((e) => [e.tag, e.assetUrl!]),
)

export const EMOJI_TAG_REGEX = /\[TUIEmoji_\w{2,}]/g

export function emojiPreviewText(raw: string) {
  return raw.replace(EMOJI_TAG_REGEX, (tag) => tagToGlyph.get(tag) || '[表情]')
}

export type EmojiSeg = { type: 'text' | 'emoji'; value: string; assetUrl?: string; tag?: string }

export function splitEmojiText(raw: string): EmojiSeg[] {
  const out: EmojiSeg[] = []
  let last = 0
  for (const m of raw.matchAll(EMOJI_TAG_REGEX)) {
    const start = m.index ?? 0
    if (start > last) out.push({ type: 'text', value: raw.slice(last, start) })
    const tag = m[0]
    out.push({
      type: 'emoji',
      value: tagToGlyph.get(tag) || tag,
      assetUrl: tagToAsset.get(tag),
      tag,
    })
    last = start + tag.length
  }
  if (last < raw.length) out.push({ type: 'text', value: raw.slice(last) })
  return out.length ? out : [{ type: 'text', value: raw }]
}

export function renderEmojiNodes(text: string): EmojiSeg[] {
  return splitEmojiText(text)
}
