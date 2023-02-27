CREATE TABLE `data_schema` (
    `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
    `type` int(10) unsigned NOT NULL DEFAULT 0,
    `server_id` int(10) unsigned NOT NULL,
    `version` text DEFAULT NULL,
    `update_lock` int(2) unsigned NOT NULL DEFAULT 0,
    `updated_at` timestamp NULL DEFAULT NULL ON UPDATE current_timestamp(),
    `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
    PRIMARY KEY (`id`),
    UNIQUE KEY `uniq_server_id_type` (`server_id`,`type`) USING BTREE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
