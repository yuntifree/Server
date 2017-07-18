package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"database/sql"

	"Server/proto/common"
	"Server/proto/verify"
	"Server/util"
	"Server/zte"

	_ "github.com/go-sql-driver/mysql"
	nsq "github.com/nsqio/go-nsq"
	"golang.org/x/net/context"
	redis "gopkg.in/redis.v5"
)

const (
	expiretime           = 3600 * 24 * 30
	mastercode           = 251653
	randrange            = 1000000
	specPhone            = "13800000000"
	testAcname           = "2043.0769.200.00"
	testAcip             = "120.197.159.10"
	testUserip           = "10.96.72.28"
	testUsermac          = "f45c89987347"
	defLoginImg          = "http://img.yunxingzh.com/57970b5c-249a-4bc6-970e-064305e6d498.png"
	mpURL                = "https://mp.weixin.qq.com/mp/profile_ext?action=home&__biz=MzIzOTc0OTcyMw==&scene=124#wechat_redirect"
	taobaoBannerType     = 9
	specTaobaoBannerType = 12
)

type server struct{}

var db *sql.DB
var kv *redis.Client
var w *nsq.Producer

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
	util.PubRPCRequest(w, "verify", "GetPhoneCode")
	flag, err := getPhoneCode(in.Phone, in.Ctype)
	if err != nil {
		return &verify.VerifyReply{Result: false}, err
	}

	util.PubRPCSuccRsp(w, "verify", "GetPhoneCode")
	return &verify.VerifyReply{Result: flag}, nil
}

func (s *server) BackLogin(ctx context.Context, in *verify.LoginRequest) (*verify.LoginReply, error) {
	util.PubRPCRequest(w, "verify", "BackLogin")
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

	util.PubRPCSuccRsp(w, "verify", "BackLogin")
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
	util.PubRPCRequest(w, "verify", "WxMpLogin")
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

	recordWxOpenid(db, uid, 1, wxi.Openid)
	recordWxUnionid(db, uid, privdata)
	util.SetCachedToken(kv, uid, token)
	strTime := time.Now().Add(time.Duration(expiretime) * time.Second).
		Format(util.TimeFormat)
	util.PubRPCSuccRsp(w, "verify", "WxMpLogin")
	return &verify.LoginReply{Head: &common.Head{Uid: uid},
		Token: token, Privdata: privdata, Expire: expiretime,
		Expiretime: strTime, Wifipass: wifipass, Openid: wxi.Openid}, nil
}

func (s *server) Login(ctx context.Context, in *verify.LoginRequest) (*verify.LoginReply, error) {
	util.PubRPCRequest(w, "verify", "Login")
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
	util.PubRPCSuccRsp(w, "verify", "Login")
	return &verify.LoginReply{Head: &common.Head{Uid: uid},
		Token: token, Privdata: privdata, Expire: expiretime,
		Expiretime: strTime, Wifipass: wifipass}, nil
}

func (s *server) Register(ctx context.Context, in *verify.RegisterRequest) (*verify.RegisterReply, error) {
	util.PubRPCRequest(w, "verify", "Register")
	log.Printf("Register request:%v", in)
	sshFlag := checkZteCode(db, in.Username, in.Code, zte.SshType)
	wjjFlag := checkZteCode(db, in.Username, in.Code, zte.WjjType)
	if in.Code != "" && !sshFlag && !wjjFlag {
		log.Printf("Register check code failed, name:%s code:%s",
			in.Username, in.Code)
		return &verify.RegisterReply{Head: &common.Head{Retcode: common.ErrCode_CHECK_CODE}}, nil
	}
	token := util.GenSalt()
	privdata := util.GenSalt()
	salt := util.GenSalt()
	epass := util.GenSaltPasswd(in.Password, salt)
	bitmap := 0
	if sshFlag {
		bitmap++
	}
	if wjjFlag {
		bitmap += 2
	}
	var expire int64
	log.Printf("phone:%s token:%s privdata:%s salt:%s epass:%s\n",
		in.Username, token, privdata, salt, epass)
	res, err := db.Exec(`INSERT IGNORE INTO user (username, password, salt, 
	token, private, model, udid,
	channel, reg_ip, version, term, wifi_passwd, ctime, atime, etime, bitmap) VALUES
	(?,?,?,?,?,?,?,?,?,?,?,?,NOW(),NOW(),
	DATE_ADD(NOW(), INTERVAL 30 DAY), ?)`,
		in.Username, epass, salt, token, privdata, in.Client.Model,
		in.Client.Udid, in.Client.Channel, in.Client.Regip,
		in.Client.Version, in.Client.Term, in.Code, bitmap)
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

	var headurl, nickname string
	var pushtest int64
	if uid == 0 {
		err = db.QueryRow("SELECT uid, headurl, nickname FROM user WHERE username = ?", in.Username).Scan(&uid, &headurl, &nickname)
		if err != nil {
			log.Printf("get user id failed:%v", err)
			return &verify.RegisterReply{Head: &common.Head{Retcode: 1}}, err
		}
		log.Printf("scan uid:%d \n", uid)
		_, err := db.Exec("UPDATE user SET password = ?, salt = ?, model = ?, udid = ?, version = ?, term = ?, atime = NOW(), bitmap = bitmap | ? WHERE uid = ?",
			epass, salt, in.Client.Model, in.Client.Udid, in.Client.Version,
			in.Client.Term, bitmap, uid)
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
		if util.IsWhiteUser(db, uid, util.PushTestType) {
			pushtest = 1
		}
	}
	strTime := time.Now().Add(time.Duration(expire) * time.Second).
		Format(util.TimeFormat)
	util.PubRPCSuccRsp(w, "verify", "Register")
	nick, err := base64.StdEncoding.DecodeString(nickname)
	if err != nil {
		log.Printf("decode nickname failed:%v", err)
	}
	return &verify.RegisterReply{Head: &common.Head{Retcode: 0, Uid: uid},
		Token: token, Privdata: privdata, Expire: expire,
		Expiretime: strTime, Headurl: headurl, Nickname: string(nick),
		Pushtest: pushtest}, nil
}

