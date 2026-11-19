/** A session-scoped adapter for CallEngine 3.4.x's public Chat event contract.
 * The real Chat keeps delivering messages normally. Only the call consumer is
 * delayed until its engine is initialized; events are never re-emitted on Chat.
 * Bounded memory/time, and no persisted invitations or credentials. */
export function createCallSignalInbox(chat: any, options: {
  active: () => boolean;
  userID: string;
  events: Record<string, string>;
  signaling: Record<string, string>;
  now?: () => number;
  maxEvents?: number;
  maxBytes?: number;
  maxAgeMs?: number;
}) {
  const now = options.now || Date.now;
  const maxEvents = options.maxEvents || 256;
  const maxBytes = options.maxBytes || 512 * 1024;
  const maxAgeMs = options.maxAgeMs || 60000;
  type Handler = { fn: (...args: any[]) => void; context: unknown };
  type Entry = { key: string; args: any[]; at: number; bytes: number };
  const handlers = new Map<string, Handler[]>();
  const detach: Array<() => void> = [];
  let buffered: Entry[] = [], bytes = 0, delivering = false, disposed = false;
  const active = () => !disposed && options.active();
  const signalNames = Object.values(options.signaling).filter(Boolean);
  const eventNames = ['MESSAGE_RECEIVED', 'ROOM_CUSTOM_DATA_RECEIVED', 'SERVER_CONFIG_UPDATED']
    .map(key => options.events[key]).filter(Boolean);
  const captured = new Set([...signalNames.map(n => `s:${n}`), ...eventNames.map(n => `e:${n}`)]);

  function prune() {
    while (buffered.length && (now() - buffered[0].at > maxAgeMs || buffered.length > maxEvents || bytes > maxBytes)) {
      bytes -= buffered.shift()!.bytes;
    }
  }
  function dispatch(key: string, args: any[]) {
    for (const listener of [...(handlers.get(key) || [])]) {
      if (!active()) return;
      try { listener.fn.apply(listener.context, args); } catch { /* One consumer cannot stop incoming chat. */ }
    }
  }
  function receive(key: string, args: any[]) {
    if (!active()) return;
    if (delivering) { dispatch(key, args); return; }
    // Call signaling in ordinary message delivery uses custom elements. Keep
    // room-control and signaling events intact, including standard/group calls.
    if (key === `e:${options.events.MESSAGE_RECEIVED}` && Array.isArray(args[0]?.data)) {
      const data = args[0].data.filter((m: any) => m?.type === 'TIMCustomElem');
      if (!data.length) return;
      args = [{ ...args[0], data }, ...args.slice(1)];
    }
    let size: number;
    try { size = new TextEncoder().encode(JSON.stringify(args)).length; } catch { return; }
    if (size > maxBytes) return;
    buffered.push({ key, args, bytes: size, at: now() }); bytes += size; prune();
  }
  for (const name of signalNames) {
    const handler = (...args: any[]) => receive(`s:${name}`, args);
    chat.addSignalingListener(name, handler);
    detach.push(() => chat.removeSignalingListener?.(name, handler));
  }
  for (const name of eventNames) {
    const handler = (...args: any[]) => receive(`e:${name}`, args);
    chat.on(name, handler);
    detach.push(() => chat.off?.(name, handler));
  }
  function add(key: string, fn: Handler['fn'], context: unknown) {
    if (!active()) return;
    const list = handlers.get(key) || [];
    if (!list.some(h => h.fn === fn && h.context === context)) list.push({ fn, context });
    handlers.set(key, list);
  }
  function remove(key: string, fn: Handler['fn'], context: unknown) {
    handlers.set(key, (handlers.get(key) || []).filter(h => h.fn !== fn || h.context !== context));
  }
  const forwarded = new Map<string, Array<{ fn: Handler['fn']; context: unknown; wrapper: Handler['fn'] }>>();
  const facade = new Proxy(chat, {
    get(target, prop) {
      if (prop === 'addSignalingListener') return (name: string, fn: Handler['fn'], context?: unknown) => add(`s:${name}`, fn, context);
      if (prop === 'removeSignalingListener') return (name: string, fn: Handler['fn'], context?: unknown) => remove(`s:${name}`, fn, context);
      if (prop === 'on') return (name: string, fn: Handler['fn'], context?: unknown) => {
        if (captured.has(`e:${name}`)) { add(`e:${name}`, fn, context); return; }
        if (!active()) return;
        const list = forwarded.get(name) || [];
        if (list.some(h => h.fn === fn && h.context === context)) return;
        const wrapper = (...args: any[]) => { if (active()) fn.apply(context, args); };
        list.push({ fn, context, wrapper }); forwarded.set(name, list); target.on(name, wrapper);
      };
      if (prop === 'off') return (name: string, fn: Handler['fn'], context?: unknown) => {
        remove(`e:${name}`, fn, context);
        const list = forwarded.get(name) || [];
        for (const h of list) if (h.fn === fn && h.context === context) target.off(name, h.wrapper);
        forwarded.set(name, list.filter(h => h.fn !== fn || h.context !== context));
      };
      // TUILogin alone owns the business Chat connection. A late CallEngine
      // login/destroy must never log out or replace the next business account.
      if (prop === 'logout' || prop === 'destroy') return async () => ({ code: 0 });
      if (prop === 'login') return async (params: { userID?: string }) => {
        if (!active() || params.userID !== options.userID || target.getLoginUser?.() !== options.userID || !target.isReady?.()) {
          throw new Error('Call session is no longer ready');
        }
        return { code: 0, data: { repeatLogin: true } };
      };
      const value = Reflect.get(target, prop, target);
      if (typeof value !== 'function') return value;
      return (...args: any[]) => {
        if (!active() && !/^(get|is)/.test(String(prop))) throw new Error('Call session changed');
        return value.apply(target, args);
      };
    },
  });
  return {
    chat: facade,
    activate() {
      if (!active() || delivering) return;
      prune();
      const terminated = new Set<string>();
      for (const entry of buffered) {
        const payload = entry.args[0]?.data || entry.args[0];
        if (entry.key === `s:${options.signaling.INVITATION_CANCELLED}` ||
            (entry.key === `s:${options.signaling.INVITATION_TIMEOUT}` &&
              (!Array.isArray(payload?.inviteeList) || payload.inviteeList.includes(options.userID)))) {
          const id = payload?.inviteID || payload?.inviteId;
          if (id) terminated.add(id);
        }
      }
      // Keep the queue active during delivery, so reentrant transport events
      // follow older buffered events instead of overtaking them.
      while (buffered.length && active()) {
        const entry = buffered.shift()!; bytes -= entry.bytes;
        const payload = entry.args[0]?.data || entry.args[0];
        if (entry.key === `s:${options.signaling.NEW_INVITATION_RECEIVED}` && terminated.has(payload?.inviteID || payload?.inviteId)) continue;
        dispatch(entry.key, entry.args);
      }
      delivering = true;
    },
    suspend() { delivering = false; handlers.clear(); },
    pendingCount() { prune(); return buffered.length; },
    dispose() {
      disposed = true; buffered = []; bytes = 0; handlers.clear();
      for (const fn of detach) fn();
      for (const [name, list] of forwarded) for (const h of list) chat.off?.(name, h.wrapper);
      forwarded.clear();
    },
  };
}
