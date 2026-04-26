-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS `redeem_codes` (
    `id`               BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `code`             VARCHAR(64)     NOT NULL COMMENT '兑换码,全局唯一',
    `credits`          BIGINT          NOT NULL COMMENT '到账积分(厘)',
    `batch_no`         VARCHAR(32)     NOT NULL DEFAULT '' COMMENT '批次号',
    `remark`           VARCHAR(255)    NOT NULL DEFAULT '',
    `created_by`       BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建管理员 ID',
    `used_by`          BIGINT UNSIGNED NULL COMMENT '兑换用户 ID; UNIQUE 保证每人限兑一次',
    `used_at`          DATETIME        NULL,
    `created_at`       DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`       DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `deleted_at`       DATETIME        NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_code` (`code`),
    UNIQUE KEY `uk_used_by` (`used_by`),
    KEY `idx_batch_created` (`batch_no`, `created_at`),
    KEY `idx_used_at` (`used_at`),
    KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='积分兑换码';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS `redeem_codes`;
-- +goose StatementEnd