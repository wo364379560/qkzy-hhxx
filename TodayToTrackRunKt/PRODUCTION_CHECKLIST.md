# 睿之禾 Android IM 交付与验收清单

## 已接入

1. 腾讯云 IM SDK 初始化、服务端动态 `sdk_app_id`/UserSig、登录、UserSig 过期刷新。
2. 真实会话、好友申请角标、好友/黑名单、群组及分页成员同步。
3. 文本、真实表情、图片、视频、文件、语音收发；个人与群聊共用同一消息链路。
4. 媒体上传/下载进度、失败重试、缓存、预览、播放器、保存系统相册；缓存文件安全命名。
5. 30 条游标历史分页、下拉加载、云端聊天搜索与本地数据库回退、断线恢复同步。
6. 回复、自己 120 秒撤回、管理员服务端撤回、最多 50 条逐条/合并多目标转发、群置顶消息。
7. 群公告 5000 UTF-8 字节服务端接口、群资料、全体/个人禁言、管理员、踢人、禁止查看成员资料及可读群提示。
8. V2TIM Signaling + LiteAVSDK TRTC 直接实现音视频通话，未使用 TUICallKit。
9. 本地通知、前后台提示音、未读角标、TIMPush 注册和通知点击路由。
10. 手机验证码/密码登录注册、改密、拉黑、完整举报页面、注销账号接口。
11. 弱网状态、UserSig 自动恢复、幂等 GET 重试、TIM 本地数据库、群成员分页及 10 万成员纯数据性能测试。

应用名称、Bundle/Application ID、SDKAppID、API 域名和 1024 图标均与 iOS 对齐；Android 主界面只有“消息 / 通讯录 / 我的”，无运动模块。

## 发布前仍需外部材料和真机验收

- 将腾讯云控制台为 `com.balabalaim.gcloud.tim.push.appdev` 生成的 `timpush-configs.json` 放入应用模块，并补齐华为、荣耀、FCM 等实际使用厂商的配置文件和控制台证书。仓库中目前没有这些密钥文件。
- 配置正式 release keystore；当前交付 APK 使用 Android Debug 签名。
- 提供两个可登录的测试手机号/验证码，在至少两台真机上验收好友、群聊、媒体、离线推送和 TRTC 音视频。
- 本机已有 AVD 的 Android 36 system image 不完整，无法启动模拟器；当前已完成编译、单元测试和 Lint，尚未完成设备端自动冒烟测试。

## 已执行的自动验证

- `./gradlew :app:testDebugUnitTest :app:assembleDebug`
- `./gradlew :app:lintDebug`
- 6 个单元测试全部通过，Lint 无 error，Debug APK 构建成功。