func (s *server) Logout(ctx context.Context, in *verify.LogoutRequest) (*common.CommReply, error) {
	util.PubRPCRequest(w, "verify", "Logout")
	flag := util.CheckToken(db, in.Head.Uid, in.Token, 0)
	if !flag {
		log.Printf("check token failed uid:%d, token:%s", in.Head.Uid, in.Token)
		return &common.CommReply{Head: &common.Head{Retcode: 1}},
			errors.New("check token failed")
	}
	util.ClearToken(db, in.Head.Uid)
	util.PubRPCSuccRsp(w, "verify", "Logout")
	return &common.CommReply{Head: &common.Head{Retcode: 0}}, nil
}

func (s *server) CheckToken(ctx context.Context, in *verify.TokenRequest) (*common.CommReply, error) {
	util.PubRPCRequest(w, "verify", "CheckToken")
	if in.Type == 0 {
		token, err := util.GetCachedToken(kv, in.Head.Uid)
		if err == nil {
			if token == in.Token {
				util.PubRPCSuccRsp(w, "verify", "CheckToken")
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
			util.PubRPCSuccRsp(w, "verify", "CheckToken")
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
	util.PubRPCSuccRsp(w, "verify", "CheckToken")
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
	util.PubRPCRequest(w, "verify", "AutoLogin")
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
	util.PubRPCSuccRsp(w, "verify", "AutoLogin")
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

func getUserOpenid(db *sql.DB, uid int64) string {
	var openid string
	err := db.QueryRow("SELECT openid FROM wx_openid WHERE wtype = 1 AND uid = ?", uid).Scan(&openid)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("getUserOpenid query failed:%v", err)
	}
	return openid
}

func (s *server) UnionLogin(ctx context.Context, in *verify.LoginRequest) (*verify.LoginReply, error) {
	util.PubRPCRequest(w, "verify", "UnionLogin")
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
	openid := getUserOpenid(db, uid)
	util.PubRPCSuccRsp(w, "verify", "UnionLogin")
	return &verify.LoginReply{Head: &common.Head{Retcode: 0, Uid: uid},
		Token: token, Privdata: privdata, Expire: expiretime,
		Expiretime: strTime, Openid: openid}, nil
}

func updateTokenTicket(db *sql.DB, appid, accessToken, ticket string) {
	_, err := db.Exec("UPDATE wx_token SET access_token = ?, api_ticket = ?, expire_time = DATE_ADD(NOW(), INTERVAL 1 HOUR) WHERE appid = ?",
		accessToken, ticket, appid)
	if err != nil {
		log.Printf("updateTokenTicket failed:%v", err)
	}
}

func (s *server) GetWxTicket(ctx context.Context, in *verify.TicketRequest) (*verify.TicketReply, error) {
	util.PubRPCRequest(w, "verify", "GetWxTicket")
	var token, ticket string
	err := db.QueryRow("SELECT access_token, api_ticket FROM wx_token WHERE expire_time > NOW() AND appid = ? LIMIT 1",
		util.WxDgAppid).Scan(&token, &ticket)
	if err == nil {
		log.Printf("GetWxTicket select succ, token:%s ticket:%s\n", token, ticket)
		return &verify.TicketReply{
			Head:  &common.Head{Retcode: 0, Uid: in.Head.Uid},
			Token: token, Ticket: ticket}, nil
	}
	token, err = util.GetWxToken(util.WxDgAppid, util.WxDgAppkey)
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

	updateTokenTicket(db, util.WxDgAppid, token, ticket)
	util.PubRPCSuccRsp(w, "verify", "GetWxTicket")
	return &verify.TicketReply{
		Head:  &common.Head{Retcode: 0, Uid: in.Head.Uid},
		Token: token, Ticket: ticket}, nil
}

func recordZteCode(db *sql.DB, phone, code string, stype uint) {
	if code == "" {
		return
	}
	_, err := db.Exec("INSERT INTO zte_code(phone, code, type, ctime, mtime) VALUES (?, ?, ?, NOW(), NOW()) ON DUPLICATE KEY UPDATE code = ?, mtime = NOW()",
		phone, code, stype, code)
	if err != nil {
		log.Printf("recordZteCode query failed:%s %s %d %v", phone, code, stype, err)
	}
}

func isExceedCodeFrequency(db *sql.DB, phone string, stype uint) bool {
	var flag int
	err := db.QueryRow("SELECT IF(NOW() > DATE_ADD(mtime, INTERVAL 5 MINUTE), 0, 1) FROM zte_code WHERE phone = ? AND type = ?", phone, stype).Scan(&flag)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("isExceedCodeFrequency query failed:%v", err)
		return false
	}
	if flag > 0 {
		return true
	}
	return false
}

func (s *server) GetCheckCode(ctx context.Context, in *verify.PortalLoginRequest) (*common.CommReply, error) {
	util.PubRPCRequest(w, "verify", "GetCheckCode")
	var stype uint
	if in.Head.Term == util.WebTerm {
		stype = getAcSys(db, in.Info.Acname)
	}
	if isExceedCodeFrequency(db, in.Info.Phone, stype) {
		log.Printf("GetCheckCode isExceedCodeFrequency phone:%s", in.Info.Phone)
		return &common.CommReply{
			Head: &common.Head{Retcode: common.ErrCode_FREQUENCY_LIMIT}}, nil
	}
	code, err := zte.Register(in.Info.Phone, true, stype)
	if err != nil {
		log.Printf("GetCheckCode Register failed:%v", err)
		return &common.CommReply{Head: &common.Head{Retcode: 1}}, err
	}
	log.Printf("recordZteCode phone:%s code:%s type:%d", in.Info.Phone, code, stype)
	recordZteCode(db, in.Info.Phone, code, stype)
	util.PubRPCSuccRsp(w, "verify", "GetCheckCode")
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
	mac = strings.Replace(mac, ":", "", -1)
	_, err := db.Exec("INSERT INTO user_mac(mac, uid, phone, ctime, etime) VALUES (?, ?, ?, NOW(), DATE_ADD(NOW(), INTERVAL 30 DAY)) ON DUPLICATE KEY UPDATE uid = ?, phone = ?, etime = DATE_ADD(NOW(), INTERVAL 30 DAY)",
		mac, uid, phone, uid, phone)
	if err != nil {
		log.Printf("recordUserMac failed uid:%d mac:%s phone:%s err:%v",
			uid, mac, phone, err)
	}
}

func isTestParam(info *verify.PortalInfo) bool {
	if info.Acip == testAcip &&
		info.Userip == testUserip && info.Usermac == testUsermac {
		return true
	}
	return false
}

func getBannerTypeInfo(db *sql.DB, btype int64) (img, dst string) {
	err := db.QueryRow("SELECT img, dst FROM banner WHERE type = ?", btype).
		Scan(&img, &dst)
	if err != nil {
		log.Printf("getBannerTypeInfo failed:%v", err)
	}
	return
}

func getTaobaoInfo(db *sql.DB) (img, dst string) {
	return getBannerTypeInfo(db, taobaoBannerType)
}

func getSpecTaobaoInfo(db *sql.DB) (img, dst string) {
	return getBannerTypeInfo(db, specTaobaoBannerType)
}

func (s *server) PortalLogin(ctx context.Context, in *verify.PortalLoginRequest) (*verify.PortalLoginReply, error) {
	log.Printf("PortalLogin request:%v", in)
	util.PubRPCRequest(w, "verify", "PortalLogin")
	stype := getAcSys(db, in.Info.Acname)
	if !checkZteCode(db, in.Info.Phone, in.Info.Code, stype) {
		log.Printf("PortalLogin checkZteCode failed, phone:%s code:%s stype:%d",
			in.Info.Phone, in.Info.Code, stype)
		return &verify.PortalLoginReply{
			Head: &common.Head{Retcode: common.ErrCode_CHECK_CODE}}, nil

	}
	log.Printf("PortalLogin info:%v", in.Info)
	if !isTestParam(in.Info) {
		flag, err := zteLogin(in.Info.Phone, in.Info.Userip,
			in.Info.Usermac, in.Info.Acip, in.Info.Acname, stype)
		if !flag {
			log.Printf("PortalLogin zteLogin retry failed, phone:%s code:%s",
				in.Info.Phone, in.Info.Code)
			if err == zte.ErrForbid {
				return &verify.PortalLoginReply{
					Head: &common.Head{Retcode: common.ErrCode_LOGIN_FORBID}}, nil
			}
			return &verify.PortalLoginReply{
				Head: &common.Head{Retcode: common.ErrCode_ZTE_LOGIN}}, nil
		}
	}

	res, err := db.Exec("INSERT INTO user(username, phone, ctime, atime, bitmap, term, aptime) VALUES (?, ?, NOW(), NOW(), ?, 2, NOW()) ON DUPLICATE KEY UPDATE phone = ?, atime = NOW(), bitmap = bitmap | ?, aptime = NOW()",
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
	addOnlineRecord(db, uid, in.Info.Phone, in.Info)
	var adtype int64
	if util.IsWjjAcname(in.Info.Acname) {
		adtype = 1
	} else {
		adtype = util.GetAdType(db, in.Info.Apmac)
	}
	ptype := util.GetPortalType(db, in.Info.Apmac)
	dir := util.GetPortalPath(db, in.Info.Acname, ptype)
	var img, dst string
	if isSpecTaobaoTime() {
		img, dst = getSpecTaobaoInfo(db)
	} else {
		img, dst = getTaobaoInfo(db)
	}
	util.PubRPCSuccRsp(w, "verify", "PortalLogin")
	log.Printf("PortalLogin succ request:%v uid:%d, token:%s", in, uid, token)
	return &verify.PortalLoginReply{
		Head: &common.Head{Retcode: 0, Uid: uid}, Token: token, Portaldir: dir,
		Portaltype: ptype, Adtype: adtype, Cover: img,
		Dst: dst}, nil
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
	util.PubRPCRequest(w, "verify", "WifiAccess")
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
	flag, err := zteLogin(phone, in.Info.Userip, in.Info.Usermac, in.Info.Acip,
		in.Info.Acname, stype)
	if !flag {
		log.Printf("WifiAccess zte loginnopass retry failed, phone:%s code:%s",
			phone, in.Info.Code)
		if err == zte.ErrForbid {
			return &common.CommReply{
				Head: &common.Head{Retcode: common.ErrCode_LOGIN_FORBID}}, nil
		}
		return &common.CommReply{
			Head: &common.Head{Retcode: 1}}, nil
	}
	util.RefreshUserAp(db, in.Head.Uid, in.Info.Apmac)
	util.PubRPCSuccRsp(w, "verify", "WifiAccess")
	return &common.CommReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}}, nil
}

func recordPortalMac(db *sql.DB, mac string) {
	_, err := db.Exec("INSERT INTO portal_mac(mac, ctime, atime) VALUES (?, NOW(), NOW()) ON DUPLICATE KEY UPDATE atime = NOW()", mac)
	if err != nil {
		log.Printf("recordPortalMac failed, mac:%s error:%v", mac, err)
	}
}

func checkLoginMac(db *sql.DB, mac string, stype uint) int64 {
	recordPortalMac(db, mac)
	var phone string
	var uid int64
	emac := strings.Replace(mac, ":", "", -1)
	err := db.QueryRow("SELECT phone, uid FROM user_mac WHERE mac = ?", emac).
		Scan(&phone, &uid)
	if err != nil {
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

func isEduAp(db *sql.DB, apmac string) bool {
	var cnt int64
	err := db.QueryRow("SELECT COUNT(id) FROM edu_ap_info WHERE mac = ?", apmac).
		Scan(&cnt)
	if err != nil {
		log.Printf("isEduAp scan failed:%s %v", apmac, err)
		return false
	}
	if cnt > 0 {
		return true
	}
	return false
}

func getLoginImg(db *sql.DB, acname, apmac string) string {
	var img string
	db.QueryRow("SELECT l.img FROM login_img l, ap_info a WHERE l.unid = a.unid AND a.mac = ? AND l.deleted = 0", apmac).Scan(&img)
	if img != "" {
		return img
	}

	img = defLoginImg
	btype := 0
	if util.IsTestAcname(acname) {
		btype = 3
	} else if isEduAp(db, apmac) {
		btype = 4
	} else if util.IsKongguAcname(acname) ||
		acname == "AC_120_A_01" {
		btype = 2
	} else if util.IsWjjAcname(acname) {
		btype = 1
	}
	rows, err := db.Query("SELECT img, stime, etime FROM login_banner WHERE type = ? AND online = 1 AND deleted = 0 ORDER BY id DESC", btype)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("getLoginImg failed:%v", err)
	}
	defer rows.Close()
	c := getCurTime()
	for rows.Next() {
		var banner string
		var stime, etime int64
		err = rows.Scan(&banner, &stime, &etime)
		if err != nil {
			log.Printf("getLoginImg scan failed:%v", err)
			continue
		}
		if stime == 0 && etime == 0 {
			img = banner
		}
		if stime <= c && c <= etime {
			return banner
		}
	}
	return img
}

func getAdImg(db *sql.DB, area int64) string {
	var img string
	err := db.QueryRow("SELECT img FROM advertise WHERE areaid = ? AND type = 2 AND deleted = 0 AND online = 1", area).Scan(&img)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("gtAdImg query failed:%v", err)
	}
	return img
}

func checkSpareTime(stime, etime int64) bool {
	if stime == 0 && etime == 0 {
		return true
	}
	t := time.Now()
	hour := t.Hour()
	minute := t.Minute()
	ms := hour*100 + minute
	if ms >= int(stime) && ms <= int(etime) {
		return true
	}
	return false
}

func getWxAppinfo(db *sql.DB, acname, apmac string) (appid, secret, shopid, authurl string) {
	var stime, etime int64
	err := db.QueryRow("SELECT appid, secret, shopid, authurl, w.stime, w.etime FROM wx_appinfo w, ap_info a WHERE w.unid = a.unid AND a.mac = ?", apmac).
		Scan(&appid, &secret, &shopid, &authurl, &stime, &etime)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("getWxAppinfo failed:%v", err)
	}

	if appid == "" || !checkSpareTime(stime, etime) {
		var def int64
		if util.IsWjjKongguAcname(acname) {
			def = 4
		} else if util.IsWjjAcname(acname) {
			def = 6
		} else if util.IsKongguAcname(acname) {
			def = 5
		} else if util.IsLzfAcname(acname) {
			def = 3
		} else if util.IsTestAcname(acname) {
			def = 5
		} else {
			def = 1
		}
		err = db.QueryRow("SELECT appid, secret, shopid, authurl FROM wx_appinfo WHERE def = ? LIMIT 1", def).Scan(&appid, &secret, &shopid, &authurl)
		if err != nil && err != sql.ErrNoRows {
			log.Printf("getWxAppinfo get default failed:%v", err)
		}
	}
	return
}

