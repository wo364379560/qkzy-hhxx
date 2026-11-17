-- Optional manual migration. The normal startup AutoMigrate adds these columns
-- and the non-unique lookup index automatically. Do NOT run twice or after
-- AutoMigrate; check SHOW COLUMNS/SHOW INDEX first. Existing tokens stay valid.
ALTER TABLE user_refresh_sessions
  ADD COLUMN previous_refresh_token_hash varchar(64) NULL,
  ADD COLUMN previous_refresh_expires_at datetime(3) NULL,
  ADD COLUMN refresh_token_ciphertext varchar(512) NULL,
  ADD INDEX idx_user_refresh_sessions_previous_refresh_token_hash (previous_refresh_token_hash);
-- All instances must use the same existing JWT_SECRET. No new secret is needed.
-- Upgrade all API instances together: an old binary cannot resolve a prior
-- token during the new 60-second grace window. Keep added columns on rollback.
