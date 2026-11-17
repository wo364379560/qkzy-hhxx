# 归属隔离：用户搜索限制 + 好友申请拦截

**日期**：2026-03-17
**状态**：待实现

---

## 背景

当前 Flutter App 通过腾讯云 IM SDK 的 `fetchUserInfo(userID)` 直接查询任意用户，
导致不同归属（`owner_id`）的用户可以互相搜索、添加好友、加入群组，违反业务隔离要求。

**目标**：同一 `owner_id` 下的用户才能互相搜索、添加好友、加群。跨归属一律拒绝。

---

## 归属模型回顾

```
admin（owner_id = admin.id）
  └── assistant（owner_id = admin.id）
        └── user（owner_id = admin.id）
```

- 用户注册时通过邀请码确定归属，`owner_id` 指向对应管理员的 `admin_users.id`
- 同一管理员下的所有用户（含其助理创建的用户）共享同一 `owner_id`
- 跨 `owner_id` 的用户视为不同"圈子"，不允许任何社交关系

---

## 方案 C：双层防护

### 第一层：后端搜索接口（主动防御）

替换 Flutter 端直接调用腾讯 SDK 的搜索，改为调用自有后端接口，后端在 DB 层过滤归属。

**接口**：`GET /api/v1/chat/users/search`

| 参数 | 类型 | 说明 |
|------|------|------|
| `q` | string | 搜索关键词，匹配 `nickname` 或 `im_user_id`，最少 1 字符 |

**鉴权**：`AuthChat` 中间件（Bearer JWT），从 `im_user_id` 反查调用者的 `owner_id`。

**响应**：

```json
{
  "code": 0,
  "data": {
    "users": [
      {
        "im_user_id": "u_abc123",
        "nickname": "张三",
        "avatar_url": "https://..."
      }
    ]
  }
}
```

**逻辑**：

```
1. 解析 JWT → im_user_id
2. DB 查调用者 → owner_id
3. SELECT im_user_id, nickname, avatar_url
   FROM users
   WHERE owner_id = ? AND status = 'active'
     AND (nickname LIKE ? OR im_user_id LIKE ?)
     AND im_user_id != ?   -- 排除自己
   LIMIT 20
4. 返回结果
```

**错误情况**：
- `q` 为空 → 400 InvalidParams
- 调用者不存在 → 401
- DB 错误 → 500

---

### 第二层：Webhook 兜底（被动防御）

利用腾讯云 IM「添加好友之前回调」拦截跨归属好友申请，即使客户端绕过搜索限制也无法建立好友关系。

**腾讯云配置**：
- 回调类型：`Sns.CallbackPrevFriendAdd`
- 回调 URL：`https://<domain>/api/v1/webhook/tencent`
- 在腾讯云 IM 控制台「回调配置」中启用

**接口**：`POST /api/v1/webhook/tencent`

无需 JWT 鉴权，使用腾讯云签名验证（`CallbackCommand` + `ClientIP` + `OptPlatform`）。

**请求体**（腾讯云发来）：

```json
{
  "CallbackCommand": "Sns.CallbackPrevFriendAdd",
  "From_Account": "u_abc123",
  "FriendItem": [
    { "To_Account": "u_xyz789", "Remark": "", "GroupName": "", "AddSource": "AddSource_Type_Search" }
  ]
}
```

**逻辑**：

```
1. 验证 CallbackCommand == "Sns.CallbackPrevFriendAdd"
2. 批量查 From_Account + 所有 To_Account 的 owner_id
3. 对每个 To_Account：
   - owner_id 相同 → ResultCode: 0（放行）
   - owner_id 不同 → ResultCode: 38000（拒绝，自定义错误码）
4. 返回逐条结果
```

**响应**（返回给腾讯云）：

```json
{
  "ActionStatus": "OK",
  "ErrorCode": 0,
  "ErrorInfo": "",
  "FriendItem": [
    { "To_Account": "u_xyz789", "ResultCode": 0, "ResultInfo": "" }
  ]
}
```

跨归属时：
```json
{ "To_Account": "u_xyz789", "ResultCode": 38000, "ResultInfo": "cross-owner not allowed" }
```

---

## Flutter 端改动

### 替换搜索调用

三处使用 `fetchUserInfo(userID)` 的地方改为调用后端 `/chat/users/search?q=<keyword>`：

| 文件 | 当前行为 | 改后行为 |
|------|----------|----------|
| `search/search_page.dart` | 本地无结果时调 `fetchUserInfo` | 改调后端搜索接口 |
| `contact_list/pages/start_c2c_chat.dart` | 本地无结果时调 `fetchUserInfo` | 改调后端搜索接口 |
| `contact_list/pages/start_group_chat.dart` | 本地无结果时调 `fetchUserInfo` | 改调后端搜索接口 |

### 新增 API 方法

在 Flutter 的 API 层新增：

```dart
// GET /api/v1/chat/users/search?q=<keyword>
Future<List<SearchUserResult>> searchUsers(String keyword);
```

返回结构：
```dart
class SearchUserResult {
  final String imUserID;
  final String nickname;
  final String? avatarURL;
}
```

---

## 文件变更清单

### 后端（Go）

| 文件 | 操作 |
|------|------|
| `repository/user.go` | 新增 `SearchByOwner(ownerID uint, keyword, selfIMUserID string) ([]User, error)` |
| `controller/chat/user.go` | 新建，`SearchUsers(ctx, imUserID, keyword)` |
| `handler/chat/user.go` | 新建，`SearchUsers` handler |
| `router/chat.go` | 注册 `GET /users/search` |
| `handler/webhook/tencent.go` | 新建，Webhook 处理逻辑 |
| `router/router.go` | 注册 `POST /api/v1/webhook/tencent` |

### Flutter

| 文件 | 操作 |
|------|------|
| `search/search_page.dart` | 替换 `fetchUserInfo` → 后端接口 |
| `contact_list/pages/start_c2c_chat.dart` | 同上 |
| `contact_list/pages/start_group_chat.dart` | 同上 |
| `api/` 或对应网络层 | 新增 `searchUsers` 方法 |

---

## 安全边界说明

| 场景 | 第一层（搜索） | 第二层（Webhook） |
|------|--------------|-----------------|
| 正常搜索跨归属用户 | 搜不到，返回空 | — |
| 绕过搜索直接发好友申请 | 无法阻止 | Webhook 拒绝 |
| 群邀请跨归属用户 | 搜不到 | 可扩展 `Sns.CallbackPrevGroupInvite` |
| 直接输入 im_user_id 发申请 | 无法阻止 | Webhook 拒绝 |

> 群邀请拦截（`Sns.CallbackPrevGroupInvite`）可作为后续扩展，当前版本暂不实现。

---

## 实现顺序

1. 后端：`repository/user.go` 新增 `SearchByOwner`
2. 后端：`controller` + `handler` + `router` 注册搜索接口
3. 后端：Webhook handler + 路由
4. 腾讯云控制台配置回调 URL
5. Flutter：新增 API 方法，替换三处 `fetchUserInfo`
