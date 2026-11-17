ALTER TABLE users
  ADD COLUMN can_create_group TINYINT(1) NOT NULL DEFAULT 0
  AFTER allow_type;
