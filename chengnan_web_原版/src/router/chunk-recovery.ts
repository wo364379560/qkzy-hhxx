import type { Router } from "vue-router";

// A failed lazy import must never silently discard drafts, sends or a call.
export function installChunkRecovery(router: Router) {
  let notice: HTMLDivElement | null = null;
  router.onError((error) => {
    if (typeof document === "undefined" || notice) return;
    const message = error instanceof Error ? error.message : String(error || "");
    if (!["Failed to fetch dynamically imported module", "Importing a module script failed", "error loading dynamically imported module", "Unable to preload CSS"].some(pattern => message.includes(pattern))) return;
    notice = document.createElement("div");
    notice.setAttribute("role", "alert");
    notice.style.cssText = "position:fixed;left:12px;right:12px;bottom:12px;z-index:99999;padding:12px 16px;background:#fff5dc;color:#47370d;border:1px solid #d4ba75;border-radius:8px;font:14px/1.6 system-ui;box-shadow:0 3px 16px #0002";
    const text = document.createElement("span");
    text.textContent = "页面暂时无法加载。请先保存未发送内容、结束通话，再重新加载。";
    const button = document.createElement("button");
    button.type = "button";
    button.textContent = "重新加载";
    button.style.cssText = "margin-left:16px;padding:6px 12px;cursor:pointer";
    button.onclick = () => {
      if (window.confirm("重新加载可能丢失未发送内容，并会中断当前通话。确认重新加载？")) window.location.reload();
    };
    notice.append(text, button);
    document.body.appendChild(notice);
  });
  router.afterEach((_to, _from, failure) => {
    if (!failure && notice) {notice.remove(); notice = null;}
  });
}
