import TUIChatEngine, {
  TUIFriendService,
  TUIConversationService,
  TUIGroupService,
  TUIChatService,
  TUIUserService,
  TUIStore,
  StoreName,
  TUITranslateService,
  AddFriendParams,
  JoinGroupParams,
  SendMessageParams,
} from "@tencentcloud/chat-uikit-engine-lite";
import { TUIGlobal } from "@tencentcloud/universal-api";
import { Toast, TOAST_TYPE } from "../../common/Toast/index";
import { searchUsersExact, precheckAddFriend } from "@/api";
import { captureFriendOwnerScope, synchronizeFriendOwner } from "@/features/account/friendOwnership";
import { withBusy } from "../../common/BusyOverlay/index";

const ADD_FRIEND_RESULT_CODE_SUCCESS = 0;
const ADD_FRIEND_RESULT_CODE_ALREADY_FRIENDS = 30015;
const ADD_FRIEND_RESULT_CODE_NEED_CONFIRM = 30539;

const resolveJoinGroupErrorMessage = (error: any) => {
  const rawMessage = String(error?.message || "").trim();
  const normalizedMessage = rawMessage.toLowerCase();

  if (
    rawMessage.includes('"code":10016') ||
    normalizedMessage.includes("before callback failed and refuse")
  ) {
    return "当前处于系统静默时段或被系统规则拒绝，暂时不能加入群组，请稍后再试";
  }

  return rawMessage || "申请入群失败";
};

const normalizeAddFriendMessage = (rawMessage: string) => {
  const normalizedMessage = rawMessage.toLowerCase();

  if (
    normalizedMessage.includes("cross-owner not allowed") ||
    rawMessage.includes("其他服务空间")
  ) {
    return TUITranslateService.t("TUIContact.当前账号属于其他服务空间无法直接添加。");
  }

  return rawMessage;
};

export const resolveAddFriendErrorMessage = (error: any) => {
  const rawMessage = String(error?.message || "").trim();
  const normalizedMessage = rawMessage.toLowerCase();
  const code = Number(error?.code || error?.data?.code || 0);

  if (
    normalizedMessage.includes("cross-owner not allowed") ||
    rawMessage.includes("其他服务空间")
  ) {
    return TUITranslateService.t("TUIContact.当前账号属于其他服务空间无法直接添加。");
  }

  if (
    rawMessage.includes('"code":10016') ||
    code === 120002 ||
    code === 38001 ||
    rawMessage.includes("静默时段") ||
    normalizedMessage.includes("before callback failed and refuse")
  ) {
    if (rawMessage.includes("静默时段")) {
      return rawMessage;
    }
    return "当前处于系统静默时段，暂时禁止添加好友，请稍后再试";
  }

  return (
    normalizeAddFriendMessage(rawMessage) ||
    TUITranslateService.t("TUIContact.申请发送失败")
  );
};

type AddFriendOutcome =
  | {
      kind: "added";
      message: string;
    }
  | {
      kind: "sent";
      message: string;
    }
  | {
      kind: "already-friend";
      message: string;
    }
  | {
      kind: "error";
      message: string;
    };

export const resolveAddFriendOutcome = (result: any): AddFriendOutcome => {
  const rawCode = Number(
    result?.data?.resultCode ??
      result?.data?.code ??
      result?.resultCode ??
      result?.code ??
      0
  );
  const rawMessage = String(
    result?.data?.resultInfo ??
      result?.data?.message ??
      result?.resultInfo ??
      result?.message ??
      ""
  ).trim();
  const normalizedMessage = normalizeAddFriendMessage(rawMessage);

  if (rawCode === ADD_FRIEND_RESULT_CODE_SUCCESS) {
    return {
      kind: "added",
      message: TUITranslateService.t("TUIContact.添加好友成功"),
    };
  }

  if (rawCode === ADD_FRIEND_RESULT_CODE_NEED_CONFIRM) {
    return {
      kind: "sent",
      message: TUITranslateService.t("TUIContact.申请已发送"),
    };
  }

  if (rawCode === ADD_FRIEND_RESULT_CODE_ALREADY_FRIENDS) {
    return {
      kind: "already-friend",
      message: TUITranslateService.t("TUIContact.已经是好友"),
    };
  }

  return {
    kind: "error",
    message:
      normalizedMessage || TUITranslateService.t("TUIContact.申请发送失败"),
  };
};

