use weixin;

CREATE TABLE IF NOT EXISTS gzh 
(
    id  bigint unsigned NOT NULL AUTO_INCREMENT,
    wx_id varchar(128) NOT NULL,
    wx_name varchar(256) NOT NULL,
    ctime   datetime NOT NULL DEFAULT '2016-01-01 00:00:00',
    PRIMARY KEY(id),
    UNIQUE KEY(wx_id)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS gzh_article
(
    id  bigint unsigned NOT NULL AUTO_INCREMENT,
    wid int unsigned NOT NULL,
    title varchar(256) NOT NULL,
    md5 varchar(32) NOT NULL,
    readnum int unsigned NOT NULL DEFAULT 0,
    likenum int unsigned NOT NULL DEFAULT 0,
    ctime   datetime NOT NULL DEFAULT '2016-01-01 00:00:00',
    PRIMARY KEY(id),
    UNIQUE KEY(wid, md5)
) ENGINE = InnoDB;
