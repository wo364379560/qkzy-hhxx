import { shallowRef } from 'vue';
import TUIChatEngine from '@tencentcloud/chat-uikit-engine-lite';
import { getProfile, getUserBriefs, syncFriendOwnerFromMe } from '@/api';
import { useAuthSession } from '@/features/auth/session';
import type { ChatProfile, FriendOwnerAssignment } from '@/features/auth/types';

export const friendOwnerSelfProfile = shallowRef<ChatProfile | null>(null);
export const friendOwnerTargets = shallowRef<Record<string, any>>({});
const pending = new Map<string, Promise<FriendOwnerAssignment | null>>();
const chatIDs = new WeakMap<object, number>();
let nextChatID = 0;
let publishedScope = '';

export function captureFriendOwnerScope() {
  const session = useAuthSession().currentSession.value;
  const chat = TUIChatEngine.chat;
  if (!chat || !session?.userID) return { key: '', active: () => false };
  if (!chatIDs.has(chat)) chatIDs.set(chat, ++nextChatID);
  const key = `${session.sdkAppID}:${session.userID}:${session.sessionFamilyID}:${chatIDs.get(chat)}`;
  return { key, active: () => {
    const current = useAuthSession().currentSession.value;
    return chat === TUIChatEngine.chat && current?.userID === session.userID && current?.sdkAppID === session.sdkAppID
      && current?.sessionFamilyID === session.sessionFamilyID && chat.getLoginUser() === session.userID;
  } };
}

export function clearFriendOwnerState() {
  publishedScope = ''; friendOwnerSelfProfile.value = null; friendOwnerTargets.value = {};
}

async function refreshOwnerProfiles(target: string, scope: ReturnType<typeof captureFriendOwnerScope>) {
  const [self, targets] = await Promise.allSettled([getProfile({ fresh: true }), getUserBriefs([target])]);
  if (!scope.active()) return;
  if (publishedScope !== scope.key) { clearFriendOwnerState(); publishedScope = scope.key; }
  if (self.status === 'fulfilled' && Number(self.value.owner_revision || 0) >= Number(friendOwnerSelfProfile.value?.owner_revision || 0)) {
    friendOwnerSelfProfile.value = self.value;
  }
  if (targets.status === 'fulfilled') {
    const row = targets.value.find(item => item.im_user_id === target);
    if (row && Number(row.owner_revision || 0) >= Number(friendOwnerTargets.value[target]?.owner_revision || 0)) {
      friendOwnerTargets.value = { ...friendOwnerTargets.value, [target]: row };
    }
  }
}

// Relationship success and assignment success are separate. This routine never
// throws into the SDK add/accept catch path or resends a friend application.
export function synchronizeFriendOwner(target: string): Promise<FriendOwnerAssignment | null> {
  const scope = captureFriendOwnerScope();
  if (!target || !scope.active()) return Promise.resolve(null);
  const key = `${scope.key}:${target}`;
  const existing = pending.get(key); if (existing) return existing;
  if (pending.size >= 32) return Promise.resolve({ state: 'pending', reason_code: 'client_queue_full', request_id: '' });
  const requestID = globalThis.crypto?.randomUUID?.() || `owner-${Math.random().toString(36).slice(2)}-${Math.random().toString(36).slice(2)}`;
  const work = (async () => {
    let result: FriendOwnerAssignment = { state: 'pending', reason_code: 'synchronization_unavailable', request_id: requestID };
    try {
      const response = await syncFriendOwnerFromMe(target, requestID);
      result = response.assignment || { state: 'pending', reason_code: 'legacy_server_response', request_id: requestID };
    } catch { /* A queued operation can continue on the server. Read the current profiles below. */ }
    if (!scope.active()) return null;
    await refreshOwnerProfiles(target, scope);
    return scope.active() ? result : null;
  })().finally(() => pending.delete(key));
  pending.set(key, work); return work;
}
