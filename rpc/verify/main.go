package main

import (
	"errors"
	"log"
	"net"

	"database/sql"

	redis "gopkg.in/redis.v5"

	"../../util"
	"../../zte"

	common "../../proto/common"
	verify "../../proto/verify"
	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

const (
	expiretime = 3600 * 24 * 30
	mastercode = 251653
	randrange  = 1000000
)

type server struct{}

var db *sql.DB
var kv *redis.Client

func checkPhoneCode(db *sql.DB, phone string, code int32) (bool, error) {
	if code == mastercode {
		return true, nil
	}

	var realcode int32
	var pid int32
	err := db.QueryRow("SELECT code, pid FROM phone_code WHERE phone = ? AND used = 0 ORDER BY pid DESC LIMIT 1", phone).Scan(&realcode, &pid)
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

func getPhoneCode(phone string, ctype int32) (bool, error) {
	log.Printf("request phone:%s, ctype:%d", phone, ctype)
	if ctype == 1 {
		if flag := util.ExistPhone(db, phone); !flag {
			return false, errors.New("phone not exist")
		}
	}

	var code int
	err := db.QueryRow("SELECT code FROM phone_code WHERE phone = ? AND used = 0 AND etime > NOW() AND timestampdiff(second, stime, now()) < 300 ORDER BY pid DESC LIMIT 1", phone).Scan(&code)
	if err != nil {
		code := util.Randn(randrange)
		_, err := db.Exec("INSERT INTO phone_code(phone, code, ctime, stime, etime) VALUES (?, ?, NOW(), NOW(), DATE_ADD(NOW(), INTERVAL 5 MINUTE))", phone, code)
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
	var uid int64
	var epass string
	var salt string
	err := db.QueryRow("SELECT uid, password, salt FROM back_login WHERE username = ?", in.Username).Scan(&uid, &epass, &salt)
	if err != nil {
		return &verify.LoginReply{Head: &common.Head{Retcode: 2}}, err
	}
	pass := util.GenSaltPasswd(in.Password, salt)
	if pass != epass {
		return &verify.LoginReply{Head: &common.Head{Retcode: 3}}, errors.New("verify password failed")
	}

	token := util.GenSalt()
	_, err = db.Exec("UPDATE back_login SET skey = ?, login_time = NOW(), expire_time = DATE_ADD(NOW(), INTERVAL 30 DAY) WHERE uid = ?", token, uid)
	if err != nil {
		return &verify.LoginReply{Head: &common.Head{Retcode: 2}}, err
	}

	return &verify.LoginReply{Head: &common.Head{Uid: uid}, Token: token}, nil
}

func recordWxOpenid(db *sql.DB, uid int64, wtype int32, openid string) {
	_, err := db.Exec("INSERT IGNORE INTO wx_openid(uid, wtype, openid, ctime) VALUES (?, ?, ?, NOW())", uid, wtype, openid)
	if err != nil {
		log.Printf("record wx openid failed uid:%d wtype:%d openid:%s\n", uid, wtype, openid)
	}
}

func recordWxUnionid(db *sql.DB, uid int64, unionid string) {
	_, err := db.Exec("INSERT INTO user_unionid(uid, unionid, ctime) VALUES(?, ?, NOW()) ON DUPLICATE KEY UPDATE unionid = ?", uid, unionid, unionid)
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
	res, err := db.Exec("INSERT IGNORE INTO user(username, headurl, sex, token, private, wifi_passwd, etime, atime, ctime) VALUES (?, ?, ?, ?, ?,?, DATE_ADD(NOW(), INTERVAL 30 DAY), NOW(), NOW())", wxi.UnionID, wxi.HeadURL, wxi.Sex, token, privdata, wifipass)
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
		err = db.QueryRow("SELECT uid, wifi_passwd FROM user WHERE username = ?", wxi.UnionID).Scan(&uid, &wifipass)
		if err != nil {
			log.Printf("search uid failed %s:%v", wxi.UnionID, err)
			return &verify.LoginReply{Head: &common.Head{Retcode: 1}}, err
		}
		_, err = db.Exec("UPDATE user SET token = ?, private = ?, etime = DATE_ADD(NOW(), INTERVAL 30 DAY), atime = NOW() WHERE uid = ?", token, privdata, uid)
		if err != nil {
			log.Printf("search uid failed %s:%v", wxi.UnionID, err)
			return &verify.LoginReply{Head: &common.Head{Retcode: 1}}, err
		}
	}

	recordWxOpenid(db, uid, 0, wxi.Openid)
	recordWxUnionid(db, uid, privdata)
	util.SetCachedToken(kv, uid, token)
	return &verify.LoginReply{Head: &common.Head{Uid: uid}, Token: token, Privdata: privdata, Expire: expiretime, Wifipass: wifipass}, nil
}

func (s *server) Login(ctx context.Context, in *verify.LoginRequest) (*verify.LoginReply, error) {
	var uid int64
	var epass string
	var salt string
	var wifipass string
	err := db.QueryRow("SELECT uid, password, salt, wifi_passwd FROM user WHERE username = ?", in.Username).Scan(&uid, &epass, &salt, &wifipass)
	if err != nil {
		return &verify.LoginReply{Head: &common.Head{Retcode: 2}}, err
	}
	pass := util.GenSaltPasswd(in.Password, salt)
	if pass != epass {
		return &verify.LoginReply{Head: &common.Head{Retcode: 3}}, errors.New("verify password failed")
	}

	token := util.GenSalt()
	privdata := util.GenSalt()

	_, err = db.Exec("UPDATE user SET token = ?, private = ?, etime = DATE_ADD(NOW(), INTERVAL 30 DAY), model = ?, udid = ? WHERE uid = ?", token, privdata, in.Model, in.Udid, uid)
	if err != nil {
		return &verify.LoginReply{Head: &common.Head{Retcode: 2}}, err
	}
	util.SetCachedToken(kv, uid, token)

	return &verify.LoginReply{Head: &common.Head{Uid: uid}, Token: token, Privdata: privdata, Expire: expiretime, Wifipass: wifipass}, nil
}

func (s *server) Register(ctx context.Context, in *verify.RegisterRequest) (*verify.RegisterReply, error) {
	token := util.GenSalt()
	privdata := util.GenSalt()
	salt := util.GenSalt()
	epass := util.GenSaltPasswd(in.Password, salt)
	log.Printf("phone:%s token:%s privdata:%s salt:%s epass:%s\n", in.Username, token, privdata, salt, epass)
	res, err := db.Exec(`INSERT IGNORE INTO user (username, password, salt, token, private, model, udid,
	channel, reg_ip, version, term, ctime, atime, etime) VALUES (?,?,?,?,?,?,?,?,?,?,?,NOW(),NOW(),
	DATE_ADD(NOW(), INTERVAL 30 DAY))`,
		in.Username, epass, salt, token, privdata, in.Client.Model, in.Client.Udid, in.Client.Channel,
		in.Client.Regip, in.Client.Version, in.Client.Term)
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
		_, err := db.Exec("UPDATE user SET token = ?, private = ?, password = ?, salt = ?, model = ?, udid = ?, version = ?, term = ?, atime = NOW(), etime = DATE_ADD(NOW(), INTERVAL 30 DAY) WHERE uid = ?",
			token, privdata, epass, salt, in.Client.Model, in.Client.Udid, in.Client.Version, in.Client.Term,
			uid)
		if err != nil {
			log.Printf("update user info failed:%v", err)
			return &verify.RegisterReply{Head: &common.Head{Retcode: 1}}, err
		}
	}
	util.SetCachedToken(kv, uid, token)
	return &verify.RegisterReply{Head: &common.Head{Retcode: 0, Uid: uid}, Token: token, Privdata: privdata, Expire: expiretime}, nil
}

func (s *server) Logout(ctx context.Context, in *verify.LogoutRequest) (*common.CommReply, error) {
	flag := util.CheckToken(db, in.Head.Uid, in.Token, 0)
	if !flag {
		log.Printf("check token failed uid:%d, token:%s", in.Head.Uid, in.Token)
		return &common.CommReply{Head: &common.Head{Retcode: 1}}, errors.New("check token failed")
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
		err = db.QueryRow("SELECT token, IF(etime > NOW(), false, true) FROM user WHERE deleted = 0 AND uid = ?", in.Head.Uid).Scan(&tk, &expire)
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
		log.Printf("CheckToken token not match, uid:%d token:%s real:%s\n", in.Head.Uid, in.Token, tk)
		return &common.CommReply{Head: &common.Head{Retcode: 1}}, nil
	}
	flag := util.CheckToken(db, in.Head.Uid, in.Token, in.Type)
	if !flag {
		log.Printf("check token failed uid:%d, token:%s", in.Head.Uid, in.Token)
		return &common.CommReply{Head: &common.Head{Retcode: 1}}, errors.New("checkToken failed")
	}
	return &common.CommReply{Head: &common.Head{Retcode: 0}}, nil
}

func checkPrivdata(db *sql.DB, uid int64, token, privdata string) bool {
	var etoken string
	var eprivdata string
	var flag bool
	err := db.QueryRow("SELECT token, private, IF(etime > NOW(), 1, 0) FROM user WHERE uid = ?", uid).Scan(&etoken, &eprivdata, &flag)
	if err != nil {
		log.Printf("query failed:%v", err)
		return false
	}

	if !flag {
		log.Printf("token expire, uid:%d, token:%s, privdata:%s", uid, token, privdata)
		return false
	}

	if etoken != token || eprivdata != privdata {
		log.Printf("check privdata failed, token:%s-%s, privdata:%s-%s", token, etoken, privdata, eprivdata)
		return false
	}
	return true
}

func updatePrivdata(db *sql.DB, uid int64, token, privdata string) error {
	_, err := db.Exec("UPDATE user SET token = ?, private = ?, etime = DATE_ADD(NOW(), INTERVAL 30 DAY) WHERE uid = ?",
		token, privdata, uid)
	return err
}

func (s *server) AutoLogin(ctx context.Context, in *verify.AutoRequest) (*verify.AutoReply, error) {
	flag := checkPrivdata(db, in.Head.Uid, in.Token, in.Privdata)
	if !flag {
		log.Printf("check privdata failed, uid:%d token:%s privdata:%s", in.Head.Uid, in.Token, in.Privdata)
		return &verify.AutoReply{Head: &common.Head{Retcode: 1}}, errors.New("check privdata failed")
	}
	token := util.GenSalt()
	privdata := util.GenSalt()
	updatePrivdata(db, in.Head.Uid, token, privdata)
	util.SetCachedToken(kv, in.Head.Uid, token)
	return &verify.AutoReply{Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}, Token: token, Privdata: privdata, Expire: expiretime}, nil
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
	return &verify.LoginReply{Head: &common.Head{Retcode: 0, Uid: uid}, Token: token, Privdata: privdata, Expire: expiretime}, nil
}

