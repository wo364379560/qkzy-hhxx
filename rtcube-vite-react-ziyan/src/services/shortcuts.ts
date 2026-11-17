const KEY = 'cn_ziyan_shortcuts'

export type ShortcutItem = {
  id: string
  text: string
}

const DEFAULTS: ShortcutItem[] = [
  { id: 's1', text: '好的，收到' },
  { id: 's2', text: '稍等一下' },
  { id: 's3', text: '在的，请问有什么事？' },
]

export function loadShortcuts(): ShortcutItem[] {
  try {
    const raw = localStorage.getItem(KEY)
    if (!raw) return DEFAULTS
    const list = JSON.parse(raw) as ShortcutItem[]
    return Array.isArray(list) && list.length ? list : DEFAULTS
  } catch {
    return DEFAULTS
  }
}

export function saveShortcuts(list: ShortcutItem[]) {
  localStorage.setItem(KEY, JSON.stringify(list))
}

export function filterShortcuts(list: ShortcutItem[], keyword: string) {
  const q = keyword.trim().toLowerCase()
  if (!q) return list
  return list.filter((s) => s.text.toLowerCase().includes(q))
}

export const INSERT_SHORTCUT_EVENT = 'cn-insert-shortcut'

let pendingShortcut = ''

export function emitInsertShortcut(text: string) {
  pendingShortcut = text
  window.dispatchEvent(new CustomEvent(INSERT_SHORTCUT_EVENT, { detail: { text } }))
}

export function takePendingShortcut() {
  const t = pendingShortcut
  pendingShortcut = ''
  return t
}
