import { TUIStore, StoreName } from "@tencentcloud/chat-uikit-engine-lite";

// One lifetime watcher per window. Retained jobs must observe A -> B -> A
// even when their original composer has been replaced by multi-select.
let ownerID = "";
let generation = 0;
let watchingUser = false;
function syncOwner() {
  const next = String(
    TUIStore.getData(StoreName.USER, "userProfile")?.userID || ""
  );
  if (next !== ownerID) {
    ownerID = next;
    generation++;
  }
}
export function captureMessageSendScope() {
  if (!watchingUser) {
    watchingUser = true;
    TUIStore.watch(StoreName.USER, { userProfile: syncOwner });
  }
  syncOwner();
  const capturedGeneration = generation;
  return () => {
    syncOwner();
    return Boolean(ownerID) && generation === capturedGeneration;
  };
}
