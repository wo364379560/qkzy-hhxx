/** Public Rui route metadata only. Native HTTPDNS credentials never belong in Web. */
export const DEFAULT_API_ORIGINS = [
  'https://api.ruizhihe.com', 'https://api.prfgsu.com', 'https://api.uktjwg.cn',
] as const;
export const BOOTSTRAP_ORIGINS = [
  ...DEFAULT_API_ORIGINS, 'https://api1.prfgsu.com', 'https://api1.uktjwg.cn',
] as const;
// Approved business candidates are independent of built-in discovery entries.
export const TRUSTED_API_ORIGINS = [...BOOTSTRAP_ORIGINS,
  'https://cdnapi1.sazuts.cn', 'https://cdnapi2.sazuts.cn',
] as const;
const MAX_STALE_MS = 24 * 60 * 60 * 1000;
export const BOOTSTRAP_PATH = '/api/v1/public/bootstrap/domain-pool';
export type DomainPoolDocument = { version: number; ttl_seconds: number; api_origins: string[] };
type StorageLike = Pick<Storage, 'getItem' | 'setItem'>;
const CACHE_KEY = 'rui.web.validated_domain_pool.r2';

export function normalizeOrigin(value: unknown): string | null {
  if (typeof value !== 'string') return null;
  try {
    const u = new URL(value.trim());
    if (u.protocol !== 'https:' || u.username || u.password || u.search || u.hash ||
        u.pathname !== '/' || u.port || !TRUSTED_API_ORIGINS.some(o => o === u.origin)) return null;
    return u.origin;
  } catch { return null; }
}

export function parseDocument(value: unknown): DomainPoolDocument | null {
  if (!value || typeof value !== 'object') return null;
  const d = value as Partial<DomainPoolDocument>;
  if (!Number.isSafeInteger(d.version) || d.version! < 0 || !Number.isInteger(d.ttl_seconds) ||
      d.ttl_seconds! < 60 || d.ttl_seconds! > 86400 || !Array.isArray(d.api_origins) ||
      d.api_origins.length < 1 || d.api_origins.length > 16) return null;
  const parsed = d.api_origins.map(normalizeOrigin);
  if (parsed.some(o => o === null)) return null;
  return { version: d.version!, ttl_seconds: d.ttl_seconds!, api_origins: [...new Set(parsed as string[])] };
}

export function parseBootstrap(value: unknown): DomainPoolDocument | null {
  const root = value as { code?: unknown; data?: { domain_pool?: unknown } } | null;
  return root?.code === 0 ? parseDocument(root.data?.domain_pool ?? root.data) : null;
}

export function relayGroup(origin: string): string {
  if (origin === 'https://api.prfgsu.com' || origin === 'https://api1.prfgsu.com') return '001';
  if (origin === 'https://api.uktjwg.cn' || origin === 'https://api1.uktjwg.cn') return '002';
  // Both names currently resolve to the same CDN distribution. Do not invent
  // a 001/002 upstream mapping; keep them together until operations confirms it.
  if (/^https:\/\/cdnapi[12]\.sazuts\.cn$/.test(origin)) return 'sazuts';
  return origin;
}

export function boundedCandidates(origins: readonly string[], preferred: string): string[] {
  if (!origins.length) return [];
  const start = Math.max(0, origins.indexOf(preferred));
  const rotated = [...origins.slice(start), ...origins.slice(0, start)];
  const second = rotated.slice(1).find(o => relayGroup(o) !== relayGroup(rotated[0])) ?? rotated[1];
  return second ? [rotated[0], second] : [rotated[0]];
}

/** Rate limits, authentication, cancellation and application errors never change routes. */
export function retryableRouteFailure(status?: number, data?: unknown, canceled = false): boolean {
  if (canceled) return false;
  if (status === undefined) return true;
  if (![502, 503, 504].includes(status)) return false;
  const code = data && typeof data === 'object' ? (data as { code?: unknown }).code : undefined;
  return code === undefined;
}

