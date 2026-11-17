import { getChat } from './ImClient'
import { isLargeAttachmentMessage } from './largeAttachment'
import { messageCopyText } from './messageActions'
import { isFileMessage, isImageMessage, isSoundMessage, isVideoMessage } from './mediaPayload'

// eslint-disable-next-line @typescript-eslint/no-explicit-any
type AnyMsg = any

export type SearchTab = 'text' | 'file' | 'media' | 'other'
export type SearchTimeKey = 'all' | 'today' | 'threeDays' | 'sevenDays'

export type SearchHit = {
  id: string
  from: string
  nick: string
  avatar: string
  time: number
  text: string
  raw: unknown
}

export const SEARCH_TABS: { key: SearchTab; label: string }[] = [
  { key: 'text', label: '文本' },
  { key: 'file', label: '文件' },
  { key: 'media', label: '图片/视频' },
  { key: 'other', label: '其他' },
]

export const SEARCH_TIMES: { key: SearchTimeKey; label: string; period: number }[] = [
  { key: 'all', label: '全部', period: 0 },
  { key: 'today', label: '今天', period: 24 * 60 * 60 },
  { key: 'threeDays', label: '近三天', period: 3 * 24 * 60 * 60 },
  { key: 'sevenDays', label: '近七天', period: 7 * 24 * 60 * 60 },
]

function typeList(tab: SearchTab, c: AnyMsg): string[] {
  const t = c.TYPES || {}
  const text = t.MSG_TEXT || 'TIMTextElem'
  const file = t.MSG_FILE || 'TIMFileElem'
  const image = t.MSG_IMAGE || 'TIMImageElem'
  const video = t.MSG_VIDEO || 'TIMVideoFileElem'
  const audio = t.MSG_AUDIO || 'TIMSoundElem'
  const custom = t.MSG_CUSTOM || 'TIMCustomElem'
  const merger = t.MSG_MERGER || 'TIMRelayElem'
  const location = t.MSG_LOCATION || 'TIMLocationElem'
  if (tab === 'text') return [text]
  if (tab === 'file') return [file, custom, text]
  if (tab === 'media') return [image, video, custom]
  return [audio, custom, merger, location]
}

function inTimeRange(time: number, period: number) {
  if (!period) return true
  const end = Math.floor(Date.now() / 1000)
  const ts = time > 1e12 ? Math.floor(time / 1000) : time
  return ts >= end - period && ts <= end
}

function matchKeyword(msg: AnyMsg, keyword: string) {
  const k = keyword.trim().toLowerCase()
  if (!k) return true
  const payload = msg?.payload || {}
  const hay = [
    payload.text,
    payload.fileName,
    payload.description,
    payload.extension,
    messageCopyText(msg),
    msg.nick,
    msg.from,
    msg.fromAccount,
  ]
    .filter(Boolean)
    .join('\n')
    .toLowerCase()
  return hay.includes(k)
}

function matchTab(msg: AnyMsg, tab: SearchTab) {
  const type = String(msg?.type || '')
  if (tab === 'text') return (type.includes('TEXT') || type.includes('Text')) && !isLargeAttachmentMessage(msg)
  if (tab === 'file') return isFileMessage(msg) || type.includes('FILE') || type.includes('File')
  if (tab === 'media') {
    return (
      isImageMessage(msg) ||
      isVideoMessage(msg) ||
      type.includes('IMAGE') ||
      type.includes('VIDEO')
    )
  }
  return (
    isSoundMessage(msg) ||
    type.includes('SOUND') ||
    type.includes('AUDIO') ||
    type.includes('CUSTOM') ||
    type.includes('MERGER') ||
    type.includes('LOCATION')
  )
}

