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

CREATE TABLE IF NOT EXISTS sales (
    sid         bigint unsigned NOT NULL AUTO_INCREMENT,
    num         int unsigned NOT NULL DEFAULT 0,
    -- 0: pre-start 1:start 2:end 3:award 4:confirm 8:expire
    status      tinyint unsigned NOT NULL DEFAULT 0,
    gid         int unsigned NOT NULL DEFAULT 0,
    total       int unsigned NOT NULL DEFAULT 0,
    remain      int unsigned NOT NULL DEFAULT 0,
    win_uid     int unsigned NOT NULL DEFAULT 0,
    win_hid     bigint unsigned NOT NULL DEFAULT 0,
    win_code_1  bigint unsigned NOT NULL DEFAULT 0,
    win_code_2  bigint unsigned NOT NULL DEFAULT 0,
    cqssc       varchar(16) NOT NULL,
    win_code    int unsigned NOT NULL DEFAULT 0,
    ctime       datetime NOT NULL DEFAULT '2016-01-01',
    etime       datetime NOT NULL DEFAULT '2016-01-01',
    atime       datetime NOT NULL DEFAULT '2016-01-01',
    PRIMARY KEY(sid),
    UNIQUE KEY(gid, num)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS sales_history (
    hid         bigint unsigned NOT NULL AUTO_INCREMENT,
    sid         bigint unsigned NOT NULL,
    num         int unsigned NOT NULL,
    uid         int unsigned NOT NULL,
    ctime       datetime NOT NULL DEFAULT '2016-01-01',
    PRIMARY KEY(hid),
    UNIQUE KEY(sid, num)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS goods (
    gid         int unsigned NOT NULL AUTO_INCREMENT,
    name        varchar(128) NOT NULL,
    image       varchar(128) NOT NULL,
    title       varchar(256) NOT NULL,
    sub_title   varchar(256) NOT NULL,
    price       int unsigned NOT NULL DEFAULT 0,
    online      tinyint unsigned NOT NULL DEFAULT 0,
    image_num   tinyint unsigned NOT NULL DEFAULT 0,
    hot_flag    tinyint unsigned NOT NULL DEFAULT 0,
    deleted     tinyint unsigned NOT NULL DEFAULT 0,
    -- 0:实物 1:充值卡
    type        tinyint unsigned NOT NULL DEFAULT 0,
    priority     int unsigned NOT NULL DEFAULT 0,
    new_rank    int unsigned NOT NULL DEFAULT 0,
    ctime       datetime NOT NULL DEFAULT '2016-01-01',
    PRIMARY KEY(gid)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS logistics (
    lid         bigint unsigned NOT NULL AUTO_INCREMENT,
    sid         bigint unsigned NOT NULL,
    -- 4:confirm 5:express 6:receipt 7:fini
    status      tinyint unsigned NOT NULL DEFAULT 0,
    uid         int unsigned NOT NULL,
    aid         int unsigned NOT NULL,
    share       tinyint unsigned NOT NULL DEFAULT 0,
    express     int unsigned NOT NULL DEFAULT 0,
    account     varchar(16) NOT NULL,
    award_account   varchar(16) NOT NULL,
    trac_num    varchar(16) NOT NULL,
    ctime       datetime NOT NULL DEFAULT '2016-01-01',
    etime       datetime NOT NULL DEFAULT '2016-01-01',
    rtime       datetime NOT NULL DEFAULT '2016-01-01',
    ftime       datetime NOT NULL DEFAULT '2016-01-01',
    PRIMARY KEY(lid),
    UNIQUE KEY(sid)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS promotion (
    id          int unsigned NOT NULL AUTO_INCREMENT,
    title       varchar(256) NOT NULL,
    target      varchar(256) NOT NULL,
    online      tinyint unsigned NOT NULL DEFAULT 0,
    deleted     tinyint unsigned NOT NULL DEFAULT 0,
    ctime       datetime NOT NULL DEFAULT '2016-01-01',
    PRIMARY KEY(id)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS slides (
    id          int unsigned NOT NULL AUTO_INCREMENT,
    image       varchar(256) NOT NULL,
    target      varchar(256) NOT NULL,
    online      tinyint unsigned NOT NULL DEFAULT 0,
    deleted     tinyint unsigned NOT NULL DEFAULT 0,
    ctime       datetime NOT NULL DEFAULT '2016-01-01',
    PRIMARY KEY(id)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS share_image (
    id          bigint unsigned NOT NULL AUTO_INCREMENT,
    hid         bigint unsigned NOT NULL,
    sid         bigint unsigned NOT NULL,
    url         varchar(128) NOT NULL,
    review      tinyint unsigned NOT NULL,
    deleted     tinyint unsigned NOT NULL,
    ruid        int unsigned NOT NULL DEFAULT 0,
    ctime       datetime NOT NULL DEFAULT '2016-01-01',
    rtime       datetime NOT NULL DEFAULT '2016-01-01',
    PRIMARY KEY(id),
    KEY(sid),
    KEY(hid)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS share_history (
    hid         bigint unsigned NOT NULL AUTO_INCREMENT,
    sid         bigint unsigned NOT NULL,
    uid         int unsigned NOT NULL,
    gid         int unsigned NOT NULL,
    title       varchar(256) NOT NULL,
    content     varchar(1024) NOT NULL,
    image_num   tinyint unsigned NOT NULL DEFAULT 0,
    ctime       datetime NOT NULL DEFAULT '2016-01-01',
    ruid        int unsigned NOT NULL DEFAULT 0,
    review_time       datetime NOT NULL DEFAULT '2016-01-01',
    reviewed    tinyint unsigned NOT NULL DEFAULT 0,
    deleted     tinyint unsigned NOT NULL DEFAULT 0,
    award       int unsigned NOT NULL DEFAULT 0,
    top_flag    tinyint unsigned NOT NULL DEFAULT 0,
    PRIMARY KEY(hid),
    KEY(sid)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS purchase_history (
    hid         bigint unsigned NOT NULL AUTO_INCREMENT,
    uid         int unsigned NOT NULL,
    sid         bigint unsigned NOT NULL,
    num         int unsigned NOT NULL DEFAULT 1,
    ctime       bigint unsigned NOT NULL DEFAULT 0,
    ack_flag    tinyint unsigned NOT NULL DEFAULT 0,
    PRIMARY KEY(hid),
    KEY(uid, sid)
) ENGINE = InnoDB;

