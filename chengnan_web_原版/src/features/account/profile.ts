import TUIChatEngine, {
  TUIUserService,
  TUIStore,
  StoreName,
} from "@tencentcloud/chat-uikit-engine-lite";
import { updateProfile, uploadAvatar } from "@/api";
import { AvatarSyncError, assertProfileConfirmed } from './avatarSync';

interface UserProfilePatch {
  wananID?: string;
  nick?: string;
  avatar?: string;
  selfSignature?: string;
  gender?: string | number;
  birthday?: number;
  allowType?: string | number;
}

function normalizeGender(value: UserProfilePatch["gender"]) {
  switch (value) {
    case TUIChatEngine.TYPES.GENDER_MALE:
    case 1:
    case "1":
      return 1;
    case TUIChatEngine.TYPES.GENDER_FEMALE:
    case 2:
    case "2":
      return 2;
    default:
      return 0;
  }
}

function normalizeAllowType(value: UserProfilePatch["allowType"]) {
  switch (value) {
    case TUIChatEngine.TYPES.ALLOW_TYPE_NEED_CONFIRM:
    case 1:
    case "1":
      return 1;
    case TUIChatEngine.TYPES.ALLOW_TYPE_DENY_ANY:
    case 2:
    case "2":
      return 2;
    default:
      return 0;
  }
}

export async function syncCurrentUserProfile(patch: UserProfilePatch) {
  const owner = TUIStore.getData(StoreName.USER, 'userProfile')?.userID;
  const apiPatch: Parameters<typeof updateProfile>[0] = {};
  const tuiPatch: Record<string, unknown> = {};

  if (patch.wananID !== undefined) {
    apiPatch.wananID = patch.wananID;
  }
  if (patch.nick !== undefined) {
    apiPatch.nickname = patch.nick;
    tuiPatch.nick = patch.nick;
  }
  if (patch.avatar !== undefined) {
    apiPatch.avatarURL = patch.avatar;
    tuiPatch.avatar = patch.avatar;
  }
  if (patch.selfSignature !== undefined) {
    apiPatch.selfSignature = patch.selfSignature;
    tuiPatch.selfSignature = patch.selfSignature;
  }
  if (patch.gender !== undefined) {
    apiPatch.gender = normalizeGender(patch.gender);
    tuiPatch.gender = patch.gender;
  }
  if (patch.birthday !== undefined) {
    apiPatch.birthday = patch.birthday;
    tuiPatch.birthday = patch.birthday;
  }
  if (patch.allowType !== undefined) {
    apiPatch.allowType = normalizeAllowType(patch.allowType);
    tuiPatch.allowType = patch.allowType;
  }

  await updateProfile(apiPatch);
  if (TUIStore.getData(StoreName.USER, 'userProfile')?.userID !== owner) {
    throw new Error('账号已切换，请重新操作');
  }
  if (Object.keys(tuiPatch).length > 0) {
    assertProfileConfirmed(await TUIUserService.updateMyProfile(tuiPatch));
  }
}

export async function uploadAndSyncAvatar(userID: string, file: File, signal?: AbortSignal) {
  const avatarURL = await uploadAvatar(userID, file, signal);
  if (signal?.aborted || TUIStore.getData(StoreName.USER, 'userProfile')?.userID !== userID) {
    throw new DOMException('上传已取消', 'AbortError');
  }
  await syncUploadedAvatar(userID, avatarURL);
  return avatarURL;
}

export async function syncUploadedAvatar(userID: string, avatarURL: string) {
  if (TUIStore.getData(StoreName.USER, 'userProfile')?.userID !== userID) throw new DOMException('账号已切换', 'AbortError');
  try { await syncCurrentUserProfile({ avatar: avatarURL }); }
  catch (error) { throw new AvatarSyncError(avatarURL, error); }
  return avatarURL;
}
