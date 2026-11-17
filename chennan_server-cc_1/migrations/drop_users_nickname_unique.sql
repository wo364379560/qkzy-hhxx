-- 取消 users.nickname 的唯一约束，允许 IM 用户昵称重复。
-- 兼容未知索引名：动态查 information_schema 后逐个删除。

SET @schema_name = DATABASE();

SELECT GROUP_CONCAT(
         CONCAT('ALTER TABLE `users` DROP INDEX `', index_name, '`')
         SEPARATOR '; '
       )
INTO @drop_sql
FROM (
  SELECT DISTINCT index_name
  FROM information_schema.statistics
  WHERE table_schema = @schema_name
    AND table_name = 'users'
    AND column_name = 'nickname'
    AND non_unique = 0
    AND index_name <> 'PRIMARY'
) AS nickname_indexes;

SET @drop_sql = IFNULL(@drop_sql, 'SELECT 1');

PREPARE stmt FROM @drop_sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
