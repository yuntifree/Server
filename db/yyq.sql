use yunxing;

CREATE TABLE IF NOT EXISTS zipcode (
    cid     int unsigned NOT NULL AUTO_INCREMENT,
    code    int unsigned NOT NULL,
    address     varchar(128) NOT NULL,
    PRIMARY KEY(cid),
    UNIQUE KEY(code)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS address (
    aid     int unsigned NOT NULL AUTO_INCREMENT,
    uid     int unsigned NOT NULL,
    consignee   varchar(32) NOT NULL,
    phone       varchar(16) NOT NULL,
    province    int unsigned NOT NULL DEFAULT 0,
    city        int unsigned NOT NULL DEFAULT 0,
    district    int unsigned NOT NULL DEFAULT 0,
    addr        varchar(128) NOT NULL DEFAULT '',
    detail      varchar(512) NOT NULL DEFAULT '',
    flag        tinyint unsigned NOT NULL DEFAULT 0,
    zip         int unsigned NOT NULL DEFAULT 0,
    deleted     tinyint unsigned NOT NULL DEFAULT 0,
    PRIMARY KEY(aid),
    KEY(uid)
) ENGINE = InnoDB;