export const generateAvatar = (item: any): string => {
  return (
    item?.avatar ||
    item?.profile?.avatar ||
    (item?.groupID &&
      "https://web.sdk.qcloud.com/im/assets/images/Public.svg") ||
    "https://web.sdk.qcloud.com/component/TUIKit/assets/avatar_21.png"
  );
};

export const generateName = (item: any): string => {
  return (
    item?.remark ||
    item?.name ||
    item?.profile?.nick ||
    item?.nick ||
    item?.groupID ||
    item?.userID ||
    ""
  );
};

export const generateContactInfoName = (item: any): string => {
  return (
    item?.remark ||
    item?.name ||
    item?.profile?.nick ||
    item?.nick ||
    item?.groupID ||
    item?.userID ||
    ""
  );
};

// Parse the basic information display content of the contactInfo module
// Group information display: group ID group type
// User information display: user ID personal signature
export const generateContactInfoBasic = (contactInfo: any): any[] => {
  const res = [];
  if (contactInfo?.groupID) {
    res.push({
      label: "群ID",
      data: contactInfo.groupID,
      copyable: false,
    });
    if (!isApplicationType(contactInfo)) {
      res.push({
        label: "群类型",
        data: contactInfo?.type || "",
        copyable: false,
      });
    }
    return res;
  }

  const nickname =
    contactInfo?._nickname ||
    contactInfo?.profile?.nick ||
    contactInfo?.nick ||
    "";
  const wananID =
    contactInfo?._wananID ||
    contactInfo?.wananID ||
    contactInfo?.profile?._wananID ||
    "";
  const phone =
    contactInfo?._phone ||
    contactInfo?.phone ||
    contactInfo?.profile?._phone ||
    "";

  // 账号、信号、注册手机号、昵称分行展示。
  res.push({
    label: "昵称",
    data: nickname || "未设置",
    copyable: false,
  });
  res.push({
    label: "UID",
    data: contactInfo?.userID || "",
    copyable: true,
  });
  res.push({
    label: "信号",
    data: wananID || "未设置",
    copyable: Boolean(wananID),
  });
  res.push({
    label: "注册手机号",
    data: phone || "未设置",
    copyable: Boolean(phone),
  });
  if (!isApplicationType(contactInfo)) {
    const ownerID = Number(contactInfo?._ownerID || 0);
    const ownerName = String(contactInfo?._ownerName || "").trim();
    res.push({
      label: "归属",
      data: ownerID > 0 ? ownerName || "未设置" : "未归属",
      copyable: false,
    });
  }

  if (!isApplicationType(contactInfo)) {
    res.push({
      label: "个性签名",
      data: contactInfo?.profile?.selfSignature || "",
      copyable: false,
    });
  }
  return res;
};

export const isApplicationType = (info: any) => {
  return (
    info?.type === TUIChatEngine?.TYPES?.SNS_APPLICATION_SENT_TO_ME ||
    info?.type === TUIChatEngine?.TYPES?.SNS_APPLICATION_SENT_BY_ME
  );
};

function hasFriendInLocalStore(userID: string): boolean {
  const friendList = TUIStore.getData(StoreName.FRIEND, "friendList") || [];
  return (
    Array.isArray(friendList) &&
    friendList.some((friend: any) => friend?.userID === userID)
  );
}

async function hasBothWayFriendOnServer(userID: string): Promise<boolean> {
  try {
    const res: any = await TUIFriendService.checkFriend({
      userIDList: [userID],
      type: TUIChatEngine.TYPES.SNS_CHECK_TYPE_BOTH,
    });
    return (
      res?.data?.successUserIDList?.[0]?.relation ===
      TUIChatEngine.TYPES.SNS_TYPE_BOTH_WAY
    );
  } catch (error: any) {
    console.warn("checkFriend error", error);
    return false;
  }
}

const FRIEND_CONFIRM_RETRY_DELAYS = [0, 150, 350, 700, 1200];

