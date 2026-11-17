import { useEffect, useMemo, useState } from 'react'
import { isHiddenConversation } from '@/im/ImClient'
import { useAppStore } from '@/store/AppStore'
import { convAvatar, convTitle, isGroupConversation } from '@/utils/conversation'
import styles from './ConversationPickerModal.module.css'
import iconSearch from '@/assets/icon/search.svg'

const MAX_PICK = 30

type Props = {
  open: boolean
  excludeId?: string
  sending?: boolean
  hint?: string
  onCancel: () => void
  onConfirm: (conversationIDs: string[]) => void
}

export function ConversationPickerModal({
  open,
  excludeId,
  sending = false,
  hint = '',
  onCancel,
  onConfirm,
}: Props) {
  const { conversations } = useAppStore()
  const [q, setQ] = useState('')
  const [filter, setFilter] = useState<'all' | 'group' | 'friend'>('all')
  const [selected, setSelected] = useState<Set<string>>(new Set())

  useEffect(() => {
    if (!open) return
    setQ('')
    setFilter('all')
    setSelected(new Set())
  }, [open])

  const list = useMemo(() => {
    const key = q.trim().toLowerCase()
    return conversations.filter((c) => {
      if (isHiddenConversation(c)) return false
      if (excludeId && c.conversationID === excludeId) return false
      const isGroup = isGroupConversation(c.conversationID)
      if (filter === 'group' && !isGroup) return false
      if (filter === 'friend' && isGroup) return false
      if (!key) return true
      return convTitle(c).toLowerCase().includes(key)
    })
  }, [conversations, q, filter, excludeId])

  const allVisibleSelected =
    list.length > 0 && list.every((c) => selected.has(c.conversationID))

  if (!open) return null

  function toggle(id: string) {
    if (!selected.has(id) && selected.size >= MAX_PICK) {
      alert(`最多选择 ${MAX_PICK} 个会话`)
      return
    }
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  function toggleSelectAll() {
    if (allVisibleSelected) {
      setSelected((prev) => {
        const next = new Set(prev)
        list.forEach((c) => next.delete(c.conversationID))
        return next
      })
      return
    }
    const next = new Set(selected)
    let hitLimit = false
    for (const c of list) {
      if (next.size >= MAX_PICK) {
        hitLimit = true
        break
      }
      next.add(c.conversationID)
    }
    setSelected(next)
    if (hitLimit) alert(`最多选择 ${MAX_PICK} 个会话`)
  }

  return (
    <div className={styles.mask}>
      <div className={styles.card}>
        <header className={styles.head}>
          <button type="button" className={styles.iconBtn} onClick={onCancel} disabled={sending}>
            ×
          </button>
          <strong>选择会话</strong>
          <button
            type="button"
            className={styles.send}
            disabled={!selected.size || sending}
            onClick={() => onConfirm([...selected])}
          >
            {sending ? '发送中…' : `发送(${selected.size})`}
          </button>
        </header>
        {hint ? <p className={styles.hint}>{hint}</p> : null}
        <div className={styles.tabs}>
          {(
            [
              ['all', '全部'],
              ['group', '群聊'],
              ['friend', '好友'],
            ] as const
          ).map(([k, label]) => (
            <button
              key={k}
              type="button"
              className={filter === k ? styles.tabOn : ''}
              onClick={() => setFilter(k)}
            >
              {label}
            </button>
          ))}
        </div>
        <label className={styles.search}>
          <img src={iconSearch} alt="" />
          <input
            type="search"
            value={q}
            onChange={(e) => setQ(e.target.value)}
            placeholder="搜索会话"
          />
        </label>
        <ul className={styles.list}>
          {list.map((c) => {
            const id = c.conversationID
            const on = selected.has(id)
            const title = convTitle(c)
            const avatar = convAvatar(c)
            const isGroup = isGroupConversation(id)
            return (
              <li key={id}>
                <button type="button" className={styles.row} onClick={() => toggle(id)}>
                  <i className={on ? styles.checkOn : styles.check} />
                  <span className={styles.avatar}>
                    {avatar ? <img src={avatar} alt="" /> : title.slice(0, 1)}
                  </span>
                  <span className={styles.meta}>
                    <b>{title}</b>
                    <em>{isGroup ? '群聊' : '好友'}</em>
                  </span>
                </button>
              </li>
            )
          })}
          {!list.length ? <li className={styles.empty}>无会话</li> : null}
        </ul>
        <footer className={styles.foot}>
          <button
            type="button"
            className={styles.selectAll}
            disabled={!list.length || sending}
            onClick={toggleSelectAll}
          >
            <i className={allVisibleSelected ? styles.checkOn : styles.check} />
            {allVisibleSelected ? '取消全选' : '全选'}
          </button>
          <span className={styles.cap}>最多 {MAX_PICK} 个</span>
        </footer>
      </div>
    </div>
  )
}
