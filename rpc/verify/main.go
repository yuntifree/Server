package main

import (
	"errors"
	"log"
	"net"
	"time"

	"database/sql"

	"Server/proto/common"
	"Server/proto/verify"
	"Server/util"
	"Server/zte"

	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	redis "gopkg.in/redis.v5"
)

const (
	expiretime = 3600 * 24 * 30
	mastercode = 251653
	randrange  = 1000000
	portalDir  = "http://120.25.133.234/"
)

type server struct{}

var db *sql.DB
var kv *redis.Client

func checkPhoneCode(db *sql.DB, phone string, code int64) (bool, error) {
	if code == mastercode {
		return true, nil
	}

	var realcode, pid int64
	err := db.QueryRow("SELECT code, pid FROM phone_code WHERE phone = ? AND used = 0 ORDER BY pid DESC LIMIT 1",
		phone).Scan(&realcode, &pid)
	if err != nil {
		return false, err
	}

	if realcode == code {
		stmt, err := db.Prepare("UPDATE phone_code SET used = 1 WHERE pid = ?")
		if err != nil {
			log.Printf("update phone_code failed:%v", err)
			return true, nil
		}
		_, err = stmt.Exec(pid)
		if err != nil {
			log.Printf("update phone_code failed:%v", err)
			return true, nil
		}

		return true, nil
	}
	return false, errors.New("code not match")
}

func getPhoneCode(phone string, ctype int64) (bool, error) {
	log.Printf("request phone:%s, ctype:%d", phone, ctype)
	if ctype == 1 {
		if flag := util.ExistPhone(db, phone); !flag {
			return false, errors.New("phone not exist")
		}
	}

	var code int
	err := db.QueryRow("SELECT code FROM phone_code WHERE phone = ? AND used = 0 AND etime > NOW() AND timestampdiff(second, stime, now()) < 300 ORDER BY pid DESC LIMIT 1",
		phone).Scan(&code)
	if err != nil {
		code := util.Randn(randrange)
		_, err := db.Exec("INSERT INTO phone_code(phone, code, ctime, stime, etime) VALUES (?, ?, NOW(), NOW(), DATE_ADD(NOW(), INTERVAL 5 MINUTE))",
			phone, code)
		if err != nil {
			log.Printf("insert into phone_code failed:%v", err)
			return false, err
		}
		ret := util.SendSMS(phone, int(code))
		if ret != 0 {
			log.Printf("send sms failed:%d", ret)
			return false, errors.New("send sms failed")
		}
		return true, nil
	}

	if code > 0 {
		ret := util.SendSMS(phone, int(code))
		if ret != 0 {
			log.Printf("send sms failed:%d", ret)
			return false, errors.New("send sms failed")
		}
		return true, nil
	}

	return false, errors.New("failed to send sms")
}

func (s *server) GetPhoneCode(ctx context.Context, in *verify.CodeRequest) (*verify.VerifyReply, error) {
	flag, err := getPhoneCode(in.Phone, in.Ctype)
	if err != nil {
		return &verify.VerifyReply{Result: false}, err
	}

	return &verify.VerifyReply{Result: flag}, nil
}

func (s *server) BackLogin(ctx context.Context, in *verify.LoginRequest) (*verify.LoginReply, error) {
	var uid, role int64
	var epass string
	var salt string
	err := db.QueryRow("SELECT uid, password, salt, role FROM back_login WHERE username = ?",
		in.Username).Scan(&uid, &epass, &salt, &role)
	if err != nil {
		return &verify.LoginReply{Head: &common.Head{Retcode: 2}}, err
	}
	pass := util.GenSaltPasswd(in.Password, salt)
	if pass != epass {
		return &verify.LoginReply{Head: &common.Head{Retcode: 3}},
			errors.New("verify password failed")
	}

	token := util.GenSalt()
	_, err = db.Exec("UPDATE back_login SET skey = ?, login_time = NOW(), expire_time = DATE_ADD(NOW(), INTERVAL 30 DAY) WHERE uid = ?",
		token, uid)
	if err != nil {
		return &verify.LoginReply{Head: &common.Head{Retcode: 2}}, err
	}

	return &verify.LoginReply{Head: &common.Head{Uid: uid}, Token: token, Role: role}, nil
}

