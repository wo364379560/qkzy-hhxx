ALTER TABLE `chat_keywords`
ADD COLUMN IF NOT EXISTS `replacement` VARCHAR(128) NOT NULL DEFAULT '**' AFTER `keyword`;
