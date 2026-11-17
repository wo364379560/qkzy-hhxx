import {
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
  type MouseEvent,
  type ReactNode,
  type UIEvent,
} from 'react'
import { splitEmojiText } from '@/im/emojiCatalog'
import { displayAtAllText } from '@/utils/atAll'
import { fetchUserProfiles } from '@/im/ImClient'
import { pinGroupMessage } from '@/im/groupPin'
import {
  deleteMessagesLocal,
  forwardMessagesMerged,
  forwardMessagesSeparate,
  messageCopyText,
  MSG_SELECT_MAX,
  parseQuoteFromMessage,
} from '@/im/messageActions'
import {
  imageUrlFromMessage,
  isFileMessage,
  isImageMessage,
  isMergerMessage,
  isSoundMessage,
  isVideoMessage,
  mergerInfoFromMessage,
  videoInfoFromMessage,
} from '@/im/mediaPayload'
import { parseLargeAttachment, resolveMessageAttachmentUrl } from '@/im/largeAttachment'
import {
  isCenteredTipMessage,
  resolveTipText,
  shouldHideGroupCreateNoise,
  shouldHideDuplicateGroupCreate,
  shouldHideRtcInviteMessage,
} from '@/im/messageTips'
import { extractRtcFromMessage, rtcBubbleText } from '@/im/rtcCall'
import { canRevokeMessage, revokeMessageSmart } from '@/im/revoke'
import { parseGiftPayload, parseStickerPayload, resolveGiftDisplayUrl } from '@/im/stickerCatalog'
import type { ChatMessage } from '@/im/types'
import { hiddenSpeakerIds } from '@/im/hiddenSpeakers'
import { addFavorite, favoriteFromMessage } from '@/services/favorites'
import { useAppStore } from '@/store/AppStore'
import {
  convTitle,
  groupIdFromConversation,
  isGroupConversation,
} from '@/utils/conversation'
import { ConversationPickerModal } from './ConversationPickerModal'
import { FileBubble, MediaHoverActs } from './FileBubble'
import { MediaViewer, type MediaPreviewItem } from './MediaViewer'
import { VoiceBubble } from './VoiceBubble'
import {
  ChatAvatarContextMenu,
  type AvatarMenuTarget,
} from './ChatAvatarContextMenu'
import { MergerDetailModal } from './MergerDetailModal'
import styles from './MessageList.module.css'
import iconCopy from '@/assets/icon/copy.svg'
import iconDelete from '@/assets/icon/delete.svg'
import iconForward from '@/assets/icon/forward.svg'
import iconPin from '@/assets/icon/pin.svg'
import iconQuote from '@/assets/icon/reply.svg'
import iconReadOne from '@/assets/icon/read_receipt_check.svg'
import iconReadAll from '@/assets/icon/read_receipt_check_all.svg'
import iconStar from '@/assets/icon/star.svg'
import iconMultiSelect from '@/assets/icon/multi_select.svg'
import iconRevoke from '@/assets/icon/name_edit.svg'

const hydratedSenders = new Set<string>()

function formatBubbleTime(ts: number) {
  if (!ts) return ''
  const d = new Date(ts < 1e12 ? ts * 1000 : ts)
  const mm = String(d.getMonth() + 1).padStart(2, '0')
  const dd = String(d.getDate()).padStart(2, '0')
  const hh = String(d.getHours()).padStart(2, '0')
  const mi = String(d.getMinutes()).padStart(2, '0')
  return `${mm}-${dd} ${hh}:${mi}`
}

type Props = {
  onOpenUserProfile?: (userId: string, nickname?: string, avatar?: string) => void
  isGroupAdmin?: boolean
  quoteMsg?: ChatMessage | null
  onQuote?: (msg: ChatMessage | null) => void
  onMultiChange?: (multi: boolean) => void
  highlightId?: string
  onPinned?: () => void
  actionsLocked?: boolean
}

function Avatar({ src, name }: { src?: string; name: string }) {
  return (
    <div className={styles.avatar}>
      {src ? <img src={src} alt="" /> : name.slice(0, 1)}
    </div>
  )
}

