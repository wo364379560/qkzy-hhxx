import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from 'react'
import { MESSAGE_WINDOW_LIMIT } from '@/config/runtime'
import {
  clearLocalSession,
  getLocalSession,
  saveLocalSession,
  type LocalSession,
} from '@/api/session'
import {
  fetchFriendApplications,
  fetchGroupApplications,
} from '@/im/friends'
import {
  currentImUserId,
  fetchConversationList,
  fetchMessageList,
  imLogin,
  imLogout,
  isHiddenConversation,
  isImReady,
  onConversationListUpdated,
  onMessageReceived,
  onMessageRevoked,
  setMessageRead,
  trimMessageWindow,
} from '@/im/ImClient'
import { extractRtcFromMessage } from '@/im/rtcCall'
import { getConversationProfile } from '@/im/profileActions'
import {
  applyPeerReadUpdates,
  hydratePeerRead,
  isMessagePeerRead,
  onMessageReadByPeer,
  sendReadReceipts,
} from '@/im/readReceipt'
import { moveMediaCache } from '@/im/mediaCache'
import { isCenteredTipMessage } from '@/im/messageTips'
import { messageCopyText } from '@/im/messageActions'
import { findMessageContext } from '@/im/messageNavigation'
import { parseQuoteFromMessage } from '@/im/quote'
import { installAudioUnlock, playIncomingMessageSound } from '@/audio/sounds'
import { requestNotifyPermission, showAppNotification } from '@/audio/notify'
import type { ChatMessage, ConversationItem } from '@/im/types'

type AppStoreValue = {
  session: LocalSession | null
  ready: boolean
  connecting: boolean
  connectError: string
  conversations: ConversationItem[]
  activeId: string
  messages: ChatMessage[]
  loadingMessages: boolean
  loadingOlder: boolean
  historyCompleted: boolean
  viewingHistory: boolean
  locateMessage: (id: string) => Promise<boolean>
  /** 会话未读合计（消息 Tab 角标） */
  totalUnread: number
  /** 新朋友角标：个人申请 + 群申请 */
  contactBadge: number
  setSession: (s: LocalSession | null) => void
  selectConversation: (id: string) => void
  loadOlderMessages: () => Promise<boolean>
  appendOutgoing: (msg: ChatMessage) => void
  patchMessage: (id: string, patch: Partial<ChatMessage>) => void
  replaceOutgoing: (localId: string, sent: ChatMessage) => void
  removeMessage: (id: string) => void
  refreshConversations: () => Promise<void>
  patchConversation: (id: string, patch: Partial<ConversationItem>) => void
  refreshContactBadge: () => Promise<void>
  logout: () => Promise<void>
  bootFromSession: (s: LocalSession) => Promise<void>
  switchAccount: (s: LocalSession) => Promise<void>
  reconnect: () => Promise<void>
}

const AppStoreContext = createContext<AppStoreValue | null>(null)

