CREATE TABLE group_notices (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    group_id BIGINT UNSIGNED NOT NULL,
    content TEXT NOT NULL,
    summary VARCHAR(255) NOT NULL DEFAULT '',
    updated_by_admin_id BIGINT UNSIGNED NULL,
    updated_by_user_id BIGINT UNSIGNED NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    UNIQUE KEY uk_group_notice_group (group_id),
    KEY idx_group_notice_admin (updated_by_admin_id),
    KEY idx_group_notice_user (updated_by_user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='群公告扩展表';