function TextBody({ text }: { text: string }) {
  return (
    <>
      {splitEmojiText(displayAtAllText(text)).map((p, i) =>
        p.type === 'emoji' ? (
          p.assetUrl ? (
            <img key={i} className={styles.emojiImg} src={p.assetUrl} alt="" />
          ) : (
            <span key={i} className={styles.emoji}>
              {p.value}
            </span>
          )
        ) : (
          <span key={i}>{p.value}</span>
        ),
      )}
    </>
  )
}

function MediaFrame({
  msg,
  children,
}: {
  msg: ChatMessage
  children: ReactNode
}) {
  const sending = typeof msg.progress === 'number' && msg.progress < 1 && !msg.sendFailed
  const pct = Math.max(0, Math.min(1, msg.progress || 0))
  return (
    <span className={styles.mediaFrame}>
      {children}
      {sending ? (
        <span className={styles.progressMask} aria-hidden>
          <svg className={styles.progressRing} viewBox="0 0 36 36">
            <circle className={styles.progressTrack} cx="18" cy="18" r="14" />
            <circle
              className={styles.progressValue}
              cx="18"
              cy="18"
              r="14"
              strokeDasharray={`${Math.max(6, pct * 88)} 88`}
            />
          </svg>
        </span>
      ) : null}
      {msg.sendFailed ? (
        <em
          className={styles.sendFail}
          title={msg.sendError || '发送失败'}
          onClick={(e) => {
            e.stopPropagation()
            alert(msg.sendError || '发送失败')
          }}
        >
          {msg.sendError?.includes('COS 未配置') ? 'COS未配置' : '发送失败'}
        </em>
      ) : null}
    </span>
  )
}

