use yunxing;

-- portal_menu portal页面菜单
CREATE TABLE IF NOT EXISTS portal_menu
(
    id      bigint unsigned NOT NULL AUTO_INCREMENT,
    -- type 0:menu 1:tab
    type    tinyint unsigned NOT NULL,
    name    varchar(32) NOT NULL,
    text    varchar(32) NOT NULL,
    icon    varchar(128) NOT NULL,
    routername varchar(32) NOT NULL,
    url     varchar(128) NOT NULL,
    priority    int unsigned NOT NULL DEFAULT 0,
    dbg     tinyint unsigned NOT NULL DEFAULT 0,
    ctime   datetime NOT NULL DEFAULT '2017-01-01',
    PRIMARY KEY(id)
) ENGINE = InnoDB;