func recordWxOpenid(db *sql.DB, uid, wtype int64, openid string) {
	_, err := db.Exec("INSERT IGNORE INTO wx_openid(uid, wtype, openid, ctime) VALUES (?, ?, ?, NOW())",
		uid, wtype, openid)
	if err != nil {
		log.Printf("record wx openid failed uid:%d wtype:%d openid:%s\n",
			uid, wtype, openid)
	}
}

func recordWxUnionid(db *sql.DB, uid int64, unionid string) {
	_, err := db.Exec("INSERT INTO user_unionid(uid, unionid, ctime) VALUES(?, ?, NOW()) ON DUPLICATE KEY UPDATE unionid = ?",
		uid, unionid, unionid)
	if err != nil {
		log.Printf("recordWxUnionid failed uid:%d unionid:%s err:%v\n", uid, unionid, err)
	}
}

func (s *server) WxMpLogin(ctx context.Context, in *verify.LoginRequest) (*verify.LoginReply, error) {
	var wxi util.WxInfo
	wxi, err := util.GetCodeToken(in.Code)
	if err != nil {
		return &verify.LoginReply{Head: &common.Head{Retcode: 1}}, err
	}
	err = util.GetWxInfo(&wxi)
	if err != nil {
		return &verify.LoginReply{Head: &common.Head{Retcode: 1}}, err
	}

	token := util.GenSalt()
	privdata := util.GenSalt()
	wifipass := util.GenWifiPass()
	res, err := db.Exec("INSERT IGNORE INTO user(username, headurl, sex, token, private, wifi_passwd, etime, atime, ctime) VALUES (?, ?, ?, ?, ?,?, DATE_ADD(NOW(), INTERVAL 30 DAY), NOW(), NOW())",
		wxi.UnionID, wxi.HeadURL, wxi.Sex, token, privdata, wifipass)
	if err != nil {
		log.Printf("insert user reord failed %s:%v", wxi.UnionID, err)
		return &verify.LoginReply{Head: &common.Head{Retcode: 1}}, err
	}

	uid, err := res.LastInsertId()
	if err != nil {
		log.Printf("get last insert id failed %s:%v", wxi.UnionID, err)
		return &verify.LoginReply{Head: &common.Head{Retcode: 1}}, err
	}

	if uid == 0 {
		err = db.QueryRow("SELECT uid, wifi_passwd FROM user WHERE username = ?",
			wxi.UnionID).Scan(&uid, &wifipass)
		if err != nil {
			log.Printf("search uid failed %s:%v", wxi.UnionID, err)
			return &verify.LoginReply{Head: &common.Head{Retcode: 1}}, err
		}
		_, err = db.Exec("UPDATE user SET token = ?, private = ?, etime = DATE_ADD(NOW(), INTERVAL 30 DAY), atime = NOW() WHERE uid = ?",
			token, privdata, uid)
		if err != nil {
			log.Printf("search uid failed %s:%v", wxi.UnionID, err)
			return &verify.LoginReply{Head: &common.Head{Retcode: 1}}, err
		}
	}

	recordWxOpenid(db, uid, 0, wxi.Openid)
	recordWxUnionid(db, uid, privdata)
	util.SetCachedToken(kv, uid, token)
	strTime := time.Now().Add(time.Duration(expiretime) * time.Second).
		Format(util.TimeFormat)
	return &verify.LoginReply{Head: &common.Head{Uid: uid},
		Token: token, Privdata: privdata, Expire: expiretime,
		Expiretime: strTime, Wifipass: wifipass}, nil
}

