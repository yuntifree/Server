use yunxing;

CREATE TABLE IF NOT EXISTS user (
    uid     bigint unsigned NOT NULL AUTO_INCREMENT,
    username    varchar(32) NOT NULL,
    phone   varchar(16) NOT NULL DEFAULT '',
    password    varchar(32) NOT NULL,
    wifi_passwd varchar(32) NOT NULL,
    salt        varchar(32) NOT NULL,
    -- term 0: android 1:ios
    term    tinyint unsigned NOT NULL DEFAULT 0,
    version int unsigned NOT NULL DEFAULT 0,
    udid    varchar(32) NOT NULL DEFAULT '',
    model   varchar(32) NOT NULL DEFAULT '',
    channel varchar(32) NOT NULL DEFAULT '',
    deleted tinyint unsigned NOT NULL DEFAULT 0,
    token   varchar(32) NOT NULL DEFAULT '',
    private varchar(32) NOT NULL DEFAULT '',
    ctime   datetime NOT NULL DEFAULT '2016-01-01 00:00:00',
    atime   datetime NOT NULL DEFAULT '2016-01-01 00:00:00',
    etime   datetime NOT NULL DEFAULT '2016-01-01 00:00:00',
    PRIMARY KEY(uid),
    UNIQUE KEY(username),
    KEY(phone)
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

CREATE TABLE IF NOT EXISTS news (
    id      bigint unsigned NOT NULL AUTO_INCREMENT,
    title   varchar(256) NOT NULL,
    img1    varchar(256) NOT NULL DEFAULT '',
    img2    varchar(256) NOT NULL DEFAULT '',
    img3    varchar(256) NOT NULL DEFAULT '',
    vid     varchar(256) NOT NULL DEFAULT '',
    source  varchar(128) NOT NULL DEFAULT '',
    dst     varchar(256) NOT NULL,
    md5     varchar(32) NOT NULL,
    stype   tinyint unsigned NOT NULL DEFAULT 0,
    deleted tinyint unsigned NOT NULL DEFAULT 0,
    ctime   datetime NOT NULL DEFAULT '2016-01-01 00:00:00',
    dtime   datetime NOT NULL DEFAULT '2016-01-01 00:00:00',
    PRIMARY KEY(id),
    UNIQUE KEY(md5)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS video LIKE news;

CREATE TABLE IF NOT EXISTS ad LIKE news;

-- data from ZTE
CREATE TABLE IF NOT EXISTS ap (
    id      int unsigned NOT NULL AUTO_INCREMENT,
    longitude   double NOT NULL,
    latitude    double NOT NULL,
    address     varchar(256) NOT NULL,
    PRIMARY KEY(id)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS ap_stat (
    sid     bigint unsigned NOT NULL AUTO_INCREMENT,
    aid     int unsigned NOT NULL,
    stime   datetime NOT NULL DEFAULT '2016-01-01 00:00:00',
    etime   datetime NOT NULL DEFAULT '2016-01-01 00:00:00',
    count   int unsigned NOT NULL DEFAULT 0,
    traffic int unsigned NOT NULL DEFAULT 0,
    PRIMARY KEY(sid),
    KEY(aid),
    KEY(stime)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS user_record (
    rid     bigint unsigned NOT NULL AUTO_INCREMENT,
    username    varchar(32) NOT NULL,
    aid     int unsigned NOT NULL,
    stime   datetime NOT NULL DEFAULT '2016-01-01 00:00:00',
    etime   datetime NOT NULL DEFAULT '2016-01-01 00:00:00',
    PRIMARY KEY(rid),
    KEY(username),
    KEY(aid)
) ENGINE = InnoDB;

