import { reactive } from "../adapter-vue";
import { TUIUserService } from "@tencentcloud/chat-uikit-engine-lite";

const userTagState = reactive<Record<string, string>>({});
const loadedUserTagSet = new Set<string>();
const pendingUserTagMap = new Map<string, Promise<void>>();

function normalizeUserID(value: string) {
  return String(value || "").trim();
}

export function getUserTag(userID: string) {
  const normalizedUserID = normalizeUserID(userID);
  if (!normalizedUserID) {
    return "";
  }
  return userTagState[normalizedUserID] || "";
}

export async function ensureUserTags(userIDs: string[]) {
  const targets = Array.from(
    new Set(
      userIDs
        .map((item) => normalizeUserID(item))
        .filter((item) => item)
        .filter((item) => !loadedUserTagSet.has(item))
        .filter((item) => !pendingUserTagMap.has(item))
    )
  );

  if (!targets.length) {
    return;
  }

  const request = TUIUserService.getUserProfile({
    userIDList: targets,
  })
    .then((res: any) => {
      const profiles = res?.data || [];
      const loadedTargets = new Set<string>();
      profiles.forEach((profile: any) => {
        const userID = normalizeUserID(profile?.userID);
        if (!userID) {
          return;
        }
        userTagState[userID] = extractUserTagFromProfile(profile);
        loadedUserTagSet.add(userID);
        loadedTargets.add(userID);
      });

      targets.forEach((userID) => {
        if (loadedTargets.has(userID)) {
          return;
        }
        userTagState[userID] = "";
        loadedUserTagSet.add(userID);
      });
    })
    .finally(() => {
      targets.forEach((userID) => pendingUserTagMap.delete(userID));
    });

  targets.forEach((userID) => pendingUserTagMap.set(userID, request));
  await request;
}

export function extractUserTagFromProfile(profile: any) {
  const fields = profile?.profileCustomField;
  if (Array.isArray(fields)) {
    for (const item of fields) {
      if (!item || typeof item !== "object") {
        continue;
      }

      const directValue =
        item.UserTag ||
        item.Tag_Profile_Custom_UserTag ||
        item["Tag_Profile_Custom_UserTag"] ||
        item.user_tag ||
        item.Tag_Profile_Custom_user_tag ||
        item["Tag_Profile_Custom_user_tag"];
      if (typeof directValue === "string" && directValue.trim()) {
        return directValue.trim();
      }

      const key = String(item.key || item.Key || "").trim();
      const value = String(item.value || item.Value || "").trim();
      if (
        value &&
        (key === "UserTag" ||
          key === "Tag_Profile_Custom_UserTag" ||
          key === "user_tag" ||
          key === "Tag_Profile_Custom_user_tag")
      ) {
        return value;
      }
    }
  } else if (fields && typeof fields === "object") {
    const directValue =
      fields.UserTag ||
      fields.Tag_Profile_Custom_UserTag ||
      fields["Tag_Profile_Custom_UserTag"] ||
      fields.user_tag ||
      fields.Tag_Profile_Custom_user_tag ||
      fields["Tag_Profile_Custom_user_tag"];
    if (typeof directValue === "string" && directValue.trim()) {
      return directValue.trim();
    }
  }

  return "";
}
