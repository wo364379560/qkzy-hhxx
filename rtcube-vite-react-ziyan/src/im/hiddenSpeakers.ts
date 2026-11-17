const KEY = 'cn-hidden-speakers'

type Store = Record<string, string[]>

function read(): Store {
  try {
    const raw = localStorage.getItem(KEY)
    if (!raw) return {}
    const parsed = JSON.parse(raw) as Store
    return parsed && typeof parsed === 'object' ? parsed : {}
  } catch {
    return {}
  }
}

function write(store: Store) {
  localStorage.setItem(KEY, JSON.stringify(store))
}

export function hiddenSpeakerIds(conversationID: string): Set<string> {
  if (!conversationID) return new Set()
  return new Set(read()[conversationID] || [])
}

export function hideSpeaker(conversationID: string, userID: string) {
  const id = String(userID || '').trim()
  if (!conversationID || !id) return
  const store = read()
  const next = new Set(store[conversationID] || [])
  next.add(id)
  store[conversationID] = [...next]
  write(store)
}

export function isSpeakerHidden(conversationID: string, userID: string) {
  return hiddenSpeakerIds(conversationID).has(userID)
}
