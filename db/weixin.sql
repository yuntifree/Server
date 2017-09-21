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

CREATE TABLE IF NOT EXISTS weixin
(
    id  bigint unsigned NOT NULL AUTO_INCREMENT,
    biz varchar(255) NOT NULL DEFAULT '',
    collect int unsigned NOT NULL DEFAULT 0 COMMENT '采集时间戳',
    ctime   datetime NOT NULL DEFAULT '2016-01-01 00:00:00',
    PRIMARY KEY(id),
    UNIQUE KEY(biz)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS post 
(
    id  bigint unsigned NOT NULL AUTO_INCREMENT,
    biz varchar(255) NOT NULL DEFAULT '',
    field_id    int unsigned NOT NULL DEFAULT 0,
    title   varchar(255) NOT NULL DEFAULT '',
    title_encode text NOT NULL DEFAULT '',
    digest varchar(500) NOT NULL DEFAULT '',
    content_url varchar(500) NOT NULL DEFAULT '',
    source_url varchar(500) NOT NULL DEFAULT '',
    cover varchar(500) NOT NULL DEFAULT '',
    is_multi tinyint unsigned NOT NULL DEFAULT 0 COMMENT '是否多图文',
    is_top tinyint unsigned NOT NULL DEFAULT 0 COMMENT '是否头条',
    datetime int unsigned NOT NULL DEFAULT 0 COMMENT '文章时间戳',
    read_num int unsigned NOT NULL DEFAULT 0 COMMENT '阅读量',
    like_num int unsigned NOT NULL DEFAULT 0 COMMENT '点赞量',
    PRIMARY KEY(id),
    KEY(biz)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS task_list
(
    id  bigint unsigned NOT NULL AUTO_INCREMENT,
    content_url varchar(255) NOT NULL DEFAULT '',
    is_load tinyint unsigned NOT NULL DEFAULT 0,
    PRIMARY KEY(id),
    UNIQUE KEY(content_url)
) ENGINE = InnoDB;


CREATE TABLE IF NOT EXISTS news_tags
(
    id  bigint unsigned NOT NULL AUTO_INCREMENT,
    content varchar(128) NOT NULL,
    ctime   datetime NOT NULL DEFAULT '2016-01-01 00:00:00',
    times   int unsigned NOT NULL DEFAULT 0,
    PRIMARY KEY(id),
    UNIQUE KEY(content)
) ENGINE = InnoDB;