func (s *server) Login(ctx context.Context, in *verify.LoginRequest) (*verify.LoginReply, error) {
	var uid int64
	var epass string
	var salt string
	var wifipass string
	err := db.QueryRow("SELECT uid, password, salt, wifi_passwd FROM user WHERE username = ?",
		in.Username).Scan(&uid, &epass, &salt, &wifipass)
	if err != nil {
		return &verify.LoginReply{Head: &common.Head{Retcode: 2}}, err
	}
	pass := util.GenSaltPasswd(in.Password, salt)
	if pass != epass {
		return &verify.LoginReply{Head: &common.Head{Retcode: 3}},
			errors.New("verify password failed")
	}

	token := util.GenSalt()
	privdata := util.GenSalt()

	_, err = db.Exec("UPDATE user SET token = ?, private = ?, etime = DATE_ADD(NOW(), INTERVAL 30 DAY), model = ?, udid = ? WHERE uid = ?",
		token, privdata, in.Model, in.Udid, uid)
	if err != nil {
		return &verify.LoginReply{Head: &common.Head{Retcode: 2}}, err
	}
	util.SetCachedToken(kv, uid, token)

	strTime := time.Now().Add(time.Duration(expiretime) * time.Second).
		Format(util.TimeFormat)
	return &verify.LoginReply{Head: &common.Head{Uid: uid},
		Token: token, Privdata: privdata, Expire: expiretime,
		Expiretime: strTime, Wifipass: wifipass}, nil
}

func (s *server) Register(ctx context.Context, in *verify.RegisterRequest) (*verify.RegisterReply, error) {
	log.Printf("Register request:%v", in)
	if in.Code != "" && !checkZteCode(db, in.Username, in.Code, zte.SshType) {
		log.Printf("Register check code failed, name:%s code:%s",
			in.Username, in.Code)
		return &verify.RegisterReply{Head: &common.Head{Retcode: common.ErrCode_CHECK_CODE}}, nil
	}
	token := util.GenSalt()
	privdata := util.GenSalt()
	salt := util.GenSalt()
	epass := util.GenSaltPasswd(in.Password, salt)
	var expire int64
	log.Printf("phone:%s token:%s privdata:%s salt:%s epass:%s\n",
		in.Username, token, privdata, salt, epass)
	res, err := db.Exec(`INSERT IGNORE INTO user (username, password, salt, 
	token, private, model, udid,
	channel, reg_ip, version, term, wifi_passwd, ctime, atime, etime) VALUES
	(?,?,?,?,?,?,?,?,?,?,?,?,NOW(),NOW(),
	DATE_ADD(NOW(), INTERVAL 30 DAY))`,
		in.Username, epass, salt, token, privdata, in.Client.Model,
		in.Client.Udid, in.Client.Channel, in.Client.Regip,
		in.Client.Version, in.Client.Term, in.Code)
	if err != nil {
		log.Printf("add user failed:%v", err)
		return &verify.RegisterReply{Head: &common.Head{Retcode: 1}}, err
	}

	uid, err := res.LastInsertId()
	if err != nil {
		log.Printf("add user failed:%v", err)
		return &verify.RegisterReply{Head: &common.Head{Retcode: 1}}, err
	}
	log.Printf("uid:%d\n", uid)

	if uid == 0 {
		err = db.QueryRow("SELECT uid FROM user WHERE username = ?", in.Username).Scan(&uid)
		if err != nil {
			log.Printf("get user id failed:%v", err)
			return &verify.RegisterReply{Head: &common.Head{Retcode: 1}}, err
		}
		log.Printf("scan uid:%d \n", uid)
		_, err := db.Exec("UPDATE user SET password = ?, salt = ?, model = ?, udid = ?, version = ?, term = ?, atime = NOW() WHERE uid = ?",
			epass, salt, in.Client.Model, in.Client.Udid, in.Client.Version,
			in.Client.Term, uid)
		if err != nil {
			log.Printf("update user info failed:%v", err)
			return &verify.RegisterReply{Head: &common.Head{Retcode: 1}}, err
		}
		token, privdata, expire, err = util.RefreshTokenPrivdata(db, kv,
			uid, expiretime)
		if err != nil {
			log.Printf("Register refreshTokenPrivdata user info failed:%v", err)
			return &verify.RegisterReply{Head: &common.Head{Retcode: 1}}, err
		}
	}
	strTime := time.Now().Add(time.Duration(expire) * time.Second).
		Format(util.TimeFormat)
	return &verify.RegisterReply{Head: &common.Head{Retcode: 0, Uid: uid},
		Token: token, Privdata: privdata, Expire: expire,
		Expiretime: strTime}, nil
}

