import { useEffect, useMemo, useState, type CSSProperties } from 'react'
import { Navigate } from 'react-router-dom'
import { CallOverlay } from '@/components/CallOverlay'
import { ChatPanel } from '@/components/ChatPanel'
import { ContactsPanel } from '@/components/ContactsPanel'
import { ConversationList } from '@/components/ConversationList'
import { GroupRightPanel } from '@/components/GroupRightPanel'
import { DiscoverPanel } from '@/components/DiscoverPanel'
import { FavoritesPanel } from '@/components/FavoritesPanel'
import { FilesPanel } from '@/components/FilesPanel'
import { MinePanel } from '@/components/MinePanel'
import { ShortcutsPanel } from '@/components/ShortcutsPanel'
import { SideNav, type MainTab } from '@/components/SideNav'
import { useI18n } from '@/i18n/I18nProvider'
import { useAppStore } from '@/store/AppStore'
import { CallSessionProvider, useCallSession } from '@/store/CallSession'
import { DetailUiProvider, useDetailUi } from '@/store/DetailUi'
import {
  groupIdFromConversation,
  isGroupConversation,
} from '@/utils/conversation'
import styles from './ChatShell.module.css'

function ShellBody() {
  const { selectConversation } = useAppStore()
  const [tab, setTab] = useState<MainTab>('messages')

  function navigateToChat(conversationID: string) {
    void selectConversation(conversationID)
    setTab('messages')
  }

  return (
    <DetailUiProvider onNavigateToChat={navigateToChat}>
      <ShellLayout tab={tab} setTab={setTab} />
      <GlobalCallFloat />
    </DetailUiProvider>
  )
}

