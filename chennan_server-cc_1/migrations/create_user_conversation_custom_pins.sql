CREATE TABLE user_conversation_custom_pins (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    owner_user_id BIGINT UNSIGNED NOT NULL,
    conversation_id VARCHAR(128) NOT NULL,
    pin_order INT NOT NULL DEFAULT 0,
    pinned_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    UNIQUE KEY uk_owner_conversation (owner_user_id, conversation_id),
    KEY idx_owner_pin_order (owner_user_id, pin_order, pinned_at),
    KEY idx_owner_updated_at (owner_user_id, updated_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户自定义会话置顶';
