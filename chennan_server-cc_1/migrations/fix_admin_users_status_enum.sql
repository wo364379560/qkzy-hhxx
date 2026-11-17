-- 检查 admin_users 表的 status 列定义
-- 如果 status 列不是 ENUM('active', 'banned')，则需要修复

-- 修复 admin_users 表的 status 字段为正确的 ENUM 类型
-- 注意：执行前请备份数据

-- 1. 先查看当前 status 列的定义
-- SHOW COLUMNS FROM admin_users LIKE 'status';

-- 2. 修改 status 列为正确的 ENUM 类型
ALTER TABLE admin_users
MODIFY COLUMN status ENUM('active', 'banned') NOT NULL DEFAULT 'active' COMMENT '账号状态';

-- 3. 验证修改结果
-- SHOW COLUMNS FROM admin_users LIKE 'status';
