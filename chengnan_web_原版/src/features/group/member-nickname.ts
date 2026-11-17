import chatStorage from "@/TUIKit/components/TUIChat/utils/chatStorage";

export const GROUP_MEMBER_NICKNAME_VISIBILITY_EVENT =
  "tui-group-member-nickname-visibility";
const FORCE_OPEN_MIGRATION_VERSION = "v2";

function getStorageKey(groupID: string, userID?: string) {
  return `group_member_nickname_visible_${userID || ""}_${groupID}`;
}

function getMigrationKey(groupID: string, userID?: string) {
  return `group_member_nickname_visible_migrated_${userID || ""}_${groupID}`;
}

export function isGroupMemberNicknameEnabled(groupID: string, userID?: string) {
  const storageKey = getStorageKey(groupID, userID);
  const migrationKey = getMigrationKey(groupID, userID);
  const migratedVersion = chatStorage.getChatStorage(migrationKey);
  const value = chatStorage.getChatStorage(storageKey);

  if (migratedVersion !== FORCE_OPEN_MIGRATION_VERSION) {
    chatStorage.setChatStorage(storageKey, true);
    chatStorage.setChatStorage(migrationKey, FORCE_OPEN_MIGRATION_VERSION);
    return true;
  }

  if (typeof value === "boolean") {
    return value;
  }
  return true;
}

export function setGroupMemberNicknameEnabled(
  groupID: string,
  userID: string | undefined,
  enabled: boolean
) {
  chatStorage.setChatStorage(getStorageKey(groupID, userID), enabled);
  chatStorage.setChatStorage(
    getMigrationKey(groupID, userID),
    FORCE_OPEN_MIGRATION_VERSION
  );
  if (typeof window !== "undefined") {
    window.dispatchEvent(
      new CustomEvent(GROUP_MEMBER_NICKNAME_VISIBILITY_EVENT, {
        detail: {
          conversationID: `GROUP${groupID}`,
          enabled,
        },
      })
    );
  }
}