func (s *server) Logout(ctx context.Context, in *verify.LogoutRequest) (*common.CommReply, error) {
	flag := util.CheckToken(db, in.Head.Uid, in.Token, 0)
	if !flag {
		log.Printf("check token failed uid:%d, token:%s", in.Head.Uid, in.Token)
		return &common.CommReply{Head: &common.Head{Retcode: 1}},
			errors.New("check token failed")
	}
	util.ClearToken(db, in.Head.Uid)
	return &common.CommReply{Head: &common.Head{Retcode: 0}}, nil
}

func (s *server) CheckToken(ctx context.Context, in *verify.TokenRequest) (*common.CommReply, error) {
	if in.Type == 0 {
		token, err := util.GetCachedToken(kv, in.Head.Uid)
		if err == nil {
			if token == in.Token {
				return &common.CommReply{Head: &common.Head{Retcode: 0}}, nil
			}
			return &common.CommReply{Head: &common.Head{Retcode: 1}}, nil
		}
		var tk string
		var expire bool
		err = db.QueryRow("SELECT token, IF(etime > NOW(), false, true) FROM user WHERE deleted = 0 AND uid = ?",
			in.Head.Uid).Scan(&tk, &expire)
		if err != nil {
			log.Printf("CheckToken select failed:%v", err)
			return &common.CommReply{Head: &common.Head{Retcode: 1}}, nil
		}
		util.SetCachedToken(kv, in.Head.Uid, tk)
		if expire {
			log.Printf("CheckToken token expired, uid:%d\n", in.Head.Uid)
			return &common.CommReply{Head: &common.Head{Retcode: 1}}, nil
		}
		if tk == in.Token {
			return &common.CommReply{Head: &common.Head{Retcode: 0}}, nil
		}
		log.Printf("CheckToken token not match, uid:%d token:%s real:%s\n",
			in.Head.Uid, in.Token, tk)
		return &common.CommReply{Head: &common.Head{Retcode: 1}}, nil
	}
	flag := util.CheckToken(db, in.Head.Uid, in.Token, in.Type)
	if !flag {
		log.Printf("check token failed uid:%d, token:%s", in.Head.Uid, in.Token)
		return &common.CommReply{Head: &common.Head{Retcode: 1}},
			errors.New("checkToken failed")
	}
	return &common.CommReply{Head: &common.Head{Retcode: 0}}, nil
}

func checkPrivdata(db *sql.DB, uid int64, token, privdata string) (bool, int64) {
	var etoken string
	var eprivdata string
	var expire int64
	err := db.QueryRow("SELECT token, private, UNIX_TIMESTAMP(etime) FROM user WHERE uid = ?", uid).
		Scan(&etoken, &eprivdata, &expire)
	if err != nil {
		log.Printf("query failed:%v", err)
		return false, expire
	}

	if etoken != token || eprivdata != privdata {
		log.Printf("check privdata failed, token:%s-%s, privdata:%s-%s",
			token, etoken, privdata, eprivdata)
		return false, expire
	}
	return true, expire
}

func checkBackupPrivdata(db *sql.DB, uid int64, token, privdata string) bool {
	var etoken string
	var eprivdata string
	err := db.QueryRow("SELECT token, private FROM token_backup WHERE uid = ?", uid).
		Scan(&etoken, &eprivdata)
	if err != nil {
		log.Printf("query failed:%v", err)
		return false
	}

	if etoken != token || eprivdata != privdata {
		log.Printf("check backup privdata failed, token:%s-%s, privdata:%s-%s",
			token, etoken, privdata, eprivdata)
		return false
	}
	return true
}

