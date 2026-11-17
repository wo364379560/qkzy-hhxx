import { useEffect, useState } from 'react'
import { submitReport } from '@/api/report'
import { emitAtMember } from '@/im/atMemberEvent'
import { addFriend, checkIsFriend, formatFriendAddError } from '@/im/friends'
import { hideSpeaker } from '@/im/hiddenSpeakers'
import { useDetailUi } from '@/store/DetailUi'
import styles from './ChatAvatarContextMenu.module.css'

export type AvatarMenuTarget = {
  x: number
  y: number
  userId: string
  nick: string
  avatar: string
}

type Props = {
  menu: AvatarMenuTarget | null
  conversationID: string
  groupId?: string
  selfId: string
  onClose: () => void
  onHidden?: () => void
}

export function ChatAvatarContextMenu({
  menu,
  conversationID,
  groupId,
  selfId,
  onClose,
  onHidden,
}: Props) {
  const { openUserProfile, navigateToChat } = useDetailUi()
  const [busy, setBusy] = useState(false)
  const [isFriend, setIsFriend] = useState(true)

  useEffect(() => {
    if (!menu || menu.userId === selfId) {
      setIsFriend(true)
      return
    }
    let cancelled = false
    void checkIsFriend(menu.userId)
      .then((v) => {
        if (!cancelled) setIsFriend(v)
      })
      .catch(() => {
        if (!cancelled) setIsFriend(false)
      })
    return () => {
      cancelled = true
    }
  }, [menu, selfId])

  useEffect(() => {
    if (!menu) return
    const close = () => onClose()
    const t = window.setTimeout(() => {
      window.addEventListener('click', close)
      window.addEventListener('scroll', close, true)
    }, 0)
    return () => {
      window.clearTimeout(t)
      window.removeEventListener('click', close)
      window.removeEventListener('scroll', close, true)
    }
  }, [menu, onClose])

  if (!menu) return null

  const isSelf = menu.userId === selfId
  const inGroup = Boolean(groupId)
  const left = Math.min(Math.max(8, menu.x), window.innerWidth - 200)
  const top = Math.min(Math.max(8, menu.y), window.innerHeight - 280)

  async function run(fn: () => Promise<void>) {
    if (busy) return
    setBusy(true)
    try {
      await fn()
      onClose()
    } catch (e) {
      alert(e instanceof Error ? e.message : '操作失败')
    } finally {
      setBusy(false)
    }
  }

  return (
    <div
      className={styles.menu}
      style={{ left, top }}
      onClick={(e) => e.stopPropagation()}
      onContextMenu={(e) => {
        e.preventDefault()
        e.stopPropagation()
      }}
    >
      {!isSelf ? (
        <button
          type="button"
          onClick={() => {
            onClose()
            navigateToChat(`C2C${menu.userId}`)
          }}
        >
          <i className={styles.icBubble} />
          <span>发送消息</span>
        </button>
      ) : null}
      {!isSelf && inGroup ? (
        <button
          type="button"
          onClick={() => {
            emitAtMember({
              groupId: groupId || '',
              userID: menu.userId,
              nick: menu.nick || menu.userId,
            })
            onClose()
            navigateToChat(conversationID)
          }}
        >
          <i className={styles.icAt} />
          <span>@ TA</span>
        </button>
      ) : null}
      <button
        type="button"
        onClick={() => {
          onClose()
          openUserProfile({
            userId: menu.userId,
            nickname: menu.nick,
            avatar: menu.avatar,
          })
        }}
      >
        <i className={styles.icCard} />
        <span>查看资料</span>
      </button>
      {!isSelf && !isFriend ? (
        <button
          type="button"
          disabled={busy}
          onClick={() =>
            void run(async () => {
              try {
                await addFriend(menu.userId)
                alert('好友申请已发送')
              } catch (e) {
                throw new Error(formatFriendAddError(e))
              }
            })
          }
        >
          <i className={styles.icAdd} />
          <span>添加好友</span>
        </button>
      ) : null}
      {!isSelf ? (
        <button
          type="button"
          disabled={busy}
          onClick={() =>
            void run(async () => {
              await submitReport({
                targetId: menu.userId,
                targetType: 'user',
                targetName: menu.nick || menu.userId,
                category: 'other',
                description: `会话 ${conversationID} 内举报`,
              })
              alert('已提交举报')
            })
          }
        >
          <i className={styles.icWarn} />
          <span>举报</span>
        </button>
      ) : null}
      {!isSelf ? (
        <button
          type="button"
          disabled={busy}
          onClick={() => {
            hideSpeaker(conversationID, menu.userId)
            onHidden?.()
            onClose()
          }}
        >
          <i className={styles.icHide} />
          <span>屏蔽此人发言</span>
        </button>
      ) : null}
    </div>
  )
}