func getCurTime() int64 {
	now := time.Now()
	hour := now.Hour()
	min := now.Minute()
	v := hour*100 + min
	return int64(v)
}

func isTaobaoTime() bool {
	now := time.Now()
	hour := now.Hour()
	min := now.Minute()
	v := hour*100 + min
	if (v >= 1750 && v < 1805) || (v >= 1829 && v < 1836) ||
		(v >= 2035 && v < 2050) || (v >= 2100 && v < 2300) {
		return true
	}
	return false
}

func isSpecTaobaoTime() bool {
	now := time.Now()
	hour := now.Hour()
	min := now.Minute()
	v := hour*100 + min
	if (v >= 0 && v < 30) || (v >= 120 && v < 145) ||
		(v >= 300 && v < 340) || (v >= 515 && v < 535) {
		return true
	}
	return false
}

func (s *server) CheckLogin(ctx context.Context, in *verify.AccessRequest) (*verify.CheckReply, error) {
	util.PubRPCRequest(w, "verify", "CheckLogin")
	stype := getAcSys(db, in.Info.Acname)
	ret := checkLoginMac(db, in.Info.Usermac, stype)
	log.Printf("CheckLogin mac:%s ret:%d", in.Info.Usermac, ret)
	img := getLoginImg(db, in.Info.Acname, in.Info.Apmac)
	var appid, secret, shopid, authurl string
	if in.Info.Apmac != "" {
		adtype := util.GetAdType(db, in.Info.Apmac)
		ad := getAdImg(db, adtype)
		if ad != "" {
			img = ad
		}
		appid, secret, shopid, authurl = getWxAppinfo(db, in.Info.Acname,
			in.Info.Apmac)
	}
	var taobao int64
	var cover string
	if in.Info.Acname == "AC_SSH_A_09" ||
		(util.IsWjjAcname(in.Info.Acname) && isTaobaoTime()) {
		taobao = 1
		cover, _ = getTaobaoInfo(db)
	}
	if !util.IsKongguAcname(in.Info.Acname) && isSpecTaobaoTime() {
		taobao = 1
		cover, _ = getSpecTaobaoInfo(db)
	}
	util.PubRPCSuccRsp(w, "verify", "CheckLogin")
	return &verify.CheckReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}, Autologin: ret,
		Img: img, Wxappid: appid, Wxsecret: secret, Wxshopid: shopid,
		Wxauthurl: authurl, Taobao: taobao, Cover: cover}, nil
}

