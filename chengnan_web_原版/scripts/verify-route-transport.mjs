import assert from 'node:assert/strict';
import { test } from 'node:test';
import { build } from 'esbuild';
import axios from 'axios';
const compiled = await build({
  stdin: { contents: `export * from './src/api/index';`, resolveDir: process.cwd(), loader: 'ts' },
  bundle: true, write: false, platform: 'browser', format: 'esm', tsconfig: 'tsconfig.json',
  plugins: [{ name: 'transport-fixtures', setup(b) {
    const mocks = {
      axios: 'export default globalThis.fixtureAxios;',
      '@/config/runtime': `export const CHAT_API_BASE_URL='https://api.ruizhihe.com/api/v1/chat';export const CHAT_API_TIMEOUT_MS=1000;export const DYNAMIC_API_ROUTING=true;`,
      '@/features/auth/device': `export const getOrCreateDeviceID=()=>'test-device';`,
    };
    b.onResolve({ filter: /.*/ }, a => a.path in mocks ? { path: a.path, namespace: 'fixture' } : undefined);
    b.onLoad({ filter: /.*/, namespace: 'fixture' }, a => ({ contents: mocks[a.path] }));
  } }],
});
const origins = ['https://api.ruizhihe.com', 'https://api.prfgsu.com', 'https://api.uktjwg.cn'];
let fixtureID = 0;
async function fixture(adapter, options = {}) {
  const values = new Map(options.cold ? [] : [['rui.web.validated_domain_pool.r2', JSON.stringify({ document: { version: 0, ttl_seconds: 3600, api_origins: origins }, active: origins[0], fetchedAt: Date.now() })]]);
  globalThis.localStorage = { getItem: k => values.get(k) || null, setItem: (k, v) => values.set(k, v) };
  globalThis.fixtureAxios = axios;
  const calls = [];
  axios.defaults.adapter = async config => { calls.push({ ...config, headers: { ...config.headers } }); return adapter(config, calls.length); };
  globalThis.fetch = options.fetch || (async () => { throw new Error('Unexpected bootstrap before TTL'); });
  const api = await import('data:text/javascript;base64,' + Buffer.from(compiled.outputFiles[0].text + `\n// fixture ${++fixtureID}`).toString('base64'));
  api.setChatAuthToken('token-A', 'user-A');
  return { api, calls };
}
const success = (config, data = { im_user_id: 'user-A' }) => ({ data: { code: 0, data }, status: 200, statusText: 'OK', headers: {}, config });

