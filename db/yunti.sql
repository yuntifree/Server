use yunxing;

CREATE TABLE IF NOT EXISTS user (
    uid     bigint unsigned NOT NULL AUTO_INCREMENT,
    username    varchar(32) NOT NULL,
    phone   varchar(16) NOT NULL DEFAULT '',
    password    varchar(32) NOT NULL,
    wifi_passwd varchar(32) NOT NULL,
    salt        varchar(32) NOT NULL,
    -- term 0: android 1:ios
    term    tinyint unsigned NOT NULL DEFAULT 0,
    nickname    varchar(32) NOT NULL,
    headurl     varchar(256) NOT NULL,
    version int unsigned NOT NULL DEFAULT 0,
    udid    varchar(32) NOT NULL DEFAULT '',
    model   varchar(32) NOT NULL DEFAULT '',
    channel varchar(32) NOT NULL DEFAULT '',
    reg_ip  varchar(32) NOT NULL DEFAULT '',
    deleted tinyint unsigned NOT NULL DEFAULT 0,
    token   varchar(32) NOT NULL DEFAULT '',
    private varchar(32) NOT NULL DEFAULT '',
    remark  varchar(128) NOT NULL DEFAULT '',
    aid     int unsigned NOT NULL DEFAULT 0,
    times   int unsigned NOT NULL DEFAULT 0,
    duration int unsigned NOT NULL DEFAULT 0,
    traffic bigint unsigned NOT NULL DEFAULT 0,
    -- 0x1:songshanhu  0x2: weijiju
    bitmap  bigint unsigned NOT NULL DEFAULT 0,
    ctime   datetime NOT NULL DEFAULT '2016-01-01 00:00:00',
    atime   datetime NOT NULL DEFAULT '2016-01-01 00:00:00',
    etime   datetime NOT NULL DEFAULT '2016-01-01 00:00:00',
    aptime   datetime NOT NULL DEFAULT '2016-01-01 00:00:00',
    PRIMARY KEY(uid),
    UNIQUE KEY(username),
    KEY(phone),
    KEY(aptime)
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

CREATE TABLE IF NOT EXISTS news ( 
    id      bigint unsigned NOT NULL AUTO_INCREMENT,
    title   varchar(256) NOT NULL,
    img1    varchar(256) NOT NULL DEFAULT '',
    img2    varchar(256) NOT NULL DEFAULT '',
    img3    varchar(256) NOT NULL DEFAULT '',
    vid     varchar(256) NOT NULL DEFAULT '',
    source  varchar(128) NOT NULL DEFAULT '',
    dst     varchar(256) NOT NULL,
    origin  varchar(256) NOT NULL,
    md5     varchar(32) NOT NULL,
    stype   tinyint unsigned NOT NULL DEFAULT 0,
    deleted tinyint unsigned NOT NULL DEFAULT 0,
    click   bigint unsigned NOT NULL DEFAULT 0,
    ctime   datetime NOT NULL DEFAULT '2016-01-01 00:00:00',
    dtime   datetime NOT NULL DEFAULT '2016-01-01 00:00:00',
    ruid    int unsigned NOT NULL DEFAULT 0,
    review  tinyint unsigned NOT NULL DEFAULT 0,
    rtime   datetime NOT NULL DEFAULT '2016-01-01 00:00:00',
    PRIMARY KEY(id),
    UNIQUE KEY(md5)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS ads ( 
    id      bigint unsigned NOT NULL AUTO_INCREMENT,
    title   varchar(256) NOT NULL,
    img1    varchar(256) NOT NULL DEFAULT '',
    img2    varchar(256) NOT NULL DEFAULT '',
    img3    varchar(256) NOT NULL DEFAULT '',
    vid     varchar(256) NOT NULL DEFAULT '',
    source  varchar(128) NOT NULL DEFAULT '',
    dst     varchar(256) NOT NULL,
    origin  varchar(256) NOT NULL,
    md5     varchar(32) NOT NULL,
    stype   tinyint unsigned NOT NULL DEFAULT 0,
    deleted tinyint unsigned NOT NULL DEFAULT 0,
    display bigint unsigned NOT NULL DEFAULT 0,
    click   bigint unsigned NOT NULL DEFAULT 0,
    ctime   datetime NOT NULL DEFAULT '2016-01-01 00:00:00',
    dtime   datetime NOT NULL DEFAULT '2016-01-01 00:00:00',
    ruid    int unsigned NOT NULL DEFAULT 0,
    review  tinyint unsigned NOT NULL DEFAULT 0,
    rtime   datetime NOT NULL DEFAULT '2016-01-01 00:00:00',
    PRIMARY KEY(id),
    UNIQUE KEY(md5)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS youku_video (
    vid      bigint unsigned NOT NULL AUTO_INCREMENT,
    id       varchar(32) NOT NULL,
    origin_id   varchar(32) NOT NULL,
    title       varchar(128) NOT NULL,
    img         varchar(1024) NOT NULL,
    play_url    varchar(256) NOT NULL,
    duration    int unsigned NOT NULL DEFAULT 0,
    source      varchar(128) NOT NULL,
    dst         varchar(256) NOT NULL,
    md5         varchar(32) NOT NULL,
    play        bigint unsigned NOT NULL DEFAULT 0,
    ctime       datetime NOT NULL DEFAULT '2016-01-01',
    deleted     tinyint unsigned NOT NULL DEFAULT 0,
    review      tinyint unsigned NOT NULL DEFAULT 0,
    ruid        int unsigned NOT NULL DEFAULT 0,
    rtime       datetime NOT NULL DEFAULT '2016-01-01',
    PRIMARY KEY(vid),
    UNIQUE KEY(md5)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS wx_openid (
    wid     int unsigned NOT NULL AUTO_INCREMENT,
    uid     int unsigned NOT NULL,
    wtype   tinyint unsigned NOT NULL DEFAULT 0,
    openid  varchar(32) NOT NULL,
    PRIMARY KEY(wid),
    UNIQUE KEY(uid, wtype)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS wx_token (
    tid     int unsigned NOT NULL AUTO_INCREMENT,
    appid   varchar(32) NOT NULL,
    secret  varchar(32) NOT NULL,
    access_token    varchar(128) NOT NULL,
    api_ticket      varchar(128) NOT NULL,
    expire_time     datetime NOT NULL DEFAULT '2016-01-01',
    PRIMARY KEY(tid),
    UNIQUE KEY(appid)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS service (
    sid     int unsigned NOT NULL AUTO_INCREMENT,
    category int unsigned NOT NULL,
    title   varchar(64) NOT NULL,
    icon    varchar(128) NOT NULL,
    dst     varchar(128) NOT NULL,
    deleted tinyint unsigned NOT NULL DEFAULT 0,
    ctime   datetime NOT NULL DEFAULT '2016-01-01 00:00:00',
    PRIMARY KEY(sid)
) ENGINE = InnoDB;

-- data from ZTE
CREATE TABLE IF NOT EXISTS ap (
    id      int unsigned NOT NULL AUTO_INCREMENT,
    longitude   double NOT NULL,
    latitude    double NOT NULL,
    address     varchar(256) NOT NULL,
    mac         varchar(32) NOT NULL,
    bd_lon      double NOT NULL,
    bd_lat      double NOT NULL,
    count       int unsigned NOT NULL DEFAULT 0,
    bandwidth   double NOT NULL DEFAULT 0,
    online      tinyint unsigned NOT NULL DEFAULT 0,
    mtime       datetime NOT NULL DEFAULT '2016-01-01',
    PRIMARY KEY(id)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS ap_stat (
    sid     bigint unsigned NOT NULL AUTO_INCREMENT,
    aid     int unsigned NOT NULL,
    stime   datetime NOT NULL DEFAULT '2016-01-01 00:00:00',
    etime   datetime NOT NULL DEFAULT '2016-01-01 00:00:00',
    count   int unsigned NOT NULL DEFAULT 0,
    traffic int unsigned NOT NULL DEFAULT 0,
    PRIMARY KEY(sid),
    KEY(aid),
    KEY(stime)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS user_record (
    rid     bigint unsigned NOT NULL AUTO_INCREMENT,
    uid     int unsigned NOT NULL,
    aid     int unsigned NOT NULL,
    stime   datetime NOT NULL DEFAULT '2016-01-01 00:00:00',
    etime   datetime NOT NULL DEFAULT '2016-01-01 00:00:00',
    PRIMARY KEY(rid),
    KEY(uid),
    KEY(aid)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS wifi (
    id      int unsigned NOT NULL AUTO_INCREMENT,
    uid     int unsigned NOT NULL,
    ssid    varchar(128) NOT NULL,
    username    varchar(128) NOT NULL,
    password    varchar(64) NOT NULL,
    longitude   double NOT NULL,
    latitude    double NOT NULL,
    deleted     tinyint unsigned NOT NULL DEFAULT 0,
    ctime       datetime NOT NULL DEFAULT '2016-01-01',
    PRIMARY KEY(id),
    KEY(uid)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS click_record (
    rid     bigint unsigned NOT NULL AUTO_INCREMENT,
    type    tinyint unsigned NOT NULL, 
    id      bigint unsigned NOT NULL,
    uid     int unsigned NOT NULL,
    ctime   datetime NOT NULL DEFAULT '2016-01-01',
    PRIMARY KEY(rid),
    UNIQUE KEY(type, id, uid)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS click_stat (
    id      bigint unsigned NOT NULL AUTO_INCREMENT,
    type    int unsigned NOT NULL,
    name    varchar(64) NOT NULL,
    ctime   date NOT NULL DEFAULT '2017-01-01',
    total   bigint unsigned NOT NULL DEFAULT 0,
    PRIMARY KEY(id),
    UNIQUE KEY(type, name, ctime)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS weather (
    wid     bigint unsigned NOT NULL AUTO_INCREMENT,
    temp    int unsigned NOT NULL,
    info    varchar(32) NOT NULL,
    type    tinyint unsigned NOT NULL DEFAULT 0,
    ctime   datetime NOT NULL DEFAULT '2016-01-01',
    dtime   datetime NOT NULL DEFAULT '2016-01-01',
    PRIMARY KEY(wid),
    UNIQUE KEY(ctime)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS user_unionid (
    id      int unsigned NOT NULL AUTO_INCREMENT,
    uid     int unsigned NOT NULL,
    unionid varchar(32) NOT NULL,
    ctime   datetime NOT NULL DEFAULT '2016-01-01',
    PRIMARY KEY(id),
    UNIQUE KEY(uid),
    KEY(unionid)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS banner (
    id      int unsigned NOT NULL AUTO_INCREMENT,
    -- 0:banner 1:flash_ad 2:activity
    type    tinyint unsigned NOT NULL DEFAULT 0,
    img     varchar(256) NOT NULL,
    dst     varchar(256) NOT NULL,
    title   varchar(256) NOT NULL,
    priority    int unsigned NOT NULL DEFAULT 0,
    online  tinyint unsigned NOT NULL DEFAULT 0,
    deleted tinyint unsigned NOT NULL DEFAULT 0,
    ctime   datetime NOT NULL DEFAULT '2016-01-01',
    etime   datetime NOT NULL DEFAULT '2016-01-01',
    PRIMARY KEY(id),
    KEY(priority),
    KEY(type)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS service_click(
    id      int unsigned NOT NULL AUTO_INCREMENT,
    sid     int unsigned NOT NULL,
    click   int unsigned NOT NULL DEFAULT 0,
    ctime   date NOT NULL DEFAULT '2016-01-01',
    PRIMARY KEY(id),
    UNIQUE KEY(sid, ctime)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS service_click_record (
    id      bigint unsigned NOT NULL AUTO_INCREMENT,
    sid     int unsigned NOT NULL,
    uid     int unsigned NOT NULL,
    ctime   datetime NOT NULL DEFAULT '2016-01-01',
    PRIMARY KEY(id),
    KEY(sid),
    KEY(uid)
) ENGINE = InnoDB;

-- OSS
CREATE TABLE IF NOT EXISTS back_login (
    uid     int unsigned NOT NULL AUTO_INCREMENT,
    username    varchar(32) NOT NULL,
    password    varchar(32) NOT NULL,
    salt        varchar(32) NOT NULL,
    login_time  datetime NOT NULL DEFAULT '2016-01-01',
    expire_time  datetime NOT NULL DEFAULT '2016-01-01',
    skey        varchar(32) NOT NULL,
    deleted     tinyint unsigned NOT NULL DEFAULT 0,
    ctime       datetime NOT NULL DEFAULT '2016-01-01',
    PRIMARY KEY(uid),
    UNIQUE KEY(username)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS tags (
    id      int unsigned NOT NULL AUTO_INCREMENT,
    category    int unsigned NOT NULL,
    content     varchar(128) NOT NULL,
    deleted     tinyint unsigned NOT NULL,
    ctime       datetime NOT NULL DEFAULT '2016-01-01',
    PRIMARY KEY(id),
    KEY(category)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS news_tags(
    id     bigint unsigned NOT NULL AUTO_INCREMENT,
    nid    bigint unsigned NOT NULL,
    tid    int unsigned NOT NULL,
    ruid   int unsigned NOT NULL DEFAULT 0,
    ctime  datetime NOT NULL DEFAULT '2016-01-01',
    PRIMARY KEY(id),
    KEY(nid)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS template (
    id      int unsigned NOT NULL AUTO_INCREMENT,
    title   varchar(256) NOT NULL,
    content varchar(4096) NOT NULL,
    online  tinyint unsigned NOT NULL DEFAULT 0,
    ruid     int unsigned NOT NULL,
    ctime   datetime NOT NULL DEFAULT '2016-01-01',
    mtime   datetime NOT NULL DEFAULT '2016-01-01',
    PRIMARY KEY(id)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS image
(
    id     bigint unsigned NOT NULL AUTO_INCREMENT,
    name    varchar(48) NOT NULL,
    uid     int unsigned NOT NULL,
    filesize    int unsigned NOT NULL,
    height      int unsigned NOT NULL,
    width       int unsigned NOT NULL,
    status      tinyint unsigned NOT NULL default 0,
    deleted     tinyint unsigned NOT NULL default 0,
    ctime       datetime NOT NULL default '0000-00-00 00:00:00',
    ftime       datetime NOT NULL default '0000-00-00 00:00:00',
    PRIMARY KEY(id),
    UNIQUE KEY(name)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS kv_config
(
    id      int unsigned NOT NULL AUTO_INCREMENT,
    name    varchar(128) NOT NULL,
    val     varchar(4096) NOT NULL,
    deleted tinyint unsigned NOT NULL DEFAULT 0,
    ctime   datetime NOT NULL DEFAULT '2016-01-01',
    PRIMARY KEY(id),
    UNIQUE KEY(name)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS ad_ban
(
    id      int unsigned NOT NULL AUTO_INCREMENT,
    term    tinyint unsigned NOT NULL,
    version int unsigned NOT NULL,
    deleted tinyint unsigned NOT NULL,
    ctime   datetime NOT NULL DEFAULT '2016-01-01',
    PRIMARY KEY(id),
    UNIQUE KEY(term, version)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS white_list 
(
    id      bigint unsigned NOT NULL AUTO_INCREMENT,
    -- 0: flash_ad 1:banner 2:activity 3:flash_ad dbg 4:live dbg
    type    int unsigned NOT NULL,
    uid     int unsigned NOT NULL,
    deleted tinyint unsigned NOT NULL DEFAULT 0,
    ctime   datetime NOT NULL DEFAULT '2016-01-01',
    PRIMARY KEY(id),
    UNIQUE KEY(type, uid)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS feedback
(
    id      bigint unsigned NOT NULL AUTO_INCREMENT,
    uid     int unsigned NOT NULL,
    content varchar(2048) NOT NULL,
    ctime   datetime NOT NULL DEFAULT '2016-01-01',
    PRIMARY KEY(id),
    KEY(uid)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS zte_code 
(
    id      bigint unsigned NOT NULL AUTO_INCREMENT,
    type    tinyint unsigned NOT NULL,
    phone   varchar(16) NOT NULL,
    code    varchar(16) NOT NULL,
    ctime   datetime NOT NULL DEFAULT '2017-01-01',
    mtime   datetime NOT NULL DEFAULT '2017-01-01',
    PRIMARY KEY(id),
    UNIQUE KEY(type, phone)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS menu 
(
    id      bigint unsigned NOT NULL AUTO_INCREMENT,
    type    int unsigned NOT NULL,
    ctype   int unsigned NOT NULL,
    title   varchar(64) NOT NULL,
    dst     varchar(512) NOT NULL,
    deleted tinyint unsigned NOT NULL,
    ctime   datetime NOT NULL DEFAULT '2016-01-01',
    PRIMARY KEY(id)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS joke
(
    id      bigint unsigned NOT NULL AUTO_INCREMENT,
    content varchar(2048) NOT NULL,
    md5     varchar(32) NOT NULL,
    dst     varchar(1024) NOT NULL,
    ctime   datetime NOT NULL DEFAULT '2016-01-01',
    dtime   datetime NOT NULL DEFAULT '2016-01-01',
    PRIMARY KEY(id),
    UNIQUE KEY(md5)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS token_backup
(
    id      bigint unsigned NOT NULL AUTO_INCREMENT,
    uid     int unsigned NOT NULL,
    token   varchar(32) NOT NULL,
    private varchar(32) NOT NULL,
    ctime   datetime NOT NULL DEFAULT '2017-01-01',
    etime   datetime NOT NULL DEFAULT '2017-01-01',
    PRIMARY KEY(id),
    UNIQUE KEY(uid)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS ac_info
(
    id      bigint unsigned NOT NULL AUTO_INCREMENT,
    -- 0:松山湖 1:卫计局
    type    int unsigned NOT NULL,
    name    varchar(32) NOT NULL,
    ip      varchar(32) NOT NULL,
    ctime   datetime NOT NULL DEFAULT '2016-01-01',
    PRIMARY KEY(id),
    KEY(type),
    UNIQUE KEY(name)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS notice
(
    id      bigint unsigned NOT NULL AUTO_INCREMENT,
    title   varchar(256) NOT NULL,
    content varchar(512) NOT NULL,
    dst      varchar(256) NOT NULL,
    ctime   datetime NOT NULL,
    etime   datetime NOT NULL,
    PRIMARY KEY(id)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS user_mac
(
    id      bigint unsigned NOT NULL AUTO_INCREMENT,
    mac     varchar(32) NOT NULL,
    phone   varchar(32) NOT NULL,
    ctime   datetime NOT NULL DEFAULT '2017-01-01',
    etime   datetime NOT NULL DEFAULT '2017-01-01',
    PRIMARY KEY(id),
    UNIQUE KEY(mac)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS portal_page
(
    id      bigint unsigned NOT NULL AUTO_INCREMENT,
    -- 0:login 1:portal
    type    tinyint unsigned NOT NULL DEFAULT 0,
    dir     varchar(64) NOT NULL,
    description varchar(512) NOT NULL,
    online  tinyint unsigned NOT NULL DEFAULT 0,
    ctime   datetime NOT NULL DEFAULT '2017-01-01',
    PRIMARY KEY(id)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS app_channel
(
    id      bigint unsigned NOT NULL AUTO_INCREMENT,
    channel varchar(64) NOT NULL,
    version int unsigned NOT NULL,
    vname   varchar(32) NOT NULL,
    downurl varchar(512) NOT NULL,
    ctime   datetime NOT NULL DEFAULT '2017-01-01',
    PRIMARY KEY(id),
    UNIQUE KEY(channel)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS portal_mac
(
    id      bigint unsigned NOT NULL AUTO_INCREMENT,
    mac     varchar(32) NOT NULL,
    ctime   datetime NOT NULL DEFAULT '2017-01-01',
    atime   datetime NOT NULL DEFAULT '2017-01-01',
    PRIMARY KEY(id),
    UNIQUE KEY(mac)
) ENGINE = InnoDB;

CREATE TABLE IF NOT EXISTS live 
(
    id      bigint unsigned NOT NULL AUTO_INCREMENT,
    live_id bigint unsigned NOT NULL,
    uid     int unsigned NOT NULL,
    avatar  varchar(128) NOT NULL,
    nickname    varchar(128) NOT NULL,
    img     varchar(128) NOT NULL,
    p_time  datetime NOT NULL DEFAULT '2017-01-01',
    location    varchar(64) NOT NULL,
    watches     int unsigned NOT NULL DEFAULT 0,
    live        int unsigned NOT NULL DEFAULT 0,
    ctime       datetime NOT NULL DEFAULT '2017-01-01',
    seq         bigint unsigned NOT NULL DEFAULT 0,
    PRIMARY KEY(id),
    UNIQUE KEY(live_id),
    KEY(seq)
) ENGINE = InnoDB;

-- 默认头像
CREATE TABLE IF NOT EXISTS default_head
(
    id      bigint unsigned NOT NULL AUTO_INCREMENT,
    -- sex 0:female 1:male
    sex     tinyint unsigned NOT NULL DEFAULT 0,
    headurl varchar(128) NOT NULL,
    deleted tinyint unsigned NOT NULL DEFAULT 0,
    description varchar(128) NOT NULL,
    age     varchar(128) NOT NULL,
    ctime   datetime NOT NULL DEFAULT '2017-01-01',
    PRIMARY KEY(id)
) ENGINE = InnoDB;
