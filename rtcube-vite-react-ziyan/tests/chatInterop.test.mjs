import assert from 'node:assert/strict'
import { test, after } from 'node:test'
import { build } from 'esbuild'
import { mkdtemp, rm } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import path from 'node:path'
import { pathToFileURL } from 'node:url'

const tmp = await mkdtemp(path.join(tmpdir(), 'ziyan-interop-'))
after(() => rm(tmp, { recursive: true, force: true }))
async function load(name) {
  const outfile = path.join(tmp, `${name}.mjs`)
  await build({
    entryPoints: [`src/im/${name}.ts`], outfile, bundle: true, platform: 'node', format: 'esm',
    plugins: [{ name: 'mock-chat', setup(b) {
      b.onResolve({ filter: /^\.\/ImClient$/ }, () => ({ path: 'chat', namespace: 'mock' }))
      b.onLoad({ filter: /.*/, namespace: 'mock' }, () => ({ contents: `
        export const getChat = () => globalThis.chatMock;
        export const TencentCloudChat = {EVENT:{ MESSAGE_READ_BY_PEER:'legacy', MESSAGE_READ_RECEIPT_RECEIVED:'receipt' }};
        export const normalizeChatMessage = x => x;
        export const fetchMessageList = (...args) => globalThis.chatMock.fetchMessageList(...args);
      ` }))
    } }],
  })
  return import(pathToFileURL(outfile).href)
}
const quote = await load('quote')
const emoji = await load('emojiWire')
const receipts = await load('readReceipt')
const navigation = await load('messageNavigation')
const message = (id, patch = {}) => ({ ID: id, conversationID: 'C2Cpeer', type: 'TIMTextElem', from: 'me', to: 'peer', flow: 'out', time: 123, ...patch })

test('web quote is readable by AtomicX and native numeric message types', () => {
  const encoded = quote.encodeQuote(message('original', { sequence: 42, type: 'TIMImageElem' }), '[图片]')
  const data = JSON.parse(encoded)
  assert.equal(data.atomicx_quote_v1.msgID, 'original')
  assert.equal(data.atomicx_quote_v1.msgAbstract, '[图片]')
  assert.equal(data.messageReply.messageType, 3)
  assert.equal(data.messageReply.messageSequence, 42)
  assert.equal(quote.parseQuoteFromMessage({ cloudCustomData: encoded }).messageID, 'original')
})

test('reads iOS-only quotes and falls back when web envelope is malformed', () => {
  const cloudCustomData = JSON.stringify({ messageReply: { messageID: null }, atomicx_quote_v1: { msgID: 'ios', msgSender: '用户', msgAbstract: '原消息' } })
  assert.deepEqual(quote.parseQuoteFromMessage({ cloudCustomData }), { messageID: 'ios', messageSender: '用户', messageAbstract: '原消息', messageTime: undefined, messageSequence: undefined })
  for (const invalid of ['', 'null', '[]', '{', '{"messageReply":{"messageID":{}}}']) {
    assert.equal(quote.parseQuoteFromMessage({ cloudCustomData: invalid }), null)
  }
})

test('known tags become native emoji without changing unknown tags or normal text', () => {
  assert.equal(emoji.emojiWireText('你好[TUIEmoji_Smile][TUIEmoji_Heart] @所有人'), '你好😊❤️ @所有人')
  assert.equal(emoji.emojiWireText('😀 [TUIEmoji_Unknown] code'), '😀 [TUIEmoji_Unknown] code')
  for (const [tag, glyph] of emoji.TAGS_AND_GLYPHS) assert.equal(emoji.emojiWireText(tag), glyph)
})

test('hydrates SDK returned read states even if input objects are not mutated', async () => {
  globalThis.chatMock = {
    getLoginUser: () => 'me',
    getMessageReadReceiptList: async () => ({ data: { messageList: [message('sent', { readReceiptInfo: { isPeerRead: true } })] } }),
  }
  const original = message('sent', { needReadReceipt: true, isPeerRead: false })
  const [hydrated] = await receipts.hydratePeerRead([original])
  assert.equal(original.isPeerRead, false)
  assert.equal(hydrated.isPeerRead, true)
})

