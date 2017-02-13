use yunxing;

CREATE TABLE IF NOT EXISTS punch (
    id      bigint unsigned NOT NULL AUTO_INCREMENT,
    aid     int unsigned NOT NULL,
    uid     int unsigned NOT NULL,
    ctime   datetime NOT NULL DEFAULT '2017-01-01',
    PRIMARY KEY(id),
    UNIQUE KEY(aid, uid),
    KEY(uid)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS ap_error (
    id      bigint unsigned NOT NULL AUTO_INCREMENT,
    aid     int unsigned NOT NULL,
    uid     int unsigned NOT NULL,
    type    tinyint unsigned NOT NULL,
    ctime   datetime NOT NULL DEFAULT '2017-01-01',
    PRIMARY KEY(id),
    UNIQUE KEY(aid, uid)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS xcx_openid (
    id     int unsigned NOT NULL AUTO_INCREMENT,
    openid  varchar(32) NOT NULL,
    skey    varchar(32) NOT NULL,
    unionid varchar(36) NOT NULL,
    sid     varchar(32) NOT NULL,
    ctime   datetime NOT NULL DEFAULT '2017-01-01',
    PRIMARY KEY(id),
    UNIQUE KEY(openid),
    KEY(sid),
    KEY(unionid)
) ENGINE = InnoDB;
