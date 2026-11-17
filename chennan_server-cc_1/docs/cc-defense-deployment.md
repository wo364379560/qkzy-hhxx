# 城南 IM 登录/注册接口 CC 防护部署说明

## 1. 审计结论

这次故障与 HTTPS 是否能抓包、是否使用 Cronet/BoringSSL、以及腾讯云 IM SDK 无直接关系。攻击者不需要真实手机，也不需要解密 IM SDK 流量：只要知道公开 API 地址，就可以用脚本、代理池、肉鸡或云主机直接请求登录、注册、短信和刷新接口。

原始代码的主要放大链路是：

1. 公开登录、注册、短信接口没有在数据库之前执行全局和 IP 限流。
2. `IPAccessBlock` 每个高危请求先查 MySQL。
3. 登录继续查用户、查冷却状态并执行 bcrypt；开启“IM 全局通用密码”时，错误密码最多执行第二次 bcrypt。
4. 短信接口每次请求执行 `AutoMigrate`，之后再执行多次 MySQL 统计查询和短信外呼。
5. Redis 默认关闭；即使开启，原限流器在 Redis 故障时也直接放行。
6. Go HTTP Server 没有读写超时，请求体也没有上限；MySQL 连接池默认无限。

这意味着 ESA 即使拦掉大部分流量，漏过来的分布式 CC 仍然能先消耗 MySQL、CPU 和短信费用。

## 2. 本补丁做了什么

### 2.1 请求进入业务前的五道闸门

`PublicProtection` 已放在 `/api/v1` 的纯内存迁移开关之后、所有访问 MySQL 的中间件之前，并且位于 `IPAccessBlock` 的查询之前：

1. 所有未登录高危接口共享一个 40 并发的总保险丝，低于默认 MySQL 50 连接上限；分散攻击多个路径也不能把各路径并发额度相加后压垮连接池。
2. 每类公开接口还有独立的非阻塞并发保险丝，满载立即返回 `503 + Retry-After`。
3. 原生/Web 登录、Refresh/Logout、后台登录/TOTP 分别使用独立的路径额度，并额外受一条较宽的共享容量上限约束，避免兄弟入口互相耗尽专属额度。
4. 每条路径按可信真实 IP 限速，用于限制单一来源。
5. bcrypt 总容量按 `GOMAXPROCS` 计算并限制在 2～8，再划分为“公开认证”和“已登录密码管理”两个互不抢占的配额（每类至少 1，后台类最多 2）；许可只包围真正的 bcrypt 计算，不在 MySQL、Turnstile、腾讯 IM 外呼期间占位。一个有效用户/管理员 Token 不能再饿死公开登录。

高危接口请求体会在进入 Handler 前完整限制为 8～32 KiB；腾讯 IM 回调限制为 256 KiB。超出上限直接返回 413。

公开附件下载入口也增加了全局、IP、文件 ID 三层限流，并在查 MySQL 前限制 `file_id`/`token` 长度。

### 2.2 多维业务限流

JSON 解析及长度校验后、进入 Controller/Turnstile/MySQL/bcrypt 前，继续按以下不可逆哈希 subject 限流：

| 接口 | 维度 |
| --- | --- |
| 登录/Web 登录 | 账号、设备 |
| 注册 | 手机号、设备 |
| 短信 | 手机号、设备 |
| 重置密码 | 手机号、设备 |
| Refresh | Token、设备 |
| 后台登录 | 管理员账号 |

设备 ID 仅作为辅助维度；攻击者能伪造或轮换设备 ID，因此防线仍以“端点全局 + 真实 IP + 账号/手机号”为主。

### 2.3 Redis 和本机兜底

