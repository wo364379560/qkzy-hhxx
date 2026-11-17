import { useCallback, useEffect, useState } from 'react'
import {
  fetchPinnedGroupMessages,
  onPinnedGroupMessageUpdated,
  pinGroupMessage,
  toPinnedItems,
  type PinnedItem,
} from '@/im/groupPin'
import { groupIdFromConversation, isGroupConversation } from '@/utils/conversation'
import styles from './PinnedMessageBar.module.css'

type Props = {
  conversationID: string
  isGroupAdmin?: boolean
  refreshTick?: number
  onJump: (messageId: string) => void
}

export function PinnedMessageBar({
  conversationID,
  isGroupAdmin = false,
  refreshTick = 0,
  onJump,
}: Props) {
  const [items, setItems] = useState<PinnedItem[]>([])
  const [index, setIndex] = useState(0)
  const [open, setOpen] = useState(false)

  const isGroup = isGroupConversation(conversationID)
  const groupID = isGroup ? groupIdFromConversation(conversationID) : ''

  const load = useCallback(async () => {
    if (!isGroup || !groupID) {
      setItems([])
      return
    }
    const list = toPinnedItems(await fetchPinnedGroupMessages(groupID))
    setItems(list)
    setIndex((i) => (list.length ? Math.min(i, list.length - 1) : 0))
  }, [isGroup, groupID])

  useEffect(() => {
    setOpen(false)
    setIndex(0)
    void load()
  }, [conversationID, load, refreshTick])

  useEffect(() => {
    if (!isGroup || !groupID) return
    return onPinnedGroupMessageUpdated((gid) => {
      if (!gid || gid === groupID) void load()
    })
  }, [isGroup, groupID, load])

  if (!items.length) return null

  const current = items[index] || items[0]
  const n = items.length

  function cycleAndJump() {
    if (!current) return
    onJump(current.id)
    if (n > 1) setIndex((i) => (i + 1) % n)
  }

  async function unpin(item: PinnedItem) {
    if (!isGroupAdmin || !groupID) return
    try {
      await pinGroupMessage(groupID, item.raw, false)
      await load()
    } catch (e) {
      alert(e instanceof Error ? e.message : '取消置顶失败')
    }
  }

  return (
    <div className={styles.wrap}>
      <div className={styles.bar}>
        <button type="button" className={styles.barMain} onClick={cycleAndJump}>
          <span className={styles.mark} />
          <span className={styles.text}>
            置顶 {index + 1}/{n} {current.nick ? `${current.nick}: ` : ''}
            {current.text}
          </span>
        </button>
        <button
          type="button"
          className={styles.caretBtn}
          aria-label={open ? '收起置顶' : '展开置顶'}
          onClick={() => setOpen((v) => !v)}
        >
          <span className={styles.caret} />
        </button>
      </div>

      {open ? (
        <div className={styles.panel}>
          <div className={styles.panelHead}>全部置顶 ({n})</div>
          <ul className={styles.list}>
            {items.map((item) => (
              <li key={item.id} className={styles.row}>
                <button
                  type="button"
                  className={styles.rowMain}
                  onClick={() => {
                    const i = items.findIndex((x) => x.id === item.id)
                    if (i >= 0) setIndex(i)
                    onJump(item.id)
                  }}
                >
                  <b>{item.nick || '未知'}</b>
                  <span>{item.text}</span>
                </button>
                {isGroupAdmin ? (
                  <button
                    type="button"
                    className={styles.remove}
                    onClick={() => void unpin(item)}
                  >
                    移除
                  </button>
                ) : null}
              </li>
            ))}
          </ul>
          <button type="button" className={styles.fold} onClick={() => setOpen(false)}>
            收起
            <i className={`${styles.caret} ${styles.caretOpen}`} />
          </button>
        </div>
      ) : null}
    </div>
  )
}