/** 等待腾讯关系链与本地推送完成，避免加好友成功后立刻发信仍命中 20009。 */
export async function waitForBothWayFriend(userID: string): Promise<boolean> {
  const normalizedUserID = String(userID || "").trim();
  if (!normalizedUserID) {
    return false;
  }
  for (const delay of FRIEND_CONFIRM_RETRY_DELAYS) {
    if (delay > 0) {
      await new Promise((resolve) => setTimeout(resolve, delay));
    }
    if (
      hasFriendInLocalStore(normalizedUserID) ||
      (await hasBothWayFriendOnServer(normalizedUserID))
    ) {
      return true;
    }
  }
  return false;
}

function patchCurrentContactRelation(
  userID: string,
  patch: Record<string, unknown>,
) {
  const currentContactInfo = TUIStore.getData(
    StoreName.CUSTOM,
    "currentContactInfo",
  );
  if (currentContactInfo?.userID !== userID) {
    return;
  }
  TUIStore.update(StoreName.CUSTOM, "currentContactInfo", {
    ...currentContactInfo,
    ...patch,
  });
}

function markCurrentContactAsFriend(userID: string) {
  patchCurrentContactRelation(userID, {
    addTime: Math.floor(Date.now() / 1000),
    _friendshipConfirmed: true,
    _friendRequestPending: false,
  });
}

function markCurrentContactRequestPending(userID: string) {
  patchCurrentContactRelation(userID, {
    _friendshipConfirmed: false,
    _friendRequestPending: true,
  });
}

export const isFriend = async (info: any): Promise<boolean> => {
  if (info?.groupID || !info?.userID) {
    return false;
  }
  if (info?.addTime || hasFriendInLocalStore(info.userID)) {
    return true;
  }
  return hasBothWayFriendOnServer(info.userID);
};

// 30015：SNS 字段与现有值相同。第一次改成功后立刻再提交同一备注会命中，SDK 还可能把 code 包丢。
const SNS_RESULT_CODE_UNCHANGED = 30015;

function collectErrorCodes(error: any): number[] {
  const raw = [
    error?.code,
    error?.errorCode,
    error?.data?.code,
    error?.data?.errorCode,
    error?.data?.failureUserIDList?.[0]?.code,
    error?.data?.failureUserIDList?.[0]?.resultCode,
  ];
  return raw
    .map((value) => Number(value))
    .filter((value) => Number.isFinite(value));
}

function isUnchangedRemarkError(error: any) {
  if (collectErrorCodes(error).includes(SNS_RESULT_CODE_UNCHANGED)) {
    return true;
  }
  const message = String(
    error?.message || error?.errorInfo || error?.data?.message || ""
  );
  return /相同|not change|unchanged|already/i.test(message);
}

async function readRemoteFriendRemark(userID: string): Promise<string | null> {
  try {
    const res: any = await TUIFriendService.getFriendProfile({
      userIDList: [userID],
    });
    const profile =
      res?.data?.friendList?.[0] ||
      res?.data?.successUserIDList?.[0] ||
      res?.data?.[0] ||
      res?.friendList?.[0];
    if (!profile) {
      return null;
    }
    return String(profile.remark ?? profile.friendRemark ?? "");
  } catch {
    return null;
  }
}

function readLocalFriendRemark(userID: string): string {
  const currentContactInfo = TUIStore.getData(
    StoreName.CUSTOM,
    "currentContactInfo"
  );
  if (currentContactInfo?.userID === userID) {
    return String(currentContactInfo.remark ?? "");
  }
  const friendList = TUIStore.getData(StoreName.FRIEND, "friendList") || [];
  const friend = Array.isArray(friendList)
    ? friendList.find((item: any) => item?.userID === userID)
    : null;
  return String(friend?.remark ?? "");
}

async function remarkAlreadyApplied(userID: string, remark: string) {
  if (readLocalFriendRemark(userID) === remark) {
    return true;
  }
  const remote = await readRemoteFriendRemark(userID);
  return remote !== null && remote === remark;
}

function toastRemarkSuccess() {
  Toast({
    message: TUITranslateService.t("TUIContact.修改备注成功"),
    type: TOAST_TYPE.SUCCESS,
  });
}

