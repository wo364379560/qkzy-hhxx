import assert from 'node:assert/strict';
import { test } from 'node:test';
import { readFileSync } from 'node:fs';
import ts from 'typescript';
const compiled = ts.transpileModule(readFileSync(new URL('../src/config/domain-pool.ts', import.meta.url), 'utf8'), {
  compilerOptions: { module: ts.ModuleKind.ES2020, target: ts.ScriptTarget.ES2020 },
}).outputText;
const policy = await import('data:text/javascript;base64,' + Buffer.from(compiled).toString('base64'));
const { createDomainPool, parseBootstrap, boundedCandidates, retryableRouteFailure, DEFAULT_API_ORIGINS, BOOTSTRAP_ORIGINS } = policy;
const doc = (version = 0, origins = [...DEFAULT_API_ORIGINS], ttl = 60) => ({ version, ttl_seconds: ttl, api_origins: origins });
const response = d => new Response(JSON.stringify({ code: 0, data: d }), { headers: { 'Content-Type': 'application/json' } });
const memory = () => { const values = new Map(); return { getItem: k => values.get(k) ?? null, setItem: (k, v) => values.set(k, v) }; };

for (const selected of DEFAULT_API_ORIGINS) test(`late first pool preserves legal selected route ${selected}`, async () => {
  let release;
  const pool = createDomainPool({storage: null, fetch: () => new Promise(resolve => { release = resolve; })});
  const pending = pool.refresh();
  pool.remember(selected, pool.snapshot().revision);
  release(response(doc(1, [DEFAULT_API_ORIGINS[1], DEFAULT_API_ORIGINS[2], DEFAULT_API_ORIGINS[0]])));
  await pending;
  assert.equal(pool.currentOrigin(), selected);
});

test('late first pool reselects when the selected route has been retired', async () => {
  let release;
  const pool = createDomainPool({storage: null, fetch: () => new Promise(resolve => { release = resolve; })});
  const pending = pool.refresh();
  pool.remember(DEFAULT_API_ORIGINS[1], pool.snapshot().revision);
  release(response(doc(1, [DEFAULT_API_ORIGINS[2]])));
  await pending;
  assert.equal(pool.currentOrigin(), DEFAULT_API_ORIGINS[2]);
});

test('three formal APIs and five bootstrap origins are independent', () => {
  assert.deepEqual([...DEFAULT_API_ORIGINS], ['https://api.ruizhihe.com', 'https://api.prfgsu.com', 'https://api.uktjwg.cn']);
  assert.equal(BOOTSTRAP_ORIGINS.length, 5);
  assert.equal(createDomainPool({ storage: null }).currentOrigin(), DEFAULT_API_ORIGINS[0]);
});
test('version zero and nested documents are valid; partial or foreign pools are rejected in full', () => {
  assert.deepEqual(parseBootstrap({ code: 0, data: doc() }), doc());
  assert.deepEqual(parseBootstrap({ code: 0, data: { domain_pool: doc() } }), doc());
  for (const version of [-1, 1.5, '0', Number.MAX_SAFE_INTEGER + 1]) assert.equal(parseBootstrap({ code: 0, data: doc(version) }), null);
  for (const ttl of [0, 59, 86401, '3600']) assert.equal(parseBootstrap({ code: 0, data: doc(0, [...DEFAULT_API_ORIGINS], ttl) }), null);
  for (const origin of ['http://api.prfgsu.com', 'https://127.0.0.1', 'https://[::1]', 'https://api.prfgsu.com:8443', 'https://u:p@api.prfgsu.com', 'https://api.prfgsu.com/api/v1', 'https://api.prfgsu.com?q=x', 'https://api.prfgsu.com#x', 'https://evil.example']) {
    assert.equal(parseBootstrap({ code: 0, data: doc(0, [DEFAULT_API_ORIGINS[0], origin]) }), null);
  }
  assert.equal(parseBootstrap({ code: 200, data: doc() }), null);
});
test('concurrent refresh coalesces and TTL suppresses traffic until expiry', async () => {
  let now = 1000, calls = 0;
  const pool = createDomainPool({ storage: null, now: () => now, fetch: async () => { calls++; return response(doc()); } });
  const p = pool.refresh();
  assert.equal(pool.refresh(), p);
  await Promise.all([p, pool.refresh(), pool.refresh()]);
  assert.equal(calls, 1);
  await pool.refresh(); assert.equal(calls, 1);
  now += 60001; await pool.refresh(); assert.equal(calls, 2);
});

