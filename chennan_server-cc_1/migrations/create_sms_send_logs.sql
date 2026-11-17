CREATE TABLE IF NOT EXISTS sms_send_logs (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    phone VARCHAR(32) NOT NULL,
    ip_address VARCHAR(64) NOT NULL DEFAULT '',
    device_id VARCHAR(128) NOT NULL DEFAULT '',
    scene VARCHAR(32) NOT NULL DEFAULT 'general',
    status VARCHAR(16) NOT NULL,
    reject_reason VARCHAR(128) NOT NULL DEFAULT '',
    provider VARCHAR(32) NOT NULL DEFAULT '',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    KEY idx_sms_send_logs_phone_created_at (phone, created_at),
    KEY idx_sms_send_logs_ip_created_at (ip_address, created_at),
    KEY idx_sms_send_logs_device_created_at (device_id, created_at),
    KEY idx_sms_send_logs_scene_created_at (scene, created_at),
    KEY idx_sms_send_logs_status_created_at (status, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='短信发送日志';
