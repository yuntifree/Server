use yunxing;

CREATE TABLE IF NOT EXISTS user_stat (
    id      bigint unsigned NOT NULL AUTO_INCREMENT,
    active  int unsigned NOT NULL DEFAULT 0,
    register    int unsigned NOT NULL DEFAULT 0,
    ctime   date NOT NULL DEFAULT '2017-01-01',
    PRIMARY KEY(id),
    UNIQUE KEY(ctime)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS online_stat (
    id      bigint unsigned NOT NULL AUTO_INCREMENT,
    unid    int unsigned NOT NULL,
    cnt     int unsigned NOT NULL,
    ctime   date NOT NULL DEFAULT '2017-01-01',
    PRIMARY KEY(id),
    UNIQUE KEY(unid, ctime)
) ENGINE = InnoDB;
