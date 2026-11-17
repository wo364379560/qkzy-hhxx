ALTER TABLE users
  ADD COLUMN last_offline_at DATETIME NULL COMMENT '最后一次从在线变为离线的时间' AFTER last_online_at,
  ADD COLUMN last_presence_status VARCHAR(16) NOT NULL DEFAULT 'unknown' COMMENT '最近一次已知在线状态：online/offline/unknown' AFTER last_offline_at,
  ADD COLUMN last_presence_checked_at DATETIME NULL COMMENT '最近一次在线状态同步时间' AFTER last_presence_status;
