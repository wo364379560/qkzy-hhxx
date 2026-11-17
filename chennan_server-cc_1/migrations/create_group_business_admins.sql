CREATE TABLE IF NOT EXISTS `group_business_admins` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `group_id` varchar(128) NOT NULL,
  `user_id` varchar(64) NOT NULL,
  `role` varchar(32) NOT NULL DEFAULT 'business_admin',
  `source` varchar(32) NOT NULL DEFAULT 'tencent_overflow',
  `status` varchar(16) NOT NULL DEFAULT 'active',
  `created_by` varchar(64) NOT NULL DEFAULT '',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_group_business_admin` (`group_id`, `user_id`),
  KEY `idx_group_business_admin_group_status` (`group_id`, `status`, `role`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
