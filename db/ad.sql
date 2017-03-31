use yunxing;

CREATE TABLE IF NOT EXISTS customer
(
    id     bigint unsigned NOT NULL AUTO_INCREMENT,
    name    varchar(128) NOT NULL,
    contact varchar(128) NOT NULL,
    phone   varchar(16) NOT NULL,
    address varchar(256) NOT NULL,
    remark  varchar(256) NOT NULL,
    ctime   datetime NOT NULL DEFAULT '2017-01-01',
    atime   datetime NOT NULL DEFAULT '2017-01-01',
    etime   datetime NOT NULL DEFAULT '2017-01-01',
    deleted tinyint unsigned NOT NULL DEFAULT 0,
    PRIMARY KEY(id)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS advertise
(
    id     bigint unsigned NOT NULL AUTO_INCREMENT,
    name    varchar(128) NOT NULL,
    version varchar(128) NOT NULL,
    adid    int unsigned NOT NULL,
    areaid  int unsigned NOT NULL,
    -- type 0:banner 1:news 2:login
    type    int unsigned NOT NULL DEFAULT 0,
    tsid    int unsigned NOT NULL,
    img     varchar(128) NOT NULL,
    abstract varchar(256) NOT NULL,
    content  varchar(512) NOT NULL,
    click   bigint unsigned NOT NULL DEFAULT 0,
    deleted tinyint unsigned NOT NULL DEFAULT 0,
    online  tinyint unsigned NOT NULL DEFAULT 0,
    puid    int unsigned NOT NULL DEFAULT 0,
    ctime   datetime NOT NULL DEFAULT '2017-01-01',
    ptime   datetime NOT NULL DEFAULT '2017-01-01',
    PRIMARY KEY(id)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS ad_click
(
    id     bigint unsigned NOT NULL AUTO_INCREMENT,
    uid    int unsigned NOT NULL,
    aid    int unsigned NOT NULL,
    phone   varchar(16) NOT NULL,
    usermac varchar(16) NOT NULL,
    userip  varchar(16) NOT NULL,
    apmac   varchar(16) NOT NULL,
    ctime   datetime NOT NULL DEFAULT '2017-01-01',
    PRIMARY KEY(id),
    KEY(uid, aid)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS ad_click_stat
(
    id     bigint unsigned NOT NULL AUTO_INCREMENT,
    aid    int unsigned NOT NULL,
    cnt    int unsigned NOT NULL DEFAULT 0,
    ctime   date NOT NULL DEFAULT '2017-01-01',
    PRIMARY KEY(id),
    UNIQUE KEY(aid, ctime)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS unit
(
    id     bigint unsigned NOT NULL AUTO_INCREMENT,
    name    varchar(128) NOT NULL,
    address varchar(256) NOT NULL,
    longitude   double NOT NULL,
    latitude    double NOT NULL,
    cnt     int unsigned NOT NULL DEFAULT 0,
    deleted tinyint unsigned NOT NULL DEFAULT 0,
    ctime   datetime NOT NULL DEFAULT '2017-01-01',
    PRIMARY KEY(id),
    KEY(name),
    UNIQUE KEY(longitude, latitude)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS area
(
    id     bigint unsigned NOT NULL AUTO_INCREMENT,
    name    varchar(128) NOT NULL,
    deleted tinyint unsigned NOT NULL DEFAULT 0,
    ctime   datetime NOT NULL DEFAULT '2017-01-01',
    PRIMARY KEY(id),
    UNIQUE KEY(name)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS area_unit
(
    id     bigint unsigned NOT NULL AUTO_INCREMENT,
    aid     bigint unsigned NOT NULL,
    unid     bigint unsigned NOT NULL,
    deleted tinyint unsigned NOT NULL DEFAULT 0,
    ctime   datetime NOT NULL DEFAULT '2017-01-01',
    PRIMARY KEY(id),
    UNIQUE KEY(aid, unid)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS unit_tag
(
    id     bigint unsigned NOT NULL AUTO_INCREMENT,
    name    varchar(128) NOT NULL,
    deleted tinyint unsigned NOT NULL DEFAULT 0,
    ctime   datetime NOT NULL DEFAULT '2017-01-01',
    PRIMARY KEY(id),
    UNIQUE KEY(name)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS unit_tag_relation
(
    id     bigint unsigned NOT NULL AUTO_INCREMENT,
    tid     bigint unsigned NOT NULL,
    unid     bigint unsigned NOT NULL,
    deleted tinyint unsigned NOT NULL DEFAULT 0,
    ctime   datetime NOT NULL DEFAULT '2017-01-01',
    PRIMARY KEY(id),
    UNIQUE KEY(tid, unid)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS timeslot
(
    id     bigint unsigned NOT NULL AUTO_INCREMENT,
    name    varchar(128) NOT NULL,
    start   int unsigned NOT NULL,
    end     int unsigned NOT NULL,
    deleted tinyint unsigned NOT NULL DEFAULT 0,
    ctime   datetime NOT NULL DEFAULT '2017-01-01',
    PRIMARY KEY(id),
    UNIQUE KEY(name)
) ENGINE = InnoDB;

