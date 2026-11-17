import { IChatOfflinePushInfo, ICallOfflinePushInfo } from './interface';

/** 与腾讯云小米证书、Flutter/服务端保持一致 */
export const XIAOMI_CHANNEL_ID = '157799';
export const XIAOMI_CHAT_TEMPLATE_ID = 'M12762';
export const OPPO_CHANNEL_ID = 'push_oplus_category_service';
export const OPPO_CATEGORY = 'IM';
export const OPPO_PRIVATE_TEMPLATE_ID = '6a8408344fae550145d04b8f';
export const OPPO_TITLE_PARAM_KEY = 'senderName';
export const OPPO_CONTENT_PARAM_KEY = 'text';

export const chatOfflinePushInfo: IChatOfflinePushInfo = {
  androidInfo: {
    XiaoMiChannelID: XIAOMI_CHANNEL_ID,
    OPPOChannelID: OPPO_CHANNEL_ID,
    OPPOCategory: OPPO_CATEGORY,
  },
  apnsInfo: {},
};

export const callOfflinePushInfo: ICallOfflinePushInfo = {};
