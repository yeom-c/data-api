CREATE TABLE `upload` (
    `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
    `file_size` int(10) unsigned NOT NULL DEFAULT 0,
    `file_type` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
    `file_name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
    `url` text CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
    `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
    PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
