import TUIChatEngine, { IMessageModel } from '@tencentcloud/chat-uikit-engine-lite';
import { JSONToObject } from '../../utils/type-check';
import {
  isLongdoCallControlPayload,
  isLongdoCallRecordPayload,
  isRawLongdoCallDisplayText,
  parseCallRecordDisplay,
} from '@/features/call/doMattxCallProtocol';

const CALL_BUSINESS_ID_LIST = ['av_call', 'rtc_call'];

const isCallBusinessID = (businessID: any): boolean => CALL_BUSINESS_ID_LIST.includes(businessID);

export function isCallMessage(message: IMessageModel): boolean {
  return Boolean(getCallMessageData(message));
}

export function getCallMessageData(message: IMessageModel): Record<string, any> | undefined {
  const payloadData = JSONToObject(message?.payload?.data);
  if (isLongdoCallControlPayload(payloadData) && !isLongdoCallRecordPayload(payloadData)) {
    return undefined;
  }
  const record = parseCallRecordDisplay(payloadData);
  if (record) {
    return {
      businessID: 'av_call',
      call_type: record.mediaType,
      actionType: record.actionType,
      call_end: record.callEnd,
      cmd: record.mediaType === 2 ? 'videoCall' : 'audioCall',
      summary: record.summary,
    };
  }
  if (payloadData?.businessID === 1 && payloadData?.data) {
    const payloadDataData = JSONToObject(payloadData.data);
    if (isCallBusinessID(payloadDataData?.businessID)) {
      return payloadDataData;
    }
  }
  if (isCallBusinessID(payloadData?.businessID)) {
    return payloadData;
  }
  return undefined;
}

export function getCallMessageDedupKey(message: IMessageModel): string {
  const callData = getCallMessageData(message);
  if (!callData) {
    return "";
  }
  const callType = Number(callData.call_type || (callData.cmd === 'videoCall' ? 2 : 1));
  const actionType = Number(callData.actionType || 0);
  const callEnd = Number(callData.call_end || 0);
  const resultKind = callEnd > 0 ? 'duration' : `action:${actionType}`;
  return `${callType}:${resultKind}`;
}

export function getCallMessageDurationSeconds(message: IMessageModel): number {
  const callData = getCallMessageData(message);
  return Number(callData?.call_end || 0);
}

export function getCallMessageCallerUserID(message: IMessageModel): string {
  const callData = getCallMessageData(message);
  const explicitCaller = String(
    callData?.caller || callData?.inviter || callData?.call_user_id || ''
  ).trim();
  if (explicitCaller) {
    return explicitCaller;
  }

  const actionType = Number(callData?.actionType || 0);
  const callEnd = Number(callData?.call_end || 0);
  if (actionType === 2 && callEnd <= 0) {
    return String(message?.from || '').trim();
  }

  return '';
}

export function getCallMessageSignalingInfo(message: IMessageModel): Record<string, any> | null {
  const callData = getCallMessageData(message);
  if (callData?.summary) {
    return {
      ...callData,
      data: JSON.stringify(callData),
    };
  }
  const signalingInfo = message?.getSignalingInfo?.();
  if (signalingInfo) {
    return signalingInfo;
  }
  if (!callData) {
    return null;
  }
  return {
    ...callData,
    data: JSON.stringify(callData),
  };
}

export function getCallMessageSummary(message: IMessageModel): string {
  const callData = getCallMessageData(message);
  const payload = message?.payload || {};
  const summary = String(callData?.summary || payload?.description || '').trim();
  if (
    summary &&
    !isRawLongdoCallDisplayText(summary) &&
    summary !== '发起通话' &&
    summary !== '[自定义消息]' &&
    summary !== '当前版本不支持该消息类型'
  ) {
    return summary;
  }

  const callType = callData?.call_type;
  const cmd = callData?.cmd;
  const mediaLabel = callType === 2 || cmd === 'videoCall' ? '视频通话' : '语音通话';
  const actionType = Number(callData?.actionType || 0);
  const callEnd = Number(callData?.call_end || 0);
  if (callEnd > 0) {
    const minutes = `${Math.floor(callEnd / 60)}`.padStart(2, '0');
    const seconds = `${callEnd % 60}`.padStart(2, '0');
    return `[${mediaLabel}] 通话时长 ${minutes}:${seconds}`;
  }
  switch (actionType) {
    case 2:
      return `[${mediaLabel}] 已取消`;
    case 4:
      return `[${mediaLabel}] 已拒绝`;
    case 5:
      return `[${mediaLabel}] 未接听`;
    default:
      return `[${mediaLabel}]`;
  }
}

export function isRoomSignalingMessage(message: IMessageModel): boolean {
  const payloadData = JSONToObject(message?.payload?.data);
  return (
    payloadData?.businessID === 'ROOM_INVITE_ACTION'
    || payloadData?.businessID === 'tuikit_engine_room'
  );
}

export function isRoomCardMessage(message: IMessageModel): boolean {
  const payloadData = JSONToObject(message?.payload?.data);
  return payloadData?.businessID === 'group_room_message';
}

export function isPluginMessage(message: IMessageModel): boolean {
  return (
    message.type === TUIChatEngine.TYPES.MSG_CUSTOM
    && (isCallMessage(message)
    || isRoomCardMessage(message)
    || isRoomSignalingMessage(message))
  );
}
