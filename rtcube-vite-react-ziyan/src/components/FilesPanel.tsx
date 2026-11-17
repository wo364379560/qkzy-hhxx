import { useEffect, useMemo, useState } from 'react'
import {
  FILE_KINDS,
  filesFromLocal,
  formatHubSize,
  formatHubTime,
  searchHubFiles,
  timeGroupLabel,
  type FileKind,
  type HubFile,
} from '@/im/fileHub'
import {
  downloadMediaAuto,
  getMediaCache,
  openCachedMedia,
  revealCachedMedia,
  subscribeMediaCache,
} from '@/im/mediaCache'
import { resolveAttachmentUrl } from '@/im/largeAttachment'
import { useAppStore } from '@/store/AppStore'
import { convTitle, isGroupConversation } from '@/utils/conversation'
import { MediaViewer, type MediaPreviewItem } from './MediaViewer'
import styles from './FilesPanel.module.css'

type NavKey = 'mine' | 'sender' | 'conv' | 'group'

type Props = {
  onOpenChat?: (conversationID: string) => void
}

const NAV: { key: NavKey; label: string }[] = [
  { key: 'mine', label: '我的文件' },
  { key: 'sender', label: '发送人' },
  { key: 'conv', label: '会话' },
  { key: 'group', label: '群文件' },
]

let hubCache: HubFile[] = []