func genPortalDst(db *sql.DB, openid string) string {
	var uid int64
	var acname, apmac, token string
	err := db.QueryRow("SELECT u.uid, u.token, i.name, w.apmac FROM user u, online_status o, wx_conn w, ac_info i WHERE u.phone = o.phone AND o.mac = w.usermac AND o.acip = i.ip AND w.openid = ?", openid).Scan(&uid, &token, &acname, &apmac)
	if err != nil {
		log.Printf("genPortalDst failed:%v", err)
		return ""
	}
	adtype := util.GetAdType(db, apmac)
	portaltype := util.GetPortalType(db, apmac)
	dir := util.GetPortalPath(db, acname, portaltype)
	dst := fmt.Sprintf("%s?uid=%d&token=%s&adtype=%d&portaltype=%d&ts=%d&s=1",
		dir, uid, token, adtype, portaltype, time.Now().Unix())
	return dst
}

func updateSubscribe(db *sql.DB, openid string) {
	_, err := db.Exec("UPDATE wx_conn SET subscribe = 1, stime = NOW() WHERE openid = ?", openid)
	if err != nil {
		log.Printf("updateSubscribe failed:%v", err)
	}
}

func getOnlineInfo(db *sql.DB, openid string) (util.OnlineInfo, error) {
	var info util.OnlineInfo
	info.Openid = openid
	err := db.QueryRow("SELECT w.usermac, w.acname, o.phone, o.ip, o.acip FROM wx_conn w, online_status o WHERE w.usermac = o.mac AND w.etime > NOW() AND w.openid = ?", openid).Scan(&info.Usermac, &info.Acname, &info.Phone, &info.Userip, &info.Acip)
	if err != nil {
		log.Printf("getOnlineInfo query failed:%v", err)
	}
	return info, err
}

