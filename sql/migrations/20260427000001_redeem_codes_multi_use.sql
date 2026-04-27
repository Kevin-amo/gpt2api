-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS `redeem_codes` (
    `id`               BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `code`             VARCHAR(64)     NOT NULL COMMENT '兑换码,全局唯一',
    `credits`          BIGINT          NOT NULL COMMENT '到账积分(厘)',
    `batch_no`         VARCHAR(32)     NOT NULL DEFAULT '' COMMENT '批次号',
    `remark`           VARCHAR(255)    NOT NULL DEFAULT '',
    `created_by`       BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建管理员 ID',
    `used_by`          BIGINT UNSIGNED NULL COMMENT '兑换用户 ID',
    `used_at`          DATETIME        NULL,
    `created_at`       DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`       DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `deleted_at`       DATETIME        NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_code` (`code`),
    KEY `idx_batch_created` (`batch_no`, `created_at`),
    KEY `idx_used_by` (`used_by`),
    KEY `idx_used_at` (`used_at`),
    KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='积分兑换码';

SET @has_batch_id := (
    SELECT COUNT(*) FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'redeem_codes' AND COLUMN_NAME = 'batch_id'
);
SET @has_batch_no := (
    SELECT COUNT(*) FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'redeem_codes' AND COLUMN_NAME = 'batch_no'
);
SET @sql := IF(
    @has_batch_id > 0 AND @has_batch_no = 0,
    'ALTER TABLE `redeem_codes` CHANGE COLUMN `batch_id` `batch_no` VARCHAR(32) NOT NULL DEFAULT '''' COMMENT ''批次号''',
    'SELECT 1'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @has_used_by_user_id := (
    SELECT COUNT(*) FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'redeem_codes' AND COLUMN_NAME = 'used_by_user_id'
);
SET @has_used_by := (
    SELECT COUNT(*) FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'redeem_codes' AND COLUMN_NAME = 'used_by'
);
SET @sql := IF(
    @has_used_by_user_id > 0 AND @has_used_by = 0,
    'ALTER TABLE `redeem_codes` CHANGE COLUMN `used_by_user_id` `used_by` BIGINT UNSIGNED NULL COMMENT ''兑换用户 ID''',
    'SELECT 1'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @sql := IF(
    EXISTS(
        SELECT 1 FROM information_schema.COLUMNS
        WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'redeem_codes' AND COLUMN_NAME = 'remark'
    ),
    'SELECT 1',
    'ALTER TABLE `redeem_codes` ADD COLUMN `remark` VARCHAR(255) NOT NULL DEFAULT '''' AFTER `batch_no`'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @sql := IF(
    EXISTS(
        SELECT 1 FROM information_schema.COLUMNS
        WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'redeem_codes' AND COLUMN_NAME = 'created_by'
    ),
    'SELECT 1',
    'ALTER TABLE `redeem_codes` ADD COLUMN `created_by` BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER `remark`'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @sql := IF(
    EXISTS(
        SELECT 1 FROM information_schema.COLUMNS
        WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'redeem_codes' AND COLUMN_NAME = 'updated_at'
    ),
    'SELECT 1',
    'ALTER TABLE `redeem_codes` ADD COLUMN `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP AFTER `created_at`'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @sql := IF(
    EXISTS(
        SELECT 1 FROM information_schema.COLUMNS
        WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'redeem_codes' AND COLUMN_NAME = 'deleted_at'
    ),
    'SELECT 1',
    'ALTER TABLE `redeem_codes` ADD COLUMN `deleted_at` DATETIME NULL AFTER `updated_at`'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @sql := IF(
    EXISTS(
        SELECT 1 FROM information_schema.COLUMNS
        WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'redeem_codes' AND COLUMN_NAME = 'used_by'
    ),
    'ALTER TABLE `redeem_codes` MODIFY COLUMN `used_by` BIGINT UNSIGNED NULL COMMENT ''兑换用户 ID''',
    'SELECT 1'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

UPDATE `redeem_codes` SET `used_by` = NULL WHERE `used_by` = 0;

SET @sql := IF(
    EXISTS(
        SELECT 1 FROM information_schema.STATISTICS
        WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'redeem_codes' AND INDEX_NAME = 'uk_used_by'
    ),
    'ALTER TABLE `redeem_codes` DROP INDEX `uk_used_by`',
    'SELECT 1'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @sql := IF(
    EXISTS(
        SELECT 1 FROM information_schema.STATISTICS
        WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'redeem_codes' AND INDEX_NAME = 'idx_used_by'
    ),
    'SELECT 1',
    'ALTER TABLE `redeem_codes` ADD KEY `idx_used_by` (`used_by`)'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @sql := IF(
    EXISTS(
        SELECT 1 FROM information_schema.STATISTICS
        WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'redeem_codes' AND INDEX_NAME = 'idx_batch_created'
    ),
    'SELECT 1',
    'ALTER TABLE `redeem_codes` ADD KEY `idx_batch_created` (`batch_no`, `created_at`)'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @sql := IF(
    EXISTS(
        SELECT 1 FROM information_schema.STATISTICS
        WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'redeem_codes' AND INDEX_NAME = 'idx_used_at'
    ),
    'SELECT 1',
    'ALTER TABLE `redeem_codes` ADD KEY `idx_used_at` (`used_at`)'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @sql := IF(
    EXISTS(
        SELECT 1 FROM information_schema.STATISTICS
        WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'redeem_codes' AND INDEX_NAME = 'idx_deleted_at'
    ),
    'SELECT 1',
    'ALTER TABLE `redeem_codes` ADD KEY `idx_deleted_at` (`deleted_at`)'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 1;
-- +goose StatementEnd