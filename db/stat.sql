use yunxing;

CREATE TABLE IF NOT EXISTS user_stat (
    id      bigint unsigned NOT NULL AUTO_INCREMENT,
    active  int unsigned NOT NULL DEFAULT 0,
    register    int unsigned NOT NULL DEFAULT 0,
    ctime   date NOT NULL DEFAULT '2017-01-01',
    PRIMARY KEY(id),
    UNIQUE KEY(ctime)
) ENGINE = InnoDB;