- 每个规则始终先执行有界的进程内固定窗口限流。
- 公开入口按“真实 IP → 单接口全局 → 兄弟接口共享全局”的顺序扣额度；已被单 IP 规则拒绝的持续攻击不会继续污染全局预算，避免一个来源把所有正常用户拖入 429。
- 进程内计数按 scope 隔离，每个 scope 最多保留 2,048 个活跃 subject、最多 128 个编译期 scope；桶满时先回收过期项，绝不淘汰活跃对象。若仍满，新的高基数 subject 不再写入本机表，而是继续交给 Redis、端点全局和 IP 保险丝，避免攻击者用 2,048 个随机手机号把整条路由投毒成 429；已有对象的本机预算仍保留。容量事件会采样告警。
- Redis 可用时再执行分布式计数。
- Redis 的 `INCRBY + PEXPIRE` 改成单个 Lua 脚本，避免计数存在但没有 TTL。
- 限流 Redis 操作总超时为 100ms；Redis 故障时本机限流继续生效。
- 限流和 Redis 故障日志做采样，避免 CC 顺便打满磁盘。
- 限流/过载日志按有界的 `action + scope/reason` 独立采样，一个嘈杂入口不会吞掉后台登录等其它安全事件。Go 访问日志只在拒绝路径显式标记“已采样”时跳过，不再用 `Retry-After` 猜测；Nginx CC 日志跳过 429、带 `Retry-After` 的保护性 503 和腾讯回调认证 401。普通 503 仍记录。Nginx CC 日志移除可被攻击者放大的 URI/User-Agent，Go 的 URI/User-Agent 均限制记录长度。

生产环境仍必须开启 Redis；本机兜底只能保护单实例，不能在多台 Go 实例之间共享总额度。

### 2.4 其他资源保护

- 短信 `sms_codes` 表迁移改到应用启动阶段，删除请求路径里的 `AutoMigrate`。
- 同手机号发送增加 1 分钟前置额度，验证码写入改成单条原子 upsert，避免并发 DELETE/CREATE 导致短信已发出但验证码未落库。
- `sms_codes` 启动迁移会创建并校验为 InnoDB；短信验证码使用事务行锁并检查删除行数，保证并发下只消费一次，连续错误 5 次后验证码失效。
- 注册/重置密码使用“两阶段验证码”：先校验但不消费，所有便宜校验通过后才做 bcrypt，bcrypt 成功后再原子消费。假验证码不能烧 bcrypt，CPU 满载返回 503 也不会吃掉合法验证码。
- Go HTTP Server 增加 ReadHeader/Read/Write/Idle 超时和 16KiB Header 上限。
- 默认只监听 `127.0.0.1:8080`，避免绕过 ESA/Nginx 直连 Go。
- 默认只信任本机 Nginx，并且只读取 `X-Real-IP`。
- MySQL 默认连接池改为 MaxOpen=50、MaxIdle=20，避免 CC 无限创建连接。
- `IPAccessBlock` 查询增加有界 15 秒进程缓存和请求 Context，重复攻击不再每次查询 MySQL。
- Refresh Token 限制为 256 字节；认证输入字段增加长度上限。
- 登录内部异常不再把 `err.Error()` 原样返回给客户端。
- 后台登录读取 2FA 配置失败时直接失败，不再因数据库异常绕过 TOTP；后台 TOTP 备份码的 bcrypt 也进入同一个 CPU 保险丝，8 个备份码 hash 在数据库事务开始前完成。
- 已登录用户的强制改密/主动改密、后台改密/重置备份码增加用户或管理员维度的 3 次/分钟、10 次/小时限制；其它后台 bcrypt 工作统一进入小容量私有配额。
- 2026-09-15：此版本按部署要求移除了腾讯 IM 回调入口的 Token、签名、时间戳及 SDKAppID 校验；业务规则仍保留。回调不再验证请求来源。
- 用户密码失败仍记录风险，但不再仅凭错误密码硬锁账号，避免攻击者用已知手机号批量制造拒绝服务；真正的阻断由 IP/账号/全局限流完成。

## 3. ESA 标准版的三条频次规则

标准版只有三条频次规则时，建议先按下表合并。先使用“观察”运行 30～60 分钟确认正常峰值，再改为拦截。原生 App API 不要使用依赖浏览器执行 JavaScript 的挑战；Web/Admin 可使用挑战或 Turnstile。

