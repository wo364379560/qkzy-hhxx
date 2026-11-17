// Compatibility adapter for chat-uikit-engine-lite 1.0.1. Its history methods
// otherwise commit pagination to the active conversation after every response.
// Install on the service itself so initial loads and quote/@ jumps are covered.
export function installScopedHistory(engine: any, service: any, store: any, names: any) {
  if (service.__scopedHistoryInstalled) return;
  service.__scopedHistoryInstalled = true;
  let revision = 0;
  const pending = new Map<string, Promise<any>>();
  const invalidate = () => { revision += 1; pending.clear(); };
  store.watch(names.CONV, { currentConversationID: invalidate });
  store.watch(names.CHAT, { messageSource: invalidate });
  const read = (key: string) => store.getData(names.CHAT, key);
  const activeID = () => store.getData(names.CONV, 'currentConversationID');

  function load(kind: 'page' | 'jump', options: any): Promise<any> {
    const chat = engine.chat;
    if (!chat?.isReady()) return Promise.reject(new Error('聊天连接尚未就绪'));
    const epoch = revision;
    const source = read('messageSource');
    const key = JSON.stringify([epoch, kind, options]);
    const existing = pending.get(key);
    if (existing) return existing;
    const isCurrent = () => revision === epoch && engine.chat === chat
      && activeID() === options.conversationID && read('messageSource') === source;
    const request = Promise.resolve().then(() => kind === 'page'
      ? chat.getMessageList(options) : chat.getMessageListHopping(options))
      .then((response: any) => {
        if (!isCurrent()) return { ...response, stale: true };
        const { messageList = [], nextReqMessageID, nextMessageSeq, nextMessageTime, isCompleted } = response.data;
        if (kind === 'page') {
          const userInfo = read('userInfo') || {};
          if (Object.keys(userInfo).length) service.updateLocalMessage(messageList, userInfo);
        }
        service.updateMessageList(messageList, 'unshift');
        if (kind === 'page') {
          store.update(names.CHAT, 'nextReqMessageID', nextReqMessageID);
        } else {
          service.delayGetHoppingFunction = undefined;
          service.hoppingConfigMap.set('nextMessageSeq', options.direction === 1 ? options.sequence : nextMessageSeq);
          service.hoppingConfigMap.set('nextMessageTime', options.direction === 1 ? options.time : nextMessageTime);
        }
        store.update(names.CHAT, 'isCompleted', isCompleted);
        if (kind === 'page' && messageList.length
          && !(store.getConversationModel(options.conversationID)?.operationType || 0)) {
          // Optional metadata must not reject a successfully committed page.
          for (const extra of [
            () => service.getMessageReactions({ messageList }),
            () => service.readReceiptHandler.getMessageReadReceiptList(messageList),
          ]) {
            try { Promise.resolve(extra()).catch(() => undefined); } catch { /* optional SDK feature */ }
          }
        }
        return response;
      }, (error: any) => {
        if (!isCurrent()) return { stale: true, data: { messageList: [] } };
        throw error;
      }).finally(() => { if (pending.get(key) === request) pending.delete(key); });
    pending.set(key, request);
    return request;
  }

  service.getMessageList = (options = {
    conversationID: activeID(), nextReqMessageID: read('nextReqMessageID'),
  }) => {
    if (read('isCompleted')) {
      return Promise.resolve({ data: { messageList: [], nextReqMessageID: '', isCompleted: true } });
    }
    const source = read('messageSource');
    if (source?.conversationID === options.conversationID
      && (service.hoppingConfigMap.get('nextMessageSeq') || service.hoppingConfigMap.get('nextMessageTime'))) {
      return service.getMessageListHopping();
    }
    return load('page', options);
  };
  service.getMessageListHopping = (options = {
    conversationID: read('messageSource')?.conversationID,
    sequence: service.hoppingConfigMap.get('nextMessageSeq'),
    time: service.hoppingConfigMap.get('nextMessageTime'),
  }) => load('jump', options);
}
