import {
  IGroupModel,
  TUIGroupService,
} from "@tencentcloud/chat-uikit-engine-lite";
import type { IGroupMember } from "../../../interface";

export interface IGroupConversationContext {
  group?: IGroupModel;
  selfMember?: IGroupMember;
  fetchedAt: number;
}

const GROUP_CONTEXT_CACHE_TTL = 3000;
let cacheGeneration = 0;
const groupConversationContextCache = new Map<
  string,
  IGroupConversationContext
>();
const groupConversationContextInflight = new Map<
  string,
  Promise<IGroupConversationContext>
>();

function getContextCacheKey(groupID: string, userID?: string) {
  return `${groupID}::${userID || ""}`;
}

async function requestGroupConversationContext(
  groupID: string,
  userID?: string
) {
  const [groupProfileResult, selfMemberProfileResult] = await Promise.all([
    TUIGroupService.getGroupProfile({ groupID }),
    userID
      ? TUIGroupService.getGroupMemberProfile({
          groupID,
          userIDList: [userID],
        })
      : Promise.resolve(undefined),
  ]);

  return {
    group: groupProfileResult?.data?.group,
    selfMember: selfMemberProfileResult?.data?.memberList?.[0],
    fetchedAt: Date.now(),
  } as IGroupConversationContext;
}

export async function getGroupConversationContext(
  groupID?: string,
  userID?: string,
  options: {
    force?: boolean;
  } = {}
) {
  if (!groupID) {
    return {
      group: undefined,
      selfMember: undefined,
      fetchedAt: Date.now(),
    } as IGroupConversationContext;
  }

  const cacheKey = getContextCacheKey(groupID, userID);
  const cachedContext = groupConversationContextCache.get(cacheKey);
  const shouldUseCache =
    !options.force &&
    cachedContext &&
    Date.now() - cachedContext.fetchedAt < GROUP_CONTEXT_CACHE_TTL;
  if (shouldUseCache) {
    return cachedContext;
  }

  const inflightRequest = groupConversationContextInflight.get(cacheKey);
  if (inflightRequest) {
    return inflightRequest;
  }

  const generation = cacheGeneration;
  const request = requestGroupConversationContext(groupID, userID)
    .then((context) => {
      if (generation !== cacheGeneration)
        throw new Error("群权限已更新，请重新确认后发送");
      if (groupConversationContextCache.size >= 200)
        groupConversationContextCache.delete(
          groupConversationContextCache.keys().next().value!
        );
      groupConversationContextCache.set(cacheKey, context);
      return context;
    })
    .finally(() => {
      if (groupConversationContextInflight.get(cacheKey) === request)
        groupConversationContextInflight.delete(cacheKey);
    });

  groupConversationContextInflight.set(cacheKey, request);
  return request;
}

export function preloadGroupConversationContext(
  groupID?: string,
  userID?: string
) {
  if (!groupID) {
    return;
  }
  void getGroupConversationContext(groupID, userID).catch(() => undefined);
}

export function clearGroupConversationContext(
  groupID?: string,
  userID?: string
) {
  cacheGeneration++;
  if (!groupID) {
    groupConversationContextCache.clear();
    groupConversationContextInflight.clear();
    return;
  }

  const cacheKey = getContextCacheKey(groupID, userID);
  groupConversationContextCache.delete(cacheKey);
  groupConversationContextInflight.delete(cacheKey);
}