function toastRemarkFailure() {
  Toast({
    message: TUITranslateService.t("TUIContact.修改备注失败"),
    type: TOAST_TYPE.ERROR,
  });
}

// Change friend‘s remark
let updateFriendRemarkPending = false;
export const updateFriendRemark = async (userID: string, remark: string) => {
  const nextRemark = String(remark ?? "");
  // eslint-disable-next-line no-control-regex
  if (nextRemark.replace(/[^\u0000-\u00ff]/g, "aa").length > 96) {
    Toast({
      message: TUITranslateService.t(
        "TUIContact.修改备注失败: 备注长度不得超过 96 字节"
      ),
      type: TOAST_TYPE.ERROR,
    });
    return;
  }
  if (updateFriendRemarkPending) {
    return;
  }
  if (readLocalFriendRemark(userID) === nextRemark) {
    applyFriendRemarkLocally(userID, nextRemark);
    toastRemarkSuccess();
    return;
  }

  updateFriendRemarkPending = true;
  try {
    await TUIFriendService.updateFriend({
      userID,
      remark: nextRemark,
    });
    applyFriendRemarkLocally(userID, nextRemark);
    toastRemarkSuccess();
  } catch (error: any) {
    const applied =
      isUnchangedRemarkError(error) ||
      (await remarkAlreadyApplied(userID, nextRemark));
    if (applied) {
      applyFriendRemarkLocally(userID, nextRemark);
      toastRemarkSuccess();
      return;
    }
    console.warn("update friend remark failed:", error);
    toastRemarkFailure();
  } finally {
    updateFriendRemarkPending = false;
  }
};

function applyFriendRemarkLocally(userID: string, remark: string) {
  const currentContactInfo = TUIStore.getData(
    StoreName.CUSTOM,
    "currentContactInfo"
  );
  if (currentContactInfo?.userID === userID) {
    TUIStore.update(StoreName.CUSTOM, "currentContactInfo", {
      ...currentContactInfo,
      remark,
    });
  }

  const friendList = TUIStore.getData(StoreName.FRIEND, "friendList") || [];
  if (Array.isArray(friendList) && friendList.length > 0) {
    const friendIndex = friendList.findIndex(
      (friend: any) => friend?.userID === userID
    );
    if (friendIndex >= 0) {
      friendList[friendIndex] = {
        ...friendList[friendIndex],
        remark,
      };
      TUIStore.update(StoreName.FRIEND, "friendList", [...friendList]);
    }
  }

  const conversationID = `C2C${userID}`;
  const conversationList =
    TUIStore.getData(StoreName.CONV, "conversationList") || [];
  if (Array.isArray(conversationList) && conversationList.length > 0) {
    const conversationIndex = conversationList.findIndex(
      (conversation: any) => conversation?.conversationID === conversationID
    );
    if (conversationIndex >= 0) {
      conversationList[conversationIndex] = {
        ...conversationList[conversationIndex],
        remark,
      };
      TUIStore.update(StoreName.CONV, "conversationList", [
        ...conversationList,
      ]);
    }
  }

  const currentConversation = TUIStore.getData(
    StoreName.CONV,
    "currentConversation"
  );
  if (currentConversation?.conversationID === conversationID) {
    TUIStore.update(StoreName.CONV, "currentConversation", {
      ...currentConversation,
      remark,
    });
  }
}

// Delete one friend
let deleteFriendPending = false;
export const deleteFriend = (userID: string) => {
  if (deleteFriendPending) {
    return Promise.resolve();
  }
  deleteFriendPending = true;
  return withBusy("正在删除好友...", async () => {
    try {
      const res: any = await TUIFriendService.deleteFriend({
        userIDList: [userID],
        type: TUIChatEngine.TYPES.SNS_DELETE_TYPE_BOTH,
      });
      const { successUserIDList } = res.data;
      if (successUserIDList[0].userID === userID) {
        Toast({
          message: TUITranslateService.t("TUIContact.删除好友成功"),
          type: TOAST_TYPE.SUCCESS,
        });
      } else {
        Toast({
          message: TUITranslateService.t("TUIContact.删除好友失败"),
          type: TOAST_TYPE.ERROR,
        });
      }
    } catch (error: any) {
      console.warn("delete friend failed:", error);
      Toast({
        message: TUITranslateService.t("TUIContact.删除好友失败"),
        type: TOAST_TYPE.ERROR,
      });
    }
  }).finally(() => {
    deleteFriendPending = false;
  });
};

