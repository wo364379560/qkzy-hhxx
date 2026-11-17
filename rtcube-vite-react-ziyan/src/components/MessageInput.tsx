import { useEffect, useMemo, useRef, useState, type KeyboardEvent } from 'react'
import { useI18n } from '@/i18n/I18nProvider'
import { insertEmojiTag } from '@/im/emojiInput'
import type { ChatMessage, GroupMember } from '@/im/types'
import { AT_MEMBER_EVENT, type AtMemberPayload } from '@/im/atMemberEvent'
import { fetchGroupDetail } from '@/im/groupActions'
import { getChat, sendCustomJsonMessage, sendTextMessage, TencentCloudChat } from '@/im/ImClient'
import { buildQuoteCloudCustomData, messageCopyText } from '@/im/messageActions'
import {
  blobToAudioFile,
  sendFileMessage,
  sendImageMessage,
  sendSoundMessage,
  sendVideoMessage,
} from '@/im/mediaSend'
import {
  buildGiftPayload,
  buildStickerPayload,
  type MediaAsset,
} from '@/im/stickerCatalog'
import {
  canRecordVoice,
  cancelVoiceRecord,
  isVoiceRecording,
  startVoiceRecord,
  stopVoiceRecord,
} from '@/im/voiceRecorder'
import { filterShortcuts, INSERT_SHORTCUT_EVENT, loadShortcuts, takePendingShortcut } from '@/services/shortcuts'
import { AT_ALL_LABEL, displayAtAllText, isAtAllUserId, textHasAtAll } from '@/utils/atAll'
import { setConversationDraft } from '@/im/profileActions'
import { useAppStore } from '@/store/AppStore'
import { groupIdFromConversation, isGroupConversation } from '@/utils/conversation'
import { captureDisplayCanvas, copyImageFile } from '@/im/screenshot'
import { AtMemberPicker } from './AtMemberPicker'
import { EmojiComposer } from './EmojiComposer'
import { EmojiPanel } from './EmojiPanel'
import { GifPanel } from './GifPanel'
import { ScreenshotCrop } from './ScreenshotCrop'
import styles from './MessageInput.module.css'
import iconEmoji from '@/assets/icon/emoji.svg'
import iconFile from '@/assets/icon/file_action.svg'
import iconImage from '@/assets/icon/image.svg'
import iconLightning from '@/assets/icon/lightning.svg'
import iconMic from '@/assets/icon/mic.svg'
import iconSearch from '@/assets/icon/search.svg'
import iconStar from '@/assets/icon/star.svg'
import iconSendVideo from '@/assets/images/camera_on.png'
import iconVideoCall from '@/assets/images/join_group_call.png'
import iconVoiceCall from '@/assets/icon/voice_call.svg'

type Props = {
  onCall?: (mediaType: 1 | 2) => void
  quoteMsg?: ChatMessage | null
  onClearQuote?: () => void
  onOpenSearch?: () => void
  actionsLocked?: boolean
  lockReason?: string
}

type Panel = 'none' | 'sticker'

function formatRecTime(sec: number) {
  const m = String(Math.floor(sec / 60)).padStart(2, '0')
  const s = String(sec % 60).padStart(2, '0')
  return `${m}:${s}`
}

const lastSpeakAt = new Map<string, number>()

type PendingImage = { id: string; file: File; url: string }

