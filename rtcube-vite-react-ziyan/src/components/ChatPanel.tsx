import { useEffect, useMemo, useRef, useState } from 'react'
import { muteAllFromGroupTip, onGroupLiveChange } from '@/im/groupLive'
import { applyMuteAll, fetchSpeakBlock, type SpeakBlock } from '@/im/speakGuard'
import { isCenteredTipMessage } from '@/im/messageTips'
import type { ChatMessage } from '@/im/types'
import { APP_NAME } from '@/config/runtime'
import { useAppStore } from '@/store/AppStore'
import { useCallSession } from '@/store/CallSession'
import {
  convAvatar,
  convTitle,
  groupIdFromConversation,
  isGroupConversation,
  peerIdFromConversation,
} from '@/utils/conversation'
import { ChatSearchPanel } from './ChatSearchPanel'
import { MessageInput } from './MessageInput'
import { MessageList } from './MessageList'
import { PinnedMessageBar } from './PinnedMessageBar'
import styles from './ChatPanel.module.css'
import iconMore from '@/assets/icon/more.svg'
import iconShousuoNor from '@/assets/images/shousuo_nor.png'
import iconShousuoSel from '@/assets/images/shousuo_sel.png'

type Props = {
  rightOpen: boolean
  onToggleRight: () => void
  onOpenGroupDetail: () => void
  onOpenUserProfile: (userId: string, nickname?: string, avatar?: string) => void
}

