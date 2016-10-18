package main

import (
	"errors"
	"log"
	"net"

	"database/sql"
	"math/rand"

	"../../util"

	common "../../proto/common"
	verify "../../proto/verify"
	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

const (
	port = ":50052"
)

type server struct{}

func checkPhoneCode(phone string, code int32) (bool, error) {
	db, err := sql.Open("mysql", "root:@/yunti?charset=utf8")
	if err != nil {
		return false, err
	}

	var realcode int32
	var pid int32
	err = db.QueryRow("SELECT code, pid FROM phone_code WHERE phone = ? AND used = 0 ORDER BY pid DESC LIMIT 1", phone).Scan(&realcode, &pid)
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

func (s *server) VerifyPhoneCode(ctx context.Context, in *verify.PhoneRequest) (*verify.VerifyReply, error) {
	flag, err := checkPhoneCode(in.Phone, in.Code)
	if err != nil {
		return &verify.VerifyReply{Result: false}, err
	}

	return &verify.VerifyReply{Result: flag}, nil
}

func getPhoneCode(phone string, ctype int32) (bool, error) {
	db, err := sql.Open("mysql", "root:@/yunti?charset=utf8")
	if err != nil {
		return false, err
	}
	flag := util.ExistPhone(db, phone)
	if ctype == 0 && !flag {
		return false, errors.New("phone not exist")
	} else if ctype == 1 && flag {
		return false, errors.New("phone already exist")
	}

	var code int
	err = db.QueryRow("SELECT code FROM phone_code WHERE phone = ? AND used = 0 AND etime > NOW() AND timestampdiff(second, stime, now()) < 300 ORDER BY pid DESC LIMIT 1", phone).Scan(&code)
	if err != nil {
		r := rand.New(rand.NewSource(99))
		code := r.Int31n(1000000)
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

func (s *server) Login(ctx context.Context, in *verify.LoginRequest) (*verify.LoginReply, error) {
	db, err := sql.Open("mysql", "root:@/yunti?charset=utf8")
	if err != nil {
		return &verify.LoginReply{Head: &common.Head{Retcode: 1}}, err
	}

	var uid int32
	var epass string
	var salt string
	err = db.QueryRow("SELECT uid, password, salt FROM user WHERE username = ?", in.Username).Scan(&uid, &epass, &salt)
	if err != nil {
		return &verify.LoginReply{Head: &common.Head{Retcode: 2}}, err
	}
	pass := util.GenSaltPasswd(in.Password, salt)
	if pass != epass {
		return &verify.LoginReply{Head: &common.Head{Retcode: 3}}, errors.New("verify password failed")
	}

	token := util.GenSalt()
	privdata := util.GenSalt()

	_, err = db.Exec("UPDATE user SET token = ?, private = ?, etime = DATE_ADD(NOW(), INTERVAL 1 HOUR) WHERE uid = ?", token, privdata, uid)
	if err != nil {
		return &verify.LoginReply{Head: &common.Head{Retcode: 2}}, err
	}

	return &verify.LoginReply{Head: &common.Head{Uid: uid}, Token: token, Privdata: privdata, Expire: 3600}, nil
}

func main() {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	verify.RegisterVerifyServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