export function MessageInput({
  onCall,
  quoteMsg,
  onClearQuote,
  onOpenSearch,
  actionsLocked = false,
  lockReason = '',
}: Props) {
  const { t } = useI18n()
  const { activeId, appendOutgoing, refreshConversations, session, conversations, patchConversation, patchMessage, replaceOutgoing } =
    useAppStore()
  const [text, setText] = useState('')
  const [sending, setSending] = useState(false)
  const [atOpen, setAtOpen] = useState(false)
  const [atKeyword, setAtKeyword] = useState('')
  const [atUsers, setAtUsers] = useState<string[]>([])
  const [slashOpen, setSlashOpen] = useState(false)
  const [slashKeyword, setSlashKeyword] = useState('')
  const [slashIndex, setSlashIndex] = useState(0)
  const skipDraftSave = useRef(true)
  const [panel, setPanel] = useState<Panel>('none')
  const [emojiOpen, setEmojiOpen] = useState(false)
  const [gifOpen, setGifOpen] = useState(false)
  const [recording, setRecording] = useState(false)
  const [recordSec, setRecordSec] = useState(0)
  const [muteAll, setMuteAll] = useState(false)
  const [speakFreqSec, setSpeakFreqSec] = useState(0)
  const [isManager, setIsManager] = useState(false)
  const imageRef = useRef<HTMLInputElement>(null)
  const videoRef = useRef<HTMLInputElement>(null)
  const fileRef = useRef<HTMLInputElement>(null)
  const [pendingImages, setPendingImages] = useState<PendingImage[]>([])
  const [shotCanvas, setShotCanvas] = useState<HTMLCanvasElement | null>(null)
  const [shotBusy, setShotBusy] = useState(false)

  const isGroup = useMemo(() => isGroupConversation(activeId), [activeId])
  const groupId = useMemo(
    () => (isGroup ? groupIdFromConversation(activeId) : ''),
    [activeId, isGroup],
  )
  const shortcuts = useMemo(
    () => filterShortcuts(loadShortcuts(), slashKeyword),
    [slashKeyword, slashOpen],
  )

  useEffect(() => {
    setSlashIndex(0)
  }, [slashKeyword, slashOpen, shortcuts.length])

  useEffect(() => {
    skipDraftSave.current = true
    const conv = conversations.find((c) => c.conversationID === activeId)
    const pending = takePendingShortcut()
    const draft = String(conv?.draftText || '')
    setText(pending ? (draft ? `${draft} ${pending}` : pending) : draft)
    setAtUsers([])
    setSlashOpen(false)
    setPendingImages((prev) => {
      prev.forEach((p) => URL.revokeObjectURL(p.url))
      return []
    })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [activeId])

  useEffect(() => {
    if (!activeId) return
    if (skipDraftSave.current) {
      skipDraftSave.current = false
      return
    }
    const t = window.setTimeout(() => {
      const draft = text
      void setConversationDraft(activeId, draft)
      patchConversation(activeId, { draftText: draft })
    }, 450)
    return () => window.clearTimeout(t)
  }, [text, activeId, patchConversation])

  useEffect(() => {
    const onInsert = (e: Event) => {
      const next = String((e as CustomEvent<{ text?: string }>).detail?.text || '')
      if (!next) return
      takePendingShortcut()
      setText((prev) => (prev ? `${prev}${/\s$/.test(prev) ? '' : ' '}${next}` : next))
      setSlashOpen(false)
    }
    window.addEventListener(INSERT_SHORTCUT_EVENT, onInsert)
    return () => window.removeEventListener(INSERT_SHORTCUT_EVENT, onInsert)
  }, [])

  useEffect(() => {
    if (!isGroup || !groupId) {
      setMuteAll(false)
      setSpeakFreqSec(0)
      setIsManager(false)
      return
    }
    let cancelled = false
    void (async () => {
      try {
        const d = await fetchGroupDetail(groupId)
        if (cancelled) return
        setMuteAll(Boolean(d.muteAllMembers))
        setSpeakFreqSec(d.speakFreqSec || 0)
        const selfId = session?.userId || ''
        let manager = Boolean(selfId && d.ownerID === selfId)
        if (selfId && !manager) {
          try {
            const res = await getChat().getGroupMemberProfile({
              groupID: groupId,
              userIDList: [selfId],
            })
            const role = String(res?.data?.memberList?.[0]?.role || '')
            manager = role === 'Admin' || role === 'Owner'
          } catch {
            /* ignore */
          }
        }
        if (!cancelled) setIsManager(manager)
      } catch {
        /* ignore */
      }
    })()
    return () => {
      cancelled = true
    }
  }, [isGroup, groupId, session?.userId])

  useEffect(() => {
    if (!isGroup || !groupId) return
    const onAt = (e: Event) => {
      const d = (e as CustomEvent<AtMemberPayload>).detail
      if (!d || d.groupId !== groupId) return
      const label = d.nick || d.userID
      setText((prev) => `${prev}${prev && !/\s$/.test(prev) ? ' ' : ''}@${label} `)
      setAtUsers((prev) => (prev.includes(d.userID) ? prev : [...prev, d.userID]))
    }
    window.addEventListener(AT_MEMBER_EVENT, onAt)
    return () => window.removeEventListener(AT_MEMBER_EVENT, onAt)
  }, [isGroup, groupId])

  useEffect(() => {
    if (!recording) {
      setRecordSec(0)
      return
    }
    setRecordSec(0)
    const timer = window.setInterval(() => {
      setRecordSec((s) => s + 1)
    }, 1000)
    return () => window.clearInterval(timer)
  }, [recording])

  useEffect(() => {
    if (recording && recordSec >= 60) void finishRecord()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [recording, recordSec])

  useEffect(() => {
    if (!actionsLocked) return
    setEmojiOpen(false)
    setGifOpen(false)
    setPanel('none')
    setAtOpen(false)
    setSlashOpen(false)
    if (recording || isVoiceRecording()) {
      cancelVoiceRecord()
      setRecording(false)
    }
    onClearQuote?.()
  }, [actionsLocked, onClearQuote, recording])

  useEffect(() => {
    return () => {
      pendingImages.forEach((p) => URL.revokeObjectURL(p.url))
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  function queueImages(files: File[]) {
    if (!files.length || actionsLocked) return
    setPendingImages((prev) => [
      ...prev,
      ...files.map((file) => ({
        id: `${Date.now()}-${Math.random().toString(16).slice(2)}`,
        file,
        url: URL.createObjectURL(file),
      })),
    ])
  }

  function removePending(id: string) {
    setPendingImages((prev) => {
      const hit = prev.find((p) => p.id === id)
      if (hit) URL.revokeObjectURL(hit.url)
      return prev.filter((p) => p.id !== id)
    })
  }

  async function sendPendingImages() {
    if (!activeId || !pendingImages.length) return
    const list = pendingImages
    setPendingImages([])
    const hooks = mediaHooks()
    for (const item of list) {
      try {
        await finishMediaSend(sendImageMessage(activeId, item.file, hooks))
        markSpoke()
      } catch (e) {
        alert(e instanceof Error ? e.message : '发送失败')
      } finally {
        URL.revokeObjectURL(item.url)
      }
    }
  }

  async function startScreenshot() {
    if (actionsLocked || shotBusy) return
    setShotBusy(true)
    try {
      assertCanSpeak()
      const canvas = await captureDisplayCanvas()
      setShotCanvas(canvas)
    } catch (e) {
      const name = e && typeof e === 'object' ? String((e as { name?: string }).name || '') : ''
      if (name === 'NotAllowedError' || name === 'AbortError') return
      alert(e instanceof Error ? e.message : '截图失败')
    } finally {
      setShotBusy(false)
    }
  }

  async function onShotConfirm(file: File) {
    setShotCanvas(null)
    void copyImageFile(file)
    queueImages([file])
  }

  function detectTriggers(value: string) {
    if (isGroup) {
      const at = value.match(/@([^\s@]*)$/)
      if (at) {
        setAtOpen(true)
        setAtKeyword(at[1] || '')
        setSlashOpen(false)
        return
      }
    }
    setAtOpen(false)
    const slash = value.match(/(?:^|\s)\/([^\n]*)$/)
    if (slash) {
      setSlashOpen(true)
      setSlashKeyword(slash[1] || '')
    } else {
      setSlashOpen(false)
      setSlashKeyword('')
    }
  }

  function onPickAt(member: GroupMember | 'all') {
    const label = member === 'all' ? AT_ALL_LABEL : member.nick
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const atAll = (TencentCloudChat as any).TYPES?.MSG_AT_ALL || '__kImSDK_MesssageAtALL__'
    const id = member === 'all' ? atAll : member.userID
    setText((prev) => {
      const m = prev.match(/^(.*)@([^\s@]*)$/s)
      if (m) return `${m[1]}@${label} `
      return `${prev}@${label} `
    })
    setAtUsers((prev) => (prev.includes(id) ? prev : [...prev, id]))
    setAtOpen(false)
  }

  function pickShortcut(content: string) {
    const body = content.trim()
    setSlashOpen(false)
    setSlashKeyword('')
    setPanel('none')
    if (!body) return
    void doSend(body)
  }

  function onComposerKeyDown(e: KeyboardEvent<HTMLDivElement>) {
    if (!slashOpen || !shortcuts.length) return false
    if (e.key === 'ArrowDown') {
      setSlashIndex((i) => (i + 1) % shortcuts.length)
      return true
    }
    if (e.key === 'ArrowUp') {
      setSlashIndex((i) => (i - 1 + shortcuts.length) % shortcuts.length)
      return true
    }
    if (e.key === 'Enter') {
      const hit = shortcuts[slashIndex] || shortcuts[0]
      if (hit) pickShortcut(hit.text)
      return true
    }
    if (e.key === 'Escape') {
      setSlashOpen(false)
      return true
    }
    return false
  }

  function mediaHooks() {
    return {
      onPending: (msg: ChatMessage) => {
        appendOutgoing(msg)
      },
      onProgress: (id: string, percent: number) => {
        patchMessage(id, { progress: percent })
      },
      onFailed: (id: string, reason?: string) => {
        patchMessage(id, { sendFailed: true, progress: 1, sendError: reason })
      },
    }
  }

  async function finishMediaSend(task: Promise<{ localId: string; sent: ChatMessage; localPreview?: string }>) {
    const result = await task
    if (result?.sent) {
      replaceOutgoing(result.localId, {
        ...result.sent,
        localPreview: result.localPreview,
        progress: 1,
      })
    }
    void refreshConversations()
  }

  function rememberSent(msg: ChatMessage | null | undefined) {
    if (!msg) return
    appendOutgoing(msg)
    void refreshConversations()
  }

  function assertCanSpeak() {
    if (actionsLocked) throw new Error(lockReason || '当前无法发言')
    if (!isGroup) return
    if (muteAll && !isManager) throw new Error('全员禁言中，暂时无法发言')
    if (speakFreqSec > 0 && !isManager) {
      const key = `${groupId}:${session?.userId || ''}`
      const wait = speakFreqSec * 1000 - (Date.now() - (lastSpeakAt.get(key) || 0))
      if (wait > 0) throw new Error(`发言过于频繁，请 ${Math.ceil(wait / 1000)} 秒后再试`)
    }
  }

  function markSpoke() {
    if (!isGroup || speakFreqSec <= 0) return
    lastSpeakAt.set(`${groupId}:${session?.userId || ''}`, Date.now())
  }

  async function doSend(raw?: string) {
    const content = displayAtAllText((raw ?? text).trim())
    if (!activeId || sending) return
    if (!content && !pendingImages.length) return
    try {
      assertCanSpeak()
    } catch (e) {
      alert(e instanceof Error ? e.message : '无法发言')
      return
    }
    if (pendingImages.length) {
      await sendPendingImages()
    }
    if (!content) return
    setSending(true)
    try {
      const ccd = quoteMsg ? buildQuoteCloudCustomData(quoteMsg) : undefined
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const atAll = (TencentCloudChat as any).TYPES?.MSG_AT_ALL || '__kImSDK_MesssageAtALL__'
      const atList = isGroup ? [...atUsers] : []
      if (isGroup && textHasAtAll(content) && !atList.some(isAtAllUserId)) {
        atList.push(atAll)
      }
      const msg = await sendTextMessage(
        activeId,
        content,
        atList.length ? atList : undefined,
        ccd,
      )
      markSpoke()
      rememberSent(msg)
      setText('')
      setAtUsers([])
      setAtOpen(false)
      setSlashOpen(false)
      setEmojiOpen(false)
      setPanel('none')
      if (activeId) {
        void setConversationDraft(activeId, '')
        patchConversation(activeId, { draftText: '' })
      }
      onClearQuote?.()
    } catch (e) {
      alert(e instanceof Error ? e.message : '发送失败')
    } finally {
      setSending(false)
    }
  }

  async function sendAsset(asset: MediaAsset) {
    if (!activeId || sending) return
    try {
      assertCanSpeak()
    } catch (e) {
      alert(e instanceof Error ? e.message : '无法发言')
      return
    }
    if (asset.kind === 'sticker' && asset.ext === 'tgs') {
      alert('电报 .tgs 播放器待接入')
      return
    }
    setSending(true)
    try {
      const payload =
        asset.kind === 'gift' ? buildGiftPayload(asset) : buildStickerPayload(asset)
      const msg = await sendCustomJsonMessage(
        activeId,
        payload,
        asset.kind === 'gift' ? `[动图] ${asset.name}` : `[贴纸] ${asset.name}`,
      )
      markSpoke()
      rememberSent(msg)
      setPanel('none')
      setGifOpen(false)
    } catch (e) {
      alert(e instanceof Error ? e.message : '发送失败')
    } finally {
      setSending(false)
    }
  }

  async function onPickFile(kind: 'image' | 'video' | 'file', file?: File | null) {
    if (!activeId || !file) return
    try {
      assertCanSpeak()
    } catch (e) {
      alert(e instanceof Error ? e.message : '无法发言')
      return
    }
    setPanel('none')
    setEmojiOpen(false)
    setGifOpen(false)
    const hooks = mediaHooks()
    try {
      await finishMediaSend(
        kind === 'image'
          ? sendImageMessage(activeId, file, hooks)
          : kind === 'video'
            ? sendVideoMessage(activeId, file, hooks)
            : sendFileMessage(activeId, file, hooks),
      )
      markSpoke()
    } catch (e) {
      alert(e instanceof Error ? e.message : '发送失败')
    }
  }

  async function finishRecord() {
    if (!activeId || !isVoiceRecording()) {
      setRecording(false)
      return
    }
    setRecording(false)
    try {
      const rec = await stopVoiceRecord()
      const file = blobToAudioFile(rec.blob, rec.mime || rec.blob.type, rec.durationSec)
      await finishMediaSend(sendSoundMessage(activeId, file, rec.durationSec, mediaHooks()))
      markSpoke()
    } catch (e) {
      cancelVoiceRecord()
      alert(e instanceof Error ? e.message : '录音失败')
    }
  }

  async function toggleRecord() {
    if (!activeId) return
    if (!canRecordVoice()) {
      alert('当前环境不支持录音')
      return
    }
    if (recording || isVoiceRecording()) {
      await finishRecord()
      return
    }
    try {
      assertCanSpeak()
    } catch (e) {
      alert(e instanceof Error ? e.message : '无法发言')
      return
    }
    try {
      await startVoiceRecord()
      setRecording(true)
      setEmojiOpen(false)
      setPanel('none')
    } catch (e) {
      alert(e instanceof Error ? e.message : '无法开始录音')
    }
  }

  function cancelRecord() {
    cancelVoiceRecord()
    setRecording(false)
    setRecordSec(0)
  }

  function pickEmoji(tag: string) {
    const next = insertEmojiTag(text, tag, text.length).text
    setText(next)
    detectTriggers(next)
  }

  if (!activeId) {
    return <div className={styles.disabled}>选择左侧会话开始聊天</div>
  }

  return (
    <div className={styles.wrap}>
      <input
        ref={imageRef}
        type="file"
        accept="image/*"
        hidden
        onChange={(e) => {
          void onPickFile('image', e.target.files?.[0])
          e.target.value = ''
        }}
      />
      <input
        ref={videoRef}
        type="file"
        accept="video/*"
        hidden
        onChange={(e) => {
          void onPickFile('video', e.target.files?.[0])
          e.target.value = ''
        }}
      />
      <input
        ref={fileRef}
        type="file"
        hidden
        onChange={(e) => {
          void onPickFile('file', e.target.files?.[0])
          e.target.value = ''
        }}
      />

      {emojiOpen && !actionsLocked ? <EmojiPanel onPick={pickEmoji} /> : null}
      {gifOpen && !actionsLocked ? <GifPanel onPick={(asset) => void sendAsset(asset)} /> : null}

      {actionsLocked ? <div className={styles.lockedHint}>{lockReason || '当前无法发言'}</div> : null}

      {quoteMsg && !actionsLocked ? (
        <div className={styles.quoteBar}>
          <div className={styles.quoteMeta}>
            <b>引用 {quoteMsg.nick || quoteMsg.from}</b>
            <span>{messageCopyText(quoteMsg)}</span>
          </div>
          <button type="button" onClick={() => onClearQuote?.()}>
            ×
          </button>
        </div>
      ) : null}

      {recording && !actionsLocked ? (
        <div className={styles.recordBar}>
          <span className={styles.recordLabel}>正在录音</span>
          <div className={styles.wave} aria-hidden>
            {Array.from({ length: 12 }).map((_, i) => (
              <i key={i} style={{ animationDelay: `${i * 0.08}s` }} />
            ))}
          </div>
          <span className={styles.recordTime}>{formatRecTime(recordSec)}</span>
          <button type="button" className={styles.recordCancel} onClick={cancelRecord}>
            取消
          </button>
          <button
            type="button"
            className={styles.recordSend}
            disabled={sending}
            onClick={() => void finishRecord()}
          >
            发送
          </button>
        </div>
      ) : null}

      <div className={styles.toolbar}>
        <button
          type="button"
          className={emojiOpen ? styles.on : ''}
          title="表情"
          disabled={actionsLocked}
          onClick={() => {
            setEmojiOpen((v) => !v)
            setGifOpen(false)
            setPanel('none')
          }}
        >
          <img src={iconEmoji} alt="表情" />
        </button>
        <button
          type="button"
          title="图片"
          disabled={actionsLocked}
          onClick={() => imageRef.current?.click()}
        >
          <img src={iconImage} alt="图片" />
        </button>
        <button
          type="button"
          className={shotBusy ? styles.on : ''}
          title="截图"
          disabled={actionsLocked || shotBusy}
          onClick={() => void startScreenshot()}
        >
          <span className={styles.shotIcon} aria-hidden>
            ✂️
          </span>
        </button>
        <button
          type="button"
          title="文件"
          disabled={actionsLocked}
          onClick={() => fileRef.current?.click()}
        >
          <img src={iconFile} alt="文件" />
        </button>
        <button
          type="button"
          title="视频"
          disabled={actionsLocked}
          onClick={() => videoRef.current?.click()}
        >
          <img src={iconSendVideo} alt="发送视频" />
        </button>
        <button
          type="button"
          className={recording ? styles.on : ''}
          title="语音"
          disabled={actionsLocked}
          onClick={() => void toggleRecord()}
        >
          <img src={iconMic} alt="语音" />
        </button>
        <button
          type="button"
          className={gifOpen ? styles.on : ''}
          title="动图"
          disabled={actionsLocked}
          onClick={() => {
            setGifOpen((v) => !v)
            setEmojiOpen(false)
            setPanel('none')
          }}
        >
          <img src={iconStar} alt="动图" />
        </button>
        <button
          type="button"
          className={slashOpen ? styles.on : ''}
          title="快捷语"
          disabled={actionsLocked}
          onClick={() => {
            setSlashOpen((v) => !v)
            setSlashKeyword('')
            setEmojiOpen(false)
            setGifOpen(false)
            setPanel('none')
          }}
        >
          <img src={iconLightning} alt="快捷语" />
        </button>
        {!isGroup ? (
          <>
            <button type="button" title="语音通话" onClick={() => onCall?.(1)}>
              <img src={iconVoiceCall} alt="语音通话" />
            </button>
            <button type="button" title="视频通话" onClick={() => onCall?.(2)}>
              <img src={iconVideoCall} alt="视频通话" />
            </button>
          </>
        ) : null}
        <button
          type="button"
          className={styles.searchBtn}
          title="搜索聊天记录"
          onClick={() => {
            setEmojiOpen(false)
            setGifOpen(false)
            onOpenSearch?.()
          }}
        >
          <img src={iconSearch} alt="搜索" />
        </button>
      </div>

      {pendingImages.length > 0 && !actionsLocked ? (
        <div className={styles.pending}>
          {pendingImages.map((p) => (
            <div key={p.id} className={styles.pendingItem}>
              <img src={p.url} alt="" />
              <button type="button" title="移除" onClick={() => removePending(p.id)}>
                ×
              </button>
            </div>
          ))}
        </div>
      ) : null}

      <div className={styles.composer}>
        <AtMemberPicker
          groupId={groupId}
          open={atOpen && !actionsLocked}
          keyword={atKeyword}
          onPick={onPickAt}
          onClose={() => setAtOpen(false)}
        />
        {slashOpen && !actionsLocked ? (
          <div className={styles.slash}>
            <div className={styles.slashHead}>快捷语 · 点击即发送</div>
            {shortcuts.map((s, i) => (
              <button
                key={s.id}
                type="button"
                className={i === slashIndex ? styles.slashOn : undefined}
                onMouseDown={(e) => {
                  e.preventDefault()
                  e.stopPropagation()
                  pickShortcut(s.text)
                }}
              >
                {s.text}
              </button>
            ))}
            {!shortcuts.length ? <div className={styles.panelEmpty}>无匹配</div> : null}
          </div>
        ) : null}
        <EmojiComposer
          value={text}
          placeholder={actionsLocked ? lockReason || '当前无法发言' : t('placeholder')}
          disabled={actionsLocked}
          onChange={(v) => {
            if (actionsLocked) return
            setText(v)
            detectTriggers(v)
          }}
          onKeyDown={onComposerKeyDown}
          onEnter={() => {
            if (!actionsLocked) void doSend()
          }}
          onPasteFiles={queueImages}
        />
        <button
          type="button"
          className={styles.send}
          disabled={actionsLocked || sending || (!text.trim() && !pendingImages.length)}
          onClick={() => void doSend()}
        >
          {t('send')}
        </button>
      </div>
      {shotCanvas ? (
        <ScreenshotCrop
          canvas={shotCanvas}
          onCancel={() => setShotCanvas(null)}
          onConfirm={(file) => void onShotConfirm(file)}
        />
      ) : null}
    </div>
  )
}
