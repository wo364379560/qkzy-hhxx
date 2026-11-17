ALTER TABLE users
ADD COLUMN community_post_disabled BOOLEAN NOT NULL DEFAULT FALSE COMMENT '是否禁止发帖',
ADD COLUMN community_comment_disabled BOOLEAN NOT NULL DEFAULT FALSE COMMENT '是否禁止评论';
