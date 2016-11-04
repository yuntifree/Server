package main

import (
	"errors"
	"log"
	"net"

	"database/sql"

	"../../util"

	common "../../proto/common"
	verify "../../proto/verify"
	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

const (
	port       = ":50052"
	servername = "service:verify"
	expiretime = 3600 * 24 * 30
	mastercode = 251653
	randrange  = 1000000
)

type server struct{}

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
	db, err := util.InitDB(false)
	if err != nil {
		log.Printf("connect mysql failed:%v", err)
		return false, err
	}
	defer db.Close()
	log.Printf("request phone:%s, ctype:%d", phone, ctype)
	flag := util.ExistPhone(db, phone)
	if ctype == 1 && !flag {
		return false, errors.New("phone not exist")
	} else if ctype == 0 && flag {
		return false, errors.New("phone already exist")
	}

	var code int
	err = db.QueryRow("SELECT code FROM phone_code WHERE phone = ? AND used = 0 AND etime > NOW() AND timestampdiff(second, stime, now()) < 300 ORDER BY pid DESC LIMIT 1", phone).Scan(&code)
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
	db, err := util.InitDB(false)
	if err != nil {
		log.Printf("connect mysql failed:%v", err)
		return &verify.LoginReply{Head: &common.Head{Retcode: 1}}, err
	}
	defer db.Close()

	var uid int64
	var epass string
	var salt string
	err = db.QueryRow("SELECT uid, password, salt FROM back_login WHERE username = ?", in.Username).Scan(&uid, &epass, &salt)
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

func (s *server) Login(ctx context.Context, in *verify.LoginRequest) (*verify.LoginReply, error) {
	db, err := util.InitDB(false)
	if err != nil {
		log.Printf("connect mysql failed:%v", err)
		return &verify.LoginReply{Head: &common.Head{Retcode: 1}}, err
	}
	defer db.Close()

	var uid int64
	var epass string
	var salt string
	var wifipass string
	err = db.QueryRow("SELECT uid, password, salt, wifi_passwd FROM user WHERE username = ?", in.Username).Scan(&uid, &epass, &salt, &wifipass)
	if err != nil {
		return &verify.LoginReply{Head: &common.Head{Retcode: 2}}, err
	}
	pass := util.GenSaltPasswd(in.Password, salt)
	if pass != epass {
		return &verify.LoginReply{Head: &common.Head{Retcode: 3}}, errors.New("verify password failed")
	}

	token := util.GenSalt()
	privdata := util.GenSalt()

	_, err = db.Exec("UPDATE user SET token = ?, private = ?, etime = DATE_ADD(NOW(), INTERVAL 30 DAY) WHERE uid = ?", token, privdata, uid)
	if err != nil {
		return &verify.LoginReply{Head: &common.Head{Retcode: 2}}, err
	}

	return &verify.LoginReply{Head: &common.Head{Uid: uid}, Token: token, Privdata: privdata, Expire: expiretime, Wifipass: wifipass}, nil
}

func (s *server) Register(ctx context.Context, in *verify.RegisterRequest) (*verify.RegisterReply, error) {
	db, err := util.InitDB(false)
	if err != nil {
		log.Printf("connect mysql failed:%v", err)
		return &verify.RegisterReply{Head: &common.Head{Retcode: 1}}, err
	}
	defer db.Close()
	flag := util.ExistPhone(db, in.Username)
	if flag {
		log.Printf("used phone:%v", in.Username)
		return &verify.RegisterReply{Head: &common.Head{Retcode: common.ErrCode_USED_PHONE}}, nil
	}
	flag, err = checkPhoneCode(db, in.Username, in.Code)
	if err != nil {
		return &verify.RegisterReply{Head: &common.Head{Retcode: 1}}, err
	}

	if !flag {
		return &verify.RegisterReply{Head: &common.Head{Retcode: 1}}, err
	}

	token := util.GenSalt()
	privdata := util.GenSalt()
	salt := util.GenSalt()
	epass := util.GenSaltPasswd(in.Password, salt)
	wifipass := util.GenWifiPass()
	res, err := db.Exec("INSERT IGNORE INTO user (username, phone, password, salt, wifi_passwd, token, private, ctime, atime, etime) VALUES (?,?,?,?,?,?,?,NOW(),NOW(),DATE_ADD(NOW(), INTERVAL 30 DAY))", in.Username, in.Username, epass, salt, wifipass, token, privdata)
	if err != nil {
		log.Printf("add user failed:%v", err)
		return &verify.RegisterReply{Head: &common.Head{Retcode: 1}}, err
	}

	uid, err := res.LastInsertId()
	if err != nil {
		log.Printf("add user failed:%v", err)
		return &verify.RegisterReply{Head: &common.Head{Retcode: 1}}, err
	}
	return &verify.RegisterReply{Head: &common.Head{Retcode: 0, Uid: uid}, Token: token, Privdata: privdata, Expire: expiretime, Wifipass: wifipass}, nil
}

func (s *server) Logout(ctx context.Context, in *verify.LogoutRequest) (*verify.LogoutReply, error) {
	db, err := util.InitDB(false)
	if err != nil {
		log.Printf("connect mysql failed:%v", err)
		return &verify.LogoutReply{Head: &common.Head{Retcode: 1}}, err
	}
	defer db.Close()
	flag := util.CheckToken(db, in.Head.Uid, in.Token, 0)
	if !flag {
		log.Printf("check token failed uid:%d, token:%s", in.Head.Uid, in.Token)
		return &verify.LogoutReply{Head: &common.Head{Retcode: 1}}, err
	}
	util.ClearToken(db, in.Head.Uid)
	return &verify.LogoutReply{Head: &common.Head{Retcode: 0}}, nil
}

func (s *server) CheckToken(ctx context.Context, in *verify.TokenRequest) (*verify.TokenReply, error) {
	db, err := util.InitDB(false)
	if err != nil {
		log.Printf("connect mysql failed:%v", err)
		return &verify.TokenReply{Head: &common.Head{Retcode: 1}}, err
	}
	defer db.Close()
	flag := util.CheckToken(db, in.Head.Uid, in.Token, in.Type)
	if !flag {
		log.Printf("check token failed uid:%d, token:%s", in.Head.Uid, in.Token)
		return &verify.TokenReply{Head: &common.Head{Retcode: 1}}, err
	}
	return &verify.TokenReply{Head: &common.Head{Retcode: 0}}, nil
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
	db, err := util.InitDB(false)
	if err != nil {
		log.Printf("connect mysql failed:%v", err)
		return &verify.AutoReply{Head: &common.Head{Retcode: 1}}, err
	}
	defer db.Close()

	flag := checkPrivdata(db, in.Head.Uid, in.Token, in.Privdata)
	if !flag {
		log.Printf("check privdata failed, uid:%d token:%s privdata:%s", in.Head.Uid, in.Token, in.Privdata)
		return &verify.AutoReply{Head: &common.Head{Retcode: 1}}, errors.New("check privdata failed")
	}
	token := util.GenSalt()
	privdata := util.GenSalt()
	updatePrivdata(db, in.Head.Uid, token, privdata)
	return &verify.AutoReply{Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}, Token: token, Privdata: privdata, Expire: expiretime}, nil
}

func main() {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	go util.ReportHandler(servername, port)

	s := grpc.NewServer()
	verify.RegisterVerifyServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
