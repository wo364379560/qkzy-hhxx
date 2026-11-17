ALTER TABLE admin_users
  MODIFY COLUMN role ENUM('super_admin','admin','assistant') NOT NULL DEFAULT 'assistant';

ALTER TABLE admin_users
  ADD COLUMN parent_admin_id BIGINT UNSIGNED NULL AFTER role,
  ADD INDEX idx_admin_users_parent_admin_id (parent_admin_id);

UPDATE admin_users
SET role = 'super_admin'
WHERE role = 'admin';

UPDATE admin_users assistant
JOIN admin_users super_admin
  ON super_admin.role = 'super_admin'
SET assistant.parent_admin_id = super_admin.id
WHERE assistant.role = 'assistant';
