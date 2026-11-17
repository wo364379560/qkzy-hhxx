# TodayToTrackRun 开发计划表

> 工程：`yundongDemo/TodayToTrackRunSwift`  
> 视觉：米白底 + 珊瑚橙（不默认荧光绿深色）  
> 服务端策略：账号复用现有腾讯云对接服务端；跑步数据独立模块 `/api/v1/run`

---

## 总览

| 阶段 | 目标 | 状态 |
|------|------|------|
| **一期 · 本地可用** | 跑步轨迹、目标打卡、天气日程、个人中心 | ✅ 基本完成 |
| **二期 · 账号同步** | 登录注册、轨迹上传/拉取 | 🟡 登录/注册/短信/重置已接真实 HTTP；跑步同步仍待接 |
| **三期 · 数据与健康** | 统计图表、Apple Health、语音播报落地 | 🔲 未开始 |
| **四期 · 训练体系** | 5K/10K/半马课程、个性化计划 | 🔲 未开始 |

---

## 一期清单（本地）

| 模块 | 能力 | 状态 | 说明 |
|------|------|------|------|
| 运动 | 高德卫星图 + 黄线轨迹 | ✅ | `RunMapViewController` |
| 运动 | 开始/暂停/结束、室内模拟 | ✅ | |
| 运动 | 结束自动缩放轨迹 | ✅ | |
| 运动 | 每公里女声播报（girlmp3 / gril_*） | ✅ | `RunVoicePrompt`，对齐 YSRun |
| 运动 | 自动暂停 / 恢复（停住约 8 秒） | ✅ | `RunLocationTracker` |
| 存储 | GRDB 本地 `run_record` / `run_point` | ✅ | |
| 存储 | `server_id` / `synced_at` 预留 | ✅ | v2 migration |
| 存储 | 消耗 kcal / 估心率 | ✅ | v3 migration |
| 地理 | 逆地理写城市名 | ✅ | 高德 Search |
| 今日 | 问候、环形周进度、最近一次数据 | ✅ | |
| 今日 | 天气卡片（聚合 API） | ✅ | 自 chixin demo 移植 |
| 今日 | 今日训练日程预览 | ✅ | |
| 计划 | 自定义目标、待办打卡、本周打卡 | ✅ | |
| 计划 | 训练日程增删勾选 | ✅ | `TrainingScheduleStore` |
| 计划 | 5K / 10K / 半马本地课程包 | ✅ | `TrainingCourseCatalog` |
| 我的 | 成就徽章、累计数据、设置开关 | ✅ | 布局学参考稿，配色自有 |
| 详情 | 起终点、心率曲线、自动 fit | ✅ | |
| 跑后 | 完成小结页（进度/消耗） | ✅ | |
| 分享 | 系统分享（截图+文案） | ✅ | 无 ShareSDK |
| UI | 大圆角卡片、等宽数字、毛玻璃导航 | ✅ | 浅色主题 |

---

## 二期清单（需服务端）

| 模块 | 能力 | 状态 | 依赖 |
|------|------|------|------|
| 接口 | `RunBackendAPI` 契约 | ✅ 桩 | `Network/RunBackendAPI.swift` |
| 配置 | `RunAPIConfig.baseURL` | 🔲 | 填真实地址 |
| 账号 | 手机号登录 / 注册 | ✅ | 对照 Android LinLan：`/login` `/register` `/sms/send` `/password/reset` |
| 账号 | Sign in with Apple（可选） | 🔲 | 产品确认后再做 |
| 同步 | 上传未同步跑步记录 | 🔲 | `RunSyncService` 已可调 |
| 同步 | 多端拉取远程列表 | 🔲 | `fetchRemoteRuns` |
| 服务端 | `/api/v1/run` 模块 + 表 | 🔲 | 独立于 IM 表 |

---

## 三期清单（数据与健康）

| 模块 | 能力 | 状态 |
|------|------|------|
| 统计 | 月度跑量柱状图 | 🔲 |
| 统计 | 配速趋势折线图 | 🔲 |
| 统计 | 个人最佳（最远/最快/最长） | 🔲 |
| Health | Apple Health 读写开关 | 🔲 |
| 运动中 | 语音播报真正播报公里/配速 | ✅ 已用 girlmp3 落地 |
| 运动中 | 自动暂停真正生效 | ✅ 已落地 |

---

## 四期清单（训练产品化）

| 模块 | 能力 | 状态 |
|------|------|------|
| 计划 | 5K / 10K / 半马课程卡片 | ✅ 本地课程包已落地 |
| 计划 | 按周生成今日训练任务 | ✅ 报名即写入日程 |
| 内容 | 饮食建议个性化（按偏好） | 🔲 目前静态文案 |

---

## 明确不做 / 暂缓

| 项 | 原因 |
|----|------|
| ShareSDK / 友盟 | 体积大、过时，用系统分享替代 |
| AMapNavi | 跑步 App 不需要导航 |
| 荧光绿深色默认主题 | 可后期加主题切换；默认保持米白+橙 |
| 独立新服务端账号体系 | 优先复用现有服务端登录 |

---

## 建议下一步（按优先级）

1. **你提供**现有服务端可访问的 `baseURL`（或确认在哪个仓库加 `/api/v1/run`）
2. 接二期：登录页 + 上传本地未同步记录
3. 再开「统计」页（柱状图 / 配速趋势）
4. 最后 HealthKit + 语音播报落地

---

## 关键文件索引

| 路径 | 作用 |
|------|------|
| `Main/TodayHomeViewController.swift` | 今日仪表盘 |
| `Main/PlanHomeViewController.swift` | 目标 + 日程 |
| `Main/MeHomeViewController.swift` | 个人中心 |
| `Run/RunMapViewController.swift` | 跑步中 |
| `Network/RunBackendAPI.swift` | 二期接口契约 |
| `Network/JuheWeatherService.swift` | 天气（自 chixin） |
| `Storage/TrainingScheduleStore.swift` | 训练日程 |
| `Storage/AppDatabase.swift` | GRDB schema |