test('pending cold bootstrap does not block GET or POST and stays single-flight', async () => {
  let release, timer, bootstrapCalls = 0;
  const pending = new Promise(resolve => { release = resolve; });
  const {api, calls} = await fixture(config => success(config), {
    cold: true,
    fetch: () => { bootstrapCalls++; return pending; },
  });
  try {
    await Promise.race([
      Promise.all([api.getProfile({fresh: true}), api.searchUsersExact('test'), api.updateProfile({nickname: 'explicit-action'})]),
      new Promise((_, reject) => { timer = setTimeout(() => reject(new Error('business request waited for bootstrap')), 250); }),
    ]);
    assert.equal(bootstrapCalls, 1);
    assert.equal(calls.length, 3);
    assert.ok(calls.every(c => c.baseURL === origins[0] + '/api/v1/chat'));
    assert.equal(calls.filter(c => c.method === 'post').length, 1);
  } finally {
    clearTimeout(timer);
    release(new Response(JSON.stringify({code: 0, data: {version: 1, ttl_seconds: 60, api_origins: origins}})));
    await new Promise(resolve => setTimeout(resolve, 0));
  }
});
function reject(config, status, data) {
  const e = new Error(status ? `HTTP ${status}` : 'Network Error');
  e.isAxiosError = true; e.config = config;
  if (status) e.response = { status, data, config, headers: {} };
  throw e;
}
test('actual Vue GET recovers once and keeps its session token', async () => {
  const { api, calls } = await fixture((config, n) => n === 1 ? reject(config) : success(config));
  assert.equal((await api.getProfile({ fresh: true })).im_user_id, 'user-A');
  assert.equal(calls.length, 2);
  assert.deepEqual(calls.map(c => c.baseURL), origins.slice(0, 2).map(o => o + '/api/v1/chat'));
  assert.ok(calls.every(c => c.headers.Authorization === 'Bearer token-A'));
});
test('actual Vue GET gateway errors recover once within one deadline', async () => {
  const { api, calls } = await fixture((config, n) => n === 1 ? reject(config, 503, '<html>gateway</html>') : success(config));
  await api.getProfile({ fresh: true });
  assert.equal(calls.length, 2);
  assert.equal(calls[0]._deadline, calls[1]._deadline);
});
for (const status of [400, 403, 413, 429, 500]) test(`Vue GET HTTP${status} keeps its route`, async () => {
  const { api, calls } = await fixture(config => reject(config, status, { code: status, message: 'rejected' }));
  await assert.rejects(api.getProfile({ fresh: true })); assert.equal(calls.length, 1);
});
test('Vue business failure does not switch routes', async () => {
  const { api, calls } = await fixture(config => reject(config, 503, { code: 50001, message: 'application failure' }));
  await assert.rejects(api.getProfile({ fresh: true })); assert.equal(calls.length, 1);
});
for (const status of [undefined, 503]) test(`Vue POST ${status || 'network error'} is never replayed`, async () => {
  const { api, calls } = await fixture(config => reject(config, status, '<html>gateway</html>'));
  await assert.rejects(api.updateProfile({ nickname: 'test' })); assert.equal(calls.length, 1);
});
test('Vue login POST is never replayed on a route failure', async () => {
  const { api, calls } = await fixture(config => reject(config));
  await assert.rejects(api.login({ phone: 'test', password: 'test' })); assert.equal(calls.length, 1);
});
test('two failed GETs stop recovery', async () => {
  const { api, calls } = await fixture(config => reject(config));
  await assert.rejects(api.getProfile({ fresh: true })); assert.equal(calls.length, 2);
});
test('account change during a failed read prevents a new-account replay', async () => {
  let apiRef;
  const { api, calls } = await fixture(config => { apiRef.setChatAuthToken('token-B', 'user-B'); return reject(config); });
  apiRef = api;
  await assert.rejects(api.getProfile({ fresh: true })); assert.equal(calls.length, 1);
});
test('late successful response cannot be consumed by another account', async () => {
  let apiRef;
  const { api, calls } = await fixture(config => { apiRef.setChatAuthToken('token-B', 'user-B'); return success(config); });
  apiRef = api;
  await assert.rejects(api.getProfile({ fresh: true }), /账号已切换/); assert.equal(calls.length, 1);
});
test('existing 401 refresh remains coalesced without changing origin', async () => {
  const { api, calls } = await fixture(config => config.headers.Authorization === 'Bearer token-A'
    ? reject(config, 401, { code: 401, message: 'expired' }) : success(config));
  let refreshes = 0;
  api.setChatUnauthorizedHandler(async () => { refreshes++; api.setChatAuthToken('token-refreshed', 'user-A'); return true; });
  await Promise.all([api.getProfile({ fresh: true }), api.searchUsersExact('test')]);
  assert.equal(refreshes, 1);
  assert.ok(calls.every(c => c.baseURL === origins[0] + '/api/v1/chat'));
  assert.equal(calls.filter(c => c.headers.Authorization === 'Bearer token-refreshed').length, 2);
});

test('a failed Vue write only checks a future route; the next explicit write can use it', async () => {
  const {api,calls}=await fixture((config,n)=>n===1?reject(config):success(config))
  const probes=[]
  globalThis.fetch=async(url,init)=>{probes.push({url,init});return new Response(JSON.stringify({code:0,data:{status:'ok'}}))}
  await assert.rejects(api.updateProfile({nickname:'first-action'}))
  assert.equal(calls.length,1)
  await api.updateProfile({nickname:'second-action'})
  assert.equal(calls.length,2);assert.equal(calls[1].baseURL,origins[1]+'/api/v1/chat')
  assert.equal(probes.length,1);assert.equal(probes[0].init.headers,undefined)
})

test('a pending real recovery still delays the next explicit write', async () => {
  let release;
  const {api,calls}=await fixture((config,n)=>n===1?reject(config):success(config), {
    fetch: () => new Promise(resolve => { release=resolve; }),
  });
  await assert.rejects(api.updateProfile({nickname:'first-action'}));
  const next=api.updateProfile({nickname:'second-action'});
  await new Promise(resolve=>setTimeout(resolve,10));
  assert.equal(calls.length,1,'no business write before pending recovery finishes');
  release(new Response(JSON.stringify({code:0,data:{status:'ok'}})));
  await next;
  assert.equal(calls.length,2);
  assert.equal(calls[1].baseURL,origins[1]+'/api/v1/chat');
});

test('a stalled real recovery cannot extend the next write beyond its total deadline', async () => {
  let release;
  const {api,calls}=await fixture((config,n)=>n===1?reject(config):success(config), {
    fetch: () => new Promise(resolve => { release=resolve; }),
  });
  await assert.rejects(api.updateProfile({nickname:'first-action'}));
  try {
    await assert.rejects(api.updateProfile({nickname:'second-action'}), /请求超时/);
    assert.equal(calls.length,1,'expired request never reaches the transport');
  } finally {
    release(new Response(JSON.stringify({code:0,data:{status:'ok'}})));
    await new Promise(resolve=>setTimeout(resolve,0));
  }
});
