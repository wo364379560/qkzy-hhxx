# Release 打包

项目 `app/build.gradle.kts` 已配置 release 签名，执行 `:app:assembleRelease` 即可生成已签名 APK。

输出：`app/build/outputs/apk/release/app-release.apk`。

Release APK 仅包含 ARM 32 位（`armeabi-v7a`）和 ARM 64 位（`arm64-v8a`）原生库。Android Studio 手动选择 release 构建时也使用此配置；debug 构建保留原有架构支持。

运动功能使用 Android 系统 GPS 和本地轨迹绘制，不再依赖高德或 Google 地图，无需地图 Key。保留运动里程、时长、配速、暂停/继续、后台记录和本机历史轨迹查看，不显示道路底图、卫星图或城市名称。已有 GPS 历史记录格式保持兼容。

2026-09-22 GPS 轨迹版验证：debug/release 构建成功，25 项 JVM 测试通过（含 4 项轨迹投影测试）。release APK 为 51,533,553 字节（51.53 MB），签名与既有版本一致；包内仅保留 ARM 双架构，已检查无高德原生库、SDK 类或地图资源。本轮未进行户外真实跑步和锁屏连续定位实测。

## Android Studio 手动签名

在 Generate Signed App Bundle or APK 中选择 APK，签名信息填写：

| 字段 | 值 |
| --- | --- |
| Key store path | 本项目的 `app/balabala.jks` |
| Key store password | `balabala` |
| Key alias | `balabala.jks`（必须包含 `.jks`） |
| Key password | `balabala` |
| Build variant | `release` |

如果提示 `Cannot find key balabala`，关闭错误提示，把 Key alias 从 `balabala` 改为 `balabala.jks`，再点击 Next。也可以用 Key alias 右侧的选择按钮选择已有条目。

此错误是别名不匹配，不需要重新生成或替换密钥库。2026-09-22 已验证密钥库包含 `balabala.jks` 私钥条目，release APK 构建及签名校验通过。

既有签名证书 SHA-256：`AD:6D:B0:F8:5F:7F:96:B4:35:E9:48:32:CF:7B:B4:26:51:45:07:92:F4:F6:B1:0F:A5:44:8D:E8:A0:6A:F3:06`。
