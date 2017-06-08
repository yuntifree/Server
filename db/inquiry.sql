use inquiry;

CREATE TABLE IF NOT EXISTS users (
    uid     bigint unsigned NOT NULL AUTO_INCREMENT,
    username    varchar(32) NOT NULL,
    phone   varchar(16) NOT NULL DEFAULT '',
    token   varchar(32) NOT NULL DEFAULT '',
    nickname    varchar(128) NOT NULL DEFAULT '',
    headurl     varchar(256) NOT NULL DEFAULT '',
    gender      tinyint unsigned NOT NULL DEFAULT 0,
    role        tinyint unsigned NOT NULL DEFAULT 0,
    ctime   datetime NOT NULL DEFAULT '2017-01-01',
    PRIMARY KEY(uid),
    UNIQUE KEY(username)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS wx_openid (
    id      int unsigned NOT NULL AUTO_INCREMENT,
    unionid varchar(36) NOT NULL,
    openid  varchar(32) NOT NULL,
    sid     varchar(32) NOT NULL,
    ctime   datetime NOT NULL DEFAULT '2017-01-01',
    PRIMARY KEY(id),
    UNIQUE KEY(openid),
    KEY(unionid),
    KEY(sid)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS doctor (
    id      int unsigned NOT NULL AUTO_INCREMENT,
    phone   varchar(16) NOT NULL,
    name    varchar(64) NOT NULL,
    title   varchar(128) NOT NULL,
    hospital varchar(256) NOT NULL,
    department varchar(128) NOT NULL,
    headurl    varchar(128) NOT NULL,
    ctime   datetime NOT NULL DEFAULT '2017-01-01',
    PRIMARY KEY(id),
    UNIQUE KEY(phone)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS phone_code (
    pid     bigint unsigned NOT NULL AUTO_INCREMENT,
    phone   varchar(16) NOT NULL DEFAULT '',
    uid     int unsigned NOT NULL DEFAULT 0,
    code    int unsigned NOT NULL DEFAULT 0,
    ctime   datetime NOT NULL DEFAULT '2016-01-01 00:00:00',
    stime   datetime NOT NULL DEFAULT '2016-01-01 00:00:00',
    etime   datetime NOT NULL DEFAULT '2016-01-01 00:00:00',
    used    tinyint unsigned NOT NULL DEFAULT 0,
    PRIMARY KEY(pid),
    KEY(phone, uid)
) ENGINE = InnoDB;

