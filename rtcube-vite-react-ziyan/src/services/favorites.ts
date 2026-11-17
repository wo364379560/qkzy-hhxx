const KEY = 'cn_ziyan_favorites_v1'
const MAX = 200

export type FavoriteItem = {
  id: string
  text: string
  from: string
  conversationID?: string
  createdAt: number
  type?: string
  payload?: Record<string, unknown>
}

export function loadFavorites(): FavoriteItem[] {
  try {
    const raw = localStorage.getItem(KEY)
    if (!raw) return []
    const list = JSON.parse(raw) as FavoriteItem[]
    return Array.isArray(list) ? list : []
  } catch {
    return []
  }
}

export function saveFavorites(list: FavoriteItem[]) {
  localStorage.setItem(KEY, JSON.stringify(list.slice(0, MAX)))
}

export function addFavorite(item: FavoriteItem) {
  const list = loadFavorites().filter((x) => x.id !== item.id)
  list.unshift(item)
  saveFavorites(list)
  return list
}

export function removeFavorite(id: string) {
  const list = loadFavorites().filter((x) => x.id !== id)
  saveFavorites(list)
  return list
}

function snapshotPayload(payload?: Record<string, unknown>) {
  try {
    return JSON.parse(JSON.stringify(payload || {})) as Record<string, unknown>
  } catch {
    return undefined
  }
}

export function favoriteFromMessage(msg: {
  ID: string
  type?: string
  from?: string
  nick?: string
  conversationID?: string
  payload?: Record<string, unknown>
  text?: string
}): FavoriteItem {
  return {
    id: msg.ID,
    text: String(msg.text || ''),
    from: msg.nick || msg.from || '',
    conversationID: msg.conversationID,
    createdAt: Date.now(),
    type: msg.type,
    payload: snapshotPayload(msg.payload),
  }
}
