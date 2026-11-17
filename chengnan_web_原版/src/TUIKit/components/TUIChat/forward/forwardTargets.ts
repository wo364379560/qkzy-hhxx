// Preserve every originally selected target, including failed model resolution.
export async function resolveForwardTargets<T extends { conversationID: string }>(items: T[], resolve: (item: T) => any) {
  return Promise.all(items.map(async (item, index) => {
    try {
      const target = await resolve(item);
      if (!target?.conversationID) throw new Error('会话资料缺失');
      return target;
    } catch {
      return { conversationID: item.conversationID || `unresolved-${index}`, resolutionError: '会话解析失败，请重新选择该目标' };
    }
  }));
}