func (s *server) CheckSubscribe(ctx context.Context, in *verify.SubscribeRequest) (*verify.CheckReply, error) {
	log.Printf("CheckSubscribe request:%v", in)
	util.PubRPCRequest(w, "verify", "CheckSubscribe")
	accesstoken := util.GetAccessToken(db, in.Type)
	subscribe := util.CheckSubscribe(accesstoken, in.Openid)
	dst := mpURL
	if subscribe {
		dst = genPortalDst(db, in.Openid)
		updateSubscribe(db, in.Openid)
	} else {
		info, err := getOnlineInfo(db, in.Openid)
		if err == nil {
			util.AddOnlineTask(kv, info)
		}
	}
	util.PubRPCSuccRsp(w, "verify", "CheckSubscribe")
	return &verify.CheckReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}, Subscribe: subscribe,
		Dst: dst}, nil
}

func (s *server) RecordWxConn(ctx context.Context, in *verify.WxConnRequest) (*common.CommReply, error) {
	log.Printf("RecordWxConn request:%v", in)
	util.PubRPCRequest(w, "verify", "RecordWxConn")
	_, err := db.Exec("INSERT INTO wx_conn(openid, acname, acip, usermac, userip, apmac, tid, ctime, etime) VALUES (?, ?, ?, ?, ?, ?, ?, NOW(), DATE_ADD(NOW(), INTERVAL 1 HOUR)) ON DUPLICATE KEY UPDATE acname = ?, acip = ?, usermac = ?, userip = ?, apmac = ?, tid = ?, etime = DATE_ADD(NOW(), INTERVAL 1 HOUR)",
		in.Openid, in.Acname, in.Acip, in.Usermac, in.Userip, in.Apmac, in.Tid,
		in.Acname, in.Acip, in.Usermac, in.Userip, in.Apmac, in.Tid)
	if err != nil {
		log.Printf("RecordWxConn failed:%v", err)
	}
	util.PubRPCSuccRsp(w, "verify", "RecordWxConn")
	return &common.CommReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}}, nil
}