// Add friend
const addFriendActions = new Set<string>();
function syncEstablishedFriendOwner(userID: string) {
 const scope=captureFriendOwnerScope();
 void synchronizeFriendOwner(userID).then(result=>{
  if (!scope.active() || !result || !['pending','blocked','conflict'].includes(result.state)) return;
  Toast({message:result.state==='conflict'?'好友已建立，归属存在冲突，请联系管理员':result.state==='blocked'?'好友已建立，当前归属不可操作，请联系管理员':'好友已建立，归属待同步',type:TOAST_TYPE.WARNING});
 });
}
export const addFriend = (params: AddFriendParams) => {
  const scope=captureFriendOwnerScope();
  const actionKey=`${scope.key}:${String(params?.to||'').trim()}`;
  if (!scope.active() || addFriendActions.has(actionKey)) return Promise.resolve();
  const to = String(params?.to || "").trim();
  const blockedMessage = "当前账号属于其他服务空间无法直接添加。";

  addFriendActions.add(actionKey);
  return withBusy("正在添加好友...", async () => {
    if (to) {
      try {
        const check = await precheckAddFriend(to);
        if (!scope.active()) return;
        if (check && check.can_add_friend === false) {
          Toast({
            message: check.reason || blockedMessage,
            type: TOAST_TYPE.ERROR,
          });
          return;
        }
      } catch {
        try {
          const users = await searchUsersExact(to);
          if (!scope.active()) return;
          const exact = users.find((item) => item.im_user_id === to);
          if (exact && exact.can_add_friend === false) {
            Toast({
              message: exact.add_blocked_reason || blockedMessage,
              type: TOAST_TYPE.ERROR,
            });
            return;
          }
        } catch {
          // 预检失败时仍走 IM 加好友，由回调兜底
        }
      }
    }

    if (!scope.active()) return;
    try {
      const result = await TUIFriendService.addFriend({
        ...params,
        type: params.type ?? TUIChatEngine.TYPES.SNS_ADD_TYPE_BOTH,
        wording: params.wording || "你好，我想加你为好友",
      });
      if (!scope.active()) return;
      const outcome = resolveAddFriendOutcome(result);

      if (outcome.kind === "added" || outcome.kind === "already-friend") {
        const userID = String(params?.to || "").trim();
        if (userID) {
          // addFriend 成功返回早于 FRIEND_LIST_UPDATED 推送，先等待服务端关系可用。
          const confirmed=await waitForBothWayFriend(userID);
          if (!scope.active()) return;
          if (confirmed) syncEstablishedFriendOwner(userID);
          markCurrentContactAsFriend(userID);
          enterC2CChat(userID);
          void sendFriendGreetingMessageIfReady(userID);
        }
        Toast({
          message:
            outcome.kind === "already-friend"
              ? outcome.message
              : TUITranslateService.t("TUIContact.添加好友成功"),
          type:
            outcome.kind === "already-friend"
              ? TOAST_TYPE.WARNING
              : TOAST_TYPE.SUCCESS,
        });
        return;
      }

      if (outcome.kind === "sent") {
        const userID = String(params?.to || "").trim();
        // 后台可能在申请回调后自动建立关系，再复查一次，避免把已成为好友误报为待验证。
        if (userID && (await waitForBothWayFriend(userID))) {
          if (!scope.active()) return;
          syncEstablishedFriendOwner(userID);
          markCurrentContactAsFriend(userID);
          enterC2CChat(userID);
          void sendFriendGreetingMessageIfReady(userID);
          Toast({
            message: TUITranslateService.t("TUIContact.添加好友成功"),
            type: TOAST_TYPE.SUCCESS,
          });
          return;
        }
        if (!scope.active()) return;
        if (userID) {
          markCurrentContactRequestPending(userID);
        }
        Toast({
          message: TUITranslateService.t("TUIContact.申请已发送"),
          type: TOAST_TYPE.SUCCESS,
        });
        return;
      }

      Toast({
        message: outcome.message,
        type: TOAST_TYPE.ERROR,
      });
    } catch (error: any) {
      if (!scope.active()) return;
      console.warn("add friend failed:", error);
      if (
        to &&
        (collectErrorCodes(error).includes(
          ADD_FRIEND_RESULT_CODE_ALREADY_FRIENDS,
        ) ||
          (await waitForBothWayFriend(to)))
      ) {
        if (!scope.active()) return;
        syncEstablishedFriendOwner(to);
        markCurrentContactAsFriend(to);
        enterC2CChat(to);
        void sendFriendGreetingMessageIfReady(to);
        Toast({
          message: TUITranslateService.t("TUIContact.已经是好友"),
          type: TOAST_TYPE.WARNING,
        });
        return;
      }
      Toast({
        message: resolveAddFriendErrorMessage(error),
        type: TOAST_TYPE.ERROR,
      });
    }
  }).finally(() => {
    addFriendActions.delete(actionKey);
  });
};