| 规则 | URI 条件 | 统计维度 | 建议初值 | 动作 |
| --- | --- | --- | --- | --- |
| 登录注册保护 | `/api/v1/chat/login`、`/api/v1/chat/web-login`、`/api/v1/chat/register`、`/api/v1/chat/password/reset`、`/api/v1/admin/login` | 客户端 IP | 1 分钟 20 次 | 仅拦截超出部分，持续 5～10 分钟 |
| 短信费用保护 | `/api/v1/chat/sms/send` | 客户端 IP | 10 分钟 10 次 | 仅拦截超出部分，持续 30～60 分钟 |
| 验证/会话保护 | `/api/v1/chat/auth/refresh`、公开 TOTP、`/api/v1/chat/login-challenges/*/(verify-sms|exchange)`、后台 TOTP | 客户端 IP | 1 分钟 120 次 | 仅拦截超出部分，持续 5～10 分钟 |

ESA 官方示例也把登录注册作为核心 API，并给出“同 IP 1 分钟 20 次”的起点：

- https://help.aliyun.com/zh/edge-security-acceleration/esa/user-guide/protect-core-business-api
- https://help.aliyun.com/zh/edge-security-acceleration/esa/user-guide/frequency-control-rules

注意：仅按 IP 的三条规则挡不住大量不同 IP。`baota-http-cc.conf` 和 Go 中的 endpoint-global 上限就是用来兜住分布式 CC 的。
Nginx 还在 API `server{}` 级提供每 IP 60 次/秒、全站 300 次/秒的基线，避免攻击者改打 ping、随机 404 或未单列接口；高危 `location` 会改用更严格的专用额度。这两个基线也要按正常峰值调，不可照搬到高 QPS 大集群。
原生/Web 登录、Refresh/Logout、后台登录/TOTP 在 Nginx 也使用独立 zone，并叠加较宽的 shared zone；不要自行合并回单一严格 zone，否则一个兄弟入口仍能提前耗尽另一个入口的全部额度。
用户 TOTP、登录挑战验证、挑战状态查询也各自使用独立 zone；挑战状态正则显式排除已登录的 `/pending`。每个专用 `location` 都显式叠加 `chennan_api_total`，避免 Nginx 的限流继承规则让 300r/s 总入口保险丝失效。

## 4. 正确的真实 IP 链路

### 4.1 ESA

在“规则 -> 转换规则 -> 修改请求头 -> ESA 到源站”新增全站规则：

- 类型：动态
- 操作：添加（同名头会覆盖客户端伪造值）
- Header：`True-Client-IP-OUT`
- 值：`ip.src`

官方配置说明：

https://help.aliyun.com/zh/edge-security-acceleration/esa/user-guide/modify-outgoing-request-header

### 4.2 Nginx

Nginx 仅对 ESA 当前回源 CIDR 信任 `True-Client-IP-OUT`。Nginx 校验后生成自己的 `X-Real-IP`，并覆盖而不是追加 XFF：

```nginx
proxy_set_header X-Real-IP $remote_addr;
proxy_set_header X-Forwarded-For $remote_addr;
proxy_set_header X-Forwarded-Proto https;
proxy_set_header X-Forwarded-Host $host;
```

禁止使用：

```nginx
proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
```

否则攻击者可在客户端请求里预先塞入伪造 XFF，污染按 IP 的业务限流。
`X-Forwarded-Proto/Host` 也必须由 Nginx 覆盖；本配置按“公网 API 只提供 HTTPS”固定为 `https`，避免 ESA 使用 HTTP 回源时生成错误的下载链接。宝塔的 80 端口应只做 301 跳转到 HTTPS。

### 4.3 Go

同机 Nginx 的生产环境配置：

```env
APP_ADDR=127.0.0.1
APP_PORT=8080
APP_TRUSTED_PROXY_CIDRS=127.0.0.1/32,::1/128
APP_REAL_IP_HEADER_ORDER=X-Real-IP
```

Docker 部署不要盲目使用回环配置：Go 需监听容器网卡，但宿主机端口必须只发布到回环地址，例如 `127.0.0.1:8080:8080`；`APP_TRUSTED_PROXY_CIDRS` 只填写 Nginx 的准确容器 IP/CIDR，不要信任整个 `10/8` 或 `172.16/12`。

## 5. 锁死源站

这是必须项；不做就能绕过 ESA 的所有规则。

