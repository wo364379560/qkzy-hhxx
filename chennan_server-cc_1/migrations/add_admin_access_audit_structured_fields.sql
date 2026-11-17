ALTER TABLE admin_access_audits
    ADD COLUMN target_type VARCHAR(32) NOT NULL DEFAULT '' AFTER target_id,
    ADD COLUMN target_username VARCHAR(64) NOT NULL DEFAULT '' AFTER target_type,
    ADD COLUMN target_display_name VARCHAR(128) NOT NULL DEFAULT '' AFTER target_username,
    ADD COLUMN action_key VARCHAR(64) NOT NULL DEFAULT '' AFTER access_type,
    ADD COLUMN action_summary VARCHAR(128) NOT NULL DEFAULT '' AFTER action_key,
    ADD COLUMN change_set_json JSON NULL AFTER action_detail;

ALTER TABLE admin_access_audits
    ADD KEY idx_target_type_created_at (target_type, created_at),
    ADD KEY idx_action_key_created_at (action_key, created_at),
    ADD KEY idx_target_username (target_username);