export function ChatPanel({
  rightOpen,
  onToggleRight,
  onOpenGroupDetail,
  onOpenUserProfile,
}: Props) {
  const { activeId, conversations, messages, session } = useAppStore()
  const call = useCallSession()
  const [isGroupAdmin, setIsGroupAdmin] = useState(false)
  const [speak, setSpeak] = useState<SpeakBlock>({
    blocked: false,
    reason: '',
    muteAll: false,
    personalMuted: false,
    isManager: false,
  })
  const [quoteMsg, setQuoteMsg] = useState<ChatMessage | null>(null)
  const [multiSelecting, setMultiSelecting] = useState(false)
  const [searchOpen, setSearchOpen] = useState(false)
  const [highlightId, setHighlightId] = useState('')
  const [pinTick, setPinTick] = useState(0)
  const muteAllHold = useRef<{ value: boolean; at: number } | null>(null)

  function jumpToMessage(id: string) {
    if (!id) return
    setHighlightId(id)
    window.setTimeout(() => setHighlightId(''), 2200)
  }

  useEffect(() => {
    setQuoteMsg(null)
    setMultiSelecting(false)
    setSearchOpen(false)
    setHighlightId('')
    muteAllHold.current = null
    setSpeak({
      blocked: false,
      reason: '',
      muteAll: false,
      personalMuted: false,
      isManager: false,
    })
  }, [activeId])

  const active = useMemo(
    () => conversations.find((c) => c.conversationID === activeId),
    [conversations, activeId],
  )
  const title = active ? convTitle(active) : APP_NAME
  const isGroup = activeId ? isGroupConversation(activeId) : false
  const lastTipId = useMemo(() => {
    const last = messages[messages.length - 1]
    if (!last || !isCenteredTipMessage(last)) return ''
    return last.ID
  }, [messages])

  useEffect(() => {
    if (!isGroup || !activeId || !session?.userId) {
      setIsGroupAdmin(false)
      setSpeak({
        blocked: false,
        reason: '',
        muteAll: false,
        personalMuted: false,
        isManager: false,
      })
      return
    }
    const gid = groupIdFromConversation(activeId)
    let cancelled = false
    ;(async () => {
      try {
        const block = await fetchSpeakBlock(gid, session.userId)
        if (cancelled) return
        const hold = muteAllHold.current
        const next =
          hold && Date.now() - hold.at < 15000 ? applyMuteAll(block, hold.value) : block
        setSpeak(next)
        setIsGroupAdmin(block.isManager)
        if (block.blocked) {
          setQuoteMsg(null)
          setMultiSelecting(false)
        }
      } catch {
        if (!cancelled) {
          setIsGroupAdmin(false)
        }
      }
    })()
    return () => {
      cancelled = true
    }
  }, [isGroup, activeId, session?.userId, lastTipId])

  useEffect(() => {
    if (!lastTipId) return
    const last = messages[messages.length - 1]
    const flag = muteAllFromGroupTip(last)
    if (flag === undefined) return
    muteAllHold.current = { value: flag, at: Date.now() }
    setSpeak((prev) => applyMuteAll(prev, flag))
  }, [lastTipId, messages])

  useEffect(() => {
    if (!isGroup || !activeId || !session?.userId) return
    const gid = groupIdFromConversation(activeId)
    const selfId = session.userId
    let timer = 0
    const refresh = () => {
      window.clearTimeout(timer)
      timer = window.setTimeout(() => {
        void fetchSpeakBlock(gid, selfId).then((block) => {
          const hold = muteAllHold.current
          if (hold && block.muteAll === hold.value) muteAllHold.current = null
          const next =
            hold && Date.now() - hold.at < 15000 ? applyMuteAll(block, hold.value) : block
          setSpeak(next)
          setIsGroupAdmin(block.isManager)
          if (next.blocked) {
            setQuoteMsg(null)
            setMultiSelecting(false)
          }
        })
      }, 160)
    }
    const off = onGroupLiveChange((e) => {
      if (e.groupID !== gid) return
      if (e.muteAllMembers !== undefined) {
        muteAllHold.current = { value: e.muteAllMembers, at: Date.now() }
        setSpeak((prev) => applyMuteAll(prev, e.muteAllMembers as boolean))
      }
      refresh()
    })
    return () => {
      window.clearTimeout(timer)
      off()
    }
  }, [isGroup, activeId, session?.userId])

  async function startCall(mediaType: 1 | 2) {
    if (!activeId || isGroup) return
    const peer = peerIdFromConversation(activeId)
    await call.startCall({ peerId: peer, peerName: title, mediaType })
  }

  function openHeaderProfile() {
    if (!activeId) return
    if (isGroup) {
      onOpenGroupDetail()
      return
    }
    const peer = peerIdFromConversation(activeId)
    onOpenUserProfile(peer, title, convAvatar(active!))
  }

  return (
    <section className={styles.pane}>
      <header className={styles.head}>
        <button type="button" className={styles.titleBtn} onClick={openHeaderProfile}>
          <strong>{activeId ? title : '未选择会话'}</strong>
        </button>
        <div className={styles.headActions}>
          {activeId ? (
            <>
              {isGroup ? (
                <button
                  type="button"
                  className={styles.iconBtn}
                  aria-label={rightOpen ? '收起右栏' : '成员栏'}
                  title={rightOpen ? '收起右栏' : '成员栏'}
                  onClick={onToggleRight}
                >
                  <img
                    className={styles.shousuo}
                    src={rightOpen ? iconShousuoSel : iconShousuoNor}
                    alt=""
                  />
                </button>
              ) : null}
              <button
                type="button"
                className={styles.iconBtn}
                aria-label={isGroup ? '详情' : '资料'}
                title={isGroup ? '群详情' : '资料'}
                onClick={openHeaderProfile}
              >
                <img src={iconMore} alt="" />
              </button>
            </>
          ) : null}
        </div>
      </header>
      {activeId ? (
        <>
          <PinnedMessageBar
            conversationID={activeId}
            isGroupAdmin={isGroupAdmin}
            refreshTick={pinTick}
            onJump={jumpToMessage}
          />
          <MessageList
            onOpenUserProfile={onOpenUserProfile}
            isGroupAdmin={isGroupAdmin}
            quoteMsg={quoteMsg}
            onQuote={setQuoteMsg}
            onMultiChange={setMultiSelecting}
            highlightId={highlightId}
            onPinned={() => setPinTick((n) => n + 1)}
            actionsLocked={speak.blocked}
          />
          {!multiSelecting ? (
            <MessageInput
              onCall={(t) => void startCall(t)}
              quoteMsg={quoteMsg}
              onClearQuote={() => setQuoteMsg(null)}
              onOpenSearch={() => setSearchOpen(true)}
              actionsLocked={speak.blocked}
              lockReason={speak.reason}
            />
          ) : null}
        </>
      ) : (
        <div className={styles.empty}>从左侧选择会话，或等待 IM 同步</div>
      )}

      {searchOpen && activeId ? (
        <ChatSearchPanel
          conversationID={activeId}
          localMessages={messages}
          onClose={() => setSearchOpen(false)}
          onJump={(id) => {
            setSearchOpen(false)
            jumpToMessage(id)
          }}
        />
      ) : null}

    </section>
  )
}
