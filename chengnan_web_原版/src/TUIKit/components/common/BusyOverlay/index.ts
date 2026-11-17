let overlay: HTMLDivElement | null = null;
let styleEl: HTMLStyleElement | null = null;
let lockCount = 0;
let autoReleaseTimer: ReturnType<typeof window.setTimeout> | null = null;
let visibleStartedAt = 0;
let currentMessage = "";

const DEFAULT_MAX_VISIBLE_MS = 45_000;

const STYLE_TEXT = `
.cn-busy-overlay {
  position: fixed;
  inset: 0;
  z-index: 99999;
  display: none;
  align-items: center;
  justify-content: center;
  background: rgba(15, 23, 42, 0.28);
  cursor: wait;
  pointer-events: auto;
  user-select: none;
}
.cn-busy-card {
  min-width: 168px;
  padding: 22px 26px;
  border-radius: 16px;
  background: #fff;
  box-shadow: 0 16px 48px rgba(15, 23, 42, 0.16);
  text-align: center;
}
.cn-busy-spinner {
  width: 28px;
  height: 28px;
  margin: 0 auto 12px;
  border: 3px solid rgba(67, 97, 238, 0.16);
  border-top-color: #4361ee;
  border-radius: 50%;
  animation: cn-busy-spin 0.8s linear infinite;
}
.cn-busy-text {
  margin: 0;
  color: #334155;
  font-size: 14px;
  line-height: 1.4;
}
@keyframes cn-busy-spin {
  to { transform: rotate(360deg); }
}
`;

function ensureOverlay() {
  if (!styleEl) {
    styleEl = document.createElement("style");
    styleEl.textContent = STYLE_TEXT;
    document.head.appendChild(styleEl);
  }
  if (!overlay) {
    overlay = document.createElement("div");
    overlay.className = "cn-busy-overlay";
    overlay.innerHTML =
      '<div class="cn-busy-card"><div class="cn-busy-spinner"></div><p class="cn-busy-text">处理中...</p></div>';
    overlay.addEventListener("click", (event) => {
      event.preventDefault();
      event.stopPropagation();
    });
    document.body.appendChild(overlay);
  }
  return overlay;
}

function clearAutoReleaseTimer() {
  if (autoReleaseTimer) {
    window.clearTimeout(autoReleaseTimer);
    autoReleaseTimer = null;
  }
}

function scheduleAutoRelease() {
  clearAutoReleaseTimer();
  autoReleaseTimer = window.setTimeout(() => {
    if (!overlay || lockCount <= 0 || overlay.style.display === "none") {
      return;
    }
    const visibleMs = visibleStartedAt ? Date.now() - visibleStartedAt : 0;
    console.warn("[BusyOverlay] auto released after timeout", {
      lockCount,
      message: currentMessage,
      visibleMs,
    });
    lockCount = 0;
    overlay.style.display = "none";
    clearAutoReleaseTimer();
  }, DEFAULT_MAX_VISIBLE_MS);
}

export function showBusy(message = "处理中...") {
  if (lockCount === 0) {
    visibleStartedAt = Date.now();
  }
  lockCount += 1;
  currentMessage = message;
  const el = ensureOverlay();
  const text = el.querySelector(".cn-busy-text");
  if (text) {
    text.textContent = message;
  }
  el.style.display = "flex";
  scheduleAutoRelease();
}

export function hideBusy() {
  lockCount = Math.max(0, lockCount - 1);
  if (lockCount === 0 && overlay) {
    overlay.style.display = "none";
    currentMessage = "";
    visibleStartedAt = 0;
    clearAutoReleaseTimer();
  }
}

export async function withBusy<T>(
  message: string,
  task: () => Promise<T>
): Promise<T> {
  showBusy(message);
  try {
    return await task();
  } finally {
    hideBusy();
  }
}
