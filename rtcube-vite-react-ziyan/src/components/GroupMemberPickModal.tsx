import { useEffect, useMemo, useState } from 'react'
import { fetchFriendList, fetchGroupMemberIdSet, type FriendRow } from '@/im/ImClient'
import type { GroupMember } from '@/im/types'
import styles from './CreateGroupModal.module.css'

function normId(id: string) {
  return String(id || '').trim()
}

export type MemberPickItem = {
  userID: string
  nick: string
  avatar: string
  inGroup?: boolean
}

type Props = {
  open: boolean
  mode: 'add' | 'remove'
  groupId?: string
  members: GroupMember[]
  excludeIds?: string[]
  onClose: () => void
  onConfirm: (userIDs: string[]) => Promise<void> | void
}

export function GroupMemberPickModal({
  open,
  mode,
  groupId,
  members,
  excludeIds = [],
  onClose,
  onConfirm,
}: Props) {
  const [q, setQ] = useState('')
  const [friends, setFriends] = useState<FriendRow[]>([])
  const [inGroupIds, setInGroupIds] = useState<Set<string>>(new Set())
  const [selected, setSelected] = useState<string[]>([])
  const [busy, setBusy] = useState(false)

  const isAdd = mode === 'add'
  const exclude = useMemo(
    () => new Set(excludeIds.map(normId).filter(Boolean)),
    [excludeIds],
  )

  useEffect(() => {
    if (!open) return
    setQ('')
    setSelected([])
    setInGroupIds(new Set())
    if (!isAdd) return
    let cancelled = false
    void (async () => {
      try {
        const [list, ids] = await Promise.all([
          fetchFriendList(),
          groupId ? fetchGroupMemberIdSet(groupId) : Promise.resolve(new Set<string>()),
        ])
        if (cancelled) return
        setInGroupIds(new Set(Array.from(ids).map(normId)))
        setFriends(list.filter((f) => normId(f.userID)))
      } catch {
        if (!cancelled) setFriends([])
      }
    })()
    return () => {
      cancelled = true
    }
  }, [open, isAdd, groupId])

  const source: MemberPickItem[] = useMemo(() => {
    if (isAdd) {
      // 已在群里的好友不显示、不可勾选
      return friends
        .map((f) => ({
          userID: normId(f.userID),
          nick: f.remark || f.nick || f.userID,
          avatar: f.avatar,
          inGroup: inGroupIds.has(normId(f.userID)),
        }))
        .filter((f) => f.userID && !exclude.has(f.userID) && !f.inGroup)
    }
    return members
      .map((m) => ({
        userID: normId(m.userID),
        nick: m.nick || m.userID,
        avatar: m.avatar,
      }))
      .filter((m) => m.userID && !exclude.has(m.userID))
  }, [isAdd, friends, members, exclude, inGroupIds])

  const filtered = useMemo(() => {
    const key = q.trim().toLowerCase()
    if (!key) return source
    return source.filter((x) => `${x.nick} ${x.userID}`.toLowerCase().includes(key))
  }, [source, q])

  const picked = source.filter((x) => selected.includes(x.userID))

  function toggle(id: string) {
    const uid = normId(id)
    if (!uid) return
    if (isAdd && inGroupIds.has(uid)) return
    setSelected((prev) => (prev.includes(uid) ? prev.filter((x) => x !== uid) : [...prev, uid]))
  }

  async function confirm() {
    if (!selected.length || busy) return
    setBusy(true)
    try {
      await onConfirm(selected)
      onClose()
    } catch (e) {
      alert(e instanceof Error ? e.message : '操作失败')
    } finally {
      setBusy(false)
    }
  }

  if (!open) return null

  return (
    <div className={styles.mask} onClick={onClose} role="presentation">
      <div className={styles.split} onClick={(e) => e.stopPropagation()}>
        <div className={styles.left}>
          <div className={styles.search}>
            <input
              value={q}
              placeholder="搜索"
              onChange={(e) => setQ(e.target.value)}
            />
          </div>
          <div className={styles.selectAll}>{isAdd ? '我的好友' : '当前群聊'}</div>
          <div className={styles.list}>
            {filtered.map((m) => {
              const on = selected.includes(m.userID)
              return (
                <button
                  key={m.userID}
                  type="button"
                  className={styles.row}
                  onClick={() => toggle(m.userID)}
                >
                  <i className={on ? styles.checkOn : styles.check} />
                  <span className={styles.avatar}>
                    {m.avatar ? <img src={m.avatar} alt="" /> : m.nick.slice(0, 1)}
                  </span>
                  <span>{m.nick}</span>
                </button>
              )
            })}
            {!filtered.length ? (
              <div className={styles.empty}>{isAdd ? '没有可添加的好友' : '没有可移除的成员'}</div>
            ) : null}
          </div>
        </div>
        <div className={styles.right}>
          <h3>已选 {picked.length} 个联系人</h3>
          <div className={styles.picked}>
            {picked.map((m) => (
              <div key={m.userID} className={styles.pickedRow}>
                <span className={styles.avatar}>
                  {m.avatar ? <img src={m.avatar} alt="" /> : m.nick.slice(0, 1)}
                </span>
                <span className={styles.pickedName}>{m.nick}</span>
                <button type="button" onClick={() => toggle(m.userID)}>
                  ×
                </button>
              </div>
            ))}
          </div>
          <p className={styles.count} style={{ color: isAdd ? undefined : '#e11d48' }}>
            {isAdd ? '添加' : '移除'}
          </p>
          <div className={styles.actions}>
            <button type="button" className={styles.cancel} onClick={onClose}>
              取消
            </button>
            <button
              type="button"
              className={styles.done}
              disabled={!selected.length || busy}
              onClick={() => void confirm()}
            >
              {busy ? '处理中' : '确定'}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
