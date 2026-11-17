import { ensureC2CCanSend } from '@/im/friends'
import { sendCustomJsonMessage, sendTextMessage } from '@/im/ImClient'
import { imageUrlFromMessage, isImageMessage } from '@/im/mediaPayload'
import { sendImageMessage } from '@/im/mediaSend'
import { GIFT_BUSINESS_ID, parseGiftPayload } from '@/im/stickerCatalog'
import type { FavoriteItem } from './favorites'

async function fileFromUrl(url: string, filename: string) {
  const res = await fetch(url)
  if (!res.ok) throw new Error('收藏资源已失效，无法转发')
  const blob = await res.blob()
  return new File([blob], filename, { type: blob.type || 'application/octet-stream' })
}

export async function forwardFavorite(item: FavoriteItem, conversationIDs: string[]) {
  if (!conversationIDs.length) throw new Error('请选择会话')
  for (const convId of conversationIDs) {
    if (convId.startsWith('C2C')) {
      await ensureC2CCanSend(convId.replace(/^C2C/, ''))
    }
    const fakeMsg = { type: item.type, payload: item.payload }
    const gift = parseGiftPayload(item.payload?.data)
    if (gift) {
      await sendCustomJsonMessage(
        convId,
        {
          businessID: GIFT_BUSINESS_ID,
          version: 1,
          giftId: gift.giftId,
          giftName: gift.giftName,
        },
        `[动图] ${gift.giftName}`,
      )
      continue
    }
    if (isImageMessage(fakeMsg)) {
      const url = imageUrlFromMessage(fakeMsg)
      if (url) {
        await sendImageMessage(convId, await fileFromUrl(url, 'favorite.jpg'))
        continue
      }
    }
    const text = String(item.text || '').trim()
    if (!text) throw new Error('这条收藏无法转发')
    await sendTextMessage(convId, text)
  }
}
