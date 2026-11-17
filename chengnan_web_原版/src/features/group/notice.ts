import type { GroupNotice } from '@/api';
export const GROUP_NOTICE_UPDATED = 'chengnan:group-notice-updated';
export function announceGroupNoticeUpdate(groupID: string, notice: GroupNotice) {
  window.dispatchEvent(new CustomEvent(GROUP_NOTICE_UPDATED, { detail: { groupID, notice } }));
}