// Enter conversation
export const enterConversation = (item: any) => {
  const conversationID = item?.groupID
    ? `GROUP${item?.groupID}`
    : `C2C${item?.userID}`;
  TUIConversationService.switchConversation(conversationID).catch(
    (error: any) => {
      console.warn("switch conversation failed:", error);
      Toast({
        message: TUITranslateService.t("TUIContact.进入会话失败"),
        type: TOAST_TYPE.ERROR,
      });
    }
  );
};

export function enterC2CChat(userID: string) {
  const trimmedUserID = String(userID || "").trim();
  if (!trimmedUserID) {
    return;
  }
  TUIStore.update(StoreName.CUSTOM, "currentContactSearchingStatus", false);
  TUIStore.update(StoreName.CUSTOM, "currentHomeNavBar", "message");
  TUIGlobal?.closeSearching?.();
  enterConversation({ userID: trimmedUserID });
}

const applicationActions = new Set<string>();
async function handleApplication(userID: string, accept: boolean) {
  const id = String(userID || '').trim();
  const scope=captureFriendOwnerScope();
  const owner = String(TUIChatEngine.chat?.getLoginUser() || '');
  const key = `${scope.key}:${id}`;
  if (!id || !owner || applicationActions.has(key)) return;
  const active = scope.active;
  applicationActions.add(key);
  try {
    const result = accept
      ? await TUIFriendService.acceptFriendApplication({ userID: id, type: TUIChatEngine.TYPES.SNS_APPLICATION_AGREE_AND_ADD })
      : await TUIFriendService.refuseFriendApplication(id);
    if (Number(result?.data?.code || 0)) throw Object.assign(new Error('申请处理未成功'), { code: result.data.code });
    if (!active()) return;
    if (accept) {
      syncEstablishedFriendOwner(id);
      markCurrentContactAsFriend(id);
      enterConversation({ userID: id });
      void sendFriendGreetingMessageIfReady(id);
    }
    Toast({ message: accept ? '已添加为好友' : '已拒绝申请', type: TOAST_TYPE.SUCCESS });
  } catch (error: any) {
    if (!active()) return;
    if (Number(error?.code) === 30614) {
      const added = await waitForBothWayFriend(id);
      if (!active()) return;
      if (added) { markCurrentContactAsFriend(id); if (accept) syncEstablishedFriendOwner(id); }
      Toast({ message: added ? '已添加为好友，请刷新列表' : '该申请已失效，请重新申请', type: added ? TOAST_TYPE.SUCCESS : TOAST_TYPE.WARNING });
    } else Toast({ message: error?.message || '好友申请处理失败', type: TOAST_TYPE.ERROR });
  } finally {
    applicationActions.delete(key);
    if (active()) {
      void TUIFriendService.getFriendList().catch(() => {});
      const sdk = TUIChatEngine.chat as any;
      void sdk?.getFriendApplicationList?.().catch(() => {});
    }
  }
}
export const acceptFriendApplication = (userID: string) => handleApplication(userID, true);
export const refuseFriendApplication = (userID: string) => handleApplication(userID, false);

