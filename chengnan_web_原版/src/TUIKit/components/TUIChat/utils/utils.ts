import TUIChatEngine, { TUITranslateService, TUIStore, StoreName, IMessageModel } from '@tencentcloud/chat-uikit-engine-lite';

export function deepCopy(data: any, hash = new WeakMap()) {
  if (typeof data !== 'object' || data === null || data === undefined) {
    return data;
  }
  if (hash.has(data)) {
    return hash.get(data);
  }
  const newData: any = Object.create(Object.getPrototypeOf(data));
  const dataKeys = Object.keys(data);
  dataKeys.forEach((value) => {
    const currentDataValue = data[value];
    if (typeof currentDataValue !== 'object' || currentDataValue === null) {
      newData[value] = currentDataValue;
    } else if (Array.isArray(currentDataValue)) {
      newData[value] = [...currentDataValue];
    } else if (currentDataValue instanceof Set) {
      newData[value] = new Set([...currentDataValue]);
    } else if (currentDataValue instanceof Map) {
      newData[value] = new Map([...currentDataValue]);
    } else {
      hash.set(data, data);
      newData[value] = deepCopy(currentDataValue, hash);
    }
  });
  return newData;
}

export const handleSkeletonSize = (
  width: number,
  height: number,
  maxWidth: number,
  maxHeight: number,
): { width: number; height: number } => {
  const widthToHeight = width / height;
  const maxWidthToHeight = maxWidth / maxHeight;
  if (width <= maxWidth && height <= maxHeight) {
    return { width, height };
  }
  if (
    (width <= maxWidth && height > maxHeight)
    || (width > maxWidth && height > maxHeight && widthToHeight <= maxWidthToHeight)
  ) {
    return { width: width * (maxHeight / height), height: maxHeight };
  }
  return { width: maxWidth, height: height * (maxWidth / width) };
};

// Image loading complete
export function getImgLoad(container: any, className: string, callback: any) {
  const images = container?.querySelectorAll(`.${className}`) || [];
  const promiseList = Array.prototype.slice.call(images).map((node: any) => {
    return new Promise((resolve: any) => {
      node.onload = () => {
        resolve(node);
      };
      node.onloadeddata = () => {
        resolve(node);
      };
      node.onprogress = () => {
        resolve(node);
      };
      if (node.complete) {
        resolve(node);
      }
    });
  });
  return Promise.all(promiseList)
    .then(() => {
      callback && callback();
    })
    .catch((e) => {
      console.error('网络异常', e);
    });
}

export const isCreateGroupCustomMessage = (message: IMessageModel) => {
  return (
    message.type === TUIChatEngine.TYPES.MSG_CUSTOM
    && message?.getMessageContent()?.businessID === 'group_create'
  );
};

export const isGroupMemberInviteCustomMessage = (message: IMessageModel) => {
  return getCustomBusinessID(message) === 'mllx_group_member_invite';
};

export const isGroupCollabCustomMessage = (message: IMessageModel) => {
  return getCustomBusinessID(message) === 'mllx_group_collab';
};

function getCustomBusinessID(message: IMessageModel) {
  if (message?.type !== TUIChatEngine.TYPES.MSG_CUSTOM) {
    return '';
  }
  try {
    const raw = message?.payload?.data;
    const payload = typeof raw === 'string' ? JSON.parse(raw || '{}') : (raw || {});
    return String(payload?.businessID || '').trim();
  } catch (_) {
    return '';
  }
}

export const getGroupMemberInviteCustomContent = (message: IMessageModel) => {
  try {
    const payload = JSON.parse(message?.payload?.data || '{}');
    if (payload?.businessID !== 'mllx_group_member_invite') {
      return message.getMessageContent();
    }
    const inviterName = String(
      payload?.inviter?.nick || payload?.inviter?.userID || '成员'
    ).trim();
    const memberNames = Array.isArray(payload?.members)
      ? payload.members
          .map((member: any) => String(member?.nick || member?.userID || '').trim())
          .filter(Boolean)
      : [];
    const fallbackText = memberNames.length
      ? `${inviterName} 邀请 ${memberNames.join('、')} 加入群聊`
      : `${inviterName} 邀请成员加入群聊`;
    return { text: String(payload?.content || '').trim() || fallbackText };
  } catch (_) {
    return message.getMessageContent();
  }
};

