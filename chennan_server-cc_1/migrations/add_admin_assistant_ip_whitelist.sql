ALTER TABLE admin_users
ADD COLUMN ip_whitelist_enabled BOOLEAN NOT NULL DEFAULT FALSE COMMENT '是否启用助理IP白名单';

CREATE TABLE admin_user_ip_whitelists (
    id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    admin_user_id INT UNSIGNED NOT NULL,
    cidr VARCHAR(64) NOT NULL COMMENT '标准化后的IP/CIDR',
    note VARCHAR(255) NOT NULL DEFAULT '' COMMENT '规则备注',
    created_by INT UNSIGNED NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_admin_user_cidr (admin_user_id, cidr),
    KEY idx_admin_user_id (admin_user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='后台助理IP白名单';
