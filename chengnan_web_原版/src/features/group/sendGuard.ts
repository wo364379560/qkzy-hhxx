import { getGroupConversationContext, clearGroupConversationContext } from '@/TUIKit/components/TUIChat/utils/groupConversationContext';
import { isGroupSendForbidden } from './sendPermission';
import TUIChatEngine from '@tencentcloud/chat-uikit-engine-lite';

const installed = new WeakSet<object>();

// One guard at the SDK boundary covers toolbar sends, resends, cards and queues.
export function installGroupSendGuard(sdk: any) {
  if (!sdk || installed.has(sdk)) return;
  installed.add(sdk);
  for (const method of ['sendMessage', 'resendMessage']) {
    const original = sdk[method].bind(sdk);
    sdk[method] = async (message: any, ...args: any[]) => {
      const owner = String(sdk.getLoginUser() || '');
      try {
        if (!owner) throw new Error('聊天服务尚未登录');
        const id = String(message.conversationID || `${message.conversationType || ''}${message.to || ''}`);
        if (id.startsWith('GROUP')) {
          const context = await getGroupConversationContext(id.slice(5), owner);
          if (!context.group || !context.selfMember) throw new Error('群权限待确认，请稍后重试');
          if (isGroupSendForbidden({ allMuted: Boolean(context.group.muteAllMembers), role: String(context.selfMember.role || ''), muteUntil: Number(context.selfMember.muteUntil || 0) })) {
            throw Object.assign(new Error('您在该群已被禁言，无法发送或转发'), { code: 10017 });
          }
        }
        if (String(sdk.getLoginUser()) !== owner) throw new Error('账号已切换，请重新操作');
      } catch (error: any) {
        throw Object.assign(new Error(error?.message || '群权限查询失败'), error, { notDispatched: true });
      }
      return original(message, ...args);
    };
  }
  for (const key of ['GROUP_LIST_UPDATED', 'GROUP_ATTRIBUTES_UPDATED', 'GROUP_SYSTEM_NOTICE_RECEIVED', 'GROUP_MEMBER_PROFILE_UPDATED', 'SDK_READY', 'KICKED_OUT']) {
    const event = TUIChatEngine.EVENT[key];
    if (typeof event === 'string') sdk.on?.(event, () => clearGroupConversationContext());
  }
}