test('SDK cache read getters survive failed/unavailable paid receipt service', async () => {
  globalThis.chatMock = { getLoginUser: () => 'me', findMessage: () => ({ get isPeerRead() { return true } }), getMessageReadReceiptList: async () => { throw Error('unavailable') } }
  assert.equal((await receipts.hydratePeerRead([message('m')]))[0].isPeerRead, true)
})

test('false and group receipt events never mark outgoing C2C messages read', () => {
  const listeners = new Map()
  globalThis.chatMock = { on: (name, cb) => listeners.set(name, cb), off: (name) => listeners.delete(name) }
  let updates
  const off = receipts.onMessageReadByPeer((x) => { updates = x })
  const msgs = [message('sent')]
  listeners.get('receipt')({ data: [{ messageID: 'sent', userID: 'peer', isPeerRead: false }] })
  assert.notEqual(receipts.applyPeerReadUpdates(msgs, updates)[0].isPeerRead, true)
  listeners.get('receipt')({ data: [{ messageID: 'sent', userID: 'other', isPeerRead: true }] })
  assert.notEqual(receipts.applyPeerReadUpdates(msgs, updates)[0].isPeerRead, true)
  listeners.get('receipt')({ data: [{ messageID: 'sent', groupID: 'group', readCount: 2 }] })
  assert.notEqual(receipts.applyPeerReadUpdates(msgs, updates)[0].isPeerRead, true)
  listeners.get('legacy')({ data: [message('sent')] })
  assert.equal(receipts.applyPeerReadUpdates(msgs, updates)[0].isPeerRead, true)
  off()
  assert.equal(listeners.size, 0)
})

test('sends receipts per conversation in batches of 30, excluding unrequested messages', async () => {
  const batches = []
  globalThis.chatMock = { getLoginUser: () => 'me', sendMessageReadReceipt: async (list) => { batches.push(list) } }
  const incoming = Array.from({ length: 65 }, (_, i) => message(`in${i}`, { from: 'peer', flow: 'in', needReadReceipt: true }))
  incoming.push(message('other', { conversationID: 'C2Cother', from: 'other', flow: 'in', needReadReceipt: true }))
  incoming.push(message('no-receipt', { from: 'peer', flow: 'in' }))
  await receipts.sendReadReceipts(incoming)
  assert.deepEqual(batches.map((b) => b.length), [30, 30, 5, 1])
  for (const batch of batches) assert.equal(new Set(batch.map((m) => m.conversationID)).size, 1)
})

test('quote navigation finds targets beyond the 400-message display window', async () => {
  let requests = 0
  globalThis.chatMock = { fetchMessageList: async (_conv, cursor) => {
    const page = Number(cursor || 0)
    requests++
    return { messages: Array.from({ length: 20 }, (_, i) => message(`page${page}-${i}`)), nextReqMessageID: String(page + 1), isCompleted: page === 30 }
  } }
  const context = await navigation.findMessageContext('C2Cpeer', 'page25-7', undefined, () => false)
  assert.equal(requests, 26)
  assert.ok(context.some((m) => m.ID === 'page25-7'))
  assert.ok(context.length <= 40)
})

test('jump uses quote sequence and ignores stale result after conversation switch', async () => {
  const calls = []
  globalThis.chatMock = { getMessageListHopping: async (opts) => {
    calls.push(opts)
    return { data: { messageList: [message('target', { conversationID: 'GROUPg' })] } }
  } }
  assert.equal((await navigation.findMessageContext('GROUPg', 'target', { messageSequence: 42 }, () => false))[0].ID, 'target')
  assert.deepEqual(calls.map((c) => [c.sequence, c.direction]), [[42, 0], [42, 1]])
  let cancelled = false
  globalThis.chatMock = { fetchMessageList: async () => { cancelled = true; return { messages: [message('target')] } } }
  assert.equal(await navigation.findMessageContext('C2Cpeer', 'target', undefined, () => cancelled), null)
})

test('missing targets and repeated history cursors terminate cleanly', async () => {
  let calls = 0
  globalThis.chatMock = { fetchMessageList: async () => { calls++; return { messages: [], nextReqMessageID: 'same', isCompleted: false } } }
  assert.equal(await navigation.findMessageContext('C2Cpeer', 'missing', undefined, () => false), null)
  assert.equal(calls, 2)
})
