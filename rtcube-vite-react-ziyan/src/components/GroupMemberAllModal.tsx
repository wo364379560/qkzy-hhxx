import { useEffect, useState, type MouseEvent } from 'react'
import { applyMemberLivePatch, onGroupLiveChange } from '@/im/groupLive'
import { fetchGroupMemberPage } from '@/im/ImClient'
import type { GroupMember } from '@/im/types'
import {
  GroupMemberContextMenu,
  type MemberMenuState,
} from './GroupMemberContextMenu'
import styles from './GroupMemberAllModal.module.css'

type Props = {
  open: boolean
  groupId: string
  selfId?: string
  selfRole?: GroupMember['role']
  onClose: () => void
}

export function GroupMemberAllModal({
  open,
  groupId,
  selfId = '',
  selfRole = 'Member',
  onClose,
}: Props) {
  const [q, setQ] = useState('')
  const [members, setMembers] = useState<GroupMember[]>([])
  const [offset, setOffset] = useState(0)
  const [hasMore, setHasMore] = useState(true)
  const [loading, setLoading] = useState(false)
  const [menu, setMenu] = useState<MemberMenuState>(null)

  useEffect(() => {
    if (!open || !groupId) return
    setQ('')
    setMembers([])
    setOffset(0)
    setHasMore(true)
    setMenu(null)
    void load(0, true)
    const off = onGroupLiveChange((e) => {
      if (e.groupID !== groupId) return
      if (e.added?.length || e.removedIds?.length || e.updated?.length) {
        setMembers((prev) => applyMemberLivePatch(prev, e).next)
      }
    })
    return () => off()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, groupId])

  async function load(nextOffset: number, replace: boolean) {
    if (loading) return
    setLoading(true)
    try {
      const page = await fetchGroupMemberPage(groupId, nextOffset, 50)
      setMembers((prev) => (replace ? page.members : [...prev, ...page.members]))
      setOffset(page.nextOffset)
      setHasMore(page.hasMore)
    } catch {
      setHasMore(false)
    } finally {
      setLoading(false)
    }
  }

  if (!open) return null

  const key = q.trim().toLowerCase()
  const shown = key
    ? members.filter((m) => `${m.nick} ${m.userID}`.toLowerCase().includes(key))
    : members

  return (
    <div className={styles.mask} onClick={onClose} role="presentation">
      <div className={styles.modal} onClick={(e) => e.stopPropagation()}>
        <header className={styles.head}>
          <h3>群聊成员</h3>
          <button type="button" onClick={onClose}>
            ✕
          </button>
        </header>
        <div className={styles.search}>
          <input value={q} placeholder="搜索成员" onChange={(e) => setQ(e.target.value)} />
        </div>
        <div className={styles.list}>
          {shown.map((m) => (
            <div
              key={m.userID}
              className={styles.row}
              onContextMenu={(e: MouseEvent) => {
                e.preventDefault()
                e.stopPropagation()
                setMenu({ x: e.clientX, y: e.clientY, member: m })
              }}
            >
              <span className={styles.avatar}>
                {m.avatar ? <img src={m.avatar} alt="" /> : (m.nick || m.userID).slice(0, 1)}
              </span>
              <div>
                <strong>{m.nick || m.userID}</strong>
                <em>{m.role === 'Owner' ? '群主' : m.role === 'Admin' ? '管理员' : ''}</em>
              </div>
            </div>
          ))}
          {hasMore && !key ? (
            <button type="button" className={styles.more} onClick={() => void load(offset, false)}>
              {loading ? '加载中…' : '加载更多'}
            </button>
          ) : null}
          {!shown.length && !loading ? <p className={styles.empty}>暂无成员</p> : null}
        </div>
      </div>
      <GroupMemberContextMenu
        menu={menu}
        groupId={groupId}
        selfId={selfId}
        selfRole={selfRole}
        onClose={() => setMenu(null)}
        onChanged={() => void load(0, true)}
      />
    </div>
  )
}
