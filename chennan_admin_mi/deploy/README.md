# 睿之禾后台 r2 部署（2026-11-04）

生产默认管理接口为 `https://api.ruizhihe.com/api/v1/admin`，与最新交接一致。后台页面仍为 `https://rzh35ht.ruizhihe.com`。不要换成四条 ESA/CDN，也不要给后台静态资源配置 HTTPDNS SDK 密钥。

本轮只读复查：CF 健康接口返回 HTTP200/code0，并允许后台 Origin；管理站同域 `/api/v1/chat/ping` 仍返回网页 HTML，同域 API 代理不能作为当前生产默认。四 ESA/CDN 的 OPTIONS204 不代表允许后台跨域。

## 发布

1. 备份管理站当前静态文件。确认宝塔 `rzh35ht.ruizhihe.com` 的实际网站目录。
2. `npm run build`，将 `dist/` 内容或对应后台 ZIP 解压到该目录，根目录直接包含 `index.html`、`assets/`。
3. 构建时不设置 `VITE_ADMIN_API_BASE_URL`，使用上述 CF 默认值。清理 HTML/CDN 缓存并强制刷新。
4. 浏览器 Network 确认登录地址为 `https://api.ruizhihe.com/api/v1/admin/login`，返回 JSON；登录、助理新增及好友归属页面分别验收。

开发环境仍使用 `/api/v1/admin` 走 Vite 代理。可选的 `nginx/baota-admin-api-proxy.conf` 只供确需同域代理的环境使用，上游也为 CF；需确认服务器 CA 文件路径、配置检查通过、同域健康接口确实返回 JSON 后，才显式构建 `VITE_ADMIN_API_BASE_URL=/api/v1/admin`。默认生产包不要求改 Nginx 代理。

## 验证与回滚

构建包含类型检查。本轮网络配置不需要执行任何 SQL，不改数据库和腾讯回调 Token。回滚时恢复此前静态文件并刷新 HTML 缓存，保留原服务和数据库。
