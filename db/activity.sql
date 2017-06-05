use yunxing;

CREATE TABLE IF NOT EXISTS reserve_info 
(
    id      int unsigned NOT NULL AUTO_INCREMENT,
    name    varchar(64) NOT NULL,
    phone   varchar(16) NOT NULL,
    sid     int unsigned NOT NULL,
    reserve_date varchar(128) NOT NULL,
    btype   tinyint unsigned NOT NULL,
    pillow  tinyint unsigned NOT NULL,
    code    int unsigned NOT NULL,
    donate  int unsigned NOT NULL DEFAULT 0,
    sms     tinyint unsigned NOT NULL DEFAULT 0,
    ctime   datetime NOT NULL DEFAULT '2017-01-01',
    dtime   datetime NOT NULL DEFAULT '2017-01-01',
    PRIMARY KEY(id),
    UNIQUE KEY(phone),
    UNIQUE KEY(code),
    KEY(donate)
) ENGINE = InnoDB;