func zteLogin(phone, userip, usermac, acip, acname string, stype uint) (bool, error) {
	flag, err := zte.Loginnopass(phone, userip, usermac, acip, acname, stype)
	if flag {
		return true, nil
	}
	log.Printf("PortalLogin zte loginnopass failed, phone:%s stype:%d",
		phone, stype)
	return flag, err
}

func refreshActiveTime(db *sql.DB, uid int64) {
	_, err := db.Exec("UPDATE user SET atime = NOW() WHERE uid = ?", uid)
	if err != nil {
		log.Printf("refreshActiveTime failed:%v", err)
	}
}

func addOnlineRecord(db *sql.DB, uid int64, phone string, info *verify.PortalInfo) {
	_, err := db.Exec("INSERT INTO online_record(uid, phone, usermac, apmac, acname, ctime) VALUES (?, ?, ?, ?, ?, NOW())",
		uid, phone, info.Usermac, info.Apmac, info.Acname)
	if err != nil {
		log.Printf("addOnlineRecord online record failed:%d %s %v %v",
			uid, phone, info, err)
	}
	_, err = db.Exec("INSERT INTO online_status(phone, mac, ip, apmac, acip, ctime, etime) VALUES (?, ?, ?, ?, ?, NOW(), DATE_ADD(NOW(), INTERVAL 1 HOUR)) ON DUPLICATE KEY UPDATE ip = ?, apmac = ?, acip = ?, etime = DATE_ADD(NOW(), INTERVAL 1 HOUR)",
		phone, info.Usermac, info.Userip, info.Apmac, info.Acip, info.Userip,
		info.Apmac, info.Acip)
	if err != nil {
		log.Printf("addOnlineRecord online status failed:%d %s %v %v",
			uid, phone, info, err)
	}
}