export function FilesPanel({ onOpenChat }: Props) {
  const { messages, conversations, session, ready } = useAppStore()
  const selfId = session?.userId || ''
  const [nav, setNav] = useState<NavKey>('mine')
  const [midKey, setMidKey] = useState('')
  const [keyword, setKeyword] = useState('')
  const [kind, setKind] = useState<FileKind>('all')
  const [asc, setAsc] = useState(false)
  const [loading, setLoading] = useState(false)
  const [cloud, setCloud] = useState<HubFile[]>(hubCache)
  const [preview, setPreview] = useState<MediaPreviewItem | null>(null)
  const [, bumpCache] = useState(0)
  const convKey = conversations.map((c) => c.conversationID).join('|')

  useEffect(() => subscribeMediaCache(() => bumpCache((n) => n + 1)), [])

  useEffect(() => {
    if (!ready) return
    let cancelled = false
    setLoading(true)
    ;(async () => {
      try {
        const list = await searchHubFiles({
          conversationIDs: convKey ? convKey.split('|') : [],
        })
        if (cancelled) return
        hubCache = list
        setCloud(list)
      } catch {
        if (!cancelled) setCloud(hubCache)
      } finally {
        if (!cancelled) setLoading(false)
      }
    })()
    return () => {
      cancelled = true
    }
  }, [ready, convKey])

  const files = useMemo(() => {
    const map = new Map<string, HubFile>()
    ;[...cloud, ...filesFromLocal(messages)].forEach((f) => {
      if (!map.has(f.id)) map.set(f.id, f)
    })
    return Array.from(map.values()).sort((a, b) => b.time - a.time)
  }, [cloud, messages])

  const midItems = useMemo(() => {
    if (nav === 'mine') return []
    if (nav === 'sender') {
      const map = new Map<string, { key: string; title: string; count: number }>()
      files.forEach((f) => {
        const key = f.from || 'unknown'
        const cur = map.get(key)
        if (cur) cur.count += 1
        else map.set(key, { key, title: f.nick || key, count: 1 })
      })
      return Array.from(map.values())
    }
    const ids = new Set<string>()
    files.forEach((f) => {
      if (f.conversationID) ids.add(f.conversationID)
    })
    conversations.forEach((c) => {
      if (nav === 'group' && !isGroupConversation(c.conversationID)) return
      ids.add(c.conversationID)
    })
    return Array.from(ids)
      .filter((id) => (nav === 'group' ? isGroupConversation(id) : true))
      .map((id) => {
        const conv = conversations.find((c) => c.conversationID === id)
        const count = files.filter((f) => f.conversationID === id).length
        return {
          key: id,
          title: conv ? convTitle(conv) : id.replace(/^(C2C|GROUP)/, ''),
          count,
        }
      })
      .sort((a, b) => b.count - a.count)
  }, [nav, files, conversations])

  useEffect(() => {
    setMidKey((prev) => (midItems.some((i) => i.key === prev) ? prev : midItems[0]?.key || ''))
  }, [nav, midItems])

  const visible = useMemo(() => {
    const q = keyword.trim().toLowerCase()
    return files
      .filter((f) => {
        if (nav === 'mine') return f.mine || (!!selfId && f.from === selfId)
        if (nav === 'sender') return !midKey || f.from === midKey
        if (nav === 'conv' || nav === 'group') return !midKey || f.conversationID === midKey
        return true
      })
      .filter((f) => (kind === 'all' ? true : f.kind === kind))
      .filter((f) => {
        if (!q) return true
        return `${f.name} ${f.nick} ${f.from}`.toLowerCase().includes(q)
      })
      .sort((a, b) => (asc ? a.time - b.time : b.time - a.time))
  }, [files, nav, midKey, kind, keyword, selfId, asc])

  const groups = useMemo(() => {
    const map = new Map<string, HubFile[]>()
    visible.forEach((f) => {
      const g = timeGroupLabel(f.time)
      const list = map.get(g) || []
      list.push(f)
      map.set(g, list)
    })
    return Array.from(map.entries())
  }, [visible])

  async function resolveHubUrl(f: HubFile, download = false) {
    if (f.fileID) {
      try {
        return await resolveAttachmentUrl(f.fileID, download)
      } catch {
        if (f.url.startsWith('http')) return f.url
        throw new Error('无法获取文件地址')
      }
    }
    if (f.url) return f.url
    throw new Error('文件地址无效')
  }

  async function openFile(f: HubFile) {
    if (f.type === 'image' || f.type === 'video') {
      try {
        const url = await resolveHubUrl(f, false)
        setPreview({ kind: f.type, url, name: f.name })
      } catch (e) {
        alert(e instanceof Error ? e.message : '无法打开')
      }
      return
    }
    const cached = getMediaCache(f.id)
    if (cached) {
      try {
        await openCachedMedia(cached, f.url)
      } catch (e) {
        alert(e instanceof Error ? e.message : '打开失败')
      }
      return
    }
    try {
      const url = await resolveHubUrl(f, true)
      if (f.from === selfId && url && !f.fileID) {
        window.open(url, '_blank', 'noopener,noreferrer')
        return
      }
      const item = await downloadMediaAuto(f.id, url, f.name)
      await openCachedMedia(item, url)
    } catch (e) {
      alert(e instanceof Error ? e.message : '下载失败')
    }
  }

  async function locateFile(f: HubFile) {
    const cached = getMediaCache(f.id)
    if (!cached) {
      alert('还没有下载到本机')
      return
    }
    try {
      await revealCachedMedia(cached)
    } catch (e) {
      alert(e instanceof Error ? e.message : '无法定位文件')
    }
  }

  return (
    <div className={`${styles.wrap} ${nav === 'mine' ? styles.noMid : ''}`}>
      <aside className={styles.nav}>
        <h3>文件</h3>
        {NAV.map((item) => (
          <button
            key={item.key}
            type="button"
            className={nav === item.key ? styles.navOn : undefined}
            onClick={() => setNav(item.key)}
          >
            {item.label}
          </button>
        ))}
      </aside>

      {nav !== 'mine' ? (
        <aside className={styles.mid}>
          {midItems.map((item) => (
            <button
              key={item.key}
              type="button"
              className={midKey === item.key ? styles.midOn : undefined}
              onClick={() => setMidKey(item.key)}
            >
              <strong>{item.title}</strong>
              <em>{item.count}</em>
            </button>
          ))}
          {!midItems.length ? <p className={styles.empty}>暂无列表</p> : null}
        </aside>
      ) : null}

      <section className={styles.main}>
        <header className={styles.toolbar}>
          <input
            value={keyword}
            placeholder="搜索文件名 / 发送人"
            onChange={(e) => setKeyword(e.target.value)}
          />
          <select value={kind} onChange={(e) => setKind(e.target.value as FileKind)}>
            {FILE_KINDS.map((k) => (
              <option key={k.key} value={k.key}>
                {k.label}
              </option>
            ))}
          </select>
          <button type="button" onClick={() => setAsc((v) => !v)}>
            {asc ? '时间正序' : '时间倒序'}
          </button>
        </header>

        {!ready ? <p className={styles.empty}>正在连接，随后加载聊天文件…</p> : null}
        {ready && loading ? <p className={styles.empty}>正在加载会话文件…</p> : null}
        {ready && !loading && !visible.length ? (
          <p className={styles.empty}>
            {nav === 'mine' && files.length
              ? '「我的文件」只看你发出去的。收到的文件请点左侧「会话」或「发送人」。'
              : '没有匹配的文件。这里只展示聊天里发过的文件，不是网盘。'}
          </p>
        ) : null}

        <div className={styles.scroller}>
          {groups.map(([label, list]) => (
            <div key={label} className={styles.group}>
              <h4>{label}</h4>
              {list.map((f) => (
                <div key={f.id} className={styles.item}>
                  <button type="button" className={styles.thumb} onClick={() => openFile(f)}>
                    {f.thumb && f.type !== 'file' && f.type !== 'audio' ? (
                      <img src={f.thumb} alt="" />
                    ) : (
                      <span>{f.type === 'audio' ? '♪' : '📄'}</span>
                    )}
                  </button>
                  <button type="button" className={styles.meta} onClick={() => openFile(f)}>
                    <strong>{f.name}</strong>
                    <em>
                      {f.nick || f.from}
                      {f.size ? ` · ${formatHubSize(f.size)}` : ''}
                      {` · ${formatHubTime(f.time)}`}
                    </em>
                  </button>
                  {getMediaCache(f.id)?.filePath ? (
                    <button type="button" className={styles.jump} onClick={() => void locateFile(f)}>
                      文件夹
                    </button>
                  ) : null}
                  {f.conversationID && onOpenChat ? (
                    <button
                      type="button"
                      className={styles.jump}
                      onClick={() => onOpenChat(f.conversationID)}
                    >
                      会话
                    </button>
                  ) : null}
                </div>
              ))}
            </div>
          ))}
        </div>
      </section>

      <MediaViewer item={preview} onClose={() => setPreview(null)} />
    </div>
  )
}
