UPDATE users
SET ip_whitelist_ip = ''
WHERE ip_whitelist_ip IS NULL;

ALTER TABLE users
MODIFY COLUMN ip_whitelist_ip TEXT NOT NULL COMMENT '用户登录 IP 白名单，多条以换行分隔';
