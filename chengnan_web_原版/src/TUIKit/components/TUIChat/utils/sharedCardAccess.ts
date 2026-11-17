function requireActive(active: () => boolean) { if (!active()) throw new DOMException('账号或名片已切换', 'AbortError'); }
function confirmed(result: any) {
  if (result?.code !== undefined && Number(result.code) !== 0) throw Object.assign(new Error('SDK操作未确认'), { code: Number(result.code) });
  return result;
}

export function groupCardJoinMode(group: any, groupID: string): 'invitation' | 'forbidden' | 'request' {
  if (groupID.includes('@TOPIC#') || ['Work', 'Private'].includes(group?.type)) return 'invitation';
  if (['DisableApply', 'ForbidJoin'].includes(group?.joinOption)) return 'forbidden';
  return 'request';
}

export async function readSharedGroupAccess(chat: any, groupID: string, active: () => boolean) {
  requireActive(active);
  const profileResult = confirmed(await chat.getGroupProfile({ groupID }));
  requireActive(active);
  const group = profileResult?.data?.group;
  if (group?.groupID !== groupID) throw new Error('群资料与请求不一致');
  const regular = confirmed(await chat.getGroupList());
  requireActive(active);
  let rows = regular?.data?.groupList || [];
  if (group.type === 'Community') {
    if (typeof chat.getJoinedCommunityList !== 'function') throw new Error('无法确认社群成员身份');
    const communities = confirmed(await chat.getJoinedCommunityList());
    requireActive(active);
    rows = rows.concat(communities?.data?.groupList || communities?.data?.communityList || []);
  }
  const member = rows.some((row: any) => row.groupID === groupID) || ['Owner', 'Admin', 'Member'].includes(group.selfInfo?.role);
  return { group, member, mode: groupCardJoinMode(group, groupID) };
}

export async function joinSharedGroupCard(chat: any, groupID: string, active: () => boolean, join: () => Promise<any>) {
  const before = await readSharedGroupAccess(chat, groupID, active);
  if (before.member) return { ...before, pending: false };
  if (before.mode !== 'request') throw new Error(before.mode === 'invitation' ? '此群需要已有成员邀请' : '该群禁止主动加入');
  confirmed(await join());
  requireActive(active);
  const after = await readSharedGroupAccess(chat, groupID, active);
  return { ...after, pending: !after.member };
}
