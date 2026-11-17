import i18next from 'i18next';
import { TUIChatKit } from '@/TUIKit/chat-kit';
import TUIChatEngine, {
  IMessageModel,
  TUIConversationService,
  TUIStore,
  StoreName,
  TUITranslateService,
} from '@tencentcloud/chat-uikit-engine-lite';
import TUINotification from '@/TUIKit/components/TUINotification/index';
import { isTypingMessage } from '@/TUIKit/components/TUINotification/utils';
import { locales } from '@/locales';
import { APP_NAME } from '@/config/runtime';
import messageSound from '@/utils/messageSound';
import { installGroupSendGuard } from '@/features/group/sendGuard';
import { clearGroupConversationContext } from '@/TUIKit/components/TUIChat/utils/groupConversationContext';

let isRuntimeInitialized = false;
let currentUserID = '';
let currentConversationID = '';
let kickedOutHandler: ((value: string) => void | Promise<void>) | null = null;

const TYPES = TUIChatEngine.TYPES;
const FRIEND_GREETING_MESSAGES = new Set([
  '我已通过你的好友验证请求，现在我们可以开始聊天了',
  'I have accepted your friend request, now we can start chatting',
]);

function onUserProfileUpdated(profile: { userID?: string } | undefined) {
  if (currentUserID !== (profile?.userID || '')) clearGroupConversationContext();
  currentUserID = profile?.userID || '';
}

function onCurrentConversationUpdated(
  conversation: { conversationID?: string } | undefined,
) {
  currentConversationID = conversation?.conversationID || '';
}

function isMessageNeedSound(message: IMessageModel): boolean {
  if (!message || !message.ID || !message.type || message.isRevoked || message.isDeleted || isTypingMessage(message)) {
    return false;
  }

  if (message.type === TYPES.MSG_GRP_TIP || message.type === TYPES.MSG_GRP_SYS_NOTICE) {
    return false;
  }

  if (!message.conversationID) {
    return false;
  }

  if (currentUserID && message.from === currentUserID) {
    return false;
  }

  return true;
}

function isFriendGreetingMessage(message: IMessageModel): boolean {
  if (
    !message
    || !message.conversationID
    || message.isRevoked
    || message.isDeleted
    || message.type !== TYPES.MSG_TEXT
  ) {
    return false;
  }

  if (currentUserID && message.from === currentUserID) {
    return false;
  }

  const text = String(message.payload?.text || '').trim();
  if (!text || !FRIEND_GREETING_MESSAGES.has(text)) {
    return false;
  }

  return message.conversationID.startsWith('C2C');
}

function autoOpenConversationForFriendGreeting(message: IMessageModel) {
  if (!isFriendGreetingMessage(message)) {
    return;
  }

  // Do not interrupt the conversation the user is already reading or replying in.
  if (
    currentConversationID &&
    currentConversationID !== message.conversationID
  ) {
    return;
  }

  TUIStore.update(StoreName.CUSTOM, 'currentHomeNavBar', 'message');
  TUIConversationService.switchConversation(message.conversationID).catch(
    (error: unknown) => {
      console.warn(
        'auto open conversation for friend greeting failed:',
        error,
      );
    },
  );
}

function onNewMessageListUpdated(newMessageList: unknown) {
  if (!Array.isArray(newMessageList)) {
    return;
  }

  const typedMessages = newMessageList as IMessageModel[];
  typedMessages.forEach(message => TUINotification.notify(message));
  typedMessages.forEach(autoOpenConversationForFriendGreeting);

  if (typedMessages.some(message => isMessageNeedSound(message))) {
    void messageSound.play();
  }
}

function onKickedOutUpdated(value: string) {
  if (value && kickedOutHandler) {
    void kickedOutHandler(value);
  }
}

export function ensureChatRuntime(options?: {
  onKickedOut?: (value: string) => void | Promise<void>;
}) {
  if (options?.onKickedOut) {
    kickedOutHandler = options.onKickedOut;
  }

  if (isRuntimeInitialized) {
    return;
  }
  isRuntimeInitialized = true;

  TUIStore.update(StoreName.USER, 'displayMessageReadReceipt', true);

  TUITranslateService.setI18next(i18next);
  TUITranslateService.provideLanguages(locales);
  TUITranslateService.useI18n();
  messageSound.installUnlockOnInteraction();

  TUINotification.setNotificationConfiguration({
    showPreviews: true,
    allowNotifications: true,
    notificationTitle: APP_NAME,
    notificationIcon: '/favicon.png',
  });

  TUIStore.watch(StoreName.USER, {
    userProfile: onUserProfileUpdated,
    kickedOut: onKickedOutUpdated,
  });

  TUIStore.watch(StoreName.CONV, {
    currentConversation: onCurrentConversationUpdated,
  });

  TUIStore.watch(StoreName.CHAT, {
    newMessageList: onNewMessageListUpdated,
  });
}

export function loginChatRuntime(options?: {
  onKickedOut?: (value: string) => void | Promise<void>;
}) {
  ensureChatRuntime(options);
  installGroupSendGuard(TUIChatEngine.chat);
  TUIChatKit.login();
}
