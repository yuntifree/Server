use inquiry;

CREATE TABLE IF NOT EXISTS users (
    uid     bigint unsigned NOT NULL AUTO_INCREMENT,
    username    varchar(32) NOT NULL,
    phone   varchar(16) NOT NULL DEFAULT '',
    token   varchar(32) NOT NULL DEFAULT '',
    -- nickname 微信昵称
    nickname    varchar(128) NOT NULL DEFAULT '',
    headurl     varchar(256) NOT NULL DEFAULT '',
    gender      tinyint unsigned NOT NULL DEFAULT 0,
    role        tinyint unsigned NOT NULL DEFAULT 0,
    doctor         int unsigned NOT NULL DEFAULT 0,
    hasrelation     tinyint unsigned NOT NULL DEFAULT 0,
    balance     int unsigned NOT NULL DEFAULT 0,
    totalfee    int unsigned NOT NULL DEFAULT 0,
    draw        int unsigned NOT NULL DEFAULT 0,
    draw_pass   varchar(32) NOT NULL DEFAULT '',
    draw_salt   varchar(32) NOT NULL DEFAULT '',
    totaldraw   int unsigned NOT NULL DEFAULT 0,
    deleted tinyint unsigned NOT NULL DEFAULT 0,
    ctime   datetime NOT NULL DEFAULT '2017-01-01',
    PRIMARY KEY(uid),
    UNIQUE KEY(username),
    KEY(phone)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS wx_openid (
    id      int unsigned NOT NULL AUTO_INCREMENT,
    unionid varchar(36) NOT NULL,
    openid  varchar(32) NOT NULL,
    skey varchar(64) NOT NULL,
    sid     varchar(32) NOT NULL,
    ctime   datetime NOT NULL DEFAULT '2017-01-01',
    PRIMARY KEY(id),
    UNIQUE KEY(openid),
    KEY(unionid),
    KEY(sid)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS wx_token (
    id      int unsigned NOT NULL AUTO_INCREMENT,
    appid   varchar(32) NOT NULL,
    secret  varchar(32) NOT NULL,
    access_token varchar(256) NOT NULL,
    expire_time datetime NOT NULL DEFAULT '2017-01-01',
    PRIMARY KEY(id),
    UNIQUE KEY(appid)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS doctor (
    id      int unsigned NOT NULL AUTO_INCREMENT,
    phone   varchar(16) NOT NULL,
    name    varchar(64) NOT NULL,
    title   varchar(128) NOT NULL,
    hospital varchar(256) NOT NULL,
    department varchar(128) NOT NULL,
    headurl    varchar(128) NOT NULL,
    fee     int unsigned NOT NULL DEFAULT 0,
    deleted tinyint unsigned NOT NULL DEFAULT 0,
    ctime   datetime NOT NULL DEFAULT '2017-01-01',
    PRIMARY KEY(id),
    UNIQUE KEY(phone)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS patient (
    id      bigint unsigned NOT NULL AUTO_INCREMENT,
    uid     int unsigned NOT NULL,
    phone   varchar(16) NOT NULL,
    -- name 真实姓名
    name        varchar(64) NOT NULL DEFAULT '',
    -- mcard 医疗卡号
    mcard       varchar(32) NOT NULL DEFAULT '',
    deleted tinyint unsigned NOT NULL DEFAULT 0,
    ctime   datetime NOT NULL DEFAULT '2017-01-01',
    PRIMARY KEY(id),
    KEY(uid)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS phone_code (
    pid     bigint unsigned NOT NULL AUTO_INCREMENT,
    phone   varchar(16) NOT NULL DEFAULT '',
    uid     int unsigned NOT NULL DEFAULT 0,
    code    int unsigned NOT NULL DEFAULT 0,
    ctime   datetime NOT NULL DEFAULT '2016-01-01 00:00:00',
    stime   datetime NOT NULL DEFAULT '2016-01-01 00:00:00',
    etime   datetime NOT NULL DEFAULT '2016-01-01 00:00:00',
    used    tinyint unsigned NOT NULL DEFAULT 0,
    PRIMARY KEY(pid),
    KEY(phone, uid)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS relations (
    id      bigint unsigned NOT NULL AUTO_INCREMENT,
    doctor  int unsigned NOT NULL,
    patient int unsigned NOT NULL,
    -- status 0-未问诊 1-问诊中 2-问诊结束
    status  tinyint unsigned NOT NULL DEFAULT 0,
    deleted tinyint unsigned NOT NULL DEFAULT 0,
    -- flag 0-没有问诊记录 1-有过问诊记录
    flag     tinyint unsigned NOT NULL DEFAULT 0,
    ctime datetime NOT NULL DEFAULT '2017-01-01',
    PRIMARY KEY(id),
    UNIQUE KEY(doctor, patient)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS inquiry_history (
    id      bigint unsigned NOT NULL AUTO_INCREMENT,
    doctor  int unsigned NOT NULL,
    patient int unsigned NOT NULL,
    pid     int unsigned NOT NULL,
    fee     int unsigned NOT NULL DEFAULT 0,
    doctor_fee int unsigned NOT NULL DEFAULT 0,
    form_id varchar(128) NOT NULL DEFAULT '',
    -- status 0-待支付 1-支付成功(问诊中) 2-问诊结束 3-申请退款 
    -- 4-退款成功
    status  tinyint unsigned NOT NULL DEFAULT 0,
    deleted tinyint unsigned NOT NULL DEFAULT 0,
    -- interval 医生第一条回复时间间隔
    interval int unsigned NOT NULL DEFAULT 0,
    ctime datetime NOT NULL DEFAULT '2017-01-01',
    etime datetime NOT NULL DEFAULT '2017-01-01',
    PRIMARY KEY(id),
    KEY(doctor, patient)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS orders (
    id      bigint unsigned NOT NULL AUTO_INCREMENT,
    oid     varchar(64) NOT NULL,
    uid     int unsigned NOT NULL,
    tuid    int unsigned NOT NULL,
    -- type 0-问诊
    type    tinyint unsigned NOT NULL DEFAULT 0,
    -- item 对应type的id
    item    int unsigned NOT NULL DEFAULT 0,
    price    int unsigned NOT NULL DEFAULT 0,
    fee    int unsigned NOT NULL DEFAULT 0,
    prepayid varchar(64) NOT NULL DEFAULT '',
    ctime datetime NOT NULL DEFAULT '2017-01-01',
    ftime datetime NOT NULL DEFAULT '2017-01-01',
    -- status 0-未支付 1-支付成功
    status  tinyint unsigned NOT NULL DEFAULT 0,
    PRIMARY KEY(id),
    UNIQUE KEY(oid)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS chat (
    id      bigint unsigned NOT NULL AUTO_INCREMENT,
    uid     int unsigned NOT NULL,
    tuid    int unsigned NOT NULL,
    -- type 0-text 1-image
    type    tinyint unsigned NOT NULL DEFAULT 0,
    content varchar(512) NOT NULL,
    ack     tinyint unsigned NOT NULL DEFAULT 0,
    ctime   datetime NOT NULL DEFAULT '2017-01-01',
    acktime   datetime NOT NULL DEFAULT '2017-01-01',
    -- hid inquiry_history id
    hid     int unsigned NOT NULL DEFAULT 0,
    PRIMARY KEY(id),
    KEY(uid),
    KEY(tuid),
    KEY(ctime)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS draw_history (
    id      bigint unsigned NOT NULL AUTO_INCREMENT,
    uid     int unsigned NOT NULL,
    fee     int unsigned NOT NULL,
    -- status 0-申请 1-审核通过 2-转账成功 3-审核失败
    -- 4-转账失败
    status  tinyint unsigned NOT NULL,
    ctime   datetime NOT NULL DEFAULT '2017-01-01',
    rtime   datetime NOT NULL DEFAULT '2017-01-01',
    ptime   datetime NOT NULL DEFAULT '2017-01-01',
    PRIMARY KEY(id),
    KEY(uid)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS qrcode (
    id      bigint unsigned NOT NULL AUTO_INCREMENT,
    path    varchar(128) NOT NULL,
    width   int unsigned NOT NULL,
    img     varchar(128) NOT NULL,
    ctime   datetime NOT NULL DEFAULT '2017-01-01',
    PRIMARY KEY(id),
    UNIQUE KEY(path, width)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS feedback (
    id      bigint unsigned NOT NULL AUTO_INCREMENT,
    uid     int unsigned NOT NULL,
    content varchar(1024) NOT NULL,
    ctime   datetime NOT NULL DEFAULT '2017-01-01',
    PRIMARY KEY(id),
    KEY(uid)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS refund_history (
    id      bigint unsigned NOT NULL AUTO_INCREMENT,
    -- hid inquiry_history id
    hid     bigint unsigned NOT NULL,
    -- interval 和医生上一次回复的时间间隔(单位s)
    interval    int unsigned NOT NULL DEFAULT 0,
    ctime   datetime NOT NULL DEFAULT '2017-01-01',
    PRIMARY KEY(id),
    KEY(hid)
) ENGINE = InnoDB;