const FRIEND_GREETING_TEXT = "我已添加你为好友，请开始聊天吧。";
const FRIEND_GREETING_TEXT_EN =
  "I have added you as a friend, let's start chatting.";
const FRIEND_GREETING_TEXTS = new Set([
  FRIEND_GREETING_TEXT,
  FRIEND_GREETING_TEXT_EN,
  "我已通过你的好友验证请求，现在我们可以开始聊天了",
  "I have accepted your friend request, now we can start chatting",
]);
const FRIEND_GREETING_DEDUP_TTL = 24 * 60 * 60 * 1000;
const friendGreetingPendingKeys = new Set<string>();

const sleep = (ms: number) =>
  new Promise((resolve) => {
    setTimeout(resolve, ms);
  });

function getFriendGreetingKey(userID: string) {
  const userInfo = TUIStore.getData(StoreName.USER, "userProfile") || {};
  const currentUserID = String(userInfo?.userID || "").trim() || "unknown";
  return `friend-greeting:${currentUserID}:${userID}`;
}

function hasRecentFriendGreeting(key: string) {
  try {
    const timestamp = Number(localStorage.getItem(key) || 0);
    return timestamp > 0 && Date.now() - timestamp < FRIEND_GREETING_DEDUP_TTL;
  } catch {
    return false;
  }
}

function markFriendGreetingSent(key: string) {
  try {
    localStorage.setItem(key, String(Date.now()));
  } catch {
    // Ignore storage failures; the in-memory pending set still prevents concurrent duplicates.
  }
}

function isRecentMessageTime(messageTime: unknown) {
  const timeInSeconds = Number(messageTime || 0);
  if (!Number.isFinite(timeInSeconds) || timeInSeconds <= 0) {
    return false;
  }
  const elapsed = Date.now() - timeInSeconds * 1000;
  return elapsed >= 0 && elapsed < FRIEND_GREETING_DEDUP_TTL;
}

function hasRecentLoadedFriendGreeting(userID: string) {
  const userInfo = TUIStore.getData(StoreName.USER, "userProfile") || {};
  const currentUserID = String(userInfo?.userID || "").trim();
  if (!currentUserID) {
    return false;
  }

  const conversationID = `C2C${userID}`;
  const messageList = TUIStore.getData(StoreName.CHAT, "messageList") || [];
  if (!Array.isArray(messageList)) {
    return false;
  }

  return messageList.some((message: any) => {
    if (
      !message ||
      message.conversationID !== conversationID ||
      message.from !== currentUserID ||
      message.type !== TUIChatEngine.TYPES.MSG_TEXT
    ) {
      return false;
    }
    const text = String(message.payload?.text || "").trim();
    return (
      FRIEND_GREETING_TEXTS.has(text) &&
      isRecentMessageTime(
        message.time || message.clientTime || message.serverTime
      )
    );
  });
}

async function sendFriendGreetingMessageIfReady(userID: string) {
  const trimmedUserID = String(userID || "").trim();
  if (!trimmedUserID) {
    return;
  }

  const greetingKey = getFriendGreetingKey(trimmedUserID);
  if (friendGreetingPendingKeys.has(greetingKey)) {
    return;
  }
  if (hasRecentFriendGreeting(greetingKey)) {
    return;
  }
  if (hasRecentLoadedFriendGreeting(trimmedUserID)) {
    markFriendGreetingSent(greetingKey);
    return;
  }
  // 先占位，避免同意按钮连点 / 列表+详情双入口把问候发出多条。
  friendGreetingPendingKeys.add(greetingKey);
  markFriendGreetingSent(greetingKey);

  try {
    // 对齐 App：不需要等对方同意好友，添加后即可发消息。
    await sleep(300);
    if (hasRecentLoadedFriendGreeting(trimmedUserID)) {
      return;
    }

    const userInfo = TUIStore.getData(StoreName.USER, "userProfile") || {};
    const language = String(userInfo?.language || "").toLowerCase();
    const greetingText = language.startsWith("en")
      ? FRIEND_GREETING_TEXT_EN
      : FRIEND_GREETING_TEXT;

    const options: SendMessageParams = {
      to: trimmedUserID,
      conversationType: TUIChatEngine.TYPES.CONV_C2C as any,
      payload: {
        text: greetingText,
      },
      needReadReceipt: false,
    };

    await TUIChatService.sendTextMessage(options);
  } catch (error) {
    console.warn("sendFriendGreetingMessageIfReady failed:", error);
  }
}