function toHit(msg: AnyMsg): SearchHit | null {
  if (!msg || msg.isRevoked || msg.isDeleted) return null
  const id = String(msg.ID || msg.id || '')
  if (!id) return null
  return {
    id,
    from: String(msg.from || msg.fromAccount || ''),
    nick: String(msg.nick || msg.nameCard || msg.from || ''),
    avatar: String(msg.avatar || ''),
    time: Number(msg.time || 0),
    text: messageCopyText(msg) || '[消息]',
    raw: msg,
  }
}

function mergeHits(lists: SearchHit[][]) {
  const map = new Map<string, SearchHit>()
  lists.flat().forEach((h) => {
    if (!map.has(h.id)) map.set(h.id, h)
  })
  return Array.from(map.values()).sort((a, b) => b.time - a.time)
}

function searchLocal(localMessages: AnyMsg[], keyword: string, tab: SearchTab, period: number) {
  return localMessages
    .filter((m) => matchTab(m, tab))
    .filter((m) => inTimeRange(Number(m?.time || 0), period))
    .filter((m) => matchKeyword(m, keyword))
    .map(toHit)
    .filter((h): h is SearchHit => !!h)
}

export async function searchConversationMessages(opts: {
  conversationID: string
  keyword: string
  tab: SearchTab
  timeKey: SearchTimeKey
  localMessages?: AnyMsg[]
}): Promise<SearchHit[]> {
  const keyword = opts.keyword.trim()
  const period = SEARCH_TIMES.find((t) => t.key === opts.timeKey)?.period || 0
  if (opts.tab === 'text' && !keyword) return []

  const localHits = searchLocal(opts.localMessages || [], keyword, opts.tab, period)

  const c = getChat()
  if (typeof c.searchCloudMessages !== 'function') return localHits

  try {
    const res = await c.searchCloudMessages({
      conversationID: opts.conversationID,
      keywordList: keyword ? [keyword] : undefined,
      keywordListMatchType: 'or',
      messageTypeList: typeList(opts.tab, c),
      timePosition: 0,
      timePeriod: period,
      count: 50,
    })
    const groups = (res?.data?.searchResultList || res?.data?.messageList || []) as AnyMsg[]
    const cloudMsgs: AnyMsg[] = []
    groups.forEach((item) => {
      if (Array.isArray(item?.messageList)) cloudMsgs.push(...item.messageList)
      else if (item?.ID || item?.payload) cloudMsgs.push(item)
    })
    const cloudHits = cloudMsgs.map(toHit).filter((h): h is SearchHit => !!h)
    return mergeHits([cloudHits, localHits])
  } catch {
    return localHits
  }
}

export function formatSearchTime(ts: number) {
  if (!ts) return ''
  const d = new Date(ts < 1e12 ? ts * 1000 : ts)
  const now = new Date()
  const hh = String(d.getHours()).padStart(2, '0')
  const mm = String(d.getMinutes()).padStart(2, '0')
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime()
  const target = new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime()
  const diff = today - target
  const oneDay = 24 * 60 * 60 * 1000
  if (diff <= 0) return `${hh}:${mm}`
  if (diff <= oneDay) return `昨天 ${hh}:${mm}`
  if (d.getFullYear() === now.getFullYear()) return `${d.getMonth() + 1}/${d.getDate()} ${hh}:${mm}`
  return `${d.getFullYear()}/${d.getMonth() + 1}/${d.getDate()}`
}

export function highlightKeyword(text: string, keyword: string) {
  const k = keyword.trim()
  if (!k) return [{ text, hit: false }]
  const out: { text: string; hit: boolean }[] = []
  const lower = text.toLowerCase()
  const needle = k.toLowerCase()
  let start = 0
  let idx = lower.indexOf(needle, start)
  while (idx >= 0) {
    if (idx > start) out.push({ text: text.slice(start, idx), hit: false })
    out.push({ text: text.slice(idx, idx + k.length), hit: true })
    start = idx + k.length
    idx = lower.indexOf(needle, start)
  }
  if (start < text.length) out.push({ text: text.slice(start), hit: false })
  return out.length ? out : [{ text, hit: false }]
}
