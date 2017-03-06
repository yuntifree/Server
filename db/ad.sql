use yunxing;

CREATE TABLE IF NOT EXISTS customers
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
