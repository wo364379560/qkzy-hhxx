import { useEffect, useRef, useState } from 'react'
import {
  SEARCH_TABS,
  SEARCH_TIMES,
  formatSearchTime,
  highlightKeyword,
  searchConversationMessages,
  type SearchHit,
  type SearchTab,
  type SearchTimeKey,
} from '@/im/searchMessages'
import styles from './ChatSearchPanel.module.css'

type Props = {
  conversationID: string
  localMessages: unknown[]
  onClose: () => void
  onJump: (id: string) => void
}

export function ChatSearchPanel({ conversationID, localMessages, onClose, onJump }: Props) {
  const [keyword, setKeyword] = useState('')
  const [tab, setTab] = useState<SearchTab>('text')
  const [timeKey, setTimeKey] = useState<SearchTimeKey>('all')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [hits, setHits] = useState<SearchHit[]>([])
  const seqRef = useRef(0)

  async function runSearch(nextKeyword = keyword, nextTab = tab, nextTime = timeKey) {
    const my = ++seqRef.current
    setLoading(true)
    setError('')
    try {
      const list = await searchConversationMessages({
        conversationID,
        keyword: nextKeyword,
        tab: nextTab,
        timeKey: nextTime,
        localMessages,
      })
      if (my !== seqRef.current) return
      setHits(list)
    } catch (e) {
      if (my !== seqRef.current) return
      setError(e instanceof Error ? e.message : '搜索失败')
      setHits([])
    } finally {
      if (my === seqRef.current) setLoading(false)
    }
  }

  useEffect(() => {
    void runSearch(keyword, tab, timeKey)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tab, timeKey, conversationID])

  useEffect(() => {
    const timer = window.setTimeout(() => {
      void runSearch(keyword, tab, timeKey)
    }, 280)
    return () => window.clearTimeout(timer)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [keyword])

  function emptyHint() {
    if (tab === 'text' && !keyword.trim()) return '请输入关键词搜索文本'
    return '没有找到相关记录'
  }

  const timeLabel = SEARCH_TIMES.find((t) => t.key === timeKey)?.label || '全部'

  return (
    <div className={styles.mask} onClick={(e) => e.target === e.currentTarget && onClose()}>
      <div className={styles.panel}>
        <div className={styles.head}>
          <div className={styles.title}>搜索会话内容</div>
          <button type="button" className={styles.close} onClick={onClose}>
            ×
          </button>
        </div>

        <div className={styles.tabs}>
          {SEARCH_TABS.map((t) => (
            <button
              key={t.key}
              type="button"
              className={tab === t.key ? styles.on : ''}
              onClick={() => setTab(t.key)}
            >
              {t.label}
            </button>
          ))}
        </div>

        <div className={styles.bar}>
          <span className={styles.mag}>⌕</span>
          <input
            type="search"
            value={keyword}
            placeholder="搜索聊天记录"
            onChange={(e) => setKeyword(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') void runSearch()
            }}
          />
          {keyword ? (
            <button
              type="button"
              className={styles.clear}
              onClick={() => {
                setKeyword('')
                void runSearch('')
              }}
            >
              ×
            </button>
          ) : null}
        </div>

        <div className={styles.times}>
          <button
            type="button"
            className={timeKey === 'all' ? styles.on : ''}
            onClick={() => setTimeKey('all')}
          >
            选择时间: {timeLabel}
          </button>
          {SEARCH_TIMES.filter((x) => x.key !== 'all').map((t) => (
            <button
              key={t.key}
              type="button"
              className={timeKey === t.key ? styles.on : ''}
              onClick={() => setTimeKey(t.key)}
            >
              {t.label}
            </button>
          ))}
        </div>

        <div className={styles.results}>
          {loading ? <div className={styles.hint}>搜索中…</div> : null}
          {!loading && error ? <div className={`${styles.hint} ${styles.err}`}>{error}</div> : null}
          {!loading && !error && !hits.length ? <div className={styles.hint}>{emptyHint()}</div> : null}
          {!loading && !error
            ? hits.map((h) => (
                <button key={h.id} type="button" className={styles.hit} onClick={() => onJump(h.id)}>
                  <div className={styles.avatar}>
                    {h.avatar ? <img src={h.avatar} alt="" /> : (h.nick || h.from).slice(0, 1)}
                  </div>
                  <div className={styles.main}>
                    <div className={styles.top}>
                      <span className={styles.name}>{h.nick || h.from}</span>
                      <span className={styles.time}>{formatSearchTime(h.time)}</span>
                    </div>
                    <div className={styles.text}>
                      {highlightKeyword(h.text, keyword).map((seg, i) =>
                        seg.hit ? <em key={i}>{seg.text}</em> : <span key={i}>{seg.text}</span>,
                      )}
                    </div>
                  </div>
                </button>
              ))
            : null}
        </div>
      </div>
    </div>
  )
}
