import { useEffect, useMemo, useState, type MouseEvent } from 'react'
import { AddFriendModal } from '@/components/AddFriendModal'
import { CreateGroupModal } from '@/components/CreateGroupModal'
import { fetchUserOnlineMap, onUserStatusUpdated } from '@/im/userStatus'
import { isHiddenConversation } from '@/im/ImClient'
import {
  deleteConversation,
  pinConversation,
  setConversationMute,
} from '@/im/profileActions'
import { APP_NAME } from '@/config/runtime'
import { useAppStore } from '@/store/AppStore'
import type { ConversationItem } from '@/im/types'
import { convAvatar, convAtPreview, convLastTime, convPreview, convTitle, formatConvTime } from '@/utils/conversation'
import styles from './ConversationList.module.css'

type Props = {
  onOpenProfile?: (userId: string, nickname?: string, avatar?: string) => void
  onOpenGroupDetail?: (groupId: string) => void
}

export function ConversationList({ onOpenProfile, onOpenGroupDetail }: Props) {
  const {
    conversations,
    activeId,
    selectConversation,
    session,
    ready,
    connecting,
    connectError,
    refreshConversations,
    patchConversation,
  } = useAppStore()
  const [q, setQ] = useState('')
  const [plusOpen, setPlusOpen] = useState(false)
  const [createOpen, setCreateOpen] = useState(false)
  const [addOpen, setAddOpen] = useState(false)
  const [onlineMap, setOnlineMap] = useState<Record<string, boolean>>({})
  const [menu, setMenu] = useState<{ x: number; y: number; conv: ConversationItem } | null>(null)

  const filtered = useMemo(() => {
    const visible = conversations.filter((c) => !isHiddenConversation(c))
    const key = q.trim().toLowerCase()
    if (!key) return visible
    return visible.filter((c) => convTitle(c).toLowerCase().includes(key))
  }, [conversations, q])

  const totalUnread = useMemo(
    () => conversations.reduce((sum, c) => sum + (c.unreadCount || 0), 0),
    [conversations],
  )

  const peerIds = useMemo(
    () =>
      conversations
        .filter((c) => c.conversationID.startsWith('C2C'))
        .map((c) => c.userID || c.conversationID.replace(/^C2C/, ''))
        .filter(Boolean),
    [conversations],
  )

  useEffect(() => {
    if (!ready || !peerIds.length) return
    let cancelled = false
    let off = () => {}
    ;(async () => {
      const map = await fetchUserOnlineMap(peerIds)
      if (!cancelled) setOnlineMap((prev) => ({ ...prev, ...map }))
      off = onUserStatusUpdated((patch) => {
        setOnlineMap((prev) => ({ ...prev, ...patch }))
      })
    })()
    return () => {
      cancelled = true
      off()
    }
  }, [ready, peerIds.join(',')])

  const statusText = ready ? '已连接' : connecting ? '连接中…' : connectError || '未连接'

  function onItemContext(e: MouseEvent, conv: ConversationItem) {
    e.preventDefault()
    e.stopPropagation()
    setPlusOpen(false)
    const w = 168
    const h = 168
    setMenu({
      x: Math.min(e.clientX, window.innerWidth - w - 8),
      y: Math.min(e.clientY, window.innerHeight - h - 8),
      conv,
    })
  }

  async function runMenu(label: string, task: () => Promise<void>) {
    const conv = menu?.conv
    setMenu(null)
    if (!conv) return
    try {
      await task()
      await refreshConversations()
    } catch (err) {
      alert(err instanceof Error ? err.message : `${label}失败`)
    }
  }

  return (
    <aside className={styles.pane}>
      <header className={styles.head}>
        <div className={styles.headMain}>
          <div className={styles.nameRow}>
            <strong>{session?.nickname || session?.userId || APP_NAME}</strong>
            <i
              className={`${styles.selfDot} ${ready ? styles.online : styles.offline}`}
              title={statusText}
            />
            <em className={styles.connText}>{statusText}</em>
          </div>
          <span className={styles.sub}>
            {totalUnread > 0 ? `${totalUnread} 未读` : '会话列表'}
          </span>
        </div>
        <div className={styles.headBtns}>
          <button
            type="button"
            className={styles.plus}
            title="添加"
            onClick={() => setPlusOpen((v) => !v)}
          >
            +
          </button>
          {plusOpen ? (
            <div className={styles.plusMenu}>
              <button
                type="button"
                onClick={() => {
                  setPlusOpen(false)
                  setAddOpen(true)
                }}
              >
                加好友
              </button>
              <button
                type="button"
                onClick={() => {
                  setPlusOpen(false)
                  setCreateOpen(true)
                }}
              >
                建群
              </button>
            </div>
          ) : null}
        </div>
      </header>

      <div className={styles.searchWrap}>
        <input
          className={styles.search}
          value={q}
          onChange={(e) => setQ(e.target.value)}
          placeholder="搜索会话"
        />
      </div>

      <div
        className={styles.list}
        onClick={() => {
          setPlusOpen(false)
          setMenu(null)
        }}
      >
        {filtered.map((c) => {
          const title = convTitle(c)
          const avatar = convAvatar(c)
          const active = c.conversationID === activeId
          const isC2C = c.conversationID.startsWith('C2C')
          const peerId = isC2C ? c.userID || c.conversationID.replace(/^C2C/, '') : ''
          const online = peerId ? Boolean(onlineMap[peerId]) : false
          const muted = Boolean(c.isMuted)
          const showUnread = c.unreadCount > 0 && !muted
          const showMuteDot = muted && c.unreadCount > 0
          const draft = !active ? String(c.draftText || '').trim() : ''
          const atInfo = convAtPreview(c, {
            userId: session?.userId || '',
            nick: session?.nickname || '',
          })
          const timeText = formatConvTime(convLastTime(c))
          const preview = draft || atInfo?.text || convPreview(c)

          return (
            <div
              key={c.conversationID}
              className={`${styles.item} ${c.isPinned ? styles.pinned : ''} ${active ? styles.active : ''}`}
              onContextMenu={(e) => onItemContext(e, c)}
            >
              <button
                type="button"
                className={styles.avatarWrap}
                title={isC2C ? '查看个人详情' : undefined}
                onClick={(e) => {
                  e.stopPropagation()
                  if (isC2C && peerId) {
                    onOpenProfile?.(peerId, title, avatar)
                    return
                  }
                  void selectConversation(c.conversationID)
                }}
              >
                <span className={styles.avatar}>
                  {avatar ? <img src={avatar} alt="" /> : title.slice(0, 1)}
                </span>
                {isC2C ? (
                  <i
                    className={`${styles.presence} ${online ? styles.online : styles.offline}`}
                    title={online ? '在线' : '离线'}
                  />
                ) : null}
                {showUnread ? (
                  <i className={styles.avatarBadge}>
                    {c.unreadCount > 99 ? '99+' : c.unreadCount}
                  </i>
                ) : null}
                {showMuteDot ? <i className={styles.muteUnreadDot} title="免打扰有消息" /> : null}
              </button>
              <button
                type="button"
                className={styles.metaBtn}
                onClick={() => void selectConversation(c.conversationID)}
              >
                <div className={styles.row}>
                  <span className={styles.title}>{title}</span>
                  <span className={styles.time}>{timeText}</span>
                </div>
                <div className={styles.previewRow}>
                  <span className={styles.preview}>
                    {draft ? <em className={styles.draftTag}>[草稿]</em> : null}
                    {!draft && atInfo ? <em className={styles.atTag}>{atInfo.tag}</em> : null}
                    {preview}
                  </span>
                  {muted ? (
                    <span className={styles.muteMark} title="消息免打扰">
                      🔕
                    </span>
                  ) : null}
                </div>
              </button>
            </div>
          )
        })}
        {!filtered.length ? <div className={styles.empty}>暂无会话</div> : null}
      </div>

      {menu ? (
        <div className={styles.ctxMask} onClick={() => setMenu(null)} onContextMenu={(e) => e.preventDefault()}>
          <div className={styles.ctxMenu} style={{ left: menu.x, top: menu.y }} onClick={(e) => e.stopPropagation()}>
            <button
              type="button"
              onClick={() =>
                void runMenu('置顶', async () => {
                  const next = !menu.conv.isPinned
                  await pinConversation(menu.conv.conversationID, next)
                  patchConversation(menu.conv.conversationID, { isPinned: next })
                })
              }
            >
              {menu.conv.isPinned ? '取消置顶' : '置顶'}
            </button>
            <button
              type="button"
              onClick={() =>
                void runMenu('免打扰', async () => {
                  const next = !menu.conv.isMuted
                  await setConversationMute(menu.conv.conversationID, next)
                  patchConversation(menu.conv.conversationID, { isMuted: next })
                })
              }
            >
              {menu.conv.isMuted ? '关闭免打扰' : '消息免打扰'}
            </button>
            <button
              type="button"
              className={styles.ctxDanger}
              onClick={() =>
                void runMenu('删除', async () => {
                  if (!window.confirm(`从消息列表移除「${convTitle(menu.conv)}」？好友/群还在通讯录里。`)) {
                    return
                  }
                  await deleteConversation(menu.conv.conversationID)
                  if (activeId === menu.conv.conversationID) selectConversation('')
                })
              }
            >
              删除
            </button>
          </div>
        </div>
      ) : null}

      <CreateGroupModal
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        onSuccess={(conversationID) => {
          void refreshConversations().then(() => selectConversation(conversationID))
        }}
      />
      <AddFriendModal
        open={addOpen}
        onClose={() => setAddOpen(false)}
        onOpenProfile={(userId, nickname, avatar) => onOpenProfile?.(userId, nickname, avatar)}
        onOpenGroup={(groupId) => {
          if (onOpenGroupDetail) onOpenGroupDetail(groupId)
          else void selectConversation(`GROUP${groupId}`)
        }}
      />
    </aside>
  )
}
