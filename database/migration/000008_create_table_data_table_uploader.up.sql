CREATE TABLE `data_table_uploader` (
    `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
    `data_table_id` int(10) unsigned NOT NULL,
    `user_id` int(10) unsigned NOT NULL,
    PRIMARY KEY (`id`) USING BTREE,
    KEY `idx_data_table_id_user_id` (`data_table_id`,`user_id`) USING BTREE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
