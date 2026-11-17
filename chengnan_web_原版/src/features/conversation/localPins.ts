import type { IConversationModel } from "@tencentcloud/chat-uikit-engine-lite";
import type { ConversationCustomPin } from "@/api";

export const CONVERSATION_LOCAL_PINS_UPDATED_KEY = "conversationLocalPinsUpdatedAt";

export type ConversationCustomPinMap = Record<string, ConversationCustomPin>;

export function toConversationCustomPinMap(
  items: ConversationCustomPin[] = []
): ConversationCustomPinMap {
  return items.reduce<ConversationCustomPinMap>((result, item) => {
    if (item?.conversationID) {
      result[item.conversationID] = item;
    }
    return result;
  }, {});
}

export function getConversationCustomPin(
  items: ConversationCustomPin[] = [],
  conversationID = ""
) {
  return items.find((item) => item?.conversationID === conversationID);
}

export function applyLocalConversationPins(
  list: IConversationModel[],
  pinMap: ConversationCustomPinMap
) {
  return [...list].map((conversation) => {
    const pin = pinMap[conversation.conversationID];
    (conversation as any).isPinned = Boolean(pin);
    (conversation as any).__localPinSortOrder = pin?.pinOrder || 0;
    return conversation;
  });
}

export function compareLocalConversationPins(
  a: IConversationModel,
  b: IConversationModel
) {
  const aPinned = Boolean((a as any).isPinned);
  const bPinned = Boolean((b as any).isPinned);
  if (aPinned !== bPinned) {
    return aPinned ? -1 : 1;
  }
  return 0;
}
