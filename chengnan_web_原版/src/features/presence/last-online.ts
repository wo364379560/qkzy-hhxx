import type { UserLastOnline } from "@/api";

export type LastOnlineDisplay = {
  lastOnlineAt?: string;
  label?: string;
};

export function resolveLastOnlineAt(item?: UserLastOnline, observedOfflineAt = 0) {
  // A newly observed online -> offline transition must not be overwritten by
  // an older API response while the Tencent state callback is still arriving.
  const times = [item?.last_offline_at, item?.last_online_at]
    .map(value => value ? Date.parse(value) : 0).filter(Number.isFinite);
  const timestamp = Math.max(0, observedOfflineAt, ...times);
  return timestamp ? new Date(timestamp).toISOString() : '';
}

export function formatLastOnlineText(value?: string | null, now = Date.now()) {
  if (!value) {
    return "";
  }

  const timestamp = new Date(value).getTime();
  if (!Number.isFinite(timestamp)) {
    return "";
  }

  const diffSeconds = Math.max(0, Math.floor((now - timestamp) / 1000));
  if (diffSeconds < 60) {
    return "刚刚在线";
  }

  const diffMinutes = Math.floor(diffSeconds / 60);
  if (diffMinutes < 60) {
    return `${diffMinutes}分钟前在线`;
  }

  const diffHours = Math.floor(diffMinutes / 60);
  if (diffHours < 24) {
    const remainingMinutes = diffMinutes % 60;
    if (remainingMinutes === 0) {
      return `${diffHours}小时前在线`;
    }
    return `${diffHours}小时${remainingMinutes}分钟前在线`;
  }

  const diffDays = Math.max(1, Math.floor(diffHours / 24));
  return `${diffDays}天前在线`;
}

export function buildLastOnlineMap(items: UserLastOnline[]) {
  const now = Date.now();
  return items.reduce<Record<string, LastOnlineDisplay>>((result, item) => {
    const userID = String(item?.im_user_id || "").trim();
    if (!userID) {
      return result;
    }

    const lastOnlineAt = resolveLastOnlineAt(item);
    result[userID] = {
      lastOnlineAt,
      label: formatLastOnlineText(lastOnlineAt, now),
    };
    return result;
  }, {});
}
