import TUIChatEngine from '@tencentcloud/chat-uikit-engine-lite';

const communityGroupIntroConfig = {
  icon: 'https://web.sdk.qcloud.com/im/assets/images/Community.png',
  label: '社群',
  type: TUIChatEngine.TYPES.GRP_COMMUNITY,
};

const groupIntroConfig = [
  communityGroupIntroConfig,
];

const findGroupIntroConfig = (type: string) => {
  return groupIntroConfig.filter((item: any) => {
    return item.type === type;
  })[0] || communityGroupIntroConfig;
};

export {
  groupIntroConfig,
  findGroupIntroConfig,
};
