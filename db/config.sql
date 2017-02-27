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

-- urban_service 58城市服务 
CREATE TABLE IF NOT EXISTS urban_service
(
    id      bigint unsigned NOT NULL AUTO_INCREMENT,
    -- type 0:android 1:ios 2:portal 3:weixin
    type    tinyint unsigned NOT NULL DEFAULT 0,
    title   varchar(64) NOT NULL,
    img     varchar(128) NOT NULL,
    dst     varchar(128) NOT NULL,
    priority    int unsigned NOT NULL DEFAULT 0,
    deleted     tinyint unsigned NOT NULL DEFAULT 0,
    click       int unsigned NOT NULL DEFAULT 0,
    ctime       datetime NOT NULL DEFAULT '2017-01-01',
    PRIMARY KEY(id),
    KEY(type)
) ENGINE = InnoDB;

-- recommend 精品推荐
CREATE TABLE IF NOT EXISTS recommend
(
    id      bigint unsigned NOT NULL AUTO_INCREMENT,
    type    tinyint unsigned NOT NULL DEFAULT 0,
    img     varchar(128) NOT NULL,
    dst     varchar(128) NOT NULL,
    click   int unsigned NOT NULL DEFAULT 0,
    priority    int unsigned NOT NULL DEFAULT 0,
    deleted     tinyint unsigned NOT NULL DEFAULT 0,
    ctime       datetime NOT NULL DEFAULT '2017-01-01',
    PRIMARY KEY(id),
    KEY(type)
) ENGINE = InnoDB;
