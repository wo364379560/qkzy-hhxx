CREATE TABLE IF NOT EXISTS group_message_pins (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    group_id BIGINT UNSIGNED NOT NULL,
    im_group_id VARCHAR(64) NOT NULL DEFAULT '',
    message_id VARCHAR(128) NOT NULL,
    message_seq BIGINT NOT NULL DEFAULT 0,
    message_sender_im_user_id VARCHAR(64) NOT NULL DEFAULT '',
    message_sender_name VARCHAR(128) NOT NULL DEFAULT '',
    message_type VARCHAR(32) NOT NULL DEFAULT '',
    message_abstract VARCHAR(255) NOT NULL DEFAULT '',
    operator_user_id BIGINT UNSIGNED NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'active',
    pinned_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    KEY idx_group_message_pins_group_status (group_id, status),
    KEY idx_group_message_pins_operator (operator_user_id),
    KEY idx_group_message_pins_message_id (message_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='群内消息置顶';
