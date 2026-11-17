import { getChat } from './ImClient'
import { parseLargeAttachment } from './largeAttachment'
import {
  fileInfoFromMessage,
  formatFileSize,
  imageUrlFromMessage,
  isFileMessage,
  isImageMessage,
  isSoundMessage,
  isVideoMessage,
  soundInfoFromMessage,
  videoInfoFromMessage,
} from './mediaPayload'
import type { ChatMessage } from './types'

export type FileKind = 'all' | 'doc' | 'sheet' | 'slide' | 'media' | 'zip' | 'audio' | 'other'
export type FileMediaType = 'image' | 'video' | 'file' | 'audio'

export type HubFile = {
  id: string
  conversationID: string
  from: string
  nick: string
  time: number
  name: string
  size: number
  url: string
  thumb: string
  kind: Exclude<FileKind, 'all'>
  type: FileMediaType
  mine?: boolean
  fileID?: string
}

export const FILE_KINDS: { key: FileKind; label: string }[] = [
  { key: 'all', label: '全部' },
  { key: 'doc', label: '文档' },
  { key: 'sheet', label: '表格' },
  { key: 'slide', label: '演示' },
  { key: 'media', label: '图片与视频' },
  { key: 'zip', label: '压缩' },
  { key: 'audio', label: '音频' },
  { key: 'other', label: '其它' },
]

function extOf(name: string) {
  const i = name.lastIndexOf('.')
  return i >= 0 ? name.slice(i + 1).toLowerCase() : ''
}