export const getGroupCollabCustomContent = (message: IMessageModel) => {
  const fallbackDesc = String(message?.payload?.description || '').trim();
  try {
    const raw = message?.payload?.data;
    const payload = typeof raw === 'string' ? JSON.parse(raw || '{}') : (raw || {});
    if (payload?.businessID !== 'mllx_group_collab') {
      return message.getMessageContent();
    }
    const text = String(payload?.text || '').trim();
    if (text) {
      return { text };
    }
    const actor = String(payload?.opUser || '成员').trim() || '成员';
    const title = String(payload?.title || '').trim();
    const action = String(payload?.action || '');
    const actionText: Record<string, string> = {
      todo_created: title ? `${actor} 创建了待办「${title}」` : `${actor} 创建了群待办`,
      todo_done: title ? `${actor} 完成了待办「${title}」` : `${actor} 完成了群待办`,
      todo_deleted: title ? `${actor} 删除了待办「${title}」` : `${actor} 删除了群待办`,
      event_created: title ? `${actor} 创建了日程「${title}」` : `${actor} 创建了共享日程`,
      event_deleted: title ? `${actor} 删除了日程「${title}」` : `${actor} 删除了共享日程`,
      ledger_created: title ? `${actor} 记账「${title}」` : `${actor} 记了一笔群账本`,
      ledger_deleted: title ? `${actor} 删除了记账「${title}」` : `${actor} 删除了群账本记录`,
    };
    return {
      text: actionText[action]
        || fallbackDesc
        || (title ? `${actor} 更新了协作「${title}」` : `${actor} 更新了群协作`),
    };
  } catch (_) {
    return fallbackDesc ? { text: fallbackDesc } : message.getMessageContent();
  }
};

/**
 * displayMessageReadReceipt: User-level control to display message read status
 * After turning off, the messages you send and receive will not have message read status
 * You will not be able to see whether the other party has read their messages, and they will also not be able to see whether you have read the messages they sent
 *
 * enabledMessageReadReceipt: App-level setting to enable read receipts
 * @return {boolean} - Returns a boolean value indicating if the message read receipt is enabled globally.
 */
export function isEnabledMessageReadReceiptGlobal(): boolean {
  return TUIStore.getData(StoreName.USER, 'displayMessageReadReceipt')
    && TUIStore.getData(StoreName.APP, 'enabledMessageReadReceipt');
}

export function shallowCopyMessage(message: IMessageModel) {
  return Object.assign({}, message);
}

// calculate timestamp
export function calculateTimestamp(timestamp: number): string {
  const todayZero = new Date().setHours(0, 0, 0, 0);
  const thisYear = new Date(
    new Date().getFullYear(),
    0,
    1,
    0,
    0,
    0,
    0,
  ).getTime();
  const target = new Date(timestamp);

  const oneDay = 24 * 60 * 60 * 1000;
  const oneWeek = 7 * oneDay;

  const diff = todayZero - target.getTime();

  function formatNum(num: number): string {
    return num < 10 ? '0' + num : num.toString();
  }

  if (diff <= 0) {
    // today, only display hour:minute
    return `${formatNum(target.getHours())}:${formatNum(target.getMinutes())}`;
  } else if (diff <= oneDay) {
    // yesterday, display yesterday:hour:minute
    return `${TUITranslateService.t('time.昨天')} ${formatNum(
      target.getHours(),
    )}:${formatNum(target.getMinutes())}`;
  } else if (diff <= oneWeek - oneDay) {
    // Within a week, display weekday hour:minute
    const weekdays = [
      '星期日',
      '星期一',
      '星期二',
      '星期三',
      '星期四',
      '星期五',
      '星期六',
    ];
    const weekday = weekdays[target.getDay()];
    return `${TUITranslateService.t('time.' + weekday)} ${formatNum(
      target.getHours(),
    )}:${formatNum(target.getMinutes())}`;
  } else if (target.getTime() >= thisYear) {
    // Over a week, within this year, display mouth/day hour:minute
    return `${target.getMonth() + 1}/${target.getDate()} ${formatNum(
      target.getHours(),
    )}:${formatNum(target.getMinutes())}`;
  } else {
    // Not within this year, display year/mouth/day hour:minute
    return `${target.getFullYear()}/${
      target.getMonth() + 1
    }/${target.getDate()} ${formatNum(target.getHours())}:${formatNum(
      target.getMinutes(),
    )}`;
  }
}
