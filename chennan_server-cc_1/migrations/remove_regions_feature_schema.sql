-- 区域功能数据库结构回滚脚本
-- 用途：
-- 1. 删除区域功能新增的表：regions、region_admin_bindings
-- 2. 删除 users 表中的 region_id 字段
--
-- 执行前请确认：
-- 1. 代码侧已经完成“区域功能”回滚
-- 2. 不再需要保留任何区域数据
-- 3. 当前环境允许物理删除该结构

DROP TABLE IF EXISTS region_admin_bindings;
DROP TABLE IF EXISTS regions;

SET @drop_users_region_column := (
  SELECT IF(
    COUNT(*) > 0,
    'ALTER TABLE users DROP COLUMN region_id',
    'SELECT 1'
  )
  FROM information_schema.columns
  WHERE table_schema = DATABASE()
    AND table_name = 'users'
    AND column_name = 'region_id'
);

PREPARE stmt_drop_users_region_column FROM @drop_users_region_column;
EXECUTE stmt_drop_users_region_column;
DEALLOCATE PREPARE stmt_drop_users_region_column;
