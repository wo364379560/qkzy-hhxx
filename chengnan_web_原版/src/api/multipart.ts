import { createSHA256 } from 'hash-wasm';

type Request = (method: 'get' | 'post', path: string, body?: unknown) => Promise<any>;
const chunkSize = 8 * 1024 * 1024;
const wait = (ms: number, signal?: AbortSignal) => new Promise<void>((resolve, reject) => {
  const abort = () => { clearTimeout(timer); reject(new DOMException('上传已暂停', 'AbortError')); };
  const timer = setTimeout(() => { signal?.removeEventListener('abort', abort); resolve(); }, ms);
  signal?.addEventListener('abort', abort, { once: true });
  if (signal?.aborted) abort();
});

export async function uploadMultipart(file: File, conversation: string, owner: string, request: Request,
  options: { signal?: AbortSignal; isVideo?: boolean; contentType?: string; onProgress?: (n: number) => void }, active: () => boolean) {
  const assertActive = () => {
    if (options.signal?.aborted) throw new DOMException('上传已暂停', 'AbortError');
    if (!active()) throw Object.assign(new Error('账号已切换，上传已暂停'), { notDispatched: true });
  };
  const hash = await createSHA256();
  hash.init();
  for (let offset = 0; offset < file.size; offset += chunkSize) {
    assertActive();
    hash.update(new Uint8Array(await file.slice(offset, offset + chunkSize).arrayBuffer()));
  }
  const fingerprint = hash.digest('hex');
  const type = options.contentType || file.type || 'application/octet-stream';
  hash.init();
  hash.update(JSON.stringify([owner, conversation, file.name, file.size, fingerprint, type, !!options.isVideo]));
  const id = hash.digest('hex');
  assertActive();
  let state = await request('post', '/upload/attachment/multipart/init', {
    request_id: id, content_hash: fingerprint, conversation_id: conversation,
    file_name: file.name, file_size: file.size, content_type: type, is_video: !!options.isVideo,
  });
  if (state.content_hash !== fingerprint || state.file_size !== file.size || state.part_size !== chunkSize) throw new Error('上传任务与原文件不一致');
  const path = `/upload/attachment/multipart/${encodeURIComponent(state.file_id)}`;
  // The deterministic request identity lets a reselected file resume after refresh.
  // The server is authoritative for uploaded bytes/ETags; no URLs are stored.
  if (state.state === 'uploading') {
    const confirmed = new Map<number, number>((state.parts || []).map((part: any) => [part.part_number, part.size]));
    let confirmedBytes = [...confirmed.values()].reduce((a, b) => a + b, 0);
    options.onProgress?.(confirmedBytes / file.size);
    for (let offset = 0, part = 1; offset < file.size; offset += chunkSize, part++) {
      const blob = file.slice(offset, offset + chunkSize);
      if (confirmed.get(part) === blob.size) continue;
      for (let attempt = 0; ; attempt++) {
        assertActive();
        try {
          const signature = await request('post', `${path}/parts/${part}`);
          assertActive();
          const response = await fetch(signature.put_url, { method: 'PUT', body: blob, signal: options.signal });
          if (!response.ok) throw new Error(`分片上传被存储服务拒绝（HTTP ${response.status}）`);
          break;
        } catch (error) {
          assertActive();
          // A lost PUT response may already be committed in COS.
          const progress = await request('get', path);
          if (progress.parts?.some((row: any) => row.part_number === part && row.size === blob.size)) break;
          if (attempt >= 2) throw error;
          await wait(500 * 2 ** attempt, options.signal);
        }
      }
      confirmedBytes += blob.size;
      options.onProgress?.(Math.min(1, confirmedBytes / file.size));
    }
    assertActive();
    state = await request('post', `${path}/complete`);
  }
  const deadline = Date.now() + 10 * 60 * 1000;
  while (state.state !== 'uploaded') {
    assertActive();
    if (['failed', 'aborted', 'cleaned'].includes(state.state)) throw new Error('文件确认失败或上传已取消，请重新选择文件');
    if (Date.now() >= deadline) throw new Error('文件仍在服务端校验，重新选择同一文件可继续查询，请勿重复发送');
    await wait(1500, options.signal);
    state = await request('get', path);
  }
  assertActive();
  options.onProgress?.(1);
  return { ...state, put_url: '', object_key: '', max_size_mb: 500, native_limit_mb: 100 };
}
