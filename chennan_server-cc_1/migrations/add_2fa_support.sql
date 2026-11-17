-- 添加 admin_users 表字段
ALTER TABLE admin_users
ADD COLUMN totp_secret VARCHAR(255) DEFAULT '' COMMENT 'TOTP密钥(加密)',
ADD COLUMN totp_enabled BOOLEAN DEFAULT FALSE COMMENT '是否已绑定TOTP',
ADD COLUMN totp_bound_at TIMESTAMP NULL COMMENT '绑定时间';

-- 创建备份码表
CREATE TABLE admin_totp_backup_codes (
    id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    admin_id INT UNSIGNED NOT NULL,
    code_hash VARCHAR(255) NOT NULL COMMENT 'bcrypt hash',
    used BOOLEAN DEFAULT FALSE,
    used_at TIMESTAMP NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_admin_id (admin_id),
    INDEX idx_used (used)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='TOTP备份码表';

CREATE TABLE system_settings (
    setting_key VARCHAR(64) PRIMARY KEY,
    value VARCHAR(255) NOT NULL,
    updated_by INT UNSIGNED NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='系统配置表';
