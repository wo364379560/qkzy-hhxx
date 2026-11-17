export type SavedMedia = {
  blobUrl: string
  filePath?: string
  name: string
}

async function fetchBlob(url: string) {
  const res = await fetch(url)
  if (!res.ok) throw new Error('下载失败')
  return res.blob()
}

function browserDownload(blob: Blob, name: string) {
  const href = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = href
  a.download = name || 'download'
  a.rel = 'noopener'
  document.body.appendChild(a)
  a.click()
  a.remove()
  return href
}

export async function saveMedia(url: string, name: string): Promise<SavedMedia> {
  const blob = await fetchBlob(url)
  const fileName = name || 'download'
  if (window.desktop?.saveFileAuto || window.desktop?.saveFile) {
    const buffer = await blob.arrayBuffer()
    const filePath = window.desktop.saveFileAuto
      ? await window.desktop.saveFileAuto({ name: fileName, buffer })
      : await window.desktop.saveFile({ name: fileName, buffer })
    const blobUrl = URL.createObjectURL(blob)
    if (!filePath) return { blobUrl, name: fileName }
    return { blobUrl, filePath, name: fileName }
  }
  const blobUrl = browserDownload(blob, fileName)
  return { blobUrl, name: fileName }
}

export async function openSavedMedia(saved: SavedMedia) {
  if (saved.filePath && window.desktop?.openPath) {
    await window.desktop.openPath(saved.filePath)
    return
  }
  const a = document.createElement('a')
  a.href = saved.blobUrl
  a.download = saved.name
  a.rel = 'noopener'
  document.body.appendChild(a)
  a.click()
  a.remove()
}
