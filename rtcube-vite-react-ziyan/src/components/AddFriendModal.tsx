import { useState } from 'react'
import {
  searchGroupByID,
  searchUsers,
  searchUsersExact,
  type SearchUserResult,
} from '@/api/search'
import { addFriend, checkIsFriend, formatFriendAddError } from '@/im/friends'
import styles from './AddFriendModal.module.css'

type UserRow = SearchUserResult & { isFriend?: boolean }

type Props = {
  open: boolean
  onClose: () => void
  onOpenProfile: (userId: string, nickname?: string, avatar?: string) => void
  onOpenGroup?: (groupId: string) => void
}

export function AddFriendModal({ open, onClose, onOpenProfile, onOpenGroup }: Props) {
  const [keyword, setKeyword] = useState('')
  const [loading, setLoading] = useState(false)
  const [tip, setTip] = useState('')
  const [users, setUsers] = useState<UserRow[]>([])
  const [group, setGroup] = useState<{ id: string; name: string } | null>(null)
  const [addingId, setAddingId] = useState('')

  if (!open) return null

  async function search() {
    setTip('')
    setUsers([])
    setGroup(null)
    const q = keyword.trim()
    if (!q) {
      setTip('请输入信号 / 手机号 / 用户 ID / 群 ID')
      return
    }
    setLoading(true)
    try {
      const [fuzzy, exact, g] = await Promise.all([
        searchUsers(q).catch(() => [] as SearchUserResult[]),
        searchUsersExact(q).catch(() => [] as SearchUserResult[]),
        searchGroupByID(q).catch(() => null),
      ])
      const map = new Map<string, SearchUserResult>()
      ;[...fuzzy, ...exact].forEach((u) => map.set(u.im_user_id, u))
      const merged = Array.from(map.values())
      const withFlag = await Promise.all(
        merged.map(async (u) => ({
          ...u,
          isFriend: await checkIsFriend(u.im_user_id).catch(() => false),
        })),
      )
      setUsers(withFlag)
      if (g) setGroup({ id: g.group_id, name: g.group_name || g.group_id })
      if (!withFlag.length && !g) setTip('未找到结果')
    } catch (e) {
      setTip(e instanceof Error ? e.message : '搜索失败')
    } finally {
      setLoading(false)
    }
  }

  async function onUserClick(u: UserRow) {
    const friend = await checkIsFriend(u.im_user_id).catch(() => false)
    if (!friend) {
      setUsers((prev) =>
        prev.map((x) => (x.im_user_id === u.im_user_id ? { ...x, isFriend: false } : x)),
      )
      setTip('对方还不是好友，请先点右侧 + 添加')
      return
    }
    onClose()
    onOpenProfile(u.im_user_id, u.nickname, u.avatar_url)
  }

  async function onAdd(u: UserRow) {
    if (u.isFriend || addingId) return
    if (u.can_add_friend === false) {
      setTip(u.add_blocked_reason || '当前账号属于其他服务空间无法直接添加。')
      return
    }
    setAddingId(u.im_user_id)
    setTip('')
    try {
      await addFriend(u.im_user_id, '你好，想加你为好友')
      setTip('已发送好友申请')
    } catch (e) {
      setTip(formatFriendAddError(e))
    } finally {
      setAddingId('')
    }
  }

  return (
    <div className={styles.mask} onClick={onClose}>
      <div className={styles.modal} onClick={(e) => e.stopPropagation()} role="dialog">
        <header className={styles.head}>
          <h3>添加好友</h3>
          <button type="button" onClick={onClose}>
            ✕
          </button>
        </header>
        <div className={styles.body}>
          <div className={styles.searchRow}>
            <input
              value={keyword}
              placeholder="信号 / 手机号 / 用户 ID / 群 ID"
              onChange={(e) => setKeyword(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') void search()
              }}
            />
            <button type="button" disabled={loading} onClick={() => void search()}>
              {loading ? '…' : '搜索'}
            </button>
          </div>
          {tip ? (
            <p className={/服务空间|无法直接|失败/.test(tip) ? styles.tipErr : styles.tip}>{tip}</p>
          ) : null}

          {group ? (
            <button
              type="button"
              className={styles.row}
              onClick={() => {
                onClose()
                onOpenGroup?.(group.id)
              }}
            >
              <div className={`${styles.avatar} ${styles.groupAv}`}>群</div>
              <div className={styles.meta}>
                <strong>{group.name}</strong>
                <span>群 {group.id}</span>
              </div>
              <span className={styles.chev}>›</span>
            </button>
          ) : null}

          {users.length ? <div className={styles.section}>用户</div> : null}
          <ul className={styles.list}>
            {users.map((u) => (
              <li key={u.im_user_id}>
                {u.isFriend ? (
                  <button type="button" className={styles.row} onClick={() => void onUserClick(u)}>
                    <div className={styles.avatar}>
                      {u.avatar_url ? (
                        <img src={u.avatar_url} alt="" />
                      ) : (
                        (u.nickname || u.im_user_id).slice(0, 1)
                      )}
                    </div>
                    <div className={styles.meta}>
                      <strong>
                        {u.nickname || u.im_user_id}
                        <em className={styles.tag}>好友</em>
                      </strong>
                      <span>
                        {u.wanan_id ? `信号 ${u.wanan_id} · ` : ''}
                        ID {u.im_user_id}
                      </span>
                    </div>
                  </button>
                ) : (
                  <div className={`${styles.row} ${styles.rowStatic}`}>
                    <div className={styles.avatar}>
                      {u.avatar_url ? (
                        <img src={u.avatar_url} alt="" />
                      ) : (
                        (u.nickname || u.im_user_id).slice(0, 1)
                      )}
                    </div>
                    <div className={styles.meta}>
                      <strong>{u.nickname || u.im_user_id}</strong>
                      <span>
                        {u.wanan_id ? `信号 ${u.wanan_id} · ` : ''}
                        ID {u.im_user_id}
                      </span>
                    </div>
                  </div>
                )}
                {!u.isFriend ? (
                  <button
                    type="button"
                    className={styles.addBtn}
                    disabled={addingId === u.im_user_id}
                    title="加好友"
                    onClick={() => void onAdd(u)}
                  >
                    +
                  </button>
                ) : (
                  <span className={styles.chev}>›</span>
                )}
              </li>
            ))}
          </ul>
        </div>
      </div>
    </div>
  )
}
