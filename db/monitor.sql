use monitor;
CREATE TABLE IF NOT EXISTS api_stat
(
    id      bigint unsigned NOT NULL AUTO_INCREMENT,
    name    varchar(32) NOT NULL,
    req     int unsigned NOT NULL DEFAULT 0,
    succrsp   int unsigned NOT NULL DEFAULT 0,
    ctime   datetime NOT NULL DEFAULT '2017-01-01',
    PRIMARY KEY(id),
    UNIQUE KEY(name, ctime)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS rpc_stat
(
    id      bigint unsigned NOT NULL AUTO_INCREMENT,
    service varchar(32) NOT NULL,
    method  varchar(32) NOT NULL,
    req     int unsigned NOT NULL DEFAULT 0,
    succrsp int unsigned NOT NULL DEFAULT 0,
    ctime   datetime NOT NULL DEFAULT '2017-01-01',
    PRIMARY KEY(id),
    UNIQUE KEY(service, method, ctime)
) ENGINE = InnoDB;
