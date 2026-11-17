/**
 * 本地文件缓存：自己发的直接打开；对方的点下载后记路径，可定位到文件夹。
 */
const STORE_KEY = 'cn_ziyan_media_cache_v1'

export type MediaCacheItem = {
  name: string
  filePath?: string
  blobUrl?: string
}

type Store = Record<string, MediaCacheItem>

const listeners = new Set<() => void>()

function readStore(): Store {
  try {
    const raw = localStorage.getItem(STORE_KEY)
    if (!raw) return {}
    return JSON.parse(raw) as Store
  } catch {
    return {}
  }
}

function writeStore(next: Store) {
  localStorage.setItem(STORE_KEY, JSON.stringify(next))
  listeners.forEach((fn) => fn())
}

export function getMediaCache(id: string): MediaCacheItem | null {
  if (!id) return null
  return readStore()[id] || null
}

export function setMediaCache(id: string, item: MediaCacheItem) {
  if (!id) return
  const store = readStore()
  store[id] = item
  writeStore(store)
}

export function moveMediaCache(fromId: string, toId: string) {
  if (!fromId || !toId || fromId === toId) return
  const store = readStore()
  const hit = store[fromId]
  if (!hit) return
  store[toId] = hit
  delete store[fromId]
  writeStore(store)
}

export function subscribeMediaCache(fn: () => void) {
  listeners.add(fn)
  return () => {
    listeners.delete(fn)
  }
}

export async function rememberOutgoingFile(id: string, file: File) {
  const blobUrl = URL.createObjectURL(file)
  // 大文件不要整份读进内存，否则 100MB+ 会把渲染进程打爆
  if (file.size > 32 * 1024 * 1024) {
    setMediaCache(id, { name: file.name, blobUrl })
    return
  }
  const buffer = await file.arrayBuffer()
  let filePath: string | undefined
  try {
    filePath = (await window.desktop?.saveFileAuto?.({ name: file.name, buffer })) || undefined
  } catch {
    filePath = undefined
  }
  setMediaCache(id, { name: file.name, blobUrl, filePath })
}

export async function downloadMediaAuto(id: string, url: string, name: string) {
  const res = await fetch(url)
  if (!res.ok) throw new Error('下载失败')
  const blob = await res.blob()
  const buffer = await blob.arrayBuffer()
  const blobUrl = URL.createObjectURL(blob)
  let filePath: string | undefined
  if (window.desktop?.saveFileAuto) {
    filePath = (await window.desktop.saveFileAuto({ name, buffer })) || undefined
  } else {
    const a = document.createElement('a')
    a.href = blobUrl
    a.download = name
    document.body.appendChild(a)
    a.click()
    a.remove()
  }
  const item = { name, blobUrl, filePath }
  setMediaCache(id, item)
  return item
}

export async function openCachedMedia(item: MediaCacheItem, fallbackUrl?: string) {
  if (item.filePath && window.desktop?.openPath) {
    await window.desktop.openPath(item.filePath)
    return
  }
  const href = item.blobUrl || fallbackUrl
  if (!href) throw new Error('文件还没下载')
  window.open(href, '_blank', 'noopener,noreferrer')
}

export async function revealCachedMedia(item: MediaCacheItem) {
  if (item.filePath && window.desktop?.showInFolder) {
    await window.desktop.showInFolder(item.filePath)
    return
  }
  if (item.filePath && window.desktop?.openPath) {
    await window.desktop.openPath(item.filePath)
    return
  }
  throw new Error('当前环境不能定位到文件夹')
}
