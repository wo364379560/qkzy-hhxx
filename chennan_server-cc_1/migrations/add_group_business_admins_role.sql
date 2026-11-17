ALTER TABLE `group_business_admins`
  ADD COLUMN `role` varchar(32) NOT NULL DEFAULT 'business_admin' AFTER `user_id`;

ALTER TABLE `group_business_admins`
  DROP INDEX `idx_group_business_admin_group_status`,
  ADD KEY `idx_group_business_admin_group_status` (`group_id`, `status`, `role`);
