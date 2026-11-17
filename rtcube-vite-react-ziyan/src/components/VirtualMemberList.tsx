import { useMemo, useRef, useState, type MouseEvent, type UIEvent } from 'react'
import type { GroupMember } from '@/im/types'
import styles from './VirtualMemberList.module.css'

const ROW_H = 48
const OVERSCAN = 8

type Props = {
  members: GroupMember[]
  hasMore: boolean
  loading: boolean
  onLoadMore: () => void
  onPick?: (m: GroupMember) => void
  onKick?: (m: GroupMember) => void
  onContextMenu?: (e: MouseEvent, m: GroupMember) => void
  emptyText?: string
}

/** 简易虚拟列表 — 大群详情/成员栏不崩溃 */
export function VirtualMemberList({
  members,
  hasMore,
  loading,
  onLoadMore,
  onPick,
  onKick,
  onContextMenu,
  emptyText = '暂无成员',
}: Props) {
  const scrollerRef = useRef<HTMLDivElement>(null)
  const [scrollTop, setScrollTop] = useState(0)
  const [viewportH, setViewportH] = useState(400)

  const { start, end, offsetY, totalH } = useMemo(() => {
    const startIdx = Math.max(0, Math.floor(scrollTop / ROW_H) - OVERSCAN)
    const visible = Math.ceil(viewportH / ROW_H) + OVERSCAN * 2
    const endIdx = Math.min(members.length, startIdx + visible)
    return {
      start: startIdx,
      end: endIdx,
      offsetY: startIdx * ROW_H,
      totalH: members.length * ROW_H,
    }
  }, [members.length, scrollTop, viewportH])

  const slice = members.slice(start, end)

  function onScroll(e: UIEvent<HTMLDivElement>) {
    const el = e.currentTarget
    setScrollTop(el.scrollTop)
    setViewportH(el.clientHeight)
    const nearBottom = el.scrollTop + el.clientHeight >= el.scrollHeight - 80
    if (nearBottom && hasMore && !loading) onLoadMore()
  }

  if (!members.length && !loading) {
    return <div className={styles.empty}>{emptyText}</div>
  }

  return (
    <div className={styles.scroller} ref={scrollerRef} onScroll={onScroll}>
      <div className={styles.phantom} style={{ height: totalH }}>
        <div className={styles.window} style={{ transform: `translateY(${offsetY}px)` }}>
          {slice.map((m) => (
            <div
              key={m.userID}
              className={styles.row}
              onContextMenu={(e: MouseEvent) => {
                if (!onContextMenu) return
                e.preventDefault()
                e.stopPropagation()
                onContextMenu(e, m)
              }}
            >
              <button type="button" className={styles.main} onClick={() => onPick?.(m)}>
                <div className={styles.avatar}>
                  {m.avatar ? <img src={m.avatar} alt="" /> : m.nick.slice(0, 1)}
                </div>
                <div className={styles.meta}>
                  <span className={styles.name}>{m.nick}</span>
                  {m.role !== 'Member' ? (
                    <span className={styles.role}>{m.role === 'Owner' ? '群主' : '管理'}</span>
                  ) : null}
                </div>
              </button>
              {onKick && m.role !== 'Owner' ? (
                <button
                  type="button"
                  className={styles.kick}
                  title="移出群聊"
                  onClick={(e) => {
                    e.stopPropagation()
                    onKick(m)
                  }}
                >
                  踢
                </button>
              ) : null}
            </div>
          ))}
        </div>
      </div>
      {loading ? <div className={styles.loading}>加载中…</div> : null}
      {!hasMore && members.length > 0 ? (
        <div className={styles.tip}>已加载 {members.length} 人（分页）</div>
      ) : null}
    </div>
  )
}