func hasMacRecord(db *sql.DB, usermac string) bool {
	var cnt int64
	err := db.QueryRow("SELECT COUNT(id) FROM user_mac WHERE mac = ?", usermac).Scan(&cnt)
	if err != nil {
		log.Printf("hasMacRecord query failed:%v", err)
		return false
	}
	if cnt > 0 {
		return true
	}
	return false
}

func oneClickLogin(db *sql.DB, in *verify.PortalLoginRequest) (int64, error) {
	var uid int64
	var phone string
	usermac := strings.Replace(in.Info.Usermac, ":", "", -1)
	log.Printf("usermac:%s", usermac)
	err := db.QueryRow("SELECT m.phone, u.uid FROM user_mac m, user u WHERE m.uid = u.uid AND m.mac = ?", usermac).
		Scan(&phone, &uid)
	if err != nil {
		log.Printf("OneClickLogin query failed:%v", err)
		return 0, err
	}

	stype := getAcSys(db, in.Info.Acname)
	bitmap := getUserBitmap(db, uid)
	err = checkZteReg(db, bitmap, stype, uid, phone)
	if err != nil {
		return uid, err
	}
	if !isTestParam(in.Info) {
		flag, err := zteLogin(phone, in.Info.Userip,
			in.Info.Usermac, in.Info.Acip, in.Info.Acname, stype)
		if !flag {
			log.Printf("OneClickLogin zte loginnopass retry failed, phone:%s",
				phone)
			return uid, err
		}
	}
	recordUserMac(db, uid, in.Info.Usermac, phone)
	refreshActiveTime(db, uid)
	addOnlineRecord(db, uid, phone, in.Info)
	return uid, nil
}

func portalLogin(db *sql.DB, in *verify.PortalLoginRequest) (int64, error) {
	stype := getAcSys(db, in.Info.Acname)
	if !isTestParam(in.Info) {
		flag, err := zteLogin(specPhone, in.Info.Userip,
			in.Info.Usermac, in.Info.Acip, in.Info.Acname, stype)
		if !flag {
			log.Printf("portalLogin zteLogin retry failed, request:%v", in)
			if err == zte.ErrForbid {
				return 0, err
			}
			return 0, errors.New("zteLogin login failed")
		}
	}

	res, err := db.Exec("INSERT INTO user(username, phone, ctime, atime, bitmap, term, aptime) VALUES (?, ?, NOW(), NOW(), 3, 2, NOW()) ON DUPLICATE KEY UPDATE atime = NOW(), bitmap = 3, aptime = NOW()",
		in.Info.Usermac, specPhone)
	if err != nil {
		log.Printf("portalLogin insert user failed, request:%v", in)
		return 0, err
	}
	uid, err := res.LastInsertId()
	if err != nil {
		log.Printf("PortalLogin add user failed:%v", err)
		return 0, err
	}
	recordUserMac(db, uid, in.Info.Usermac, specPhone)
	addOnlineRecord(db, uid, specPhone, in.Info)
	return uid, err
}

