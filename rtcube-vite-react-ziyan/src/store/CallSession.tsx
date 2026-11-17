/**
 * 全局通话会话 — 对齐 LingDoSpace DoMattxCallService
 * TRTC 必须用 strRoomId=`mllx_${roomId}`，与 Android 同一房间
 */
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
import {
  bindTrtcMedia,
  enterTrtcRoom,
  exitTrtcRoom,
  setTrtcMicMuted,
  setTrtcSpeakerOn,
  startRemoteVideos,
} from '@/im/trtcClient'
import {
  acceptInvite,
  cancelInvite,
  inviteCall,
  MLLX_TRTC_CALL,
  rejectInvite,
  RTC_CALL_TIMEOUT_SEC,
  sendAvCallSummary,
  setActiveInviteId,
  watchCallSignaling,
  watchHangupMessages,
  watchIncomingCalls,
  type RtcMediaType,
} from '@/im/rtcCall'
import { getChat, TencentCloudChat, currentImUserId } from '@/im/ImClient'
import { startLoopSound, stopLoopSound } from '@/audio/sounds'
import { showAppNotification } from '@/audio/notify'
import { useAppStore } from './AppStore'

type CallStatus = 'idle' | 'calling' | 'incoming' | 'connected' | 'ended'

type CallSessionValue = {
  open: boolean
  status: CallStatus
  mediaType: RtcMediaType
  peerId: string
  peerName: string
  roomId: string
  inviteID: string
  muted: boolean
  speakerOn: boolean
  durationSec: number
  startCall: (opts: {
    peerId: string
    peerName?: string
    mediaType: RtcMediaType
  }) => Promise<void>
  accept: () => Promise<void>
  reject: () => Promise<void>
  cancel: () => Promise<void>
  hangup: () => Promise<void>
  toggleMute: () => Promise<void>
  toggleSpeaker: () => Promise<void>
}

const Ctx = createContext<CallSessionValue | null>(null)

// eslint-disable-next-line @typescript-eslint/no-explicit-any
type AnyChat = any

async function sendHangupOnlineOnly(peerId: string, roomId: string, mediaType: RtcMediaType) {
  try {
    const chat = getChat()
    const TYPES = (TencentCloudChat as AnyChat).TYPES
    const selfId = currentImUserId() || ''
    let nick = selfId
    try {
      nick = localStorage.getItem('cn_ziyan_nickname') || selfId
    } catch {
      /* ignore */
    }
    const data = JSON.stringify({
      businessID: MLLX_TRTC_CALL,
      version: 1,
      mediaType,
      roomId: Number(roomId) || 0,
      caller: selfId,
      nickname: nick,
      action: 'hangup',
    })
    const message = chat.createCustomMessage({
      to: peerId,
      conversationType: TYPES.CONV_C2C,
      payload: { data, description: '[通话结束]', extension: '' },
    })
    // 尽量只在线投递，避免落成「自定义消息」
    await chat.sendMessage(message, { onlineUserOnly: true })
  } catch {
    /* ignore */
  }
}

