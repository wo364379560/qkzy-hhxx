/** Keep transport details after converting an Axios error to a user-facing error. */
export class ApiError extends Error {
  readonly status?: number;
  readonly code?: string | number;
  readonly retryAfterMs?: number;
  readonly stage?: string;
  readonly requestId?: string;
  readonly providerStatus?: number;
  readonly providerCode?: string;

  constructor(message: string, details: { status?: number; code?: string | number; retryAfterMs?: number; stage?: string; requestId?: string; providerStatus?: number; providerCode?: string } = {}) {
    super(message);
    this.name = "ApiError";
    this.status = details.status;
    this.code = details.code;
    this.retryAfterMs = details.retryAfterMs;
    this.stage = details.stage; this.requestId = details.requestId;
    this.providerStatus = details.providerStatus; this.providerCode = details.providerCode;
  }
}

export function apiErrorDetails(error: any) {
  const raw = error?.response?.headers?.["retry-after"];
  const seconds = raw === undefined ? NaN : Number(raw);
  const retryAfterMs = Number.isFinite(seconds)
    ? Math.max(0, seconds * 1000)
    : raw ? Math.max(0, Date.parse(String(raw)) - Date.now()) : undefined;
  const diagnostic = error?.response?.data?.data;
  return {
    ...(diagnostic?.stage ? { stage: String(diagnostic.stage), requestId: diagnostic.request_id, providerStatus: diagnostic.provider_status, providerCode: diagnostic.cos_code } : {}),
    status: error?.response?.status as number | undefined,
    code: error?.response?.data?.code ?? error?.code,
    retryAfterMs: Number.isFinite(retryAfterMs) ? retryAfterMs : undefined,
  };
}

export function isInvalidSessionError(error: unknown) {
  // Only a rejected refresh credential is definitive. A business API 401 is
  // handled separately and can simply mean its short-lived access token expired.
  return (error as ApiError)?.status === 401;
}

export function isRetryableSessionError(error: unknown) {
  const status = (error as ApiError)?.status;
  return status === undefined || status === 408 || status === 425 || status === 429 || status >= 500;
}

export function createRefreshBackoff() {
  let attempts = 0;
  let nextAttemptAt = 0;
  let lastError: unknown;
  return {
    reset() { attempts = 0; nextAttemptAt = 0; lastError = undefined; },
    assertReady() { if (lastError && Date.now() < nextAttemptAt) throw lastError; },
    fail(error: unknown) {
      lastError = error;
      attempts++;
      const delayMs = Math.max(
        Math.min(30_000, 2000 * 2 ** (attempts - 1)) + Math.floor(Math.random() * 500),
        (error as ApiError)?.retryAfterMs || 0,
      );
      nextAttemptAt = Date.now() + delayMs;
      // Two automatic retries, then a visible manual retry. Never poll forever
      // in 50–100 background windows, and never retry a permanent 4xx response.
      return { delayMs, retryAutomatically: attempts <= 2 && isRetryableSessionError(error) };
    },
  };
}

/** Bounds UI waiting only; it does not cancel an SDK operation or retry it. */
export function withTimeout<T>(operation: Promise<T>, timeoutMs: number): Promise<T> {
  return new Promise((resolve, reject) => {
    const timer = setTimeout(() => reject(new ApiError("连接暂时不可用，请重试", { code: "ETIMEDOUT" })), timeoutMs);
    operation.then(value => { clearTimeout(timer); resolve(value); }, error => { clearTimeout(timer); reject(error); });
  });
}
