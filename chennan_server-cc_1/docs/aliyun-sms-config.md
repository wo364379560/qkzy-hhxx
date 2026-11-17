# 阿里云短信配置说明

## 环境变量配置

在 `.env` 文件中配置以下参数：

```bash
# 短信服务商：dev=开发模式（仅打印日志），aliyun=阿里云短信
SMS_PROVIDER=aliyun

# 阿里云 AccessKey ID（RAM 用户或主账号）
SMS_ACCESS_KEY_ID=LTAI5tXXXXXXXXXXXXXX

# 阿里云 AccessKey Secret
SMS_ACCESS_KEY_SECRET=xxxxxxxxxxxxxxxxxxxxxxxxxx

# 短信签名名称（需在阿里云短信控制台申请）
SMS_SIGN_NAME=万安IM

# 短信模板 CODE（需在阿里云短信控制台申请）
SMS_TEMPLATE_CODE=SMS_123456789
```

## 获取配置参数步骤

### 1. 获取 AccessKey

1. 登录 [阿里云控制台](https://ram.console.aliyun.com/manage/ak)
2. 进入 **访问控制 RAM** > **用户** > **创建用户**（或使用现有用户）
3. 勾选 **OpenAPI 调用访问**
4. 创建后会显示 **AccessKey ID** 和 **AccessKey Secret**（仅显示一次，请妥善保存）

**权限配置：**
- 为该 RAM 用户添加权限：`AliyunDysmsFullAccess`（短信服务完全访问权限）

### 2. 申请短信签名

1. 进入 [短信服务控制台](https://dysms.console.aliyun.com/overview)
2. 点击 **国内消息** > **签名管理** > **添加签名**
3. 填写签名信息：
   - **签名名称**：万安IM（或您的应用名称）
   - **签名来源**：选择适合的类型（如：App应用）
   - **签名用途**：验证码
   - 上传相关证明材料
4. 提交审核，通过后获得签名名称（如：`万安IM`）

### 3. 申请短信模板

1. 在短信服务控制台，点击 **模板管理** > **添加模板**
2. 填写模板信息：
   - **模板类型**：验证码
   - **模板名称**：验证码通知
   - **模板内容**：`您的验证码为：${code}，该验证码5分钟内有效，请勿泄漏于他人。`
   - **申请说明**：用于用户注册/登录验证
3. 提交审核，通过后获得模板 CODE（如：`SMS_123456789`）

**注意：**
- 模板中的变量必须使用 `${code}` 格式
- 代码中会自动传入 `{"code":"123456"}` 格式的参数

## 开发模式

开发/测试环境可以使用 `dev` 模式，验证码会直接打印到日志中：

```bash
SMS_PROVIDER=dev
SMS_ACCESS_KEY_ID=
SMS_ACCESS_KEY_SECRET=
SMS_SIGN_NAME=
SMS_TEMPLATE_CODE=
```

启动服务后，发送验证码时会在日志中看到：

```
[INFO] [DEV SMS] phone=13012345678 code=123456
```

## 验证码规则

- **长度**：6 位数字
- **有效期**：5 分钟
- **存储**：MySQL 数据库（`sms_codes` 表）
- **一次性**：验证成功后自动删除

## 费用说明

阿里云短信按量计费：
- 验证码短信：约 0.045 元/条
- 需要在阿里云账户中充值
- 建议设置费用预警

## 常见问题

### 1. 签名或模板审核不通过？
- 确保提供的证明材料真实有效
- 签名需与应用名称一致
- 模板内容需符合规范，不能包含营销内容

### 2. 发送失败？
- 检查 AccessKey 权限是否正确
- 确认签名和模板已审核通过
- 查看日志中的错误信息

### 3. 如何测试？
- 使用 `SMS_PROVIDER=dev` 模式进行本地测试
- 正式环境切换为 `SMS_PROVIDER=aliyun`

## 安全建议

1. **不要将 AccessKey 提交到代码仓库**
2. 使用 RAM 子账号，遵循最小权限原则
3. 定期轮换 AccessKey
4. 生产环境使用环境变量注入，不使用 .env 文件
5. 启用 API 调用监控和告警
