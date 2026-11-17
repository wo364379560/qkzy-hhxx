import TUICore, { TUIConstants } from '@tencentcloud/tui-core-lite';
import { TUITranslateService } from '@tencentcloud/chat-uikit-engine-lite';
import { isPC } from '../../../utils/env';
import { getProfile } from '@/api';
import createGroup from '../../../assets/icon/start-group.svg';
import C2C from '../../../assets/icon/icon-c2c.svg';
import { CONV_CREATE_TYPE } from '../../../constant';
import { Toast, TOAST_TYPE } from '../../common/Toast/index';

export interface IMenuItem {
  icon?: string;
  text: string;
  data: {
    name: string;
    children?: any[];
  };
  listener?: {
    onClicked: (...args: any[]) => void;
  };
}

export default class ConversationHeaderServer {
  static instance: ConversationHeaderServer;

  static getInstance(): ConversationHeaderServer {
    if (!ConversationHeaderServer.instance) {
      ConversationHeaderServer.instance = new ConversationHeaderServer();
    }
    return ConversationHeaderServer.instance;
  }

  public getMenu(): any[] {
    const list = this.generateMenuList();
    if (!isPC && list.length > 0) {
      return [{
        text: TUITranslateService.t('TUIConversation.发起会话'),
        data: {
          name: 'all',
          children: list,
        },
      }];
    }
    return list;
  }

  private generateMenuList(): any[] {
    const list = [
      {
        icon: C2C,
        text: TUITranslateService.t('TUIConversation.发起单聊'),
        data: {
          name: CONV_CREATE_TYPE.TYPEC2C,
        },
        listener: {
          onClicked: this.createConversation.bind(this),
        },
      },
      {
        icon: createGroup,
        text: TUITranslateService.t('TUIConversation.发起群聊'),
        data: {
          name: CONV_CREATE_TYPE.TYPEGROUP,
        },
        listener: {
          onClicked: this.createConversation.bind(this),
        },
      },
    ];
    return list;
  }

  private createConversation(item: IMenuItem) {
    if (item.data.name === CONV_CREATE_TYPE.TYPEGROUP) {
      this.ensureCurrentUserCanCreateGroup().then((canCreate) => {
        if (!canCreate) {
          return;
        }
        this.callCreateConversation(item);
      });
      return;
    }
    this.callCreateConversation(item);
  }

  private callCreateConversation(item: IMenuItem) {
    // Create a conversation and notify conversationServer via TUICore.callService
    TUICore.callService({
      serviceName: TUIConstants.TUIConversation.SERVICE.NAME,
      method: TUIConstants.TUIConversation.SERVICE.METHOD.CREATE_CONVERSATION,
      params: item,
    });
  }

  private async ensureCurrentUserCanCreateGroup() {
    try {
      const profile = await getProfile();
      if (profile?.can_create_group) {
        return true;
      }
      Toast({
        message: '当前账号未开启建群权限，请联系管理员',
        type: TOAST_TYPE.WARNING,
      });
      return false;
    } catch (error: any) {
      Toast({
        message: error?.message || '获取建群权限失败',
        type: TOAST_TYPE.ERROR,
      });
      return false;
    }
  }
}
