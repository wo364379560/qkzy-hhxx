# 聊天礼物资源（gift）

把礼物图片放这里即可自动进面板（Vite `import.meta.glob`）：

- 支持：`.png` `.webp` `.gif` `.svg` `.json`
- 文件名即 `giftId`，例如 `rose.png` → giftId=`rose`
- 也兼容目录名 `src/assets/chatgit/`（拼写兼容）

发送协议：`businessID = gift_v1`（见 `src/im/stickerCatalog.ts`）

气泡只展示礼物图，**不显示「发消息」文字**。
