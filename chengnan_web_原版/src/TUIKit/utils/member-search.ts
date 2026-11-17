export function matchesMemberKeyword(
  item: Record<string, any> = {},
  keyword: string
) {
  const normalizedKeyword = keyword.trim().toLowerCase();
  if (!normalizedKeyword) {
    return true;
  }

  const candidates = [
    item?.remark,
    item?.friendRemark,
    item?.profile?.remark,
    item?.profile?.friendRemark,
    item?.userProfile?.remark,
    item?.userProfile?.friendRemark,
    item?.friendProfile?.remark,
    item?.friendProfile?.friendRemark,
    item?.nick,
    item?.profile?.nick,
    item?.userProfile?.nick,
    item?.friendProfile?.nick,
    item?.nickname,
    item?.name,
    item?.nameCard,
    item?.userID,
  ]
    .filter(Boolean)
    .map((value) => String(value).toLowerCase());

  return candidates.some((value) => value.includes(normalizedKeyword));
}

export function resolveUserDisplayName(item: Record<string, any> = {}) {
  const candidates = [
    item?.remark,
    item?.friendRemark,
    item?.profile?.remark,
    item?.profile?.friendRemark,
    item?.userProfile?.remark,
    item?.userProfile?.friendRemark,
    item?.friendProfile?.remark,
    item?.friendProfile?.friendRemark,
    item?.nameCard,
    item?.profile?.nick,
    item?.userProfile?.nick,
    item?.friendProfile?.nick,
    item?.nick,
    item?.nickname,
    item?.name,
    item?.userID,
  ];

  for (const value of candidates) {
    const normalized = String(value || "").trim();
    if (normalized) {
      return normalized;
    }
  }

  return "";
}

function firstText(candidates: any[]) {
  for (const value of candidates) {
    const normalized = String(value || "").trim();
    if (normalized) {
      return normalized;
    }
  }
  return "";
}

function getProfile(item: Record<string, any> = {}) {
  return item?.profile || item?.userProfile || item?.friendProfile || {};
}

export function normalizeFriendProfile(item: Record<string, any> = {}) {
  const profile = getProfile(item);
  const remark = firstText([
    item?.remark,
    item?.friendRemark,
    profile?.remark,
    profile?.friendRemark,
    item?.userProfile?.remark,
    item?.userProfile?.friendRemark,
    item?.friendProfile?.remark,
    item?.friendProfile?.friendRemark,
  ]);
  const userID = firstText([
    item?.userID,
    profile?.userID,
    item?.userProfile?.userID,
    item?.friendProfile?.userID,
  ]);
  const nick = firstText([
    profile?.nick,
    item?.nick,
    item?.nickname,
    item?.name,
    item?.userProfile?.nick,
    item?.friendProfile?.nick,
  ]);
  const avatar = firstText([
    profile?.avatar,
    item?.avatar,
    item?.userProfile?.avatar,
    item?.friendProfile?.avatar,
  ]);

  return {
    ...item,
    ...profile,
    profile,
    userID,
    nick,
    avatar,
    remark,
    friendRemark: firstText([item?.friendRemark, profile?.friendRemark, remark]),
  };
}

export function mergeFriendProfileList(
  ...lists: Array<Array<Record<string, any>>>
) {
  const merged = new Map<string, Record<string, any>>();

  lists.forEach((list) => {
    (list || []).forEach((item) => {
      const normalized = normalizeFriendProfile(item);
      const userID = String(normalized?.userID || "").trim();
      if (!userID) {
        return;
      }

      const previous = merged.get(userID) || {};
      const remark = firstText([
        normalized?.remark,
        normalized?.friendRemark,
        previous?.remark,
        previous?.friendRemark,
      ]);

      merged.set(userID, {
        ...previous,
        ...normalized,
        userID,
        remark,
        friendRemark: firstText([normalized?.friendRemark, remark]),
      });
    });
  });

  return Array.from(merged.values());
}
