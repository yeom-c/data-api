CREATE TABLE `server` (
    `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
    `env` varchar(255) NOT NULL,
    `name` varchar(255) NOT NULL,
    `description` varchar(255) DEFAULT NULL,
    `updated_at` timestamp NULL DEFAULT NULL ON UPDATE current_timestamp(),
    `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
    PRIMARY KEY (`id`),
    UNIQUE KEY `uniq_env` (`env`) USING BTREE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
