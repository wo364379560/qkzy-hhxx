import type { IMessageModel } from "@tencentcloud/chat-uikit-engine-lite";

interface IConversationMessageSnapshot {
  allMessages: IMessageModel[];
  visibleMessages: IMessageModel[];
  lastMessage?: IMessageModel;
  isCompleted: boolean;
  updatedAt: number;
}

const SNAPSHOT_TTL = 60 * 1000;
const MAX_SNAPSHOT_COUNT = 20;
const MAX_SNAPSHOT_MESSAGE_COUNT = 300;
const conversationMessageSnapshotCache = new Map<
  string,
  IConversationMessageSnapshot
>();

function pruneSnapshots() {
  const now = Date.now();
  Array.from(conversationMessageSnapshotCache.entries()).forEach(
    ([conversationID, snapshot]) => {
      if (now - snapshot.updatedAt > SNAPSHOT_TTL) {
        conversationMessageSnapshotCache.delete(conversationID);
      }
    }
  );

  while (conversationMessageSnapshotCache.size > MAX_SNAPSHOT_COUNT) {
    const oldestKey = conversationMessageSnapshotCache.keys().next().value;
    if (!oldestKey) {
      break;
    }
    conversationMessageSnapshotCache.delete(oldestKey);
  }
}

export function setConversationMessageSnapshot(
  conversationID: string,
  snapshot: Omit<IConversationMessageSnapshot, "updatedAt">
) {
  if (!conversationID) {
    return;
  }

  if (
    snapshot.allMessages.length > MAX_SNAPSHOT_MESSAGE_COUNT ||
    snapshot.visibleMessages.length > MAX_SNAPSHOT_MESSAGE_COUNT
  ) {
    conversationMessageSnapshotCache.delete(conversationID);
    pruneSnapshots();
    return;
  }

  conversationMessageSnapshotCache.set(conversationID, {
    allMessages: [...snapshot.allMessages],
    visibleMessages: [...snapshot.visibleMessages],
    lastMessage: snapshot.lastMessage,
    isCompleted: snapshot.isCompleted,
    updatedAt: Date.now(),
  });
  pruneSnapshots();
}

export function getConversationMessageSnapshot(conversationID: string) {
  if (!conversationID) {
    return undefined;
  }

  const snapshot = conversationMessageSnapshotCache.get(conversationID);
  if (!snapshot) {
    return undefined;
  }

  if (Date.now() - snapshot.updatedAt > SNAPSHOT_TTL) {
    conversationMessageSnapshotCache.delete(conversationID);
    return undefined;
  }

  return {
    allMessages: [...snapshot.allMessages],
    visibleMessages: [...snapshot.visibleMessages],
    lastMessage: snapshot.lastMessage,
    isCompleted: snapshot.isCompleted,
  };
}
