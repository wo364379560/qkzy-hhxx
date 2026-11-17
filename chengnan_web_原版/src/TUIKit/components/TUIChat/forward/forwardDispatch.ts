import { cloneDeep } from "lodash";
import type { ForwardItem, ForwardTarget } from "./forwardBatch";

interface ForwardSDK {
  getLoginUser: () => String | string;
  getGroupMemberProfile: (options: { groupID: string; userIDList: string[] }) => Promise<any>;
  createForwardMessage: (options: any) => any;
  createMergerMessage: (options: any) => any;
  sendMessage: (message: any, options: any) => Promise<any>;
  resendMessage: (message: any, options: any) => Promise<any>;
}

// Keep the captured SDK and account across permission reads and dispatch.
export function createCapturedForwardSender(
  sdk: ForwardSDK,
  update: (messages: any[], mode: string) => void,
  isActive: () => boolean,
  ownerID: string
) {
  const assertActive = () => {
    if (!isActive() || !ownerID || String(sdk.getLoginUser()) !== ownerID)
      throw Object.assign(new Error("账号已切换，消息未发送"), {
        notDispatched: true,
      });
  };
  // Only query the sender's personal mute, never the whole group/friend/blacklist.
  // A short cache coalesces consecutive messages without retaining a stale
  // permission for the lifetime of a long forwarding job. Explicit retries use
  // a new sender and therefore recheck immediately.
  const muteReads = new Map<string, { at: number; request: Promise<number>; until?: number }>();
  const requirePersonalMutePermission = async (conversationID: string) => {
    if (!conversationID.startsWith('GROUP')) return;
    const groupID = conversationID.slice(5);
    assertActive();
    let entry = muteReads.get(groupID);
    if (!entry || Date.now() - entry.at >= 3000) {
      const request = sdk.getGroupMemberProfile({ groupID, userIDList: [ownerID] }).then(result => {
        assertActive();
        const member = result?.data?.memberList?.find((item: any) => item.userID === ownerID);
        if (!member) throw new Error('无法确认群内禁言状态，请稍后重试');
        const until = Number(member.muteUntil || 0);
        if (!Number.isFinite(until)) throw new Error('群内禁言状态无效，请稍后重试');
        const cached = muteReads.get(groupID);
        if (cached?.request === request) cached.until = until;
        return until;
      });
      entry = { at: Date.now(), request };
      muteReads.set(groupID, entry);
      request.catch(() => { if (muteReads.get(groupID)?.request === request) muteReads.delete(groupID); });
    }
    try {
      const until = await entry.request;
      assertActive();
      if ((until > 100_000_000_000 ? until : until * 1000) > Date.now()) {
        throw Object.assign(new Error('您在该群已被单独禁言，暂时无法发送或转发消息'), { code: 10017 });
      }
    } catch (error: any) {
      // No SDK message was created/sent, so this is never an unknown send result.
      throw Object.assign(new Error((typeof error === 'string' ? error : error?.message) || '群内禁言状态查询失败，请重试'), error, { notDispatched: true });
    }
  };
  const assertPersonalMutePermission = (conversationID: string) => {
    if (!conversationID.startsWith('GROUP')) return;
    const until = muteReads.get(conversationID.slice(5))?.until;
    if (until === undefined) throw Object.assign(new Error('请先确认群内禁言状态后重试'), {notDispatched:true});
    if ((until > 100_000_000_000 ? until : until * 1000) > Date.now()) {
      throw Object.assign(new Error('您在该群已被单独禁言，暂时无法发送或转发消息'), {code:10017,notDispatched:true});
    }
  };
  // Updating the visible chat list is not the delivery acknowledgement. A UI
  // exception after an SDK ACK must never turn a delivered message into a retry.
  const updateSafely = (messages: any[], mode: string) => {
    if (!isActive() || String(sdk.getLoginUser()) !== ownerID) return;
    try { update(messages, mode); }
    catch { console.warn('Forward chat display update failed; preserving SDK delivery result'); }
  };
  const settle = async (message: any, operation: Promise<any>) => {
    try {
      const result = await operation;
      if (isActive() && String(sdk.getLoginUser()) === ownerID)
        updateSafely(
          result?.data?.messageList || [result?.data?.message || message],
          "edit"
        );
      return result;
    } catch (error: any) {
      const failedMessage = error?.data?.message || message;
      if (failedMessage.status !== "success") failedMessage.status = "fail";
      if (isActive() && String(sdk.getLoginUser()) === ownerID)
        updateSafely([failedMessage], "edit");
      throw Object.assign(new Error((typeof error === "string" ? error : error?.message) || "转发失败"), error, {
        data: { ...error?.data, message: failedMessage },
      });
    }
  };
  return {
    // The batch awaits this BEFORE its dispatch pacing slot. There must be no
    // network awaits between acquiring that slot and the actual SDK send.
    prepare(target: ForwardTarget) {
      if (target.conversation.resolutionError) throw Object.assign(new Error(`解析阶段：${target.conversation.resolutionError}`), { notDispatched: true });
      return requirePersonalMutePermission(String(target.conversation.conversationID));
    },
    send(target: ForwardTarget, item: ForwardItem, merge: boolean) {
      assertActive();
      assertPersonalMutePermission(String(target.conversation.conversationID));
      const conversation = cloneDeep(target.conversation);
      const messages = cloneDeep(item.messages);
      const { needMerge, params, mergeInfo, ...sendOptions } = cloneDeep(
        item.sendOptions || {}
      );
      const conversationType =
        conversation.type ||
        (String(conversation.conversationID).startsWith("GROUP")
          ? "GROUP"
          : "C2C");
      const to = String(conversation.conversationID).slice(
        conversationType.length
      );
      const payload = merge
        ? {
            messageList: messages,
            // The title and abstracts are snapshots, never a live account/profile lookup.
            title:
              mergeInfo?.title ||
              (messages[0]?.conversationType === "GROUP"
                ? "群聊的聊天记录"
                : "聊天记录"),
            abstractList: mergeInfo?.abstractList?.length
              ? mergeInfo.abstractList
              : messages.map(
                  (message) =>
                    `${message.nick || message.from || ""}: ${String(
                      message.payload?.text ||
                        message.payload?.fileName ||
                        "[非文字消息]"
                    ).slice(0, 100)}`
                ),
            compatibleText:
              mergeInfo?.compatibleText || "请升级客户端查看聊天记录",
          }
        : messages[0];
      assertActive();
      const message = (
        merge
          ? sdk.createMergerMessage.bind(sdk)
          : sdk.createForwardMessage.bind(sdk)
      )({
        to,
        conversationType,
        ...params,
        payload,
      });
      assertActive();
      updateSafely([message], "send");
      assertActive();
      return settle(message, sdk.sendMessage(message, sendOptions));
    },
    resend(message: any, item: ForwardItem) {
      assertActive();
      assertPersonalMutePermission(String(message.conversationID || `${message.conversationType}${message.to}`));
      const { needMerge, params, mergeInfo, ...options } = cloneDeep(
        item.sendOptions || {}
      );
      message.status = "unSend";
      updateSafely([message], "resend");
      assertActive();
      return settle(message, sdk.resendMessage(message, options));
    },
  };
}
