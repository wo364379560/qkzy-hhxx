-- Rui additive reliability migration, 2026-11-01.
-- Reviewed against MySQL 8.0 in an isolated database. Back up production first.
-- Existing chat_attachments/users tables stay in place. Do not run both this file and the optional recovery ALTER script.

CREATE TABLE IF NOT EXISTS `forward_jobs` (
  `id` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL,
  `sender` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL,
  `request_id` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL,
  `fingerprint` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `protocol` bigint NOT NULL DEFAULT '0',
  `state` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `target_count` bigint NOT NULL DEFAULT '0',
  `message_count` bigint NOT NULL DEFAULT '0',
  `uploaded_targets` bigint NOT NULL DEFAULT '0',
  `uploaded_messages` bigint NOT NULL DEFAULT '0',
  `materialized` bigint NOT NULL DEFAULT '0',
  `fill_at` bigint NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  KEY `idx_forward_jobs_sender` (`sender`),
  KEY `idx_forward_jobs_state` (`state`),
  KEY `idx_forward_jobs_fill_at` (`fill_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `forward_targets` (
  `job_id` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL,
  `ordinal` bigint NOT NULL,
  `identity` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL,
  `conversation_id` varchar(128) COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(128) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  PRIMARY KEY (`job_id`,`ordinal`),
  UNIQUE KEY `idx_forward_target_identity` (`job_id`,`identity`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `forward_messages` (
  `job_id` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL,
  `ordinal` bigint NOT NULL,
  `fingerprint` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL,
  `body` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `preview` varchar(256) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  PRIMARY KEY (`job_id`,`ordinal`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `forward_deliveries` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `job_id` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL,
  `sender` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL,
  `conversation_id` varchar(128) COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(128) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `ordinal` bigint DEFAULT NULL,
  `preview` varchar(256) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `message` text COLLATE utf8mb4_unicode_ci NOT NULL,
  `status` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL,
  `next_at` bigint DEFAULT NULL,
  `lease_until` bigint DEFAULT NULL,
  `claim` varchar(64) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `attempts` bigint DEFAULT NULL,
  `random` int unsigned DEFAULT NULL,
  `sequence` int unsigned DEFAULT NULL,
  `error` varchar(256) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `last_error_command` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `last_error_code` bigint NOT NULL DEFAULT '0',
  `last_error_info` varchar(512) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `last_tencent_request_id` varchar(128) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `result` varchar(128) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_forward_order` (`job_id`,`conversation_id`,`ordinal`),
  KEY `idx_forward_status` (`job_id`,`status`),
  KEY `idx_forward_deliveries_sender` (`sender`),
  KEY `idx_forward_due` (`status`,`next_at`),
  KEY `idx_forward_deliveries_lease_until` (`lease_until`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `forward_retries` (
  `job_id` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL,
  `retry_id` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `cursor` bigint unsigned NOT NULL DEFAULT '0',
  `upper_id` bigint unsigned NOT NULL DEFAULT '0',
  `done` tinyint(1) NOT NULL DEFAULT '0',
  PRIMARY KEY (`job_id`,`retry_id`),
  KEY `idx_forward_retries_done` (`done`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `forward_gates` (
  `id` varchar(160) COLLATE utf8mb4_unicode_ci NOT NULL,
  `next_at` bigint DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `chat_multipart_uploads` (
  `id` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL,
  `file_id` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL,
  `owner` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL,
  `fingerprint` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL,
  `upload_id` varchar(256) COLLATE utf8mb4_unicode_ci NOT NULL,
  `part_size` bigint NOT NULL,
  `state` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL,
  `lease_until` bigint DEFAULT NULL,
  `attempts` bigint DEFAULT NULL,
  `expires_at` datetime(3) DEFAULT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_chat_multipart_uploads_file_id` (`file_id`),
  KEY `idx_chat_multipart_uploads_owner` (`owner`),
  KEY `idx_chat_multipart_uploads_state` (`state`),
  KEY `idx_chat_multipart_uploads_lease_until` (`lease_until`),
  KEY `idx_chat_multipart_uploads_expires_at` (`expires_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

SET @rui_ddl = (SELECT IF(COUNT(*) = 0, 'ALTER TABLE `user_refresh_sessions` ADD COLUMN `previous_refresh_token_hash` varchar(64) NULL', 'SELECT 1') FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'user_refresh_sessions' AND COLUMN_NAME = 'previous_refresh_token_hash');
PREPARE rui_stmt FROM @rui_ddl;
EXECUTE rui_stmt;
DEALLOCATE PREPARE rui_stmt;

SET @rui_ddl = (SELECT IF(COUNT(*) = 0, 'ALTER TABLE `user_refresh_sessions` ADD COLUMN `previous_refresh_expires_at` datetime(3) NULL', 'SELECT 1') FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'user_refresh_sessions' AND COLUMN_NAME = 'previous_refresh_expires_at');
PREPARE rui_stmt FROM @rui_ddl;
EXECUTE rui_stmt;
DEALLOCATE PREPARE rui_stmt;

SET @rui_ddl = (SELECT IF(COUNT(*) = 0, 'ALTER TABLE `user_refresh_sessions` ADD COLUMN `refresh_token_ciphertext` varchar(512) NULL', 'SELECT 1') FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'user_refresh_sessions' AND COLUMN_NAME = 'refresh_token_ciphertext');
PREPARE rui_stmt FROM @rui_ddl;
EXECUTE rui_stmt;
DEALLOCATE PREPARE rui_stmt;

SET @rui_ddl = (SELECT IF(COUNT(*) = 0, 'ALTER TABLE `group_notices` ADD COLUMN `pinned` tinyint(1) NOT NULL DEFAULT 0', 'SELECT 1') FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'group_notices' AND COLUMN_NAME = 'pinned');
PREPARE rui_stmt FROM @rui_ddl;
EXECUTE rui_stmt;
DEALLOCATE PREPARE rui_stmt;

SET @rui_ddl = (SELECT IF(COUNT(*) = 0, 'CREATE INDEX idx_user_refresh_sessions_previous_refresh_token_hash ON user_refresh_sessions(previous_refresh_token_hash)', 'SELECT 1') FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'user_refresh_sessions' AND INDEX_NAME = 'idx_user_refresh_sessions_previous_refresh_token_hash');
PREPARE rui_stmt FROM @rui_ddl;
EXECUTE rui_stmt;
DEALLOCATE PREPARE rui_stmt;
