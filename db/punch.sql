use yunxing;

CREATE TABLE IF NOT EXISTS punch (
    id      bigint unsigned NOT NULL AUTO_INCREMENT,
    aid     int unsigned NOT NULL,
    uid     int unsigned NOT NULL,
    ctime   datetime NOT NULL DEFAULT '2017-01-01',
    PRIMARY KEY(id),
    UNIQUE KEY(aid),
    KEY(uid)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS punch_praise (
    id      bigint unsigned NOT NULL AUTO_INCREMENT,
    uid     int unsigned NOT NULL,
    aid     int unsigned NOT NULL,
    ctime   datetime NOT NULL DEFAULT '2017-01-01',
    PRIMARY KEY(id),
    UNIQUE KEY(uid, aid)
) ENGINE = InnoDB;
