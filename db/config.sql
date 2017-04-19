use yunxing;

-- portal_menu portal页面菜单
CREATE TABLE IF NOT EXISTS portal_menu
(
    id      bigint unsigned NOT NULL AUTO_INCREMENT,
    -- type 0:menu 1:tab 2:menu v20170411 3:tab v20170411
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

-- hospital_department_category 医院科室类别
CREATE TABLE IF NOT EXISTS hospital_department_category
(
    id      bigint unsigned NOT NULL AUTO_INCREMENT,
    name    varchar(128) NOT NULL,
    ctime       datetime NOT NULL DEFAULT '2017-01-01',
    PRIMARY KEY(id)
) ENGINE = InnoDB;

-- hospital_department_info 医院科室信息
CREATE TABLE IF NOT EXISTS hospital_department_info
(
    id      bigint unsigned NOT NULL AUTO_INCREMENT,
    -- hid 医院id
    hid     int unsigned NOT NULL, 
    -- cid 科室类别
    cid     int unsigned NOT NULL,
    name    varchar(128) NOT NULL,
    click   int unsigned NOT NULL DEFAULT 0,
    detail  varchar(4096) NOT NULL,
    stime   date NOT NULL,
    ctime   datetime NOT NULL,
    PRIMARY KEY(id),
    KEY(hid, cid)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS education_video
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

CREATE TABLE IF NOT EXISTS custom_portal
(
    id      bigint unsigned NOT NULL AUTO_INCREMENT,
    -- unid ap unit
    unid    int unsigned NOT NULL,
    -- type 0:医院 
    type    int unsigned NOT NULL DEFAULT 0,
    tid     int unsigned NOT NULL DEFAULT 0,
    deleted     tinyint unsigned NOT NULL DEFAULT 0,
    ctime       datetime NOT NULL DEFAULT '2017-01-01',
    PRIMARY kEY(id)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS wx_mp_info
(
    id      bigint unsigned NOT NULL AUTO_INCREMENT,
    -- type 0: local 1: hot
    type    int unsigned NOT NULL,
    -- subtype  0-东莞 1-生活 2-媒体 3-娱乐 4-美食 5-教育 
    -- 6-科技 7-金融 8-电影 9-音乐 10-汽车 11-读书
    subtype int unsigned NOT NULL DEFAULT 0,
    wxid    varchar(64) NOT NULL,
    name    varchar(256) NOT NULL,
    abstract    varchar(1024) NOT NULL,
    icon    varchar(256) NOT NULL,
    dst     varchar(256) NOT NULL,
    deleted tinyint unsigned NOT NULL DEFAULT 0,
    ctime       datetime NOT NULL DEFAULT '2017-01-01',
    PRIMARY kEY(id),
    UNIQUE KEY(wxid),
    KEY(type)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS wx_mp_article
(
    id      bigint unsigned NOT NULL AUTO_INCREMENT,
    wid     int unsigned NOT NULL,
    -- type 对应wx_mp_info中subtype字段 
    type    int unsigned NOT NULL,
    title   varchar(256) NOT NULL,
    img     varchar(256) NOT NULL,
    dst     varchar(256) NOT NULL,
    deleted tinyint unsigned NOT NULL DEFAULT 0,
    ctime       datetime NOT NULL DEFAULT '2017-01-01',
    PRIMARY kEY(id),
    KEY(wid)
) ENGINE = InnoDB;
