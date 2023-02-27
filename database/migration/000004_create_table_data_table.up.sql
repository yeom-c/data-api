CREATE TABLE `data_table` (
    `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
    `type` int(10) unsigned NOT NULL DEFAULT 0,
    `name` varchar(255) NOT NULL,
    `sheet_name` varchar(255) NOT NULL,
    `latest_version` int(10) unsigned NOT NULL DEFAULT 0,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uniq_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
