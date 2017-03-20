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

-- online_service 上网服务
CREATE TABLE IF NOT EXISTS online_service
(
    id      bigint unsigned NOT NULL AUTO_INCREMENT,
    img     varchar(128) NOT NULL,
    dst     varchar(128) NOT NULL,
    click   int unsigned NOT NULL DEFAULT 0,
    priority    int unsigned NOT NULL DEFAULT 0,
    title       varchar(64) NOT NULL,
    deleted     tinyint unsigned NOT NULL DEFAULT 0,
    ctime       datetime NOT NULL DEFAULT '2017-01-01',
    PRIMARY KEY(id)
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

-- hospital 医院基本信息
CREATE TABLE IF NOT EXISTS hospital
(
    id      bigint unsigned NOT NULL AUTO_INCREMENT,
    title   varchar(256) NOT NULL,
    img     varchar(128) NOT NULL,
    ctime       datetime NOT NULL DEFAULT '2017-01-01',
    deleted     tinyint unsigned NOT NULL DEFAULT 0,
    PRIMARY KEY(id)
) ENGINE = InnoDB;

-- hospital_info 医院信息 
CREATE TABLE IF NOT EXISTS hospital_info
(
    id      bigint unsigned NOT NULL AUTO_INCREMENT,
    -- hid 医院id
    hid     bigint unsigned NOT NULL,
    -- type 0:医院介绍 1:患者服务
    type    int unsigned NOT NULL NULL DEFAULT 0,
    img     varchar(128) NOT NULL,
    dst     varchar(128) NOT NULL,
    title   varchar(128) NOT NULL,
    click   int unsigned NOT NULL DEFAULT 0,
    priority    int unsigned NOT NULL,
    deleted     tinyint unsigned NOT NULL DEFAULT 0,
    ctime       datetime NOT NULL DEFAULT '2017-01-01',
    PRIMARY KEY(id),
    KEY(hid)
) ENGINE = InnoDB;

CREATE TABLE education_video
(
    id      bigint unsigned NOT NULL AUTO_INCREMENT,
    title   varchar(256) NOT NULL,
    dst     varchar(128) NOT NULL,
    click   int unsigned NOT NULL DEFAULT 0,
    priority    int unsigned NOT NULL,
    deleted     tinyint unsigned NOT NULL DEFAULT 0,
    ctime       datetime NOT NULL DEFAULT '2017-01-01',
    PRIMARY KEY(id)
) ENGINE = InnoDB;
