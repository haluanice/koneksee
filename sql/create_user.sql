CREATE TABLE `users` (
  `user_id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `user_name` varchar(32) NOT NULL,
  `mobile_phone` varchar(19) NOT NULL,
  `profile_picture` varchar(128) NOT NULL,
  `token` varchar(128) NOT NULL,
  PRIMARY KEY (`user_id`),
  UNIQUE KEY `uq_mobile_phone` (`mobile_phone`)
) ENGINE=InnoDB AUTO_INCREMENT=38 DEFAULT CHARSET=utf8;
