import {
  getChatAttachmentUploadSettings,
  uploadChatAttachment,
} from '@/api/attachment'
import { playSound } from '@/audio/sounds'
import { currentImUserId, formatSendError, getChat, TencentCloudChat } from './ImClient'
import {
  DEFAULT_MAX_ATTACHMENT_MB,
  buildLargeAttachmentPayload,
  explainChatAttachmentError,
  isNativeSizeLimitError,
  isVideoFileName,
  largeAttachmentDescription,
  readVideoDuration,
  shouldRouteToCOS,
} from './largeAttachment'
import { rememberOutgoingFile } from './mediaCache'
import type { ChatMessage } from './types'

// eslint-disable-next-line @typescript-eslint/no-explicit-any
type AnyChat = any

export type MediaSendHooks = {
  onPending: (msg: ChatMessage) => void
  onProgress: (id: string, percent: number) => void
  onFailed?: (id: string, reason?: string) => void
}

export type MediaSendResult = {
  localId: string
  sent: ChatMessage
  localPreview?: string
}

function convTarget(conversationID: string) {
  const isGroup = conversationID.startsWith('GROUP')
  const to = conversationID.replace(/^(C2C|GROUP)/, '')
  const TYPES = (TencentCloudChat as AnyChat).TYPES
  return {
    isGroup,
    to,
    conversationType: isGroup ? TYPES.CONV_GROUP : TYPES.CONV_C2C,
  }
}

function readPercent(raw: unknown) {
  if (typeof raw === 'number' && Number.isFinite(raw)) return Math.max(0, Math.min(1, raw))
  if (raw && typeof raw === 'object') {
    const n = Number((raw as { percent?: number; progress?: number }).percent ?? (raw as { progress?: number }).progress)
    if (Number.isFinite(n)) return Math.max(0, Math.min(1, n > 1 ? n / 100 : n))
  }
  return 0
}

function makePending(
  conversationID: string,
  type: string,
  payload: ChatMessage['payload'],
  localPreview?: string,
): ChatMessage {
  const { to } = convTarget(conversationID)
  return {
    ID: `local_${Date.now()}_${Math.random().toString(36).slice(2, 8)}`,
    type,
    conversationID,
    from: currentImUserId() || '',
    to,
    flow: 'out',
    time: Math.floor(Date.now() / 1000),
    status: 'unSend',
    progress: 0,
    localPreview,
    payload,
  }
}

async function sendCreated(
  pending: ChatMessage,
  created: unknown,
  hooks?: MediaSendHooks,
  fallback?: () => Promise<MediaSendResult>,
): Promise<MediaSendResult> {
  const chat = getChat()
  try {
    const res = await chat.sendMessage(created)
    playSound('newmsg')
    return {
      localId: pending.ID,
      sent: res?.data?.message as ChatMessage,
      localPreview: pending.localPreview,
    }
  } catch (e) {
    if (fallback && isNativeSizeLimitError(e)) {
      return fallback()
    }
    hooks?.onFailed?.(pending.ID, formatSendError(e))
    throw new Error(formatSendError(e))
  }
}

async function sendNativeOrCos(
  pending: ChatMessage,
  createNative: () => unknown,
  goCos: () => Promise<MediaSendResult>,
  hooks?: MediaSendHooks,
) {
  try {
    const created = createNative()
    return await sendCreated(pending, created, hooks, goCos)
  } catch (e) {
    if (isNativeSizeLimitError(e)) return goCos()
    throw e instanceof Error ? e : new Error(formatSendError(e))
  }
}

async function sendViaCos(
  conversationID: string,
  file: File,
  pending: ChatMessage,
  options: { isVideo: boolean; durationSeconds?: number },
  hooks?: MediaSendHooks,
): Promise<MediaSendResult> {
  try {
    const settings = await getChatAttachmentUploadSettings()
    if (!settings || settings.enabled === false) throw new Error('文件上传功能已关闭')
    const maxMb = Number(settings.max_size_mb || DEFAULT_MAX_ATTACHMENT_MB) || DEFAULT_MAX_ATTACHMENT_MB
    if (file.size > maxMb * 1024 * 1024) {
      throw new Error(`文件超过最大限制，最大 ${maxMb}MB`)
    }
    const isVideo = options.isVideo || isVideoFileName(file.name, file.type)
    const presign = await uploadChatAttachment(file, conversationID, {
      isVideo,
      onProgress: (p) => hooks?.onProgress(pending.ID, p),
    })
    const info = {
      fileID: presign.file_id,
      fileName: presign.file_name || file.name,
      fileSize: file.size,
      contentType: presign.content_type || file.type || 'application/octet-stream',
      isVideo: presign.is_video || isVideo,
      downloadURL: presign.download_url || '',
      previewURL: presign.preview_url || presign.download_url || '',
      durationSeconds: options.durationSeconds,
    }
    const chat = getChat()
    const { to, conversationType, isGroup } = convTarget(conversationID)
    const created = chat.createCustomMessage({
      to,
      conversationType,
      payload: {
        data: JSON.stringify(buildLargeAttachmentPayload(info)),
        description: largeAttachmentDescription(info),
        extension: 'chat_large_attachment',
      },
      needReadReceipt: !isGroup,
    })
    hooks?.onProgress(pending.ID, 1)
    return sendCreated(pending, created, hooks)
  } catch (e) {
    const reason = explainChatAttachmentError(e)
    console.error('[cos-upload]', e)
    hooks?.onFailed?.(pending.ID, reason)
    throw new Error(reason)
  }
}

