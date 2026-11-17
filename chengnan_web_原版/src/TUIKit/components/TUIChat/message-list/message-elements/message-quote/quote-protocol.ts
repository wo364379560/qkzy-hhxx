import type { IQuoteContent } from './interface';

const text = (value: unknown): string => typeof value === 'string' ? value.trim() : '';
const number = (value: unknown, fallback = 0): number => {
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : fallback;
};

function normalize(value: unknown, native: boolean): IQuoteContent | undefined {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return;
  const payload = value as Record<string, unknown>;
  const messageID = text(payload[native ? 'msgID' : 'messageID']);
  if (!messageID) return;
  return {
    messageID,
    messageSender: text(payload[native ? 'msgSender' : 'messageSender']),
    messageAbstract: text(payload[native ? 'msgAbstract' : 'messageAbstract']),
    messageType: number(payload.messageType ?? 1, 1),
    messageSequence: number(payload.messageSequence),
    messageTime: number(payload.messageTime),
    version: number(payload.version ?? 1, 1),
  };
}

export function decodeMessageQuote(raw: unknown): IQuoteContent | undefined {
  if (typeof raw !== 'string' || !raw.trim()) return;
  try {
    const root = JSON.parse(raw);
    if (!root || typeof root !== 'object' || Array.isArray(root)) return;
    return normalize(root.messageReply, false) || normalize(root.atomicx_quote_v1, true);
  } catch { return; }
}

/** Preserve the web envelope and add the AtomicX envelope used by native clients. */
export function encodeMessageQuote(message: any): string {
  const types: Record<string, number> = { TIMTextElem: 1, TIMCustomElem: 2, TIMImageElem: 3, TIMSoundElem: 4, TIMVideoFileElem: 5, TIMFileElem: 6, TIMLocationElem: 7, TIMFaceElem: 8, TIMGroupTipElem: 9, TIMRelayElem: 10 };
  const labels: Record<number, string> = { 2: '[自定义消息]', 3: '[图片]', 4: '[语音]', 5: '[视频]', 6: '[文件]', 7: '[位置]', 8: '[表情]', 9: '[群提示]', 10: '[聊天记录]' };
  const messageType = types[message.type] || 1;
  const messageSender = message.nameCard || message.nick || message.from || '';
  const messageAbstract = messageType === 1 ? String(message.payload?.text || '').slice(0, 100) : labels[messageType];
  const reply = { messageID: message.ID, messageSender, messageAbstract, messageType, messageTime: message.time || 0, messageSequence: message.sequence || 0, version: 1 };
  return JSON.stringify({ messageReply: reply, atomicx_quote_v1: { msgID: reply.messageID, msgSender: messageSender, msgAbstract: messageAbstract, messageType, messageTime: reply.messageTime, messageSequence: reply.messageSequence } });
}