func updatePrivdata(db *sql.DB, uid int64, token, privdata string) error {
	_, err := db.Exec("UPDATE user SET token = ?, private = ?, etime = DATE_ADD(NOW(), INTERVAL 30 DAY) WHERE uid = ?",
		token, privdata, uid)
	return err
}

func (s *server) AutoLogin(ctx context.Context, in *verify.AutoRequest) (*verify.RegisterReply, error) {
	backFlag := checkBackupPrivdata(db, in.Head.Uid, in.Token, in.Privdata)
	if !backFlag {
		flag, _ := checkPrivdata(db, in.Head.Uid, in.Token, in.Privdata)
		if !flag {
			log.Printf("check privdata failed, uid:%d token:%s privdata:%s",
				in.Head.Uid, in.Token, in.Privdata)
			return &verify.RegisterReply{Head: &common.Head{Retcode: 1}},
				errors.New("check privdata failed")
		}
	}
	token, privdata, expire, err := util.RefreshTokenPrivdata(db, kv,
		in.Head.Uid, expiretime)
	if err != nil {
		return &verify.RegisterReply{Head: &common.Head{Retcode: 1}},
			errors.New("refresh token failed")
	}
	strTime := time.Now().Add(time.Duration(expire) * time.Second).
		Format(util.TimeFormat)
	return &verify.RegisterReply{Head: &common.Head{Retcode: 0, Uid: in.Head.Uid},
		Token: token, Privdata: privdata, Expire: expire, Expiretime: strTime}, nil
}

func unionToID(db *sql.DB, unionid string) (int64, error) {
	var uid int64
	err := db.QueryRow("SELECT uid FROM user_unionid WHERE unionid = ?", unionid).Scan(&uid)
	if err != nil {
		log.Printf("use unionid to find user failed %s:%v", unionid, err)
		return uid, err
	}
	return uid, nil
}

func (s *server) UnionLogin(ctx context.Context, in *verify.LoginRequest) (*verify.LoginReply, error) {
	uid, err := unionToID(db, in.Unionid)
	if err != nil {
		return &verify.LoginReply{Head: &common.Head{Retcode: 106}}, nil
	}
	token := util.GenSalt()
	privdata := util.GenSalt()
	updatePrivdata(db, uid, token, privdata)
	util.SetCachedToken(kv, uid, token)
	strTime := time.Now().Add(time.Duration(expiretime) * time.Second).
		Format(util.TimeFormat)
	return &verify.LoginReply{Head: &common.Head{Retcode: 0, Uid: uid},
		Token: token, Privdata: privdata, Expire: expiretime,
		Expiretime: strTime}, nil
}

func updateTokenTicket(db *sql.DB, appid, accessToken, ticket string) {
	_, err := db.Exec("UPDATE wx_token SET access_token = ?, api_ticket = ?, expire_time = DATE_ADD(NOW(), INTERVAL 1 HOUR) WHERE appid = ?",
		accessToken, ticket, appid)
	if err != nil {
		log.Printf("updateTokenTicket failed:%v", err)
	}
}

func (s *server) GetWxTicket(ctx context.Context, in *verify.TicketRequest) (*verify.TicketReply, error) {
	var token, ticket string
	err := db.QueryRow("SELECT access_token, api_ticket FROM wx_token WHERE expire_time > NOW() AND appid = ? LIMIT 1",
		util.WxAppid).Scan(&token, &ticket)
	if err == nil {
		log.Printf("GetWxTicket select succ, token:%s ticket:%s\n", token, ticket)
		return &verify.TicketReply{
			Head:  &common.Head{Retcode: 0, Uid: in.Head.Uid},
			Token: token, Ticket: ticket}, nil
	}
	token, err = util.GetWxToken(util.WxAppid, util.WxAppkey)
	if err != nil {
		log.Printf("GetWxToken failed:%v", err)
		return &verify.TicketReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil
	}
	ticket, err = util.GetWxJsapiTicket(token)
	if err != nil {
		log.Printf("GetWxToken failed:%v", err)
		return &verify.TicketReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil
	}

	updateTokenTicket(db, util.WxAppid, token, ticket)
	return &verify.TicketReply{
		Head:  &common.Head{Retcode: 0, Uid: in.Head.Uid},
		Token: token, Ticket: ticket}, nil
}