test('failed background refresh keeps defaults and the existing 30-second backoff', async () => {
  let now=1000,calls=0;
  const pool=createDomainPool({storage:null,now:()=>now,fetch:async()=>{calls++;throw new Error('offline');}});
  await pool.refresh();assert.equal(calls,2);
  await pool.refresh();assert.equal(calls,2);
  assert.deepEqual(pool.snapshot().origins,[...DEFAULT_API_ORIGINS]);
  now+=30001;await pool.refresh();assert.equal(calls,4);
});
test('stale and same-version conflicting content cannot overwrite a newer pool', async () => {
  let current = doc(8, [DEFAULT_API_ORIGINS[1]]);
  const pool = createDomainPool({ storage: null, fetch: async () => response(current) });
  await pool.refresh();
  for (const next of [doc(7), doc(8), doc(8, [DEFAULT_API_ORIGINS[1]], 120)]) {
    current = next; await pool.refresh(true); assert.equal(pool.snapshot().document.version, 8);
    assert.deepEqual(pool.snapshot().origins, [DEFAULT_API_ORIGINS[1]]);
  }
  current = doc(9, [BOOTSTRAP_ORIGINS[3]]); await pool.refresh(true);
  assert.deepEqual(pool.snapshot().origins, [BOOTSTRAP_ORIGINS[3]]);
  assert.equal(pool.currentOrigin(), BOOTSTRAP_ORIGINS[3]);
});
test('retired routes and late requests cannot rejoin the published pool', async () => {
  let current = doc(1);
  const pool = createDomainPool({ storage: null, fetch: async () => response(current) });
  await pool.refresh();
  const revision = pool.snapshot().revision;
  pool.remember(DEFAULT_API_ORIGINS[1], revision);
  current = doc(2, [DEFAULT_API_ORIGINS[2]]); await pool.refresh(true);
  pool.remember(DEFAULT_API_ORIGINS[1], revision);
  assert.equal(pool.currentOrigin(), DEFAULT_API_ORIGINS[2]);
  assert.equal(pool.alternative(DEFAULT_API_ORIGINS[2]), undefined);
});
test('cache restores validated membership; refreshing never erases account keys', async () => {
  let now = 1000;
  const storage = memory(); storage.setItem('user-token', 'untouched');
  const pool = createDomainPool({ storage, now: () => now, fetch: async () => response(doc()) });
  await pool.refresh(); pool.remember(DEFAULT_API_ORIGINS[2], pool.snapshot().revision);
  let calls = 0;
  const restored = createDomainPool({ storage, now: () => now, fetch: async () => { calls++; return response(doc()); } });
  assert.equal(restored.currentOrigin(), DEFAULT_API_ORIGINS[2]);
  await restored.refresh(); assert.equal(calls, 0);
  now += 60001; await restored.refresh(); assert.equal(calls, 1);
  assert.equal(storage.getItem('user-token'), 'untouched');
});
test('anonymous bootstrap is capped at two attempts and rotates all five trusted entries', async () => {
  const calls = [];
  const pool = createDomainPool({ storage: null, fetch: async (url, init) => {
    calls.push(String(url)); assert.equal(init.credentials, 'omit'); assert.equal(init.redirect, 'error');
    assert.equal(init.cache, 'no-store'); assert.equal(init.headers, undefined);
    return new Response('<html>unavailable</html>', { status: 503 });
  } });
  for (let i = 0; i < 5; i++) { const before = calls.length; await pool.refresh(true); assert.equal(calls.length - before, 2); }
  assert.equal(new Set(calls.map(u => new URL(u).origin)).size, 5);
  assert.deepEqual(pool.snapshot().origins, [...DEFAULT_API_ORIGINS]);
});
test('independent relay is preferred to its CDN alias and singleton pools remain bounded', () => {
  const routes = [BOOTSTRAP_ORIGINS[1], BOOTSTRAP_ORIGINS[3], BOOTSTRAP_ORIGINS[2]];
  assert.deepEqual(boundedCandidates(routes, routes[0]), [routes[0], routes[2]]);
  assert.deepEqual(boundedCandidates(routes.slice(0, 1), routes[0]), routes.slice(0, 1));
});
test('bootstrap rejects oversized, HTML and foreign data without replacing valid routes', async () => {
  for (const make of [() => new Response('x'.repeat(65537)), () => new Response('<html>200</html>'), () => response(doc(1, ['https://evil.example']))]) {
    const pool = createDomainPool({ storage: null, fetch: async () => make() });
    await pool.refresh(); assert.deepEqual(pool.snapshot().origins, [...DEFAULT_API_ORIGINS]);
  }
});
test('resolution/bootstrap timeout is bounded and does not loop indefinitely', async () => {
  let calls = 0;
  const pool = createDomainPool({ storage: null, bootstrapTimeoutMs: 10, fetch: async (_, init) => {
    calls++; return new Promise((_, reject) => init.signal.addEventListener('abort', () => reject(new DOMException('timeout', 'AbortError'))));
  } });
  const start = Date.now(); await pool.refresh();
  assert.equal(calls, 2); assert.ok(Date.now() - start < 500);
});
test('rate limits, auth, application errors and cancellation do not fail over', () => {
  for (const status of [400, 401, 403, 404, 408, 413, 429, 500]) assert.equal(retryableRouteFailure(status), false);
  assert.equal(retryableRouteFailure(503, { code: 50001, message: 'business failure' }), false);
  assert.equal(retryableRouteFailure(undefined, undefined, true), false);
  assert.equal(retryableRouteFailure(503, '<html>gateway error</html>'), true);
  assert.equal(retryableRouteFailure(undefined), true);
});