1. 在 ESA“安全防护 -> 源站防护”取得当前回源 CIDR。
2. 云安全组/防火墙的 80、443 只允许这些 CIDR。
3. 8080、3306、6379 不允许公网访问；22 只允许管理 IP。
4. 删除指向源 IP 的测试域名和备用 A 记录。
5. 源 IP 已经泄露时，完成白名单后最好更换源 IP。

ESA 官方要求把当前回源 IP 列表加入源站白名单，并在列表变化时更新：

https://help.aliyun.com/zh/edge-security-acceleration/esa/user-guide/origin-protection

该官方页面的套餐表明确列出 375 元/月的标准版支持“源站防护”，所以现有套餐无需为了这项能力另买产品。

不要在启用 Nginx RealIP 后简单写 `allow ESA_CIDR; deny all;`，因为 `$remote_addr` 已经变成客户端 IP，容易误封所有用户。源站白名单优先放在云安全组/系统防火墙层。

## 6. 宝塔部署步骤

项目提供三个配置文件：

1. `deploy/nginx/baota-http-cc.conf`：放进 Nginx `http{}`。
2. `deploy/nginx/baota-proxy-common.conf`：保存为 `/www/server/nginx/conf/chennan_proxy_common.conf`。
3. `deploy/nginx/baota-site-api.conf`：粘贴进宝塔该 API 域名现有 `server{}`，保留原证书配置。

必须先替换 `__REPLACE_WITH_ESA_CIDR_1__`，并添加 ESA 控制台展示的全部 IPv4/IPv6 CIDR。
同时用配置中的 `chennan_cc` 那一行替换该 API `server{}` 原有的 `access_log`，不要同时保留两条，否则旧日志仍会逐条记录攻击请求。
文件里的登录挑战和附件下载是正则 `location`，需放在宝塔原有的宽泛正则 `location` 之前。
若现有配置有 `location ^~ /api/`，它会跳过所有正则匹配；应移除这个宽泛的 `^~` 或把对应限流直接合入该前缀块。Go 层仍会兜底，但不能因此省略 Nginx 检查。

应用生产环境建议：

```env
GIN_MODE=release
APP_ADDR=127.0.0.1
APP_PORT=8080
APP_TRUSTED_PROXY_CIDRS=127.0.0.1/32,::1/128
APP_REAL_IP_HEADER_ORDER=X-Real-IP

REDIS_ENABLED=true
REDIS_ADDR=127.0.0.1:6379
REDIS_PASSWORD=请设置强密码

# 此版本已移除回调鉴权，此兼容配置项不参与回调处理
TENCENT_IM_CALLBACK_TOKEN=

MYSQL_MAX_OPEN_CONNS=50
MYSQL_MAX_IDLE_CONNS=20
MYSQL_OPTIONS=charset=utf8mb4&parseTime=true&loc=Local&timeout=2s&readTimeout=15s&writeTimeout=15s

TURNSTILE_ENABLED=true
WEB_TURNSTILE_ENABLED=true

# 生产环境绝不能开启
DANGEROUS_AUTH_BYPASS_ENABLED=false
```

上线命令顺序：

```bash
# 在有 Go 1.26 工具链的构建机执行
gofmt -w internal/app/server/handler/authlimit \
  internal/pkg/authwork \
  internal/app/server/controller/admin \
  internal/app/server/controller/chat/auth.go \
  internal/app/server/handler/admin/auth.go \
  internal/app/server/handler/admin/totp.go \
  internal/app/server/handler/chat/auth.go \
  internal/app/server/handler/chat/attachment_upload.go \
  internal/app/server/handler/chat/login_challenge.go \
  internal/app/server/handler/public/backend_domain_pool.go \
  internal/app/server/handler/webhook/tencent.go \
  internal/app/server/middleware/ip_access_block.go \
  internal/app/server/middleware/ip_access_block_test.go \
  internal/app/server/middleware/public_protection.go \
  internal/app/server/middleware/public_protection_test.go \
  internal/app/server/ratelimit \
  internal/app/server/repository/login_risk_control.go \
  internal/app/server/repository/totp_backup_code.go \
  internal/app/server/router/router.go \
  internal/app/server/entry.go \
  internal/pkg/app/setup/database.go \
  internal/pkg/config \
  internal/pkg/handlers/gin_handlers.go \
  internal/pkg/handlers/gin_handlers_test.go \
  internal/pkg/sms/sms.go \
  internal/pkg/totp/totp.go
go test ./...
go build ./cmd/server

# 服务器上先检查再平滑加载，不能直接 restart
/www/server/nginx/sbin/nginx -t
/www/server/nginx/sbin/nginx -s reload
```

