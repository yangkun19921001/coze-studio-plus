-- Modify "mcp_plugin" table
ALTER TABLE `opencoze`.`mcp_plugin` ADD COLUMN `user_id` bigint unsigned NOT NULL DEFAULT 0 COMMENT "User ID, 0 for system plugins" AFTER `plugin_id`;
-- Add index for user_id to improve query performance
ALTER TABLE `opencoze`.`mcp_plugin` ADD INDEX `idx_user_id` (`user_id`);

