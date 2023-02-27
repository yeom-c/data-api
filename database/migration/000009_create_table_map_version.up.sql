CREATE TABLE `map_version` (
    `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
    `version` int(10) unsigned NOT NULL,
    `status` int(10) unsigned NOT NULL DEFAULT 0,
    `error` longtext DEFAULT NULL,
    `data` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`data`)),
    `memo_title` varchar(255) DEFAULT '',
    `memo` text DEFAULT '',
    `data_table_id` int(10) unsigned NOT NULL,
    `user_id` int(10) unsigned NOT NULL,
    `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
    PRIMARY KEY (`id`) USING BTREE,
    KEY `idx_data_table_id` (`data_table_id`) USING BTREE,
    KEY `idx_user_id` (`user_id`) USING BTREE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;