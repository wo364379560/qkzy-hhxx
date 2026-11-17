/** 群 @ 展示名：群昵称 > 对方昵称，不用好友备注。 */
export function resolveGroupAtDisplayName(item: {
  isAll?: boolean;
  nameCard?: string;
  nick?: string;
  userID?: string;
} | null | undefined): string {
  if (!item) return '';
  if (item.isAll) {
    return String(item.nick || '所有人').trim();
  }
  const nameCard = String(item.nameCard || '').trim();
  if (nameCard) return nameCard;
  const nick = String(item.nick || '').trim();
  if (nick) return nick;
  return String(item.userID || '').trim();
}
