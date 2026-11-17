import TUIChatEngine, {
  TUIChatService,
  TUIFriendService,
  TUIStore,
  StoreName,
  TUITranslateService,
  IConversationModel,
  SendMessageParams,
} from "@tencentcloud/chat-uikit-engine-lite";
import { Toast, TOAST_TYPE } from "../../common/Toast/index";
import { isEnabledMessageReadReceiptGlobal } from "../utils/utils";
import { ITipTapEditorContent } from "../../../interface";
import { enableSampleTaskStatus } from "../../../utils/enableSampleTaskStatus";
import OfflinePushInfoManager, {
  IOfflinePushInfoCreateParams,
} from "../offlinePushInfoManager/index";
import {
  getSelectedAttachmentFile,
  shouldUseLargeAttachmentUpload,
} from "./fileLimits";
import {
  showLargeAttachmentSendError,
  uploadAndSendLargeAttachmentMessage,
} from "./largeVideoMessage";
import { sendTextWithCapturedQuote } from "./textMessageQuote";
import { waitForBothWayFriend } from "../../TUIContact/utils/index";

export const sendMessageErrorCodeMap: Map<number, string> = new Map([
  [3123, "文本包含本地审核拦截词"],
  [4004, "图片消息失败,无效的图片格式"],
  [4005, "文件消息失败,禁止发送违规封禁的文件"],
  [7004, "文件不存在,请检查文件路径是否正确"],
  [7005, "文件大小超出了限制,如果上传文件,最大限制是100MB"],
  [8001, "消息长度超出限制,消息长度不要超过12K"],
  [80001, "消息或者资料中文本存在敏感内容,发送失败"],
  [80004, "消息中图片存在敏感内容,发送失败"],
  [10017, "当前已被禁言，暂时无法发送消息，解除禁言后可继续发送。"],
  [20012, "当前已被禁言，暂时无法发送消息，解除禁言后可继续发送。"],
  [20007, "对方已将你加入黑名单，消息发送失败"],
]);

const isNotFriendSendError = (error: { code?: number; message?: string }) =>
  error?.code === 20009 || /not friends/i.test(String(error?.message || ""));

const isBlacklistSendError = (error: { code?: number; message?: string }) =>
  error?.code === 20007 || /blacklist/i.test(String(error?.message || ""));

async function tryAddFriendForDirectSend(userID?: string) {
  const trimmedUserID = String(userID || "").trim();
  if (!trimmedUserID) {
    return false;
  }
  try {
    const result = await TUIFriendService.addFriend({
      to: trimmedUserID,
      source: "AddSource_Type_Web",
      wording: "你好，我想加你为好友",
      type: TUIChatEngine.TYPES.SNS_ADD_TYPE_BOTH,
    });
    if (Number(result?.data?.code || 0) === 30539) {
      return false;
    }
    return waitForBothWayFriend(trimmedUserID);
  } catch (error) {
    console.warn("tryAddFriendForDirectSend failed:", error);
    return false;
  }
}

export const resolveSendMessageErrorMessage = (error: {
  code?: number;
  message?: string;
}) => {
  if (isBlacklistSendError(error)) {
    return TUITranslateService.t("TUIChat.对方已将你加入黑名单，消息发送失败");
  }

  if (isNotFriendSendError(error)) {
    return TUITranslateService.t("TUIChat.当前消息发送失败，可添加好友后重试");
  }

  const mappedMessageKey = error?.code
    ? sendMessageErrorCodeMap.get(error.code)
    : "";
  if (mappedMessageKey) {
    return TUITranslateService.t(`TUIChat.${mappedMessageKey}`);
  }

  return error?.message || TUITranslateService.t("TUIChat.发送失败");
};

const resolveTypedSendMessageErrorMessage = (
  error: { code?: number; message?: string },
  contentType?: string
) => {
  if (error?.code === 7005 && contentType === "video") {
    return TUITranslateService.t("TUIChat.视频文件大小不能超过200MB");
  }

  return resolveSendMessageErrorMessage(error);
};