test('cold bootstrap promotes a responding published route after the active entry fails', async () => {
  const pool = createDomainPool({ storage: null, fetch: async url => String(url).startsWith(DEFAULT_API_ORIGINS[0])
    ? new Response('unavailable', { status: 503 }) : response(doc()) });
  await pool.refresh(); assert.equal(pool.currentOrigin(), DEFAULT_API_ORIGINS[1]);
  assert.deepEqual(pool.snapshot().origins, [...DEFAULT_API_ORIGINS]);
});
test('future-route preparation uses at most two anonymous health reads', async () => {
  const probes = [];
  const pool = createDomainPool({ storage: null, fetch: async (url, init) => {
    if (String(url).endsWith('/domain-pool')) return response(doc());
    probes.push({ url, init });
    return new Response(JSON.stringify({ code: 0, data: { status: 'ok' } }));
  } });
  await pool.refresh(); await pool.recover(DEFAULT_API_ORIGINS[0]);
  assert.equal(pool.currentOrigin(), DEFAULT_API_ORIGINS[1]); assert.equal(probes.length, 1);
  assert.equal(probes[0].init.headers, undefined); assert.equal(probes[0].init.credentials, 'omit');
});
test('HTML, auth and malformed health replies cannot become a future route', async () => {
  let probes = 0;
  const pool = createDomainPool({ storage: null, fetch: async url => {
    if (String(url).endsWith('/domain-pool')) return response(doc());
    probes++; return probes === 1 ? new Response('<html>OK</html>') : new Response(JSON.stringify({ code: 401, data: { status: 'ok' } }));
  } });
  await pool.refresh(); await pool.recover(DEFAULT_API_ORIGINS[0]);
  assert.equal(probes, 2); assert.equal(pool.currentOrigin(), DEFAULT_API_ORIGINS[0]);
});
test('late health replies cannot restore a retired origin', async () => {
  let current = doc(1), release;
  const pool = createDomainPool({ storage: null, fetch: async url => {
    if (String(url).endsWith('/domain-pool')) return response(current);
    return new Promise(resolve => { release = () => resolve(new Response(JSON.stringify({ code: 0, data: { status: 'ok' } }))); });
  } });
  await pool.refresh(); const recovery = pool.recover(DEFAULT_API_ORIGINS[0]);
  assert.equal(pool.recover(DEFAULT_API_ORIGINS[0]), recovery);
  current = doc(2, [DEFAULT_API_ORIGINS[2]]); await pool.refresh(true);
  release(); await recovery; assert.equal(pool.currentOrigin(), DEFAULT_API_ORIGINS[2]);
});

test('metadata waiting shares the caller deadline without canceling shared work', async () => {
  let finish;
  const operation=new Promise(resolve=>{finish=resolve});
  const started=Date.now();
  await assert.rejects(policy.waitWithinDeadline(operation,started+15),/请求超时/);
  assert.ok(Date.now()-started<300);
  finish('shared-work-completed');assert.equal(await operation,'shared-work-completed');
});
