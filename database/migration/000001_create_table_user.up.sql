CREATE TABLE `user` (
    `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
    `employee_id` int(10) unsigned DEFAULT NULL,
    `email` varchar(255) NOT NULL,
    `name` varchar(255) NOT NULL,
    `hashed_password` varchar(255) DEFAULT '',
    `password_changed_at` timestamp NULL DEFAULT NULL,
    `position` varchar(255) DEFAULT '',
    `color` varchar(255) NULL DEFAULT '',
    `joined_at` date DEFAULT NULL,
    `retired_at` date DEFAULT NULL,
    `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
    PRIMARY KEY (`id`),
    UNIQUE KEY `uniq_email` (`email`) USING BTREE,
    UNIQUE KEY `uniq_employee_id` (`employee_id`) USING BTREE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