export async function sendImageMessage(
  conversationID: string,
  file: File,
  hooks?: MediaSendHooks,
) {
  const preview = URL.createObjectURL(file)
  const pending = makePending(
    conversationID,
    'TIMImageElem',
    {
      imageInfoArray: [{ url: preview, sizeType: 1 }],
      fileName: file.name,
      fileSize: file.size,
    },
    preview,
  )
  hooks?.onPending(pending)
  void rememberOutgoingFile(pending.ID, file)
  const chat = getChat()
  const { to, conversationType, isGroup } = convTarget(conversationID)
  const created = chat.createImageMessage({
    to,
    conversationType,
    payload: { file },
    needReadReceipt: !isGroup,
    onProgress: (p: unknown) => hooks?.onProgress(pending.ID, readPercent(p)),
  })
  return sendCreated(pending, created, hooks)
}

export async function sendFileMessage(
  conversationID: string,
  file: File,
  hooks?: MediaSendHooks,
) {
  const asVideo = isVideoFileName(file.name, file.type)
  const duration = asVideo ? await readVideoDuration(file) : 0
  const pending = makePending(conversationID, asVideo ? 'TIMVideoFileElem' : 'TIMFileElem', {
    fileName: file.name,
    fileSize: file.size,
    ...(asVideo
      ? {
          remoteVideoUrl: '',
          videoUrl: '',
          videoSecond: duration,
        }
      : {}),
  })
  hooks?.onPending(pending)
  void rememberOutgoingFile(pending.ID, file)
  const goCos = () =>
    sendViaCos(conversationID, file, pending, { isVideo: asVideo, durationSeconds: duration }, hooks)
  if (shouldRouteToCOS(file.size, duration, asVideo)) {
    return goCos()
  }
  const chat = getChat()
  const { to, conversationType, isGroup } = convTarget(conversationID)
  return sendNativeOrCos(
    pending,
    () =>
      asVideo
        ? chat.createVideoMessage({
            to,
            conversationType,
            payload: { file },
            needReadReceipt: !isGroup,
            onProgress: (p: unknown) => hooks?.onProgress(pending.ID, readPercent(p)),
          })
        : chat.createFileMessage({
            to,
            conversationType,
            payload: { file },
            needReadReceipt: !isGroup,
            onProgress: (p: unknown) => hooks?.onProgress(pending.ID, readPercent(p)),
          }),
    goCos,
    hooks,
  )
}

export async function sendVideoMessage(
  conversationID: string,
  file: File,
  hooks?: MediaSendHooks,
) {
  const preview = URL.createObjectURL(file)
  const duration = await readVideoDuration(file)
  const pending = makePending(
    conversationID,
    'TIMVideoFileElem',
    {
      remoteVideoUrl: preview,
      videoUrl: preview,
      fileName: file.name,
      fileSize: file.size,
      videoSecond: duration,
    },
    preview,
  )
  hooks?.onPending(pending)
  void rememberOutgoingFile(pending.ID, file)
  const goCos = () =>
    sendViaCos(conversationID, file, pending, { isVideo: true, durationSeconds: duration }, hooks)
  if (shouldRouteToCOS(file.size, duration, true)) {
    return goCos()
  }
  const chat = getChat()
  const { to, conversationType, isGroup } = convTarget(conversationID)
  return sendNativeOrCos(
    pending,
    () =>
      chat.createVideoMessage({
        to,
        conversationType,
        payload: { file },
        needReadReceipt: !isGroup,
        onProgress: (p: unknown) => hooks?.onProgress(pending.ID, readPercent(p)),
      }),
    goCos,
    hooks,
  )
}

export async function sendSoundMessage(
  conversationID: string,
  file: File,
  second: number,
  hooks?: MediaSendHooks,
) {
  const preview = URL.createObjectURL(file)
  const sec = Math.max(1, Math.round(second))
  const pending = makePending(
    conversationID,
    'TIMSoundElem',
    { url: preview, second: sec },
    preview,
  )
  hooks?.onPending(pending)
  void rememberOutgoingFile(pending.ID, file)
  const chat = getChat()
  const { to, conversationType, isGroup } = convTarget(conversationID)
  const create =
    typeof chat.createAudioMessage === 'function'
      ? chat.createAudioMessage.bind(chat)
      : chat.createSoundMessage?.bind(chat)
  if (!create) throw new Error('当前 SDK 不支持语音消息')
  const created = create({
    to,
    conversationType,
    payload: { file, second: sec },
    needReadReceipt: !isGroup,
    onProgress: (p: unknown) => hooks?.onProgress(pending.ID, readPercent(p)),
  })
  try {
    if (created?.payload) created.payload.second = sec
  } catch {
    /* ignore */
  }
  return sendCreated(pending, created, hooks)
}

export function blobToAudioFile(blob: Blob, mime: string, _durationSec?: number) {
  if (blob instanceof File && blob.name) return blob
  const type = mime || blob.type || 'audio/wav'
  const ext = type.includes('wav')
    ? 'wav'
    : type.includes('mpeg') || type.includes('mp3')
      ? 'mp3'
      : type.includes('ogg')
        ? 'ogg'
        : type.includes('mp4')
          ? 'm4a'
          : 'webm'
  return new File([blob], `voice_${Date.now()}.${ext}`, { type })
}