如果 `nginx -t` 报 `limit_req_zone directive is not allowed here`，说明把 `baota-http-cc.conf` 错放进了 `server{}`；它只能出现在 `http{}`。

## 7. 上线验收

1. 正常手机请求后，`/www/wwwlogs/chennan_cc.log` 应显示：
   - `client=` 为手机公网 IP；
   - `edge=` 为 ESA 回源节点 IP；
   - Go 日志 IP 与 `client=` 一致。
2. 从外网直接访问源 IP 必须超时或被安全组拒绝；伪造 `True-Client-IP-OUT` 也不能成功。
3. 连续请求登录接口超过阈值应得到 429；CPU/数据库连接不应随攻击流量线性增长。
4. 请求体超过接口上限应在 Go 或 Nginx 返回 413。
5. Redis 停止后，单实例仍应被本机限流；日志每分钟最多出现一次 Redis 降级警告。
6. 在预发布环境做并发测试，观察 Go CPU、MySQL active connections、Redis latency、429/503 和正常登录成功率。不要直接在生产环境无上限压测。

## 8. 阈值如何调

本补丁中的值是 1～3 千用户、小型单实例的保守起点，不是所有机器的固定答案。

1. 先统计 7 天正常峰值，按 endpoint、IP、账号分别看每秒和每分钟分布。
2. IP 阈值通常设在正常 P99 的 2～3 倍，照顾公司/校园/NAT 共享出口。
3. endpoint-global 必须低于当前服务器能稳定承受的 QPS；它不能按攻击流量峰值设置。
4. bcrypt 保护以 CPU 和登录 P95 为准。429 很少但 503 多，说明并发容量/机器 CPU 不足；两者都很多，说明攻击仍在回源或阈值过低。
5. 短信总量应按可接受的费用上限反推，当前 Go 默认最多 30 次/分钟、600 次/小时；业务实际更小可继续降低。

## 9. 仍需继续处理的安全债务

本补丁先解决“CC 先打到昂贵资源”的问题，以下项目不要忽略：

1. `DANGEROUS_AUTH_BYPASS_ENABLED=false` 只从进程层禁用“IM 全局通用密码”。万能验证码由后台「账号设置」写入数据库后生效，不再被该环境变量拦截。
2. 本补丁已统一普通账号不存在、密码错误和账号冷却的登录提示，避免直接枚举手机号；仍需避免其它找回/注册接口泄漏过多账号状态。
3. 短信持久统计仍会执行多次 MySQL 查询；下一阶段应使用 Redis Lua 预占短信额度，发送失败再补偿。
4. `IPAccessBlock` 已有 15 秒本机缓存；多实例或要求后台封禁立即生效时，应升级为 Redis 缓存并发布失效通知。
5. 后台和 Web 开启 Turnstile；原生 App 使用 App Attest、Play Integrity 或厂商设备风控。不要把客户端自报 `device_id` 当作可信设备证明。

## 10. 关于是否再买腾讯云 WAF

已经有 ESA 标准版时，当前优先级不是叠加购买另一家 WAF，而是：配置 ESA 三条规则、启用源站防护、锁死源 IP、上线本补丁和 Nginx 全局限流。

截至 2026-08-25 查阅的腾讯云官方价格页，SaaS/云原生 WAF 高级版基础实例为 3,880 元/月；同页列出的“API 安全”高级版加购为 5,000 元/实例/月、BOT 流量管理为 8,000 元/实例/月，并不都包含在 3,880 元里。它比现有 ESA 标准版贵很多。只有在 ESA 调优后仍缺少特定 BOT/业务安全能力，或合规要求指定腾讯产品时，再考虑 PoC：

https://cloud.tencent.com/document/product/627/11730
