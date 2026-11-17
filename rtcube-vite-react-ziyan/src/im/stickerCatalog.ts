/**
 * 动图：src/assets/imagegif（优先）+ chatgift / chatgit 兜底
 * 电报贴纸：public/stickers/*.tgs（勿用 import.meta.glob 扫 src 下 .tgs，会白屏）
 */
export const GIFT_BUSINESS_ID = 'gift_v1'
export const STICKER_BUSINESS_ID = 'sticker_v1'

export type MediaAsset = {
  id: string
  name: string
  kind: 'gift' | 'sticker'
  url: string
  ext: string
  animated: boolean
}

const imageGifModules = import.meta.glob('../assets/imagegif/*.{png,webp,gif,svg,json}', {
  eager: true,
  query: '?url',
  import: 'default',
}) as Record<string, string>

const fallbackGiftModules = {
  ...import.meta.glob('../assets/chatgit/*.{png,webp,gif,svg,json}', {
    eager: true,
    query: '?url',
    import: 'default',
  }),
  ...import.meta.glob('../assets/chatgift/*.{png,webp,gif,svg,json}', {
    eager: true,
    query: '?url',
    import: 'default',
  }),
} as Record<string, string>

const stickerImgModules = import.meta.glob('../assets/stickers/**/*.{json,png,webp,gif,svg}', {
  eager: true,
  query: '?url',
  import: 'default',
}) as Record<string, string>

const PUBLIC_TGS = ['1f600', '1f601', '1f602', '1f603', '1f604', '1f605', '1f606']

const WORD_ORDER: Record<string, number> = {
  one: 1,
  two: 2,
  three: 3,
  four: 4,
  five: 5,
  six: 6,
  seven: 7,
  eight: 8,
  nine: 9,
  ten: 10,
  eleven: 11,
  twelve: 12,
  thirteen: 13,
  fourteen: 14,
  fifteen: 15,
  sixteen: 16,
  seventeen: 17,
  eighteen: 18,
  nineteen: 19,
  twenty: 20,
  twenty_one: 21,
  twenty_two: 22,
  twenty_three: 23,
  twenty_four: 24,
  twenty_five: 25,
  twenty_six: 26,
  twenty_seven: 27,
  twenty_eight: 28,
  twenty_nine: 29,
  thirty: 30,
  thirty_one: 31,
  'thirty-one': 31,
  thirty_two: 32,
  thirty_three: 33,
  thirty_four: 34,
  thirty_five: 35,
  'thirty_five@': 35,
  thirty_six: 36,
  thirty_seven: 37,
  thirty_eight: 38,
  thirty_nine: 39,
  forty: 40,
  forty_one: 41,
  forty_two: 42,
  forty_three: 43,
  forty_four: 44,
  forty_five: 45,
  forty_six: 46,
  forty_seven: 47,
  forty_eight: 48,
  forty_nine: 49,
  fifty: 50,
  fifity: 50,
  fifty_one: 51,
  fifity_one: 51,
  fifty_two: 52,
  fifity_two: 52,
  fifty_three: 53,
  fifity_three: 53,
  fifty_four: 54,
  fifity_four: 54,
}

function baseName(path: string) {
  const file = path.split('/').pop() || path
  return file.replace(/\.[^.]+$/, '')
}

function giftSortKey(id: string) {
  const n = WORD_ORDER[id]
  if (typeof n === 'number') return n
  const digits = id.match(/\d+/)
  if (digits) return Number(digits[0])
  return 9999
}

function toAssets(modules: Record<string, string>, kind: 'gift' | 'sticker'): MediaAsset[] {
  return Object.entries(modules)
    .map(([path, url]) => {
      const id = baseName(path)
      const ext = (path.split('.').pop() || '').toLowerCase()
      const order = giftSortKey(id)
      return {
        id,
        name: kind === 'gift' && order < 9999 ? `动图${order}` : id,
        kind,
        url,
        ext,
        animated: ext === 'gif' || ext === 'tgs' || ext === 'webp' || ext === 'json',
      }
    })
    .sort((a, b) => {
      const da = giftSortKey(a.id)
      const db = giftSortKey(b.id)
      if (da !== db) return da - db
      return a.id.localeCompare(b.id)
    })
}

function publicTgsAssets(): MediaAsset[] {
  const base = (import.meta.env.BASE_URL || '/').replace(/\/?$/, '/')
  return PUBLIC_TGS.map((name) => ({
    id: `sticker_${name}`,
    name,
    kind: 'sticker' as const,
    url: `${base}stickers/${name}.tgs`,
    ext: 'tgs',
    animated: true,
  }))
}

const imageGifts = toAssets(imageGifModules, 'gift')
const fallbackGifts = toAssets(fallbackGiftModules, 'gift')

export const GIFT_LIST = imageGifts.length ? imageGifts : fallbackGifts
export const STICKER_LIST = [...toAssets(stickerImgModules, 'sticker'), ...publicTgsAssets()]

export function findGiftById(id: string) {
  const raw = String(id || '')
  const key = raw.replace(/^gift_/, '')
  return GIFT_LIST.find((g) => g.id === raw || g.id === key || g.name === raw) || null
}

export function parseGiftPayload(raw: unknown): { giftId: string; giftName: string; url?: string } | null {
  try {
    const data = typeof raw === 'string' ? JSON.parse(raw) : raw
    if (!data || (data as { businessID?: string }).businessID !== GIFT_BUSINESS_ID) return null
    const d = data as { giftId?: string; giftName?: string; url?: string; giftUrl?: string }
    return {
      giftId: d.giftId || '',
      giftName: d.giftName || '动图',
      url: d.giftUrl || d.url,
    }
  } catch {
    return null
  }
}

export function resolveGiftDisplayUrl(payload: { giftId: string; url?: string }) {
  const local = findGiftById(payload.giftId)
  if (local?.url) return local.url
  const remote = payload.url || ''
  if (remote.startsWith('http') || remote.startsWith('data:') || remote.startsWith('/')) return remote
  return remote
}

export function parseStickerPayload(raw: unknown): { stickerId: string; stickerName: string } | null {
  try {
    const data = typeof raw === 'string' ? JSON.parse(raw) : raw
    if (!data || (data as { businessID?: string }).businessID !== STICKER_BUSINESS_ID) return null
    const d = data as { stickerId?: string; stickerName?: string }
    return { stickerId: d.stickerId || '', stickerName: d.stickerName || '贴纸' }
  } catch {
    return null
  }
}

export function buildGiftPayload(asset: MediaAsset) {
  return {
    businessID: GIFT_BUSINESS_ID,
    version: 1,
    giftId: asset.id,
    giftName: asset.name,
  }
}

export function buildStickerPayload(asset: MediaAsset) {
  return {
    businessID: STICKER_BUSINESS_ID,
    version: 1,
    stickerId: asset.id,
    stickerName: asset.name,
    format: asset.ext === 'tgs' ? 'tgs' : asset.ext === 'json' ? 'lottie' : 'image',
  }
}
