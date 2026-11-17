import assert from 'node:assert/strict'
import { test } from 'node:test'
import { build } from 'esbuild'
import path from 'node:path'
const root = process.cwd()
const bundle = await build({
  stdin: { contents: `export * from './src/api/chatHttp'; export {routedFetch} from './src/api/transport'; export * as auth from './src/api/auth'; export * as search from './src/api/search'; export {uploadAvatarFile} from './src/api/media';`, resolveDir: root, loader: 'ts' },
  bundle: true, write: false, platform: 'browser', format: 'esm', tsconfig: 'tsconfig.app.json',
  define: { 'import.meta.env.DEV': 'false', 'import.meta.env.VITE_CHAT_API_BASE_URL': 'undefined', 'import.meta.env.VITE_API_BASE_URL': 'undefined' },
  plugins: [{ name: 'session-fixture', setup(b) {
    b.onResolve({ filter: /^\.\/session$/ }, () => ({ path: 'session', namespace: 'fixture' }))
    b.onLoad({ filter: /.*/, namespace: 'fixture' }, () => ({ contents: `export const getSessionContext=()=>globalThis.fixtureSession; export const getToken=()=>globalThis.fixtureToken;` }))
  } }],
})
let id = 0
const origins = ['https://api.ruizhihe.com', 'https://api.prfgsu.com', 'https://api.uktjwg.cn']
const ok = data => new Response(JSON.stringify({ code: 0, data }), { headers: { 'Content-Type': 'application/json' } })
async function fixture(fetcher, cached = true, healthyRecovery = false) {
  const values = new Map()
  globalThis.fixtureSession = 'session-A'
  globalThis.fixtureToken = 'test-session-token'
  if (cached) values.set('rui.web.validated_domain_pool.r2', JSON.stringify({ document: { version: 0, ttl_seconds: 3600, api_origins: origins }, active: origins[0], fetchedAt: Date.now() }))
  globalThis.localStorage = { getItem: k => values.get(k) || null, setItem: (k, v) => values.set(k, v) }
  const calls = [], probes = []
  globalThis.fetch = async (url, init) => {
    if (String(url).endsWith('/api/v1/chat/ping')) {
      probes.push({url, init})
      if (healthyRecovery) return ok({status:'ok'})
      throw new TypeError('fixture health unavailable')
    }
    calls.push({ url, init }); return fetcher(url, init, calls.length)
  }
  const api = await import('data:text/javascript;base64,' + Buffer.from(bundle.outputFiles[0].text + `\n// fixture ${++id}`).toString('base64'))
  return { api, calls, probes }
}
for (const method of ['POST', 'DELETE']) {
  for (const failure of ['network', '503']) test(`${method} ${failure} sends exactly one write and retains the active route`, async () => {
    const { api, calls } = await fixture(() => {
      if (failure === 'network') throw new TypeError('connection lost after send')
      return new Response('<html>gateway</html>', { status: 503 })
    })
    await assert.rejects(method === 'POST' ? api.chatPost('/action', { once: true }) : api.chatDelete('/action'))
    assert.equal(calls.length, 1)
    assert.equal(api.getActiveApiOrigin(), origins[0])
  })
}
for (const status of [400, 401, 403, 404, 413, 429, 500]) test(`GET HTTP${status} never switches routes`, async () => {
  const { api, calls } = await fixture(() => new Response(JSON.stringify({ code: status, message: 'rejected' }), { status }))
  await assert.rejects(api.chatGet('/read'))
  assert.equal(calls.length, 1)
  assert.equal(api.getActiveApiOrigin(), origins[0])
})
test('successful transport with application error never retries', async () => {
  const { api, calls } = await fixture(() => new Response(JSON.stringify({ code: 50001, message: 'application error' })))
  await assert.rejects(api.chatGet('/read'), /application error/)
  assert.equal(calls.length, 1)
})
test('503 with a business envelope never retries', async () => {
  const { api, calls } = await fixture(() => new Response(JSON.stringify({ code: 50001, message: 'business failed' }), { status: 503 }))
  await assert.rejects(api.chatGet('/read')); assert.equal(calls.length, 1)
})
for (const failure of ['network', '503']) test(`GET ${failure} recovers once using a published origin and the same token`, async () => {
  const { api, calls } = await fixture((_, __, count) => {
    if (count > 1) return ok({ hello: true })
    if (failure === 'network') throw new TypeError('DNS failed')
    return new Response('<html>gateway</html>', { status: 503 })
  })
  assert.deepEqual(await api.chatGet('/read'), { hello: true })
  assert.deepEqual(calls.map(c => new URL(c.url).origin), origins.slice(0, 2))
  assert.ok(calls.every(c => c.init.headers.Authorization === 'Bearer test-session-token'))
  assert.ok(calls.every(c => c.init.redirect === 'error' && c.init.credentials === 'omit'))
  assert.equal(api.getActiveApiOrigin(), origins[1])
})
test('GET performs at most two business requests', async () => {
  const { api, calls } = await fixture(() => { throw new TypeError('DNS failed') })
  await assert.rejects(api.chatGet('/read')); assert.equal(calls.length, 2)
})
test('429 response body failure cannot cause failover', async () => {
  const { api, calls } = await fixture(() => ({ status: 429, text: async () => { throw new TypeError('stream lost') } }))
  await assert.rejects(api.chatGet('/read')); assert.equal(calls.length, 1)
})
test('cancellation never starts an alternate request', async () => {
  const { api, calls } = await fixture(() => ok({}))
  const controller = new AbortController(); controller.abort()
  await assert.rejects(api.routedFetch('/read', { signal: controller.signal }), { name: 'AbortError' })
  assert.equal(calls.length, 0)
})
test('timeout leaves time for one read recovery under the parent deadline', async () => {
  const { api, calls } = await fixture((_, init, count) => {
    if (count > 1) return ok({ recovered: true })
    return new Promise((_, reject) => init.signal.addEventListener('abort', () => reject(new DOMException('timeout', 'AbortError'))))
  })
  const start = Date.now()
  const response = await api.routedFetch('/read', {}, '/api/v1/chat', 80)
  assert.equal((await response.json()).data.recovered, true)
  assert.equal(calls.length, 2); assert.ok(Date.now() - start < 250)
})
test('switching accounts during a failed read cannot replay with another account token', async () => {
  const { api, calls } = await fixture(() => { globalThis.fixtureSession = 'session-B'; globalThis.fixtureToken = 'new-token'; throw new TypeError('old request failed') })
  await assert.rejects(api.chatGet('/read'), /账号已切换/)
  assert.equal(calls.length, 1)
})
test('switching accounts during bootstrap prevents the business request entirely', async () => {
  const { api, calls } = await fixture(() => { globalThis.fixtureSession = 'session-B'; return ok({ version: 0, ttl_seconds: 3600, api_origins: origins }) }, false)
  await assert.rejects(api.chatGet('/read'), /账号已切换/)
  assert.equal(calls.length, 1); assert.match(calls[0].url, /public\/bootstrap/)
})
test('login uses the shared route but never replays an ambiguous POST', async () => {
  const { api, calls } = await fixture(() => { throw new TypeError('login response lost') })
  await assert.rejects(api.auth.loginByPassword('test-account', 'test-password'))
  assert.equal(calls.length, 1); assert.match(calls[0].url, /api\/v1\/chat\/web-login$/)
  assert.equal(calls[0].init.headers.Authorization, undefined)
})
test('search uses the same route recovery as chat', async () => {
  const { api, calls } = await fixture((_, __, count) => count === 1 ? Promise.reject(new TypeError('DNS failed')) : ok({ users: [{ im_user_id: 'test-user' }] }))
  assert.equal((await api.search.searchUsers('test'))[0].im_user_id, 'test-user')
  assert.equal(calls.length, 2)
})
test('avatar upload does not fall back to another POST or COS after an ambiguous write', async () => {
  const { api, calls } = await fixture(() => { throw new TypeError('upload response lost') })
  await assert.rejects(api.uploadAvatarFile(new File(['test'], 'test.png', { type: 'image/png' }), { userId: 'test-user' }))
  assert.equal(calls.length, 1); assert.equal(calls[0].init.method, 'POST')
})
test('legacy API prefix is appended once and its POST is not replayed', async () => {
  const { api, calls } = await fixture(() => new Response('<html>gateway</html>', { status: 503 }))
  const response = await api.routedFetch('/im/sync/group/from/app', { method: 'POST', body: '{}' }, '/api')
  assert.equal(response.status, 503); assert.equal(calls.length, 1)
  assert.equal(calls[0].url, origins[0] + '/api/im/sync/group/from/app')
})

test('a failed write prepares a healthy route for the next explicit user action', async () => {
  const {api,calls,probes} = await fixture((_,__,count) => {
    if (count === 1) throw new TypeError('write response lost')
    return ok({saved:true})
  }, true, true)
  await assert.rejects(api.chatPost('/write', {once:1}))
  assert.equal(calls.length,1)
  assert.equal((await api.chatPost('/write', {once:2})).saved,true)
  assert.equal(calls.length,2)
  assert.deepEqual(calls.map(c=>new URL(c.url).origin),origins.slice(0,2))
  assert.equal(probes.length,1);assert.equal(probes[0].init.headers,undefined)
})
