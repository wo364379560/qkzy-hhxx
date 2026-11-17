interface ILinkItem {
  label: string;
  url: string;
}

interface IStepItem {
  label: string;
  url: string;
}

interface IAdvCardItem {
  label: string;
  subLabel: string;
  btnText: string;
  url: string;
}

interface IApkQRCodeItem {
  label: string;
  url: string;
}

const Link: {
  adv: ILinkItem;
  demo: ILinkItem;
  im: ILinkItem;
  privacy: ILinkItem;
  agreement: ILinkItem;
  statement: ILinkItem;
  personalInfo: ILinkItem;
  thirdPartyInfo: ILinkItem;
  minorTerms: ILinkItem;
  product: ILinkItem;
  customMessage: ILinkItem;
  contact: ILinkItem;
  intl: ILinkItem;
  stepList: IStepItem[];
  advList: IAdvCardItem[];
  apkQRCodeList: IApkQRCodeItem[];
} = {
  adv: {
    label: '首购低至1折, 复购7.5折起! 立即选购',
    url: 'https://cloud.tencent.com/act/pro/imnew?from=16262',
  },
  demo: {
    label: '体验更多Demo',
    url: 'https://cloud.tencent.com/document/product/269/36852',
  },
  im: {
    label: '访问官网',
    url: 'https://cloud.tencent.com/product/im',
  },
  privacy: {
    label: '隐私条例',
    url: 'https://ys.ruizhihexx.xyz/privacy_protocol.html',
  },
  agreement: {
    label: '用户协议',
    url: 'https://ys.ruizhihexx.xyz/user_agreement.html',
  },
  statement: {
    label: '免责声明',
    url: 'https://ys.ruizhihexx.xyz/statement_protocol.html',
  },
  personalInfo: {
    label: '个人信息收集清单',
    url: 'https://ys.ruizhihexx.xyz/personal_info_protocol.html',
  },
  thirdPartyInfo: {
    label: '第三方信息共享清单',
    url: 'https://ys.ruizhihexx.xyz/third_info_share_list.html',
  },
  minorTerms: {
    label: '未成年人使用条款',
    url: 'https://ys.ruizhihexx.xyz/weichengnian.html',
  },
  product: {
    label: '产品文档',
    url: 'https://cloud.tencent.com/document/product/269/1499#.E7.BE.A4.E7.BB.84.E5.8A.9F.E8.83.BD',
  },
  customMessage: {
    label: '自定义消息',
    url: 'https://web.sdk.qcloud.com/im/doc/zh-cn/SDK.html#createCustomMessage',
  },
  contact: {
    label: '联系我们',
    url: 'https://cloud.tencent.com/document/product/269/59590',
  },
  intl: {
    label: '国际站',
    url: 'https://web.sdk.qcloud.com/im/demo/en/index.html#/home',
  },
  stepList: [],
  advList: [],
  apkQRCodeList: [],
};

const qrList = [
  {
    icon: 'https://web.sdk.qcloud.com/im/assets/images/react_native.svg',
    name: 'React Native',
    link: 'https://web.sdk.qcloud.com/component/TUIKit/chat_react-native_demo_apk.png',
    detail: '扫描二维码下载',
  },
  {
    icon: 'https://web.sdk.qcloud.com/im/assets/images/uniapp.svg',
    name: 'Uniapp',
    link: 'https://qcloudimg.tencent-cloud.cn/raw/c1fed062d91cd95fdfb57059edcd5890.png',
    detail: '扫描二维码下载',
  },
  {
    icon: 'https://web.sdk.qcloud.com/im/assets/images/Android.svg',
    name: 'Android',
    link: 'https://web.sdk.qcloud.com/im/assets/images/android.png',
    detail: '扫描二维码下载',
  },
  {
    icon: 'https://web.sdk.qcloud.com/im/assets/images/iOS.svg',
    name: 'iOS',
    link: 'https://web.sdk.qcloud.com/im/assets/images/ios.png',
    detail: '扫描二维码下载',
  },
  {
    icon: 'https://web.sdk.qcloud.com/im/assets/images/Miniprogram.svg',
    name: '小程序',
    link: 'https://web.sdk.qcloud.com/im/assets/images/mini.png',
    detail: '微信扫码进入',
  },
];

const mobileList = [
  {
    type: 'android',
    link: 'https://web.sdk.qcloud.com/im/assets/images/android-mobile.svg',
    url: 'https://comm.qq.com/im_demo_download/index.html#/mobile-demo',
  },
  {
    type: 'iphone',
    link: 'https://web.sdk.qcloud.com/im/assets/images/iphone-mobile.svg',
    url: 'https://apps.apple.com/cn/app/%E8%85%BE%E8%AE%AF%E4%BA%91im/id1112479040',
  },
  {
    type: 'miniprogram',
    link: 'https://web.sdk.qcloud.com/im/assets/images/mini-mobile.svg',
  },
];

export { Link, qrList, mobileList };