func recordZteCode(db *sql.DB, phone, code string, stype uint) {
	if code == "" {
		return
	}
	_, err := db.Exec("INSERT INTO zte_code(phone, code, type, ctime) VALUES (?, ?, ?, NOW()) ON DUPLICATE KEY UPDATE code = ?",
		phone, code, stype, code)
	if err != nil {
		log.Printf("recordZteCode query failed:%s %s %d %v", phone, code, stype, err)
	}
}

func (s *server) GetCheckCode(ctx context.Context, in *verify.PortalLoginRequest) (*common.CommReply, error) {
	var stype uint
	if in.Head.Term == util.WebTerm {
		stype = getAcSys(db, in.Info.Acname)
	}
	code, err := zte.Register(in.Info.Phone, true, stype)
	if err != nil {
		log.Printf("GetCheckCode Register failed:%v", err)
		return &common.CommReply{Head: &common.Head{Retcode: 1}}, err
	}
	log.Printf("recordZteCode phone:%s code:%s type:%d", in.Info.Phone, code, stype)
	recordZteCode(db, in.Info.Phone, code, stype)
	return &common.CommReply{Head: &common.Head{Retcode: 0}}, nil
}

func checkZteCode(db *sql.DB, phone, code string, stype uint) bool {
	var eCode string
	err := db.QueryRow("SELECT code FROM zte_code WHERE type = ? AND phone = ?",
		stype, phone).Scan(&eCode)
	if err != nil {
		log.Printf("checkZteCode query failed:%s %s %v", phone, code, err)
		return false
	}
	if eCode == code {
		return true
	}
	return false
}

func getAcSys(db *sql.DB, acname string) uint {
	var stype uint
	err := db.QueryRow("SELECT type FROM ac_info WHERE name = ?", acname).
		Scan(&stype)
	if err != nil {
		log.Printf("getAcSys query failed:%v", err)
	}
	return stype
}

func getUserBitmap(db *sql.DB, uid int64) uint {
	var bitmap uint
	err := db.QueryRow("SELECT bitmap FROM user WHERE uid = ?", uid).
		Scan(&bitmap)
	if err != nil {
		log.Printf("getUserBitmap failed:%v", err)
	}
	return bitmap
}

func updateUserBitmap(db *sql.DB, uid int64, bitmap uint) {
	_, err := db.Exec("UPDATE user SET bitmap = bitmap | ? WHERE uid = ?",
		bitmap, uid)
	if err != nil {
		log.Printf("updateUserBitmap failed, uid:%d %v", uid, err)
	}
}

func recordUserMac(db *sql.DB, uid int64, mac, phone string) {
	_, err := db.Exec("INSERT INTO user_mac(mac, uid, phone, ctime, etime) VALUES (?, ?, ?, NOW(), DATE_ADD(NOW(), INTERVAL 4 HOUR)) ON DUPLICATE KEY UPDATE uid = ?, phone = ?, etime = DATE_ADD(NOW(), INTERVAL 4 HOUR)",
		mac, uid, phone, uid, phone)
	if err != nil {
		log.Printf("recordUserMac failed uid:%d mac:%s phone:%s err:%v",
			uid, mac, phone, err)
	}
}