export interface SendMessagesResult {
  delivery?: "failed" | "unknown";
  hasError: boolean;
  handledErrorType?: "not_friend";
  notFriendTarget?: {
    userID: string;
    nick?: string;
    avatar?: string;
  };
}

const buildNotFriendTarget = (
  conversation: IConversationModel
): SendMessagesResult["notFriendTarget"] => {
  const userProfile = conversation?.userProfile;
  const userID = userProfile?.userID || "";
  if (!userID) {
    return undefined;
  }

  return {
    userID,
    nick: userProfile?.nick,
    avatar: userProfile?.avatar,
  };
};

export const createOfflinePushInfo = (conversation: IConversationModel) => {
  return OfflinePushInfoManager.create({
    conversation,
    messageType: "",
    payload: {},
  });
};

/**
 * This function only processes five message types: Text/TextAt/Image/Video/File
 * @param messageList
 * @param currentConversation
 */
export const sendMessages = async (
  messageList: ITipTapEditorContent[],
  currentConversation: IConversationModel,
  isActive: () => boolean = () => true,
  quoteSnapshot?: string,
  canSend: () => boolean = () => true
): Promise<SendMessagesResult> => {
  // In case of messageJumping, the sent message is automatically cleared and returns to the bottom
  if (TUIStore.getData(StoreName.CHAT, "messageSource")) {
    TUIStore.update(StoreName.CHAT, "messageSource", undefined);
  }

  if (!isActive()) return { hasError: true };
  let hasError = false;
  let delivery: "failed" | "unknown" | undefined;
  for (const content of messageList || []) {
    let retriedAfterAddFriend = false;
    while (true) {
      if (!isActive()) return { hasError: true };
      try {
        if (!canSend()) return { hasError: true, delivery: "failed" };
        const options: SendMessageParams = {
          to:
            currentConversation?.groupProfile?.groupID ||
            currentConversation?.userProfile?.userID,
          conversationType: currentConversation?.type as any,
          payload: {},
          needReadReceipt: isEnabledMessageReadReceiptGlobal(),
        };
        // handle message typing
        let textMessageContent;
        const sendMessageOptions = {
          offlinePushInfo: {},
        };
        const offlinePushInfoCreateParams: IOfflinePushInfoCreateParams = {
          conversation: currentConversation,
          payload: content.payload,
          messageType: "",
        };
        switch (content?.type) {
          case "text":
            textMessageContent = JSON.parse(
              JSON.stringify(content.payload?.text)
            );
            // Do not send empty messages
            if (!textMessageContent) {
              break;
            }
            options.payload = {
              text: textMessageContent,
            };
            offlinePushInfoCreateParams.messageType =
              TUIChatEngine.TYPES.MSG_TEXT;
            sendMessageOptions.offlinePushInfo = OfflinePushInfoManager.create(
              offlinePushInfoCreateParams
            );
            if (content.payload?.atUserList)
              options.payload.atUserList = content.payload.atUserList;
            if (quoteSnapshot !== undefined) {
              options.cloudCustomData = quoteSnapshot;
              await sendTextWithCapturedQuote(
                options,
                sendMessageOptions,
                isActive
              );
            } else if (content.payload?.atUserList) {
              await TUIChatService.sendTextAtMessage(
                options,
                sendMessageOptions
              );
            } else {
              await TUIChatService.sendTextMessage(options, sendMessageOptions);
            }
            break;
          case "image":
            options.payload = {
              file: content.payload?.file,
            };
            offlinePushInfoCreateParams.messageType =
              TUIChatEngine.TYPES.MSG_IMAGE;
            sendMessageOptions.offlinePushInfo = OfflinePushInfoManager.create(
              offlinePushInfoCreateParams
            );
            await TUIChatService.sendImageMessage(options, sendMessageOptions);
            break;
          case "video":
            if (shouldUseLargeAttachmentUpload(content.payload?.file)) {
              const attachmentFile = getSelectedAttachmentFile(
                content.payload?.file
              );
              if (attachmentFile) {
                await uploadAndSendLargeAttachmentMessage(
                  currentConversation,
                  attachmentFile,
                  { isVideo: true, isActive }
                );
                break;
              }
            }
            options.payload = {
              file: content.payload?.file,
            };
            offlinePushInfoCreateParams.messageType =
              TUIChatEngine.TYPES.MSG_VIDEO;
            sendMessageOptions.offlinePushInfo = OfflinePushInfoManager.create(
              offlinePushInfoCreateParams
            );
            await TUIChatService.sendVideoMessage(options, sendMessageOptions);
            break;
          case "file":
            if (shouldUseLargeAttachmentUpload(content.payload?.file)) {
              const attachmentFile = getSelectedAttachmentFile(
                content.payload?.file
              );
              if (attachmentFile) {
                await uploadAndSendLargeAttachmentMessage(
                  currentConversation,
                  attachmentFile,
                  { isActive }
                );
                break;
              }
            }
            options.payload = {
              file: content.payload?.file,
            };
            offlinePushInfoCreateParams.messageType =
              TUIChatEngine.TYPES.MSG_FILE;
            sendMessageOptions.offlinePushInfo = OfflinePushInfoManager.create(
              offlinePushInfoCreateParams
            );
            await TUIChatService.sendFileMessage(options, sendMessageOptions);
            break;
          default:
            break;
        }
        enableSampleTaskStatus("sendMessage");
        break;
      } catch (error: any) {
        if (!isActive()) return { hasError: true };
        if (
          !retriedAfterAddFriend &&
          isNotFriendSendError(error || {}) &&
          currentConversation?.type === TUIChatEngine.TYPES.CONV_C2C
        ) {
          retriedAfterAddFriend = true;
          const added = await tryAddFriendForDirectSend(
            currentConversation?.userProfile?.userID
          );
          if (added) {
            continue;
          }
        }
        hasError = true;
        const knownRejection =
          error?.notDispatched ||
          [
            2024, 2105, 2453, 2454, 3123, 4004, 4005, 7004, 7005, 8001, 80001,
            80004, 10010, 10017, 20007, 20009, 20012, 60018, 10023,
          ].includes(Number(error?.code));
        delivery = knownRejection ? "failed" : "unknown";
        if (isNotFriendSendError(error || {})) {
          return {
            hasError: true,
            delivery: "failed",
            handledErrorType: "not_friend",
            notFriendTarget: buildNotFriendTarget(currentConversation),
          };
        }

        if (shouldUseLargeAttachmentUpload(content.payload?.file)) {
          showLargeAttachmentSendError();
        } else {
          Toast({
            message: resolveTypedSendMessageErrorMessage(
              error || {},
              content.type
            ),
            type: TOAST_TYPE.ERROR,
          });
        }
        // If the message fails to be sent and the message is a reference message, clear the reference message information
        if (
          quoteSnapshot === undefined &&
          TUIStore.getData(StoreName.CHAT, "quoteMessage")
        ) {
          TUIStore.update(StoreName.CHAT, "quoteMessage", {});
        }
        break;
      }
    }
  }

  return { hasError, delivery };
};

export const handleMessageWithTyping = (cloudCustomData: any) => {
  if (!cloudCustomData) {
    cloudCustomData = {};
  }
  cloudCustomData.messageFeature = {
    needTyping: 1,
    version: 1,
  };
  return cloudCustomData;
};

export const sendTyping = (inputContentEmpty: boolean, inputBlur: boolean) => {
  if (!inputContentEmpty && !inputBlur) {
    TUIChatService.enterTypingState();
  } else {
    TUIChatService.leaveTypingState();
  }
};
