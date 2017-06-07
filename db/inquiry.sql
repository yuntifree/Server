use yunxing;

CREATE TABLE IF NOT EXISTS users (
    uid     bigint unsigned NOT NULL AUTO_INCREMENT,
    username    varchar(32) NOT NULL,
    phone   varchar(16) NOT NULL DEFAULT '',
    token   varchar(32) NOT NULL DEFAULT '',
    nickname    varchar(128) NOT NULL DEFAULT '',
    headurl     varchar(256) NOT NULL DEFAULT '',
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
