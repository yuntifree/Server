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
    tsid    int unsigned NOT NULL,
    img     varchar(128) NOT NULL,
    abstract varchar(256) NOT NULL,
    content  varchar(512) NOT NULL,
    deleted tinyint unsigned NOT NULL DEFAULT 0,
    ctime   datetime NOT NULL DEFAULT '2017-01-01',
    PRIMARY KEY(id)
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
    UNIQUE KEY(name)
) ENGINE = InnoDB;
