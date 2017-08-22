use yunxing;

CREATE TABLE IF NOT EXISTS label 
(
    id  bigint unsigned NOT NULL AUTO_INCREMENT,
    type int unsigned NOT NULL DEFAULT 0,
    content varchar(256) NOT NULL DEFAULT '',
    ctime   datetime NOT NULL DEFAULT '2016-01-01 00:00:00',
    PRIMARY KEY(id),
    KEY(type)
) ENGINE = InnoDB;


CREATE TABLE IF NOT EXISTS user_label
(
    id  bigint unsigned NOT NULL AUTO_INCREMENT,
    uid int unsigned NOT NULL,
    lid int unsigned NOT NULL,
    deleted tinyint unsigned NOT NULL DEFAULT 0,
    ctime   datetime NOT NULL DEFAULT '2016-01-01 00:00:00',
    PRIMARY KEY(id),
    UNIQUE KEY(uid, lid)
) ENGINE = InnoDB;
