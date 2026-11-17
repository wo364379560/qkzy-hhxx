type CacheEntry<T> = {
  value: T;
  expiresAt: number;
};

type AsyncReadCacheOptions = {
  ttlMs: number;
  maxEntries?: number;
  now?: () => number;
};

export function createAsyncReadCache<T>({
  ttlMs,
  maxEntries = 200,
  now = Date.now,
}: AsyncReadCacheOptions) {
  const values = new Map<string, CacheEntry<T>>();
  const inflight = new Map<string, Promise<T>>();
  const keyVersions = new Map<string, number>();
  let clearVersion = 0;

  function trim() {
    while (values.size > maxEntries) {
      const oldestKey = values.keys().next().value;
      if (oldestKey === undefined) {
        return;
      }
      values.delete(oldestKey);
    }
  }

  function get(key: string, loader: () => Promise<T>) {
    const cached = values.get(key);
    if (cached && cached.expiresAt > now()) {
      values.delete(key);
      values.set(key, cached);
      return Promise.resolve(cached.value);
    }
    if (cached) {
      values.delete(key);
    }

    const pending = inflight.get(key);
    if (pending) {
      return pending;
    }

    const requestKeyVersion = keyVersions.get(key) || 0;
    const requestClearVersion = clearVersion;
    let request: Promise<T>;
    request = Promise.resolve()
      .then(loader)
      .then((value) => {
        if (
          clearVersion === requestClearVersion &&
          (keyVersions.get(key) || 0) === requestKeyVersion
        ) {
          values.delete(key);
          values.set(key, { value, expiresAt: now() + ttlMs });
          trim();
        }
        return value;
      })
      .finally(() => {
        if (inflight.get(key) === request) {
          inflight.delete(key);
        }
      });
    inflight.set(key, request);
    return request;
  }

  function invalidate(key: string) {
    values.delete(key);
    inflight.delete(key);
    keyVersions.set(key, (keyVersions.get(key) || 0) + 1);
  }

  function clear() {
    clearVersion += 1;
    values.clear();
    inflight.clear();
    keyVersions.clear();
  }

  return { get, invalidate, clear };
}