async function readBootstrap(response: Response): Promise<unknown> {
  if (Number(response.headers.get('Content-Length')) > 65536) throw new Error('Oversized bootstrap');
  const reader = response.body?.getReader();
  if (!reader) {
    const text = await response.text();
    if (new TextEncoder().encode(text).length > 65536) throw new Error('Oversized bootstrap');
    return JSON.parse(text);
  }
  const chunks: Uint8Array[] = [];
  let length = 0;
  try {
    while (true) {
      const next = await reader.read();
      if (next.done) break;
      length += next.value.byteLength;
      if (length > 65536) throw new Error('Oversized bootstrap');
      chunks.push(next.value);
    }
  } finally { await reader.cancel().catch(() => {}); }
  const bytes = new Uint8Array(length);
  let offset = 0;
  for (const chunk of chunks) { bytes.set(chunk, offset); offset += chunk.byteLength; }
  return JSON.parse(new TextDecoder().decode(bytes));
}

export function createDomainPool(options: {
  storage?: StorageLike | null;
  fetch?: typeof fetch;
  now?: () => number;
  bootstrapTimeoutMs?: number;
} = {}) {
  const now = options.now ?? Date.now;
  const request: typeof fetch = options.fetch ?? ((...args) => globalThis.fetch(...args));
  let storage = options.storage;
  if (storage === undefined) { try { storage = globalThis.localStorage; } catch { storage = null; } }
  let document: DomainPoolDocument | null = null;
  let origins: string[] = [...DEFAULT_API_ORIGINS];
  let active = origins[0];
  let refreshAfter = 0;
  let revision = 0;
  let bootstrapCursor = 0;
  let inflight: Promise<void> | null = null;
  let recovering: Promise<void> | null = null;
  let recoveryCursor = 0;
  const cooldowns = new Map<string, number>();
  try {
    const cached = JSON.parse(storage?.getItem(CACHE_KEY) || 'null');
    const valid = parseDocument(cached?.document);
    if (valid && Number.isFinite(cached.fetchedAt) && cached.fetchedAt >= 0 && cached.fetchedAt <= now()) {
      document = valid;
      origins = valid.api_origins;
      active = origins.includes(cached.active) ? cached.active : origins[0];
      refreshAfter = cached.fetchedAt + valid.ttl_seconds * 1000;
    }
  } catch { /* Bad public cache must not affect the user's session. */ }
  let fetchedAt = document ? refreshAfter - document.ttl_seconds * 1000 : 0;
  function persist() {
    if (document) try { storage?.setItem(CACHE_KEY, JSON.stringify({ document, active, fetchedAt })); } catch { /* quota/private mode */ }
  }
  function accepts(next: DomainPoolDocument) {
    return !document || next.version > document.version || JSON.stringify(next) === JSON.stringify(document);
  }
  function apply(next: DomainPoolDocument) {
    if (!accepts(next)) return false;
    if (JSON.stringify(next) !== JSON.stringify(document)) revision++;
    document = next;
    origins = [...next.api_origins];
    // A late first pool must preserve a route already selected successfully.
    // Re-select only when the validated pool no longer contains that route.
    if (!origins.includes(active)) active = origins[0];
    fetchedAt = now();
    refreshAfter = fetchedAt + next.ttl_seconds * 1000;
    persist();
    return true;
  }
  function refresh(force = false): Promise<void> {
    if (inflight) return inflight;
    if (!force && now() < refreshAfter) return Promise.resolve();
    inflight = (async () => {
      const candidates = boundedCandidates(BOOTSTRAP_ORIGINS, BOOTSTRAP_ORIGINS[bootstrapCursor]);
      const failed = new Set<string>();
      for (const origin of candidates) {
        const controller = new AbortController();
        const timer = setTimeout(() => controller.abort(), options.bootstrapTimeoutMs ?? 3000);
        try {
          const response = await request(origin + BOOTSTRAP_PATH, {
            signal: controller.signal, credentials: 'omit', redirect: 'error', cache: 'no-store',
          });
          if (response.status === 200) {
            const next = parseBootstrap(await readBootstrap(response));
            if (next && apply(next)) {
              if (failed.has(active) && origins.includes(origin)) { active = origin; persist(); }
              return;
            }
          }
          failed.add(origin);
        } catch { failed.add(origin); /* Keep the last validated pool. */ }
        finally { clearTimeout(timer); }
      }
      bootstrapCursor = (bootstrapCursor + 1) % BOOTSTRAP_ORIGINS.length;
      refreshAfter = now() + 30000;
    })().finally(() => { inflight = null; });
    return inflight;
  }
  function recover(failed: string): Promise<void> {
    if (recovering) return recovering;
    if (active !== failed) return Promise.resolve();
    const expectedRevision = revision;
    cooldowns.set(failed, now() + 30000);
    const available = origins.filter(o => o !== failed && (cooldowns.get(o) || 0) <= now());
    const start = available.length ? recoveryCursor % available.length : 0;
    const alternatives = [...available.slice(start), ...available.slice(0, start)];
    recoveryCursor++;
    const preferred = alternatives.find(o => relayGroup(o) !== relayGroup(failed)) ?? alternatives[0];
    recovering = (async () => {
      for (const origin of boundedCandidates(alternatives, preferred)) {
        const controller = new AbortController();
        const timer = setTimeout(() => controller.abort(), options.bootstrapTimeoutMs ?? 3000);
        try {
          const response = await request(origin + '/api/v1/chat/ping', {
            signal: controller.signal, credentials: 'omit', redirect: 'error', cache: 'no-store',
          });
          if (response.status !== 200) { cooldowns.set(origin, now() + 30000); continue; }
          const value = await readBootstrap(response) as { code?: unknown; data?: { status?: unknown } };
          if (value?.code !== 0 || value.data?.status !== 'ok') { cooldowns.set(origin, now() + 30000); continue; }
          if (revision === expectedRevision && active === failed && origins.includes(origin)) {
            active = origin; persist();
          }
          return;
        } catch { cooldowns.set(origin, now() + 30000); /* Never replay a write. */ }
        finally { clearTimeout(timer); }
      }
    })().finally(() => { recovering = null; });
    return recovering;
  }
  return {
    refresh,
    recover,
    waitForRecovery: () => recovering ?? Promise.resolve(),
    currentOrigin: () => {
      // Keep the last version as an anti-rollback watermark, but do not route
      // business traffic through a cache indefinitely during discovery outages.
      if (document && now() > fetchedAt + document.ttl_seconds * 1000 + MAX_STALE_MS) {
        throw new Error('线路信息已过期，请连接网络后重试');
      }
      return active;
    },
    snapshot: () => ({ origins: [...origins], revision, document }),
    alternative(failed: string): string | undefined {
      const start = origins.indexOf(failed);
      const ordered = start < 0 ? origins : [...origins.slice(start + 1), ...origins.slice(0, start)];
      const rest = ordered.filter(o => o !== failed);
      return rest.find(o => relayGroup(o) !== relayGroup(failed)) ?? rest[0];
    },
    remember(origin: string, expectedRevision: number) {
      if (revision === expectedRevision && origins.includes(origin)) { active = origin; persist(); }
    },
  };
}

/** A caller may time out without canceling public work shared with other callers. */
export function waitWithinDeadline<T>(operation: Promise<T>, deadline: number): Promise<T> {
  const remaining = deadline - Date.now();
  if (remaining <= 0) return Promise.reject(new Error('请求超时，请检查网络后重试'));
  return new Promise<T>((resolve, reject) => {
    const timer = setTimeout(() => reject(new Error('请求超时，请检查网络后重试')), remaining);
    operation.then(value => { clearTimeout(timer); resolve(value); }, error => { clearTimeout(timer); reject(error); });
  });
}

export const domainPool = createDomainPool();
