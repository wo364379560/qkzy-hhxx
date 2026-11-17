import { useState } from 'react'
import { ConversationPickerModal } from './ConversationPickerModal'
import { loadFavorites, removeFavorite, type FavoriteItem } from '@/services/favorites'
import { forwardFavorite } from '@/services/favoriteForward'
import styles from './FavoritesPanel.module.css'

export function FavoritesPanel() {
  const [list, setList] = useState<FavoriteItem[]>(() => loadFavorites())
  const [pickItem, setPickItem] = useState<FavoriteItem | null>(null)
  const [sending, setSending] = useState(false)

  function onRemove(id: string) {
    setList(removeFavorite(id))
  }

  async function onConfirm(ids: string[]) {
    if (!pickItem) return
    setSending(true)
    try {
      await forwardFavorite(pickItem, ids)
      setPickItem(null)
    } catch (e) {
      alert(e instanceof Error ? e.message : '转发失败')
    } finally {
      setSending(false)
    }
  }

  return (
    <div className={styles.pane}>
      <h3>收藏</h3>
      <p className={styles.desc}>长按消息可收藏，可转发到好友或群聊</p>
      <ul className={styles.list}>
        {list.map((item) => (
          <li key={item.id} className={styles.card}>
            <div className={styles.meta}>
              {item.from} · {new Date(item.createdAt).toLocaleString()}
            </div>
            <div className={styles.body}>{item.text || '[消息]'}</div>
            <div className={styles.acts}>
              <button type="button" className={styles.forward} onClick={() => setPickItem(item)}>
                转发
              </button>
              <button type="button" className={styles.remove} onClick={() => onRemove(item.id)}>
                取消收藏
              </button>
            </div>
          </li>
        ))}
      </ul>
      {!list.length ? <p className={styles.desc}>还没有收藏</p> : null}
      <ConversationPickerModal
        open={Boolean(pickItem)}
        sending={sending}
        hint="转发收藏"
        onCancel={() => {
          if (sending) return
          setPickItem(null)
        }}
        onConfirm={(ids) => void onConfirm(ids)}
      />
    </div>
  )
}
