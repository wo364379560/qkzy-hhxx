import { useEffect, useRef, type ClipboardEvent, type KeyboardEvent } from 'react'
import { splitEmojiText } from '@/im/emojiCatalog'
import { filesFromClipboard } from '@/im/screenshot'
import styles from './EmojiComposer.module.css'

type Props = {
  value: string
  placeholder?: string
  disabled?: boolean
  onChange: (value: string) => void
  onEnter?: () => void
  onKeyDown?: (e: KeyboardEvent<HTMLDivElement>) => boolean | void
  onPasteFiles?: (files: File[]) => void
}

function escapeHtml(s: string) {
  return s
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}

function renderHtml(text: string) {
  const segs = splitEmojiText(text || '')
  if (!segs.length) return ''
  return segs
    .map((s) => {
      const tag = s.tag ? encodeURIComponent(s.tag) : ''
      if (s.type === 'emoji' && s.assetUrl) {
        return `<img class="${styles.face}" src="${s.assetUrl}" alt="" data-tag="${tag}" />`
      }
      if (s.type === 'emoji') {
        return `<span class="${styles.glyph}" data-tag="${tag}">${escapeHtml(s.value)}</span>`
      }
      return escapeHtml(s.value).replace(/\n/g, '<br>')
    })
    .join('')
}

function serializeNode(root: Node): string {
  let out = ''
  root.childNodes.forEach((node) => {
    if (node.nodeType === Node.TEXT_NODE) {
      out += node.textContent || ''
      return
    }
    if (node.nodeName === 'BR') {
      out += '\n'
      return
    }
    if (node instanceof HTMLElement) {
      const tag = node.getAttribute('data-tag')
      if (tag) {
        out += decodeURIComponent(tag)
        return
      }
      out += serializeNode(node)
    }
  })
  return out.replace(/\u00a0/g, ' ')
}

function placeCaretEnd(el: HTMLElement) {
  const range = document.createRange()
  range.selectNodeContents(el)
  range.collapse(false)
  const sel = window.getSelection()
  sel?.removeAllRanges()
  sel?.addRange(range)
}

export function EmojiComposer({
  value,
  placeholder = '请输入消息',
  disabled,
  onChange,
  onEnter,
  onKeyDown: onKeyDownProp,
  onPasteFiles,
}: Props) {
  const elRef = useRef<HTMLDivElement>(null)
  const composing = useRef(false)
  const lastEmitted = useRef<string | null>(null)

  useEffect(() => {
    const el = elRef.current
    if (!el || composing.current) return
    if (value === lastEmitted.current) return
    lastEmitted.current = value
    el.innerHTML = renderHtml(value)
    if (value) placeCaretEnd(el)
  }, [value])

  function emit(next: string) {
    lastEmitted.current = next
    onChange(next)
  }

  function syncFromDom() {
    const el = elRef.current
    if (!el) return
    emit(serializeNode(el))
  }

  function onKeyDown(e: KeyboardEvent<HTMLDivElement>) {
    if (disabled) {
      e.preventDefault()
      return
    }
    if (onKeyDownProp?.(e)) {
      e.preventDefault()
      return
    }
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      onEnter?.()
    }
  }

  function onPaste(e: ClipboardEvent<HTMLDivElement>) {
    if (disabled) {
      e.preventDefault()
      return
    }
    const images = filesFromClipboard(e.clipboardData)
    if (images.length) {
      e.preventDefault()
      onPasteFiles?.(images)
      return
    }
    e.preventDefault()
    const t = e.clipboardData.getData('text/plain') || ''
    if (!t) return
    lastEmitted.current = null
    onChange(value + t)
  }

  return (
    <div
      ref={elRef}
      className={styles.box}
      contentEditable={!disabled}
      role="textbox"
      data-placeholder={placeholder}
      suppressContentEditableWarning
      onKeyDown={onKeyDown}
      onInput={() => {
        if (!composing.current) syncFromDom()
      }}
      onPaste={onPaste}
      onCompositionStart={() => {
        composing.current = true
      }}
      onCompositionEnd={() => {
        composing.current = false
        syncFromDom()
      }}
    />
  )
}
