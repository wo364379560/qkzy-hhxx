import { EMOJI_TAG_REGEX } from './emojiCatalog'

const TAG_RE = /\[TUIEmoji_[A-Za-z0-9_]+\]$/

export function insertEmojiTag(
  text: string,
  tag: string,
  caret: number,
): { text: string; caret: number } {
  const before = text.slice(0, caret)
  const after = text.slice(caret)
  const next = `${before}${tag}${after}`
  return { text: next, caret: before.length + tag.length }
}

export function deleteBehindCaret(
  text: string,
  caret: number,
): { text: string; caret: number } | null {
  if (caret <= 0) return null
  const before = text.slice(0, caret)
  const m = before.match(TAG_RE)
  if (!m) return null
  const start = caret - m[0].length
  return {
    text: text.slice(0, start) + text.slice(caret),
    caret: start,
  }
}

export function countEmojiTags(text: string) {
  return (text.match(new RegExp(EMOJI_TAG_REGEX.source, 'g')) || []).length
}