func updateTokenTicket(db *sql.DB, appid, accessToken, ticket string) {
	_, err := db.Exec("UPDATE wx_token SET access_token = ?, api_ticket = ?, expire_time = DATE_ADD(NOW(), INTERVAL 1 HOUR) WHERE appid = ?", accessToken, ticket, appid)
	if err != nil {
		log.Printf("updateTokenTicket failed:%v", err)
	}
}

func (s *server) GetWxTicket(ctx context.Context, in *verify.TicketRequest) (*verify.TicketReply, error) {
	var token, ticket string
	err := db.QueryRow("SELECT access_token, api_ticket FROM wx_token WHERE expire_time > NOW() AND appid = ? LIMIT 1", util.WxAppid).Scan(&token, &ticket)
	if err == nil {
		log.Printf("GetWxTicket select succ, token:%s ticket:%s\n", token, ticket)
		return &verify.TicketReply{Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}, Token: token, Ticket: ticket}, nil
	}
	token, err = util.GetWxToken(util.WxAppid, util.WxAppkey)
	if err != nil {
		log.Printf("GetWxToken failed:%v", err)
		return &verify.TicketReply{Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil
	}
	ticket, err = util.GetWxJsapiTicket(token)
	if err != nil {
		log.Printf("GetWxToken failed:%v", err)
		return &verify.TicketReply{Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil
	}

	updateTokenTicket(db, util.WxAppid, token, ticket)
	return &verify.TicketReply{Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}, Token: token, Ticket: ticket}, nil
}

func recordZteCode(db *sql.DB, phone, code string) {
	_, err := db.Exec("INSERT INTO zte_code(phone, code, ctime) VALUES (?, ?, NOW()) ON DUPLICATE KEY UPDATE code = ?",
		phone, code, code)
	if err != nil {
		log.Printf("recordZteCode query failed:%s %s %v", phone, code, err)
	}
}

func (s *server) GetCheckCode(ctx context.Context, in *verify.CodeRequest) (*common.CommReply, error) {
	code, err := zte.Register(in.Phone)
	if err != nil {
		log.Printf("GetCheckCode Register failed:%v", err)
		return &common.CommReply{Head: &common.Head{Retcode: 1}}, err
	}
	recordZteCode(db, in.Phone, code)
	return &common.CommReply{Head: &common.Head{Retcode: 0}}, nil
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

	s := grpc.NewServer()
	verify.RegisterVerifyServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