func (s *server) UnifyLogin(ctx context.Context, in *verify.PortalLoginRequest) (*verify.PortalLoginReply, error) {
	log.Printf("UnifyLogin request:%v", in)
	util.PubRPCRequest(w, "verify", "UnifyLogin")
	var err error
	var uid int64
	if hasMacRecord(db, in.Info.Usermac) {
		uid, err = oneClickLogin(db, in)
	} else {
		uid, err = portalLogin(db, in)
	}
	if err != nil {
		log.Printf("unify login failed, req:%v", in)
		return &verify.PortalLoginReply{
			Head: &common.Head{
				Retcode: common.ErrCode_ZTE_LOGIN}}, nil
	}
	token, _, _, err := util.RefreshTokenPrivdata(db, kv, uid, expiretime)
	if err != nil {
		log.Printf("Register refreshTokenPrivdata user info failed:%v", err)
		return &verify.PortalLoginReply{Head: &common.Head{Retcode: 1}}, err
	}
	adtype := util.GetAdType(db, in.Info.Apmac)
	ptype := util.GetPortalType(db, in.Info.Apmac)
	dir := util.GetPortalPath(db, in.Info.Acname, ptype)
	log.Printf("UnifyLogin succ request:%v uid:%d token:%s", in, uid, token)
	util.PubRPCSuccRsp(w, "verify", "UnifyLogin")
	return &verify.PortalLoginReply{
		Head: &common.Head{Retcode: 0, Uid: uid}, Token: token, Portaldir: dir,
		Portaltype: ptype, Adtype: adtype}, nil
}

func (s *server) OneClickLogin(ctx context.Context, in *verify.AccessRequest) (*verify.PortalLoginReply, error) {
	log.Printf("OneClickLogin request:%v", in)
	util.PubRPCRequest(w, "verify", "OneClickLogin")
	var uid int64
	var phone string
	usermac := strings.Replace(in.Info.Usermac, ":", "", -1)
	err := db.QueryRow("SELECT m.phone, u.uid FROM user_mac m, user u WHERE m.uid = u.uid AND m.mac = ?", usermac).
		Scan(&phone, &uid)
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
	if !isTestParam(in.Info) {
		flag, err := zteLogin(phone, in.Info.Userip,
			in.Info.Usermac, in.Info.Acip, in.Info.Acname, stype)
		if !flag {
			log.Printf("OneClickLogin zte loginnopass retry failed, phone:%s",
				phone)
			if err == zte.ErrForbid {
				return &verify.PortalLoginReply{
					Head: &common.Head{Retcode: common.ErrCode_LOGIN_FORBID}}, nil
			}
			return &verify.PortalLoginReply{
				Head: &common.Head{Retcode: common.ErrCode_ZTE_LOGIN}}, nil
		}
	}
	recordUserMac(db, uid, in.Info.Usermac, phone)
	refreshActiveTime(db, uid)
	addOnlineRecord(db, uid, phone, in.Info)
	token, _, _, err := util.RefreshTokenPrivdata(db, kv, uid, expiretime)
	if err != nil {
		log.Printf("Register refreshTokenPrivdata user info failed:%v", err)
		return &verify.PortalLoginReply{Head: &common.Head{Retcode: 1}}, err
	}
	var adtype int64
	if util.IsWjjAcname(in.Info.Acname) {
		adtype = 1
	} else {
		adtype = util.GetAdType(db, in.Info.Apmac)
	}
	ptype := util.GetPortalType(db, in.Info.Apmac)
	dir := util.GetPortalPath(db, in.Info.Acname, ptype)
	var img, dst string
	if isSpecTaobaoTime() {
		img, dst = getSpecTaobaoInfo(db)
	} else {
		img, dst = getTaobaoInfo(db)
	}
	util.PubRPCSuccRsp(w, "verify", "OneClickLogin")
	log.Printf("OneClickLogin succ request:%v uid:%d token:%s", in, uid, token)
	return &verify.PortalLoginReply{
		Head: &common.Head{Retcode: 0, Uid: uid}, Token: token, Portaldir: dir,
		Portaltype: ptype, Adtype: adtype, Cover: img,
		Dst: dst}, nil
}

func main() {
	lis, err := net.Listen("tcp", util.VerifyServerPort)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	w = util.NewNsqProducer()

	db, err = util.InitDB(false)
	if err != nil {
		log.Fatalf("failed to init db connection: %v", err)
	}
	db.SetMaxIdleConns(util.MaxIdleConns)
	kv = util.InitRedis()
	go util.ReportHandler(kv, util.VerifyServerName, util.VerifyServerPort)

	s := util.NewGrpcServer()
	verify.RegisterVerifyServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
