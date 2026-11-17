import { TUIGroupService, TUIStore, StoreName } from '@tencentcloud/chat-uikit-engine-lite';
import { uploadGroupAvatar } from '@/api';
import { AvatarSyncError, assertAvatarActive, assertProfileConfirmed } from '@/features/account/avatarSync';

export async function uploadAndSyncGroupAvatar(groupID: string, file: File, active: () => boolean = () => true, signal?: AbortSignal) {
  const owner = TUIStore.getData(StoreName.USER, 'userProfile')?.userID;
  const current = () => !signal?.aborted && active() && TUIStore.getData(StoreName.USER, 'userProfile')?.userID === owner;
  assertAvatarActive(current);
  const avatarURL = await uploadGroupAvatar(groupID, file, signal);
  assertAvatarActive(current);
  await syncUploadedGroupAvatar(groupID, avatarURL, current);
  return avatarURL;
}

export async function syncUploadedGroupAvatar(groupID: string, avatarURL: string, active: () => boolean) {
  assertAvatarActive(active);
  try { assertProfileConfirmed(await TUIGroupService.updateGroupProfile({ groupID, avatar: avatarURL })); }
  catch (error) { throw new AvatarSyncError(avatarURL, error); }
  assertAvatarActive(active);
  return avatarURL;
}
