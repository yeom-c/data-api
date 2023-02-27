CREATE TABLE `upload_ref` (
    `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
    `upload_id` int(10) unsigned NOT NULL,
    `ref_table` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
    `ref_id` int(10) unsigned NOT NULL,
    `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
    PRIMARY KEY (`id`),
    KEY `idx_upload_id` (`upload_id`) USING BTREE,
    KEY `idx_ref_table_ref_id` (`ref_table`,`ref_id`) USING BTREE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
