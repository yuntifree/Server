use yunxing;

CREATE TABLE IF NOT EXISTS score_item
(
    id  int unsigned NOT NULL AUTO_INCREMENT,
    -- type 0-限量
    type int unsigned NOT NULL DEFAULT 0,
    score int unsigned NOT NULL,
    title varchar(256) NOT NULL DEFAULT '',
    img   varchar(128) NOT NULL DEFAULT '',
    online tinyint unsigned NOT NULL DEFAULT 0,
    deleted tinyint unsigned NOT NULL DEFAULT 0,
    ctime datetime NOT NULL DEFAULT '2017-01-01',
    PRIMARY KEY(id),
    KEY(type)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS user_score_item
(
    id  bigint unsigned NOT NULL AUTO_INCREMENT,
    uid int unsigned NOT NULL,
    item    int unsigned NOT NULL,
    total   int unsigned NOT NULL DEFAULT 0,
    used    int unsigned NOT NULL DEFAULT 0,
    -- status 1-已购买 2-已使用
    status  tinyint unsigned NOT NULL DEFAULT 0,
    ctime datetime NOT NULL DEFAULT '2017-01-01',
    PRIMARY KEY(id),
    UNIQUE KEY(uid, item)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS signin_history
(
    id  bigint unsigned NOT NULL AUTO_INCREMENT,
    uid int unsigned NOT NULL,
    ctime date NOT NULL DEFAULT '2017-01-01',
    PRIMARY KEY(id),
    UNIQUE KEY(uid, ctime)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS exchange_history
(
    id  bigint unsigned NOT NULL AUTO_INCREMENT,
    uid int unsigned NOT NULL,
    item int unsigned NOT NULL,
    num int unsigned NOT NULL DEFAULT 1,
    score int unsigned NOT NULL DEFAULT 0,
    ctime datetime NOT NULL DEFAULT '2017-01-01',
    PRIMARY KEY(id),
    KEY(uid, item),
    KEY(ctime)
) ENGINE = InnoDB;