export function classifyFile(name: string, type: FileMediaType): Exclude<FileKind, 'all'> {
  if (type === 'image' || type === 'video') return 'media'
  if (type === 'audio') return 'audio'
  const ext = extOf(name)
  if (['doc', 'docx', 'pdf', 'txt', 'md', 'rtf', 'pages'].includes(ext)) return 'doc'
  if (['xls', 'xlsx', 'csv', 'numbers'].includes(ext)) return 'sheet'
  if (['ppt', 'pptx', 'key'].includes(ext)) return 'slide'
  if (['zip', 'rar', '7z', 'tar', 'gz'].includes(ext)) return 'zip'
  if (['mp3', 'wav', 'aac', 'm4a', 'flac', 'ogg'].includes(ext)) return 'audio'
  if (['jpg', 'jpeg', 'png', 'gif', 'webp', 'bmp', 'mp4', 'mov', 'avi', 'mkv'].includes(ext)) {
    return 'media'
  }
  return 'other'
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function toHubFile(msg: any): HubFile | null {
  if (!msg || msg.isRevoked || msg.isDeleted) return null
  const id = String(msg.ID || msg.id || '')
  if (!id) return null
  let type: FileMediaType | '' = ''
  let name = ''
  let size = 0
  let url = ''
  let thumb = ''
  let fileID = ''
  const att = parseLargeAttachment(msg)
  if (att) {
    type = att.isVideo ? 'video' : 'file'
    name = att.fileName
    size = att.fileSize
    url = att.previewURL || att.downloadURL || ''
    thumb = att.coverURL || ''
    fileID = att.fileID
  } else if (isImageMessage(msg)) {
    type = 'image'
    url = imageUrlFromMessage(msg)
    name = String(msg.payload?.imageInfoArray?.[0]?.name || msg.payload?.fileName || '图片')
    size = Number(msg.payload?.imageInfoArray?.[0]?.size || msg.payload?.fileSize || 0)
    thumb = url
  } else if (isVideoMessage(msg)) {
    const v = videoInfoFromMessage(msg)
    type = 'video'
    url = v.url
    thumb = v.snapshot
    name = String(msg.payload?.videoName || msg.payload?.fileName || '视频')
    size = Number(msg.payload?.videoSize || msg.payload?.fileSize || 0)
  } else if (isSoundMessage(msg)) {
    const s = soundInfoFromMessage(msg)
    type = 'audio'
    url = s.url
    name = String(msg.payload?.fileName || `语音${s.second ? ` ${s.second}s` : ''}` || '语音')
    size = Number(msg.payload?.fileSize || 0)
  } else if (isFileMessage(msg)) {
    const f = fileInfoFromMessage(msg)
    type = 'file'
    url = f.url
    name = f.name
    size = f.size
  }
  if (!type || (!url && !fileID)) return null
  const from = String(msg.from || msg.fromAccount || '')
  return {
    id,
    conversationID: String(msg.conversationID || ''),
    from,
    nick: String(msg.nick || msg.nameCard || msg.from || ''),
    time: Number(msg.time || 0),
    name,
    size,
    url,
    thumb,
    kind: classifyFile(name, type),
    type,
    mine: msg.flow === 'out' || msg.flow === 'outward',
    fileID: fileID || undefined,
  }
}

function mergeFiles(lists: HubFile[][]) {
  const map = new Map<string, HubFile>()
  lists.flat().forEach((f) => {
    if (!map.has(f.id)) map.set(f.id, f)
  })
  return Array.from(map.values()).sort((a, b) => b.time - a.time)
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function extractCloudMsgs(res: any) {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const groups = (res?.data?.searchResultList || res?.data?.messageList || []) as any[]
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const cloudMsgs: any[] = []
  groups.forEach((item) => {
    if (Array.isArray(item?.messageList)) cloudMsgs.push(...item.messageList)
    else if (item?.ID || item?.payload) cloudMsgs.push(item)
  })
  return cloudMsgs
}

const FILE_KEY_BATCHES = [
  ['jpg', 'png', 'jpeg', 'gif', 'webp'],
  ['mp4', 'mov', 'pdf', 'xls', 'xlsx'],
  ['doc', 'docx', 'zip', 'apk', 'wav'],
]

async function collectByMessageList(conversationIDs: string[]) {
  const c = getChat()
  const pages: HubFile[][] = []
  for (const conversationID of conversationIDs.filter(Boolean).slice(0, 50)) {
    try {
      const res = await c.getMessageList({ conversationID, count: 40 })
      const list = (res?.data?.messageList || []) as unknown[]
      pages.push(list.map(toHubFile).filter((x): x is HubFile => !!x))
    } catch {
      /* 单会话失败跳过 */
    }
  }
  return mergeFiles(pages)
}

async function collectByCloudSearch(opts?: {
  conversationID?: string
  keyword?: string
  conversationIDs?: string[]
}) {
  const c = getChat()
  if (typeof c.searchCloudMessages !== 'function') return []
  const TYPES = c.TYPES || {}
  const typeList = [
    TYPES.MSG_FILE || 'TIMFileElem',
    TYPES.MSG_IMAGE || 'TIMImageElem',
    TYPES.MSG_VIDEO || 'TIMVideoFileElem',
    TYPES.MSG_AUDIO || 'TIMSoundElem',
    TYPES.MSG_CUSTOM || 'TIMCustomElem',
    TYPES.MSG_TEXT || 'TIMTextElem',
  ]
  const keyword = opts?.keyword?.trim()
  const batches = keyword ? [[keyword]] : [...FILE_KEY_BATCHES, ['CN_ATTACHMENT_V1', 'chat_large_attachment']]
  const pages: HubFile[][] = []

  async function run(conversationID: string | undefined, keywordList: string[]) {
    const res = await c.searchCloudMessages({
      conversationID,
      keywordList,
      keywordListMatchType: 'or',
      messageTypeList: typeList,
      timePosition: 0,
      timePeriod: 0,
      count: 50,
    })
    return extractCloudMsgs(res).map(toHubFile).filter((x): x is HubFile => !!x)
  }

  for (const keys of batches) {
    try {
      pages.push(await run(opts?.conversationID, keys))
    } catch {
      /* 云端搜失败时靠会话历史 */
    }
  }
  return mergeFiles(pages)
}

export async function searchHubFiles(opts?: {
  conversationID?: string
  keyword?: string
  conversationIDs?: string[]
}): Promise<HubFile[]> {
  const ids = opts?.conversationID
    ? [opts.conversationID]
    : (opts?.conversationIDs || []).filter(Boolean)
  const [localHist, cloud] = await Promise.all([
    collectByMessageList(ids),
    collectByCloudSearch(opts).catch(() => [] as HubFile[]),
  ])
  return mergeFiles([localHist, cloud])
}

export function filesFromLocal(messages: ChatMessage[]): HubFile[] {
  return messages.map(toHubFile).filter((x): x is HubFile => !!x)
}

export function formatHubSize(n: number) {
  return formatFileSize(n)
}

export function timeGroupLabel(ts: number): string {
  if (!ts) return '更早'
  const d = new Date(ts < 1e12 ? ts * 1000 : ts)
  const now = new Date()
  const startToday = new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime()
  const weekday = new Date(startToday).getDay() || 7
  const startWeek = startToday - (weekday - 1) * 86400000
  if (d.getTime() >= startWeek) return '本周'
  if (d.getFullYear() === now.getFullYear()) {
    return `${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
  }
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`
}

export function formatHubTime(ts: number) {
  if (!ts) return ''
  const d = new Date(ts < 1e12 ? ts * 1000 : ts)
  const hh = String(d.getHours()).padStart(2, '0')
  const mm = String(d.getMinutes()).padStart(2, '0')
  return `${d.getMonth() + 1}/${d.getDate()} ${hh}:${mm}`
}