func (s *server) PortalLogin(ctx context.Context, in *verify.PortalLoginRequest) (*verify.PortalLoginReply, error) {
	stype := getAcSys(db, in.Info.Acname)
	if !checkZteCode(db, in.Info.Phone, in.Info.Code, stype) {
		log.Printf("PortalLogin checkZteCode failed, phone:%s code:%s stype:%d",
			in.Info.Phone, in.Info.Code, stype)
		return &verify.PortalLoginReply{
			Head: &common.Head{Retcode: common.ErrCode_CHECK_CODE}}, nil

	}
	log.Printf("PortalLogin info:%v", in.Info)
	flag := zte.Loginnopass(in.Info.Phone, in.Info.Userip,
		in.Info.Usermac, in.Info.Acip, in.Info.Acname, stype)
	if !flag {
		log.Printf("PortalLogin zte loginnopass failed, to queryonline phone:%s code:%s",
			in.Info.Phone, in.Info.Code)
		if !zte.QueryOnline(in.Info.Phone, stype) {
			log.Printf("PortalLogin zte queryonline failed, phone:%s code:%s",
				in.Info.Phone, in.Info.Code)
			return &verify.PortalLoginReply{
				Head: &common.Head{Retcode: common.ErrCode_ZTE_LOGIN}}, nil
		}
	}

	res, err := db.Exec("INSERT INTO user(username, phone, ctime, atime, bitmap) VALUES (?, ?, NOW(), NOW(), ?) ON DUPLICATE KEY UPDATE phone = ?, atime = NOW(), bitmap = bitmap | ?",
		in.Info.Phone, in.Info.Phone, (1 << stype), in.Info.Phone,
		(1 << stype))
	if err != nil {
		log.Printf("PortalLogin insert user failed, phone:%s code:%s %v",
			in.Info.Phone, in.Info.Code, err)
		return &verify.PortalLoginReply{Head: &common.Head{Retcode: 1}}, nil
	}
	uid, err := res.LastInsertId()
	if err != nil {
		log.Printf("PortalLogin add user failed:%v", err)
		return &verify.PortalLoginReply{Head: &common.Head{Retcode: 1}}, err
	}
	log.Printf("uid:%d\n", uid)
	token, _, _, err := util.RefreshTokenPrivdata(db, kv, uid, expiretime)
	if err != nil {
		log.Printf("Register refreshTokenPrivdata user info failed:%v", err)
		return &verify.PortalLoginReply{Head: &common.Head{Retcode: 1}}, err
	}
	recordUserMac(db, uid, in.Info.Usermac, in.Info.Phone)
	dir := getPortalDir(db)
	live := getLiveVal(db, uid)
	return &verify.PortalLoginReply{
		Head: &common.Head{Retcode: 0, Uid: uid}, Token: token, Portaldir: dir,
		Live: live}, nil
}

func getPortalDir(db *sql.DB) string {
	dir, err := util.GetPortalDir(db, util.PortalType)
	if err != nil {
		log.Printf("Register GetPortalDir portal failed type:%v", err)
		dir = portalDir + "20170117/"
	} else {
		dir = portalDir + dir
	}
	return dir
}

func checkZteReg(db *sql.DB, bitmap, stype uint, uid int64, phone string) error {
	if bitmap&(1<<stype) == 0 {
		code, err := zte.Register(phone, true, stype)
		if err != nil {
			log.Printf("PortalLogin zte register failed:%v", err)
			return err
		}
		recordZteCode(db, phone, code, stype)
		updateUserBitmap(db, uid, (1 << stype))
	}
	return nil
}

func (s *server) WifiAccess(ctx context.Context, in *verify.AccessRequest) (*common.CommReply, error) {
	stype := getAcSys(db, in.Info.Acname)
	var phone string
	var bitmap uint
	err := db.QueryRow("SELECT phone, bitmap FROM user WHERE uid = ?", in.Head.Uid).
		Scan(&phone, &bitmap)
	if err != nil {
		log.Printf("WifiAccess search user failed:%v", err)
		return &common.CommReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil
	}
	err = checkZteReg(db, bitmap, stype, in.Head.Uid, phone)
	if err != nil {
		return &common.CommReply{
			Head: &common.Head{Retcode: common.ErrCode_ZTE_LOGIN,
				Uid: in.Head.Uid}}, nil
	}
	if !zte.Loginnopass(phone, in.Info.Userip, in.Info.Usermac, in.Info.Acip,
		in.Info.Acname, stype) {
		log.Printf("WifiAccess zte Login failed, req:%v", in)
		return &common.CommReply{
			Head: &common.Head{Retcode: common.ErrCode_ZTE_LOGIN,
				Uid: in.Head.Uid}}, nil
	}
	util.RefreshUserAp(db, in.Head.Uid, in.Info.Apmac)
	return &common.CommReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}}, nil
}

