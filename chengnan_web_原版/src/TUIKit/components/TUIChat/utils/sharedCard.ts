export interface SharedCard { kind: 'user' | 'group'; id: string; name: string; avatar: string; version: number }

export function parseSharedCard(raw: unknown, depth = 0): SharedCard | null {
  if (depth > 4) return null;
  if (typeof raw === 'string') {
    if (raw.length > 16384) return null;
    try { raw = JSON.parse(raw); } catch { return null; }
  }
  if (!raw || typeof raw !== 'object' || Array.isArray(raw)) return null;
  const data = raw as Record<string, any>;
  if (['contact_card_v1', 'group_card_v1', 'zhihuan_contact_card'].includes(data.businessID)) {
    const version = Number(data.version ?? 1);
    const kind = data.businessID === 'group_card_v1' ? 'group' : 'user';
    const id = String(kind === 'group' ? data.groupId || data.groupID || '' : data.userId || data.userID || '').trim();
    if (![1, 2].includes(version) || !id || id.length > 128 || /\s|[\x00-\x1f]/.test(id)) return null;
    const name = String(kind === 'group' ? data.groupName || id : data.nickname || data.nick || id).trim().slice(0, 128);
    let avatar = '';
    try { const url = new URL(data.avatar || ''); if (['https:', 'http:'].includes(url.protocol) && !url.username && !url.password) avatar = url.href; } catch { /* use placeholder */ }
    return { kind, id, name, avatar, version };
  }
  for (const key of ['payload', 'data', 'customElem', 'custom']) { const card = parseSharedCard(data[key], depth + 1); if (card) return card; }
  return null;
}

export function createSharedCardPayload(card: SharedCard) {
  const data = { businessID: card.kind === 'group' ? 'group_card_v1' : 'contact_card_v1', version: 2,
    ...(card.kind === 'group' ? { groupId: card.id, groupName: card.name } : { userId: card.id, nickname: card.name }), avatar: card.avatar };
  if (!parseSharedCard(data)) throw new Error('名片资料无效');
  return { data: JSON.stringify(data), description: `[${card.kind === 'group' ? '群' : '个人'}名片] ${card.name}`, extension: '睿名片' };
}