function ShellLayout({
  tab,
  setTab,
}: {
  tab: MainTab
  setTab: (tab: MainTab) => void
}) {
  const { t } = useI18n()
  const { activeId, ready, connecting, connectError, reconnect, selectConversation } =
    useAppStore()
  const { openGroupDetail, openUserProfile } = useDetailUi()
  const [rightOpen, setRightOpen] = useState(true)
  const [leftW, setLeftW] = useState(280)
  const [rightW, setRightW] = useState(280)

  const isGroup = useMemo(
    () => (activeId ? isGroupConversation(activeId) : false),
    [activeId],
  )
  const groupId = useMemo(
    () => (isGroup ? groupIdFromConversation(activeId) : ''),
    [activeId, isGroup],
  )

  useEffect(() => {
    if (isGroup) setRightOpen(true)
  }, [isGroup, groupId])

  const showChat = tab === 'messages'
  const showFiles = tab === 'files'
  const showRight = showChat && isGroup && rightOpen

  const gridStyle: CSSProperties = showChat
    ? showRight
      ? { gridTemplateColumns: `68px ${leftW}px 4px minmax(0, 1fr) 4px ${rightW}px` }
      : { gridTemplateColumns: `68px ${leftW}px 4px minmax(0, 1fr)` }
    : showFiles
      ? { gridTemplateColumns: '68px minmax(0, 1fr)' }
      : { gridTemplateColumns: `68px minmax(240px, 320px) minmax(0, 1fr)` }

  function openFriendChat(userId: string) {
    void selectConversation(`C2C${userId}`)
    setTab('messages')
  }

  function openGroupChat(groupId: string) {
    void selectConversation(`GROUP${groupId}`)
    setTab('messages')
  }

  return (
    <div className={styles.shell} style={gridStyle}>
      <SideNav active={tab} onChange={setTab} />

      {tab === 'messages' ? (
        <>
          <div className={styles.left}>
            {!ready ? (
              <div className={styles.connectBanner}>
                <span>{connecting ? t('connecting') : connectError || t('connecting')}</span>
                {connectError ? (
                  <button type="button" onClick={() => void reconnect()}>
                    {t('reconnect')}
                  </button>
                ) : null}
              </div>
            ) : null}
            <ConversationList
              onOpenProfile={(userId, nickname, avatar) =>
                openUserProfile({ userId, nickname, avatar })
              }
              onOpenGroupDetail={(groupId) => openGroupDetail(groupId)}
            />
          </div>
          <div
            className={styles.handle}
            onMouseDown={(e) => {
              e.preventDefault()
              const startX = e.clientX
              const startW = leftW
              const onMove = (ev: MouseEvent) => {
                setLeftW(Math.min(420, Math.max(220, startW + ev.clientX - startX)))
              }
              const onUp = () => {
                window.removeEventListener('mousemove', onMove)
                window.removeEventListener('mouseup', onUp)
              }
              window.addEventListener('mousemove', onMove)
              window.addEventListener('mouseup', onUp)
            }}
          />
          <div className={styles.center}>
            <ChatPanel
              rightOpen={rightOpen}
              onToggleRight={() => setRightOpen((v) => !v)}
              onOpenGroupDetail={() => groupId && openGroupDetail(groupId)}
              onOpenUserProfile={(userId, nickname, avatar) =>
                openUserProfile({ userId, nickname, avatar })
              }
            />
          </div>
          {showRight ? (
            <>
              <div
                className={styles.handle}
                onMouseDown={(e) => {
                  e.preventDefault()
                  const startX = e.clientX
                  const startW = rightW
                  const onMove = (ev: MouseEvent) => {
                    setRightW(Math.min(420, Math.max(240, startW - (ev.clientX - startX))))
                  }
                  const onUp = () => {
                    window.removeEventListener('mousemove', onMove)
                    window.removeEventListener('mouseup', onUp)
                  }
                  window.addEventListener('mousemove', onMove)
                  window.addEventListener('mouseup', onUp)
                }}
              />
              <div className={styles.right}>
                <GroupRightPanel
                  groupId={groupId}
                  onOpenSettings={() => openGroupDetail(groupId)}
                  onOpenMember={(m, restricted) =>
                    openUserProfile({
                      userId: m.userID,
                      nickname: m.nick,
                      avatar: m.avatar,
                      restricted,
                    })
                  }
                />
              </div>
            </>
          ) : null}
        </>
      ) : showFiles ? (
        <div className={styles.filesFill}>
          <FilesPanel
            onOpenChat={(id) => {
              void selectConversation(id)
              setTab('messages')
            }}
          />
        </div>
      ) : (
        <>
          <div className={styles.left}>
            {tab === 'contacts' ? (
              <ContactsPanel
                onOpenChat={openFriendChat}
                onOpenGroup={openGroupChat}
                onOpenProfile={(userId, nickname, avatar) =>
                  openUserProfile({ userId, nickname, avatar })
                }
              />
            ) : tab === 'mine' ? (
              <MinePanel />
            ) : tab === 'shortcuts' ? (
              <ShortcutsPanel onPick={() => setTab('messages')} />
            ) : tab === 'favorites' ? (
              <FavoritesPanel />
            ) : tab === 'discover' ? (
              <DiscoverPanel />
            ) : (
              <MinePanel />
            )}
          </div>
          <div className={styles.center}>
            <div className={styles.emptyMain}>从左侧继续，或切回「消息」</div>
          </div>
        </>
      )}
    </div>
  )
}

function GlobalCallFloat() {
  const call = useCallSession()
  const status =
    call.status === 'calling' ||
    call.status === 'incoming' ||
    call.status === 'connected' ||
    call.status === 'ended'
      ? call.status
      : 'ended'
  return (
    <CallOverlay
      open={call.open}
      mediaType={call.mediaType}
      peerName={call.peerName || call.peerId || '通话'}
      status={status}
      durationSec={call.durationSec}
      muted={call.muted}
      speakerOn={call.speakerOn}
      onAccept={() => void call.accept()}
      onReject={() => void call.reject()}
      onCancel={() => void call.cancel()}
      onHangup={() => void call.hangup()}
      onToggleMute={() => void call.toggleMute()}
      onToggleSpeaker={() => void call.toggleSpeaker()}
    />
  )
}

export function ChatShell() {
  const { session } = useAppStore()
  if (!session?.userId || !session.userSig) {
    return <Navigate to="/login" replace />
  }
  return (
    <CallSessionProvider>
      <ShellBody />
    </CallSessionProvider>
  )
}
