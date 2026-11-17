import TUIChatEngine, {
  TUIChatService,
  TUIStore,
  StoreName,
  SendMessageParams,
  SendMessageOptions,
} from "@tencentcloud/chat-uikit-engine-lite";
import type { IQuoteContent } from "../message-list/message-elements/message-quote/interface";

type QuoteContent = IQuoteContent & { messageRootID?: string };

function parseCustomData(value: string) {
  try {
    const data = JSON.parse(value || "{}");
    return data && typeof data === "object" && !Array.isArray(data) ? data : {};
  } catch {
    return {};
  }
}

// Match the installed TUIKit quote/reply wire format without reading live editor
// state at dispatch time or letting its async success reset a newer quote.
export function captureTextQuote(conversationID: string): string {
  const quote = TUIStore.getData(StoreName.CHAT, "quoteMessage");
  const message = quote?.message;
  if (
    !message ||
    message.conversationID !== conversationID ||
    !["quote", "reply"].includes(quote.type)
  )
    return "";
  const types = TUIChatEngine.TYPES;
  const formats: Record<string, [number, string]> = {
    [types.MSG_TEXT]: [1, message.payload?.text || ""],
    [types.MSG_CUSTOM]: [2, "[自定义消息]"],
    [types.MSG_IMAGE]: [3, "[图片]"],
    [types.MSG_AUDIO]: [4, "[语音]"],
    [types.MSG_VIDEO]: [5, "[视频]"],
    [types.MSG_FILE]: [6, "[文件]"],
    [types.MSG_LOCATION]: [7, ""],
    [types.MSG_FACE]: [8, "[表情]"],
    [types.MSG_GRP_TIP]: [9, ""],
    [types.MSG_MERGER]: [10, message.payload?.title || ""],
  };
  const [messageType, messageAbstract] = formats[message.type] || [0, ""];
  const messageReply: QuoteContent = {
    messageAbstract,
    messageSender: message.nick || message.from,
    messageID: message.ID,
    messageType,
    messageTime: message.time,
    messageSequence: message.sequence,
    version: 1,
  };
  if (quote.type === "reply") {
    messageReply.messageRootID =
      parseCustomData(message.cloudCustomData).messageReply?.messageRootID ||
      message.ID;
  }
  return JSON.stringify({ messageReply });
}

export async function updateTextReplyRoot(cloudCustomData: string) {
  const reply: QuoteContent = parseCustomData(cloudCustomData).messageReply;
  if (!reply?.messageRootID) return;
  try {
    const root = TUIChatService.getEngine().chat.findMessage(
      reply.messageRootID
    );
    if (!root) return;
    const data = parseCustomData(root.cloudCustomData);
    const replies = Array.isArray(data.messageReplies?.replies)
      ? data.messageReplies.replies
      : [];
    if (
      replies.some((item: IQuoteContent) => item.messageID === reply.messageID)
    )
      return;
    data.messageReplies = {
      ...data.messageReplies,
      replies: [
        ...replies,
        {
          messageAbstract: reply.messageAbstract,
          messageSender: reply.messageSender,
          messageID: reply.messageID,
        },
      ],
    };
    await TUIChatService.modifyMessage({
      ...root,
      cloudCustomData: JSON.stringify(data),
    });
  } catch (error) {
    console.warn("[TUIChat] update text reply root failed", error);
  }
}

export async function sendTextWithCapturedQuote(
  options: SendMessageParams,
  sendOptions: SendMessageOptions,
  isActive: () => boolean
) {
  const chat = TUIChatService.getEngine().chat;
  if (!options.to || !options.conversationType)
    throw new Error("Missing text message recipient");
  const messageOptions = {
    ...options,
    to: options.to,
    conversationType: options.conversationType,
  };
  const message = options.payload.atUserList
    ? chat.createTextAtMessage(messageOptions)
    : chat.createTextMessage(messageOptions);
  if (!isActive()) return;
  TUIChatService.updateMessageList([message], "send");
  try {
    const response = await chat.sendMessage(message, sendOptions);
    if (isActive()) {
      TUIChatService.updateMessageList([response.data.message], "edit");
      void updateTextReplyRoot(options.cloudCustomData || "");
    }
    return response;
  } catch (error: any) {
    if (!error?.data?.message) message.status = "fail";
    const failedMessage = error?.data?.message || message;
    if (isActive()) TUIChatService.updateMessageList([failedMessage], "edit");
    throw {
      ...error,
      message: error?.message,
      data: { ...error?.data, message: failedMessage },
    };
  }
}
