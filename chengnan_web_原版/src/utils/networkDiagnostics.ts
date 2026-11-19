// Deliberate allowlist: never pass Axios configs, payloads, URL paths, user IDs,
// credentials, error messages or supplier response bodies into diagnostics.
export function newTraceID() { return globalThis.crypto?.randomUUID?.() || `r-${Math.random().toString(36).slice(2)}`; }
type Phase = 'route_wait' | 'route_failed' | 'response' | 'browser_network' | 'im_ready' | 'im_failed';
export function tracePhase(phase: Phase, started: number, detail: { requestId: string; serverRequestId?: unknown; origin?: string; status?: number; code?: unknown }) {
  let origin: string | undefined;
  try { origin = detail.origin ? new URL(detail.origin).origin : undefined; } catch { /* omit invalid origin */ }
  const serverID = detail.serverRequestId;
  const safeServerID = typeof serverID === 'string' && /^(?:[a-f0-9]{32}|[a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12})$/i.test(serverID);
  console.info('[connection]', { phase, duration_ms: Math.max(0, Date.now() - started),
    request_id: detail.requestId, ...(safeServerID ? { server_request_id: serverID } : {}), ...(origin ? { origin } : {}),
    ...(Number.isInteger(detail.status) ? { status: detail.status } : {}),
    ...(typeof detail.code === 'number' && Number.isFinite(detail.code) ? { code: detail.code } : {}) });
}