function renderNormalBody(
  msg: ChatMessage,
  selfId: string,
  opts: {
    multi?: boolean
    onOpenMerger?: (m: ChatMessage) => void
    onJumpQuote?: (id: string) => void
    onOpenMedia?: (item: MediaPreviewItem) => void
  },
) {
  const quote = parseQuoteFromMessage(msg)
  const t = String(msg.type || '')
  const mine = msg.from === selfId || msg.flow === 'out'
  const body = (() => {
    if (isImageMessage(msg)) {
      const url = imageUrlFromMessage(msg)
      return (
        <div className={`${styles.fileRow} ${mine ? styles.fileRowMine : ''}`}>
          {mine ? <MediaHoverActs msgId={msg.ID} url={url} name="图片.jpg" mine /> : null}
          <MediaFrame msg={msg}>
            {url ? (
              <button
                type="button"
                className={styles.mediaLink}
                onClick={(e) => {
                  if (opts.multi) return
                  e.stopPropagation()
                  opts.onOpenMedia?.({ kind: 'image', url, name: '图片.jpg' })
                }}
              >
                <img className={styles.thumb} src={url} alt="图片" />
              </button>
            ) : (
              '[图片]'
            )}
          </MediaFrame>
          {!mine ? <MediaHoverActs msgId={msg.ID} url={url} name="图片.jpg" mine={false} /> : null}
        </div>
      )
    }
    if (isSoundMessage(msg)) {
      return (
        <MediaFrame msg={msg}>
          <VoiceBubble msg={msg} />
        </MediaFrame>
      )
    }
    if (isVideoMessage(msg)) {
      const v = videoInfoFromMessage(msg)
      const att = parseLargeAttachment(msg)
      const videoName = att?.fileName || '视频.mp4'
      if (!v.url && !v.snapshot) {
        return (
          <MediaFrame msg={msg}>
            <FileBubble
              msg={msg}
              mine={mine}
              multi={opts.multi}
              onOpenVideo={(url, name) => opts.onOpenMedia?.({ kind: 'video', url, name })}
            />
          </MediaFrame>
        )
      }
      return (
        <div className={`${styles.fileRow} ${mine ? styles.fileRowMine : ''}`}>
          {mine ? (
            <MediaHoverActs msgId={msg.ID} url={v.url} name={videoName} mine msg={msg} />
          ) : null}
          <MediaFrame msg={msg}>
            <button
              type="button"
              className={styles.mediaLink}
              onClick={(e) => {
                if (opts.multi) return
                e.stopPropagation()
                void (async () => {
                  try {
                    const url =
                      (att?.fileID ? await resolveMessageAttachmentUrl(msg, false) : '') || v.url
                    if (!url) return
                    opts.onOpenMedia?.({ kind: 'video', url, name: videoName })
                  } catch (err) {
                    alert(err instanceof Error ? err.message : '无法打开视频')
                  }
                })()
              }}
            >
              {v.url ? (
                <video className={styles.thumbVideo} src={v.url} poster={v.snapshot || undefined} />
              ) : (
                <img className={styles.thumb} src={v.snapshot} alt="视频封面" />
              )}
            </button>
          </MediaFrame>
          {!mine ? (
            <MediaHoverActs msgId={msg.ID} url={v.url} name={videoName} mine={false} msg={msg} />
          ) : null}
        </div>
      )
    }
    if (isFileMessage(msg)) {
      return (
        <MediaFrame msg={msg}>
          <FileBubble
            msg={msg}
            mine={mine}
            multi={opts.multi}
            onOpenVideo={(url, name) => opts.onOpenMedia?.({ kind: 'video', url, name })}
          />
        </MediaFrame>
      )
    }
    if (isMergerMessage(msg)) {
      const info = mergerInfoFromMessage(msg)
      return (
        <button
          type="button"
          className={styles.mergerBubble}
          onClick={(e) => {
            if (opts.multi) return
            e.stopPropagation()
            opts.onOpenMerger?.(msg)
          }}
        >
          <b>{info.title}</b>
          {info.abstracts.slice(0, 3).map((line, i) => (
            <span key={i}>{line}</span>
          ))}
          <em>聊天记录</em>
        </button>
      )
    }
    if (t === 'TIMCustomElem' || t.includes('Custom')) {
      const rtc = extractRtcFromMessage(msg)
      if (rtc) {
        const text = rtcBubbleText(rtc, msg.from === selfId || msg.flow === 'out')
        const video = rtc.mediaType === 2 || text.includes('视频')
        return (
          <span className={styles.callBubble}>
            <span
              className={`${styles.callIcon} ${video ? styles.callVideo : styles.callAudio}`}
              aria-hidden
            />
            {text}
          </span>
        )
      }
      const gift = parseGiftPayload(msg.payload?.data)
      if (gift) {
        const url = resolveGiftDisplayUrl(gift)
        return (
          <span className={styles.media}>
            {url ? <img src={url} alt={gift.giftName} /> : <em>[动图] {gift.giftName}</em>}
          </span>
        )
      }
      const sticker = parseStickerPayload(msg.payload?.data)
      if (sticker) return <em>[贴纸] {sticker.stickerName}</em>
      const desc = msg.payload?.description || ''
      if (desc && !/自定义消息|custom/i.test(desc)) return desc
      return '[消息]'
    }
    if (t === 'TIMTextElem' || t.includes('Text')) {
      return <TextBody text={msg.payload?.text || ''} />
    }
    return msg.payload?.text ? <TextBody text={msg.payload.text} /> : msg.payload?.description || '[消息]'
  })()

  return (
    <>
      {quote ? (
        <button
          type="button"
          className={styles.quoteRef}
          onClick={(e) => {
            e.stopPropagation()
            if (quote.messageID) opts.onJumpQuote?.(quote.messageID)
          }}
        >
          <b>{quote.messageSender || '引用'}</b>
          <span>{quote.messageAbstract}</span>
        </button>
      ) : null}
      {body}
    </>
  )
}