// Dismiss group
export const dismissGroup = (groupID: string) => {
  TUIGroupService.dismissGroup(groupID)
    .then(() => {
      Toast({
        message: TUITranslateService.t("TUIContact.解散群聊成功"),
        type: TOAST_TYPE.SUCCESS,
      });
      TUIGlobal?.updateContactSearch && TUIGlobal?.updateContactSearch();
    })
    .catch((error: any) => {
      console.warn("dismiss group failed:", error);
      Toast({
        message: TUITranslateService.t("TUIContact.解散群聊失败"),
        type: TOAST_TYPE.ERROR,
      });
    });
};

// Quit group
export const quitGroup = (groupID: string) => {
  TUIGroupService.quitGroup(groupID)
    .then(() => {
      Toast({
        message: TUITranslateService.t("TUIContact.退出群组成功"),
        type: TOAST_TYPE.SUCCESS,
      });
    })
    .catch((error: any) => {
      console.warn("quit group failed:", error);
      Toast({
        message: TUITranslateService.t("TUIContact.退出群组失败"),
        type: TOAST_TYPE.ERROR,
      });
    });
};

// Join group
let joinGroupPending = false;
export const joinGroup = (groupID: string, applyMessage?: string) => {
  if (joinGroupPending) {
    return Promise.resolve();
  }
  joinGroupPending = true;
  return withBusy("正在加入群聊...", async () => {
    try {
      const imResponse: { data: { status?: string } } = await TUIGroupService.joinGroup({
        groupID,
        applyMessage,
      } as JoinGroupParams);
      switch (imResponse?.data?.status) {
        case TUIChatEngine.TYPES.JOIN_STATUS_WAIT_APPROVAL: // Wait for administrator approval
          Toast({
            message: TUITranslateService.t("TUIContact.等待管理员同意"),
            type: TOAST_TYPE.SUCCESS,
          });
          break;
        case TUIChatEngine.TYPES.JOIN_STATUS_SUCCESS: // Join group successfully
          Toast({
            message: TUITranslateService.t("TUIContact.加群成功"),
            type: TOAST_TYPE.SUCCESS,
          });
          break;
        case TUIChatEngine.TYPES.JOIN_STATUS_ALREADY_IN_GROUP: // Already in the group
          Toast({
            message: TUITranslateService.t("TUIContact.您已是群成员"),
            type: TOAST_TYPE.SUCCESS,
          });
          break;
        default:
          break;
      }
    } catch (error: any) {
      console.warn("join group failed:", error);
      Toast({
        message: resolveJoinGroupErrorMessage(error),
        type: TOAST_TYPE.ERROR,
      });
    }
  }).finally(() => {
    joinGroupPending = false;
  });
};

// Add to blacklist
export const addToBlacklist = (
  userID: string,
  successCallBack?: () => void
) => {
  TUIUserService.addToBlacklist({
    userIDList: [userID],
  })
    .then(() => {
      successCallBack && successCallBack();
    })
    .catch((error: any) => {
      console.warn("add to blacklist failed:", error);
      Toast({
        message: TUITranslateService.t("TUIContact.加入黑名单失败"),
        type: TOAST_TYPE.ERROR,
      });
    });
};

// Remove from Blacklist
export const removeFromBlacklist = (
  userID: string,
  successCallBack?: () => void
) => {
  TUIUserService.removeFromBlacklist({
    userIDList: [userID],
  })
    .then(() => {
      successCallBack && successCallBack();
    })
    .catch((error: any) => {
      console.warn("remove from blacklist failed:", error);
      Toast({
        message: TUITranslateService.t("TUIContact.移除黑名单失败"),
        type: TOAST_TYPE.ERROR,
      });
    });
};