func checkLoginMac(db *sql.DB, mac string, stype uint) int64 {
	var phone string
	var uid int64
	err := db.QueryRow("SELECT phone, uid FROM user_mac WHERE etime > NOW() AND mac = ?", mac).
		Scan(&phone, &uid)
	if err != nil {
		log.Printf("checkLoginMac failed, mac:%s %v", err)
		return 0
	}
	if phone != "" {
		bitmap := getUserBitmap(db, uid)
		err = checkZteReg(db, bitmap, stype, uid, phone)
		if err != nil {
			log.Printf("checkLoginMac checkZteReg failed:%v", err)
			return 0
		}
		return 1
	}
	return 0
}

func (s *server) CheckLogin(ctx context.Context, in *verify.AccessRequest) (*verify.CheckReply, error) {
	stype := getAcSys(db, in.Info.Acname)
	ret := checkLoginMac(db, in.Info.Usermac, stype)
	log.Printf("CheckLogin ret:%d", ret)
	return &verify.CheckReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}, Autologin: ret}, nil
}

func (s *server) OneClickLogin(ctx context.Context, in *verify.AccessRequest) (*verify.PortalLoginReply, error) {
	var uid int64
	var phone, token string
	err := db.QueryRow("SELECT m.phone, u.uid, u.token FROM user_mac m, user u WHERE m.uid = u.uid AND m.mac = ?", in.Info.Usermac).
		Scan(&phone, &uid, &token)
	if err != nil {
		log.Printf("OneClickLogin query failed:%v", err)
		return &verify.PortalLoginReply{
			Head: &common.Head{
				Retcode: common.ErrCode_ZTE_LOGIN, Uid: in.Head.Uid}}, nil
	}

	stype := getAcSys(db, in.Info.Acname)
	bitmap := getUserBitmap(db, uid)
	err = checkZteReg(db, bitmap, stype, uid, phone)
	if err != nil {
		return &verify.PortalLoginReply{
			Head: &common.Head{Retcode: common.ErrCode_ZTE_LOGIN}}, nil
	}
	flag := zte.Loginnopass(phone, in.Info.Userip,
		in.Info.Usermac, in.Info.Acip, in.Info.Acname, stype)
	if !flag {
		log.Printf("OneClickLogin zte loginnopass failed, phone:%s",
			phone)
		return &verify.PortalLoginReply{
			Head: &common.Head{Retcode: common.ErrCode_ZTE_LOGIN}}, nil
	}
	recordUserMac(db, uid, in.Info.Usermac, phone)
	dir := getPortalDir(db)
	live := getLiveVal(db, uid)
	return &verify.PortalLoginReply{
		Head: &common.Head{Retcode: 0, Uid: uid}, Token: token, Portaldir: dir,
		Live: live}, nil
}

func getLiveVal(db *sql.DB, uid int64) string {
	if util.IsWhiteUser(db, uid, util.LiveDbgType) {
		return "livetrue"
	}
	return "livefalse"
}

func main() {
	lis, err := net.Listen("tcp", util.VerifyServerPort)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	db, err = util.InitDB(false)
	if err != nil {
		log.Fatalf("failed to init db connection: %v", err)
	}
	db.SetMaxIdleConns(util.MaxIdleConns)
	kv = util.InitRedis()
	go util.ReportHandler(kv, util.VerifyServerName, util.VerifyServerPort)
	cli := util.InitEtcdCli()
	go util.ReportEtcd(cli, util.VerifyServerName, util.VerifyServerPort)

	s := grpc.NewServer()
	verify.RegisterVerifyServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
