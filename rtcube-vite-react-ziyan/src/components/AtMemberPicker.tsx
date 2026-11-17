import { useCallback, useEffect, useState } from 'react'
import { AT_MEMBER_PAGE_SIZE } from '@/config/runtime'
import { searchGroupMembersPage } from '@/im/ImClient'
import type { GroupMember } from '@/im/types'
import styles from './AtMemberPicker.module.css'

type Props = {
  groupId: string
  open: boolean
  keyword: string
  onPick: (member: GroupMember | 'all') => void
  onClose: () => void
}

/** 大群 @：搜索 + 分页，禁止一次拉全员 */
export function AtMemberPicker({ groupId, open, keyword, onPick, onClose }: Props) {
  const [members, setMembers] = useState<GroupMember[]>([])
  const [offset, setOffset] = useState(0)
  const [hasMore, setHasMore] = useState(true)
  const [loading, setLoading] = useState(false)

  const reload = useCallback(async () => {
    if (!open || !groupId) return
    setLoading(true)
    try {
      const page = await searchGroupMembersPage(groupId, keyword, 0, AT_MEMBER_PAGE_SIZE)
      setMembers(page.members)
      setOffset(page.nextOffset)
      setHasMore(page.hasMore)
    } catch (e) {
      console.error(e)
      setMembers([])
      setHasMore(false)
    } finally {
      setLoading(false)
    }
  }, [open, groupId, keyword])

  useEffect(() => {
    const t = window.setTimeout(() => void reload(), 200)
    return () => window.clearTimeout(t)
  }, [reload])

  async function loadMore() {
    if (loading || !hasMore) return
    setLoading(true)
    try {
      const page = await searchGroupMembersPage(groupId, keyword, offset, AT_MEMBER_PAGE_SIZE)
      setMembers((prev) => {
        const seen = new Set(prev.map((m) => m.userID))
        return [...prev, ...page.members.filter((m) => !seen.has(m.userID))]
      })
      setOffset(page.nextOffset)
      setHasMore(page.hasMore)
    } finally {
      setLoading(false)
    }
  }

  if (!open) return null

  return (
    <div className={styles.panel} role="listbox">
      <div className={styles.head}>
        <span>选择提醒的人</span>
        <button type="button" onClick={onClose}>
          关闭
        </button>
      </div>
      <button type="button" className={styles.item} onClick={() => onPick('all')}>
        @全体成员
      </button>
      <div
        className={styles.list}
        onScroll={(e) => {
          const el = e.currentTarget
          if (el.scrollTop + el.clientHeight >= el.scrollHeight - 40) void loadMore()
        }}
      >
        {members.map((m) => (
          <button
            key={m.userID}
            type="button"
            className={styles.item}
            onClick={() => onPick(m)}
          >
            <span className={styles.avatar}>
              {m.avatar ? <img src={m.avatar} alt="" /> : m.nick.slice(0, 1)}
            </span>
            <span>{m.nick}</span>
          </button>
        ))}
        {loading ? <div className={styles.tip}>加载中…</div> : null}
        {!loading && !members.length ? <div className={styles.tip}>无匹配（分页搜索）</div> : null}
      </div>
    </div>
  )
}