export function MessageList({
  onOpenUserProfile,
  isGroupAdmin = false,
  onQuote,
  onMultiChange,
  highlightId,
  onPinned,
  actionsLocked = false,
}: Props) {
  const {
    messages,
    loadingMessages,
    loadingOlder,
    historyCompleted,
    loadOlderMessages,
    locateMessage,
    viewingHistory,
    selectConversation,
    session,
    activeId,
    conversations,
    patchMessage,
    removeMessage,
  } = useAppStore()
  const bottomRef = useRef<HTMLDivElement>(null)
  const scrollerRef = useRef<HTMLDivElement>(null)
  const skipStickRef = useRef(false)
  const nearBottomRef = useRef(true)
  const pendingRestoreRef = useRef<{ height: number; top: number } | null>(null)
  const olderLockRef = useRef(false)
  const selfId = session?.userId || ''
  const isGroup = isGroupConversation(activeId)
  const groupId = isGroup ? groupIdFromConversation(activeId) : ''
  const [menu, setMenu] = useState<{ x: number; y: number; msg: ChatMessage } | null>(null)
  const [avatarMenu, setAvatarMenu] = useState<AvatarMenuTarget | null>(null)
  const [hideTick, setHideTick] = useState(0)
  const hiddenIds = useMemo(() => hiddenSpeakerIds(activeId), [activeId, hideTick])
  const [multi, setMulti] = useState(false)
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [pickerOpen, setPickerOpen] = useState(false)
  const [forwardMerged, setForwardMerged] = useState(false)
  const [forwardMsgs, setForwardMsgs] = useState<ChatMessage[]>([])
  const [forwarding, setForwarding] = useState(false)
  const [mergerMsg, setMergerMsg] = useState<ChatMessage | null>(null)
  const [preview, setPreview] = useState<MediaPreviewItem | null>(null)
  const rowRefs = useRef<Record<string, HTMLDivElement | null>>({})

  const [locating, setLocating] = useState(false)
  const jumpGeneration = useRef(0)
  useEffect(() => {
    jumpGeneration.current++
    setLocating(false)
    return () => { jumpGeneration.current++ }
  }, [activeId])

  async function jumpToMessage(id: string, silent = false) {
    const gen = ++jumpGeneration.current
    setLocating(true)
    nearBottomRef.current = false
    pendingRestoreRef.current = null
    skipStickRef.current = true
    try {
      if (!await locateMessage(id)) {
        if (gen === jumpGeneration.current && !silent) alert('未找到原消息，可能已删除或超出历史消息保留期限')
        return
      }
      // Wait for React to commit the retrieved history before reading row refs.
      for (let attempt = 0; attempt < 12; attempt++) {
        if (gen !== jumpGeneration.current) return
        const el = rowRefs.current[id]
        if (el) {
          el.scrollIntoView({ behavior: 'smooth', block: 'center' })
          el.classList.add(styles.flash)
          window.setTimeout(() => el.classList.remove(styles.flash), 1600)
          return
        }
        await new Promise((resolve) => window.setTimeout(resolve, 50))
      }
      if (gen === jumpGeneration.current && !silent) alert('原消息当前已被隐藏，无法定位')
    } catch {
      if (gen === jumpGeneration.current && !silent) alert('加载原消息失败，请重试')
    } finally {
      if (gen === jumpGeneration.current) setLocating(false)
    }
  }

  useEffect(() => {
    if (!highlightId) return
    const timer = window.setTimeout(() => jumpToMessage(highlightId, false), 40)
    return () => window.clearTimeout(timer)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [highlightId])

  useEffect(() => {
    nearBottomRef.current = true
    skipStickRef.current = false
    pendingRestoreRef.current = null
    olderLockRef.current = false
    hydratedSenders.clear()
    setMulti(false)
    setSelected(new Set())
    setMenu(null)
    setAvatarMenu(null)
  }, [activeId])

  useLayoutEffect(() => {
    const el = scrollerRef.current
    if (!el) return
    const pending = pendingRestoreRef.current
    if (pending) {
      el.scrollTop = pending.top + (el.scrollHeight - pending.height)
      pendingRestoreRef.current = null
      skipStickRef.current = false
      return
    }
    if (skipStickRef.current) {
      skipStickRef.current = false
      return
    }
    if (nearBottomRef.current) {
      el.scrollTop = el.scrollHeight
    }
  }, [messages.length, activeId, loadingMessages])

  useEffect(() => {
    if (locating || viewingHistory || loadingMessages || loadingOlder || historyCompleted || !activeId) return
    const el = scrollerRef.current
    if (!el) return
    if (el.scrollHeight <= el.clientHeight + 32) {
      void loadOlderMessages()
    }
  }, [
    messages.length,
    locating,
    viewingHistory,
    loadingMessages,
    loadingOlder,
    historyCompleted,
    activeId,
    loadOlderMessages,
  ])

  async function loadOlderKeepingPlace() {
    const el = scrollerRef.current
    if (!el || locating || viewingHistory || olderLockRef.current || historyCompleted) return
    olderLockRef.current = true
    pendingRestoreRef.current = { height: el.scrollHeight, top: el.scrollTop }
    skipStickRef.current = true
    try {
      const added = await loadOlderMessages()
      if (!added) {
        pendingRestoreRef.current = null
        skipStickRef.current = false
      }
    } finally {
      olderLockRef.current = false
    }
  }

  function onScrollerScroll(e: UIEvent<HTMLDivElement>) {
    const el = e.currentTarget
    const distBottom = el.scrollHeight - el.scrollTop - el.clientHeight
    nearBottomRef.current = distBottom < 80
    if (el.scrollTop < 64) void loadOlderKeepingPlace()
  }

  useEffect(() => {
    onMultiChange?.(multi)
  }, [multi, onMultiChange])

  useEffect(() => {
    if (!isGroup || !messages.length) return
    let cancelled = false
    ;(async () => {
      const need = messages
        .filter((m) => m.from && m.from !== selfId && !m.nick && !hydratedSenders.has(m.from))
        .map((m) => m.from)
      const uniq = [...new Set(need)].slice(0, 30)
      if (!uniq.length) return
      uniq.forEach((id) => hydratedSenders.add(id))
      try {
        const profiles = await fetchUserProfiles(uniq)
        if (cancelled) return
        for (const p of profiles) {
          if (!p.userID) continue
          messages
            .filter((m) => m.from === p.userID && !m.nick)
            .forEach((m) => patchMessage(m.ID, { nick: p.nick, avatar: p.avatar }))
        }
      } catch {
        /* ignore */
      }
    })()
    return () => {
      cancelled = true
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isGroup, messages.length, selfId, activeId])

  useEffect(() => {
    if (!menu && !avatarMenu) return
    const close = () => {
      setMenu(null)
      setAvatarMenu(null)
    }
    const t = window.setTimeout(() => {
      window.addEventListener('click', close)
      window.addEventListener('contextmenu', close)
    }, 0)
    return () => {
      window.clearTimeout(t)
      window.removeEventListener('click', close)
      window.removeEventListener('contextmenu', close)
    }
  }, [menu, avatarMenu])

  useEffect(() => {
    if (!actionsLocked) return
    setMulti(false)
    setSelected(new Set())
    setPickerOpen(false)
    setForwardMsgs([])
  }, [actionsLocked])

  function openMenuAt(msg: ChatMessage, x: number, y: number) {
    if (isCenteredTipMessage(msg) || multi) return
    const mw = 176
    const mh = 320
    setAvatarMenu(null)
    setMenu({
      x: Math.min(Math.max(8, x), window.innerWidth - mw - 8),
      y: Math.min(Math.max(8, y), window.innerHeight - mh - 8),
      msg,
    })
  }

  function onContextMenu(e: MouseEvent, msg: ChatMessage) {
    e.preventDefault()
    e.stopPropagation()
    openMenuAt(msg, e.clientX, e.clientY)
  }

  function onAvatarContextMenu(
    e: MouseEvent,
    userId: string,
    nick?: string,
    avatar?: string,
  ) {
    e.preventDefault()
    e.stopPropagation()
    if (!userId) return
    setMenu(null)
    setAvatarMenu({
      x: e.clientX,
      y: e.clientY,
      userId,
      nick: nick || userId,
      avatar: avatar || '',
    })
  }

  function toggleSelect(id: string) {
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else if (next.size >= MSG_SELECT_MAX) alert(`最多选择 ${MSG_SELECT_MAX} 条消息`)
      else next.add(id)
      return next
    })
  }

  function toggleSelectAllMessages() {
    const ids = messages
      .filter((m) => !isCenteredTipMessage(m) && !shouldHideRtcInviteMessage(m))
      .map((m) => m.ID)
      .slice(-MSG_SELECT_MAX)
    const allOn = ids.length > 0 && ids.every((id) => selected.has(id))
    setSelected(allOn ? new Set() : new Set(ids))
  }

  function exitMulti() {
    setMulti(false)
    setSelected(new Set())
  }

  async function doRevoke(msg: ChatMessage) {
    setMenu(null)
    try {
      const groupID = isGroup ? groupIdFromConversation(activeId) : undefined
      const { silent } = await revokeMessageSmart(msg, {
        selfId,
        isGroupAdmin,
        groupID,
      })
      patchMessage(msg.ID, { isRevoked: true, revoker: silent ? selfId : msg.from })
    } catch (err) {
      alert(err instanceof Error ? err.message : '撤回失败')
    }
  }

  async function doCopy(msg: ChatMessage) {
    setMenu(null)
    try {
      await navigator.clipboard.writeText(messageCopyText(msg))
    } catch {
      alert('复制失败')
    }
  }

  async function doDelete(msgs: ChatMessage[]) {
    setMenu(null)
    try {
      await deleteMessagesLocal(msgs)
      msgs.forEach((m) => removeMessage(m.ID))
      exitMulti()
    } catch (err) {
      alert(err instanceof Error ? err.message : '删除失败')
    }
  }

  function startForward(msgs: ChatMessage[], merged: boolean) {
    setMenu(null)
    if (!msgs.length) return
    setForwardMsgs(msgs)
    setForwardMerged(merged)
    setPickerOpen(true)
  }

  async function onPickerConfirm(ids: string[]) {
    if (!ids.length || !forwardMsgs.length || forwarding) return
    setForwarding(true)
    try {
      if (forwardMerged) {
        const conv = conversations.find((c) => c.conversationID === activeId)
        const title = isGroup ? '群聊的聊天记录' : `${conv ? convTitle(conv) : '聊天'}的聊天记录`
        await forwardMessagesMerged(forwardMsgs, ids, title)
      } else {
        await forwardMessagesSeparate(forwardMsgs, ids)
      }
      setPickerOpen(false)
      setForwardMsgs([])
      exitMulti()
    } catch (err) {
      alert(err instanceof Error ? err.message : '转发失败')
    } finally {
      setForwarding(false)
    }
  }

  function enterMulti(msg: ChatMessage) {
    setMenu(null)
    setMulti(true)
    setSelected(new Set([msg.ID]))
  }

  function doFavorite(msg: ChatMessage) {
    setMenu(null)
    addFavorite(
      favoriteFromMessage({
        ID: msg.ID,
        type: msg.type,
        from: msg.from,
        nick: msg.nick,
        conversationID: msg.conversationID || activeId,
        payload: msg.payload,
        text: messageCopyText(msg),
      }),
    )
    alert('已收藏')
  }

  async function doPin(msg: ChatMessage) {
    setMenu(null)
    if (!isGroup) return
    try {
      await pinGroupMessage(groupIdFromConversation(activeId), msg, true)
      onPinned?.()
    } catch (e) {
      alert(e instanceof Error ? e.message : '置顶失败')
    }
  }

  const selectedMsgs = messages.filter((m) => selected.has(m.ID))
  const selectableIds = messages
    .filter((m) => !isCenteredTipMessage(m) && !shouldHideRtcInviteMessage(m))
    .map((m) => m.ID)
    .slice(-MSG_SELECT_MAX)
  const allMsgsSelected =
    selectableIds.length > 0 && selectableIds.every((id) => selected.has(id))

  return (
    <div className={styles.list}>
      {locating || viewingHistory ? (
        <div className={styles.historyHint}>
          {locating ? '正在定位原消息…' : '正在查看引用消息'}
          <button type="button" className={styles.historyHintBtn} onClick={() => {
            jumpGeneration.current++
            setLocating(false)
            nearBottomRef.current = true
            selectConversation(activeId)
          }}>返回最新消息</button>
        </div>
      ) : null}
      {multi ? (
        <div className={styles.multiBar}>
          <button type="button" onClick={exitMulti}>
            ×
          </button>
          <strong>已选 {selected.size}/{MSG_SELECT_MAX}</strong>
          <button
            type="button"
            className={styles.dangerIcon}
            disabled={!selected.size}
            onClick={() => void doDelete(selectedMsgs)}
            title="删除"
          >
            删
          </button>
        </div>
      ) : null}

      <div className={styles.scroller} ref={scrollerRef} onScroll={onScrollerScroll}>
        {loadingMessages ? <div className={styles.status}>加载消息…</div> : null}
        {!loadingMessages && !messages.length ? (
          <div className={styles.status}>暂无消息</div>
        ) : null}
        {!viewingHistory && !loadingMessages && messages.length ? (
          <div className={styles.historyHint}>
            {loadingOlder ? (
              '加载更早的消息…'
            ) : historyCompleted ? (
              '没有更多消息了'
            ) : (
              <button type="button" className={styles.historyHintBtn} onClick={() => void loadOlderKeepingPlace()}>
                查看更早消息
              </button>
            )}
          </div>
        ) : null}
        {messages.map((m) => {
        if (shouldHideRtcInviteMessage(m)) return null
        if (shouldHideGroupCreateNoise(m) || shouldHideDuplicateGroupCreate(m, messages)) return null
        const tip = resolveTipText(m, selfId)
        if (tip) {
          return (
            <div key={m.ID} ref={(el) => { rowRefs.current[m.ID] = el }} className={styles.center}>
              <em className={styles.tip}>{tip}</em>
            </div>
          )
        }
        if (isCenteredTipMessage(m)) return null
        if (m.from && m.from !== selfId && hiddenIds.has(m.from)) {
          return null
        }
        const mine = m.flow === 'out' || m.from === selfId
        const nick = m.nick || m.from
        const showPeerMeta = !mine && isGroup
        const checked = selected.has(m.ID)
        return (
          <div
            key={m.ID}
            ref={(el) => {
              rowRefs.current[m.ID] = el
            }}
            className={`${styles.row} ${mine ? styles.mine : styles.peer} ${multi ? styles.rowMulti : ''} ${highlightId === m.ID ? styles.flash : ''}`}
            onClick={() => {
              if (multi) toggleSelect(m.ID)
            }}
          >
            {multi ? (
              <i className={checked ? styles.selOn : styles.sel} aria-hidden />
            ) : null}
            {!mine ? (
              <button
                type="button"
                className={styles.avatarBtn}
                onClick={(e) => {
                  e.stopPropagation()
                  onOpenUserProfile?.(m.from, m.nick, m.avatar)
                }}
                onContextMenu={(e) => onAvatarContextMenu(e, m.from, m.nick, m.avatar)}
              >
                <Avatar src={m.avatar} name={nick} />
              </button>
            ) : null}
            <div className={styles.bubbleCol}>
              {showPeerMeta ? (
                <button
                  type="button"
                  className={styles.nick}
                  onClick={(e) => {
                    e.stopPropagation()
                    onOpenUserProfile?.(m.from, m.nick, m.avatar)
                  }}
                  onContextMenu={(e) => onAvatarContextMenu(e, m.from, m.nick, m.avatar)}
                >
                  {nick}
                </button>
              ) : null}
              <div className={`${styles.bubbleRow} ${mine ? styles.bubbleRowMine : ''}`}>
                <div
                  className={styles.bubbleStack}
                  onContextMenu={(e) => onContextMenu(e, m)}
                >
                  <div className={styles.bubble}>
                    {renderNormalBody(m, selfId, {
                      multi,
                      onOpenMerger: (msg) => {
                        if (!multi) setMergerMsg(msg)
                      },
                      onJumpQuote: jumpToMessage,
                      onOpenMedia: (item) => {
                        if (!multi) setPreview(item)
                      },
                    })}
                  </div>
                  <div className={`${styles.meta} ${mine ? styles.metaMine : ''}`}>
                    {mine && !isGroup ? (
                      <>
                        <img
                          className={`${styles.ticks} ${m.isPeerRead ? styles.ticksRead : styles.ticksUnread}`}
                          src={m.isPeerRead ? iconReadAll : iconReadOne}
                          alt=""
                        />
                        <span>{m.sendFailed || m.status === 'fail' ? '发送失败' : m.status === 'unSend' ? '发送中' : m.isPeerRead ? '已读' : '未读'}</span>
                      </>
                    ) : null}
                    <span className={styles.time}>{formatBubbleTime(m.time)}</span>
                  </div>
                </div>
              </div>
            </div>
            {mine ? (
              <button
                type="button"
                className={styles.avatarBtn}
                onClick={(e) => {
                  e.stopPropagation()
                  onOpenUserProfile?.(selfId, session?.nickname, session?.avatar)
                }}
                onContextMenu={(e) =>
                  onAvatarContextMenu(e, selfId, session?.nickname, session?.avatar)
                }
              >
                <Avatar src={session?.avatar} name={session?.nickname || selfId} />
              </button>
            ) : null}
          </div>
        )
      })}
      <div ref={bottomRef} />
      </div>

      {multi ? (
        <div className={styles.forwardBar}>
          <button type="button" className={styles.selectAll} onClick={toggleSelectAllMessages}>
            <i className={allMsgsSelected ? styles.selOn : styles.sel} aria-hidden />
            {allMsgsSelected ? '取消全选' : `全选${selected.size ? `(${selected.size})` : ''}`}
          </button>
          <div className={styles.forwardActions}>
            <button
              type="button"
              disabled={!selected.size}
              onClick={() => startForward(selectedMsgs, false)}
            >
              逐条转发
            </button>
            <button
              type="button"
              className={styles.forwardPrimary}
              disabled={!selected.size}
              onClick={() => startForward(selectedMsgs, true)}
            >
              合并转发
            </button>
          </div>
        </div>
      ) : null}

      {menu ? (
        <div className={styles.menu} style={{ left: menu.x, top: menu.y }} onClick={(e) => e.stopPropagation()}>
          <button type="button" onClick={() => void doCopy(menu.msg)}>
            <span>复制</span>
            <img src={iconCopy} alt="" />
          </button>
          {!actionsLocked ? (
            <button
              type="button"
              onClick={() => {
                setMenu(null)
                onQuote?.(menu.msg)
              }}
            >
              <span>引用</span>
              <img src={iconQuote} alt="" />
            </button>
          ) : null}
          {!actionsLocked ? (
            <button type="button" onClick={() => startForward([menu.msg], false)}>
              <span>转发</span>
              <img src={iconForward} alt="" />
            </button>
          ) : null}
          <button type="button" className={styles.menuDanger} onClick={() => void doDelete([menu.msg])}>
            <span>删除</span>
            <img src={iconDelete} alt="" />
          </button>
          <button type="button" onClick={() => doFavorite(menu.msg)}>
            <span>收藏</span>
            <img src={iconStar} alt="" />
          </button>
          {isGroup ? (
            <button type="button" onClick={() => void doPin(menu.msg)}>
              <span>置顶</span>
              <img src={iconPin} alt="" />
            </button>
          ) : null}
          {canRevokeMessage(menu.msg, { selfId, isGroupAdmin }) ? (
            <button type="button" onClick={() => void doRevoke(menu.msg)}>
              <span>撤回</span>
              <img src={iconRevoke} alt="" />
            </button>
          ) : null}
          {!actionsLocked ? (
            <button type="button" onClick={() => enterMulti(menu.msg)}>
              <span>多选</span>
              <img src={iconMultiSelect} alt="" />
            </button>
          ) : null}
        </div>
      ) : null}

      <ChatAvatarContextMenu
        menu={avatarMenu}
        conversationID={activeId}
        groupId={groupId}
        selfId={selfId}
        onClose={() => setAvatarMenu(null)}
        onHidden={() => setHideTick((n) => n + 1)}
      />

      <ConversationPickerModal
        open={pickerOpen}
        excludeId={activeId}
        sending={forwarding}
        hint={forwardMerged ? '合并转发' : forwardMsgs.length > 1 ? '逐条转发' : ''}
        onCancel={() => {
          if (forwarding) return
          setPickerOpen(false)
          setForwardMsgs([])
        }}
        onConfirm={(ids) => void onPickerConfirm(ids)}
      />
      <MediaViewer item={preview} onClose={() => setPreview(null)} />
      <MergerDetailModal
        open={Boolean(mergerMsg)}
        message={mergerMsg}
        onClose={() => setMergerMsg(null)}
      />
    </div>
  )
}
