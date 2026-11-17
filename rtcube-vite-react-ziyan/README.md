# 诚楠 IM · Electron 自研壳（rtcube-vite-react-ziyan）

> 规格：[`../docs/IM_SPEC.md`](../docs/IM_SPEC.md) · [`../docs/IM_PRODUCT.md`](../docs/IM_PRODUCT.md)

- **Vite + React + TypeScript + Electron**
- **官方 SDK**：`@tencentcloud/chat`（**不接 TUIKit / TUICallKit**）
- **音视频**：自研信令 `rtc_call_v1` + `trtc-sdk-v5`（对齐 iOS `TXLiteAVSDK_TRTC` / `TXIMSDK_Plus`）
- **三栏**：左会话 / 中聊天 / 右公告+成员
- **大群防崩**：成员 `getGroupMemberList` 分页（50/页）、`@` 搜索分页、消息窗口上限 400、虚拟列表

## 与旧工程关系

| 工程 | 说明 |
|------|------|
| `rtcube-vite-react` | 仍含 TUIKit，仅作交互参考 |
| `rtcube-vite-react-ziyan`（本目录） | **自研主线**，剥掉 TUIKit |
| `web-h5` | Vue H5 自研壳，同一份 SPEC |

## 域名

| 用途 | 主 | 备 |
|------|----|----|
| API | `https://api.prfgsu.com` | `https://api.uktjwg.cn`（另有 `https://api.ruizhihe.com`） |
| 网页访问入口 | 尚未确认部署域名；开发地址 `http://localhost:5175` | — |
| 域名下发 ① | `https://api.prfgsu.com/api/v1/public/bootstrap/domain-pool` | — |
| 域名下发 ② | `https://api.uktjwg.cn/api/v1/public/bootstrap/domain-pool` | — |

开发代理默认使用 `https://api.prfgsu.com`（`vite.config.ts`）。生产启动时依次请求两个下发接口的 `data.api_origins`，超时（3 秒）或配置无效时使用预置双 API；腾讯 IM SDKAppID 默认 `1600163137`。`VITE_CHAT_API_BASE_URL` / `VITE_API_BASE_URL` 显式配置仍优先。

全体禁言通过腾讯 IM `updateGroupProfile({ muteAllMembers })` 同步，App 需监听全体禁言及群资料变更事件，并更新输入框与群提示；这里不额外发送普通群消息。

## 启动

```bash
cd rtcube-vite-react-ziyan
cp .env.example .env.local   # 按需改 SDKAppID
npm install
npm run dev                  # 浏览器 http://localhost:5175
# 或
npm run electron:dev
```

## 大群说明（腾讯侧）

群成员**不是**一次拉全员：

```ts
chat.getGroupMemberList({ groupID, offset, count }) // count ≤ 100
// 回包 data.offset = 下一页起点；本工程默认 count=50
```

右栏成员、`@` 选人均走分页；有 `searchGroupMembers` 时优先云搜，否则只在当前页过滤并继续分页，**禁止扫全员**。

## 通话

- 自定义消息 / 信令 `businessID = rtc_call_v1`
- timeout 45s，与 iOS `ShangLan_RTCCallMessageParser` 对齐
- UI：`CallOverlay`；TRTC 进房后续接满 `trtc-sdk-v5`

## 2026-09-25 网页消息互通修复

- 单聊已读：合并 SDK 缓存和已读接口返回的消息状态；区分基础已读事件与高级回执中的 `isPeerRead`，不把未读回执误判为已读。回执按会话分组，每批最多 30 条；气泡明确显示已读/未读文字。
- 表情：网页输入仍支持旧 `[TUIEmoji_*]` 标记，发送前转为 Unicode，原生端无需 TUIKit 表情解析器即可显示；面板图片按 Android 同款资源的零起始下标映射。已经发送的历史消息不被改写。
- 引用：发送同时带 `messageReply` 和 `atomicx_quote_v1`，读取兼容两者，并携带原消息时间、群消息序号和原生消息类型。
- 定位：点击网页引用，先定位当前消息；窗口外的消息通过 SDK 时间/序号跳拉，缺少定位字段的旧引用逐页查找，内存只保留少量上下文。切换会话或返回最新消息会取消旧定位结果；原消息不可用时明确提示。查看旧消息时可点击“返回最新消息”。
- 域名检查：`src/config/domains.ts` 的 `web` 字段虽与 API 域名相同，但 2026-09-25 检查两个 API 根路径均返回 HTTP 404，不能作为已部署聊天网页的证明。未修改线上域名或发布站点。

验证：`npm run test:interop`（10 项自动回归）、`npm run build` 通过。SDK 返回值、回执分批、引用协议、超过 400 条消息的定位、取消定位及无效游标均有回归用例；构建仍提示现有大体积 JS 分包警告。

待实机验收：两个真实账号双向发送文字/表情/引用；手机读网页消息后观察网页状态；引用超过当前消息窗口的原消息，检查滚动高亮及返回最新；查找中切换会话，确认无串聊。手机端点击引用与旧消息表情展示需要核对实际配套 App 工程；本次只修改本目录，尚未部署。

SDK 协议依据：[已读回执](https://cloud.tencent.com/document/product/269/75344)、[历史消息定位](https://cloud.tencent.com/document/product/269/75322)。高级回执需对应 SDK 服务能力；社群不支持群已读回执，本次展示修复适用于单聊。
