package main

import (
	"database/sql"
	"encoding/base64"
	"log"
	"net"

	redis "gopkg.in/redis.v5"

	"Server/proto/common"
	"Server/proto/punch"
	"Server/util"
	"Server/weixin"

	simplejson "github.com/bitly/go-simplejson"
	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/net/context"
)

const (
	expiretime = 3600 * 24 * 30
)

type server struct{}

var db *sql.DB

var kv *redis.Client

func (s *server) SubmitCode(ctx context.Context, in *punch.CodeRequest) (*punch.LoginReply, error) {
	log.Printf("SubmitCode request uid:%d code:%s", in.Head.Uid, in.Code)
	openid, skey, err := weixin.GetSession(in.Code)
	if err != nil {
		log.Printf("SubmitCode GetSession failed:%v", err)
		return &punch.LoginReply{
			Head: &common.Head{Retcode: common.ErrCode_ILLEGAL_CODE}}, nil
	}
	sid := util.GenSalt()
	var uid int64
	err = db.QueryRow("SELECT uid FROM user u, xcx_openid x WHERE u.username = x.unionid AND x.openid = ?", openid).Scan(&uid)
	if err != nil {
		_, err = db.Exec("INSERT INTO xcx_openid(openid, skey, sid, ctime) VALUES (?, ?, ?, NOW()) ON DUPLICATE KEY UPDATE skey = ?, sid = ?",
			openid, skey, sid, skey, sid)
		if err != nil {
			log.Printf("record failed:%v", err)
			return &punch.LoginReply{
				Head: &common.Head{Retcode: 1}}, nil
		}
	}
	if uid == 0 {
		log.Printf("user not found, openid:%s", openid)
		return &punch.LoginReply{
			Head: &common.Head{Retcode: 0}, Flag: 0, Sid: sid}, nil
	}

	token, _, _, err := util.RefreshTokenPrivdata(db, kv, uid, expiretime)
	return &punch.LoginReply{
		Head: &common.Head{Retcode: 0}, Flag: 1, Uid: uid, Token: token}, nil
}

func checkSign(skey, rawdata, signature string) bool {
	data := rawdata + skey
	sign := util.Sha1(data)
	return sign == signature
}

func decryptData(skey, encrypted, iv string) ([]byte, error) {
	src, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return []byte(""), err
	}
	key, err := base64.StdEncoding.DecodeString(skey)
	if err != nil {
		return []byte(""), err
	}
	vec, err := base64.StdEncoding.DecodeString(iv)
	if err != nil {
		return []byte(""), err
	}
	dst, err := util.AesDecrypt(src, key, vec)
	if err != nil {
		return []byte(""), err
	}
	return dst, nil
}

func (s *server) Login(ctx context.Context, in *punch.LoginRequest) (*punch.LoginReply, error) {
	log.Printf("Login request:%v", in)
	var skey, unionid, openid string
	err := db.QueryRow("SELECT skey, unionid, openid FROM xcx_openid WHERE sid = ?", in.Sid).
		Scan(&skey, &unionid, &openid)
	if err != nil {
		log.Printf("illegal sid:%s", in.Sid)
		return &punch.LoginReply{
			Head: &common.Head{Retcode: 1}}, nil
	}
	if !checkSign(skey, in.Rawdata, in.Signature) {
		log.Printf("check signature failed sid:%s", in.Sid)
		return &punch.LoginReply{
			Head: &common.Head{Retcode: 1}}, nil
	}
	var uid int64
	if unionid == "" { //has login
		dst, err := decryptData(skey, in.Encrypteddata, in.Iv)
		if err != nil {
			log.Printf("aes decrypt failed sid:%s skey:%s", in.Sid, skey)
			return &punch.LoginReply{
				Head: &common.Head{Retcode: 1}}, nil
		}
		js, err := simplejson.NewJson(dst)
		if err != nil {
			log.Printf("parse plaintext failed:%s", string(dst))
			return &punch.LoginReply{
				Head: &common.Head{Retcode: 1}}, nil
		}
		unionid, err := js.Get("unionId").String()
		if err != nil {
			log.Printf("get unionid failed:%v", err)
			return &punch.LoginReply{
				Head: &common.Head{Retcode: 1}}, nil
		}
		_, err = db.Exec("UPDATE xcx_openid SET unionid = ? WHERE openid = ?",
			unionid, openid)
		if err != nil {
			log.Printf("update unionid failed:%v", err)
			return &punch.LoginReply{
				Head: &common.Head{Retcode: 1}}, nil
		}
		db.QueryRow("SELECT uid FROM user WHERE username = ?", unionid).Scan(&uid)
		if uid == 0 {
			nickname, _ := js.Get("nickName").String()
			headurl, _ := js.Get("avatarUrl").String()
			gender, _ := js.Get("gender").Int64()
			sex := 0
			if gender == 1 {
				sex = 1
			}
			res, err := db.Exec("INSERT IGNORE INTO user(username, nickname, headurl, sex, term, channel, ctime) VALUES (?, ?, ?, ?, 2, 'xcx', NOW())",
				unionid, nickname, headurl, sex)
			if err != nil {
				log.Printf("create user failed:%v", err)
				return &punch.LoginReply{
					Head: &common.Head{Retcode: 1}}, nil
			}
			uid, _ = res.LastInsertId()
		}
	} else {
		db.QueryRow("SELECT uid FROM user WHERE username = ?", unionid).Scan(&uid)
	}
	if uid == 0 {
		log.Printf("select user failed sid:%s", in.Sid)
		return &punch.LoginReply{
			Head: &common.Head{Retcode: 1}}, nil
	}

	token, _, _, err := util.RefreshTokenPrivdata(db, kv, uid, expiretime)
	return &punch.LoginReply{
		Head: &common.Head{Retcode: 0}, Uid: uid, Token: token}, nil
}

func main() {
	lis, err := net.Listen("tcp", util.PunchServerPort)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	db, err = util.InitDB(true)
	if err != nil {
		log.Fatalf("failed to init db connection: %v", err)
	}
	db.SetMaxIdleConns(util.MaxIdleConns)

	kv = util.InitRedis()
	go util.ReportHandler(kv, util.PunchServerName, util.PunchServerPort)

	s := util.NewGrpcServer()
	punch.RegisterPunchServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