export function CallSessionProvider({ children }: { children: ReactNode }) {
  const { session, ready } = useAppStore()
  const [status, setStatus] = useState<CallStatus>('idle')
  const [mediaType, setMediaType] = useState<RtcMediaType>(1)
  const [peerId, setPeerId] = useState('')
  const [peerName, setPeerName] = useState('')
  const [roomId, setRoomId] = useState('')
  const [inviteID, setInviteID] = useState('')
  const [muted, setMuted] = useState(false)
  const [speakerOn, setSpeakerOn] = useState(true)
  const [durationSec, setDurationSec] = useState(0)
  const timerRef = useRef(0)
  const offSigRef = useRef<() => void>(() => {})
  const tipSentRef = useRef(false)
  const connectedAtRef = useRef(0)
  const endingRef = useRef(false)
  const statusRef = useRef<CallStatus>('idle')
  const peerIdRef = useRef('')
  const peerNameRef = useRef('')
  const mediaTypeRef = useRef<RtcMediaType>(1)
  const roomIdRef = useRef('')
  const inviteIDRef = useRef('')
  const sessionRef = useRef(session)

  statusRef.current = status
  peerIdRef.current = peerId
  peerNameRef.current = peerName
  mediaTypeRef.current = mediaType
  roomIdRef.current = roomId
  inviteIDRef.current = inviteID
  sessionRef.current = session

  useEffect(() => {
    if (status === 'calling') startLoopSound('avPlay')
    else if (status === 'incoming') {
      startLoopSound('avRing')
      showAppNotification({
        title: peerNameRef.current || peerIdRef.current || '来电',
        body: mediaTypeRef.current === 2 ? '邀请你视频通话' : '邀请你语音通话',
        tag: `call-${inviteIDRef.current || peerIdRef.current}`,
        force: true,
      })
    } else {
      stopLoopSound()
    }
    return () => {
      if (status === 'calling' || status === 'incoming') stopLoopSound()
    }
  }, [status])

  const clearTimer = () => {
    if (timerRef.current) {
      window.clearTimeout(timerRef.current)
      timerRef.current = 0
    }
  }

  const sendTipOnce = useCallback(
    async (actionType: number, callEnd = 0, to?: string, media?: RtcMediaType) => {
      const peer = to || peerIdRef.current
      if (!peer || tipSentRef.current) return
      tipSentRef.current = true
      try {
        await sendAvCallSummary({
          to: peer,
          mediaType: media ?? mediaTypeRef.current,
          actionType,
          callEnd: callEnd > 0 ? callEnd : undefined,
        })
      } catch {
        tipSentRef.current = false
      }
    },
    [],
  )

  const reset = useCallback(async () => {
    clearTimer()
    offSigRef.current()
    offSigRef.current = () => {}
    setActiveInviteId('')
    setStatus('idle')
    setPeerId('')
    setPeerName('')
    setRoomId('')
    setInviteID('')
    setMuted(false)
    setSpeakerOn(true)
    setDurationSec(0)
    connectedAtRef.current = 0
    tipSentRef.current = false
    endingRef.current = false
    try {
      await exitTrtcRoom()
    } catch {
      /* ignore */
    }
    stopLoopSound()
  }, [])

  const enterRoomSafe = useCallback(async (room: string, media: RtcMediaType) => {
    const s = sessionRef.current
    if (!s?.userSig || !s.sdkAppId) throw new Error('缺少通话凭证，请重新登录')
    await enterTrtcRoom({
      sdkAppId: s.sdkAppId,
      userId: s.userId,
      userSig: s.userSig,
      roomId: room,
      video: media === 2,
    })
    window.setTimeout(() => {
      void bindTrtcMedia(media === 2)
      if (media === 2) void startRemoteVideos()
    }, 400)
  }, [])

  const startCall = useCallback(
    async (opts: { peerId: string; peerName?: string; mediaType: RtcMediaType }) => {
      if (!opts.peerId) return
      if (statusRef.current !== 'idle') return
      tipSentRef.current = false
      endingRef.current = false
      setPeerId(opts.peerId)
      setPeerName(opts.peerName || opts.peerId)
      setMediaType(opts.mediaType)
      setStatus('calling')
      try {
        const res = await inviteCall({
          userID: opts.peerId,
          mediaType: opts.mediaType,
          peerName: opts.peerName,
        })
        setInviteID(res.inviteID)
        setRoomId(res.roomId)
        offSigRef.current()
        offSigRef.current = watchCallSignaling({
          onAccepted: (id) => {
            if (id !== res.inviteID) return
            clearTimer()
            void (async () => {
              try {
                stopLoopSound()
                await enterRoomSafe(res.roomId, opts.mediaType)
                if (opts.mediaType === 2) await startRemoteVideos()
                connectedAtRef.current = Date.now()
                setDurationSec(0)
                setStatus('connected')
              } catch (e) {
                console.error('[call] caller enterRoom failed', e)
                alert(e instanceof Error ? e.message : '进房失败')
                try {
                  await sendHangupOnlineOnly(opts.peerId, res.roomId, opts.mediaType)
                } catch {
                  /* ignore */
                }
                // 对端已接听时不要标「未接听」，用最短时长摘要
                await sendTipOnce(1, 1, opts.peerId, opts.mediaType)
                await reset()
              }
            })()
          },
          onRejected: (id) => {
            if (id !== res.inviteID) return
            void (async () => {
              // 被叫已发「已拒绝」时，主叫不再重复发 tip
              tipSentRef.current = true
              await reset()
            })()
          },
          onCancelled: (id) => {
            if (id !== res.inviteID) return
            void reset()
          },
          onTimeout: (id) => {
            if (id !== res.inviteID) return
            void (async () => {
              await sendTipOnce(5, 0, opts.peerId, opts.mediaType)
              await reset()
            })()
          },
        })
        timerRef.current = window.setTimeout(() => {
          void (async () => {
            if (statusRef.current !== 'calling') return
            try {
              await cancelInvite(res.inviteID, opts.mediaType, res.roomId)
            } catch {
              /* ignore */
            }
            await sendTipOnce(5, 0, opts.peerId, opts.mediaType)
            await reset()
          })()
        }, RTC_CALL_TIMEOUT_SEC * 1000)
      } catch (e) {
        await reset()
        alert(e instanceof Error ? e.message : '发起通话失败')
      }
    },
    [reset, sendTipOnce, enterRoomSafe],
  )

  const accept = useCallback(async () => {
    const id = inviteIDRef.current
    const room = roomIdRef.current
    const media = mediaTypeRef.current
    if (!id || !room) return
    // 先停超时，避免进房过程中误发「未接听」
    clearTimer()
    try {
      stopLoopSound()
      await acceptInvite(id, media, room)
      await enterRoomSafe(room, media)
      if (media === 2) await startRemoteVideos()
      connectedAtRef.current = Date.now()
      setDurationSec(0)
      setStatus('connected')
    } catch (e) {
      console.error('[call] accept/enter failed', e)
      alert(e instanceof Error ? e.message : '接听失败')
      await reset()
    }
  }, [enterRoomSafe, reset])

  const toggleMute = useCallback(async () => {
    const next = !muted
    setMuted(next)
    try {
      await setTrtcMicMuted(next)
    } catch {
      /* ignore */
    }
  }, [muted])

  const toggleSpeaker = useCallback(async () => {
    const next = !speakerOn
    setSpeakerOn(next)
    try {
      await setTrtcSpeakerOn(next)
    } catch {
      /* ignore */
    }
  }, [speakerOn])

  const reject = useCallback(async () => {
    try {
      if (inviteIDRef.current) {
        await rejectInvite(inviteIDRef.current, mediaTypeRef.current, roomIdRef.current)
      }
      await sendTipOnce(4)
    } finally {
      await reset()
    }
  }, [reset, sendTipOnce])

  const cancel = useCallback(async () => {
    try {
      if (inviteIDRef.current) {
        await cancelInvite(inviteIDRef.current, mediaTypeRef.current, roomIdRef.current)
      }
      await sendTipOnce(2)
    } finally {
      await reset()
    }
  }, [reset, sendTipOnce])

  const hangup = useCallback(async () => {
    if (endingRef.current) return
    endingRef.current = true
    const peer = peerIdRef.current
    const room = roomIdRef.current
    const media = mediaTypeRef.current
    const st = statusRef.current
    try {
      if (st === 'calling' && inviteIDRef.current) {
        await cancelInvite(inviteIDRef.current, media, room)
        await sendTipOnce(2, 0, peer, media)
      } else if (st === 'connected') {
        if (peer && room) await sendHangupOnlineOnly(peer, room, media)
        const sec =
          connectedAtRef.current > 0
            ? Math.max(1, Math.floor((Date.now() - connectedAtRef.current) / 1000))
            : 0
        await sendTipOnce(sec > 0 ? 1 : 5, sec, peer, media)
      }
    } finally {
      await reset()
    }
  }, [reset, sendTipOnce])

  // 全局来电监听
  useEffect(() => {
    if (!ready || !session?.userId) return
    const off = watchIncomingCalls((info) => {
      if (statusRef.current !== 'idle') return
      tipSentRef.current = false
      endingRef.current = false
      setPeerId(info.peer)
      setPeerName(info.peerName || info.peer)
      setMediaType(info.mediaType)
      setRoomId(info.roomId)
      setInviteID(info.inviteID)
      setStatus('incoming')
      timerRef.current = window.setTimeout(() => {
        void (async () => {
          if (statusRef.current !== 'incoming') return
          try {
            await rejectInvite(info.inviteID, info.mediaType, info.roomId)
          } catch {
            /* ignore */
          }
          await sendTipOnce(5, 0, info.peer, info.mediaType)
          await reset()
        })()
      }, RTC_CALL_TIMEOUT_SEC * 1000)
      offSigRef.current()
      offSigRef.current = watchCallSignaling({
        onCancelled: (id) => {
          if (info.inviteID && id !== info.inviteID) return
          void reset()
        },
        onTimeout: (id) => {
          if (info.inviteID && id !== info.inviteID) return
          void (async () => {
            await sendTipOnce(5, 0, info.peer, info.mediaType)
            await reset()
          })()
        },
      })
    })
    return () => {
      off()
    }
  }, [ready, session?.userId, reset, sendTipOnce])

  // 对端挂断：对端会发 av_call 摘要，本端只退房，避免双条
  useEffect(() => {
    if (!ready) return
    return watchHangupMessages((info) => {
      if (statusRef.current !== 'connected' && statusRef.current !== 'calling') return
      if (roomIdRef.current && String(info.roomId) !== String(roomIdRef.current)) return
      void (async () => {
        if (endingRef.current) return
        endingRef.current = true
        tipSentRef.current = true
        await reset()
      })()
    })
  }, [ready, reset])

  const value = useMemo<CallSessionValue>(
    () => ({
      open: status !== 'idle',
      status: status === 'idle' ? 'ended' : status,
      mediaType,
      peerId,
      peerName,
      roomId,
      inviteID,
      muted,
      speakerOn,
      durationSec,
      startCall,
      accept,
      reject,
      cancel,
      hangup,
      toggleMute,
      toggleSpeaker,
    }),
    [
      status,
      mediaType,
      peerId,
      peerName,
      roomId,
      inviteID,
      muted,
      speakerOn,
      durationSec,
      startCall,
      accept,
      reject,
      cancel,
      hangup,
      toggleMute,
      toggleSpeaker,
    ],
  )

  return <Ctx.Provider value={value}>{children}</Ctx.Provider>
}

export function useCallSession() {
  const ctx = useContext(Ctx)
  if (!ctx) throw new Error('useCallSession outside provider')
  return ctx
}
