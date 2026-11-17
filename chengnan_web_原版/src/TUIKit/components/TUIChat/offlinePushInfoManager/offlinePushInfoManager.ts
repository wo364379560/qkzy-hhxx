import TUIChatEngine, {
  IConversationModel,
  StoreName,
  TUIStore,
  TUITranslateService,
} from "@tencentcloud/chat-uikit-engine-lite";
import { transformTextWithKeysToEmojiNames } from "../emoji-config";
import {
  IChatOfflinePushInfo,
  IOfflinePushInfoCreateParams,
} from "./interface";
import {
  chatOfflinePushInfo,
  callOfflinePushInfo,
  OPPO_CATEGORY,
  OPPO_CHANNEL_ID,
  OPPO_CONTENT_PARAM_KEY,
  OPPO_PRIVATE_TEMPLATE_ID,
  OPPO_TITLE_PARAM_KEY,
  XIAOMI_CHANNEL_ID,
  XIAOMI_CHAT_TEMPLATE_ID,
} from "./info";
import { DEFAULT_DESC, PUSH_SCENE } from "./const";

const XIAOMI_TEMPLATE_VALUE_LIMIT = 128;

class OfflinePushInfoManager {
  private name = "OfflinePushInfoManager";
  private static instance: OfflinePushInfoManager | null = null;

  private offlinePushInfo: any = {};

  private constructor() {
    this.offlinePushInfo = {
      [PUSH_SCENE.CHAT]: chatOfflinePushInfo,
      [PUSH_SCENE.CALL]: callOfflinePushInfo,
    };
  }

  public static getInstance(): OfflinePushInfoManager {
    if (!OfflinePushInfoManager.instance) {
      OfflinePushInfoManager.instance = new OfflinePushInfoManager();
    }
    return OfflinePushInfoManager.instance;
  }

  public getOfflinePushInfo(scene: PUSH_SCENE) {
    if (!Object.values(PUSH_SCENE).includes(scene)) {
      console.error(
        `${this.name} getOfflinePushInfo scene: ${scene} is invalid`
      );
      return null;
    }
    return this.offlinePushInfo[scene];
  }

  private genTitle(conversation: IConversationModel, userInfo: any) {
    let title =
      typeof conversation?.getShowName === "function"
        ? conversation.getShowName()
        : conversation?.groupProfile?.name ||
          conversation?.userProfile?.nick ||
          conversation?.userProfile?.userID ||
          conversation?.conversationID ||
          "";
    if (conversation.type === TUIChatEngine.TYPES.CONV_C2C) {
      title = userInfo?.nick || userInfo?.userID;
    }
    return title;
  }

  private genDesc(messageType: string, payload: any) {
    let desc = "";
    if (messageType === TUIChatEngine.TYPES.MSG_TEXT) {
      desc = transformTextWithKeysToEmojiNames(payload.text);
    }
    if (messageType === TUIChatEngine.TYPES.MSG_CUSTOM) {
      desc = payload.description;
    }
    return (
      desc || TUITranslateService.t(`TUIChat.${DEFAULT_DESC[messageType]}`)
    );
  }

  /** 共用 title/description：轻度规范化，不套小米严规则。 */
  private normalizeDisplayText(value: string, fallback = "消息"): string {
    let result = String(value || "")
      .replace(/\s+/g, " ")
      .trim();
    if (!result) result = fallback;
    if ([...result].length > XIAOMI_TEMPLATE_VALUE_LIMIT) {
      result = [...result].slice(0, XIAOMI_TEMPLATE_VALUE_LIMIT).join("");
    }
    return result;
  }

  private createOppoTemplateParameters(senderName: string, content: string) {
    return {
      OPPOChannelID: OPPO_CHANNEL_ID,
      OPPOCategory: OPPO_CATEGORY,
      OPPOPrivateMsgTemplateId: OPPO_PRIVATE_TEMPLATE_ID,
      OPPOPrivateTitleParameters: {
        [OPPO_TITLE_PARAM_KEY]: this.normalizeDisplayText(senderName, "新消息"),
      },
      OPPOPrivateContentParameters: {
        [OPPO_CONTENT_PARAM_KEY]: this.normalizeDisplayText(
          content,
          "你有一条新消息"
        ),
      },
    };
  }

  /** 仅用于 XiaoMiTemplateParam。 */
  private sanitizeXiaomiKeyword(value: string, fallback = "消息"): string {
    let result = "";
    let lastWasSpace = true;
    for (const char of String(value || "").trim()) {
      const code = char.codePointAt(0) ?? 0;
      if (/\s/.test(char)) {
        if (!lastWasSpace) {
          result += " ";
          lastWasSpace = true;
        }
        continue;
      }
      if (this.isXiaomiDisallowedRune(code)) {
        continue;
      }
      result += char;
      lastWasSpace = false;
    }
    result = result.trim();
    if (!result) {
      result = this.sanitizeXiaomiKeyword(fallback, "消息");
    }
    if (!result) result = "消息";
    if ([...result].length > XIAOMI_TEMPLATE_VALUE_LIMIT) {
      result = [...result].slice(0, XIAOMI_TEMPLATE_VALUE_LIMIT).join("");
    }
    return result;
  }

  private isXiaomiDisallowedRune(r: number): boolean {
    if (r === 0x200d || r === 0x20e3 || r === 0xfe0e || r === 0xfe0f) {
      return true;
    }
    if (r >= 0x1f000 && r <= 0x1ffff) return true;
    if (r >= 0x2600 && r <= 0x27bf) return true;
    if (r >= 0x2300 && r <= 0x23ff) return true;
    if (r >= 0x2b00 && r <= 0x2bff) return true;
    if (r < 0x20) return true;
    return false;
  }

  public create(options: IOfflinePushInfoCreateParams): IChatOfflinePushInfo {
    const { conversation, messageType = "", payload = {} } = options || {};
    const userInfo = TUIStore.getData(StoreName.USER, "userProfile");
    const isGroup = conversation.type === TUIChatEngine.TYPES.CONV_GROUP;
    const rawSender = userInfo?.nick || userInfo?.userID || "成员";
    const rawTitle = this.genTitle(conversation, userInfo);
    const rawDesc = this.genDesc(messageType, payload);

    const title = this.normalizeDisplayText(rawTitle);
    const description = this.normalizeDisplayText(rawDesc);
    const entity = {
      sender: isGroup ? conversation.groupProfile?.groupID : userInfo?.userID,
      nickName: userInfo?.nick,
      chatType: isGroup ? 2 : 1,
      version: 1,
      action: 1,
    };
    const base = this.offlinePushInfo[PUSH_SCENE.CHAT] || {};
    return {
      title,
      description,
      extension: JSON.stringify({ entity }),
      ...base,
      androidInfo: {
        ...(base.androidInfo || {}),
        XiaoMiChannelID: XIAOMI_CHANNEL_ID,
        XiaoMiTemplateId: XIAOMI_CHAT_TEMPLATE_ID,
        XiaoMiTemplateParam: {
          keywords1: this.sanitizeXiaomiKeyword(rawSender, "成员"),
          keywords2: this.sanitizeXiaomiKeyword(rawDesc, description),
        },
        ...this.createOppoTemplateParameters(rawSender, rawDesc),
      },
    };
  }
}

export default OfflinePushInfoManager;