export function AppStoreProvider({ children }: { children: ReactNode }) {
  const [session, setSessionState] = useState<LocalSession | null>(() => getLocalSession())
  const [ready, setReady] = useState(false)
  const [connecting, setConnecting] = useState(false)
  const [connectError, setConnectError] = useState('')
  const [conversations, setConversations] = useState<ConversationItem[]>([])
  const [activeId, setActiveId] = useState('')
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [loadingMessages, setLoadingMessages] = useState(false)
  const [loadingOlder, setLoadingOlder] = useState(false)
  const [historyCompleted, setHistoryCompleted] = useState(true)
  const [contactBadge, setContactBadge] = useState(0)
  const [viewingHistory, setViewingHistory] = useState(false)
  const viewingHistoryRef = useRef(false)
  const navigationGenRef = useRef(0)
  const messagesRef = useRef(messages)
  messagesRef.current = messages
  const activeIdRef = useRef(activeId)
  const selfIdRef = useRef(session?.userId || '')
  const selectGenRef = useRef(0)
  const connectLockRef = useRef(false)
  const msgCursorRef = useRef('')
  const historyDoneRef = useRef(true)
  const loadingOlderRef = useRef(false)
  activeIdRef.current = activeId
  selfIdRef.current = session?.userId || ''

  const totalUnread = useMemo(
    () => conversations.reduce((sum, c) => sum + (c.unreadCount || 0), 0),
    [conversations],
  )

  const setSession = useCallback((s: LocalSession | null) => {
    setSessionState(s)
  }, [])

  const refreshConversations = useCallback(async () => {
    if (!isImReady()) return
    try {
      const list = await fetchConversationList()
      setConversations(Array.isArray(list) ? list : [])
    } catch (e) {
      console.error('refreshConversations', e)
    }
  }, [])

  const patchConversation = useCallback((id: string, patch: Partial<ConversationItem>) => {
    if (!id) return
    setConversations((prev) =>
      prev.map((c) => (c.conversationID === id ? { ...c, ...patch } : c)),
    )
  }, [])

  const refreshContactBadge = useCallback(async () => {
    if (!isImReady()) return
    try {
      const [friends, groups] = await Promise.all([
        fetchFriendApplications().catch(() => []),
        fetchGroupApplications().catch(() => []),
      ])
      setContactBadge(friends.length + groups.length)
    } catch {
      /* ignore */
    }
  }, [])

  const connectIm = useCallback(async (s: LocalSession) => {
    if (connectLockRef.current) return
    connectLockRef.current = true
    setConnecting(true)
    setConnectError('')
    if (!isImReady()) setReady(false)
    try {
      await imLogin(s.userId, s.userSig, s.sdkAppId)
      setReady(true)
      setConnecting(false)
      try {
        const list = await fetchConversationList()
        setConversations(Array.isArray(list) ? list : [])
      } catch (e) {
        console.error('fetchConversationList', e)
      }
      void refreshContactBadge()
      const uid = s.userId
      void (async () => {
        for (const ms of [800, 2000]) {
          await new Promise<void>((r) => window.setTimeout(r, ms))
          if (!isImReady() || currentImUserId() !== uid) return
          try {
            const again = await fetchConversationList()
            setConversations(Array.isArray(again) ? again : [])
          } catch {
            /* ignore */
          }
        }
      })()
    } catch (e) {
      console.error('IM connect failed', e)
      setReady(false)
      setConnectError(e instanceof Error ? e.message : 'IM 连接失败')
      throw e
    } finally {
      connectLockRef.current = false
      setConnecting(false)
    }
  }, [refreshContactBadge])

  const bootFromSession = useCallback(
    async (s: LocalSession) => {
      if (session?.userId && session.userId !== s.userId) {
        try {
          await imLogout()
        } catch {
          /* 切换账号时本地仍继续清理 */
        }
        ++selectGenRef.current
        setReady(false)
        setConversations([])
        activeIdRef.current = ''
        navigationGenRef.current++
        viewingHistoryRef.current = false
        setViewingHistory(false)
        setActiveId('')
        setMessages([])
        setContactBadge(0)
        msgCursorRef.current = ''
        historyDoneRef.current = true
        setHistoryCompleted(true)
        setLoadingOlder(false)
      }
      setSessionState(s)
      try {
        await connectIm(s)
      } catch {
        /* connectError 已写入 */
      }
    },
    [connectIm, session?.userId],
  )

  const switchAccount = useCallback(
    async (s: LocalSession) => {
      if (connectLockRef.current) throw new Error('账号正在连接，请稍后再试')
      if (session?.userId === s.userId && currentImUserId() === s.userId && isImReady()) return
      try {
        await imLogout()
      } catch {
        /* SDK 退出失败不阻断本地切换 */
      }
      ++selectGenRef.current
      setReady(false)
      setConnectError('')
      setConversations([])
      activeIdRef.current = ''
      navigationGenRef.current++
      viewingHistoryRef.current = false
      setViewingHistory(false)
      setActiveId('')
      setMessages([])
      setContactBadge(0)
      msgCursorRef.current = ''
      historyDoneRef.current = true
      setHistoryCompleted(true)
      setLoadingOlder(false)
      saveLocalSession(s)
      setSessionState(s)
      await connectIm(s)
    },
    [connectIm, session?.userId],
  )

  const reconnect = useCallback(async () => {
    const s = getLocalSession()
    if (!s) throw new Error('未登录')
    await connectIm(s)
  }, [connectIm])

  const logout = useCallback(async () => {
    try {
      await imLogout()
    } catch {
      /* ignore */
    }
    clearLocalSession()
    setSessionState(null)
    setReady(false)
    setConnecting(false)
    setConnectError('')
    setConversations([])
    activeIdRef.current = ''
    selectGenRef.current++
    navigationGenRef.current++
    viewingHistoryRef.current = false
    setViewingHistory(false)
    setActiveId('')
    setMessages([])
    setContactBadge(0)
    msgCursorRef.current = ''
    historyDoneRef.current = true
    setHistoryCompleted(true)
    setLoadingOlder(false)
  }, [])

  const selectConversation = useCallback(async (id: string) => {
    if (id && isHiddenConversation({ conversationID: id })) return
    const gen = ++selectGenRef.current
    activeIdRef.current = id
    setActiveId(id)
    navigationGenRef.current++
    if (viewingHistoryRef.current) setMessages([])
    viewingHistoryRef.current = false
    setViewingHistory(false)
    if (!id || !isImReady()) {
      setMessages([])
      return
    }
    setLoadingMessages(true)
    setLoadingOlder(false)
    loadingOlderRef.current = false
    msgCursorRef.current = ''
    historyDoneRef.current = false
    setHistoryCompleted(false)
    try {
      try {
        await getConversationProfile(id)
      } catch {
        /* 新会话首次拉取可能尚未建档 */
      }
      const { messages: list, nextReqMessageID, isCompleted } = await fetchMessageList(id)
      if (gen !== selectGenRef.current) return
      msgCursorRef.current = nextReqMessageID
      historyDoneRef.current = isCompleted
      setHistoryCompleted(isCompleted)
      const hydrated = await hydratePeerRead(list)
      if (gen !== selectGenRef.current) return
      const mapped = hydrated.map((m) => ({ ...m, isPeerRead: isMessagePeerRead(m) }))
      setMessages((prev) => {
        const ids = new Set(mapped.map((m) => m.ID).filter(Boolean))
        const extras = prev.filter(
          (m) => m.conversationID === id && m.ID && !ids.has(m.ID),
        )
        const merged = extras.length
          ? [...mapped, ...extras].sort(
              (a, b) => (Number(a.time) || 0) - (Number(b.time) || 0),
            )
          : mapped
        return trimMessageWindow(merged)
      })
      await setMessageRead(id)
      const incoming = hydrated.filter((m) => m.from !== selfIdRef.current)
      void sendReadReceipts(incoming)
      const convList = await fetchConversationList()
      if (gen !== selectGenRef.current) return
      setConversations(Array.isArray(convList) ? convList : [])
    } catch (e) {
      console.error('fetchMessageList', e)
      if (gen === selectGenRef.current) setMessages([])
    } finally {
      if (gen === selectGenRef.current) setLoadingMessages(false)
    }
  }, [])

  const loadOlderMessages = useCallback(async () => {
    const id = activeIdRef.current
    if (viewingHistoryRef.current || !id || !isImReady()) return false
    if (historyDoneRef.current || loadingOlderRef.current) return false
    const cursor = msgCursorRef.current
    if (!cursor) {
      historyDoneRef.current = true
      setHistoryCompleted(true)
      return false
    }
    loadingOlderRef.current = true
    setLoadingOlder(true)
    const gen = selectGenRef.current
    try {
      const { messages: older, nextReqMessageID, isCompleted } = await fetchMessageList(id, cursor)
      if (gen !== selectGenRef.current || activeIdRef.current !== id) return false
      const stuck = !nextReqMessageID || nextReqMessageID === cursor
      msgCursorRef.current = stuck ? '' : nextReqMessageID
      const done = isCompleted || stuck
      historyDoneRef.current = done
      setHistoryCompleted(done)
      const hydrated = await hydratePeerRead(older)
      if (gen !== selectGenRef.current || activeIdRef.current !== id) return false
      const mapped = hydrated.map((m) => ({ ...m, isPeerRead: isMessagePeerRead(m) }))
      let added = 0
      let hitLimit = false
      setMessages((prev) => {
        if (prev.length >= MESSAGE_WINDOW_LIMIT) {
          hitLimit = true
          return prev
        }
        const ids = new Set(prev.map((m) => m.ID).filter(Boolean))
        const extra = mapped.filter((m) => m.ID && !ids.has(m.ID))
        added = extra.length
        if (!extra.length) return prev
        return trimMessageWindow(
          [...extra, ...prev].sort((a, b) => (Number(a.time) || 0) - (Number(b.time) || 0)),
        )
      })
      if (hitLimit) {
        historyDoneRef.current = true
        setHistoryCompleted(true)
        return false
      }
      return added > 0
    } catch (e) {
      console.error('loadOlderMessages', e)
      return false
    } finally {
      loadingOlderRef.current = false
      if (gen === selectGenRef.current) setLoadingOlder(false)
    }
  }, [])

  const locateMessage = useCallback(async (messageID: string) => {
    const id = activeIdRef.current
    if (!id || !messageID || !isImReady()) return false
    const nav = ++navigationGenRef.current
    if (messagesRef.current.some((m) => m.ID === messageID && m.conversationID === id)) return true
    const gen = selectGenRef.current
    const cancelled = () => nav !== navigationGenRef.current || gen !== selectGenRef.current || activeIdRef.current !== id
    const quote = messagesRef.current.map(parseQuoteFromMessage).find((q) => q?.messageID === messageID) || undefined
    const context = await findMessageContext(id, messageID, quote, cancelled)
    if (cancelled() || !context?.length) return false
    const hydrated = await hydratePeerRead(context)
    if (cancelled()) return false
    // Invalidate any older-page request still in flight before replacing the window.
    selectGenRef.current++
    loadingOlderRef.current = false
    setLoadingOlder(false)
    viewingHistoryRef.current = true
    setViewingHistory(true)
    setMessages(hydrated)
    return true
  }, [])

  const appendOutgoing = useCallback((msg: ChatMessage) => {
    if (viewingHistoryRef.current && msg.conversationID === activeIdRef.current) {
      void selectConversation(activeIdRef.current)
    }
    setMessages((prev) => {
      const cur = activeIdRef.current
      if (msg.conversationID && cur && msg.conversationID !== cur) return prev
      if (msg.ID && prev.some((m) => m.ID === msg.ID)) return prev
      return trimMessageWindow([...prev, msg])
    })
  }, [selectConversation])

  const patchMessage = useCallback((id: string, patch: Partial<ChatMessage>) => {
    setMessages((prev) => prev.map((m) => (m.ID === id ? { ...m, ...patch } : m)))
  }, [])

  const replaceOutgoing = useCallback((localId: string, sent: ChatMessage) => {
    if (sent.ID) moveMediaCache(localId, sent.ID)
    setMessages((prev) => {
      const local = prev.find((m) => m.ID === localId)
      const localSecond = Number((local?.payload as { second?: number } | undefined)?.second || 0)
      const sentSecond = Number((sent.payload as { second?: number } | undefined)?.second || 0)
      const merged: ChatMessage = {
        ...sent,
        localPreview: sent.localPreview || local?.localPreview,
        progress: 1,
        sendFailed: false,
        payload: {
          ...(sent.payload || {}),
          ...(localSecond > 0 && sentSecond <= 0 ? { second: localSecond } : null),
        },
      }
      if (sent.ID && prev.some((m) => m.ID === sent.ID && m.ID !== localId)) {
        return prev.filter((m) => m.ID !== localId)
      }
      return prev.map((m) => (m.ID === localId ? merged : m))
    })
  }, [])

  const removeMessage = useCallback((id: string) => {
    setMessages((prev) => prev.filter((m) => m.ID !== id))
  }, [])

  useEffect(() => {
    const offUnlock = installAudioUnlock()
    const onFirst = () => requestNotifyPermission()
    window.addEventListener('pointerdown', onFirst, { once: true })
    return () => {
      offUnlock()
      window.removeEventListener('pointerdown', onFirst)
    }
  }, [])

  useEffect(() => {
    if (!session?.userId || !session.userSig || !session.sdkAppId) return
    let offConv = () => {}
    let offMsg = () => {}
    let offRevoke = () => {}
    let offRead = () => {}
    let cancelled = false
    let badgeTimer = 0

    ;(async () => {
      try {
        if (!isImReady()) {
          await connectIm(session)
        } else {
          setReady(true)
          // 关键：已 ready 也必须重拉会话，否则 HMR/重挂载会空列表
          await refreshConversations()
          void refreshContactBadge()
        }
        if (cancelled) return

        offConv = onConversationListUpdated(() => {
          void refreshConversations()
        })
        offMsg = onMessageReceived((list) => {
          const cur = activeIdRef.current
          const me = selfIdRef.current
          if (cur) {
            const hit = list.filter((m) => m.conversationID === cur)
            if (hit.length && !viewingHistoryRef.current) {
              setMessages((prev) => {
                const seen = new Set(prev.map((m) => m.ID))
                const extra = hit.filter((m) => m.ID && !seen.has(m.ID))
                if (!extra.length) return prev
                return trimMessageWindow([...prev, ...extra])
              })
              void setMessageRead(cur)
              void sendReadReceipts(hit.filter((m) => m.from !== me))
            }
          }
          for (const m of list) {
            if (me && m.from === me) continue
            if (extractRtcFromMessage(m)) continue
            const isGroup = String(m.conversationID || '').startsWith('GROUP')
            const system = isGroup || isCenteredTipMessage(m)
            playIncomingMessageSound({ system })
            const title = m.nick || m.from || '新消息'
            const body = messageCopyText(m) || '[消息]'
            showAppNotification({
              title,
              body,
              tag: m.conversationID || m.ID,
              force: !cur || m.conversationID !== cur,
            })
            break
          }
          void refreshConversations()
        })
        offRevoke = onMessageRevoked((list) => {
          setMessages((prev) =>
            prev.map((m) => {
              const hit = list.find((r) => r.ID === m.ID)
              if (!hit) return m
              return {
                ...m,
                isRevoked: true,
                revoker: hit.revoker || m.revoker,
              }
            }),
          )
        })
        try {
          offRead = onMessageReadByPeer((updates) => {
            setMessages((prev) => applyPeerReadUpdates(prev, updates))
          })
        } catch {
          offRead = () => {}
        }

        badgeTimer = window.setInterval(() => {
          void refreshContactBadge()
        }, 30000)
      } catch {
        /* connectIm 已写 error */
      }
    })()

    return () => {
      cancelled = true
      offConv()
      offMsg()
      offRevoke()
      offRead()
      if (badgeTimer) window.clearInterval(badgeTimer)
    }
  }, [
    session?.userId,
    session?.userSig,
    session?.sdkAppId,
    connectIm,
    refreshConversations,
    refreshContactBadge,
  ])

  const value = useMemo<AppStoreValue>(
    () => ({
      session,
      ready,
      connecting,
      connectError,
      conversations,
      activeId,
      messages,
      loadingMessages,
      loadingOlder,
      historyCompleted,
      viewingHistory,
      locateMessage,
      totalUnread,
      contactBadge,
      setSession,
      selectConversation,
      loadOlderMessages,
      appendOutgoing,
      patchMessage,
      replaceOutgoing,
      removeMessage,
      refreshConversations,
      patchConversation,
      refreshContactBadge,
      logout,
      bootFromSession,
      switchAccount,
      reconnect,
    }),
    [
      session,
      ready,
      connecting,
      connectError,
      conversations,
      activeId,
      messages,
      loadingMessages,
      loadingOlder,
      historyCompleted,
      viewingHistory,
      locateMessage,
      totalUnread,
      contactBadge,
      setSession,
      selectConversation,
      loadOlderMessages,
      appendOutgoing,
      patchMessage,
      replaceOutgoing,
      removeMessage,
      refreshConversations,
      patchConversation,
      refreshContactBadge,
      logout,
      bootFromSession,
      switchAccount,
      reconnect,
    ],
  )

  return <AppStoreContext.Provider value={value}>{children}</AppStoreContext.Provider>
}

export function useAppStore() {
  const ctx = useContext(AppStoreContext)
  if (!ctx) throw new Error('useAppStore must be used within AppStoreProvider')
  return ctx
}
